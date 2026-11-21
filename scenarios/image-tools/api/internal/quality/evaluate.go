package quality

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	qualityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/quality"
	"google.golang.org/protobuf/proto"
)

const MaxCases = 64

type (
	Loader func(context.Context, string) ([]byte, error)
	Judge  interface {
		Score(context.Context, JudgeRequest) (*qualityv1.JudgeResult, error)
	}
)
type JudgeRequest struct {
	Profile, Rubric, Instruction, ReferenceText, OutputText, ExpectedModel string
	Input, Reference, Output                                               []byte
}
type Evaluator struct{ Judge Judge }

func Validate(m *qualityv1.EvaluationManifest) error {
	if m == nil || m.SchemaVersion != 1 {
		return fmt.Errorf("quality manifest schema_version must be 1")
	}
	if strings.TrimSpace(m.Operation) == "" || strings.TrimSpace(m.DatasetId) == "" || strings.TrimSpace(m.CandidateModel) == "" || strings.TrimSpace(m.CandidateRevision) == "" {
		return fmt.Errorf("operation, dataset_id, candidate_model and candidate_revision are required")
	}
	p := profile(m.Profile)
	if p == nil {
		return fmt.Errorf("unknown quality profile %q", m.Profile)
	}
	if len(m.Cases) == 0 || len(m.Cases) > MaxCases {
		return fmt.Errorf("quality manifest requires 1..%d cases", MaxCases)
	}
	if m.Judge != "" && m.Judge != "required" && m.Judge != "disabled" {
		return fmt.Errorf("judge must be required or disabled")
	}
	if m.Judge != "disabled" && m.JudgeModel == "" {
		return fmt.Errorf("judge_model is required to bind the local judge identity")
	}
	seen := map[string]bool{}
	for _, c := range m.Cases {
		if c == nil || strings.TrimSpace(c.Id) == "" || seen[c.Id] {
			return fmt.Errorf("cases require unique nonempty ids")
		}
		seen[c.Id] = true
		if m.Profile == "text" || m.Profile == "json" {
			if c.Reference != "" || c.Output != "" {
				return fmt.Errorf("case %s: text/JSON profiles use reference_text and output_text; put the source image in input", c.Id)
			}
			if c.ReferenceText == "" {
				return fmt.Errorf("case %s: reference_text is required", c.Id)
			}
			if len(c.ReferenceText) > 100_000 || len(c.OutputText) > 100_000 {
				return fmt.Errorf("case %s: text exceeds 100000 bytes", c.Id)
			}
		} else if c.Output == "" {
			return fmt.Errorf("case %s: output image is required", c.Id)
		} else if c.Reference == "" {
			if c.ReferenceSha256 != "" {
				return fmt.Errorf("case %s: reference hash provided without a reference image", c.Id)
			}
			if m.Judge == "disabled" {
				return fmt.Errorf("case %s: image output without a pixel reference requires the local judge", c.Id)
			}
			if m.Operation != "text_to_image" && c.Input == "" {
				return fmt.Errorf("case %s: source image is required when scoring %s without a pixel reference", c.Id, m.Operation)
			}
		}
		if len(c.Instruction) > 16000 {
			return fmt.Errorf("case %s: instruction exceeds 16000 bytes", c.Id)
		}
	}
	if len(m.Budgets) == 0 {
		return fmt.Errorf("at least one explicit quality budget is required")
	}
	available := map[string]bool{}
	for _, metric := range p.Metrics {
		available[metric] = true
	}
	if m.Judge != "disabled" {
		for _, metric := range []string{"judge.fidelity", "judge.instruction", "judge.artifacts"} {
			available[metric] = true
		}
	}
	for _, b := range m.Budgets {
		if b == nil || !available[b.Metric] {
			return fmt.Errorf("budget metric is unavailable for profile %s and judge mode", m.Profile)
		}
		if b.Scope != "" && b.Scope != "case" && b.Scope != "mean" {
			return fmt.Errorf("budget scope must be case or mean")
		}
		if b.Minimum == nil && b.Maximum == nil && b.MaxRegression == nil && b.MinImprovement == nil {
			return fmt.Errorf("budget %s requires minimum, maximum, max_regression or min_improvement", b.Metric)
		}
		for _, v := range []*float64{b.Minimum, b.Maximum, b.MaxRegression, b.MinImprovement} {
			if v != nil && (math.IsNaN(*v) || math.IsInf(*v, 0)) {
				return fmt.Errorf("budget %s must be finite", b.Metric)
			}
		}
		if b.Minimum != nil && b.Maximum != nil && *b.Minimum > *b.Maximum {
			return fmt.Errorf("budget %s minimum exceeds maximum", b.Metric)
		}
		if b.MaxRegression != nil && *b.MaxRegression < 0 {
			return fmt.Errorf("budget %s max_regression must be nonnegative", b.Metric)
		}
		if b.MinImprovement != nil && *b.MinImprovement <= 0 {
			return fmt.Errorf("budget %s min_improvement must be positive", b.Metric)
		}
	}
	return nil
}

