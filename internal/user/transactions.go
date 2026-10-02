package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

var _ Repository = (*PostgresRepository)(nil)

func (r *PostgresRepository) GetByID(ctx context.Context, id int64) (User, error) {
	const q = `
		SELECT id, email, name, created_at, updated_at
		FROM users
		WHERE id = $1`

	var u User
	err := r.pool.QueryRow(ctx, q, id).
		Scan(&u.ID, &u.Email, &u.Name, &u.CreatedAt, &u.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("query user by id: %w", err)
	}
	return u, nil
}

func (r *PostgresRepository) Create(ctx context.Context, u User) (User, error) {
	const q = `
		INSERT INTO users (email, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	err := r.pool.QueryRow(ctx, q, u.Email, u.Name, u.CreatedAt, u.UpdatedAt).
		Scan(&u.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrEmailConflict
		}
		return User{}, fmt.Errorf("insert user: %w", err)
	}
	return u, nil
}

func (r *PostgresRepository) Update(ctx context.Context, u User) (User, error) {
	const q = `
		UPDATE users
		SET email = $2, name = $3, updated_at = $4
		WHERE id = $1
		RETURNING created_at`

	err := r.pool.QueryRow(ctx, q, u.ID, u.Email, u.Name, u.UpdatedAt).
		Scan(&u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrEmailConflict
		}
		return User{}, fmt.Errorf("update user: %w", err)
	}
	return u, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
