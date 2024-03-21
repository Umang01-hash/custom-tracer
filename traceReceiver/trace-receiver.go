package traceReceiver

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
}

func (tr *traceReceiver) Start(ctx context.Context, host component.Host) error {
	tr.host = host
	ctx = context.Background()
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

	interval, _ := time.ParseDuration(tr.config.Interval)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				//err := tr.processTraces(ctx, nil)
				//if err != nil {
				//	tr.logger.Error(err.Error())
				//}
				tr.logger.Info("I should start processing traces now!")
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (tr *traceReceiver) Shutdown(ctx context.Context) error {
	tr.cancel()

	return nil
}

//func (tr *traceReceiver) processTraces(ctx context.Context, td *ptrace.Traces) error {
//	fmt.Println(td)
//	for i := 0; i < td.ResourceSpans().Len(); i++ {
//		rs := td.ResourceSpans().At(i)
//
//		for j := 0; j < rs.ScopeSpans().Len(); j++ {
//			ilss := rs.ScopeSpans().At(j)
//
//			// Iterate over the spans in the scope span
//			for k := 0; k < ilss.Spans().Len(); k++ {
//				span := ilss.Spans().At(k)
//
//				spanJSON, err := json.Marshal(span)
//				if err != nil {
//					return err
//				}
//
//				query := `INSERT INTO traces (data) VALUES (?)`
//				_, err = tr.db.ExecContext(ctx, query, string(spanJSON))
//				if err != nil {
//					return err
//				}
//			}
//
//		}
//	}
//
//	return nil
//}
