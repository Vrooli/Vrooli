package library

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"device-studio/internal/httpx"
	domain "device-studio/internal/library"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
)

type intakePreviewOperations interface {
	Preview(context.Context, string, uint32, string) (domain.IntakePreview, error)
}

func newIntakePreviewHandler(service intakePreviewOperations, db *database.RoutedDB, roots *filerouting.RoutedRoots, logger *log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, httpx.CodeInternal, "geometry intake preview is unavailable")
			return
		}
		ctx, err := pinIntakeRoute(r.Context(), db, roots)
		if err != nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, httpx.CodeInternal, "routed intake storage is unavailable")
			return
		}
		revision, err := parseIntakeRevision(mux.Vars(r)["revision"])
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "candidate revision is invalid")
			return
		}
		digest := r.URL.Query().Get("revision_sha256")
		preview, err := service.Preview(ctx, mux.Vars(r)["candidate_id"], revision, digest)
		if err != nil {
			writeIntakeHTTPError(w, logger, err)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", strconv.Itoa(len(preview.PNG)))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("X-Intake-Candidate-ID", preview.CandidateID)
		w.Header().Set("X-Intake-Revision", strconv.FormatUint(uint64(preview.Revision), 10))
		w.Header().Set("X-Intake-Revision-SHA256", preview.RevisionSHA256)
		w.Header().Set("X-Intake-GLB-SHA256", preview.GLBSHA256)
		w.Header().Set("X-Intake-Preview-SHA256", preview.PreviewSHA256)
		w.Header().Set("X-Intake-Preview-Vertex-Colors", strconv.FormatBool(preview.VertexColors))
		w.Header().Set("X-Intake-Preview-Scale", "not-estimated")
		w.Header().Set("X-Intake-Preview-Review", "unreviewed")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(preview.PNG); err != nil && logger != nil {
			logger.Printf("library intake preview response interrupted: %v", err)
		}
	})
}
