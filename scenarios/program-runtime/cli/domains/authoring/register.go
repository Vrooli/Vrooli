package authoring

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	programsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs/programs_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// evalTimeout bounds one full corpus run.
const evalTimeout = 30 * time.Minute

func authoringEvalPasses(status string, floorMet bool) bool {
	return status == "measured" && floorMet
}

// Register exposes the versioned authoring corpus through the same typed
// service as program submission. The API deliberately returns unavailable
// when no code-authoring model route is configured; that state is meaningful
// evidence and must not be converted into a zero score by the CLI.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	client, base := cliapp.NewConnectHTTPClient(core)
	// One eval authors and executes every corpus case, so the wall time is a
	// multiple of a model round-trip rather than a single request. The shared
	// client deadline is sized for ordinary reads and cuts the run off before
	// it can produce a verdict, which reads as a transport failure rather than
	// as a measurement.
	client = &http.Client{Timeout: evalTimeout}
	h := programsconnect.NewProgramServiceClient(client, base)
	receiptSummary := ""
	return cliapp.LoadFromManifestPrimitives(manifest, "authoring", map[string]cliapp.PrimitiveHandler{
		"vrooli.program_runtime.v1.programs.ProgramService.RunAuthoringEval": cliapp.ProtoList(func(ctx cliapp.OperationContext) (*programsv1.RunAuthoringEvalResponse, error) {
			suite, err := resolveAuthoringSuitePath(ctx.Flag("suite"))
			if err != nil {
				return nil, err
			}
			r, err := h.RunAuthoringEval(context.Background(), connect.NewRequest(&programsv1.RunAuthoringEvalRequest{
				Suite: suite, NoGate: ctx.BoolFlag("no-gate"),
			}))
			if err != nil {
				return nil, cliapp.WrapAPIError("run authoring eval", err, nil)
			}
			receiptSummary = ""
			if path := ctx.Flag("receipt-file"); path != "" {
				digest, writeErr := writeAuthoringReceipt(path, r.Msg)
				if writeErr != nil {
					return nil, writeErr
				}
				receiptSummary = fmt.Sprintf("Raw API response receipt: %s (sha256=%s).", path, digest)
			}
			if !ctx.BoolFlag("no-gate") && !authoringEvalPasses(r.Msg.GetStatus(), r.Msg.GetFloorMet()) {
				detail := fmt.Sprintf("authoring eval %s: met=%d floor=%d", r.Msg.GetStatus(), r.Msg.GetMet(), r.Msg.GetFloor())
				if receiptSummary != "" {
					detail += "; " + receiptSummary
				}
				return r.Msg, fmt.Errorf("%s", detail)
			}
			return r.Msg, nil
		}, func(_ cliapp.OperationContext, result *programsv1.RunAuthoringEvalResponse) cliapp.ListReport {
			status := result.GetStatus()
			if status == "measured" && !result.GetFloorMet() {
				status = "below_floor"
			}
			summary := []string{authoringScoreSummary(result, status)}
			if receiptSummary != "" {
				summary = append(summary, receiptSummary)
			}
			for _, miss := range result.GetRuleMisses() {
				summary = append(summary, fmt.Sprintf("Rule miss: %s=%d.", miss.GetRuleId(), miss.GetCount()))
			}
			results := make([]string, 0, len(result.GetResults()))
			for _, item := range result.GetResults() {
				verdict := "miss"
				if item.GetFirstAttemptOk() {
					verdict = "met"
				}
				line := fmt.Sprintf("%s: %s rule=%s cause=%s detail=%s", item.GetCaseId(), verdict, item.GetRuleId(), item.GetCause(), item.GetFailureDetail())
				if timing := authoringCaseTiming(item); timing != "" {
					line += " " + timing
				}
				results = append(results, line)
			}
			return cliapp.ListReport{Summary: summary, ResultsHeading: "Corpus cases", Results: results, ListShaped: true, ResultCount: len(results)}
		}),
	})
}

