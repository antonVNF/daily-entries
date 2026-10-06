package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/antonVNF/daily-entries/internal/config"
	"github.com/antonVNF/daily-entries/internal/logger"
	database "github.com/antonVNF/daily-entries/internal/repository"
	"github.com/antonVNF/daily-entries/internal/server"
	"github.com/antonVNF/daily-entries/internal/user"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.NewLogger(cfg.Log)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.LogAttrs(ctx, slog.LevelInfo, "config loaded", cfg.LogAttrs()...)

	db, err := database.New(ctx, cfg.Postgres)
	if err != nil {
		return fmt.Errorf("init database: %w", err)
	}
	defer db.Close()

	userRepo := user.NewPostgresRepository(db.Pool())
	userService := user.NewService(userRepo)
	api := server.NewServer(userService, log)

	srv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           api.Handler(),
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("server started", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listen and serve: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	log.Info("server stopped gracefully")
	return nil
}
