package handlers

import (
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

func NewRouter(logger *logrus.Logger) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(NewStructuredLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(PrometheusHandler())
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		render.PlainText(w, r, "sup")
	})

	return r
}
