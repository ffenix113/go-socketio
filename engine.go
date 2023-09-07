package socketio

import (
	"context"
	"encoding/json"
	"net"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ffenix113/go-socketio/engineio"
)

const (
	// CatchAllEvent is special event, that will be selected
	// if no handlers are registered for received event type.
	CatchAllEvent    = "*"
	DefaultNamespace = "/"
)

type DataCodec interface {
	MarashalJSON(data any) ([]byte, error)
	UnmarshalJSONTo(data []byte, to any) error
}

type Engine struct {
	ioEngine *engineio.Engine

	codec DataCodec

	mu              sync.RWMutex
	userIDToSockets map[string][]*Socket
	engineToSocket  map[*engineio.Socket]*Socket

	handlers map[string]SocketEventHandler

	OnConnect    SocketEventHandler
	OnDisconnect SocketEventHandler

	metrics *Metrics
}

func NewEngine(reg prometheus.Registerer, pingInterval, pingTimeout time.Duration, read engineio.ReadBytes, write engineio.WriteBytes, codec DataCodec) *Engine {
	if codec == nil {
		codec = NewJSONCodec()
	}

	var noopHandler SocketEventHandler = func(s *Socket, event string, data []byte) (any, error) {
		return nil, nil
	}

	e := &Engine{
		userIDToSockets: make(map[string][]*Socket),
		engineToSocket:  make(map[*engineio.Socket]*Socket),

		codec: codec,

		handlers:     make(map[string]SocketEventHandler),
		OnConnect:    noopHandler,
		OnDisconnect: noopHandler,

		metrics: NewMetrics(reg),
	}

	e.ioEngine = engineio.NewEngine(reg, pingInterval, pingTimeout, read, write, e.ioPacketHandler)
	e.ioEngine.OnDisconnect = e.onDisconnect

	return e
}

func (e *Engine) GetCodec() DataCodec {
	return e.codec
}

// AddClient creates and stores Socket.
//
// TODO: Better init handshake. Wait for client to send connect & auth before moving forward.
func (e *Engine) AddClient(conn net.Conn) *engineio.Socket {
	cl := e.ioEngine.NewClient(conn)
	if cl == nil {
		return nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.engineToSocket[cl] = e.NewSocket(cl)

	return cl
}

func (e *Engine) addToNamespace(ioSocket *engineio.Socket, namespace string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	socket := e.engineToSocket[ioSocket]

	if namespace != DefaultNamespace {
		e.writeToClient(socket, Packet{
			Type:      PacketTypeConnectError,
			Namespace: namespace,
			Data:      Marshal(ErrorData{Error: "only default namespace is supported"}),
		})

		return
	}

	type connData struct {
		SID string `json:"sid"`
	}

	e.writeToClient(socket, Packet{
		Type:      PacketTypeConnect,
		Namespace: DefaultNamespace,
		Data:      Marshal(connData{SID: engineio.SocketID}),
	})
}

func (e *Engine) RemoveSocket(ioSocket *engineio.Socket) *Socket {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.RemoveSocketUnsafe(ioSocket)
}

func (e *Engine) RemoveSocketUnsafe(ioSocket *engineio.Socket) *Socket {
	socket, ok := e.engineToSocket[ioSocket]
	if !ok {
		return nil
	}

	sockets := e.userIDToSockets[socket.UserID]
	for i, s := range sockets {
		if s == socket {
			sockets = append(sockets[:i], sockets[i+1:]...)
			e.userIDToSockets[socket.UserID] = sockets

			break
		}
	}

	delete(e.engineToSocket, ioSocket)

	return socket
}

// On adds event listener to specified event.
//
// To add catch-all listener use `On(socketio.CatchAllEvent, ...)`.
func (e *Engine) On(event string, handler SocketEventHandler) {
	e.handlers[event] = handler
}

func (e *Engine) Broadcast(_ context.Context, event string, data any) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.engineToSocket) == 0 {
		return nil
	}

	bts, _ := e.codec.MarashalJSON(data)
	dataBts, _ := json.Marshal([2]any{event, json.RawMessage(bts)})

	packet := Packet{
		Type:      PacketTypeEvent,
		Namespace: DefaultNamespace,
		Data:      dataBts,
	}

	for _, socket := range e.engineToSocket {
		e.writeToClient(socket, packet)
	}

	return nil
}

