package ip

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/m8rmclaren/website/internal/cache"
)

// Use ip-api.com to lookup IP address info
const (
	ipAPIBaseURL = "http://ip-api.com/json"
)

var (
	FailedToLookupIPAddress = errors.New("failed to lookup ip address")
)

type IPGeoInfo struct {
	Query       string  `json:"query"`
	Status      string  `json:"status"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	Zip         string  `json:"zip"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Timezone    string  `json:"timezone"`
	Isp         string  `json:"isp"`
	Org         string  `json:"org"`
	As          string  `json:"as"`
}

// result is sent to waiting callers for a specific host/ip.
type result struct {
	ips []net.IP
	err error
}
type Result struct {
	Err  error
	Data *IPGeoInfo
}

type ipLookup struct {
	logger echo.Logger
	cache  cache.Cache

	ipQueue    chan string
	workers    int
	results    chan Result
	wg         sync.WaitGroup
	mu         sync.Mutex
	stopCtx    context.Context
	stopCancel context.CancelFunc
	started    bool
}

func NewIpLookupService(ctx context.Context, logger echo.Logger, cache cache.Cache) (*ipLookup, error) {
	lookupService := &ipLookup{
		logger: logger,
		cache:  cache,
	}

	lookupService.startWorkers()

	return lookupService, nil
}

func (s *ipLookup) SubmitForLookup(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case s.ipQueue <- ip:
		// pass
	default:
		// Accept lossy enqueue
	}
}

func (s *ipLookup) startWorkers() {
	if s.started {
		return
	}
	s.started = true

	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go func(workerID int) {
			defer s.wg.Done()
			s.workerLoop(workerID)
		}(i)
	}
}

func (s *ipLookup) workerLoop(workerID int) {
	for {
		select {
		case ip := <-s.ipQueue:
			// run lookup with context, allow per-lookup timeout
			geoInfo, err := s.doLookup(s.stopCtx, ip)

			// put into cache if success
			if err == nil {
				geoJsonBytes, err := json.Marshal(geoInfo)
				if err != nil {

				}
				s.cache.Set(s.stopCtx, ip, geoJsonBytes)
			}

			res := result{ips: geoInfo, err: err}

			// TODO publish res to result chan
		case <-s.stopCtx.Done():
			return
		}
	}
}

func (i *ipLookup) doLookup(ctx context.Context, ip string) (*IPGeoInfo, error) {
	if ip == "" {
		return nil, fmt.Errorf("no ip provided for lookup")
	}

	url := fmt.Sprintf("%s/json/%s", ipAPIBaseURL, ip)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", FailedToLookupIPAddress, err)
	}

	client := http.Client{Timeout: 3 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", FailedToLookupIPAddress, err)
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	var out IPGeoInfo
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}

	return &out, nil
}
