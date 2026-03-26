package event

import "encoding/json"

// Event is the envelope written to and read from a Redis stream for every
// Discord gateway DISPATCH event. The stream key encodes the event type
// (e.g. "discord:MESSAGE_CREATE"), so Type is included here for convenience
// when routing after deserialization.
type Event struct {
	Type     string          `json:"type"`
	ShardID  int             `json:"shard_id"`
	Sequence int64           `json:"sequence"`
	Data     json.RawMessage `json:"data"`
}
