// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import "runtime"

type Config struct {
	Brokers           []string `config:"brokers"`
	Topics            []string `config:"topics"`
	ClientID          string   `config:"client_id"`
	Group             string   `config:"group"`
	Offset            string   `config:"offset"`
	Codec             string   `config:"codec"`
	PublishMode       string   `config:"publish_mode"`
	ChannelBufferSize int      `config:"channel_buffer_size"`
	ChannelWorkers    int      `config:"channel_workers"`
}

var DefaultConfig = Config{
	Brokers:           []string{"localhost:9092"},
	Topics:            []string{"watch"},
	ClientID:          "beat",
	Group:             "kafkabeat",
	Offset:            "newest",
	Codec:             "json",
	PublishMode:       "default",
	ChannelBufferSize: 256,
	ChannelWorkers:    runtime.NumCPU(),
}
