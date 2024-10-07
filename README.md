# GORM Libsql Driver

[![CI](https://github.com/bryanvaz/gorm-libsql/actions/workflows/ci.yml/badge.svg)](https://github.com/bryanvaz/gorm-libsql/actions)

## Usage

```go
import (
  "github.com/bryanvaz/gorm-libsql"
  "gorm.io/gorm"
)

// Local File
db, err := gorm.Open(libsql.Open("file:my.db"), &gorm.Config{})

// Remote Turso Database
db, err := gorm.Open(libsql.Open("libsql://[DATABASE].turso.io?authToken=[TOKEN]"), &gorm.Config{})

// Remote Libsql Database
db, err := gorm.Open(libsql.Open("http://<hostname>:<port>"), &gorm.Config{})

// Embedded Replica (db stored in temp dir)
import "database/sql"

conn, err := libsql.NewEmbeddedReplicaConnector("/tmp/local_replica.db", "libsql://[DATABASE].turso.io?authToken=[TOKEN]")
if err != nil {
  panic(err)
}
db, err := gorm.Open(libsql.New(libsql.Config{ Conn: sql.OpenDB(conn) }), &gorm.Config{})
// To manually sync replica
replicated, err := conn.Sync()
```

## Description

This driver properly wraps the CGO version of the libsql client, allowing you to use both local and remote databases
as well as embedded replicas. The CGO version is required to use the embedded replica feature.

It is possible to use the pure go libsql client `github.com/tursodatabase/libsql-client-go/libsql`
using the custom libsql connector method. However, as this driver still includes
the CGO version of the libsql client, you may still run into issues.

**NOTE: This driver cannot be used in parallel with any other libsql or sqlite3 CGO drivers.
The symbols will conflict**

## Development Notes

* Since libsql aims to be a drop-in replacement for sqlite, the goal of this wrapper
  is to minimize the amount of code that needs to be written to support libsql, beyond
  the base `gorm-sqlite` driver.
* In order to minimize the maintenance burden and version churn, only the most basic connection scenario is
  covered by the built in Open and New methods. For production use, you should initialize
  a libsql connection outside of the gorm driver, then use this wrapper to provide
  gorm ergonomics around your connection.

## Testing

To run tests locally, run `docker compose up` to stand up a local libsql database.

## Acknowledgements

This driver is largely based on the [`go-gorm/sqlite`](https://github.com/go-gorm/sqlite) driver
by [jinzhu](https://github.com/jinzhu).

## Author

Bryan Vaz - https://github.com/bryanvaz
