package socketio

import (
	"bytes"
	"encoding/json"
	"strconv"
)

const Version = 5

const (
	PacketTypeConnect      PacketType = "0"
	PacketTypeDisconnect   PacketType = "1"
	PacketTypeEvent        PacketType = "2"
	PacketTypeAck          PacketType = "3"
	PacketTypeConnectError PacketType = "4"
	PacketTypeBinaryEvent  PacketType = "5"
	PacketTypeBinaryAck    PacketType = "6"
)

type PacketType string

type Packet struct {
	Type      PacketType
	Namespace string
	AckID     *int
	Data      json.RawMessage
	// TODO: add binary
}

type ErrorData struct {
	Error string `json:"error"`
}

func (p Packet) MarshalBinary() (data []byte, err error) {
	var b bytes.Buffer

	b.WriteString(string(p.Type))
	if p.Namespace != "/" {
		b.WriteString(p.Namespace)
		b.WriteByte(',')
	}
	if p.AckID != nil {
		b.WriteString(strconv.FormatInt(int64(*p.AckID), 10))
	}
	if len(p.Data) != 0 {
		b.Write(p.Data)
	}

	return b.Bytes(), nil
}

func UnmarshalPacket(data []byte) (p Packet, err error) {
	err = p.UnmarshalBinary(data)
	return p, err
}

func (p *Packet) UnmarshalBinary(data []byte) error {
	p.Type = PacketType(data[0])
	data = data[1:]
	p.Namespace = "/"
	if len(data) == 0 {
		return nil
	}

	if len(data) != 0 && data[0] == '/' {
		idx := bytes.IndexByte(data, ',')
		p.Namespace = string(data[:idx])
		data = data[idx+1:]
	}

	if ack, idx := readInt(data); ack != nil {
		p.AckID = ack
		data = data[idx:]
	}

	if len(data) != 0 {
		p.Data = data
	}

	return nil
}

func readInt(data []byte) (*int, int) {
	if len(data) == 0 {
		return nil, 0
	}

	var ptr int
	for data[ptr] >= '0' && data[ptr] <= '9' {
		ptr++
	}

	if ptr == 0 {
		return nil, 0
	}

	val, _ := strconv.Atoi(string(data[:ptr]))
	return &val, ptr
}

func Marshal(val any) []byte {
	b, _ := json.Marshal(val)
	return b
}
