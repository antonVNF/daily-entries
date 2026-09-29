package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var ErrMissing = errors.New("required variable is not set")
var ErrInvalid = errors.New("invalid value")

type Config struct {
	Log      Log
	HTTP     HTTP
	Postgres Postgres
}

type Log struct {
	Level  string
	Format string
}

type HTTP struct {
	Addr              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

type Postgres struct {
	DSN             string
	ApplicationName string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	ConnectTimeout  time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{}

	cfg.Log.Level = strEnv("LOG_LEVEL", "info")
	cfg.Log.Format = strEnv("LOG_FORMAT", "json")
	if err := validateLog(cfg.Log); err != nil {
		return nil, err
	}

	cfg.HTTP.Addr = strEnv("HTTP_ADDR", ":8080")

	var err error
	if cfg.HTTP.ReadTimeout, err = durationEnv("HTTP_READ_TIMEOUT", 15*time.Second); err != nil {
		return nil, err
	}
	if cfg.HTTP.ReadHeaderTimeout, err = durationEnv("HTTP_READ_HEADER_TIMEOUT", 5*time.Second); err != nil {
		return nil, err
	}
	if cfg.HTTP.WriteTimeout, err = durationEnv("HTTP_WRITE_TIMEOUT", 15*time.Second); err != nil {
		return nil, err
	}
	if cfg.HTTP.IdleTimeout, err = durationEnv("HTTP_IDLE_TIMEOUT", 60*time.Second); err != nil {
		return nil, err
	}
	if cfg.HTTP.ShutdownTimeout, err = durationEnv("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second); err != nil {
		return nil, err
	}
	if err := validateHTTP(cfg.HTTP); err != nil {
		return nil, err
	}

	cfg.Postgres.DSN = strEnv("POSTGRES_DSN", "")
	if cfg.Postgres.DSN == "" {
		return nil, fmt.Errorf("POSTGRES_DSN: %w", ErrMissing)
	}
	if err := validateDSN(cfg.Postgres.DSN); err != nil {
		return nil, err
	}

	cfg.Postgres.ApplicationName = strEnv("POSTGRES_APP_NAME", "api")

	if cfg.Postgres.MaxConns, err = intEnv("POSTGRES_MAX_CONNS", 10); err != nil {
		return nil, err
	}
	if cfg.Postgres.MinConns, err = intEnv("POSTGRES_MIN_CONNS", 2); err != nil {
		return nil, err
	}

	if cfg.Postgres.MaxConns <= 0 {
		return nil, fmt.Errorf("POSTGRES_MAX_CONNS=%d: %w: must be positive", cfg.Postgres.MaxConns, ErrInvalid)
	}
	if cfg.Postgres.MinConns < 0 {
		return nil, fmt.Errorf("POSTGRES_MIN_CONNS=%d: %w: must not be negative", cfg.Postgres.MinConns, ErrInvalid)
	}
	if cfg.Postgres.MinConns > cfg.Postgres.MaxConns {
		return nil, fmt.Errorf("POSTGRES_MIN_CONNS (%d) > POSTGRES_MAX_CONNS (%d): %w",
			cfg.Postgres.MinConns, cfg.Postgres.MaxConns, ErrInvalid)
	}

	if cfg.Postgres.MaxConnLifetime, err = durationEnv("POSTGRES_MAX_CONN_LIFETIME", time.Hour); err != nil {
		return nil, err
	}
	if cfg.Postgres.MaxConnIdleTime, err = durationEnv("POSTGRES_MAX_CONN_IDLE_TIME", 5*time.Minute); err != nil {
		return nil, err
	}
	if cfg.Postgres.ConnectTimeout, err = durationEnv("POSTGRES_CONNECT_TIMEOUT", 5*time.Second); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) LogAttrs() []slog.Attr {
	return []slog.Attr{
		slog.String("log_level", c.Log.Level),
		slog.String("log_format", c.Log.Format),
		slog.String("http_addr", c.HTTP.Addr),
		slog.Duration("http_read_timeout", c.HTTP.ReadTimeout),
		slog.Duration("http_read_header_timeout", c.HTTP.ReadHeaderTimeout),
		slog.Duration("http_write_timeout", c.HTTP.WriteTimeout),
		slog.Duration("http_idle_timeout", c.HTTP.IdleTimeout),
		slog.Duration("http_shutdown_timeout", c.HTTP.ShutdownTimeout),
		slog.String("pg_dsn", c.Postgres.Redacted()),
		slog.String("pg_app_name", c.Postgres.ApplicationName),
		slog.Int("pg_max_conns", int(c.Postgres.MaxConns)),
		slog.Int("pg_min_conns", int(c.Postgres.MinConns)),
		slog.Duration("pg_max_conn_lifetime", c.Postgres.MaxConnLifetime),
		slog.Duration("pg_max_conn_idle_time", c.Postgres.MaxConnIdleTime),
		slog.Duration("pg_connect_timeout", c.Postgres.ConnectTimeout),
	}
}

func (p Postgres) Redacted() string {
	u, err := url.Parse(p.DSN)
	if err != nil {
		return "<invalid dsn>"
	}
	if u.User != nil {
		if _, hasPass := u.User.Password(); hasPass {
			u.User = url.UserPassword(u.User.Username(), "REDACTED")
		}
	}
	return u.String()
}

func strEnv(key, def string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	return strings.TrimSpace(v)
}

func intEnv(key string, def int32) (int32, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def, nil
	}
	v = strings.TrimSpace(v)

	n, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s=%q: %w: %v", key, v, ErrInvalid, err)
	}
	return int32(n), nil
}

func durationEnv(key string, def time.Duration) (time.Duration, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def, nil
	}
	v = strings.TrimSpace(v)

	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s=%q: %w: %v", key, v, ErrInvalid, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("%s=%v: %w: must be positive", key, d, ErrInvalid)
	}
	return d, nil
}

func validateLog(l Log) error {
	switch l.Level {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("LOG_LEVEL=%q: %w: must be one of debug|info|warn|error", l.Level, ErrInvalid)
	}
	switch l.Format {
	case "json", "text":
	default:
		return fmt.Errorf("LOG_FORMAT=%q: %w: must be json or text", l.Format, ErrInvalid)
	}
	return nil
}

func validateHTTP(h HTTP) error {
	_, portStr, err := net.SplitHostPort(h.Addr)
	if err != nil {
		return fmt.Errorf("HTTP_ADDR=%q: %w: must be in host:port format: %v", h.Addr, ErrInvalid, err)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("HTTP_ADDR=%q: %w: invalid port %q", h.Addr, ErrInvalid, portStr)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("HTTP_ADDR=%q: %w: port must be in [1, 65535]", h.Addr, ErrInvalid)
	}
	return nil
}

func validateDSN(dsn string) error {
	u, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("POSTGRES_DSN: %w: %v", ErrInvalid, err)
	}
	switch u.Scheme {
	case "postgres", "postgresql":
	default:
		return fmt.Errorf("POSTGRES_DSN: %w: unsupported scheme %q", ErrInvalid, u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("POSTGRES_DSN: %w: host is empty", ErrInvalid)
	}
	return nil
}
