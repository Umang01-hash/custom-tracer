package traceReceiver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.uber.org/zap"
	"net/http"

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
	Tags      map[string]string `json:"tags,omitempty"`
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

	tr.mux.HandleFunc("/api/v2/traces", tr.getTracesByTraceID)

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
	txn, err := tr.db.Begin()
	if err != nil {
		tr.logger.Error("error in initiating transaction to store trace")
		return err
	}

	defer func() {
		if err != nil {
			tr.logger.Error("error occurred, rolling back transaction")
			if rbErr := txn.Rollback(); rbErr != nil {
				tr.logger.Error("failed to rollback transaction", zap.Error(rbErr))
			}
			return
		}
		if commitErr := txn.Commit(); commitErr != nil {
			tr.logger.Error("failed to commit transaction", zap.Error(commitErr))
		}
	}()

	var (
		lastTraceID string
		traceMap    = make(map[string]uint64)
	)

	for _, span := range spans {
		if span.TraceID != lastTraceID {
			res, err := txn.ExecContext(ctx, "INSERT INTO traces (trace_id, timestamp) VALUES (?, ?)",
				span.TraceID, span.Timestamp)
			if err != nil {
				tr.logger.Error(err.Error())
				return err
			}

			traceID, err := res.LastInsertId()
			if err != nil {
				tr.logger.Error(err.Error())
				return err
			}

			traceMap[span.TraceID] = uint64(traceID)
			lastTraceID = span.TraceID
		}

		tagsJSON, err := json.Marshal(span.Tags)
		if err != nil {
			tr.logger.Error(err.Error())
			return err
		}

		_, err = txn.ExecContext(ctx, "INSERT INTO spans (trace_id, parent_id, name, duration, timestamp, tags) VALUES (?, ?, ?, ?, ?, ?)",
			traceMap[span.TraceID], span.ParentID, span.Name, span.Duration, span.Timestamp, tagsJSON)
		if err != nil {
			tr.logger.Error(err.Error())
			return err
		}
	}

	return nil
}

func (tr *traceReceiver) getTracesByTraceID(w http.ResponseWriter, r *http.Request) {
	traceID := r.URL.Query().Get("traceID")
	if traceID == "" {
		http.Error(w, "traceID parameter is required", http.StatusBadRequest)
		return
	}

	var id int64
	err := tr.db.QueryRowContext(r.Context(), "SELECT id FROM traces WHERE trace_id = ?", traceID).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "trace not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("failed to query traces table: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// Query the database for spans with trace_id and order by timestamp
	rows, err := tr.db.QueryContext(r.Context(), "SELECT * FROM spans WHERE trace_id = ? ORDER BY timestamp", id)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to query spans: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var spans []span

	for rows.Next() {
		var (
			s        span
			tagsJSON []byte
		)
		err := rows.Scan(&s.ID, &s.TraceID, &s.ParentID, &s.Name, &s.Duration, &s.Timestamp, &tagsJSON)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to scan span row: %v", err), http.StatusInternalServerError)
			return
		}

		if err := json.Unmarshal(tagsJSON, &s.Tags); err != nil {
			http.Error(w, fmt.Sprintf("failed to scan span row: %v", err), http.StatusInternalServerError)
			return
		}
		spans = append(spans, s)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("error iterating over span rows: %v", err), http.StatusInternalServerError)
		return
	}

	// Encode spans as JSON and write response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(spans); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode spans: %v", err), http.StatusInternalServerError)
		return
	}
}
