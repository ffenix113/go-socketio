package socketio

import (
	"github.com/ffenix113/go-socketio/engineio"
)

type SocketEventHandler func(s *Socket, event string, data []byte) (any, error)

type Socket struct {
	UserID string

	cl           *engineio.Socket
	socketEngine *Engine
}

func (e *Engine) NewSocket(cl *engineio.Socket) *Socket {
	return &Socket{
		cl:           cl,
		socketEngine: e,
	}
}

func (s *Socket) Emit(event string, data any) {
	bts, _ := s.socketEngine.codec.MarashalJSON([]any{event, data})

	p, _ := Packet{
		Type:      PacketTypeEvent,
		Namespace: DefaultNamespace,
		Data:      bts,
	}.MarshalBinary()

	s.cl.Write(engineio.Packet{
		Type: engineio.PacketTypeMessage,
		Data: p,
	})
}

func (s *Socket) Server() *Engine {
	return s.socketEngine
}

func (s *Socket) Close() {
	s.socketEngine.OnDisconnect(s, "", nil)
}
