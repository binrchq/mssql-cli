package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
)

// Terminal 终端接口，用于输入输出
type Terminal interface {
	io.Reader
	io.Writer
}

// CLI SQL Server 交互式命令行客户端
type CLI struct {
	term          Terminal
	host          string
	port          int
	username      string
	password      string
	database      string
	db            *sql.DB
	reader        *Reader
	serverInfo    ServerInfo
	timingEnabled bool
	maxRows       int
}

// ServerInfo SQL Server 服务器信息
type ServerInfo struct {
	Version       string
	ProductLevel  string
	Edition       string
	ServerName    string
}

// NewCLI 创建新的 SQL Server CLI 实例
func NewCLI(term Terminal, host string, port int, username, password, database string) *CLI {
	return &CLI{
		term:     term,
		host:     host,
		port:     port,
		username: username,
		password: password,
		database: database,
		reader:   NewReader(term),
		maxRows:  1000,
	}
}

// Connect 连接到 SQL Server
func (c *CLI) Connect() error {
	connStr := fmt.Sprintf("server=%s;port=%d;user id=%s;password=%s;database=%s;connection timeout=10",
		c.host, c.port, c.username, c.password, c.database)

	var err error
	c.db, err = sql.Open("sqlserver", connStr)
	if err != nil {
		return err
	}

	c.db.SetMaxOpenConns(10)
	c.db.SetMaxIdleConns(5)
	c.db.SetConnMaxLifetime(time.Hour)

	if err := c.db.Ping(); err != nil {
		c.db.Close()
		return err
	}

	c.fetchServerInfo()
	c.showWelcome()

	return nil
}

// fetchServerInfo 获取服务器信息
func (c *CLI) fetchServerInfo() {
	c.db.QueryRow("SELECT @@VERSION").Scan(&c.serverInfo.Version)
	c.db.QueryRow("SELECT @@SERVERNAME").Scan(&c.serverInfo.ServerName)
	c.db.QueryRow("SELECT SERVERPROPERTY('ProductLevel')").Scan(&c.serverInfo.ProductLevel)
	c.db.QueryRow("SELECT SERVERPROPERTY('Edition')").Scan(&c.serverInfo.Edition)
}

// showWelcome 显示欢迎信息
func (c *CLI) showWelcome() {
	fmt.Fprintf(c.term, "Microsoft SQL Server\n")
	fmt.Fprintf(c.term, "Server: %s:%d\n", c.host, c.port)
	fmt.Fprintf(c.term, "Edition: %s %s\n", c.serverInfo.Edition, c.serverInfo.ProductLevel)
	fmt.Fprintf(c.term, "\n")
}

// Start 启动交互式命令行
func (c *CLI) Start() error {
	for {
		prompt := c.getPrompt()
		fmt.Fprintf(c.term, prompt)

		sqlStr := c.readMultiLine()
		if sqlStr == "" {
			continue
		}

		sqlStr = strings.TrimSpace(sqlStr)

		if c.handleSpecialCommand(sqlStr) {
			if strings.ToLower(sqlStr) == "exit" || strings.ToLower(sqlStr) == "quit" {
				return nil
			}
			continue
		}

		c.executeSQL(sqlStr)
	}
}

// getPrompt 获取提示符
func (c *CLI) getPrompt() string {
	return fmt.Sprintf("%s> ", c.database)
}

// readMultiLine 读取多行 SQL
func (c *CLI) readMultiLine() string {
	var lines []string

	for {
		line, err := c.reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				return ""
			}
			return ""
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" && len(lines) == 0 {
			return ""
		}

		lines = append(lines, line)

		// SQL Server 使用 GO 作为批处理分隔符
		if strings.ToUpper(trimmed) == "GO" {
			// 移除最后的 GO
			lines = lines[:len(lines)-1]
			break
		}

		// 或者以分号结束
		if strings.HasSuffix(trimmed, ";") {
			break
		}

		fmt.Fprintf(c.term, "  -> ")
	}

	result := strings.Join(lines, "\n")
	result = strings.TrimSuffix(strings.TrimSpace(result), ";")
	return result
}