// Evaluate always retains every case. Failed decoding, missing evidence and
// unavailable judges make the report incomplete and non-passing; they do not
// disappear from the denominator. Baselines are verified before comparison.
func (e *Evaluator) Evaluate(ctx context.Context, m *qualityv1.EvaluationManifest, load Loader, baseline *qualityv1.EvaluationReport, emit func(int, string)) (*qualityv1.EvaluationReport, error) {
	if err := Validate(m); err != nil {
		return nil, err
	}
	if baseline != nil {
		if err := VerifyReport(baseline); err != nil {
			return nil, fmt.Errorf("baseline: %w", err)
		}
	}
	start := time.Now()
	r := &qualityv1.EvaluationReport{SchemaVersion: 1, EvaluatorVersion: Version, Name: m.Name, Operation: m.Operation, Profile: m.Profile, DatasetId: m.DatasetId, CandidateModel: m.CandidateModel, CandidateRevision: m.CandidateRevision, StartedAt: start.UTC().Format(time.RFC3339Nano), Complete: true, Means: map[string]float64{}, Budgets: m.Budgets, JudgeModel: m.JudgeModel}
	r.PolicySha256 = hashMessage(&qualityv1.EvaluationManifest{Budgets: m.Budgets})
	rubric := profile(m.Profile).Rubric
	if m.Rubric != "" {
		rubric += "\nAdditional task rubric: " + m.Rubric
	}
	suite := &qualityv1.EvaluationManifest{SchemaVersion: 1, Operation: m.Operation, Profile: m.Profile, DatasetId: m.DatasetId, Judge: m.Judge, JudgeModel: m.JudgeModel, Rubric: Version + "\n" + rubric}
	if suite.Judge == "" {
		suite.Judge = "required"
	}
	for i, c := range m.Cases {
		if ctx.Err() != nil {
			return nil, context.Cause(ctx)
		}
		caseStart := time.Now()
		s := &qualityv1.CaseScore{Id: c.Id, Cohort: c.Cohort, Metrics: map[string]float64{}}
		input := readAsset(ctx, load, c.Input, c.InputSha256, "input", s)
		var ref, out []byte
		if m.Profile != "text" && m.Profile != "json" {
			if c.Reference != "" {
				ref = readAsset(ctx, load, c.Reference, c.ReferenceSha256, "reference", s)
			}
			out = readAsset(ctx, load, c.Output, c.OutputSha256, "output", s)
		}
		s.InputSha256, s.ReferenceSha256, s.OutputSha256 = hashOptional(input), hashOptional(ref), hashOptional(out)
		if m.Profile == "text" || m.Profile == "json" {
			s.ReferenceSha256 = hash([]byte(c.ReferenceText))
			s.OutputSha256 = hash([]byte(c.OutputText))
			if c.ReferenceSha256 != "" && !strings.EqualFold(c.ReferenceSha256, s.ReferenceSha256) {
				s.Errors = append(s.Errors, "reference text SHA-256 mismatch")
			}
			if c.OutputSha256 != "" && !strings.EqualFold(c.OutputSha256, s.OutputSha256) {
				s.Errors = append(s.Errors, "output text SHA-256 mismatch")
			}
		}
		identity := proto.Clone(c).(*qualityv1.EvaluationCase)
		identity.Input, identity.Reference, identity.Output = "", "", ""
		identity.InputSha256, identity.ReferenceSha256, identity.OutputSha256 = s.InputSha256, s.ReferenceSha256, ""
		identity.OutputText = ""
		s.CaseSha256 = hashMessage(identity)
		suite.Cases = append(suite.Cases, identity)
		if len(s.Errors) == 0 {
			var err error
			if m.Profile == "text" || m.Profile == "json" {
				s.Metrics, err = textMetrics(m.Profile, c.ReferenceText, c.OutputText)
			} else if len(ref) == 0 {
				if m.Operation == "text_to_image" {
					s.Warnings = append(s.Warnings, "reference-image metrics omitted: text-to-image has no canonical pixel reference")
				} else {
					s.Warnings = append(s.Warnings, "reference-image metrics omitted: no canonical pixel reference; the local judge uses the supplied source image and instruction")
				}
			} else {
				s.Metrics, s.Warnings, err = imageMetrics(m.Profile, ref, out)
			}
			if err != nil {
				s.Errors = append(s.Errors, err.Error())
			}
		}
		if m.Judge != "disabled" && len(s.Errors) == 0 {
			if e.Judge == nil {
				s.Errors = append(s.Errors, "required local judge is unavailable")
			} else {
				j, err := e.Judge.Score(ctx, JudgeRequest{Profile: m.Profile, Rubric: rubric, Instruction: c.Instruction, ReferenceText: c.ReferenceText, OutputText: c.OutputText, ExpectedModel: m.JudgeModel, Input: input, Reference: ref, Output: out})
				if err != nil {
					s.Errors = append(s.Errors, "judge: "+err.Error())
				} else if err = validateJudge(j, m.JudgeModel); err != nil {
					s.Errors = append(s.Errors, err.Error())
				} else {
					s.Judge = j
					s.Metrics["judge.fidelity"], s.Metrics["judge.instruction"], s.Metrics["judge.artifacts"] = j.Fidelity, j.Instruction, j.Artifacts
				}
			}
		}
		if len(s.Errors) > 0 {
			r.Complete = false
			for _, err := range s.Errors {
				r.Violations = append(r.Violations, &qualityv1.GateViolation{CaseId: c.Id, Reason: err})
			}
		}
		s.DurationMs = time.Since(caseStart).Milliseconds()
		r.Cases = append(r.Cases, s)
		if emit != nil {
			emit((i+1)*90/len(m.Cases), "scored "+c.Id)
		}
	}
	// Ordering is presentation only. Baselines pair by case identity, never by
	// array position; sorting prevents a shuffled file from changing a suite.
	sort.Slice(suite.Cases, func(i, j int) bool { return suite.Cases[i].Id < suite.Cases[j].Id })
	r.SuiteSha256 = hashMessage(suite)
	counts := map[string]int{}
	for _, s := range r.Cases {
		for k, v := range s.Metrics {
			r.Means[k] += v
			counts[k]++
		}
	}
	for k, n := range counts {
		r.Means[k] /= float64(n)
	}
	if baseline != nil {
		r.BaselineSha256 = baseline.ReportSha256
		if baseline.SuiteSha256 != r.SuiteSha256 || !baseline.Complete || baseline.EvaluatorVersion != Version {
			r.Violations = append(r.Violations, &qualityv1.GateViolation{Reason: "baseline suite/evaluator identity differs or baseline is incomplete"})
			r.Complete = false
			baseline = nil
		}
	}
	applyBudgets(r, baseline, m.Budgets, counts)
	r.Passed = r.Complete && len(r.Violations) == 0
	provesImprovement := false
	for _, b := range m.Budgets {
		provesImprovement = provesImprovement || b.MinImprovement != nil
	}
	r.PromotionEligible = r.Passed && provesImprovement && m.Judge != "disabled" && baseline != nil && baseline.CandidateModel != "" && (baseline.CandidateModel != r.CandidateModel || baseline.CandidateRevision != r.CandidateRevision)
	r.Limits = []string{"Reference metrics measure registered pixel/text agreement; learned judgments are model opinions and are not calibrated human preferences.", "Qualification applies only to these retained cases and the reported candidate/evaluator/judge identities."}
	if m.Judge == "disabled" {
		r.Limits = append(r.Limits, "Judge disabled: deterministic regression evidence only; this report cannot qualify a model promotion.")
	}
	r.DurationMs = time.Since(start).Milliseconds()
	SealReport(r)
	return r, nil
}

