package hub

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/vrooli/api-core/decisionquality"
)

// qualitySummary is the one line a list row can show: the first two findings,
// in the producer's words. It is derived from the stored report, so a rule
// change reaches old rows on the next backfill without a second column.
func qualitySummary(report string) string {
	if strings.TrimSpace(report) == "" {
		return ""
	}
	var parsed struct {
		Grade    string `json:"grade"`
		Findings []struct {
			Message string `json:"message"`
		} `json:"findings"`
	}
	if json.Unmarshal([]byte(report), &parsed) != nil || parsed.Grade == "" || parsed.Grade == "ready" {
		return ""
	}
	var parts []string
	for _, finding := range parsed.Findings {
		if message := strings.TrimSpace(finding.Message); message != "" {
			parts = append(parts, message)
		}
		if len(parts) == 2 {
			break
		}
	}
	if len(parts) == 0 {
		return ""
	}
	summary := strings.Join(parts, "; ")
	runes := []rune(summary)
	if len(runes) > 180 {
		summary = string(runes[:177]) + "…"
	}
	return summary
}

// qualityInput builds the model's input from a sealed request and the findings
// the Hub already produced. Findings are mapped, never re-derived.
func qualityInput(spec AskSpec, warnings []PacketWarning) decisionquality.Input {
	options := make([]decisionquality.Option, 0, len(spec.Options))
	for _, option := range spec.Options {
		options = append(options, decisionquality.Option{Key: option.Key, Label: option.Label})
	}
	input := decisionquality.Input{
		Question:             spec.Question,
		Title:                spec.Title,
		Options:              options,
		Recommended:          spec.Recommended,
		RecommendationReason: spec.RecommendationReason,
		BigPicture:           spec.BigPicture,
		Packet:               spec.Packet,
		QuestionCount:        len(spec.Questions),
		Refusable:            spec.AgentSent,
	}
	for _, warning := range warnings {
		input.Findings = append(input.Findings, qualityFinding(warning))
	}
	return input
}

// qualityFinding maps one Hub warning into the operator-facing model. Owner is
// the part that can fix it; a page finding is the page's, a related-ask finding
// is the Hub's, and everything else is the producer's.
func qualityFinding(warning PacketWarning) decisionquality.Finding {
	severity := decisionquality.Advice
	if warning.Severity == WarningBlocking {
		severity = decisionquality.Blocking
	}
	owner := decisionquality.Producer
	code := warning.Code
	switch {
	case strings.HasPrefix(code, "explainer_"), strings.HasPrefix(code, "page_"), code == WarnCommandHidden:
		owner = decisionquality.Page
	case code == WarnRelatedOpen:
		owner = decisionquality.Hub
	}
	return decisionquality.Finding{
		Code: code, Severity: severity, Field: warning.Field, Owner: owner, Message: warning.Message,
	}
}

// assessQuality runs the model over an already-sealed request and its findings.
func assessQuality(spec AskSpec, warnings []PacketWarning) decisionquality.Report {
	return decisionquality.Assess(qualityInput(spec, warnings))
}

// recordQuality stores the readiness report for one ask and returns the grade
// and JSON. A failure is the caller's to log: readiness never fails a send.
func (s *Service) recordQuality(ctx context.Context, askID string, spec AskSpec, warnings []PacketWarning) (string, string, error) {
	// A status update is read, not answered: it has no options or
	// recommendation by construction, so the answer-shape checks do not apply.
	if spec.Kind == "status" {
		return "", "", nil
	}
	report := assessQuality(spec, warnings)
	encoded, err := json.Marshal(report)
	if err != nil {
		return "", "", err
	}
	if _, err := s.db.ExecContext(ctx,
		`UPDATE asks SET quality_grade=?, quality_report=? WHERE id=?`,
		string(report.Grade), string(encoded), askID); err != nil {
		return "", "", err
	}
	return string(report.Grade), string(encoded), nil
}

// BackfillQuality grades open asks that were written before readiness existed.
// It is bounded and safe to run at boot: a failure on one ask is logged and the
// rest continue. It returns how many reports it stored.
func (s *Service) BackfillQuality(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 300
	}
	// Re-grade every open ask, not only the ungraded ones: a rule or prompt
	// change must reach the report without a manual reset. Bounded so boot
	// never waits on a large backlog.
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM asks
		WHERE state IN ('pending','escalated') AND kind <> 'status'
		ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return 0, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	stored := 0
	for _, id := range ids {
		if ctx.Err() != nil {
			return stored, ctx.Err()
		}
		if _, _, err := s.ReassessStoredAsk(ctx, id); err != nil {
			s.log.Printf("readiness: backfill %s: %v", id, err)
			continue
		}
		stored++
	}
	return stored, nil
}

// ReassessStoredAsk recomputes readiness from the stored ask and its current
// attachments. It is the backfill path and the repair path; the write path
// calls recordQuality with the findings it already has.
func (s *Service) ReassessStoredAsk(ctx context.Context, askID string) (string, string, error) {
	record, err := scanAsk(s.db.QueryRowContext(ctx, askSelect+` WHERE a.id = ?`, askID))
	if err != nil {
		return "", "", err
	}
	attachments, err := s.loadAttachments(ctx, askID)
	if err != nil {
		return "", "", err
	}
	spec := specFromAsk(record, attachments)
	warnings := s.checkPacket(ctx, spec, s.ReadAttachmentBlob, askID, record.Sender)
	return s.recordQuality(ctx, askID, spec, warnings)
}

// specFromAsk reconstructs the request terms the packet check reads from a
// stored record. It is not a full AskSpec: only the fields the checks use.
func specFromAsk(record AskRecord, attachments []AskAttachment) AskSpec {
	return AskSpec{
		Question:             record.Question,
		Title:                record.Title,
		Options:              record.Options,
		Recommended:          record.Recommended,
		RecommendationReason: record.RecommendationReason,
		BigPicture:           record.BigPicture,
		TechnicalDetails:     record.TechnicalDetails,
		Packet:               record.Packet,
		Kind:                 record.Kind,
		Source:               record.Source,
		SourceEventType:      record.SourceEventType,
		CreatedBy:            record.CreatedBy,
		CreatedConversation:  record.CreatedConversation,
		CreatedSubject:       record.CreatedSubject,
		SubjectKey:           record.SubjectKey,
		Sender:               record.Sender,
		Questions:            record.Questions,
		Attachments:          attachments,
		// The send path distinguishes an agent/owner direct ask from a verified
		// scenario service by identity, and only the former may be refused. A
		// stored record keeps the identity, so the backfill reconstructs the same
		// rule rather than treating every named sender as an agent.
		AgentSent:        strings.HasPrefix(record.CreatedBy, "run:") || strings.HasPrefix(record.CreatedBy, "owner:"),
		VerifiedProducer: record.CreatedBy != "" || record.SourceEventType != "",
	}
}
