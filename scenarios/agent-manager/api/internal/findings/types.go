package findings

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Finding struct {
	ID                 uuid.UUID `db:"id" json:"id"`
	RunID              uuid.UUID `db:"run_id" json:"runId"`
	InvestigationRunID uuid.UUID `db:"investigation_run_id" json:"investigationRunId"`
	Category           string    `db:"category" json:"category"`
	Severity           string    `db:"severity" json:"severity"`
	Recommendation     string    `db:"recommendation_text" json:"recommendation"`
	Evidence           string    `db:"evidence" json:"evidence,omitempty"`
	TargetPath         string    `db:"target_path" json:"targetPath,omitempty"`
	Fingerprint        string    `db:"fingerprint" json:"fingerprint"`
	Decision           string    `db:"operator_decision" json:"decision,omitempty"`
	CreatedAt          time.Time `db:"created_at" json:"createdAt"`
	Occurrences        int       `db:"occurrences" json:"occurrences,omitempty"`
	TargetMeasure      string    `db:"target_measure" json:"targetMeasure,omitempty"`
	BeforeValue        *float64  `db:"before_value" json:"beforeValue,omitempty"`
	AfterValue         *float64  `db:"after_value" json:"afterValue,omitempty"`
	Effectiveness      string    `db:"effectiveness" json:"effectiveness,omitempty"`
	FrictionTopic      string    `db:"friction_topic" json:"frictionTopic,omitempty"`
}

func Fingerprint(recommendation, targetPath string) string {
	normalize := func(value string) string {
		return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
	}
	digest := sha256.Sum256([]byte(normalize(targetPath) + "\x00" + normalize(recommendation)))
	return hex.EncodeToString(digest[:])
}

func Effectiveness(before, after *float64) string {
	if before == nil || after == nil {
		return "not_yet_measurable"
	}
	if *after < *before {
		return "effective"
	}
	return "ineffective"
}
