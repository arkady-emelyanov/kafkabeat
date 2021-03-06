# Kafkabeat

[![Build Status](https://travis-ci.org/arkady-emelyanov/kafkabeat.svg?branch=master)](https://travis-ci.org/arkady-emelyanov/kafkabeat)

Kafkabeat is an [Elastic Beat](https://www.elastic.co/products/beats) that read events from Kafka topics and 
forward them to any [supported output](https://www.elastic.co/guide/en/beats/filebeat/6.3/configuring-output.html).

* Beats: v6.4.0
* Sarama: 1.18.0 (Kafka 2.0.0 supported) 

## How it works?

Kafkabeat is supporting two event processing modes via so-called codecs: `plain` and `json`.

Plain codec is a dumb codec, kafka message value is converted into string and forwarded. For example,
direct output to ElasticSearch for kafka message: `{"hello": "world"}` gives you document:

```
{
    "@timestamp": ...
    "message": "{\"hello\": \"world\"}",
    ...
}
```

JSON codec instead, will unpack kafka message value:
```
{
    "@timestamp": ...
    "hello": "world",
    ...
}
```

It's quite useful in combination with Kafka Streams.


### Configuration

```yaml
kafkabeat:
  # Brokers to connect to.
  # Defaults to ["localhost:9092"].
  brokers: ["localhost:9092"]

  # Topics to fetch data.
  # Defaults to ["watch"].
  topics: ["watch"]

  # Consumer ClientID. Defaults to beat.
  client_id: "beat"

  # Consumer group.
  group: "kafkabeat"

  # The initial offset to use if no offset was previously committed.
  # Should be "newest" or "oldest". Defaults to "newest".
  offset: "newest"

  # Codec to use. Can be either "plain" or "json".
  # @see README.md for detailed explanation.
  # Defaults to "json".
  codec: "json"

  # Timestamp key used by JSON decoder
  #timestamp_key: "@timestamp"

  # Timestamp layout used by JSON decoder
  #timestamp_layout: "2006-01-02T15:04:05.000Z"

  # Event publish mode: "default", "send" or "drop_if_full".
  # Defaults to "default"
  # @see https://github.com/elastic/beats/blob/v6.3.1/libbeat/beat/pipeline.go#L119
  # for detailed explanation.
  #publish_mode: "default"

  # Channel buffer size.
  # Defaults to 256
  # @see https://github.com/Shopify/sarama/blob/v1.17.0/config.go#L262
  # for detailed explanation
  #channel_buffer_size: 256
  
  # Number of concurrent publish workers.
  # General suggestion keep number of workers equal or lower than CPU cores available.
  # Defaults to number of available cores
  #channel_workers: 8
```

### Timestamp

For plain codec, timestamp field will be set either as provided by Kafka message (requires Kafka 0.10+),
or as current time.

For json codec, before fallback to Kafka message timestamp, top-level field defined on configuration parameter `timestamp_key` (defaults to `"@timestamp"`)
with layout defined on configuration parameter `timestamp_layout` (defaults to `"2006-01-02T15:04:05.000Z"`) will be analyzed.

### Examples

For given sample event:
```json
{"@timestamp":"2018-07-13T13:51:20.177Z","message":"hello kafkabeat!","nested":{"beat":"kafkabeat"}}
```

Json codec will emit following event (Elastic output):
```json
{
    "_index" : "kafkabeat-6.3.1-2018.07.13",
    "_type" : "doc",
    "_id" : "AWSUEi7911YtWZ9JcUG4",
    "_score" : 1.0,
    "_source" : {
        "@timestamp" : "2018-07-13T13:51:20.177Z",
        "message" : "hello kafkabeat!",
        "nested" : {
            "beat" : "kafkabeat"
        },
        "beat" : {
            "name" : "example.com",
            "hostname" : "example.com",
            "version" : "6.3.1"
        },
        "host" : {
            "name" : "example.com"
        }
    }
}
```

Plain codec (Elastic output):
```json
{
    "_index" : "kafkabeat-6.3.1-2018.07.13",
    "_type" : "doc",
    "_id" : "AWSUFGvE11YtWZ9JcUG5",
    "_score" : 1.0,
    "_source" : {
        "@timestamp" : "2018-07-13T14:38:27.808Z",
        "message" : "{\"@timestamp\":\"2018-07-13T13:51:20.177Z\",\"message\":\"hello kafkabeat!\",\"nested\":{\"beat\":\"kafkabeat\"}}",
        "beat" : {
            "version" : "6.3.1",
            "name" : "example.com",
            "hostname" : "example.com"
        },
        "host" : {
            "name" : "example.com"
        }
    }
}
```


### Notes about field mappings

In case of json, it's quite handy to disable template upload, by providing top-level
configuration option: `setup.template.enabled: false` and manually manage mappings.