func readAsset(ctx context.Context, load Loader, key, expected, label string, s *qualityv1.CaseScore) []byte {
	if key == "" {
		if expected != "" {
			s.Errors = append(s.Errors, label+" hash provided without asset")
		}
		return nil
	}
	if load == nil {
		s.Errors = append(s.Errors, "asset loader unavailable")
		return nil
	}
	b, err := load(ctx, key)
	if err != nil {
		s.Errors = append(s.Errors, label+": "+err.Error())
		return nil
	}
	if len(b) == 0 || len(b) > 32<<20 {
		s.Errors = append(s.Errors, label+": empty or over 32 MiB")
		return nil
	}
	if expected != "" && !strings.EqualFold(expected, hash(b)) {
		s.Errors = append(s.Errors, label+" SHA-256 mismatch")
		return nil
	}
	return b
}

func validateJudge(j *qualityv1.JudgeResult, model string) error {
	if j == nil || !j.Validated || j.Provider != "ollama" || j.Model != model || strings.TrimSpace(j.Rationale) == "" {
		return fmt.Errorf("judge result is unvalidated, incomplete, nonlocal or model identity differs")
	}
	// Promotion evidence must identify the actual local route and the resource
	// claim that admitted it. A Judge implementation's Validated bit alone is
	// not an attestation: test doubles and alternate implementations must obey
	// the same evidence contract as GatewayJudge.
	if strings.TrimSpace(j.RouteEventId) == "" || strings.TrimSpace(j.CapacityClaimId) == "" || j.CapacityOwnership != "resource:ollama" || len(j.RubricSha256) != 64 {
		return fmt.Errorf("judge result lacks route, capacity or rubric evidence")
	}
	for _, v := range []float64{j.Fidelity, j.Instruction, j.Artifacts} {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1 {
			return fmt.Errorf("judge score is outside 0..1")
		}
	}
	return nil
}
func hash(b []byte) string { v := sha256.Sum256(b); return hex.EncodeToString(v[:]) }
func hashOptional(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return hash(b)
}

