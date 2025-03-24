package redis

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.zzfly.net/geminiapi/util/log"
	"go.zzfly.net/geminiapi/util/trace"
)

var (
	// Singleton instance of Redis client
	client     *redis.Client
	clientOnce sync.Once
	// Key for storing API keys in Redis
	apiKeysKey = "gemini"
	// Current position for round-robin selection
	currentPos = 0
	// Mutex for thread-safe access to currentPos
	posMutex = &sync.Mutex{}
)

// GetClient returns a Redis client instance
func GetClient() *redis.Client {
	clientOnce.Do(func() {
		// Get Redis configuration from environment variables
		redisHost := getEnv("REDIS_HOST", "redis")
		redisPort := getEnv("REDIS_PORT", "6379")
		redisPassword := getEnv("REDIS_PASSWORD", "")
		redisDB := getEnvAsInt("REDIS_DB", 0)

		// Create Redis client
		client = redis.NewClient(&redis.Options{
			Addr:     redisHost + ":" + redisPort,
			Password: redisPassword,
			DB:       redisDB,
		})

		// Test connection
		ctx, cancel := context.WithTimeout(trace.WrapTraceInfo(context.Background()), 5*time.Second)
		defer cancel()

		_, err := client.Ping(ctx).Result()
		if err != nil {
			log.Error(ctx, "Failed to connect to Redis: %v", err)
			// Set client to nil so we can fall back to environment variables
			client = nil
		} else {
			log.Info(ctx, "Successfully connected to Redis at %s:%s", redisHost, redisPort)
		}
	})

	return client
}

// InitializeAPIKeys initializes API keys in Redis from environment variable
func InitializeAPIKeys(ctx context.Context) error {
	client := GetClient()
	if client == nil {
		log.Info(ctx, "Redis not available, skipping API keys initialization")
		return nil
	}

	// Check if keys already exist in Redis
	count, err := client.LLen(ctx, apiKeysKey).Result()
	if err != nil {
		return err
	}

	// If keys already exist, don't reinitialize
	if count > 0 {
		log.Info(ctx, "API keys already initialized in Redis (%d keys)", count)
		return nil
	}

	// Get API keys from environment variable
	apiKeys := getAPIKeysFromEnv()
	if len(apiKeys) == 0 {
		log.Info(ctx, "No API keys found in environment variable")
		return nil
	}

	// Add keys to Redis list
	for _, key := range apiKeys {
		_, err := client.RPush(ctx, apiKeysKey, key).Result()
		if err != nil {
			return err
		}
	}

	log.Info(ctx, "Initialized %d API keys in Redis", len(apiKeys))
	return nil
}

// GetAPIKey returns an API key using round-robin selection
func GetAPIKey(ctx context.Context) (string, error) {
	client := GetClient()
	if client == nil {
		// Fallback to environment variable if Redis is not available
		log.Info(ctx, "Redis not available, falling back to environment variable")
		return getFallbackAPIKey(), nil
	}

	// Get the number of keys in the list
	count, err := client.LLen(ctx, apiKeysKey).Result()
	if err != nil {
		log.Error(ctx, "Failed to get API keys count from Redis: %v", err)
		return getFallbackAPIKey(), nil
	}

	if count == 0 {
		log.Info(ctx, "No API keys found in Redis, falling back to environment variable")
		return getFallbackAPIKey(), nil
	}

	// Get next position in round-robin fashion
	posMutex.Lock()

	// Increment position and wrap around if needed
	currentPos = (currentPos + 1) % int(count)
	position := currentPos

	posMutex.Unlock()

	// Get the API key at the current position
	apiKey, err := client.LIndex(ctx, apiKeysKey, int64(position)).Result()
	if err != nil {
		log.Error(ctx, "Failed to get API key from Redis: %v", err)
		return getFallbackAPIKey(), nil
	}

	return apiKey, nil
}

// Helper functions

// getEnv returns the value of an environment variable or a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvAsInt returns the value of an environment variable as an integer or a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// getAPIKeysFromEnv gets API keys from environment variable
func getAPIKeysFromEnv() []string {
	keyStr := os.Getenv("API_KEY")
	if keyStr == "" {
		return []string{}
	}

	// Clean up the string
	keyStr = cleanAPIKeyString(keyStr)

	// Split by comma
	keys := splitAPIKeys(keyStr)

	return keys
}

// getFallbackAPIKey gets a random API key from environment variable
func getFallbackAPIKey() string {
	keys := getAPIKeysFromEnv()
	count := len(keys)
	if count == 0 {
		return ""
	}

	// Use simple round-robin for fallback too
	posMutex.Lock()
	position := currentPos % count
	currentPos = (currentPos + 1) % count
	posMutex.Unlock()

	return keys[position]
}

// cleanAPIKeyString cleans up the API key string
func cleanAPIKeyString(keyStr string) string {
	// Trim spaces, and remove leading/trailing commas
	keyStr = os.ExpandEnv(keyStr)
	return keyStr
}

// splitAPIKeys splits the API key string by comma
func splitAPIKeys(keyStr string) []string {
	// Split by comma and filter out empty strings
	var keys []string
	for _, key := range splitAndTrim(keyStr, ",") {
		if key != "" {
			keys = append(keys, key)
		}
	}
	return keys
}

// splitAndTrim splits a string by a separator and trims spaces from each part
func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	for _, part := range strings.Split(s, sep) {
		part = strings.TrimSpace(part)
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}
