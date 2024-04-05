# GoFr Trace Exporter

The gofr trace exporter is a component designed to export traces from gofr applications to various 
OpenTelemetry-compatible endpoints. It offers a custom exporter for sending traces to various OpenTelemetry-compatible backends.

## Features:

- Exports trace data to OpenTelemetry-compatible endpoints.
- Supports configuration of the endpoint URL and protocol (currently HTTP).
- Designed for extensibility to support additional protocols (gRPC, etc.).

## Example Usage

- Create a new instance of the Custom Exporter with the desired endpoint.
- Register the exporter with your OpenTelemetry tracer provider.
- Instrument your application to use the registered exporter for exporting traces.


```go
...

    "go.opentelemetry.io/otel/sdk/trace"
    "gofr.dev/gofr-tracer/exporter"
)

... 

func initTracer() {
    exporter := exporter.NewCustomExporter("http://localhost:9411/api/spans")
    
    batcher := sdktrace.NewBatchSpanProcessor(exporter)

    
}

```