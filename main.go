package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/OSShip/sessions/internal/config"
	"github.com/OSShip/sessions/internal/events"
	"github.com/OSShip/sessions/internal/handler"
	"github.com/OSShip/sessions/internal/jitsi"
	"github.com/OSShip/sessions/internal/scheduler"
	"github.com/OSShip/sessions/internal/store"
	"github.com/OSShip/utils/observability"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()
	observability.InitSentry("sessions")
	defer observability.FlushSentry(2 * time.Second)
	logger := observability.InitLogger("sessions")

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("database connected")

	pub := events.New(cfg.KafkaBrokers)
	defer pub.Close()
	logger.Info("kafka publisher ready", "brokers", cfg.KafkaBrokers)

	st := store.New(pool)
	h := &handler.Handler{Store: st, Events: pub, Jitsi: jitsi.JitsiInfo{ApiKey: cfg.JitsiApiKey, AppID: cfg.JitsiAppID, PrivateKeyFilepath: cfg.JitsiPrivateKeyFilename}}
	scheduler.StartReminders(ctx, st, pub)
	logger.Info("reminder scheduler started", "interval", "5m", "window", "30m")

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(observability.SentryHTTPMiddleware)
	r.Use(observability.SentryRecoverMiddleware("sessions"))
	r.Use(observability.SentryErrorMiddleware("sessions"))
	r.Use(observability.RequestLogMiddleware("sessions"))
	r.Use(observability.PrometheusMiddleware("sessions"))

	r.Get("/health", observability.HealthHandler("sessions"))
	r.Get("/metrics", observability.MetricsHandler().ServeHTTP)

	r.Post("/", h.Create)
	r.Get("/listings/{listingId}", h.ListByListing)
	r.Patch("/{id}", h.Update)
	r.Post("/{id}/join", h.Join)
	r.Post("/progress", h.AddProgress)
	r.Get("/progress", h.ListProgress)

	logger.Info("sessions listening", "port", cfg.Port, "jitsi_base", cfg.JitsiBaseURL)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		logger.Error("server failed", "err", err)
		os.Exit(1)
	}
}
