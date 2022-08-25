package xssdemo

import (
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

const (
	// ref: https://pkg.go.dev/database/sql#DB.SetMaxIdleConns
	// maximum number of connections in the idle connection pool
	maxIdelConnections = 256
	// maximum number of open connections to the database
	maxOpenConnections = 256
)

var dsn = fmt.Sprintf(
	"%s:%s@tcp(%s:%s)/%s",
	// "test", "test", "db", "3306", "test",
	os.Getenv("DB_USER"),
	os.Getenv("DB_PASSWORD"),
	os.Getenv("DB_HOST"),
	os.Getenv("DB_PORT"),
	os.Getenv("DB_DATABASE"),
) + "?parseTime=true&collation=utf8mb4_bin"

func NewDB() (*sqlx.DB, error) {
	logger.Printf("connect to %s\n", dsn)
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(maxIdelConnections)
	db.SetMaxOpenConns(maxOpenConnections)

	return db, nil
}
