# Gofr Trace Collector

The gofr trace collector is a component designed to receive and process traces from applications instrumented with OpenTelemetry. It provides endpoints for receiving traces via HTTP requests and stores them in a database for
further analysis and visualization. It includes a TraceReceiver struct that processes incoming traces and stores 
them in a database.

## Features:

- Accepts trace data through a user-defined handler function.
- Decouples receiving logic from storage implementation.
- Integrates with sql database systems for trace storage.

## API Endpoints
- `POST` **/api/spans**: Endpoint for receiving traces from instrumented applications.
- `GET` **/api/traces**: Endpoint for querying stored traces based on trace IDs or other criteria.

## Further Requirements:

- Support for receiving traces through gRPC protocol also.