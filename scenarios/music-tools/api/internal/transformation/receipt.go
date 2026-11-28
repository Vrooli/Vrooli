// Package transformation owns the evidence contract for edits to existing
// audio. A successful transformation job records the receipt in its durable
// job metadata; consumers must verify the owner job rather than trust a
// caller-supplied copy.
package transformation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"strings"

	"music-tools/internal/jobs"
)

const ReceiptMetaKey = "derivation_receipt_v1"

var sha256RE = regexp.MustCompile(`^[0-9a-f]{64}$`)

// DerivationReceipt is the immutable source-to-output evidence emitted by a
// completed remix or sample job. Source rights remain owned by the source
// catalog; InheritedRightsScope records the ceiling the consumer verified
// against its source rights decision and the transformation model/tool lane.
type DerivationReceipt struct {
	SchemaVersion        string  `json:"schema_version"`
	JobID                string  `json:"job_id"`
	Kind                 string  `json:"kind"`
	SourceTrackSHA256    string  `json:"source_track_sha256"`
	DerivedTrackSHA256   string  `json:"derived_track_sha256"`
	SourceStartSeconds   float64 `json:"source_start_seconds"`
	SourceEndSeconds     float64 `json:"source_end_seconds"`
	DerivedStartSeconds  float64 `json:"derived_start_seconds"`
	DerivedEndSeconds    float64 `json:"derived_end_seconds"`
	ResultRef            string  `json:"result_ref"`
	OperationVersion     string  `json:"operation_version"`
	ToolName             string  `json:"tool_name"`
	ToolVersion          string  `json:"tool_version"`
	ModelID              string  `json:"model_id,omitempty"`
	ModelRevision        string  `json:"model_revision,omitempty"`
	LicenseLane          string  `json:"license_lane,omitempty"`
	InheritedRightsScope string  `json:"inherited_rights_scope"`
}

// ReadJobReceipt accepts evidence only from a successful Music Tools job whose
// terminal record agrees with every identity field in the receipt.
func ReadJobReceipt(job jobs.Job) (DerivationReceipt, error) {
	if job.State != jobs.StateSucceeded {
		return DerivationReceipt{}, errors.New("transformation receipt requires a succeeded job")
	}
	if job.Operation != "transform.remix" && job.Operation != "transform.sample" {
		return DerivationReceipt{}, errors.New("job operation is not a remix or sample transformation")
	}
	raw := strings.TrimSpace(job.Meta[ReceiptMetaKey])
	if raw == "" {
		return DerivationReceipt{}, errors.New("job has no derivation receipt")
	}
	var receipt DerivationReceipt
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return DerivationReceipt{}, fmt.Errorf("decode derivation receipt: %w", err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return DerivationReceipt{}, errors.New("derivation receipt must contain one JSON value")
	}
	if err := validateReceipt(job, receipt); err != nil {
		return DerivationReceipt{}, err
	}
	return receipt, nil
}

// EncodeJobReceipt validates the owner-produced receipt before it enters the
// durable job result. The worker must compute DerivedTrackSHA256 from the
// bytes actually persisted at ResultRef before calling this helper.
func EncodeJobReceipt(jobID, operation, resultRef string, receipt DerivationReceipt) (string, error) {
	job := jobs.Job{ID: jobID, Operation: operation, State: jobs.StateSucceeded, ResultRef: resultRef}
	if receipt.JobID == "" {
		receipt.JobID = jobID
	}
	if receipt.ResultRef == "" {
		receipt.ResultRef = resultRef
	}
	if err := validateReceipt(job, receipt); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func validateReceipt(job jobs.Job, receipt DerivationReceipt) error {
	if receipt.SchemaVersion != "1" || receipt.JobID == "" || receipt.JobID != job.ID ||
		receipt.Kind != strings.TrimPrefix(job.Operation, "transform.") ||
		(receipt.Kind != "remix" && receipt.Kind != "sample") ||
		!sha256RE.MatchString(receipt.SourceTrackSHA256) || !sha256RE.MatchString(receipt.DerivedTrackSHA256) ||
		receipt.SourceTrackSHA256 == receipt.DerivedTrackSHA256 || receipt.ResultRef == "" || receipt.ResultRef != job.ResultRef ||
		!finiteNonnegative(receipt.SourceStartSeconds) || !finitePositiveAfter(receipt.SourceEndSeconds, receipt.SourceStartSeconds) ||
		!finiteNonnegative(receipt.DerivedStartSeconds) || !finitePositiveAfter(receipt.DerivedEndSeconds, receipt.DerivedStartSeconds) ||
		strings.TrimSpace(receipt.OperationVersion) == "" || strings.TrimSpace(receipt.ToolName) == "" || strings.TrimSpace(receipt.ToolVersion) == "" ||
		(receipt.InheritedRightsScope != "unknown" && receipt.InheritedRightsScope != "personal_only" && receipt.InheritedRightsScope != "commercial" && receipt.InheritedRightsScope != "blocked") {
		return errors.New("derivation receipt does not match a valid completed transformation job")
	}
	if (receipt.ModelID == "") != (receipt.ModelRevision == "") {
		return errors.New("model id and revision must be supplied together")
	}
	if receipt.ModelID != "" && strings.TrimSpace(receipt.LicenseLane) == "" {
		return errors.New("model-backed derivation receipt requires its license lane")
	}
	return nil
}

func finiteNonnegative(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}
func finitePositiveAfter(value, start float64) bool {
	return finiteNonnegative(value) && !math.IsNaN(start) && !math.IsInf(start, 0) && value > start
}
