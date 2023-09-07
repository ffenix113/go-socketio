package engineio

import (
	"bytes"
	"encoding/json"
)

var probeBts = []byte("probe")

var pingPacketBts = func() []byte {
	b, _ := (&Packet{Type: PacketTypePing}).MarshalBinary()
	return b
}()

var closePacketBts = func() []byte {
	b, _ := (&Packet{Type: PacketTypeClose}).MarshalBinary()
	return b
}()

var PacketSeparator = []byte{'\x1e'}

const (
	PacketTypeOpen    PacketType = "0"
	PacketTypeClose   PacketType = "1"
	PacketTypePing    PacketType = "2"
	PacketTypePong    PacketType = "3"
	PacketTypeMessage PacketType = "4"
	PacketTypeUpgrade PacketType = "5"
	PacketTypeNoop    PacketType = "6"
)

type PacketType string

type Packet struct {
	Type PacketType
	Data json.RawMessage
}

type Packets []Packet

func (p Packets) MarshalBinary() (data []byte, err error) {
	var b bytes.Buffer

	for i := range p {
		if i != 0 {
			b.Write(PacketSeparator)
		}

		m, _ := p[i].MarshalBinary()
		b.Write(m)
	}

	return b.Bytes(), nil
}

func (p *Packets) UnmarshalBinary(data []byte) error {
	bytesPckt := bytes.Split(data, PacketSeparator)

	*p = make(Packets, 0, len(bytesPckt))
	for _, bts := range bytesPckt {
		pckt := Packet{}
		pckt.UnmarshalBinary(bts)
		*p = append(*p, pckt)
	}

	return nil
}

func (p Packet) MarshalBinary() (data []byte, err error) {
	if len(p.Data) == 0 {
		return []byte(p.Type), nil
	}

	return []byte(string(p.Type) + string(p.Data)), nil
}

func (p *Packet) UnmarshalBinary(data []byte) error {
	p.Type = PacketType(data[0])
	p.Data = data[1:]

	return nil
}
