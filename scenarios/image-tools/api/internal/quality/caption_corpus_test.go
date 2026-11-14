package quality

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	qualityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/quality"
)

type captionCorpus struct {
	SchemaVersion int    `json:"schema_version"`
	DatasetID     string `json:"dataset_id"`
	Safety        string `json:"safety"`
	Images        []struct {
		ID         string `json:"id"`
		Input      string `json:"input"`
		SHA256     string `json:"sha256"`
		Difficulty string `json:"difficulty"`
		Caption    string `json:"caption"`
		Question   string `json:"question"`
		Answer     string `json:"answer"`
	} `json:"images"`
}

func TestCaptionReferenceCorpusIsFrozenAndResolvable(t *testing.T) {
	const manifest = "testdata/caption-v1/corpus.json"
	raw, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var corpus captionCorpus
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatal(err)
	}
	if corpus.SchemaVersion != 1 || corpus.DatasetID != "image-tools-caption-v1" {
		t.Fatalf("unexpected corpus identity: schema=%d dataset=%q", corpus.SchemaVersion, corpus.DatasetID)
	}
	if !strings.Contains(strings.ToLower(corpus.Safety), "no client uploads") || !strings.Contains(strings.ToLower(corpus.Safety), "real-person likenesses") {
		t.Fatalf("corpus safety boundary is missing: %q", corpus.Safety)
	}
	if len(corpus.Images) != 5 {
		t.Fatalf("corpus has %d images; want five frozen cases", len(corpus.Images))
	}
	seen := map[string]bool{}
	difficulties := map[string]bool{}
	for _, c := range corpus.Images {
		if c.ID == "" || seen[c.ID] {
			t.Fatalf("empty or duplicate case id %q", c.ID)
		}
		seen[c.ID] = true
		if c.Caption == "" || c.Question == "" || c.Answer == "" {
			t.Fatalf("case %q is missing a frozen reference", c.ID)
		}
		difficulties[c.Difficulty] = true
		input := filepath.Clean(filepath.Join(filepath.Dir(manifest), c.Input))
		data, err := os.ReadFile(input)
		if err != nil {
			t.Fatalf("case %q input %s: %v", c.ID, input, err)
		}
		sum := sha256.Sum256(data)
		if got := hex.EncodeToString(sum[:]); got != c.SHA256 {
			t.Fatalf("case %q input hash changed: got %s want %s", c.ID, got, c.SHA256)
		}
	}
	for _, band := range []string{"easy", "medium", "hard"} {
		if !difficulties[band] {
			t.Fatalf("corpus omits difficulty band %q", band)
		}
	}
}

func TestCaptionCorpusBuildsSeparatePromotionManifests(t *testing.T) {
	data, err := os.ReadFile("testdata/caption-v1/corpus.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus captionCorpus
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}

	for _, task := range []string{"caption", "visual-question-answering"} {
		t.Run(task, func(t *testing.T) {
			manifest := &qualityv1.EvaluationManifest{
				SchemaVersion: 1, Name: "caption-v1-" + task, Operation: "caption", Profile: "text",
				DatasetId: corpus.DatasetID, CandidateModel: "candidate-under-test", CandidateRevision: "sha256:record-after-capture",
				Judge: "required", JudgeModel: "qwen3-vl:4b",
				Budgets: []*qualityv1.MetricBudget{
					{Metric: "text.similarity", Scope: "case", Minimum: float64p(.35)},
					{Metric: "judge.fidelity", Scope: "case", Minimum: float64p(.70)},
					{Metric: "judge.instruction", Scope: "case", Minimum: float64p(.75)},
					{Metric: "judge.artifacts", Scope: "case", Minimum: float64p(.80)},
					{Metric: "judge.fidelity", Scope: "mean", Minimum: float64p(.80)},
					{Metric: "judge.fidelity", Scope: "mean", MinImprovement: float64p(.01)},
					{Metric: "judge.instruction", Scope: "mean", Minimum: float64p(.80)},
					{Metric: "judge.artifacts", Scope: "mean", Minimum: float64p(.80)},
				},
			}
			for _, c := range corpus.Images {
				reference := c.Caption
				instruction := "Describe the visible image accurately."
				if task == "visual-question-answering" {
					reference = c.Answer
					instruction = c.Question
				}
				manifest.Cases = append(manifest.Cases, &qualityv1.EvaluationCase{
					Id: c.ID, Input: filepath.Clean(filepath.Join("testdata/caption-v1", c.Input)), InputSha256: c.SHA256,
					ReferenceText: reference, Instruction: instruction, Cohort: c.Difficulty,
				})
			}
			if err := Validate(manifest); err != nil {
				t.Fatalf("frozen corpus does not produce a valid promotion manifest: %v", err)
			}
			if len(manifest.Cases) != 5 || manifest.Judge != "required" || manifest.JudgeModel != "qwen3-vl:4b" {
				t.Fatalf("incomplete promotion manifest: %#v", manifest)
			}
		})
	}
}

func float64p(v float64) *float64 { return &v }
