package beater

import (
	"encoding/json"
	"time"

	"github.com/Shopify/sarama"
	"github.com/elastic/beats/libbeat/beat"
)

// Decoder decoder interface
type decoder interface {
	Decode(msg *sarama.ConsumerMessage) *beat.Event
}

type jsonDecoder struct {
	timestampKey    string
	timestampLayout string
	timeNowFn       func() time.Time
}

// JSON decoder
func newJSONDecoder(timestampKey, timestampLayout string) *jsonDecoder {
	return &jsonDecoder{
		timestampKey:    timestampKey,
		timestampLayout: timestampLayout,
		timeNowFn:       time.Now,
	}
}

func (d *jsonDecoder) Decode(msg *sarama.ConsumerMessage) *beat.Event {
	fields := map[string]interface{}{}
	if err := json.Unmarshal(msg.Value, &fields); err != nil {
		return nil
	}

	// special @timestamp field handling
	var ts time.Time
	if val, exists := fields[d.timestampKey]; exists {
		delete(fields, d.timestampKey) // drop timestamp key

		if s, ok := val.(string); ok {
			if p, err := time.Parse(d.timestampLayout, s); err == nil {
				ts = time.Time(p)
			}
		}
	}

	if ts.IsZero() {
		if msg.Timestamp.IsZero() {
			ts = d.timeNowFn()
		} else {
			ts = msg.Timestamp
		}
	}

	return &beat.Event{
		Timestamp: ts,
		Fields:    fields,
	}
}

// Plain decoder
type plainDecoder struct {
	timeNowFn func() time.Time
}

func newPlainDecoder() *plainDecoder {
	return &plainDecoder{
		timeNowFn: time.Now,
	}
}

func (d *plainDecoder) Decode(msg *sarama.ConsumerMessage) *beat.Event {
	fields := map[string]interface{}{
		"message": string(msg.Value),
	}

	ts := msg.Timestamp
	if ts.IsZero() {
		ts = d.timeNowFn()
	}

	return &beat.Event{
		Timestamp: ts,
		Fields:    fields,
	}
}