// The API consumes suite paths on its own filesystem and otherwise resolves
// them against an unspecified process working directory. Resolve user input
// on the CLI host before sending it so a relative path has stable semantics.
func resolveAuthoringSuitePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	resolved, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve authoring suite path %q: %w", path, err)
	}
	return resolved, nil
}

func authoringScoreSummary(result *programsv1.RunAuthoringEvalResponse, status string) string {
	type panelStats struct {
		cases    int
		attempts int
		met      int
		working  []int64
	}
	panels := map[string]*panelStats{"primary": {}, "heldout": {}}
	attempted, met := 0, 0
	working := make([]int64, 0, result.GetMet())
	for _, item := range result.GetResults() {
		panel := "primary"
		if strings.HasPrefix(item.GetCaseId(), "heldout-") {
			panel = "heldout"
		}
		stats := panels[panel]
		stats.cases++
		caseStatus, hasStatus := protoString(item, "case_status")
		attemptCount, hasAttemptCount := protoInt64(item, "attempt_count")
		caseAttempted := false
		if hasStatus && caseStatus != "" {
			caseAttempted = caseStatus == "measured"
		} else if hasAttemptCount {
			caseAttempted = attemptCount > 0 || item.GetAuthored() || item.GetFirstAttemptOk() || item.GetCause() != "" || item.GetFailureDetail() != ""
		} else {
			caseAttempted = item.GetAuthored() || item.GetFirstAttemptOk() || item.GetCause() != "" || item.GetFailureDetail() != ""
		}
		if caseAttempted {
			attempted++
			stats.attempts++
		}
		if item.GetFirstAttemptOk() {
			met++
			stats.met++
			if value, ok := protoInt64(item, "elapsed_to_first_working_ms"); ok {
				working = append(working, value)
				stats.working = append(stats.working, value)
			}
		}
	}
	summary := fmt.Sprintf("Authoring eval: status=%s first_attempt=%d/%d attempted (fixed tasks=%d) floor=%d unavailable=%d not_attempted=%d; primary=%d/%d attempted (cases=%d); heldout=%d/%d attempted (cases=%d).",
		status, met, attempted, result.GetCases(), result.GetFloor(), result.GetUnavailable(), result.GetNotAttempted(),
		panels["primary"].met, panels["primary"].attempts, panels["primary"].cases,
		panels["heldout"].met, panels["heldout"].attempts, panels["heldout"].cases)
	summary += durationSummary("first-attempt time-to-working", working)
	summary += durationSummary("primary time-to-working", panels["primary"].working)
	summary += durationSummary("heldout time-to-working", panels["heldout"].working)
	return summary
}

func durationSummary(label string, samples []int64) string {
	if len(samples) == 0 {
		return " " + label + " (n=0): median/p90 unavailable."
	}
	median, p90 := durationPercentiles(samples)
	return fmt.Sprintf(" %s (n=%d): median=%dms p90=%dms.", label, len(samples), median, p90)
}

