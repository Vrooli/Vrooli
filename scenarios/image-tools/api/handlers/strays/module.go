// Package strays exposes model-storage containment to operators: the sidecar
// environment pins and disk floor in force, the scan for model bytes outside
// image-tools' receipts, and the adopt/reclaim actions on what it finds.
package strays

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"image-tools/internal/diskpolicy"
	"image-tools/internal/httpx"
	"image-tools/internal/modelenv"
	"image-tools/internal/modelstray"
	"image-tools/internal/module"

	"github.com/gorilla/mux"
)

// Service is the scanner the routes drive (satisfied by *modelstray.Scanner).
type Service interface {
	Scan(ctx context.Context) (modelstray.Report, error)
	Adopt(ctx context.Context, id string) (modelstray.Receipt, error)
	Reclaim(ctx context.Context, id string) (modelstray.Receipt, error)
}

// ContainmentReport is what `image-tools models containment` prints.
type ContainmentReport struct {
	Env  modelenv.Report   `json:"env"`
	Disk diskpolicy.Status `json:"disk"`
}

type actionRequest struct {
	ID string `json:"id"`
}

// Module mounts the containment routes.
func Module(svc Service, logger *log.Logger) module.Module {
	return module.Module{
		Name: "strays",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/v1/models/containment", func(w http.ResponseWriter, req *http.Request) {
				disk, err := diskpolicy.Describe(modelenv.Root(), os.Getenv)
				if err != nil {
					httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "disk policy could not be read")
					return
				}
				writeJSON(w, ContainmentReport{Env: modelenv.Describe(), Disk: disk})
			}).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/models/strays", func(w http.ResponseWriter, req *http.Request) {
				report, err := svc.Scan(req.Context())
				if err != nil {
					logger.Printf("strays: scan: %v", err)
					httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "stray model scan failed")
					return
				}
				writeJSON(w, report)
			}).Methods(http.MethodGet)
			for _, action := range []struct {
				path string
				run  func(context.Context, string) (modelstray.Receipt, error)
			}{{"/api/v1/models/strays/adopt", svc.Adopt}, {"/api/v1/models/strays/reclaim", svc.Reclaim}} {
				run := action.run
				r.HandleFunc(action.path, func(w http.ResponseWriter, req *http.Request) {
					req.Body = http.MaxBytesReader(w, req.Body, 4<<10)
					var body actionRequest
					dec := json.NewDecoder(req.Body)
					dec.DisallowUnknownFields()
					var extra any
					if err := dec.Decode(&body); err != nil || strings.TrimSpace(body.ID) == "" || dec.Decode(&extra) != io.EOF {
						httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "provide {\"id\": \"<finding id from models strays>\"}")
						return
					}
					receipt, err := run(req.Context(), strings.TrimSpace(body.ID))
					if err != nil {
						logger.Printf("strays: %s %s: %v", req.URL.Path, body.ID, err)
						httpx.WriteError(w, http.StatusConflict, httpx.CodeInvalidRequest, err.Error())
						return
					}
					writeJSON(w, receipt)
				}).Methods(http.MethodPost)
			}
		},
		Endpoints: Endpoints,
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func descriptor(id, path, method, summary, description string, request, response *module.Schema, errs []module.ErrorDesc) module.EndpointDescriptor {
	return module.EndpointDescriptor{
		ID: id, Path: path, Method: method, Summary: summary, Description: description, Category: "models",
		Request: request, Response: response, Errors: errs,
		RESTException: &module.RESTException{
			Reason: module.RESTReasonOpsProbe,
			Note:   "Operator diagnostic for model-storage containment; the payload is the scan/receipt shape, not a Connect procedure.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request:  module.RESTPayload{Transport: "json", Conformance: "external_shape"},
				Response: module.RESTPayload{Transport: "json", Conformance: "external_shape"},
				Error:    module.RESTPayload{ProtoFullName: "vrooli.image_tools.v1.shared.ErrorEnvelope", Transport: "json", Conformance: "protojson"},
			},
		},
	}
}

// Endpoints describes the routes for the generated endpoint inventory.
var Endpoints = []module.EndpointDescriptor{
	descriptor("model_containment", "/api/v1/models/containment", http.MethodGet,
		"Report the sidecar environment pins and the disk floor",
		"Lists every cache/home variable pinned under the managed model root, the offline flags inference runs with, the floor value with its reasons and the room above it.",
		&module.Schema{Type: "object"},
		&module.Schema{Type: "object", Properties: map[string]string{"env": "object", "disk": "object"}},
		[]module.ErrorDesc{{Status: http.StatusInternalServerError, Code: httpx.CodeInternal, Description: "The disk policy could not be read"}}),
	descriptor("model_strays", "/api/v1/models/strays", http.MethodGet,
		"Scan for model bytes outside image-tools' receipts",
		"Scans library default cache locations and the managed root; changes nothing. Each finding carries bytes and one action: adopt, reclaim or report_only.",
		&module.Schema{Type: "object"},
		&module.Schema{Type: "object", Properties: map[string]string{"scanned": "array", "findings": "array", "totals": "object"}},
		[]module.ErrorDesc{{Status: http.StatusInternalServerError, Code: httpx.CodeInternal, Description: "The scan failed"}}),
	descriptor("model_strays_adopt", "/api/v1/models/strays/adopt", http.MethodPost,
		"Adopt a stray finding into managed storage",
		"Moves the finding into the managed root (or records its receipt in place) and appends an adoption receipt.",
		&module.Schema{Type: "object", Properties: map[string]string{"id": "string"}},
		&module.Schema{Type: "object", Properties: map[string]string{"action": "string", "path": "string", "destination": "string", "bytes": "integer"}},
		[]module.ErrorDesc{{Status: http.StatusBadRequest, Code: httpx.CodeInvalidRequest, Description: "No finding id"}, {Status: http.StatusConflict, Code: httpx.CodeInvalidRequest, Description: "The finding is gone or cannot be adopted"}}),
	descriptor("model_strays_reclaim", "/api/v1/models/strays/reclaim", http.MethodPost,
		"Reclaim a stray finding that is safe to delete",
		"Deletes a duplicate of an already-managed copy or untracked managed bytes old enough that no install can own them, and appends a reclaim receipt. Other software's files are never reclaimed.",
		&module.Schema{Type: "object", Properties: map[string]string{"id": "string"}},
		&module.Schema{Type: "object", Properties: map[string]string{"action": "string", "path": "string", "bytes": "integer"}},
		[]module.ErrorDesc{{Status: http.StatusBadRequest, Code: httpx.CodeInvalidRequest, Description: "No finding id"}, {Status: http.StatusConflict, Code: httpx.CodeInvalidRequest, Description: "The finding is gone or cannot be reclaimed"}}),
}
