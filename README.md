# SQL Server CLI

[![Go Reference](https://pkg.go.dev/badge/github.com/binrchq/mssql-cli.svg)](https://pkg.go.dev/github.com/binrchq/mssql-cli)
[![Go Report Card](https://goreportcard.com/badge/github.com/binrchq/mssql-cli)](https://goreportcard.com/report/github.com/binrchq/mssql-cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A standalone SQL Server (MSSQL) interactive CLI client for Go applications.

## Features

- 🚀 Full T-SQL support
- 📝 GO batch separator support
- 🔧 System stored procedures
- ⏱️ Query timing
- 💾 Connection pooling
- 📊 Table format output

## Installation

```bash
go get github.com/binrchq/mssql-cli
```

## Quick Start

```go
package main

import (
    "log"
    "os"
    
    mssqlcli "github.com/binrchq/mssql-cli"
)

func main() {
    cli := mssqlcli.NewCLI(
        os.Stdin,
        "localhost",
        1433,
        "sa",
        "password",
        "mydb",
    )
    
    if err := cli.Connect(); err != nil {
        log.Fatal(err)
    }
    defer cli.Close()
    
    if err := cli.Start(); err != nil {
        log.Fatal(err)
    }
}
```

## Supported Commands

### T-SQL Commands
- All standard SQL Server T-SQL syntax
- `USE <database>` - Switch database

### System Stored Procedures
- `sp_help [table]` - Show table info
- `sp_databases` - List databases
- `sp_tables` - List tables
- `sp_columns <table>` - List columns
- `sp_who` - Show connections

### Batch Separator
Use `GO` to execute a batch of T-SQL statements.

## Special Commands

- `help` - Show help
- `exit`, `quit` - Exit
- `timing` - Toggle timing
- `clear`, `cls` - Clear screen

## Requirements

- Go 1.21 or higher
- SQL Server 2016 or higher

## Dependencies

- [github.com/denisenkom/go-mssqldb](https://github.com/denisenkom/go-mssqldb) - SQL Server driver
- [github.com/chzyer/readline](https://github.com/chzyer/readline) - Readline library

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Author

Maintained by [binrc](https://github.com/binrchq).

## Related Projects

- [mysql-cli](https://github.com/binrchq/mysql-cli) - MySQL CLI
- [postgres-cli](https://github.com/binrchq/postgres-cli) - PostgreSQL CLI
