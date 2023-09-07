package socketio

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type AdapterSender interface {
	Broadcast(ctx context.Context, event string, data any) error
	EmitForUser(ctx context.Context, userID, event string, data any) error
}

type AdapterReceiver interface {
	// ReceivedNew will be called when new event is received.
	//
	// userID may be empty if this is a broadcast event.
	//
	// Currently all notifications will be processed here,
	// without difference between user or broadcast notifications or
	// if user in received event is present on this server.
	ReceivedNew(ctx context.Context, userID, event string, data json.RawMessage)
}

var _ AdapterSender = RedisAdapter{}

type PushData struct {
	UserID string
	Event  string
	Data   json.RawMessage
}

type RedisAdapter struct {
	r     redis.UniversalClient
	recvr AdapterReceiver

	codec DataCodec

	eventsChannel string
}

func NewRedisAdapter(r redis.UniversalClient, recvr AdapterReceiver, codec DataCodec, eventsChannel string) AdapterSender {
	if codec == nil {
		codec = NewJSONCodec()
	}

	if eventsChannel == "" {
		eventsChannel = "events:websocket"
	}

	adapter := RedisAdapter{
		r:     r,
		recvr: recvr,

		codec: codec,

		eventsChannel: eventsChannel,
	}

	go adapter.Listen(context.Background())

	return adapter
}

func (a RedisAdapter) Broadcast(ctx context.Context, event string, data any) error {
	dataBytes, _ := a.codec.MarashalJSON(data)

	if err := a.send(ctx, PushData{
		Event: event,
		Data:  dataBytes,
	}); err != nil {
		return fmt.Errorf("broadcast: %w", err)
	}

	return nil
}

func (a RedisAdapter) EmitForUser(ctx context.Context, userID string, event string, data any) error {
	dataBytes, _ := a.codec.MarashalJSON(data)

	if err := a.send(ctx, PushData{
		UserID: userID,
		Event:  event,
		Data:   dataBytes,
	}); err != nil {
		return fmt.Errorf("emitForUser: %w", err)
	}

	return nil
}

func (a RedisAdapter) send(ctx context.Context, data PushData) error {
	bts, _ := json.Marshal(data)
	resp := a.r.Publish(ctx, a.eventsChannel, bts)
	if err := resp.Err(); err != nil {
		return fmt.Errorf("send websocket event from adapter: %w", err)
	}

	return nil
}

func (a RedisAdapter) Listen(ctx context.Context) error {
	s := a.r.Subscribe(ctx, a.eventsChannel)
	defer s.Close()

	evCh := s.Channel()
	for msg := range evCh {
		var d PushData
		if err := json.Unmarshal([]byte(msg.Payload), &d); err != nil {
			continue
		}

		a.recvr.ReceivedNew(ctx, d.UserID, d.Event, d.Data)
	}

	return nil
}
