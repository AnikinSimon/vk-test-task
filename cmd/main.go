package main

import (
	"context"
	"flag"
	"github.com/AnikinSimon/vk-test-task/internal/app"
	"github.com/AnikinSimon/vk-test-task/internal/config"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

var (
	cfgPath = flag.String("f", "configs/config.yaml", "path to the app's config")
)

func main() {
	cfg, err := config.LoadConfig(*cfgPath)

	if err != nil {
		panic(err)
	}

	log := setupLogger(cfg.Env)
	ctx, cancel := context.WithCancel(context.Background())

	log.Info("starting application ", slog.Any("env", cfg))

	application := app.New(ctx, log, cfg.GRPCServerCfg.Port)

	go application.GRPCServer.MustRunRPC()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	sign := <-stop
	cancel()

	log.Info("Stopping application", slog.String("signal", sign.String()))

	application.GRPCServer.Stop()

	log.Info("application stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}
