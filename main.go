package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/OSShip/sessions/internal/config"
	"github.com/OSShip/sessions/internal/events"
	"github.com/OSShip/sessions/internal/handler"
	"github.com/OSShip/sessions/internal/scheduler"
	"github.com/OSShip/sessions/internal/store"
	"github.com/OSShip/utils/observability"
)

func main() {
	cfg := config.Load()
	observability.InitSentry("sessions")
	defer observability.FlushSentry(2 * time.Second)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	pub := events.New(cfg.KafkaBrokers)
	defer pub.Close()

	st := store.New(pool)
	h := &handler.Handler{Store: st, Events: pub, JitsiBase: cfg.JitsiBaseURL}
	scheduler.StartReminders(ctx, st, pub)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(observability.SentryRecoverMiddleware("sessions"))
	r.Use(observability.SentryErrorMiddleware("sessions"))
	r.Use(middleware.Recoverer)
	r.Use(observability.PrometheusMiddleware("sessions"))

	r.Get("/health", observability.HealthHandler("sessions"))
	r.Get("/metrics", observability.MetricsHandler().ServeHTTP)

	r.Post("/", h.Create)
	r.Get("/listings/{listingId}", h.ListByListing)
	r.Patch("/{id}", h.Update)
	r.Post("/{id}/join", h.Join)
	r.Post("/progress", h.AddProgress)
	r.Get("/progress", h.ListProgress)

	log.Printf("sessions listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, r))
}
