package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	//"go.opentelemetry.io/otel/trace"
	//"go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama"
)

type parentContext struct {
	TraceID    string
	SpanID     string
	TraceFlags string
	TraceState string
	Remote     bool
}

const (
	service     = "trace-demo"
	environment = "production"
	id          = 1
)

// tracerProvider returns an OpenTelemetry TracerProvider configured to use
// the Jaeger exporter that will send spans to the provided url. The returned
// TracerProvider will also use a Resource configured with all the information
// about the application.
func tracerProvider(url string) (*tracesdk.TracerProvider, error) {
	// Create the Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(url)))
	if err != nil {
		return nil, err
	}

	tp := tracesdk.NewTracerProvider(
		// Always be sure to batch in production.
		tracesdk.WithBatcher(exp),
		tracesdk.WithSampler(tracesdk.AlwaysSample()),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(service),
			attribute.String("environment", environment),
			attribute.Int64("ID", id),
		)),
	)
	return tp, nil
}

func usage() {
	log.Printf("Usage: child [-s server] [-creds file] [-t] <subject>\n")
	flag.PrintDefaults()
}

func showUsageAndExit(exitcode int) {
	usage()
	os.Exit(exitcode)
}

func printMsg(m *nats.Msg, i int) {
	log.Printf("[#%d] Received on [%s]: '%s'", i, m.Subject, string(m.Data))
}

func main() {
	var urls = flag.String("s", nats.DefaultURL, "The nats server URLs (separated by comma)")
	var userCreds = flag.String("creds", "", "User Credentials File")
	var showTime = flag.Bool("t", false, "Display timestamps")
	var showHelp = flag.Bool("h", false, "Show help message")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	if *showHelp {
		showUsageAndExit(0)
	}

	args := flag.Args()
	if len(args) != 1 {
		showUsageAndExit(1)
	}

	// Connect Options.
	opts := []nats.Option{nats.Name("NATS Sample Subscriber")}
	opts = setupConnOptions(opts)

	// Use UserCredentials
	if *userCreds != "" {
		opts = append(opts, nats.UserCredentials(*userCreds))
	}

	// Connect to NATS
	nc, err := nats.Connect(*urls, opts...)
	if err != nil {
		log.Fatal(err)
	}

	subj, i := args[0], 0

	nc.Subscribe(subj, func(msg *nats.Msg) {
		tp, err := tracerProvider("http://localhost:14268/api/traces")
		if err != nil {
			log.Fatal(err)
		}

		// Register our TracerProvider as the global so any imported
		// instrumentation in the future will default to using it.
		otel.SetTracerProvider(tp)
		propagator := propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{})
		otel.SetTextMapPropagator(propagator)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Cleanly shutdown and flush telemetry when the application exits.
		defer func(ctx context.Context) {
			// Do not make the application hang when it is shutdown.
			ctx, cancel = context.WithTimeout(ctx, time.Second*5)

			defer cancel()
			if err := tp.Shutdown(ctx); err != nil {
				log.Fatal(err)
			}
		}(ctx)
		//tr := tp.Tracer("ads-validator")
		i += 1
		log.Print("ParentSpanId: ", msg.Header.Values("parentSpanId"))
		log.Print("Span Context from header: ", msg.Header.Values("spanContext"))
		log.Print("Span Context from header[0]: ", msg.Header.Values("spanContext")[0])
		spanCtxt := parentContext{}
		err = json.Unmarshal([]byte(msg.Header.Values("spanContext")[0]), &spanCtxt)
		log.Print("Span Context config ", spanCtxt)
		if err != nil {
			log.Fatal(err)
		}
		ctx = context.Background()
		ctx = context.WithValue(ctx, "TraceID", spanCtxt.TraceID)
		ctx = context.WithValue(ctx, "SpanID", spanCtxt.SpanID)
		ctx = context.WithValue(ctx, "TraceFlags", spanCtxt.TraceFlags)
		ctx = context.WithValue(ctx, "TraceState", spanCtxt.TraceState)
		ctx = context.WithValue(ctx, "Remote", false)

		//tr := tp.Tracer("message-handler")
		//spanCtx := trace.NewSpanContext(spanCtxtConfig)
		//span := trace.SpanFromContext(ctx)
		//_, span := tr.Start(ctx, "messageConsumerSpan")

		//ctx = otel.GetTextMapPropagator().Extract(context.Background(), otelsarama.NewConsumerMessageCarrier(&msg))

		tr := tp.Tracer("consumer")
		_, span := tr.Start(ctx, "consume message", trace.WithAttributes(
			semconv.MessagingOperationProcess,
		))
		defer span.End()

		log.Print("Is valid:", span.SpanContext().IsValid())
		spanContextJson, err := span.SpanContext().MarshalJSON()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("child Span context is : %v", string(spanContextJson))
		span.End()
		tp.ForceFlush(ctx)
		printMsg(msg, i)
	})
	nc.Flush()

	if err := nc.LastError(); err != nil {
		log.Fatal(err)
	}

	log.Printf("Listening on [%s]", subj)
	if *showTime {
		log.SetFlags(log.LstdFlags)
	}

	runtime.Goexit()
}

func setupConnOptions(opts []nats.Option) []nats.Option {
	totalWait := 10 * time.Minute
	reconnectDelay := time.Second

	opts = append(opts, nats.ReconnectWait(reconnectDelay))
	opts = append(opts, nats.MaxReconnects(int(totalWait/reconnectDelay)))
	opts = append(opts, nats.DisconnectHandler(func(nc *nats.Conn) {
		log.Printf("Disconnected: will attempt reconnects for %.0fm", totalWait.Minutes())
	}))
	opts = append(opts, nats.ReconnectHandler(func(nc *nats.Conn) {
		log.Printf("Reconnected [%s]", nc.ConnectedUrl())
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		log.Fatalf("Exiting: %v", nc.LastError())
	}))
	return opts
}
