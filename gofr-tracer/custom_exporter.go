package gofr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go.opentelemetry.io/otel/attribute"
	"net/http"
	"time"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type CustomExporter struct {
	endpoint string
	client   *http.Client
}

func NewCustomExporter(endpoint string) *CustomExporter {
	return &CustomExporter{
		endpoint: endpoint,
		client:   &http.Client{Timeout: 5 * time.Second}, // Set a timeout for HTTP requests
	}
}

type Span struct {
	TraceID   string            `json:"traceId"`
	ID        string            `json:"id"`
	ParentID  string            `json:"parentId,omitempty"`
	Name      string            `json:"name"`
	Timestamp int64             `json:"timestamp"`
	Duration  int64             `json:"duration"`
	Tags      map[string]string `json:"tags,omitempty"`
}

func (e *CustomExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	return e.processSpans(ctx, spans)
}

// Shutdown shuts down the exporter.
func (e *CustomExporter) Shutdown(context.Context) error {
	return nil
}

func (e *CustomExporter) processSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	if len(spans) == 0 {
		return nil
	}

	convertedSpans := convertSpans(spans)

	payload, err := json.Marshal(convertedSpans)
	if err != nil {
		return fmt.Errorf("failed to marshal spans: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response status code: %d", resp.StatusCode)
	}

	return nil
}

func convertSpans(spans []sdktrace.ReadOnlySpan) []Span {
	convertedSpans := make([]Span, 0, len(spans))

	for _, s := range spans {
		convertedSpan := Span{
			TraceID:   s.SpanContext().TraceID().String(),
			ID:        s.SpanContext().SpanID().String(),
			ParentID:  s.Parent().SpanID().String(),
			Name:      s.Name(),
			Timestamp: s.StartTime().UnixNano() / int64(time.Millisecond),
			Duration:  s.EndTime().Sub(s.StartTime()).Milliseconds(),
			Tags:      make(map[string]string, len(s.Attributes())+len(s.Resource().Attributes())),
		}

		for _, kv := range s.Attributes() {
			k, v := attributeToStringPair(kv)
			convertedSpan.Tags[k] = v
		}

		for _, kv := range s.Resource().Attributes() {
			k, v := attributeToStringPair(kv)
			convertedSpan.Tags[k] = v
		}

		convertedSpans = append(convertedSpans, convertedSpan)
	}

	return convertedSpans
}

func attributeToStringPair(kv attribute.KeyValue) (string, string) {
	switch kv.Value.Type() {
	// For slice attributes, serialize as JSON list string.
	case attribute.BOOLSLICE:
		data, _ := json.Marshal(kv.Value.AsBoolSlice())
		return (string)(kv.Key), (string)(data)
	case attribute.INT64SLICE:
		data, _ := json.Marshal(kv.Value.AsInt64Slice())
		return (string)(kv.Key), (string)(data)
	case attribute.FLOAT64SLICE:
		data, _ := json.Marshal(kv.Value.AsFloat64Slice())
		return (string)(kv.Key), (string)(data)
	case attribute.STRINGSLICE:
		data, _ := json.Marshal(kv.Value.AsStringSlice())
		return (string)(kv.Key), (string)(data)
	default:
		return (string)(kv.Key), kv.Value.Emit()
	}
}