package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo-contrib/jaegertracing"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func handleRoot(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World")
}

func Fibonacci(n int) (uint64, error) {
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
	num, _ := strconv.Atoi(c.Param("num"))
	fibo, _ := Fibonacci(num)

	time.Sleep(40 * time.Millisecond)
	sp := jaegertracing.CreateChildSpan(c, "100ms sleep")
	defer sp.Finish()

	time.Sleep(100 * time.Millisecond)

	return c.String(http.StatusOK, strconv.FormatUint(fibo, 10))
}

func main() {
	e := echo.New()
	e.GET("/", handleRoot)
	e.GET("/fibo/:num", handleGetFibonacci)
	e.Use(middleware.Recover())

	c := jaegertracing.New(e, nil)
	defer c.Close()

	e.Logger.Fatal(e.Start(":3000"))
}
