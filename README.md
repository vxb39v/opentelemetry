# opentelemetry

# Setup
## Jaeger
Install Jaeger All in One docker image using this guide 
https://www.jaegertracing.io/docs/1.25/getting-started/

Or by copying the following to your terminal

```
docker run -d --name jaeger \
  -e COLLECTOR_ZIPKIN_HOST_PORT=:9411 \
  -p 5775:5775/udp \
  -p 6831:6831/udp \
  -p 6832:6832/udp \
  -p 5778:5778 \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 14250:14250 \
  -p 9411:9411 \
  jaegertracing/all-in-one:1.25

```


## Nats
Install Nats docker image using this guide
https://docs.nats.io/nats-server/installation


Or by copying the following to your terminal


```
docker pull nats:latest
docker run -p 4222:4222 -ti nats:latest
```

# Build

go into the opentelemetry project then

```
cd childApp
go build -a -o child

cd ../mainApp
go build -a -o main
```

# Run


Start the childApp first, as this is the message listener


Usage: child [-s server] [-creds file] [-t] <subject>

```
./child -s nats://127.0.0.1:4222 -t test_queue
```


Main app is used to publish messages

Usage: main [-s server] [-creds file] <subject> <msg>

```
./main -s nats://127.0.0.1:4222 test_queue 'Hello World'
```


# Verify

Verify the logs in Jaeger using a browser and navigating to 


http://localhost:16686/

