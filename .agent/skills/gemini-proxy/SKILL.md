---
name: Gemini Proxy Development
description: Comprehensive guide for developing, testing, and understanding the gemini-proxy project.
---

# Gemini Proxy Development Skill

This skill provides context and instructions for working with the `gemini-proxy` project.

## Project Overview

`gemini-proxy` is a Go-based proxy service.
- **Module Name**: `go.zzfly.net/geminiapi`
- **Main Entry Point**: `main.go`
- **Port**: Listens on port `80` (internally).

## Directory Structure

- `api/`: Contains the API logic and handlers entry point (`MainHandle`).
- `handler/`: Specific request handlers.
- `util/`: Utility packages.
  - `redis/`: Redis integration for API key management.
  - `log/`: Logging utilities.
  - `trace/`: Tracing utilities.
- `Dockerfile`: Multi-stage Docker build (Go 1.21 builder -> Debian slim).

## Development

### Prerequisites

- Go 1.18+
- Redis (optional for basic compilation, but required for `InitializeAPIKeys` at runtime)

### Running Locally

```bash
# Set necessary environment variables if needed (e.g. for Redis)
go run main.go
```

### Building

```bash
go build -o gemini main.go
```

### Docker

Build and run using Docker:

```bash
docker build -t gemini-proxy .
docker run -p 8080:80 gemini-proxy
```

## Critical Components

### API Key Initialization
The application initializes API keys from Redis on startup.
- Function: `redis.InitializeAPIKeys(ctx)` in `main.go`.
- Failure in Redis initialization is logged but does not crash the server immediately (non-fatal error logged).

### HTTP Server
Uses standard `net/http`.
- Handler: `api.MainHandle`

## Common Tasks

- **Adding a new handler**: Create the handler in `handler/` and register it in `api/`.
- **Modifying Redis logic**: Check `util/redis/`.
