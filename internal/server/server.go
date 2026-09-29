package server

import "github.com/jackc/pgx/v5/pgxpool"

type Server struct {
	port string
	pool *pgxpool.Pool
	db   *pgx
}

func New()
