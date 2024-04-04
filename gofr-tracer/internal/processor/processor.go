package processor

import (
	"encoding/json"
	"gofr.dev/gofr-tracer/internal/model"
	"gofr.dev/pkg/gofr"

	"strings"
)

type TraceReceiver struct{}

func ProcessTraces(c *gofr.Context, spans []model.Span) error {
	txn, err := c.SQL.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if rbErr := txn.Rollback(); rbErr != nil {
				c.Logger.Errorf("failed to rollback transaction: %v", rbErr)
			}
			return
		}
		if commitErr := txn.Commit(); commitErr != nil {
			c.Logger.Errorf("failed to commit transaction: %v", commitErr)
		}
	}()

	var lastTraceID string
	traceMap := make(map[string]uint64)

	for _, span := range spans {
		if span.TraceID != lastTraceID {
			res, err := txn.Exec("INSERT INTO traces (trace_id, timestamp) VALUES (?, ?)",
				span.TraceID, span.Timestamp)
			if err != nil {
				return err
			}

			traceID, err := res.LastInsertId()
			if err != nil {
				return err
			}

			traceMap[span.TraceID] = uint64(traceID)
			lastTraceID = span.TraceID
		}

		tagsJSON, err := json.Marshal(span.Tags)
		if err != nil {
			return err
		}

		escapedJSON := strings.ReplaceAll(string(tagsJSON), "'", "\\'")

		_, err = txn.Exec("INSERT INTO spans (trace_id, parent_id, name, duration, timestamp, tags) VALUES (?, ?, ?, ?, ?, ?)",
			traceMap[span.TraceID], span.ParentID, span.Name, span.Duration, span.Timestamp, escapedJSON)
		if err != nil {
			return err
		}
	}

	return nil
}
