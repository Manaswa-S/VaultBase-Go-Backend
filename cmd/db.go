package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	sqlc "main.go/internal/sqlc/generate"
)

var PostgresPool *pgxpool.Pool
var Queries *sqlc.Queries


func InitDB() error {
	var err error
	ctx := context.Background()

	connStr := os.Getenv("PostgresqlConnectionString")

	PostgresPool, err = pgxpool.New(ctx, connStr)
	if err != nil {
		return err 
	}

	Queries = sqlc.New(PostgresPool)
	err = PostgresPool.Ping(ctx)
	if err != nil {
		fmt.Println("DB ping failed : " + err.Error())
	}
	return nil
}