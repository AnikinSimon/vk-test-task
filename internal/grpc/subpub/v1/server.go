package subpubv1

import (
	"context"
	"errors"
	"github.com/AnikinSimon/vk-test-task/pkg/subpub"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"time"
)

type Config struct {
	Port             int           `mapstructure:"port"`
	KeepAliveTimeout time.Duration `mapstructure:"keep_alive_timeout"`
}

type Server struct {
	UnimplementedPubSubServer
	sb  subpub.SubPub
	ctx context.Context
}

func Register(ctx context.Context, gRPC *grpc.Server, sb subpub.SubPub) {
	serveApi := &Server{sb: sb, ctx: ctx}
	RegisterPubSubServer(gRPC, serveApi)
}

func (s *Server) Subscribe(req *SubscribeRequest, stream PubSub_SubscribeServer) error {
	key := req.GetKey()
	if key == "" {
		return status.Error(codes.InvalidArgument, "key is required")
	}

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	sub, err := s.sb.Subscribe(key, func(msg interface{}) {
		data, ok := msg.(string)
		if !ok {
			return
		}
		if err := stream.Send(&Event{Data: data}); err != nil {
			cancel()
		}
	})
	if err != nil {
		return status.Error(codes.Internal, "failed to subscribe")
	}
	defer sub.Unsubscribe()

	select {
	case <-ctx.Done():
	case <-s.ctx.Done():
		cancel()
	}
	return nil
}

func (s *Server) Publish(ctx context.Context, req *PublishRequest) (*emptypb.Empty, error) {
	key := req.GetKey()
	data := req.GetData()

	if key == "" {
		return nil, status.Error(codes.InvalidArgument, "key is required")
	}

	if err := s.sb.Publish(key, data); err != nil {
		if errors.Is(err, subpub.ErrSubjectNotFound) {
			return nil, status.Error(codes.NotFound, "subject not found")
		}
		return nil, status.Error(codes.Internal, "failed to publish")
	}

	return &emptypb.Empty{}, nil
}
