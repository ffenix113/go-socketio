package engineio

import (
	"net"
	"sync/atomic"
	"time"
)

// SocketID is only used on initial connection
// and when adding to namespace.
// It is not used on reconnections, neither to deferentiate socket clients.
const SocketID = "9Cx9Ds4C"

type Socket struct {
	engine *Engine
	conn   net.Conn

	send chan Packet

	closed int32
}

func (c *Socket) Write(p Packet) {
	select {
	case c.send <- p:
	default:
	}
}

func (c *Socket) Close() error {
	// Close only once.
	// Unfortunately this is a ad-hoc temporary solution.
	// FIXME: implement non-recursive way of closing connection.
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return nil
	}

	close(c.send)
	c.engine.metrics.CurrentClients.Dec()

	c.engine.OnDisconnect(c)

	return nil
}

// read and writeRoutine are copied from
// https://github.com/gorilla/websocket/blob/76ecc29eff79f0cedf70c530605e486fc32131d1/examples/chat/client.go

func (c *Socket) readRoutine(readTimeout, pingTimeout time.Duration, packetHandler PacketHandler) {
	defer func() {
		c.Close()
	}()

	for {
		packet, err := c.engine.readPacket(c.conn)
		if err != nil {
			break
		}

		_ = c.conn.SetReadDeadline(time.Now().Add(pingTimeout))
		_ = c.conn.SetWriteDeadline(time.Now().Add(pingTimeout))

		switch packet.Type {
		case PacketTypePong:
		case PacketTypeMessage:
			packetHandler(c, packet)
		}
	}
}

func (c *Socket) writeRoutine(pingInterval time.Duration) {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()

		_ = c.conn.Close()
	}()

	for {
		select {
		case packet, ok := <-c.send:
			if !ok {
				_ = c.engine.Write(c.conn, closePacketBts)

				return
			}

			// Add queued chat messages to the current websocket message.
			b, _ := packet.MarshalBinary()
			if err := c.engine.Write(c.conn, b); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.engine.Write(c.conn, pingPacketBts); err != nil {
				return
			}
		}
	}
}
