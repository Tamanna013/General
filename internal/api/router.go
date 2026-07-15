package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(h *Handlers) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(loggerMiddleware)
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/objects", h.UploadObject)
		r.Get("/objects/{id}", h.GetObject)
		r.Delete("/objects/{id}", h.DeleteObject)
		r.Get("/objects/{id}/download", h.DownloadObject)

		r.Post("/objects/{id}/versions", h.UploadVersion)
		r.Get("/objects/{id}/versions", h.ListVersions)
		r.Get("/objects/{id}/versions/{versionId}", h.GetVersion)
		r.Delete("/objects/{id}/versions/{versionId}", h.DeleteVersion)
		r.Get("/objects/{id}/versions/{versionId}/download", h.DownloadVersion)
		r.Post("/objects/{id}/versions/{versionId}/rollback", h.RollbackVersion)
	})

	return r
}

func loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		
		next.ServeHTTP(ww, r)
		
		slog.Info("request", 
			"method", r.Method, 
			"path", r.URL.Path, 
			"status", ww.Status(), 
			"duration", time.Since(start),
		)
	})
}
