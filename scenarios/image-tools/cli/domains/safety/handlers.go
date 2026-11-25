// Package safety is the CLI's Responsible-Use policy surface. `safety policy`
// mirrors the Connect SafetyService.GetPolicy RPC — it reports the RESOLVED
// deployment policy (tier preset + per-control overrides; local defaults to
// permissive, public turns on consent + content scan + provenance + rate-limit)
// and the per-operation consent weights.
package safety

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/types/known/timestamppb"

	safetyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/safety"
	safetyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/safety/safety_v1connect"
)

// GroupName is the manifest group name this package owns.
const GroupName = "safety"

type handlers struct {
	core *cliapp.ScenarioApp
}

func newHandlers(core *cliapp.ScenarioApp) *handlers { return &handlers{core: core} }

// policy mirrors SafetyService.GetPolicy (Connect discovery).
func (h *handlers) policy(ctx cliapp.RunContext) error {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
	client := safetyconnect.NewSafetyServiceClient(httpClient, baseURL)
	resp, err := client.GetPolicy(context.Background(), connect.NewRequest(&safetyv1.GetPolicyRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get policy", err, nil)
	}
	p := resp.Msg
	results := []string{
		fmt.Sprintf("tier:               %s", tierLabel(p.GetTier())),
		fmt.Sprintf("require consent:    %t", p.GetRequireConsent()),
		fmt.Sprintf("force NSFW scan:    %t", p.GetForceNsfwScan()),
		fmt.Sprintf("require provenance: %t", p.GetRequireProvenance()),
		fmt.Sprintf("rate limit/min:     %d", p.GetRateLimitPerMin()),
	}
	for _, ow := range p.GetOpWeights() {
		results = append(results, fmt.Sprintf("op %-16s %s", ow.GetOperation(), weightLabel(ow.GetWeight())))
	}
	return cliapp.RenderProtoList(ctx, p, cliapp.ListReport{
		Summary:        []string{p.GetSummary()},
		ResultsHeading: "Policy",
		Results:        results,
	})
}

func tierLabel(t safetyv1.DeploymentTier) string {
	switch t {
	case safetyv1.DeploymentTier_DEPLOYMENT_TIER_PUBLIC:
		return "public"
	case safetyv1.DeploymentTier_DEPLOYMENT_TIER_LOCAL:
		return "local"
	default:
		return "local"
	}
}

func weightLabel(w safetyv1.ConsentWeight) string {
	switch w {
	case safetyv1.ConsentWeight_CONSENT_WEIGHT_HIGH:
		return "high (consent-gated when the consent control is on)"
	case safetyv1.ConsentWeight_CONSENT_WEIGHT_LOW:
		return "low"
	default:
		return "none"
	}
}

// feedback mirrors SafetyService.RecordSafetyFeedback. The manifest aliases it
// as `safety mark-legitimate`; the default label is legitimate (a false
// positive), so the common correction is one command with no flags.
func (h *handlers) feedback(ctx cliapp.RunContext) error {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
	client := safetyconnect.NewSafetyServiceClient(httpClient, baseURL)
	label, err := feedbackLabel(ctx.Flag("label"))
	if err != nil {
		return err
	}
	jobID := strings.TrimSpace(ctx.Positional("job_id"))
	resp, err := client.RecordSafetyFeedback(context.Background(), connect.NewRequest(&safetyv1.RecordSafetyFeedbackRequest{
		JobId:     jobID,
		OutputRef: strings.TrimSpace(ctx.Flag("ref")),
		Label:     label,
		Note:      ctx.Flag("note"),
		Actor:     actorName(ctx.Flag("actor")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("record safety feedback for job %q", jobID), err, nil)
	}
	item := resp.Msg
	return cliapp.RenderProtoMutation(ctx, item, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Recorded %s feedback for job %s.", labelName(item.GetLabel()), jobID)},
		Changes: []string{
			formatFeedbackCase(item),
		},
		NextCommand: []string{
			"`safety feedback-stats` — see the labelled corpus's false-positive rate",
			fmt.Sprintf("`safety unmark %s` — remove this judgement", jobID),
		},
	})
}

// unmark mirrors SafetyService.ClearSafetyFeedback.
func (h *handlers) unmark(ctx cliapp.RunContext) error {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
	client := safetyconnect.NewSafetyServiceClient(httpClient, baseURL)
	jobID := strings.TrimSpace(ctx.Positional("job_id"))
	resp, err := client.ClearSafetyFeedback(context.Background(), connect.NewRequest(&safetyv1.ClearSafetyFeedbackRequest{
		JobId: jobID, OutputRef: strings.TrimSpace(ctx.Flag("ref")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("remove safety feedback for job %q", jobID), err, nil)
	}
	result := fmt.Sprintf("No judgement existed for job %s.", jobID)
	if resp.Msg.GetCleared() {
		result = fmt.Sprintf("Removed the judgement for job %s.", jobID)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{result}})
}

// feedbackList mirrors SafetyService.ListSafetyFeedback.
func (h *handlers) feedbackList(ctx cliapp.RunContext) error {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
	client := safetyconnect.NewSafetyServiceClient(httpClient, baseURL)
	request := &safetyv1.ListSafetyFeedbackRequest{
		ReasonCode: strings.TrimSpace(ctx.Flag("reason")),
		ModelId:    strings.TrimSpace(ctx.Flag("model")),
		Operation:  strings.TrimSpace(ctx.Flag("operation")),
		JobId:      strings.TrimSpace(ctx.Flag("job")),
	}
	if raw := strings.TrimSpace(ctx.Flag("label")); raw != "" {
		label, err := feedbackLabel(raw)
		if err != nil {
			return err
		}
		request.Label = label
	}
	limit, err := parseLimit(ctx.Flag("limit"), 100, 1000)
	if err != nil {
		return err
	}
	request.Limit = int32(limit)
	if raw := strings.TrimSpace(ctx.Flag("offset")); raw != "" {
		offset, err := strconv.Atoi(raw)
		if err != nil || offset < 0 {
			return fmt.Errorf("invalid --offset %q: expected a non-negative integer", raw)
		}
		request.Offset = int32(offset)
	}
	resp, err := client.ListSafetyFeedback(context.Background(), connect.NewRequest(request))
	if err != nil {
		return cliapp.WrapAPIError("list safety feedback", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetCases()))
	for _, item := range resp.Msg.GetCases() {
		results = append(results, formatFeedbackCase(item))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d case(s) of %d.", len(resp.Msg.GetCases()), resp.Msg.GetTotal())},
		ResultsHeading: "Labelled reviewer verdicts",
		Results:        results,
		RetrievalHints: []string{
			"`safety corpus --out corpus.jsonl` — export the corpus for calibration",
			"`safety feedback-stats` — false-positive rate by reason and profile",
		},
	})
}

// feedbackStats mirrors SafetyService.GetSafetyFeedbackStats.
func (h *handlers) feedbackStats(ctx cliapp.RunContext) error {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
	client := safetyconnect.NewSafetyServiceClient(httpClient, baseURL)
	since, err := parseSince(ctx.Flag("since"))
	if err != nil {
		return err
	}
	resp, err := client.GetSafetyFeedbackStats(context.Background(), connect.NewRequest(&safetyv1.GetSafetyFeedbackStatsRequest{Since: since}))
	if err != nil {
		return cliapp.WrapAPIError("get safety feedback stats", err, nil)
	}
	stats := resp.Msg
	results := []string{
		fmt.Sprintf("total: %d · false positives: %d · confirmed: %d · uncertain: %d", stats.GetTotal(), stats.GetFalsePositives(), stats.GetConfirmed(), stats.GetUncertain()),
		fmt.Sprintf("false-positive rate: %s (%d decisive)", percent(stats.GetFalsePositiveRate()), stats.GetFalsePositives()+stats.GetConfirmed()),
	}
	for _, bucket := range stats.GetByReason() {
		results = append(results, fmt.Sprintf("reason %-24s cases=%d fp=%d confirmed=%d rate=%s",
			bucket.GetReasonCode(), bucket.GetCases(), bucket.GetFalsePositives(), bucket.GetConfirmed(), percent(bucket.GetFalsePositiveRate())))
	}
	for _, bucket := range stats.GetByProfile() {
		results = append(results, fmt.Sprintf("profile %-23s cases=%d fp=%d confirmed=%d rate=%s",
			orNone(bucket.GetProfileName()), bucket.GetCases(), bucket.GetFalsePositives(), bucket.GetConfirmed(), percent(bucket.GetFalsePositiveRate())))
	}
	return cliapp.RenderProtoList(ctx, stats, cliapp.ListReport{
		Summary:        []string{"Labelled reviewer corpus"},
		ResultsHeading: "Feedback stats",
		Results:        results,
		RetrievalHints: []string{"`safety feedback-list` — inspect the individual judgements"},
	})
}

// quarantine mirrors SafetyService.ListQuarantinedOutputs: the bounded admin
// review queue. Local deployments are ungated; a public deployment requires the
// configured admin token, which the CLI sends from
// IMAGE_TOOLS_SAFETY_QUARANTINE_TOKEN.
func (h *handlers) quarantine(ctx cliapp.RunContext) error {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
	client := safetyconnect.NewSafetyServiceClient(httpClient, baseURL)
	limit, err := parseLimit(ctx.Flag("limit"), 100, 500)
	if err != nil {
		return err
	}
	request := &safetyv1.ListQuarantinedOutputsRequest{Limit: int32(limit)}
	if raw := strings.TrimSpace(ctx.Flag("offset")); raw != "" {
		offset, err := strconv.Atoi(raw)
		if err != nil || offset < 0 {
			return fmt.Errorf("invalid --offset %q: expected a non-negative integer", raw)
		}
		request.Offset = int32(offset)
	}
	call := connect.NewRequest(request)
	if token := strings.TrimSpace(os.Getenv("IMAGE_TOOLS_SAFETY_QUARANTINE_TOKEN")); token != "" {
		call.Header().Set("Authorization", "Bearer "+token)
	}
	resp, err := client.ListQuarantinedOutputs(context.Background(), call)
	if err != nil {
		return cliapp.WrapAPIError("list quarantined outputs", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetOutputs()))
	for _, item := range resp.Msg.GetOutputs() {
		results = append(results, formatQuarantinedOutput(item))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d quarantined output(s) of %d pending.", len(resp.Msg.GetOutputs()), resp.Msg.GetTotal())},
		ResultsHeading: "Quarantined refusals",
		Results:        results,
		RetrievalHints: []string{
			"`GET /api/v1/safety/quarantine/<id>?download=1` — download one (append `X-Image-Tools-Admin-Token` on a public deployment)",
			"`safety feedback <job> --ref <original_ref> --label legitimate` — label the refusal",
		},
	})
}

func formatQuarantinedOutput(item *safetyv1.QuarantinedOutput) string {
	line := fmt.Sprintf("id=%d job=%s ref=%s reason=%s score=%.2f",
		item.GetId(), item.GetJobId(), orNone(item.GetOriginalRef()), orNone(item.GetReasonCode()), item.GetScore())
	if item.GetExpiresAt() != nil {
		line += " expires=" + item.GetExpiresAt().AsTime().Format(time.RFC3339)
	}
	return line
}

// corpus mirrors SafetyService.ExportSafetyCorpus and writes JSON Lines (the
// command is also the supported calibration handoff to
// scripts/safety-corpus-replay.py).
func (h *handlers) corpus(ctx cliapp.RunContext) error {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(h.core)
	client := safetyconnect.NewSafetyServiceClient(httpClient, baseURL)
	request := &safetyv1.ExportSafetyCorpusRequest{}
	if raw := strings.TrimSpace(ctx.Flag("label")); raw != "" {
		label, err := feedbackLabel(raw)
		if err != nil {
			return err
		}
		request.Label = label
	}
	since, err := parseSince(ctx.Flag("since"))
	if err != nil {
		return err
	}
	request.Since = since
	limit, err := parseLimit(ctx.Flag("limit"), 5000, 20000)
	if err != nil {
		return err
	}
	request.Limit = int32(limit)
	resp, err := client.ExportSafetyCorpus(context.Background(), connect.NewRequest(request))
	if err != nil {
		return cliapp.WrapAPIError("export safety corpus", err, nil)
	}
	if out := strings.TrimSpace(ctx.Flag("out")); out != "" {
		if err := os.WriteFile(out, []byte(resp.Msg.GetJsonl()), 0o644); err != nil {
			return fmt.Errorf("write corpus %q: %w", out, err)
		}
		_, err := fmt.Fprintf(ctx.Stdout(), "Exported %d case(s) to %s.\n", resp.Msg.GetCases(), out)
		return err
	}
	_, err = fmt.Fprint(ctx.Stdout(), resp.Msg.GetJsonl())
	return err
}

func feedbackLabel(raw string) (safetyv1.SafetyFeedbackLabel, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "legitimate", "false-positive", "false_positive", "fp":
		return safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_FALSE_POSITIVE, nil
	case "confirmed", "confirm", "correct":
		return safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_CONFIRMED, nil
	case "uncertain", "unsure":
		return safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_UNCERTAIN, nil
	default:
		return safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_UNSPECIFIED,
			fmt.Errorf("invalid --label %q: expected legitimate, confirmed or uncertain", raw)
	}
}

