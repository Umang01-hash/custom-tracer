package migrations

import (
	"gofr.dev/pkg/gofr/migration"
)

const createIndexTraceID = `create index idx_traces_trace_id
on traces (trace_id);`

const createIndexParentID = `create index idx_spans_parent_id 
on spans (parent_id);`

const createIndexSpanTraceID = `create index idx_spans_trace_id
on spans (trace_id);`

func createIndices() migration.Migrate {
	return migration.Migrate{
		UP: func(d migration.Datasource) error {
			_, err := d.SQL.Exec(createIndexTraceID)
			if err != nil {
				return err
			}

			_, err = d.SQL.Exec(createIndexParentID)
			if err != nil {
				return err
			}

			_, err = d.SQL.Exec(createIndexSpanTraceID)
			if err != nil {
				return err
			}

			return nil
		},
	}
}
