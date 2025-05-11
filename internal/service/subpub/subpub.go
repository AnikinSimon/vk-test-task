package subpub_service

import (
	"context"
	"github.com/AnikinSimon/vk-test-task/pkg/subpub"
)

type PubSubService struct {
	bus subpub.SubPub
}

func NewPubSubService() *PubSubService {
	return &PubSubService{bus: subpub.NewSubPub()}
}

func (p *PubSubService) Subscribe(subject string, cb subpub.MessageHandler) (subpub.Subscription, error) {
	return p.bus.Subscribe(subject, cb)
}

func (p *PubSubService) Publish(subject string, msg interface{}) error {
	return p.bus.Publish(subject, msg)
}

func (p *PubSubService) Close(ctx context.Context) error {
	return p.bus.Close(ctx)
}
