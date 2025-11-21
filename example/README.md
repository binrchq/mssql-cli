# SQL Server CLI 使用示例

## 基本使用

```go
cli := mssqlcli.NewCLI(term, "localhost", 1433, "sa", "password", "testdb")
cli.Connect()
cli.Start()
```

## 特点

- 支持 GO 批处理分隔符
- 系统存储过程支持
- T-SQL 语法

## 常用命令

- `USE <database>` - 切换数据库
- `sp_help [table]` - 显示表信息
- `sp_databases` - 列出数据库
- `sp_tables` - 列出表
- `GO` - 执行批处理

## SQL 命令

- `SELECT` - 查询
- `INSERT` - 插入
- `UPDATE` - 更新
- `DELETE` - 删除
- `CREATE TABLE` - 创建表
- `ALTER TABLE` - 修改表

