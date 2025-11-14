package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pr-reviewer-service/internal/config"
	"pr-reviewer-service/internal/database"
	"pr-reviewer-service/internal/handlers"
	"pr-reviewer-service/internal/logger"
	"pr-reviewer-service/internal/repository"
	"pr-reviewer-service/internal/service"

	"github.com/gorilla/mux"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.Logger.Level)
	log.Info("Starting PR Reviewer Assignment Service")
	log.Info("Configuration loaded successfully")

	ctx := context.Background()
	dbCfg := database.Config{
		URL:             cfg.Database.URL,
		MaxConns:        cfg.Database.MaxConns,
		MinConns:        cfg.Database.MinConns,
		MaxConnLifetime: cfg.Database.MaxConnLifetime,
		MaxConnIdleTime: cfg.Database.MaxConnIdleTime,
	}

	db, err := database.New(ctx, dbCfg)
	if err != nil {
		log.Fatal("Failed to connect to database: %v", err)
	}
	defer func() {
		log.Info("Closing database connection")
		db.Close()
	}()

	log.Info("Successfully connected to database")

	// инициализируем слои приложения
	repo := repository.New(db.Pool())
	svc := service.New(repo, log)
	handler := handlers.New(svc, log)

	r := mux.NewRouter()
	handler.SetupRoutes(r)

	// добавляем middleware для логирования и восстановления после паники
	r.Use(loggingMiddleware(log))
	r.Use(recoveryMiddleware(log))

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// запускаем сервер в отдельной горутине
	go func() {
		log.Info("Server starting on port %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start: %v", err)
		}
	}()

	// ждем сигнала завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Server shutting down gracefully...")

	// graceful shutdown с таймаутом 30 секунд
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Server forced to shutdown: %v", err)
	}

	log.Info("Server stopped")
}

func loggingMiddleware(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// оборачиваем writer чтобы перехватить status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)
			log.Info("%s %s %d %v", r.Method, r.RequestURI, rw.statusCode, duration)
		})
	}
}

func recoveryMiddleware(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Error("Panic recovered: %v", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
