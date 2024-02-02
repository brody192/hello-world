package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	extmiddleware "github.com/brody192/ext/middleware"

	"github.com/brody192/ext/handler"
	"github.com/brody192/ext/respond"
	"github.com/brody192/ext/utilities"
	"github.com/brody192/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	type portKey struct{}

	var r = chi.NewRouter()

	r.MethodNotAllowed(handler.MethodNotAllowedStatusText)

	r.Use(middleware.RealIP)
	r.Use(extmiddleware.Logger(logger.Stdout))
	r.Use(middleware.Recoverer)
	r.Use(extmiddleware.LimitBytes(200))
	r.Use(middleware.NoCache)
	r.Use(cors.AllowAll().Handler)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	})

	handler.MatchMethods(r, []string{http.MethodGet, http.MethodPost, http.MethodHead}, "/*",
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodHead {
				return
			}

			host, _, err := net.SplitHostPort(r.Host)
			if err != nil {
				host = r.Host
			}

			port, ok := r.Context().Value(portKey{}).(string)
			if !ok {
				respond.PlainText(w, "no port found for incoming request", http.StatusInternalServerError)
				return
			}

			greeting := fmt.Sprintf("Hello, World! - Port %s - Host %s", port, host)

			respond.PlainText(w, greeting, http.StatusOK)
		},
	)

	ports := []string{
		strings.TrimPrefix(utilities.EnvPortOr("3000"), ":"),
	}

	if envPorts := strings.TrimSpace(os.Getenv("PORTS")); envPorts != "" {
		if unquotedEnv, err := strconv.Unquote(envPorts); err == nil {
			envPorts = unquotedEnv
		}

		for _, port := range strings.Split(envPorts, ",") {
			port = strings.TrimSpace(port)

			if p, err := strconv.Atoi(port); err != nil || p < 1024 || p > 65535 {
				logger.Stderr.Warn("port from env PORTS invalid", slog.String("invalid_port", port))
				continue
			}

			ports = append(ports, port)
		}
	}

	errChan := make(chan error, 1)

	logger.Stdout.Info("starting server(s)", slog.String("ports", strings.Join(ports, ",")))

	for _, port := range ports {
		go func(port string) {
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

			errChan <- server.ListenAndServe()
		}(port)
	}

	for err := range errChan {
		if err != nil {
			logger.Stderr.Error("server exited with error", logger.ErrAttr(err))
		}
	}
}
