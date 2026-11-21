package quality

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	qualityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/quality"
	"google.golang.org/protobuf/encoding/protojson"
	"image-tools/internal/httpx"
	"image-tools/internal/jobs"
	internalquality "image-tools/internal/quality"
)

func (h *Deps) evaluate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 256<<20)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		httpx.WriteError(w, 400, httpx.CodeInvalidRequest, "invalid evaluation multipart body: "+err.Error())
		return
	}
	defer r.MultipartForm.RemoveAll()
	raw := r.FormValue("manifest")
	if len(raw) > 2<<20 {
		httpx.WriteError(w, 413, httpx.CodeInvalidRequest, "manifest exceeds 2 MiB")
		return
	}
	m := &qualityv1.EvaluationManifest{}
	if err := protojson.Unmarshal([]byte(raw), m); err != nil {
		httpx.WriteError(w, 400, httpx.CodeInvalidRequest, "manifest: "+err.Error())
		return
	}
	if err := internalquality.Validate(m); err != nil {
		httpx.WriteError(w, 400, httpx.CodeInvalidRequest, err.Error())
		return
	}
	baseline := r.FormValue("baseline")
	if len(baseline) > 4<<20 {
		httpx.WriteError(w, 413, httpx.CodeInvalidRequest, "baseline exceeds 4 MiB")
		return
	}
	if baseline != "" {
		b := &qualityv1.EvaluationReport{}
		if err := protojson.Unmarshal([]byte(baseline), b); err != nil {
			httpx.WriteError(w, 400, httpx.CodeInvalidRequest, "baseline: "+err.Error())
			return
		}
		if err := internalquality.VerifyReport(b); err != nil {
			httpx.WriteError(w, 400, httpx.CodeInvalidRequest, err.Error())
			return
		}
	}
	// Resolve only multipart parts, never a caller-supplied server path. Inputs
	// are retained by the blob owner before the durable task is admitted.
	prefix := "inputs/quality/" + uuid.NewString() + "/"
	refs := map[string]string{}
	for _, c := range m.Cases {
		for _, field := range []*string{&c.Input, &c.Reference, &c.Output} {
			if *field == "" {
				continue
			}
			key := *field
			if ref, ok := refs[key]; ok {
				*field = ref
				continue
			}
			f, _, err := r.FormFile(key)
			if err != nil {
				httpx.WriteError(w, 400, httpx.CodeInvalidRequest, fmt.Sprintf("missing image part %q", key))
				return
			}
			data, err := io.ReadAll(io.LimitReader(f, (32<<20)+1))
			_ = f.Close()
			if err != nil || len(data) == 0 || len(data) > 32<<20 {
				httpx.WriteError(w, 413, httpx.CodeInvalidRequest, "invalid or oversized image part")
				return
			}
			ref := fmt.Sprintf("%s%d", prefix, len(refs))
			if err := h.Store.Put(r.Context(), ref, bytes.NewReader(data), "application/octet-stream"); err != nil {
				httpx.WriteError(w, 500, httpx.CodeInternal, "retain quality input: "+err.Error())
				return
			}
			refs[key] = ref
			*field = ref
		}
	}
	encoded, err := protojson.Marshal(m)
	if err != nil {
		httpx.WriteError(w, 500, httpx.CodeInternal, err.Error())
		return
	}
	task := internalquality.Task{Manifest: encoded}
	if strings.TrimSpace(baseline) != "" {
		task.Baseline = []byte(baseline)
	}
	payload, err := json.Marshal(task)
	if err != nil {
		httpx.WriteError(w, 500, httpx.CodeInternal, err.Error())
		return
	}
	job, err := h.Jobs.Submit(r.Context(), jobs.Spec{Operation: internalquality.Operation, Lane: jobs.LaneCPU, Priority: jobs.PriorityBatch, Payload: payload})
	if err != nil {
		httpx.WriteError(w, 503, httpx.CodeInternal, err.Error())
		return
	}
	httpx.WriteProto(w, http.StatusAccepted, &qualityv1.SubmitEvaluationResponse{JobId: job.ID})
}
