package cmd

import (
	"context"
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

	return nil
}