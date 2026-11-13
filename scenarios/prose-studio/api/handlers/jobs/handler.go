package jobs

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	internal "prose-studio/internal/jobs"
)

type Handler struct{ service *internal.Service }

func New(service *internal.Service) *Handler { return &Handler{service: service} }
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func bad(w http.ResponseWriter, e error) {
	status := http.StatusInternalServerError
	if errors.Is(e, internal.ErrInvalid) {
		status = http.StatusBadRequest
	} else if errors.Is(e, sql.ErrNoRows) {
		status = http.StatusNotFound
	}
	write(w, status, map[string]string{"error": e.Error()})
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	d.DisallowUnknownFields()
	return d.Decode(v)
}
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	path := strings.Trim(strings.TrimPrefix(strings.TrimPrefix(r.URL.Path, "/api/v1/jobs/"), "/api/v1/jobs"), "/")
	switch path {
	case "openings":
		if r.Method == http.MethodGet {
			v, e := h.service.Openings(ctx)
			if e != nil {
				bad(w, e)
				return
			}
			write(w, 200, map[string]any{"openings": v})
			return
		}
		if r.Method == http.MethodPost {
			var v internal.Opening
			if e := decode(w, r, &v); e != nil {
				bad(w, e)
				return
			}
			out, e := h.service.CreateOpening(ctx, v)
			if e != nil {
				bad(w, e)
				return
			}
			write(w, 201, map[string]any{"opening": out})
			return
		}
	case "facts":
		if r.Method == http.MethodGet {
			v, e := h.service.Facts(ctx)
			if e != nil {
				bad(w, e)
				return
			}
			write(w, 200, map[string]any{"facts": v})
			return
		}
		if r.Method == http.MethodPost {
			var v internal.Fact
			if e := decode(w, r, &v); e != nil {
				bad(w, e)
				return
			}
			out, e := h.service.CreateFact(ctx, v)
			if e != nil {
				bad(w, e)
				return
			}
			write(w, 201, map[string]any{"fact": out})
			return
		}
	case "resumes":
		if r.Method == http.MethodGet {
			v, e := h.service.Resumes(ctx)
			if e != nil {
				bad(w, e)
				return
			}
			write(w, 200, map[string]any{"resumes": v})
			return
		}
		if r.Method == http.MethodPost {
			var v internal.Resume
			if e := decode(w, r, &v); e != nil {
				bad(w, e)
				return
			}
			out, e := h.service.CreateResume(ctx, v)
			if e != nil {
				bad(w, e)
				return
			}
			write(w, 201, map[string]any{"resume": out})
			return
		}
	case "cover-letters":
		if r.Method == http.MethodGet {
			v, e := h.service.Letters(ctx)
			if e != nil {
				bad(w, e)
				return
			}
			write(w, 200, map[string]any{"cover_letters": v})
			return
		}
		if r.Method == http.MethodPost {
			var v internal.CoverLetter
			if e := decode(w, r, &v); e != nil {
				bad(w, e)
				return
			}
			out, e := h.service.CreateLetter(ctx, v)
			if e != nil {
				bad(w, e)
				return
			}
			write(w, 201, map[string]any{"cover_letter": out})
			return
		}
	case "sources":
		if r.Method == http.MethodGet {
			write(w, 200, map[string]any{"sources": h.service.Sources()})
			return
		}
	case "discover":
		if r.Method == http.MethodPost {
			var in struct {
				Query string `json:"query"`
			}
			if e := decode(w, r, &in); e != nil {
				bad(w, e)
				return
			}
			out, e := h.service.Discover(ctx, in.Query)
			if e != nil {
				bad(w, e)
				return
			}
			write(w, 200, out)
			return
		}
	}
	if strings.HasPrefix(path, "openings/") && r.Method == http.MethodPatch {
		id := strings.TrimPrefix(path, "openings/")
		var patch map[string]any
		if e := decode(w, r, &patch); e != nil {
			bad(w, e)
			return
		}
		out, e := h.service.UpdateOpening(ctx, id, patch)
		if e != nil {
			bad(w, e)
			return
		}
		write(w, 200, map[string]any{"opening": out})
		return
	}
	if strings.HasPrefix(path, "resumes/") && strings.HasSuffix(path, "/review") && r.Method == http.MethodPost {
		id := strings.TrimSuffix(strings.TrimPrefix(path, "resumes/"), "/review")
		out, e := h.service.ReviewResume(ctx, id)
		if e != nil {
			bad(w, e)
			return
		}
		write(w, http.StatusOK, map[string]any{"resume": out, "review": out.ReviewFindings})
		return
	}
	write(w, http.StatusNotFound, map[string]string{"error": "route not found"})
}
