package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

func (s *server) recoverPanic(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				s.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()

		next(w, r)
	}
}

func (s *server) rateLimit(next http.HandlerFunc) http.HandlerFunc {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(w http.ResponseWriter, r *http.Request) {
		if s.limiter.enabled {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				s.serverErrorResponse(w, r, err)
				return
			}

			mu.Lock()

			if _, found := clients[ip]; !found {
				clients[ip] = &client{limiter: rate.NewLimiter(2, 4)}
			}

			clients[ip].lastSeen = time.Now()

			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				s.rateLimitExceededResponse(w, r)
				return
			}

			mu.Unlock()

			next(w, r)
		}
	}
}