// EmitForUser emits provided event to all sockets
// for this user connected to this engine.
//
// It is only useful if `UserID` field is set on socket.
func (e *Engine) EmitForUser(_ context.Context, userID, event string, data any) error {
	e.metrics.EmitForUserCalls.Inc()

	e.mu.RLock()
	defer e.mu.RUnlock()

	sockets, ok := e.userIDToSockets[userID]
	if !ok {
		return nil
	}

	bts, _ := e.codec.MarashalJSON(data)
	dataBts, _ := json.Marshal([2]any{event, json.RawMessage(bts)})

	packet := Packet{
		Type:      PacketTypeEvent,
		Namespace: DefaultNamespace,
		Data:      dataBts,
	}

	for _, socket := range sockets {
		e.writeToClient(socket, packet)
	}

	return nil
}

// ReceivedNew is used for adapter only.
//
// This method should not be used anywhere in the code
// apart from adapter implementation.
func (e *Engine) ReceivedNew(ctx context.Context, userID, event string, data json.RawMessage) {
	if userID != "" {
		e.EmitForUser(ctx, userID, event, data)
		return
	}

	e.Broadcast(ctx, event, data)
}

func (e *Engine) onDisconnect(ioSocket *engineio.Socket) {
	// FIXME: This should not check for nil,
	// but currently this can be called multiple times for single client.
	if socket := e.RemoveSocket(ioSocket); socket != nil {
		socket.Close()
		e.OnDisconnect(socket, "", nil)
	}
}

func (e *Engine) ioPacketHandler(cl *engineio.Socket, enginePacket engineio.Packet) {
	var packet Packet
	if err := packet.UnmarshalBinary(enginePacket.Data); err != nil {
		// TODO: better error handling here
		return
	}

	type packetData [2]json.RawMessage

	e.mu.RLock()
	socket := e.engineToSocket[cl]
	e.mu.RUnlock()

	switch packet.Type {
	case PacketTypeConnect:
		_, err := e.OnConnect(socket, "", packet.Data)
		if err != nil {
			e.writeToClient(socket, Packet{
				Type:      PacketTypeConnectError,
				Namespace: packet.Namespace,
				Data:      Marshal(ErrorData{Error: err.Error()}),
			})
		}

		if userID := socket.UserID; userID != "" {
			e.userIDToSockets[socket.UserID] = append(e.userIDToSockets[socket.UserID], socket)
		}

		e.addToNamespace(cl, packet.Namespace)
	case PacketTypeDisconnect:
		e.onDisconnect(cl)
	case PacketTypeEvent:
		var data packetData
		if err := e.codec.UnmarshalJSONTo(packet.Data, &data); err != nil {
			panic(err)
		}

		eventName := string(data[0][1 : len(data[0])-1])

		handler := e.handlers[eventName]
		if handler == nil {
			// Try catch-all handler to handle event.
			handler = e.handlers[CatchAllEvent]
		}

		if handler == nil {
			return
		}

		resp, err := handler(socket, eventName, data[1])

		if packet.AckID != nil {
			packet.Type = PacketTypeAck

			data := [1]any{resp}
			if err != nil {
				data = [1]any{ErrorData{Error: err.Error()}}
			}

			packet.Data = Marshal(data)

			e.writeToClient(socket, packet)
		}
	}
}

func (e *Engine) writeToClient(socket *Socket, p Packet) {
	data, _ := p.MarshalBinary()

	ioPacket := engineio.Packet{
		Type: engineio.PacketTypeMessage,
		Data: data,
	}

	e.ioEngine.Send(socket.cl, ioPacket)
}
