package main

import (
	"context"
	"net/http"

	"go.zzfly.net/geminiapi/api"
	"go.zzfly.net/geminiapi/util/log"
	"go.zzfly.net/geminiapi/util/redis"
	"go.zzfly.net/geminiapi/util/trace"
)

func main() {
	// Initialize API keys in Redis with trace information
	ctx := trace.WrapTraceInfo(context.Background())
	err := redis.InitializeAPIKeys(ctx)
	if err != nil {
		log.Error(ctx, "Failed to initialize API keys in Redis: %v", err)
		// Continue execution even if Redis initialization fails
	}

	// Listen on port 8080
	log.Info(ctx, "Starting server on port 8080")
	err = http.ListenAndServe(":8080", http.HandlerFunc(api.MainHandle))
	if err != nil {
		panic(err)
	}
}
