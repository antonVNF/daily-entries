package main

import (
	"awesomeProject3/internal/config"
	logger2 "awesomeProject3/internal/logger"
	"context"
	"fmt"
	"net/http"
)

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))

}

func main() {
	cfg := config.Load()
	port := cfg.Port
	logger := logger2.NewLogger(cfg.LogLevel)
	ctx := context.Context(context.Background())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	logger.Info("listen and serve")
	err := server.ListenAndServe()
	if err != nil {
		logger.Error("")
	}
}
