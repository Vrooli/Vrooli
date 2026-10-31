package artifacts

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	corestorage "github.com/vrooli/api-core/storage"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/encoding/protojson"
)

type RunArtifact struct {
	Dir        string      `json:"dir"`
	Workflow   string      `json:"workflow"`
	Timeline   string      `json:"timeline,omitempty"`
	Latest     string      `json:"latest"`
	References []Reference `json:"references,omitempty"`
}

// Reference is the provider-neutral metadata for one durable artifact. The
// latest summary is a redacted derivative: it contains no workflow payload,
// request data, or credentials, only execution identity and outcome fields.
// Consumers can safely link it into a combined evidence manifest without
// depending on Workflow Health's internal artifact layout.
type Reference struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	URI       string `json:"uri"`
	MediaType string `json:"media_type,omitempty"`
	Checksum  string `json:"checksum"`
	Redacted  bool   `json:"redacted"`
}

type WorkflowLatest struct {
	RunID       string         `json:"run_id"`
	AssetID     string         `json:"asset_id"`
	AssetPath   string         `json:"asset_path"`
	ExecutionID string         `json:"execution_id,omitempty"`
	Status      string         `json:"status"`
	Success     bool           `json:"success"`
	Error       string         `json:"error,omitempty"`
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt time.Time      `json:"completed_at"`
	DurationMs  int64          `json:"duration_ms"`
	Summary     TimelineCounts `json:"summary"`
}

type TimelineCounts struct {
	Entries int `json:"entries"`
	Logs    int `json:"logs"`
}

type Writer struct {
	scenarioDir string
	runID       string
}

func NewWriter(scenarioDir, runID string) *Writer {
	return &Writer{scenarioDir: filepath.Clean(scenarioDir), runID: strings.TrimSpace(runID)}
}

func (w *Writer) WriteWorkflow(assetID, assetPath string, timeline *bastimeline.ExecutionTimeline, latest WorkflowLatest) (RunArtifact, error) {
	if w == nil {
		return RunArtifact{}, fmt.Errorf("artifact writer is nil")
	}
	runID := w.runID
	if runID == "" {
		runID = "manual"
	}
	resolver, err := corestorage.NewResolver(corestorage.ResolverConfig{AppID: "vrooli", Profile: corestorage.ProfileAuto})
	if err != nil {
		return RunArtifact{}, fmt.Errorf("create artifact resolver: %w", err)
	}
	target := sanitize(filepath.Base(w.scenarioDir))
	dir, err := resolver.EnsureArtifactDir(corestorage.Options{ScenarioID: "workflow-health"}, corestorage.ArtifactRef{
		Owner: "workflow-health", Domain: "run-artifacts", Class: corestorage.ClassState,
		Segments: []string{target, "runs", sanitize(runID), sanitize(assetID)},
	}, 0o755)
	if err != nil {
		return RunArtifact{}, fmt.Errorf("create artifact dir: %w", err)
	}
	out := RunArtifact{
		Dir:      filepath.ToSlash(dir),
		Workflow: filepath.ToSlash(assetPath),
	}
	if timeline != nil {
		data, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: false}.Marshal(timeline)
		if err != nil {
			return out, fmt.Errorf("marshal timeline: %w", err)
		}
		path := filepath.Join(dir, "timeline.json")
		if err := os.WriteFile(path, prettyJSON(data), 0o644); err != nil {
			return out, fmt.Errorf("write timeline: %w", err)
		}
		out.Timeline = filepath.ToSlash(path)
	}
	path := filepath.Join(dir, "latest.json")
	data, err := json.MarshalIndent(latest, "", "  ")
	if err != nil {
		return out, fmt.Errorf("marshal latest: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return out, fmt.Errorf("write latest: %w", err)
	}
	out.Latest = filepath.ToSlash(path)
	checksum, err := checksumFile(path)
	if err != nil {
		return out, fmt.Errorf("checksum latest: %w", err)
	}
	out.References = []Reference{{
		ID:        assetID + ":latest",
		Kind:      "workflow-summary",
		URI:       out.Latest,
		MediaType: "application/json",
		Checksum:  "sha256:" + checksum,
		Redacted:  true,
	}}
	return out, nil
}

func checksumFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func Counts(timeline *bastimeline.ExecutionTimeline) TimelineCounts {
	if timeline == nil {
		return TimelineCounts{}
	}
	return TimelineCounts{Entries: len(timeline.GetEntries()), Logs: len(timeline.GetLogs())}
}

func sanitize(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func prettyJSON(data []byte) []byte {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return append(data, '\n')
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return append(data, '\n')
	}
	return append(out, '\n')
}
