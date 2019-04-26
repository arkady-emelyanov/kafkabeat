// +build !integration

package beater

import (
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/elastic/beats/libbeat/common"
)

var (
	testNowValue = time.Date(1981, time.May, 5, 10, 15, 35, 9583543, time.UTC)
)

func newTestJSONDecoder() decoder {
	return &jsonDecoder{
		timestampKey:    "@timestamp",
		timestampLayout: common.TsLayout,
		timeNowFn: func() time.Time {
			return testNowValue
		},
	}
}

func newTestPlainDecoder() decoder {
	return &plainDecoder{
		timeNowFn: func() time.Time {
			return testNowValue
		},
	}
}

func TestJSONDecoderWithoutTimestamp(t *testing.T) {
	d := newTestJSONDecoder()

	msg := &sarama.ConsumerMessage{
		Value: []byte(`{ "field": "value" }`),
	}
	e := d.Decode(msg)

	if e == nil {
		t.Fatal("Event must be generated")
	}
	if e.Fields["field"] != "value" {
		t.Error("Expected field=value keypair, but not found on event")
	}
	if e.Timestamp != testNowValue {
		t.Errorf("Expected %v", testNowValue)
		t.Errorf("   found %v", e.Timestamp)
	}
}

func TestJSONDecoderWithMessageTimestamp(t *testing.T) {
	ts := time.Date(2019, time.April, 26, 17, 16, 10, 945958000, time.UTC)
	d := newTestJSONDecoder()
	msg := &sarama.ConsumerMessage{
		Value:     []byte(`{ "field": "value" }`),
		Timestamp: ts,
	}
	e := d.Decode(msg)

	if e == nil {
		t.Fatal("Event must be generated")
	}
	if e.Fields["field"] != "value" {
		t.Error("Expected field=value keypair, but not found on event")
	}
	if e.Timestamp != ts {
		t.Errorf("Expected %v", ts)
		t.Errorf("   found %v", e.Timestamp)
	}
}

func TestJSONDecoderWithValidJSONTimestamp(t *testing.T) {
	ts := time.Date(2019, time.April, 26, 17, 16, 10, 945000000, time.UTC)
	d := newTestJSONDecoder()
	msg := &sarama.ConsumerMessage{
		Value: []byte(`{ "@timestamp": "2019-04-26T17:16:10.945Z", "field": "value" }`),
	}
	e := d.Decode(msg)

	if e == nil {
		t.Fatal("Event must be generated")
	}
	if e.Fields["field"] != "value" {
		t.Error("Expected field=value keypair, but not found on event")
	}
	if e.Timestamp != ts {
		t.Errorf("Expected %v", ts)
		t.Errorf("   found %v", e.Timestamp)
	}
}

func TestJSONDecoderWithInvalidJSONTimestamp(t *testing.T) {
	d := newTestJSONDecoder()
	msg := &sarama.ConsumerMessage{
		Value: []byte(`{ "@timestamp": "2019-04-26T17:16:10.945759Z", "field": "value" }`),
	}
	e := d.Decode(msg)

	if e == nil {
		t.Fatal("Event must be generated")
	}
	if e.Fields["field"] != "value" {
		t.Error("Expected field=value keypair, but not found on event")
	}
	if e.Timestamp != testNowValue {
		t.Errorf("Expected %v", testNowValue)
		t.Errorf("   found %v", e.Timestamp)
	}
}

func TestPlainDecoderWithoutTimestamp(t *testing.T) {
	d := newTestPlainDecoder()
	msg := &sarama.ConsumerMessage{
		Value: []byte(`mymessage`),
	}
	e := d.Decode(msg)

	if e == nil {
		t.Fatal("Event must be generated")
	}
	if e.Fields["message"] != "mymessage" {
		t.Error("Expected message=mymessage keypair, but not found on event")
	}
	if e.Timestamp != testNowValue {
		t.Errorf("Expected %v", testNowValue)
		t.Errorf("   found %v", e.Timestamp)
	}
}

func TestPlainDecoderWithMessageTimestamp(t *testing.T) {
	ts := time.Date(2019, time.April, 26, 17, 16, 10, 945958000, time.UTC)
	d := newTestPlainDecoder()
	msg := &sarama.ConsumerMessage{
		Value:     []byte(`mymessage`),
		Timestamp: ts,
	}
	e := d.Decode(msg)

	if e == nil {
		t.Fatal("Event must be generated")
	}
	if e.Fields["message"] != "mymessage" {
		t.Error("Expected message=mymessage keypair, but not found on event")
	}
	if e.Timestamp != ts {
		t.Errorf("Expected %v", ts)
		t.Errorf("   found %v", e.Timestamp)
	}
}