// handleSpecialCommand 处理特殊命令
func (c *CLI) handleSpecialCommand(cmd string) bool {
	cmdLower := strings.ToLower(strings.TrimSpace(cmd))

	if cmdLower == "exit" || cmdLower == "quit" {
		fmt.Fprintf(c.term, "\n")
		return true
	}

	if cmdLower == "help" {
		c.showHelp()
		return true
	}

	if cmdLower == "timing" {
		c.timingEnabled = !c.timingEnabled
		if c.timingEnabled {
			fmt.Fprintf(c.term, "Timing enabled\n")
		} else {
			fmt.Fprintf(c.term, "Timing disabled\n")
		}
		return true
	}

	if cmdLower == "clear" || cmdLower == "cls" {
		fmt.Fprintf(c.term, "\033[2J\033[H")
		return true
	}

	// SQL Server 特有命令
	if strings.HasPrefix(cmdLower, "use ") {
		parts := strings.Fields(cmd)
		if len(parts) >= 2 {
			c.useDatabase(parts[1])
		}
		return true
	}

	return false
}

// executeSQL 执行 SQL 语句
func (c *CLI) executeSQL(sqlStr string) {
	startTime := time.Now()

	sqlStr = strings.TrimSpace(sqlStr)
	if sqlStr == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if isQuery(sqlStr) {
		c.executeQuery(ctx, sqlStr, startTime)
	} else {
		c.executeCommand(ctx, sqlStr, startTime)
	}
}

// executeQuery 执行查询语句
func (c *CLI) executeQuery(ctx context.Context, sqlStr string, startTime time.Time) {
	rows, err := c.db.QueryContext(ctx, sqlStr)
	if err != nil {
		c.printError(err)
		return
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	colTypes, _ := rows.ColumnTypes()

	c.displayTable(rows, cols, colTypes, startTime)
}

// displayTable 以表格形式显示结果
func (c *CLI) displayTable(rows *sql.Rows, cols []string, colTypes []*sql.ColumnType, startTime time.Time) {
	colWidths := make([]int, len(cols))
	for i, col := range cols {
		colWidths[i] = len(col)
		if colWidths[i] < 4 {
			colWidths[i] = 4
		}
		if colWidths[i] > 50 {
			colWidths[i] = 50
		}
	}

	var allRows [][]string
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		valPtrs := make([]interface{}, len(cols))
		for i := range vals {
			valPtrs[i] = &vals[i]
		}
		rows.Scan(valPtrs...)

		rowStrs := make([]string, len(vals))
		for i, v := range vals {
			if v == nil {
				rowStrs[i] = "NULL"
			} else {
				switch val := v.(type) {
				case []byte:
					rowStrs[i] = string(val)
				case time.Time:
					rowStrs[i] = val.Format("2006-01-02 15:04:05")
				default:
					rowStrs[i] = fmt.Sprintf("%v", v)
				}
			}

			if len(rowStrs[i]) > colWidths[i] {
				if len(rowStrs[i]) > 50 {
					colWidths[i] = 50
					rowStrs[i] = rowStrs[i][:47] + "..."
				} else {
					colWidths[i] = len(rowStrs[i])
				}
			}
		}
		allRows = append(allRows, rowStrs)

		if len(allRows) >= c.maxRows {
			break
		}
	}

	c.printSeparator(colWidths)
	fmt.Fprintf(c.term, "| ")
	for i, col := range cols {
		fmt.Fprintf(c.term, "%-*s | ", colWidths[i], col)
	}
	fmt.Fprintf(c.term, "\n")
	c.printSeparator(colWidths)

	for _, row := range allRows {
		fmt.Fprintf(c.term, "| ")
		for i, val := range row {
			fmt.Fprintf(c.term, "%-*s | ", colWidths[i], val)
		}
		fmt.Fprintf(c.term, "\n")
	}
	c.printSeparator(colWidths)

	rowCount := len(allRows)
	if rowCount == 0 {
		fmt.Fprintf(c.term, "(0 rows affected)\n")
	} else if rowCount == 1 {
		fmt.Fprintf(c.term, "(1 row affected)\n")
	} else {
		fmt.Fprintf(c.term, "(%d rows affected)\n", rowCount)
	}

	if c.timingEnabled {
		elapsed := time.Since(startTime).Seconds()
		fmt.Fprintf(c.term, "Time: %.3f sec\n", elapsed)
	}
	fmt.Fprintf(c.term, "\n")
}

