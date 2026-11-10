package quality

import (
	"context"
	"net/http"
	"sort"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	qualityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/quality"
	qualityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/quality/quality_v1connect"
	"image-tools/internal/jobs"
	"image-tools/internal/module"
	"image-tools/internal/operations"
	"image-tools/internal/ops"
	internalquality "image-tools/internal/quality"
)

type Scheduler interface {
	Submit(context.Context, jobs.Spec) (jobs.Job, error)
}
type Deps struct {
	Store internalquality.BlobStore
	Jobs  Scheduler
}

func Module(store internalquality.BlobStore, scheduler Scheduler) module.Module {
	h := &Deps{Store: store, Jobs: scheduler}
	path, handler := qualityconnect.NewQualityServiceHandler(h)
	return module.Module{Name: "quality", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		r.HandleFunc("/api/v1/quality/evaluate", h.evaluate).Methods(http.MethodPost)
	}, Endpoints: Endpoints}
}
func (h *Deps) ListQualityProfiles(context.Context, *connect.Request[qualityv1.ListQualityProfilesRequest]) (*connect.Response[qualityv1.ListQualityProfilesResponse], error) {
	names := []string{}
	for _, op := range operations.All() {
		names = append(names, op.Name)
	}
	for _, op := range ops.List() {
		names = append(names, op.Name)
	}
	sort.Strings(names)
	return connect.NewResponse(&qualityv1.ListQualityProfilesResponse{EvaluatorVersion: internalquality.Version, Profiles: internalquality.Profiles(), Operations: names}), nil
}
