package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ezhigval/go-toolkit/httputil"
	"github.com/ezhigval/go-toolkit/logger"
	tkmw "github.com/ezhigval/go-toolkit/middleware"
	tkpgx "github.com/ezhigval/go-toolkit/pgx"
	tkredis "github.com/ezhigval/go-toolkit/redis"
	"github.com/ezhigval/realtime-chat/internal/config"
	"github.com/ezhigval/realtime-chat/internal/handler"
	"github.com/ezhigval/realtime-chat/internal/hub"
	"github.com/ezhigval/realtime-chat/internal/presence"
	"github.com/ezhigval/realtime-chat/internal/pubsub"
	"github.com/ezhigval/realtime-chat/internal/repository"
	"github.com/ezhigval/realtime-chat/internal/service"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg := config.MustLoad()
	log := logger.New(logger.Config{Level: cfg.LogLevel, Format: cfg.LogFormat})
	ctx := context.Background()

	pool, err := tkpgx.NewPool(ctx, tkpgx.Config{URL: cfg.DatabaseURL, MaxConns: 20})
	if err != nil {
		log.Error("postgres failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	rdb := tkredis.NewClient(tkredis.Config{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer func() { _ = tkredis.Close(rdb) }()

	if err := tkpgx.Ping(ctx, pool); err != nil {
		log.Error("postgres ping failed", "error", err)
		os.Exit(1)
	}
	if err := tkredis.Ping(ctx, rdb); err != nil {
		log.Error("redis ping failed", "error", err)
		os.Exit(1)
	}

	rooms := repository.NewRoomRepository(pool)
	messages := repository.NewMessageRepository(pool)
	presenceStore := presence.NewStore(rdb)
	chatHub := hub.New(log)
	bus := pubsub.NewBus(rdb, cfg.InstanceID, log)
	defer func() { _ = bus.Close() }()

	svc := service.NewChatService(rooms, messages, presenceStore, bus, chatHub)
	rest := handler.NewREST(svc)
	ws := handler.NewWS(svc, log)

	r := chi.NewRouter()
	r.Use(tkmw.RequestID, tkmw.RealIP, tkmw.Recoverer(log), tkmw.AccessLog(log))
	r.Use(chimw.Timeout(60 * time.Second))

	r.Get("/health", httputil.HealthHandler(map[string]func() error{
		"postgres": func() error { return tkpgx.Ping(ctx, pool) },
		"redis":    func() error { return tkredis.Ping(ctx, rdb) },
	}))

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/rooms", rest.ListRooms)
		r.Post("/rooms", rest.CreateRoom)
		r.Get("/rooms/{id}/messages", rest.ListMessages)
		r.Get("/rooms/{id}/presence", rest.ListPresence)
	})
	r.Get("/ws", ws.ServeHTTP)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // websockets
	}

	go func() {
		log.Info("server started", "port", cfg.Port, "instance", cfg.InstanceID)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown failed", "error", err)
	}
	log.Info("server stopped")
}
