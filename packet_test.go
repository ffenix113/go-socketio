package socketio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPacket_UnmarshalBinary(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want Packet
		err  error
	}{
		{
			name: "Connect",
			data: []byte("0"),
			want: Packet{
				Type:      PacketTypeConnect,
				Namespace: "/",
			},
		},
		{
			name: "Connect with namespace",
			data: []byte("0/test,"),
			want: Packet{
				Type:      PacketTypeConnect,
				Namespace: "/test",
			},
		},
		{
			name: "Message with data",
			data: []byte(`2{"data": true}`),
			want: Packet{
				Type:      PacketTypeEvent,
				Namespace: "/",
				Data:      []byte(`{"data": true}`),
			},
		},
		{
			name: "Message with ack and data",
			data: []byte(`20{"data": true}`),
			want: Packet{
				Type:      PacketTypeEvent,
				Namespace: "/",
				AckID:     ptr(0),
				Data:      []byte(`{"data": true}`),
			},
		},
		{
			name: "Message with namespace, ack and data",
			data: []byte(`2/a,255{"data": true}`),
			want: Packet{
				Type:      PacketTypeEvent,
				Namespace: "/a",
				AckID:     ptr(255),
				Data:      []byte(`{"data": true}`),
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Run("Unmarshal", func(t *testing.T) {
				p, err := UnmarshalPacket(test.data)
				assert.Equal(t, test.err, err)
				assert.Equal(t, test.want, p)
			})
			t.Run("Marshal", func(t *testing.T) {
				data, err := test.want.MarshalBinary()
				assert.Equal(t, nil, err)
				assert.Equal(t, test.data, data)
			})
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}