func authoringCaseTiming(item *programsv1.AuthoringCaseResult) string {
	parts := make([]string, 0, 9)
	attemptCount, hasAttemptCount := protoInt64(item, "attempt_count")
	if !hasAttemptCount {
		attemptCount = 0
		if item.GetAuthored() || item.GetFirstAttemptOk() || item.GetCause() != "" || item.GetFailureDetail() != "" {
			attemptCount = 1
		}
	}
	if attemptCount > 0 {
		if value, ok := protoInt64(item, "generation_elapsed_ms"); ok {
			parts = append(parts, fmt.Sprintf("generation_ms=%d", value))
		}
	}
	if item.GetAuthored() || item.GetFirstAttemptOk() {
		for _, field := range []struct {
			name  string
			label string
		}{
			{name: "execution_elapsed_ms", label: "execution_ms"},
			{name: "oracle_elapsed_ms", label: "oracle_ms"},
		} {
			if value, ok := protoInt64(item, field.name); ok {
				parts = append(parts, fmt.Sprintf("%s=%d", field.label, value))
			}
		}
	}
	if value, ok := protoInt64(item, "elapsed_to_first_working_ms"); ok && item.GetFirstAttemptOk() {
		parts = append(parts, fmt.Sprintf("time_to_working_ms=%d", value))
	}
	if caseStatus, ok := protoString(item, "case_status"); ok && caseStatus != "" {
		parts = append(parts, "case_status="+caseStatus)
	}
	if attempts, ok := protoInt64(item, "attempt_count"); ok {
		if attempts == 0 && (item.GetAuthored() || item.GetFirstAttemptOk() || item.GetCause() != "" || item.GetFailureDetail() != "") {
			attempts = 1
		}
		parts = append(parts, fmt.Sprintf("attempts=%d", attempts))
	}
	if retries, ok := protoInt64(item, "retry_count"); ok {
		parts = append(parts, fmt.Sprintf("retries=%d", retries))
	}
	if digest, ok := protoString(item, "source_sha256"); ok && digest != "" {
		parts = append(parts, "source_sha256="+digest)
	}
	if digest, ok := protoString(item, "task_sha256"); ok && digest != "" {
		parts = append(parts, "task_sha256="+digest)
	}
	if len(parts) == 0 {
		return ""
	}
	return "timing{" + strings.Join(parts, " ") + "}"
}

func protoInt64(message proto.Message, fieldName string) (int64, bool) {
	reflected := message.ProtoReflect()
	field := reflected.Descriptor().Fields().ByName(protoreflect.Name(fieldName))
	if field == nil || (field.HasPresence() && !reflected.Has(field)) {
		return 0, false
	}
	return reflected.Get(field).Int(), true
}

func protoString(message proto.Message, fieldName string) (string, bool) {
	reflected := message.ProtoReflect()
	field := reflected.Descriptor().Fields().ByName(protoreflect.Name(fieldName))
	if field == nil || (field.HasPresence() && !reflected.Has(field)) {
		return "", false
	}
	return reflected.Get(field).String(), true
}

// durationPercentiles returns the conventional median (averaging the middle
// pair for an even sample) and nearest-rank p90 from a copy of the input.
func durationPercentiles(samples []int64) (median, p90 int64) {
	if len(samples) == 0 {
		return 0, 0
	}
	sorted := append([]int64(nil), samples...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	middle := len(sorted) / 2
	if len(sorted)%2 == 0 {
		median = (sorted[middle-1] + sorted[middle]) / 2
	} else {
		median = sorted[middle]
	}
	p90Index := (9*len(sorted)+9)/10 - 1
	if p90Index < 0 {
		p90Index = 0
	}
	return median, sorted[p90Index]
}

// writeAuthoringReceipt retains the exact typed API response as deterministic
// proto JSON. The returned digest is explicitly a digest of these raw response
// bytes; it is not the server-authored self-hash proposed for a full timing
// receipt. Create-exclusive semantics keep one measurement from silently
// replacing another.
func writeAuthoringReceipt(path string, result *programsv1.RunAuthoringEvalResponse) (string, error) {
	options := protojson.MarshalOptions{UseProtoNames: true, Multiline: true, Indent: "  ", EmitUnpopulated: true}
	data, err := options.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal authoring response receipt: %w", err)
	}
	data = append(data, '\n')
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("create authoring response receipt %q: %w", path, err)
	}
	removePartial := true
	defer func() {
		_ = file.Close()
		if removePartial {
			_ = os.Remove(path)
		}
	}()
	if _, err := file.Write(data); err != nil {
		return "", fmt.Errorf("write authoring response receipt %q: %w", path, err)
	}
	if err := file.Sync(); err != nil {
		return "", fmt.Errorf("sync authoring response receipt %q: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("close authoring response receipt %q: %w", path, err)
	}
	removePartial = false
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}
