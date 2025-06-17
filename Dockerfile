# syntax=docker/dockerfile:1.4

# Render tailwi
FROM node:20-alpine as tailwind
WORKDIR /app

COPY package.json yarn.lock* package-lock.json* pnpm-lock.yaml* ./
RUN npm ci

# Copy only what's needed for Tailwind build
COPY assets/css ./assets/css
COPY tailwind.config.ts ./

# Generate output.css
RUN npx tailwindcss -i ./assets/css/input.css -o ./static/output.css

# Render the templ files
FROM ghcr.io/a-h/templ:latest as templ
WORKDIR /workspace
COPY --chown=65532:65532 go.mod go.sum ./
COPY --chown=65532:65532 view/ view/
# COPY --chown=65532:65532 . .
RUN ["templ", "generate"]

# Build the service binary
FROM golang:1.23.2 as builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
# Download dependencies using the GitHub token for private modules
# https://go.dev/ref/mod#module-cache
RUN --mount=type=cache,target=/go/pkg/mod go mod download


# Copy the Go Modules manifests
COPY go.mod go.mod
# Copy the go source
COPY cmd/ cmd/
COPY internal/ internal/
COPY static/ static/
COPY --from=tailwind /app/static/output.css static/output.css
COPY --from=templ /workspace .

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o service cmd/service/main.go

# Service
FROM alpine:latest as service
WORKDIR /app
COPY --from=builder /workspace/service /app/service
USER 65532:65532
ENTRYPOINT ["/app/service"]