func labelName(label safetyv1.SafetyFeedbackLabel) string {
	switch label {
	case safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_FALSE_POSITIVE:
		return "legitimate"
	case safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_CONFIRMED:
		return "confirmed"
	case safetyv1.SafetyFeedbackLabel_SAFETY_FEEDBACK_LABEL_UNCERTAIN:
		return "uncertain"
	default:
		return "unspecified"
	}
}

func formatFeedbackCase(item *safetyv1.SafetyFeedbackCase) string {
	line := fmt.Sprintf("%s %s — %s action=%s reason=%s review=%s score=%.2f",
		item.GetJobId(), orNone(item.GetOutputRef()), labelName(item.GetLabel()),
		orNone(item.GetAction()), orNone(item.GetReasonCode()), orNone(item.GetSafetyReview()), item.GetScore())
	if item.GetModelId() != "" {
		line += " model=" + item.GetModelId()
	}
	if item.GetProfileName() != "" {
		line += " profile=" + item.GetProfileName()
	}
	if item.GetRecordedAt() != nil {
		line += " at " + item.GetRecordedAt().AsTime().Format(time.RFC3339)
	}
	if item.GetNote() != "" {
		line += " — " + item.GetNote()
	}
	return line
}

func parseSince(raw string) (*timestamppb.Timestamp, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	at, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, fmt.Errorf("invalid --since %q: expected RFC3339 (e.g. 2026-11-23T00:00:00Z)", raw)
	}
	return timestamppb.New(at), nil
}

func parseLimit(raw string, fallback, maxLimit int) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return 0, fmt.Errorf("invalid --limit %q: expected a positive integer", raw)
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return limit, nil
}

func actorName(raw string) string {
	if actor := strings.TrimSpace(raw); actor != "" {
		return actor
	}
	return "operator"
}

func percent(rate float64) string {
	return fmt.Sprintf("%.1f%%", rate*100)
}

func orNone(value string) string {
	if strings.TrimSpace(value) == "" {
		return "(none)"
	}
	return value
}
