package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"project-atlas/internal/api"
	"project-atlas/internal/config"
	"project-atlas/internal/download"
	"project-atlas/internal/metadata"
	"project-atlas/internal/storage"
	"project-atlas/internal/upload"
	"project-atlas/internal/version"
)

func main() {
	// Configure slog to use JSON handler
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Initialize database
	pool, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("postgres ping failed", "error", err)
		os.Exit(1)
	}

	// Initialize dependencies
	repo := metadata.NewPostgresRepository(pool)
	store := storage.NewLocalChunkStore(cfg.StorageRoot)
	
	uploadSvc := upload.NewService(repo, store, cfg.ChunkSizeBytes)
	downSvc := download.NewService(repo, store)
	versionSvc := version.NewService(repo, store)

	handlers := api.NewHandlers(repo, uploadSvc, downSvc, versionSvc, cfg.MaxUploadSizeBytes)
	router := api.NewRouter(handlers)

	// Start server
	slog.Info("starting server", "addr", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, router); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
