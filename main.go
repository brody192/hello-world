package main

import (
	"net/http"
	"time"

	"github.com/brody192/basiclogger"
	"github.com/brody192/ext/handler"
	extmiddleware "github.com/brody192/ext/middleware"
	"github.com/brody192/ext/respond"
	"github.com/brody192/ext/utilities"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
)

func main() {
	var r = chi.NewRouter()

	r.MethodNotAllowed(handler.MethodNotAllowedStatusText)

	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(extmiddleware.LimitBytes(200))
	r.Use(httprate.LimitByIP(10, 15*time.Second))
	r.Use(middleware.NoCache)
	r.Use(cors.AllowAll().Handler)

	handler.MatchMethods(r, []string{http.MethodGet, http.MethodPost, http.MethodHead}, "/*",
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodHead {
				return
			}

			respond.PlainText(w, "Hello, World!", http.StatusOK)
		},
	)

	var port = utilities.EnvPortOr("3001")

	var s = &http.Server{
		Addr:              port,
		Handler:           r,
		ReadTimeout:       1 * time.Minute,
		WriteTimeout:      1 * time.Minute,
		ReadHeaderTimeout: 1 * time.Second,
	}

	basiclogger.InfoBasic.Println("starting server on port " + port[1:])
	basiclogger.Error.Fatalln(s.ListenAndServe())
}
