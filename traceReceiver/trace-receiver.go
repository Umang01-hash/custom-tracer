package traceReceiver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.uber.org/zap"

	_ "github.com/go-sql-driver/mysql"
)

type traceReceiver struct {
	host         component.Host
	cancel       context.CancelFunc
	logger       *zap.Logger
	nextConsumer consumer.Traces
	config       *Config
	db           *sql.DB
	mux          *http.ServeMux
}

type span struct {
	TraceID   string            `json:"traceId"`
	ID        string            `json:"id"`
	ParentID  string            `json:"parentId,omitempty"`
	Name      string            `json:"name"`
	Timestamp int64             `json:"timestamp"`
	Duration  int64             `json:"duration"`
	Tags      map[string]string `json:"tags"`
}

func (tr *traceReceiver) Start(ctx context.Context, host component.Host) error {
	tr.host = host
	ctx, tr.cancel = context.WithCancel(ctx)

	var err error
	tr.db, err = sql.Open("mysql", "root:password@tcp(localhost:2001)/test")
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	err = tr.db.Ping()
	if err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	tr.mux = http.NewServeMux()
	tr.mux.HandleFunc("/api/v2/spans", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "Method %s not allowed", r.Method)
			return
		}

		var traces []span
		if err := json.NewDecoder(r.Body).Decode(&traces); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Error decoding traces: %v", err)
			return
		}

		// Process the received traces
		if err := tr.processTraces(ctx, traces); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Error processing traces: %v", err)
			return
		}

		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, "Traces received successfully")
	})

	address := fmt.Sprintf(":%v", tr.config.Port)

	go func() {
		tr.logger.Info("Starting trace receiver on HTTP", zap.String("address", address))
		if err := http.ListenAndServe(address, tr.mux); !errors.Is(err, http.ErrServerClosed) {
			tr.logger.Error("Failed to start HTTP server", zap.Error(err))
		}
	}()

	return nil
}

func (tr *traceReceiver) Shutdown(ctx context.Context) error {
	tr.cancel()

	return nil
}

func (tr *traceReceiver) processTraces(ctx context.Context, spans []span) error {
	for _, span := range spans {
		spanJSON, err := json.Marshal(span)
		if err != nil {
			return err
		}

		query := `INSERT INTO traces (data) VALUES (?)`
		_, err = tr.db.ExecContext(ctx, query, string(spanJSON))
		if err != nil {
			return err
		}
	}

	return nil
}
