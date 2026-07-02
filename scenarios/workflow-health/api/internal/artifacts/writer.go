package artifacts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/encoding/protojson"
)

type RunArtifact struct {
	Dir      string `json:"dir"`
	Workflow string `json:"workflow"`
	Timeline string `json:"timeline,omitempty"`
	Latest   string `json:"latest"`
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
	dir := filepath.Join(w.scenarioDir, "coverage", "workflow-health", "runs", sanitize(runID), sanitize(assetID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return RunArtifact{}, fmt.Errorf("create artifact dir: %w", err)
	}
	out := RunArtifact{
		Dir:      rel(w.scenarioDir, dir),
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
		out.Timeline = rel(w.scenarioDir, path)
	}
	path := filepath.Join(dir, "latest.json")
	data, err := json.MarshalIndent(latest, "", "  ")
	if err != nil {
		return out, fmt.Errorf("marshal latest: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return out, fmt.Errorf("write latest: %w", err)
	}
	out.Latest = rel(w.scenarioDir, path)
	return out, nil
}

func Counts(timeline *bastimeline.ExecutionTimeline) TimelineCounts {
	if timeline == nil {
		return TimelineCounts{}
	}
	return TimelineCounts{Entries: len(timeline.GetEntries()), Logs: len(timeline.GetLogs())}
}

func rel(root, path string) string {
	r, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(r)
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