func hashMessage(m proto.Message) string {
	b, _ := proto.MarshalOptions{Deterministic: true}.Marshal(m)
	return hash(b)
}

// SealReport binds all evidence and presentation fields. The handler seals
// again after it assigns the durable job/result identity.
func SealReport(r *qualityv1.EvaluationReport) { r.ReportSha256 = ""; r.ReportSha256 = hashMessage(r) }

func VerifyReport(r *qualityv1.EvaluationReport) error {
	if r == nil || r.ReportSha256 == "" {
		return fmt.Errorf("report has no content identity")
	}
	c := proto.Clone(r).(*qualityv1.EvaluationReport)
	expected := c.ReportSha256
	c.ReportSha256 = ""
	if hashMessage(c) != expected {
		return fmt.Errorf("report SHA-256 mismatch")
	}
	return nil
}

func applyBudgets(r, baseline *qualityv1.EvaluationReport, budgets []*qualityv1.MetricBudget, counts map[string]int) {
	base := map[string]*qualityv1.CaseScore{}
	if baseline != nil {
		for _, s := range baseline.Cases {
			base[s.Id] = s
		}
	}
	for _, b := range budgets {
		if b.Scope == "mean" {
			v, ok := r.Means[b.Metric]
			ok = ok && counts[b.Metric] == len(r.Cases)
			var bv *float64
			if baseline != nil {
				if x, has := baseline.Means[b.Metric]; has {
					bv = &x
				}
			}
			checkBudget(r, "mean", b, v, ok, bv)
		} else {
			for _, s := range r.Cases {
				v, ok := s.Metrics[b.Metric]
				var bv *float64
				if bs := base[s.Id]; bs != nil && bs.CaseSha256 == s.CaseSha256 {
					if x, has := bs.Metrics[b.Metric]; has {
						bv = &x
					}
				}
				checkBudget(r, s.Id, b, v, ok, bv)
			}
		}
	}
}

func checkBudget(r *qualityv1.EvaluationReport, id string, b *qualityv1.MetricBudget, v float64, ok bool, baseline *float64) {
	add := func(reason string, observed, limit *float64) {
		r.Violations = append(r.Violations, &qualityv1.GateViolation{CaseId: id, Metric: b.Metric, Reason: reason, Observed: observed, Limit: limit})
	}
	if !ok {
		add("required metric is missing", nil, nil)
		return
	}
	if b.Minimum != nil && v < *b.Minimum {
		add("below minimum", &v, b.Minimum)
	}
	if b.Maximum != nil && v > *b.Maximum {
		add("above maximum", &v, b.Maximum)
	}
	if b.MaxRegression != nil {
		if baseline == nil {
			add("regression budget requires comparable baseline evidence", nil, nil)
		} else {
			loss := (*baseline - v) * Direction[b.Metric]
			if loss > *b.MaxRegression+1e-12 {
				add("regression exceeds budget", &loss, b.MaxRegression)
			}
		}
	}
	if b.MinImprovement != nil {
		if baseline == nil {
			add("improvement budget requires comparable baseline evidence", nil, nil)
		} else {
			gain := (v - *baseline) * Direction[b.Metric]
			if gain+1e-12 < *b.MinImprovement {
				add("improvement below required gain", &gain, b.MinImprovement)
			}
		}
	}
}
