package main

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

const (
	service     = "otel-echo"
	environment = "dev"
	id          = 1
)

const fiboname = "otel-fibo"

func tracerProvider(url string) (*tracesdk.TracerProvider, error) {
	// Create the Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(url)))
	if err != nil {
		return nil, err
	}
	tp := tracesdk.NewTracerProvider(
		// Always be sure to batch in production.
		tracesdk.WithBatcher(exp),
		// Record information about this application in a Resource.
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(service),
			attribute.String("environment", environment),
			attribute.Int64("ID", id),
		)),
	)
	return tp, nil
}

func handleRoot(c echo.Context) error {
	_, span := otel.Tracer("handleRoot").Start(context.Background(), "handleGetFibonacci")
	defer span.End()
	return c.String(http.StatusOK, "Hello, World")
}

func Fibonacci(n int, c context.Context) (uint64, error) {
	if n <= 1 {
		return uint64(n), nil
	}

	var n2, n1 uint64 = 0, 1
	for i := int(2); i < n; i++ {
		n2, n1 = n1, n1+n2
	}

	return n2 + n1, nil
}

func handleGetFibonacci(c echo.Context) error {
	newCtx, span := otel.Tracer(fiboname).Start(context.Background(), "handleGetFibonacci")
	defer span.End()

	num, _ := strconv.Atoi(c.Param("num"))
	fibo, _ := Fibonacci(num, newCtx)

	_, span2 := otel.Tracer(fiboname).Start(newCtx, "40 ms span")
	time.Sleep(40 * time.Millisecond)
	span2.End()

	_, span3 := otel.Tracer(fiboname).Start(newCtx, "100 ms span")
	defer span3.End()
	time.Sleep(100 * time.Millisecond)

	return c.String(http.StatusOK, strconv.FormatUint(fibo, 10))
}

func main() {

	tp, err := tracerProvider("http://localhost:14268/api/traces")
	if err != nil {
		log.Fatal(err)
	}

	// Register our TracerProvider as the global so any imported
	// instrumentation in the future will default to using it.
	otel.SetTracerProvider(tp)

	e := echo.New()
	e.GET("/", handleRoot)
	e.GET("/fibo/:num", handleGetFibonacci)
	e.Use(middleware.Recover())

	e.Logger.Fatal(e.Start(":3000"))
}
