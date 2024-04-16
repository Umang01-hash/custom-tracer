package migrations

import (
	"gofr.dev/pkg/gofr/migration"
)

const createTableTraces = `create table traces
(
    id        bigint unsigned auto_increment
        primary key,
    trace_id  char(32) not null,
    timestamp bigint   not null,
    constraint trace_id
        unique (trace_id)
);
`

const createTableSpans = `create table if not exists spans
(
    id             bigint unsigned auto_increment
        primary key,
    trace_id       char(32)     not null,
    parent_id      varchar(255) null,
    name           varchar(255) not null,
    duration       bigint       not null,
    timestamp      bigint       not null,
    tags           json         null,
    local_endpoint json null
);
`

func createTableEmployee() migration.Migrate {
	return migration.Migrate{
		UP: func(d migration.Datasource) error {
			_, err := d.SQL.Exec(createTableTraces)
			if err != nil {
				return err
			}

			_, err = d.SQL.Exec(createTableSpans)
			if err != nil {
				return err
			}

			return nil
		},
	}
}
