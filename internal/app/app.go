package app

import (
	"context"
	grpcapp "github.com/AnikinSimon/vk-test-task/internal/app/grpc"
	subpub_service "github.com/AnikinSimon/vk-test-task/internal/service/subpub"
	"log/slog"
)

type App struct {
	GRPCServer *grpcapp.App
}

func New(
	ctx context.Context,
	log *slog.Logger,
	grpcPort int,
) *App {
	pubSubService := subpub_service.NewPubSubService()

	grpcApp, err := grpcapp.New(ctx, log, pubSubService, grpcPort)

	if err != nil {
		panic(err)
	}

	return &App{
		GRPCServer: grpcApp,
	}
}
