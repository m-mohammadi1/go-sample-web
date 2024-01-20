package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (app *application) routes() http.Handler {
	mux := chi.NewRouter()

	// register middleware
	mux.Use(middleware.Recoverer)
	// enable cors

	// authentication routes - auth and refresh handler
	mux.Post("/auth", app.authenticate)
	mux.Post("/refresh-token", app.refresh)

	// test handlers
	mux.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		var payload = struct {
			Message string `json:"message"`
		}{
			Message: "Hello World",
		}

		_ = app.writeJSON(w, http.StatusOK, payload)
	})

	// protected routes
	mux.Route("/users", func(mux chi.Router) {

		mux.Get("/", app.allUsers)
		mux.Get("/{userID}", app.getUser)
		mux.Put("/", app.insertUser)
		mux.Patch("/", app.updateUser)

	})

	return mux
}
