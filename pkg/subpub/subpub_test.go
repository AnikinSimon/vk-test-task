package subpub

import (
	"context"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestSubscribePublish(t *testing.T) {
	sp := NewSubPub()
	defer sp.Close(context.Background())

	var receivedMsg interface{}
	var wg sync.WaitGroup
	wg.Add(1)

	handler := func(msg interface{}) {
		receivedMsg = msg
		wg.Done()
	}

	sub, err := sp.Subscribe("test", handler)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer sub.Unsubscribe()

	testMsg := "hello world"
	if err := sp.Publish("test", testMsg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	wg.Wait()

	if receivedMsg != testMsg {
		t.Errorf("Expected message %v, got %v", testMsg, receivedMsg)
	}
}

func TestMultipleSubscribers(t *testing.T) {
	sp := NewSubPub()
	defer sp.Close(context.Background())

	const numSubscribers = 5
	var receivedCount int
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(numSubscribers)

	handler := func(msg interface{}) {
		mu.Lock()
		receivedCount++
		mu.Unlock()
		wg.Done()
	}

	for i := 0; i < numSubscribers; i++ {
		sub, err := sp.Subscribe("test", handler)
		if err != nil {
			t.Fatalf("Subscribe failed: %v", err)
		}
		defer sub.Unsubscribe()
	}

	if err := sp.Publish("test", "message"); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	wg.Wait()

	if receivedCount != numSubscribers {
		t.Errorf("Expected %d subscribers to receive message, got %d", numSubscribers, receivedCount)
	}
}

func TestUnsubscribe(t *testing.T) {
	sp := NewSubPub()
	defer sp.Close(context.Background())

	var receivedCount int
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(1) // ожидаем только одно сообщение

	handler := func(msg interface{}) {
		mu.Lock()
		receivedCount++
		mu.Unlock()
		wg.Done()
	}

	_, err := sp.Subscribe("test", handler)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	sub2, err := sp.Subscribe("test", handler)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// отписываемся перед публикацией
	sub2.Unsubscribe()

	if err := sp.Publish("test", "message"); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	wg.Wait()

	if receivedCount != 1 {
		t.Errorf("Expected 1 subscriber to receive message, got %d", receivedCount)
	}
}

func TestSlowSubscriber(t *testing.T) {
	sp := NewSubPub()
	defer sp.Close(context.Background())

	var fastReceived, slowReceived bool
	var fastWg, slowWg sync.WaitGroup
	fastWg.Add(1)
	slowWg.Add(1)

	// Быстрый подписчик
	fastHandler := func(msg interface{}) {
		fastReceived = true
		fastWg.Done()
	}

	// Медленный подписчик
	slowHandler := func(msg interface{}) {
		time.Sleep(100 * time.Millisecond) // имитируем медленную обработку
		slowReceived = true
		slowWg.Done()
	}

	fastSub, err := sp.Subscribe("test", fastHandler)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer fastSub.Unsubscribe()

	slowSub, err := sp.Subscribe("test", slowHandler)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer slowSub.Unsubscribe()

	if err := sp.Publish("test", "message"); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Проверяем, что быстрый подписчик получил сообщение быстро
	fastDone := make(chan struct{})
	go func() {
		fastWg.Wait()
		close(fastDone)
	}()

	select {
	case <-fastDone:
		if !fastReceived {
			t.Error("Fast subscriber did not receive message")
		}
	case <-time.After(50 * time.Millisecond):
		t.Error("Fast subscriber was blocked by slow subscriber")
	}

	// Проверяем, что медленный подписчик все же получил сообщение
	slowWg.Wait()
	if !slowReceived {
		t.Error("Slow subscriber did not receive message")
	}
}

func TestMessageOrder(t *testing.T) {
	sp := NewSubPub()
	defer sp.Close(context.Background())

	var received []int
	var mu sync.Mutex
	var wg sync.WaitGroup
	const numMessages = 10
	wg.Add(numMessages)

	handler := func(msg interface{}) {
		mu.Lock()
		received = append(received, msg.(int))
		mu.Unlock()
		wg.Done()
	}

	sub, err := sp.Subscribe("test", handler)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer sub.Unsubscribe()

	for i := 0; i < numMessages; i++ {
		if err := sp.Publish("test", i); err != nil {
			t.Fatalf("Publish failed: %v", err)
		}
	}

	wg.Wait()

	for i := 0; i < numMessages; i++ {
		if received[i] != i {
			t.Errorf("Message out of order. Expected %d at position %d, got %d", i, i, received[i])
			break
		}
	}
}

func TestCloseWithContext(t *testing.T) {
	sp := NewSubPub()

	// Создаем подписчика с медленной обработкой
	var slowWg sync.WaitGroup
	slowWg.Add(1)
	slowHandler := func(msg interface{}) {
		time.Sleep(200 * time.Millisecond)
		slowWg.Done()
	}

	_, err := sp.Subscribe("test", slowHandler)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Публикуем сообщение
	if err := sp.Publish("test", "message"); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Закрываем с контекстом, который отменится быстрее чем обработчик закончит работу
	err = sp.Close(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}

	// Даем время медленному обработчику завершиться
	slowWg.Wait()
}

func TestPublishAfterClose(t *testing.T) {
	sp := NewSubPub()

	// Закрываем шину
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := sp.Close(ctx)
	assert.NoError(t, err)

	err = sp.Publish("test", "message")
	assert.ErrorIs(t, err, errorSubPubClosed)
}

func TestSubscribeAfterClose(t *testing.T) {
	sp := NewSubPub()

	// Закрываем шину
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := sp.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Пытаемся подписаться после закрытия
	_, err := sp.Subscribe("test", func(msg interface{}) {})
	assert.ErrorIs(t, err, errorSubPubClosed)
}
