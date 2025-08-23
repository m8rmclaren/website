package cache

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Cache now has Get as well.
type Cache interface {
	Set(ctx context.Context, key string, value []byte) error
	Get(ctx context.Context, key string) ([]byte, bool, error)
}

// cache entry with expiration timestamp.
type entry struct {
	val []byte
	exp time.Time // zero => no expiration
}

// SimpleCache is a thread-safe in-memory cache with a default TTL.
type SimpleCache struct {
	mu   sync.RWMutex
	data map[string]entry
	ttl  time.Duration
}

// NewSimpleCache creates a cache with the given default TTL.
// ttl == 0 means entries never expire.
func NewSimpleCache(ttl time.Duration) *SimpleCache {
	return &SimpleCache{
		data: make(map[string]entry),
		ttl:  ttl,
	}
}

func (c *SimpleCache) Set(ctx context.Context, key string, value []byte) error {
	if key == "" {
		return errors.New("key cannot be empty")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// copy-on-set to avoid outside mutation
	cp := make([]byte, len(value))
	copy(cp, value)

	var exp time.Time
	if c.ttl > 0 {
		exp = time.Now().Add(c.ttl)
	}

	c.mu.Lock()
	c.data[key] = entry{val: cp, exp: exp}
	c.mu.Unlock()
	return nil
}

// Get returns (value, ok, err). ok=false if missing or expired.
// It lazily evicts expired items.
func (c *SimpleCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if key == "" {
		return nil, false, errors.New("key cannot be empty")
	}
	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()
	default:
	}

	now := time.Now()

	// Fast path: read lock
	c.mu.RLock()
	e, ok := c.data[key]
	if ok && !isExpired(e, now) {
		// copy-on-get to protect internal storage
		out := make([]byte, len(e.val))
		copy(out, e.val)
		c.mu.RUnlock()
		return out, true, nil
	}
	c.mu.RUnlock()

	// If missing or expired, handle eviction (if needed) under write lock.
	if ok && isExpired(e, now) {
		c.mu.Lock()
		// recheck to avoid TOCTOU if another goroutine updated it
		if cur, still := c.data[key]; still && isExpired(cur, now) {
			delete(c.data, key)
		}
		c.mu.Unlock()
	}
	return nil, false, nil
}

func isExpired(e entry, now time.Time) bool {
	return !e.exp.IsZero() && now.After(e.exp)
}
