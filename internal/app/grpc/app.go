package grpcapp

import (
	"context"
	"fmt"
	subpubv1 "github.com/AnikinSimon/vk-test-task/internal/grpc/subpub/v1"
	"github.com/AnikinSimon/vk-test-task/pkg/subpub"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log/slog"
	"net"
)

type App struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	portRPC    int
}

func New(
	ctx context.Context,
	log *slog.Logger,
	subpubService subpub.SubPub,
	portRPC int,
) (*App, error) {
	gRPCServer := grpc.NewServer()

	reflection.Register(gRPCServer)

	subpubv1.Register(ctx, gRPCServer, subpubService)

	return &App{
		log:        log,
		gRPCServer: gRPCServer,
		portRPC:    portRPC,
	}, nil
}

func (a *App) MustRunRPC() {
	if err := a.RunRPC(); err != nil {
		panic(err)
	}
}

func (a *App) RunRPC() error {
	const op = "grpcapp.RunRPC"

	log := a.log.With(
		slog.String("op", op),
		slog.Int("port", a.portRPC),
	)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.portRPC))

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("gRPC server is running", slog.String("addr", l.Addr().String()))

	if err := a.gRPCServer.Serve(l); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (a *App) Stop() {
	const op = "grpcapp.Stop"

	a.log.With(slog.String("op", op)).Info("Stopping gRPC server", slog.Int("port", a.portRPC))

	a.gRPCServer.GracefulStop()
}
