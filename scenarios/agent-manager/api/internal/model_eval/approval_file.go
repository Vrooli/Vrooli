package model_eval

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ResolveTopicDir follows the operator's retained-research home. A deployment
// may point it elsewhere with VROOLI_RESEARCH_ROOT; the per-user home is the
// established local default.
func ResolveTopicDir() (string, error) {
	root := strings.TrimSpace(os.Getenv("VROOLI_RESEARCH_ROOT"))
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil || strings.TrimSpace(home) == "" {
			return "", errors.New("research topic home is unavailable")
		}
		root = filepath.Join(home, ".vrooli", "research")
	}
	return filepath.Join(root, "topics", "local-model-performance"), nil
}

var answerLinePattern = regexp.MustCompile(`^` + regexp.QuoteMeta(AdmissionDecisionID) + `:\s*(approve|defer|other)\b`)

// TopicFileApprovalSource reads only a current operator answer from the
// research topic's INBOX.md. Historical answers and the pending decision row
// alone never grant dispatch authority.
type TopicFileApprovalSource struct {
	TopicDir string
}

func (s TopicFileApprovalSource) ReadDecision(_ context.Context, id string) (Approval, error) {
	if id != AdmissionDecisionID || strings.TrimSpace(s.TopicDir) == "" {
		return Approval{}, errors.New("LM2 decision source is not configured")
	}
	decisionPath := filepath.Join(s.TopicDir, "DECISIONS.md")
	packetPath := filepath.Join(s.TopicDir, "work", AdmissionDecisionID, "BRIEF.md")
	inboxPath := filepath.Join(s.TopicDir, "INBOX.md")
	decision, err := os.ReadFile(decisionPath)
	if err != nil || !strings.Contains(string(decision), AdmissionDecisionID) {
		return Approval{}, errors.New("registered LM2 decision is unavailable")
	}
	packet, err := os.ReadFile(packetPath)
	if err != nil || !strings.Contains(strings.ReplaceAll(string(packet), "`", ""), AdmissionScope) {
		return Approval{}, errors.New("LM2 approval scope changed or is unavailable")
	}
	inbox, err := os.ReadFile(inboxPath)
	if err != nil {
		return Approval{}, errors.New("current operator inbox is unavailable")
	}
	var answer string
	var recordedAt time.Time
	operatorAnswer := false
	for _, line := range strings.Split(string(inbox), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "### ") {
			operatorAnswer = strings.Contains(strings.ToLower(trimmed), "from operator") && strings.Contains(strings.ToLower(trimmed), "answer")
			recordedAt = parseInboxTimestamp(trimmed)
			continue
		}
		if !operatorAnswer {
			continue
		}
		match := answerLinePattern.FindStringSubmatch(trimmed)
		if len(match) == 2 {
			answer = strings.ToLower(match[1])
		}
	}
	if answer == "" || recordedAt.IsZero() {
		return Approval{}, errors.New("no current operator answer for LM2 admission")
	}
	return Approval{DecisionID: id, Answer: answer, Scope: AdmissionScope, RecordedAt: recordedAt}, nil
}

func parseInboxTimestamp(header string) time.Time {
	value := strings.TrimSpace(strings.TrimPrefix(header, "### "))
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC()
		}
	}
	if end := strings.Index(value, " · "); end > 0 {
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
			if parsed, err := time.Parse(layout, strings.TrimSpace(value[:end])); err == nil {
				return parsed.UTC()
			}
		}
	}
	return time.Time{}
}
