package socketio

import "encoding/json"

var _ DataCodec = jsonCodec{}

type jsonCodec struct{}

func NewJSONCodec() jsonCodec {
	return jsonCodec{}
}

func (jsonCodec) MarashalJSON(data any) ([]byte, error) {
	return json.Marshal(data)
}

func (jsonCodec) UnmarshalJSONTo(data []byte, to any) error {
	return json.Unmarshal(data, to)
}
