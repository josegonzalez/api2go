package api2go

import (
	"context"
	"database/sql"
	"strings"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// NewSpan is a convenience function for creating a namespaced span
func NewSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	ctx, span := otel.Tracer("").Start(ctx, prefixKey(name))
	return ctx, span
}

// WithAttribute is a convenience function for adding an attribute to a given span
// This will also add the namespaced prefix to the keys
func WithAttribute(span trace.Span, name string, value interface{}) {
	switch val := value.(type) {
	case uuid.UUID:
		span.SetAttributes(attribute.String(prefixKey(name), val.String()))
	case []string:
		span.SetAttributes(attribute.String(prefixKey(name), strings.Join(val, ",")))
	case string:
		span.SetAttributes(attribute.String(prefixKey(name), val))
	case sql.NullString:
		if val.Valid {
			span.SetAttributes(attribute.String(prefixKey(name), val.String))
		} else {
			span.SetAttributes(attribute.String(prefixKey(name), "NULL"))
		}
	case int:
		span.SetAttributes(attribute.Int(prefixKey(name), val))
	case int64:
		span.SetAttributes(attribute.Int64(prefixKey(name), val))
	case int32:
		span.SetAttributes(attribute.Int64(prefixKey(name), int64(val)))
	case uint:
		span.SetAttributes(attribute.Int(prefixKey(name), int(val)))
	case float64:
		span.SetAttributes(attribute.Float64(prefixKey(name), val))
	case bool:
		span.SetAttributes(attribute.Bool(prefixKey(name), val))
	case []uuid.UUID:
		var stringSlice []string
		for _, uuid := range val {
			stringSlice = append(stringSlice, uuid.String())
		}
		span.SetAttributes(attribute.String(prefixKey(name), strings.Join(stringSlice, ",")))
	}
}

func prefixKey(name string) string {
	return strings.TrimSuffix(name, "/")
}
