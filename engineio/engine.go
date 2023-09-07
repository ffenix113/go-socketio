package engineio

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const Version = "4"

// ReadTimeout defines how much time server should wait
// on read before giving up.
var ReadTimeout = 2 * time.Second

type TransportType string

type OpenPacket struct {
	SID          string   `json:"sid,omitempty"`
	Upgrades     []string `json:"upgrades"`
	PingInterval int      `json:"pingInterval,omitempty"`
	PingTimeout  int      `json:"pingTimeout,omitempty"`
	MaxPayload   int      `json:"maxPayload,omitempty"`
}

type ReadBytes func(io.ReadWriter) ([]byte, error)
type WriteBytes func(io.Writer, []byte) error
type PacketHandler func(*Socket, Packet)

type Engine struct {
	PingInterval time.Duration
	PingTimeout  time.Duration

	Read  ReadBytes
	Write WriteBytes

	PacketHandler PacketHandler

	OnDisconnect func(socket *Socket)

	metrics *Metrics

	mu sync.RWMutex
}

func NewEngine(reg prometheus.Registerer, pingInterval, pingTimeout time.Duration, read ReadBytes, write WriteBytes, packetHandler PacketHandler) *Engine {
	return &Engine{
		PingInterval: pingInterval,
		PingTimeout:  pingTimeout,

		Read:  read,
		Write: write,

		PacketHandler: packetHandler,

		metrics: NewMetrics(reg),
	}
}

// NewClient creates and inserts socket to the engine.
//
// TODO: split this to NewClient(net.Conn) and AddClient(*Socket)
func (e *Engine) NewClient(conn net.Conn) *Socket {
	e.mu.Lock()
	defer e.mu.Unlock()

	cl := &Socket{
		engine: e,
		conn:   conn,
		send:   make(chan Packet, 16),
	}

	e.metrics.CurrentClients.Inc()

	// FIXME: Refactor this to a separate method that will start running only after initial handshake.
	go cl.readRoutine(ReadTimeout, e.PingTimeout, e.PacketHandler)
	go cl.writeRoutine(e.PingInterval)

	e.sendOpenPacket(cl)

	return cl
}

func (e *Engine) readPacket(conn net.Conn) (Packet, error) {
	data, err := e.Read(conn)
	if err != nil {
		return Packet{}, fmt.Errorf("read Engine.IO packet: %w", err)
	}

	var packet Packet
	if err := packet.UnmarshalBinary(data); err != nil {
		return Packet{}, fmt.Errorf("unmarshal Engine.IO packet: %w", err)
	}

	return packet, nil
}

func (e *Engine) Send(cl *Socket, packet Packet) {
	cl.Write(packet)
}

func (e *Engine) sendOpenPacket(cl *Socket) {
	p := OpenPacket{
		SID:          SocketID,
		Upgrades:     []string{},
		PingInterval: int(e.PingInterval / time.Millisecond),
		PingTimeout:  int(e.PingTimeout / time.Millisecond),
		// This value if random
		MaxPayload: 1000000,
	}

	// json here is a requirement for SocketIO
	bts, _ := json.Marshal(p)

	cl.Write(Packet{
		Type: PacketTypeOpen,
		Data: bts,
	})
}
