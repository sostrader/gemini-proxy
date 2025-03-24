package trace

import (
	"context"
	"github.com/google/uuid"
)

const ctxKeyTraceId = "traceId"

// GetTraceId returns the trace id from the context
func GetTraceId(ctx context.Context) string {
	val := ctx.Value(ctxKeyTraceId)
	if val == nil {
		return "no-trace-id"
	}
	return val.(string)
}

// WrapTraceInfo wraps the context with a trace id
func WrapTraceInfo(ctx context.Context) context.Context {
	s, _ := uuid.NewUUID()
	return context.WithValue(ctx, ctxKeyTraceId, s.String())
}
