package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type Request struct {
	Payload string
}

type Client interface {
	SendRequest(ctx context.Context, request Request) error
	WithLimiter(ctx context.Context, requests []Request)
}
type client struct {
}

func (c *client) SendRequest(ctx context.Context, request Request) error {
	time.Sleep(100 * time.Millisecond)
	fmt.Println("sending request", request.Payload)
	return nil
}

var rps = 100

func (c *client) WithLimiter(ctx context.Context, requests []Request) {
	ticker := time.NewTicker(time.Second / time.Duration(rps))
	for _, v := range requests {
		<-ticker.C
		go func() {
			c.SendRequest(ctx, v)
		}()
	}
}

// TODO С ОГРАНИЧЕНИЕМ ОДНОВРЕМЕННО РАБОТАЮЩИХ ГОРУТИН
var gorLimit = 100

/*func (c *client) WithLimiter(ctx context.Context, requests []Request) {
	tokens := make(chan struct{}, gorLimit)

	for range gorLimit {
		tokens <- struct{}{}
	}

	for _, v := range requests {
		<-tokens
		go func() {
			c.SendRequest(ctx, v)
			tokens <- struct{}{}
		}()
	}

	for range gorLimit {
		<-tokens
	}
}*/

func main() {
	ctx := context.Background()
	c := client{}
	req := make([]Request, 1000)
	for i := 0; i < 1000; i++ {
		req[i] = Request{Payload: strconv.Itoa(i)}
	}
	c.WithLimiter(ctx, req)

}

// TODO ПРИМЕНЕНИЕ ДЛЯ ОГРАНИЧЕНИЯ ЗАПРОСОВ В СЕКУНДУ ДЛЯ MIDDLEWARE
var tokens chan struct{}

func Init(max int) {
	tokens = make(chan struct{}, max)

	for i := 0; i < max; i++ {
		tokens <- struct{}{}
	}
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-tokens:
			defer func() {
				tokens <- struct{}{}
			}()
			next.ServeHTTP(w, r)
		case <-time.After(3 * time.Second):
			http.Error(w, "server busy", http.StatusTooManyRequests)
			return
		}
	})
}
