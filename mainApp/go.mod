module dev/mainApp/ot

go 1.15

replace (
	go.opentelemetry.io/otel => ../../../../go/src/github.com/opentelemetry/opentelemetry-go/
	go.opentelemetry.io/otel/exporters/jaeger => ../../../../go/src/github.com/opentelemetry/opentelemetry-go/exporters/jaeger
	go.opentelemetry.io/otel/oteltest => ../../../../go/src/github.com/opentelemetry/opentelemetry-go/oteltest
	go.opentelemetry.io/otel/sdk => ../../../../go/src/github.com/opentelemetry/opentelemetry-go/sdk
	go.opentelemetry.io/otel/trace => ../../../../go/src/github.com/opentelemetry/opentelemetry-go/trace
)

replace go.opentelemetry.io/otel/example/jaeger => ./

require (
	github.com/nats-io/nats.go v1.11.0
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/stretchr/testify v1.7.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama v0.22.0 // indirect
	go.opentelemetry.io/otel v1.0.0-RC2
	go.opentelemetry.io/otel/exporters/jaeger v1.0.0-RC1
	go.opentelemetry.io/otel/sdk v1.0.0-RC2
	go.opentelemetry.io/otel/trace v1.0.0-RC2
)
