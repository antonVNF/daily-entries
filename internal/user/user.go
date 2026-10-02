package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
)

// Доменные ошибки.
var (
	ErrNotFound      = errors.New("user not found")
	ErrEmailConflict = errors.New("email already taken")
	ErrInvalidEmail  = errors.New("invalid email")
)

// User — доменная модель пользователя.
type User struct {
	ID        int64
	Email     string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Repository описывает то что нужно домену от хранилища.
type Repository interface {
	GetByID(ctx context.Context, id int64) (User, error)
	Create(ctx context.Context, u User) (User, error)
	Update(ctx context.Context, u User) (User, error)
	Delete(ctx context.Context, id int64) error
}

// Service реализует бизнес-сценарии работы с пользователями.
type Service struct {
	repo Repository
}

// NewService создаёт Service с заданным хранилищем.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Get возвращает пользователя по id.
func (s *Service) Get(ctx context.Context, id int64) (User, error) {
	if id <= 0 {
		return User{}, fmt.Errorf("get user: invalid id %d", id)
	}

	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return User{}, fmt.Errorf("get user %d: %w", id, err)
	}
	return u, nil
}

// Create создаёт нового пользователя.
func (s *Service) Create(ctx context.Context, u User) (User, error) {
	email := normalizeEmail(u.Email)
	name := strings.TrimSpace(u.Name)

	if email == "" {
		return User{}, fmt.Errorf("create user: %w: empty email", ErrInvalidEmail)
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return User{}, fmt.Errorf("create user: %w: %q", ErrInvalidEmail, u.Email)
	}
	if name == "" {
		return User{}, fmt.Errorf("create user: empty name")
	}

	now := time.Now().UTC()
	created, err := s.repo.Create(ctx, User{
		Email:     email,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return created, nil
}

// Update обновляет имя и email существующего пользователя.
func (s *Service) Update(ctx context.Context, u User) (User, error) {
	if u.ID <= 0 {
		return User{}, fmt.Errorf("update user: invalid id %d", u.ID)
	}

	email := normalizeEmail(u.Email)
	name := strings.TrimSpace(u.Name)

	if email == "" {
		return User{}, fmt.Errorf("update user: %w: empty email", ErrInvalidEmail)
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return User{}, fmt.Errorf("update user: %w: %q", ErrInvalidEmail, u.Email)
	}
	if name == "" {
		return User{}, fmt.Errorf("update user: empty name")
	}

	current, err := s.repo.GetByID(ctx, u.ID)
	if err != nil {
		return User{}, fmt.Errorf("update user %d: %w", u.ID, err)
	}

	current.Email = email
	current.Name = name
	current.UpdatedAt = time.Now().UTC()

	updated, err := s.repo.Update(ctx, current)
	if err != nil {
		return User{}, fmt.Errorf("update user %d: %w", u.ID, err)
	}
	return updated, nil
}

// Delete удаляет пользователя по id.
func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("delete user: invalid id %d", id)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete user %d: %w", id, err)
	}
	return nil
}

func normalizeEmail(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}
