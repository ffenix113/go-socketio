package socketio

import (
	"context"
	"io"
	"net"
	"time"
)

func write(writer io.Writer, bytes []byte) error {
	_, err := writer.Write(bytes)
	return err
}

func read(rw io.ReadWriter) ([]byte, error) {
	bts := make([]byte, 1024)
	_, err := rw.Read(bts)
	if err != nil {
		return nil, err
	}

	return bts, nil
}

var _ net.Conn = &Conn{}

type Conn struct {
	send    chan []byte
	receive chan func() []byte

	ctx    context.Context
	cancel context.CancelFunc
}

func NewConn() *Conn {
	c := &Conn{
		send:    make(chan []byte, 16),
		receive: make(chan func() []byte, 16),
	}

	c.ctx, c.cancel = context.WithCancel(context.Background())

	return c
}

func (c *Conn) Read(b []byte) (n int, err error) {
	if c.ctx.Err() != nil {
		return 0, c.ctx.Err()
	}

	btsFn, ok := <-c.receive
	if !ok {
		return 0, io.EOF
	}

	bts := btsFn()
	copy(b, bts)

	return len(bts), nil
}

func (c *Conn) Write(b []byte) (n int, err error) {
	if c.ctx.Err() != nil {
		return 0, c.ctx.Err()
	}

	c.send <- b

	return len(b), nil
}

func (c *Conn) Close() error {
	c.cancel()
	close(c.send)
	close(c.receive)

	return nil
}

func (c *Conn) LocalAddr() net.Addr {
	// TODO implement me
	panic("implement me")
}

func (c *Conn) RemoteAddr() net.Addr {
	// TODO implement me
	panic("implement me")
}

func (c *Conn) SetDeadline(t time.Time) error {
	// TODO implement me
	panic("implement me")
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	// TODO implement me
	panic("implement me")
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	// TODO implement me
	panic("implement me")
}
