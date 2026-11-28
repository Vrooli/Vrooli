package creation

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "creation"

type job struct {
	JobID        string `json:"job_id"`
	ProjectID    string `json:"project_id"`
	CandidateID  string `json:"candidate_id"`
	State        string `json:"state"`
	Version      int64  `json:"version"`
	ReviewStatus string `json:"review_status"`
	Message      string `json:"message,omitempty"`
}

type handlers struct{ core *cliapp.ScenarioApp }

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := handlers{core: core}
	g, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"start":  cliapp.Action(h.start, h.report),
		"get":    cliapp.Action(h.get, h.report),
		"cancel": cliapp.Action(h.cancel, h.report),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("creation: load CLI manifest: %w", err)
	}
	return g, nil
}

func (h handlers) start(ctx cliapp.OperationContext) (job, error) {
	if !ctx.BoolFlag("confirm-local-ai") {
		return job{}, errors.New("confirm --confirm-local-ai after reviewing the local-only disclosure")
	}
	var dimensions [3]float64
	for i, name := range []string{"width-mm", "height-mm", "depth-mm"} {
		value, err := strconv.ParseFloat(strings.TrimSpace(ctx.Flag(name)), 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 || value > 1000 {
			return job{}, fmt.Errorf("%s must be a finite value in (0, 1000]", name)
		}
		dimensions[i] = value
	}
	projectID := strings.TrimSpace(ctx.Flag("project-id"))
	if projectID == "" {
		return job{}, errors.New("project-id is required")
	}
	brief := strings.TrimSpace(ctx.Flag("brief"))
	refs := splitReferences(ctx.Flag("reference-media-ids"))
	if brief == "" || len(brief) > 4000 || len(refs) < 1 || len(refs) > 4 {
		return job{}, errors.New("provide a brief and one to four comma-separated project image IDs")
	}
	candidateID, err := randomID("creation")
	if err != nil {
		return job{}, err
	}
	key, err := randomID("request")
	if err != nil {
		return job{}, err
	}
	body, err := json.Marshal(map[string]any{"candidate_id": candidateID, "idempotency_key": key, "brief": brief, "envelope_mm": dimensions, "reference_media_ids": refs, "confirm_local_ai": true})
	if err != nil {
		return job{}, err
	}
	path := "/projects/" + urlPathSegment(projectID) + "/geometry-creations"
	return h.request(http.MethodPost, path, body)
}

func (h handlers) get(ctx cliapp.OperationContext) (job, error) {
	return h.request(http.MethodGet, "/geometry-creations/"+urlPathSegment(ctx.Positional("job-id")), nil)
}

func (h handlers) cancel(ctx cliapp.OperationContext) (job, error) {
	version, err := strconv.ParseInt(strings.TrimSpace(ctx.Flag("expected-version")), 10, 64)
	if err != nil || version <= 0 {
		return job{}, errors.New("expected-version must be a positive job version")
	}
	body, _ := json.Marshal(map[string]int64{"version": version})
	return h.request(http.MethodPost, "/geometry-creations/"+urlPathSegment(ctx.Positional("job-id"))+"/cancel", body)
}

func (h handlers) request(method, path string, body []byte) (job, error) {
	data, err := h.core.Request(method, path, nil, body)
	if err != nil {
		return job{}, cliapp.WrapAPIError("local geometry creation", err, nil)
	}
	var result job
	if err := json.Unmarshal(data, &result); err != nil {
		return job{}, fmt.Errorf("decode geometry job status: %w", err)
	}
	if result.JobID == "" || result.CandidateID == "" || result.ReviewStatus != "unreviewed" || result.Version <= 0 {
		return job{}, errors.New("server returned incomplete or promoted geometry job status")
	}
	return result, nil
}

func (handlers) report(_ cliapp.OperationContext, result job) cliapp.MutationReport {
	lines := []string{fmt.Sprintf("Geometry job %s is %s (version %d); candidate %s remains %s.", result.JobID, result.State, result.Version, result.CandidateID, result.ReviewStatus)}
	if result.Message != "" {
		lines = append(lines, result.Message)
	}
	next := []string{fmt.Sprintf("`creation get %s` — read durable job status", result.JobID)}
	if result.State == "queued" || result.State == "running" || result.State == "checkpointed" {
		next = append(next, fmt.Sprintf("`creation cancel %s --expected-version %d` — request cancellation", result.JobID, result.Version))
	}
	if result.State == "succeeded" {
		next = append(next, fmt.Sprintf("`library intake-get %s --revision 1` — inspect the unreviewed result", result.CandidateID))
	}
	return cliapp.MutationReport{Result: lines, NextCommand: next}
}

func splitReferences(raw string) []string {
	seen := map[string]bool{}
	var refs []string
	for _, part := range strings.Split(raw, ",") {
		id := strings.TrimSpace(part)
		if id != "" && !seen[id] {
			refs = append(refs, id)
			seen[id] = true
		}
	}
	return refs
}

func randomID(prefix string) (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("create request ID: %w", err)
	}
	return prefix + "-" + hex.EncodeToString(bytes[:]), nil
}

func urlPathSegment(value string) string { return url.PathEscape(strings.TrimSpace(value)) }
