package api

import (
	"context"
	"encoding/json"
	"fmt"
	"go.zzfly.net/geminiapi/handler"
	"go.zzfly.net/geminiapi/util/log"
	"go.zzfly.net/geminiapi/util/trace"
	"net/http"
)

const ctxKeyRespWriter = "respWriter"

type Response struct {
	Code int    `json:"code"`
	Body string `json:"body"`
}

func MainHandle(w http.ResponseWriter, r *http.Request) {
	ctx := getCtx(r, w)

	// Log request details
	log.Info(ctx, "Received request - Method: %s, Path: %s", r.Method, r.URL.Path)
	log.Info(ctx, "Request headers: %v", logHeaders(r.Header))
	log.Info(ctx, "Request query parameters: %v", r.URL.Query())

	// Get the API key from query parameters
	apiKey := getFromQuery(r, "key", "")
	log.Info(ctx, "API Key from query: %s", maskAPIKey(apiKey))

	// Construct the full URL path with query parameters
	fullPath := r.URL.Path
	if r.URL.RawQuery != "" {
		fullPath += "?" + r.URL.RawQuery
	}

	in := handler.SendToGeminiInput{
		Url:         fullPath,
		ContentType: r.Header.Get("Content-Type"),
		APIKey:      apiKey,
		Payload:     r.Body,
		Method:      r.Method,
		Headers:     r.Header,
	}

	log.Info(ctx, "start request: %s", in.Url)
	geminiResp, err := handler.SendToGemini(ctx, in)
	if err != nil {
		log.Error(ctx, "send to gemini err: %v", err)
		doStdResponse(ctx, Response{Code: 500, Body: "Internal server error. details: " + err.Error()})
		return
	}

	log.Info(ctx, "end request: %s", in.Url)
	doGeminiResponse(ctx, geminiResp)
}

// getFromHeader returns the value of the header key or the default value
func getFromHeader(r *http.Request, key string, defaultV string) string {
	value := r.Header.Get(key)
	if value == "" {
		return defaultV
	}

	return value
}

// getFromQuery returns the value of the query key or the default value
func getFromQuery(r *http.Request, key string, defaultV string) string {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultV
	}

	return value
}

// getCtx returns a context with the response writer and a trace id
func getCtx(r *http.Request, w http.ResponseWriter) context.Context {
	ctx := r.Context()
	ctx = context.WithValue(ctx, ctxKeyRespWriter, w)
	return trace.WrapTraceInfo(ctx)
}

// doStdResponse writes a self response to the response writer
func doStdResponse(ctx context.Context, resp Response) {
	marshal, err := json.Marshal(resp)
	if err != nil {
		log.Error(ctx, "could not marshal response: %v", err)
	}

	w := ctx.Value(ctxKeyRespWriter).(http.ResponseWriter)
	w.Header().Set("X-Trace-Id", trace.GetTraceId(ctx))
	w.Header().Set("X-Content-From", "agent")

	_, err = fmt.Fprintf(w, string(marshal))
	if err != nil {
		log.Error(ctx, "could not write std response: %v", err)
	}
}

// doGeminiResponse writes a gemini response to the response writer
func doGeminiResponse(ctx context.Context, resp *handler.GeminiResponse) {
	w := ctx.Value(ctxKeyRespWriter).(http.ResponseWriter)

	// Set standard headers
	w.Header().Set("X-Trace-Id", trace.GetTraceId(ctx))
	w.Header().Set("X-Content-From", "gemini")

	// Copy all headers from the Gemini API response
	for key, values := range resp.Headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set content type if not already set
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}

	// Set the status code from the Gemini API response
	w.WriteHeader(resp.StatusCode)

	// Write the response body
	_, err := w.Write(resp.Body)
	if err != nil {
		log.Error(ctx, "could not write gemini response: %v", err)
	}
}

// logHeaders returns a map of headers for logging
func logHeaders(headers http.Header) map[string]string {
	result := make(map[string]string)
	for name, values := range headers {
		if len(values) > 0 {
			result[name] = values[0]
		}
	}
	return result
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