// printSeparator 打印表格分隔线
func (c *CLI) printSeparator(colWidths []int) {
	fmt.Fprintf(c.term, "+")
	for _, width := range colWidths {
		fmt.Fprintf(c.term, "%s+", strings.Repeat("-", width+2))
	}
	fmt.Fprintf(c.term, "\n")
}

// executeCommand 执行非查询语句
func (c *CLI) executeCommand(ctx context.Context, sqlStr string, startTime time.Time) {
	result, err := c.db.ExecContext(ctx, sqlStr)
	if err != nil {
		c.printError(err)
		return
	}

	affected, _ := result.RowsAffected()
	elapsed := time.Since(startTime).Seconds()

	if affected == 0 {
		fmt.Fprintf(c.term, "(0 rows affected)\n")
	} else if affected == 1 {
		fmt.Fprintf(c.term, "(1 row affected)\n")
	} else {
		fmt.Fprintf(c.term, "(%d rows affected)\n", affected)
	}

	if c.timingEnabled {
		fmt.Fprintf(c.term, "Time: %.3f sec\n", elapsed)
	}
	fmt.Fprintf(c.term, "\n")
}

// useDatabase 切换数据库
func (c *CLI) useDatabase(dbName string) {
	_, err := c.db.Exec(fmt.Sprintf("USE [%s]", dbName))
	if err != nil {
		fmt.Fprintf(c.term, "Error: %v\n", err)
		return
	}
	c.database = dbName
	fmt.Fprintf(c.term, "Changed database context to '%s'.\n", dbName)
}

// printError 打印错误信息
func (c *CLI) printError(err error) {
	fmt.Fprintf(c.term, "Msg 50000, Level 16, State 1\n")
	fmt.Fprintf(c.term, "%s\n\n", err.Error())
}

// showHelp 显示帮助信息
func (c *CLI) showHelp() {
	help := `
SQL Server Commands
===================

General:
  help                    Show this help
  exit, quit              Exit
  clear, cls              Clear screen
  timing                  Toggle timing
  GO                      Execute batch (SQL Server style)

Database:
  USE <database>          Change database

Query Commands:
  SELECT ...              Query data
  INSERT ...              Insert data
  UPDATE ...              Update data
  DELETE ...              Delete data
  
Schema Commands:
  CREATE TABLE ...        Create table
  ALTER TABLE ...         Alter table
  DROP TABLE ...          Drop table
  CREATE INDEX ...        Create index
  
System Stored Procedures:
  sp_help [table]         Show table info
  sp_databases            List databases
  sp_tables               List tables
  sp_columns <table>      List columns
  sp_who                  Show active connections
  
Information Schema:
  SELECT * FROM INFORMATION_SCHEMA.TABLES
  SELECT * FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = 'table'
  
System Views:
  SELECT * FROM sys.databases
  SELECT * FROM sys.tables
  SELECT * FROM sys.columns WHERE object_id = OBJECT_ID('table')

For more information: https://docs.microsoft.com/sql/
`
	fmt.Fprintf(c.term, help)
}

// Close 关闭数据库连接
func (c *CLI) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// isQuery 判断是否是查询语句
func isQuery(sqlStr string) bool {
	upper := strings.ToUpper(strings.TrimSpace(sqlStr))

	queryPrefixes := []string{
		"SELECT", "SHOW", "WITH", "EXPLAIN",
		"EXEC sp_help", "EXEC sp_databases", "EXEC sp_tables",
		"EXEC sp_columns", "EXEC sp_who",
	}

	for _, prefix := range queryPrefixes {
		if strings.HasPrefix(upper, prefix) {
			return true
		}
	}

	return false
}

// ParseInt 安全地解析整数
func parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

