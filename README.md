# Kafkabeat

[![Build Status](https://travis-ci.org/arkady-emelyanov/kafkabeat.svg?branch=master)](https://travis-ci.org/arkady-emelyanov/kafkabeat)

Kafkabeat is an [Elastic Beat](https://www.elastic.co/products/beats) that read events from Kafka topics and 
forward them to any [supported output](https://www.elastic.co/guide/en/beats/filebeat/6.3/configuring-output.html).

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

  # Event publish mode: "default", "send" or "drop_if_full".
  # Defaults to "default"
  # @see https://github.com/elastic/beats/blob/v6.3.1/libbeat/beat/pipeline.go#L119
  # for detailed explanation.
  #publish_mode: "default"
```

### Timestamp

For plain codec, timestamp field will be set either as provided by Kafka message (requires Kafka 0.10+),
or as current time.

For json codec, before fallback to Kafka message timestamp, top-level field "@timestamp" 
with expected layout `"2006-01-02T15:04:05.000Z"` will be analyzed.

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

## Building, Running and Packaging Kafkabeat

### Requirements

* [Golang](https://golang.org/dl/) 1.7+

### Build

Ensure that this folder is at the following location:
`${GOPATH}/src/github.com/arkady-emelyanov/kafkabeat`

To build the binary for Kafkabeat run the command below. This will generate a binary
in the same directory with the name kafkabeat.

```
make
```

### Run

To run Kafkabeat with debugging output enabled, run:

```
docker-compose up
./kafkabeat -c kafkabeat.docker-compose.yml -e -d "*"
```

### Update

Each beat has a template for the mapping in elasticsearch and a documentation for the fields
which is automatically generated based on `fields.yml` by running the following command.

```
make update
```

### Cleanup

To clean up the build directory and generated artifacts, run:

```
make clean
```

### Clone

To clone Kafkabeat from the git repository, run the following commands:

```
mkdir -p ${GOPATH}/src/github.com/arkady-emelyanov/kafkabeat
git clone https://github.com/arkady-emelyanov/kafkabeat ${GOPATH}/src/github.com/arkady-emelyanov/kafkabeat
```


For further development, 
check out the [beat developer guide](https://www.elastic.co/guide/en/beats/libbeat/current/new-beat.html).


## Packaging

The beat frameworks provides tools to crosscompile and package your 
beat for different platforms. This requires [docker](https://www.docker.com/) 
and vendoring as described above. To build packages of your beat, 
run the following command:

```
make package
```

This will fetch and create all images required for the build process. 
The whole process to finish can take several minutes.
