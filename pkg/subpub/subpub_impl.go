// subpub.go
package subpub

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrSubPubClosed      = errors.New("subpub already closed")
	ErrSubjectNotFound   = errors.New("subject not found")
	ErrNilMessageHandler = errors.New("nil message handler")
)

const (
	bufferSize = 64
)

func NewSubPub() SubPub {
	return newSubPub()
}

type subPubImpl struct {
	mu          sync.RWMutex
	subscribers map[string]map[*subscriber]struct{}
	closed      bool
	closeCh     chan struct{}
	wg          sync.WaitGroup
}

func newSubPub() *subPubImpl {
	return &subPubImpl{
		subscribers: make(map[string]map[*subscriber]struct{}),
		closeCh:     make(chan struct{}),
	}
}

type subscriber struct {
	handler    MessageHandler
	msgCh      chan interface{}
	unsubOnce  sync.Once
	unsubFn    func()
	bufferSize int
}

func (sp *subPubImpl) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if sp.closed {
		return nil, ErrSubPubClosed
	}

	if cb == nil {
		return nil, ErrNilMessageHandler
	}

	sub := &subscriber{
		handler:    cb,
		msgCh:      make(chan interface{}, bufferSize),
		bufferSize: bufferSize,
	}

	sub.unsubFn = func() {
		sp.mu.Lock()
		defer sp.mu.Unlock()
		if subs, ok := sp.subscribers[subject]; ok {
			delete(subs, sub)
		}
		close(sub.msgCh)
	}

	if _, ok := sp.subscribers[subject]; !ok {
		sp.subscribers[subject] = make(map[*subscriber]struct{})
	}
	sp.subscribers[subject][sub] = struct{}{}

	sp.wg.Add(1)
	go func() {
		defer sp.wg.Done()
		for msg := range sub.msgCh {
			sub.handler(msg)
		}
	}()

	return sub, nil
}

func (s *subscriber) Unsubscribe() {
	s.unsubOnce.Do(s.unsubFn)
}

func (sp *subPubImpl) Publish(subject string, msg interface{}) error {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	if sp.closed {
		return ErrSubPubClosed
	}

	if subs, ok := sp.subscribers[subject]; ok {
		for sub := range subs {
			select {
			case sub.msgCh <- msg:
			default:
				go func(s *subscriber, m interface{}) {
					s.msgCh <- m
				}(sub, msg)
			}
		}
		return nil
	} else {
		return ErrSubjectNotFound
	}
}

func (sp *subPubImpl) Close(ctx context.Context) error {
	sp.mu.Lock()
	if sp.closed {
		sp.mu.Unlock()
		return nil
	}
	sp.closed = true
	close(sp.closeCh)
	subscribersCopy := make([]*subscriber, 0)
	for _, subs := range sp.subscribers {
		for sub := range subs {
			subscribersCopy = append(subscribersCopy, sub)
		}
	}
	sp.subscribers = nil
	sp.mu.Unlock()

	for _, sub := range subscribersCopy {
		sub.Unsubscribe()
	}

	done := make(chan struct{})
	go func() {
		sp.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}
