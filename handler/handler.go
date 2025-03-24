package handler

import (
	"context"
	"fmt"
	"go.zzfly.net/geminiapi/util/log"
	"go.zzfly.net/geminiapi/util/redis"
	"go.zzfly.net/geminiapi/util/trace"
	"io"
	"net/http"
	"net/url"
	"time"
)

const PROXY_URL = "https://generativelanguage.googleapis.com"

var httpClient = http.Client{
	Timeout: 30 * time.Second,
}

type SendToGeminiInput struct {
	Url         string
	ContentType string
	APIKey      string
	Payload     io.Reader
	Method      string
	Headers     http.Header
}

// GeminiResponse represents the response from Gemini API
type GeminiResponse struct {
	Body       []byte
	StatusCode int
	Headers    http.Header
}

// SendToGemini sends a request to gemini
func SendToGemini(ctx context.Context, in SendToGeminiInput) (*GeminiResponse, error) {
	// Construir a URL completa usando o caminho da requisição original
	fullUrl := PROXY_URL + in.Url
	apiKey := getAPIKey(in)
	parse, err := url.Parse(fullUrl)
	if err != nil {
		return nil, fmt.Errorf("could not parse url: %w", err)
	}
	// Preservar os parâmetros de consulta originais e adicionar a chave API
	query := parse.Query()
	// Remove any existing key parameter from the URL
	query.Del("key")
	// Add our API key
	query.Set("key", apiKey)
	parse.RawQuery = query.Encode()
	fullUrl = parse.String()

	if len(apiKey) < 8 {
		return nil, fmt.Errorf("invalid api key: %s", apiKey)
	}

	log.Info(ctx, "using api key: %s", maskAPIKey(apiKey))
	log.Info(ctx, "Final request URL: %s", fullUrl)

	// Create a new request with the appropriate method
	req, err := http.NewRequestWithContext(ctx, in.Method, fullUrl, in.Payload)
	if err != nil {
		log.Error(ctx, "Failed to create request: %v", err)
		return nil, fmt.Errorf("could not create request: %w", err)
	}

	// Set content type and copy other relevant headers
	if in.ContentType != "" {
		req.Header.Set("Content-Type", in.ContentType)
	}

	// Copy other headers if provided
	if in.Headers != nil {
		for key, values := range in.Headers {
			if key != "Content-Type" && key != "Host" {
				for _, value := range values {
					req.Header.Add(key, value)
				}
			}
		}
	}

	// Send the request
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Error(ctx, "HTTP request failed: %v", err)
		return nil, fmt.Errorf("could not send request: %w", err)
	}
	log.Info(ctx, "Response status code: %d", resp.StatusCode)

	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(ctx, "Failed to read response body: %v", err)
		return nil, fmt.Errorf("could not read response body: %w", err)
	}

	// Return the complete response
	return &GeminiResponse{
		Body:       body,
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
	}, nil
}

// getAPIKey returns the api key from the input or from Redis/env
func getAPIKey(in SendToGeminiInput) string {
	// If API key is provided in the request, use it
	if in.APIKey != "" {
		return in.APIKey
	}

	// Get API key from Redis using round-robin selection with trace info
	ctx := trace.WrapTraceInfo(context.Background())
	apiKey, err := redis.GetAPIKey(ctx)
	if err != nil {
		log.Error(ctx, "Failed to get API key from Redis: %v", err)
		// Fallback to environment variable is handled inside redis.GetAPIKey
	}

	return apiKey
}

// maskAPIKey masks an API key for secure logging
func maskAPIKey(apiKey string) string {
	if apiKey == "" {
		return "<empty>"
	}
	if len(apiKey) < 8 {
		return "<too_short>"
	}
	return apiKey[0:4] + "****" + apiKey[len(apiKey)-4:]
}
