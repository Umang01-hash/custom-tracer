package gofr

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"gofr.dev/pkg/gofr/logging"
)

func Test_ExportSpans(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := logging.NewLogger(logging.INFO)
	exporter := NewCustomExporter("http://example.com", logger)

	tests := []struct {
		desc  string
		spans []sdktrace.ReadOnlySpan
	}{
		{"Empty Spans Slice", []sdktrace.ReadOnlySpan{}},
		{"Success case", provideSampleSpan(t)},
	}

	for i, tc := range tests {
		err := exporter.ExportSpans(context.Background(), tc.spans)

		assert.Nil(t, err, "TEST[%d], Failed.\n%s", i, tc.desc)
	}
}

func Test_ExportSpansError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	server.Close()

	exporter := NewCustomExporter(server.URL, logging.NewLogger(logging.INFO))

	err := exporter.ExportSpans(context.Background(), provideSampleSpan(t))
	assert.Error(t, err, "Expected error for failed request")
}

func provideSampleSpan(t *testing.T) []sdktrace.ReadOnlySpan {
	tp := sdktrace.NewTracerProvider()

	defer func(tp *sdktrace.TracerProvider, ctx context.Context) {
		err := tp.Shutdown(ctx)
		if err != nil {
			t.Error(err)
		}
	}(tp, context.Background())

	otel.SetTracerProvider(tp)

	tracer := otel.Tracer("test-tracer")

	_, span := tracer.Start(context.Background(), "test-span")
	span.End()

	ro := span.(sdktrace.ReadOnlySpan)

	return []sdktrace.ReadOnlySpan{ro}
}

func Test_attributeToStringPair(t *testing.T) {
	tests := []struct {
		name           string
		keyValue       attribute.KeyValue
		expectedKey    string
		expectedValue  string
		expectedErrMsg string
	}{
		{
			name:           "BoolSlice",
			keyValue:       attribute.BoolSlice("boolKey", []bool{true, false}),
			expectedKey:    "boolKey",
			expectedValue:  `[true,false]`,
			expectedErrMsg: "",
		},
		{
			name:           "Int64Slice",
			keyValue:       attribute.Int64Slice("int64Key", []int64{1, 2, 3}),
			expectedKey:    "int64Key",
			expectedValue:  `[1,2,3]`,
			expectedErrMsg: "",
		},
		{
			name:           "Float64Slice",
			keyValue:       attribute.Float64Slice("float64Key", []float64{1.1, 2.2, 3.3}),
			expectedKey:    "float64Key",
			expectedValue:  `[1.1,2.2,3.3]`,
			expectedErrMsg: "",
		},
		{
			name:           "StringSlice",
			keyValue:       attribute.StringSlice("stringKey", []string{"a", "b", "c"}),
			expectedKey:    "stringKey",
			expectedValue:  `["a","b","c"]`,
			expectedErrMsg: "",
		},
		{
			name:           "OtherTypes",
			keyValue:       attribute.String("otherKey", "otherValue"),
			expectedKey:    "otherKey",
			expectedValue:  "otherValue",
			expectedErrMsg: "",
		},
	}

	for _, tt := range tests {
		key, value := attributeToStringPair(tt.keyValue)
		assert.Equal(t, tt.expectedKey, key, "Key mismatch")
		assert.Equal(t, tt.expectedValue, value, "Value mismatch")

	}
}
