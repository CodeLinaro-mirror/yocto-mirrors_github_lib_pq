pq is a Go PostgreSQL driver for database/sql.

All [maintained versions of PostgreSQL] are supported. Older versions may work,
but this is not tested. [API docs].

[maintained versions of PostgreSQL]: https://www.postgresql.org/support/versioning
[API docs]: https://pkg.go.dev/github.com/lib/pq

Connecting
----------
Use the `postgres` driver name in the `sql.Open()` call:

```go
package main

import (
    "database/sql"
    "log"

    // Import to register the driver.
    _ "github.com/lib/pq"
)

func main() {
    // Or as URL: postgresql://localhost/pqgo
    db, err := sql.Open("postgres", "host=localhost dbname=pqgo")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // db.Open() only creates a connection pool, and doesn't actually establish
    // a connection to the database. To ensure the connection works you need to
    // do *something* with a connection.
    err = db.Ping()
    if err != nil {
        log.Fatal(err)
    }
}
```

You can also use `pq.Config`:

```go
cfg := pq.Config{
    Host: "localhost",
    Port: 5432,
    User: "pqgo",
}
// Or: create a new Config from the defaults, environment, and DSN.
// cfg, err := pq.NewConfig("host=postgres dbname=pqgo")
// if err != nil {
//     log.Fatal(err)
// }

c, err := pq.NewConnectorConfig(cfg)
if err != nil {
    log.Fatal(err)
}

// Create connection pool.
db := sql.OpenDB(c)
defer db.Close()

// Make sure it works.
err = db.Ping()
if err != nil {
    log.Fatal(err)
}
```

The DSN works identical to PostgreSQL's libpq, and most parameters are supported
by pq and should behave identical. Both key=value and postgres:// URL-style
connection strings are supported. See the doc comments on the [Config struct]
for the full list and documentation.

The most notable difference is that you can use any [run-time parameter] (such
as search_path or work_mem) in the connection string. This is different from
libpq, which does not allow run-time parameters in the connection string,
instead requiring you to supply them in the options parameter.

For example:

    sql.Open("postgres", "dbname=pqgo work_mem=100kB =xyz")

The libpq way (which also works in pq) would be to use options= like so:

    sql.Open("postgres", "dbname=pqgo options='-c work_mem=100kB -c search_path=xyz'")

[Config struct]: https://pkg.go.dev/github.com/lib/pq#Config

[run-time parameter]: http://www.postgresql.org/docs/current/static/runtime-config.html

Errors
------
Errors from PostgreSQL are returned as [pq.Error]; the Error() string contains
the error message and error code:

    pq: duplicate key value violates unique constraint "users_lower_idx" (23505)

The ErrorWithDetail() string also contains the DETAIL and CONTEXT fields, if
present. For example for the above error this helpfully contains the duplicate
value:

    ERROR:   duplicate key value violates unique constraint "users_lower_idx" (23505)
    DETAIL:  Key (lower(email))=(a@example.com) already exists.

Or for an invalid syntax error like this:

    pq: invalid input syntax for type json (22P02)

It contains the context where this error occurred:

    ERROR:   invalid input syntax for type json (22P02)
    DETAIL:  Token "asd" is invalid.
    CONTEXT: line 5, column 8:

          3 | 'def',
          4 | 123,
          5 | 'foo', 'asd'::jsonb
                     ^

[pq.Error]: https://pkg.go.dev/github.com/lib/pq#Error

PostgreSQL features
-------------------

### Bulk imports with `COPY [..] FROM STDIN`
You can perform bulk imports by preparing a `COPY [..] FROM STDIN` statement
inside a transaction. The returned `sql.Stmt` can then be repeatedly executed to
copy data. After all data has been processed you should call Exec() once with no
arguments to flush all buffered data.

See [the docs] and [example]. XXX
Example_copyFromStdin

### Using custom `tls.Config`


ExampleRegisterTLSConfig




Caveats
-------

### LastInsertId

sql.Result.LastInsertId() is not supported, because the PostgreSQL protocol does
not have this facility. Use a query with `returning [cols]` instead:

    db.QueryRow(`insert into tbl [..] returning id_col`).Scan(..)
    // Or multiple rows:
    db.Query(`insert into tbl (row1), (row2) returning id_col`)

This will also work in SQLite and MariaDB with the same syntax. MS-SQL and
Oracle have a similar facility (with a different syntax).


### timestamps

XXX

- uses databases's timezone
- timestamp without timezone -> NOT UTC!
- date without timezone -> does use UTC!

### bytea with copy

Development
-----------
### Running tests
Tests need to be run against a PostgreSQL database; you can use Docker compose
to start one:

    docker compose up -d

This starts the latest PostgreSQL; use `docker compose up -d pg«v»` to start a
different version.

In addition, your `/etc/hosts` currently needs an entry:

    127.0.0.1 postgres postgres-invalid

Or you can use any other PostgreSQL instance; see
`testdata/init/docker-entrypoint-initdb.d` for the required setup. You can use
the standard `PG*` environment variables to control the connection details; it
uses the following defaults:

    PGHOST=localhost
    PGDATABASE=pqgo
    PGUSER=pqgo
    PGSSLMODE=disable
    PGCONNECT_TIMEOUT=20

`PQTEST_BINARY_PARAMETERS` can be used to add `binary_parameters=yes` to all
connection strings:

    PQTEST_BINARY_PARAMETERS=1 go test

Tests can be run against pgbouncer with:

    docker compose up -d pgbouncer pg18
    PGPORT=6432 go test ./...

and pgpool with:

    docker compose up -d pgpool pg18
    PGPORT=7432 go test ./...

### Protocol debug output
You can use PQGO_DEBUG=1 to make the driver print the communication with
PostgreSQL to stderr; this works anywhere (test or applications) and can be
useful to debug protocol problems.

For example:

    % PQGO_DEBUG=1 go test -run TestSimpleQuery
    CLIENT → Startup                 69  "\x00\x03\x00\x00database\x00pqgo\x00user [..]"
    SERVER ← (R) AuthRequest          4  "\x00\x00\x00\x00"
    SERVER ← (S) ParamStatus         19  "in_hot_standby\x00off\x00"
    [..]
    SERVER ← (Z) ReadyForQuery        1  "I"
             START conn.query
             START conn.simpleQuery
    CLIENT → (Q) Query                9  "select 1\x00"
    SERVER ← (T) RowDescription      29  "\x00\x01?column?\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x17\x00\x04\xff\xff\xff\xff\x00\x00"
    SERVER ← (D) DataRow              7  "\x00\x01\x00\x00\x00\x011"
             END conn.simpleQuery
             END conn.query
    SERVER ← (C) CommandComplete      9  "SELECT 1\x00"
    SERVER ← (Z) ReadyForQuery        1  "I"
    CLIENT → (X) Terminate            0  ""
    PASS
    ok      github.com/lib/pq       0.010s
