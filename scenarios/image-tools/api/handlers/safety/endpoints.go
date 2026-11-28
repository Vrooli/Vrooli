package safety

import (
	"image-tools/internal/module"

	safetyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/safety/safety_v1connect"
)

// Endpoints describes the safety module's surface. GetPolicy is a generated
// Connect procedure (pure policy discovery); enforcement lives on the AI submit
// edge, so the safety domain exposes no execution endpoint of its own.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "safety_get_policy",
		Path:        safetyconnect.SafetyServiceGetPolicyProcedure,
		Method:      "POST",
		Summary:     "Get the resolved Responsible-Use deployment policy",
		Description: "Returns the resolved Responsible-Use policy for the running deployment tier (local = unrestricted; public = consent-gated + NSFW auto-scan + provenance + rate-limit), plus the per-operation consent-weight table. Pure discovery; enforcement happens server-side on the AI submit edge.",
		Category:    "safety",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"tier":               "DeploymentTier",
				"require_consent":    "bool",
				"force_nsfw_scan":    "bool",
				"require_provenance": "bool",
				"rate_limit_per_min": "int32",
				"op_weights":         "array<OpWeight>",
				"summary":            "string",
			},
		},
		Examples: []module.Example{
			{Name: "Get the active policy", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.safety.SafetyService/GetPolicy -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "safety_record_feedback",
		Path:        safetyconnect.SafetyServiceRecordSafetyFeedbackProcedure,
		Method:      "POST",
		Summary:     "Label one reviewer verdict as legitimate, correct or uncertain",
		Description: "Records the operator's judgement of an output the safety reviewer refused or flagged, snapshotting the reviewer evidence so the case survives job retention. Re-recording the same job/output replaces the label. Feedback is calibration data: it never restores a deleted output, unblurs a result, or changes a stored verdict.",
		Category:    "safety",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"id":               "int64",
				"job_id":           "string",
				"output_ref":       "string",
				"label":            "SafetyFeedbackLabel",
				"action":           "string",
				"reason_code":      "string",
				"safety_review":    "string",
				"score":            "double",
				"profile_name":     "string",
				"reviewer_version": "string",
				"recorded_at":      "Timestamp",
			},
		},
	},
	{
		ID:          "safety_clear_feedback",
		Path:        safetyconnect.SafetyServiceClearSafetyFeedbackProcedure,
		Method:      "POST",
		Summary:     "Remove one reviewer-feedback judgement",
		Description: "Deletes the recorded judgement for one job output. Returns cleared=false when no judgement existed.",
		Category:    "safety",
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"cleared": "bool"}},
	},
	{
		ID:          "safety_list_feedback",
		Path:        safetyconnect.SafetyServiceListSafetyFeedbackProcedure,
		Method:      "POST",
		Summary:     "List labelled reviewer verdicts newest first",
		Description: "Returns the refusal/flag corpus, optionally narrowed by label, reason code, model or operation, with the total matching count.",
		Category:    "safety",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"cases": "array<SafetyFeedbackCase>",
				"total": "int32",
			},
		},
	},
	{
		ID:          "safety_feedback_stats",
		Path:        safetyconnect.SafetyServiceGetSafetyFeedbackStatsProcedure,
		Method:      "POST",
		Summary:     "Summarise the labelled reviewer corpus",
		Description: "Aggregates false-positive and confirmed counts with the false-positive rate overall, per reason code and per reviewer profile, optionally since a timestamp.",
		Category:    "safety",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"total":               "int64",
				"false_positives":     "int64",
				"confirmed":           "int64",
				"uncertain":           "int64",
				"false_positive_rate": "double",
				"by_label":            "array<SafetyFeedbackLabelCount>",
				"by_reason":           "array<SafetyFeedbackReasonStat>",
				"by_profile":          "array<SafetyFeedbackProfileStat>",
			},
		},
	},
	{
		ID:          "safety_export_corpus",
		Path:        safetyconnect.SafetyServiceExportSafetyCorpusProcedure,
		Method:      "POST",
		Summary:     "Export the labelled reviewer corpus as JSON Lines",
		Description: "Renders one JSON object per line for offline calibration: the label, the reviewer's evidence, and the per-image finding when the case names one variation. The line schema is documented in SECURITY.md and read by scripts/safety-corpus-replay.py.",
		Category:    "safety",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"jsonl": "string",
				"cases": "int32",
			},
		},
	},
	{
		ID:          "safety_list_quarantined_outputs",
		Path:        safetyconnect.SafetyServiceListQuarantinedOutputsProcedure,
		Method:      "POST",
		Summary:     "List refused outputs retained for admin review",
		Description: "Returns the pending quarantine entries: job, original ref, reason, score, evidence and expiry. The bytes are not part of the response; they are served only by the dedicated quarantine route. Local deployments are ungated; a public deployment requires the configured admin token.",
		Category:    "safety",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"outputs": "array<QuarantinedOutput>",
				"total":   "int32",
			},
		},
	},
	{
		ID:          "safety_quarantine_image",
		Path:        "/api/v1/safety/quarantine/{id}",
		Method:      "GET",
		Summary:     "Fetch one quarantined refusal for admin review",
		Description: "Streams the retained bytes of a pending quarantine entry. The ordinary /api/v1/blobs/ route refuses the quarantine namespace. Local deployments are ungated; a public deployment requires the admin token. Add ?download=1 for an attachment.",
		Category:    "safety",
		Response:    &module.Schema{Type: "binary"},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No pending quarantine entry with that id"},
			{Status: 410, Code: "gone", Description: "The quarantine entry expired"},
			{Status: 403, Code: "forbidden", Description: "A public deployment requires the admin token"},
		},
		RESTException: &module.RESTException{
			Reason: module.RESTReasonBrowserSurface,
			Note:   "The review UI must load the retained image directly after an explicit reveal; a Connect unary RPC would base64-encode the bytes and cannot stream them.",
		},
	},
}
