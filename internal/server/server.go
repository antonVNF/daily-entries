package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/antonVNF/daily-entries/internal/models"
	"github.com/antonVNF/daily-entries/internal/user"
)

type Server struct {
	users *user.Service
	log   *slog.Logger
}

func NewServer(users *user.Service, log *slog.Logger) *Server {
	return &Server{
		users: users,
		log:   log,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /users/{id}", s.handleGetUser)
	mux.HandleFunc("POST /users", s.handleCreateUser)
	mux.HandleFunc("PUT /users/{id}", s.handleUpdateUser)
	mux.HandleFunc("DELETE /users/{id}", s.handleDeleteUser)
	return mux
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, msg any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(msg); err != nil {
		s.log.Error("encode response failed", slog.Any("error", err))
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, msg string) {
	s.writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	u, err := s.users.Get(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, user.ErrNotFound):
			s.writeError(w, http.StatusNotFound, "user not found")
		default:
			s.log.ErrorContext(r.Context(), "get user failed",
				slog.Int64("id", id),
				slog.Any("error", err))
			s.writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	s.writeJSON(w, http.StatusOK, models.ToResponse(u))
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	u := user.User{
		ID:    id,
		Email: req.Email,
		Name:  req.Name,
	}

	updated, err := s.users.Update(r.Context(), u)
	if err != nil {
		switch {
		case errors.Is(err, user.ErrNotFound):
			s.writeError(w, http.StatusNotFound, "user not found")
		case errors.Is(err, user.ErrInvalidEmail):
			s.writeError(w, http.StatusBadRequest, "invalid email")
		case errors.Is(err, user.ErrEmailConflict):
			s.writeError(w, http.StatusConflict, "email already taken")
		default:
			s.log.ErrorContext(r.Context(), "update user failed",
				slog.Int64("id", id),
				slog.Any("error", err))
			s.writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	s.writeJSON(w, http.StatusOK, models.ToResponse(updated))
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	u := user.User{
		Email: req.Email,
		Name:  req.Name,
	}

	created, err := s.users.Create(r.Context(), u)
	if err != nil {
		switch {
		case errors.Is(err, user.ErrEmailConflict):
			s.writeError(w, http.StatusConflict, "email conflict")
		case errors.Is(err, user.ErrInvalidEmail):
			s.writeError(w, http.StatusBadRequest, "invalid email")
		default:
			s.log.ErrorContext(r.Context(), "create user failed", slog.Any("error", err))
			s.writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	s.writeJSON(w, http.StatusCreated, models.ToResponse(created))
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := s.users.Delete(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, user.ErrNotFound):
			s.writeError(w, http.StatusNotFound, "user not found")
		default:
			s.log.ErrorContext(r.Context(), "delete user failed",
				slog.Int64("id", id),
				slog.Any("error", err))
			s.writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
