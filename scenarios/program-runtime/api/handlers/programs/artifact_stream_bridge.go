package programs

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	"program-runtime/internal/artifactfile"
	"program-runtime/internal/bindings"
	"program-runtime/internal/sessions"
)

const runEvidenceProgramName = "test-genie.run-evidence"

type ArtifactStreamBridge struct {
	Sessions sessionGetter
	Programs programIdentityResolver
	Bindings artifactStreamOpener
}

type sessionGetter interface {
	Get(context.Context, string) (*sessions.Session, error)
}

type programIdentityResolver interface {
	ExecutionIdentity(context.Context, string) (string, string, string, error)
	Get(context.Context, string) (*programsv1.Program, error)
}

type artifactStreamOpener interface {
	OpenTestGenieArtifactStream(context.Context, string, string, string, string, string, []string, bindings.InvocationMetadata, *http.Client) (bindings.TestGenieArtifact, io.ReadCloser, error)
}

type artifactStreamRequest struct {
	SessionID  string `json:"session_id"`
	ProgramID  string `json:"program_id"`
	Scenario   string `json:"scenario"`
	RunID      string `json:"run_id"`
	ArtifactID string `json:"artifact_id"`
	Path       string `json:"path"`
	Overwrite  bool   `json:"overwrite"`
}

type artifactStreamReceipt struct {
	Scenario         string `json:"scenario"`
	RunID            string `json:"run_id"`
	ArtifactID       string `json:"artifact_id"`
	Kind             string `json:"kind"`
	OutputPath       string `json:"output_path"`
	Bytes            int64  `json:"bytes"`
	SHA256           string `json:"sha256"`
	LegacyDiscovered bool   `json:"legacy_discovered"`
}

func (b *ArtifactStreamBridge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		artifactBridgeError(w, http.StatusMethodNotAllowed, "POST is required")
		return
	}
	if b == nil || b.Sessions == nil || b.Programs == nil || b.Bindings == nil {
		artifactBridgeError(w, http.StatusServiceUnavailable, "artifact stream effect is unavailable")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request artifactStreamRequest
	if err := decoder.Decode(&request); err != nil {
		artifactBridgeError(w, http.StatusBadRequest, "invalid artifact stream request")
		return
	}
	if strings.TrimSpace(request.SessionID) == "" || strings.TrimSpace(request.ProgramID) == "" {
		artifactBridgeError(w, http.StatusBadRequest, "session_id and program_id are required")
		return
	}
	programName, digest, identitySession, err := b.Programs.ExecutionIdentity(r.Context(), request.ProgramID)
	if err != nil || identitySession != request.SessionID || programName != runEvidenceProgramName {
		artifactBridgeError(w, http.StatusForbidden, "only the pinned Test Genie run-evidence program may stream artifacts")
		return
	}
	program, err := b.Programs.Get(r.Context(), request.ProgramID)
	if err != nil || program.GetStatus() != programsv1.ProgramStatus_PROGRAM_STATUS_RUNNING {
		artifactBridgeError(w, http.StatusConflict, "artifact stream effect requires an active program execution")
		return
	}
	session, err := b.Sessions.Get(r.Context(), request.SessionID)
	if err != nil || session.State != "running" || strings.TrimSpace(session.SandboxWorkspace) == "" {
		artifactBridgeError(w, http.StatusForbidden, "artifact transfer requires a running session with an admitted workspace")
		return
	}
	metadata := bindings.InvocationMetadata{SessionID: session.ID, ProgramID: request.ProgramID, Provenance: program.GetProvenance().String()}
	ref, stream, err := b.Bindings.OpenTestGenieArtifactStream(r.Context(), programName, digest, request.Scenario, request.RunID, request.ArtifactID, grantNames(session.Grants), metadata, &http.Client{Timeout: 2 * time.Minute})
	if err != nil {
		artifactBridgeError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer stream.Close()
	result, err := artifactfile.Publish(session.SandboxWorkspace, request.Path, stream, ref.SizeBytes, artifactfile.DefaultMaxBytes, request.Overwrite)
	if err != nil {
		artifactBridgeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeArtifactBridgeJSON(w, artifactStreamReceipt{Scenario: ref.Target, RunID: ref.RunID, ArtifactID: ref.ArtifactID, Kind: ref.Kind, OutputPath: result.Path, Bytes: result.Bytes, SHA256: result.SHA256, LegacyDiscovered: ref.Legacy})
}

func grantNames(grants map[string]struct{}) []string {
	out := make([]string, 0, len(grants))
	for grant := range grants {
		out = append(out, grant)
	}
	return out
}

func artifactBridgeError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": detail})
}

func writeArtifactBridgeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		// The transfer is already committed. A response encoding failure is not
		// permission to retry or remove the successful artifact.
		return
	}
}
