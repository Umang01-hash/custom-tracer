package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"gofr.dev/gofr-tracer/internal/model"
	"gofr.dev/gofr-tracer/internal/processor"
	"gofr.dev/pkg/gofr"
)

func main() {
	app := gofr.New()

	app.POST("/api/spans", PostHandler)
	app.GET("/api/traces", GetHandler)

	app.Run()
}

func GetHandler(c *gofr.Context) (interface{}, error) {
	var spans []model.Span

	traceID := c.Request.Param("traceID")
	if traceID == "" {
		c.Logger.Errorf("traceID missing!")
		return errors.New("missing traceID"), nil
	}

	var id int64

	err := c.SQL.QueryRowContext(c, "SELECT id FROM traces WHERE trace_id = ?", traceID).Scan(&id)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, errors.New(fmt.Sprintf("trace not found for traceID : %v", traceID))
		default:
			return nil, errors.New(fmt.Sprintf("failed to query traces table: %v", err))
		}
	}

	rows, err := c.SQL.QueryContext(c, "SELECT * FROM spans WHERE trace_id = ? ORDER BY timestamp", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			s        model.Span
			tagsJSON []byte
		)
		err := rows.Scan(&s.ID, &s.TraceID, &s.ParentID, &s.Name, &s.Duration, &s.Timestamp, &tagsJSON)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(tagsJSON, &s.Tags); err != nil {
			return nil, err
		}
		spans = append(spans, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return spans, err
}

func PostHandler(c *gofr.Context) (interface{}, error) {
	var spans []model.Span

	err := c.Bind(&spans)
	if err != nil {
		c.Logger.Errorf("error binding request body: %v", err)
		return nil, err
	}

	err = processor.ProcessTraces(c, spans)
	if err != nil {
		c.Logger.Error(err)
		return nil, err
	}

	return "Traces received successfully", err
}
