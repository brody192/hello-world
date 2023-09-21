package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/brody192/ext/exthandler"
	"github.com/brody192/ext/extmiddleware"
	"github.com/brody192/ext/extrespond"
	"github.com/brody192/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"golang.org/x/exp/slog"
)

type portKey struct{}

func main() {
	var r = chi.NewRouter()

	r.MethodNotAllowed(exthandler.MethodNotAllowedStatusText)

	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(extmiddleware.LimitBytes(200))
	r.Use(middleware.NoCache)
	r.Use(cors.AllowAll().Handler)

	exthandler.MatchMethods(r, []string{http.MethodGet, http.MethodPost, http.MethodHead}, "/*",
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodHead {
				return
			}

			port, ok := r.Context().Value(portKey{}).(string)
			if !ok {
				extrespond.PlainText(w, "no port found for incoming request", http.StatusInternalServerError)
				return
			}

			greeting := fmt.Sprintf("Hello, World! - Port %s", port)

			extrespond.PlainText(w, greeting, http.StatusOK)
		},
	)

	ports := []string{"8000", "8001", "8002", "8003", "8004", "8005"}

	errChan := make(chan error, 1)

	for _, port := range ports {
		server := &http.Server{
			Addr:    ":" + port,
			Handler: r,
			ConnContext: func(ctx context.Context, c net.Conn) context.Context {
				_, port, err := net.SplitHostPort(c.LocalAddr().String())
				if err != nil {
					return ctx
				}

				return context.WithValue(ctx, portKey{}, port)

			},
			ReadTimeout:       1 * time.Minute,
			WriteTimeout:      1 * time.Minute,
			ReadHeaderTimeout: 1 * time.Second,
		}

		go func(port string) {
			logger.Stdout.Info("starting server", slog.String("port", port))
			errChan <- server.ListenAndServe()
		}(port)
	}

	if err := <-errChan; err != nil {
		logger.Stderr.Error("server exited", logger.ErrAttr(err))
	}
}
