package subpub

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
	defer sub.Unsubscribe()

	testMsg := "hello world"
	err = sp.Publish("test", testMsg)
	require.NoError(t, err)

	wg.Wait()

	assert.Equal(t, testMsg, receivedMsg)
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
		require.NoError(t, err)
		defer sub.Unsubscribe()
	}

	err := sp.Publish("test", "message")
	require.NoError(t, err)

	wg.Wait()

	assert.Equal(t, numSubscribers, receivedCount)
}

func TestUnsubscribe(t *testing.T) {
	sp := NewSubPub()
	defer sp.Close(context.Background())

	var receivedCount int
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(1)

	handler := func(msg interface{}) {
		mu.Lock()
		receivedCount++
		mu.Unlock()
		wg.Done()
	}

	_, err := sp.Subscribe("test", handler)
	require.NoError(t, err)

	sub2, err := sp.Subscribe("test", handler)
	require.NoError(t, err)

	sub2.Unsubscribe()

	err = sp.Publish("test", "message")
	require.NoError(t, err)

	wg.Wait()

	assert.Equal(t, 1, receivedCount)
}

func TestSlowSubscriber(t *testing.T) {
	sp := NewSubPub()
	defer sp.Close(context.Background())

	var fastReceived, slowReceived bool
	var fastWg, slowWg sync.WaitGroup
	fastWg.Add(1)
	slowWg.Add(1)

	fastHandler := func(msg interface{}) {
		fastReceived = true
		fastWg.Done()
	}

	slowHandler := func(msg interface{}) {
		time.Sleep(100 * time.Millisecond)
		slowReceived = true
		slowWg.Done()
	}

	fastSub, err := sp.Subscribe("test", fastHandler)
	require.NoError(t, err)
	defer fastSub.Unsubscribe()

	slowSub, err := sp.Subscribe("test", slowHandler)
	require.NoError(t, err)
	defer slowSub.Unsubscribe()

	err = sp.Publish("test", "message")
	require.NoError(t, err)

	fastDone := make(chan struct{})
	go func() {
		fastWg.Wait()
		close(fastDone)
	}()

	select {
	case <-fastDone:
		assert.True(t, fastReceived)
	case <-time.After(50 * time.Millisecond):
		t.Error("Fast subscriber was blocked by slow subscriber")
	}

	slowWg.Wait()
	assert.True(t, slowReceived)
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
	assert.NoError(t, err)
	defer sub.Unsubscribe()

	for i := 0; i < numMessages; i++ {
		err := sp.Publish("test", i)
		assert.NoError(t, err)
	}

	wg.Wait()

	for i := 0; i < numMessages; i++ {
		assert.Equal(t, i, received[i])
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
	require.NoError(t, err)

	err = sp.Publish("test", "message")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = sp.Close(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

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
	assert.ErrorIs(t, err, errSubPubClosed)
}

func TestSubscribeAfterClose(t *testing.T) {
	sp := NewSubPub()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := sp.Close(ctx)
	require.NoError(t, err)

	_, err = sp.Subscribe("test", func(msg interface{}) {})
	assert.ErrorIs(t, err, errSubPubClosed)
}
