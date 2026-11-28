package model_eval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
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
		runtimeHome, err := repocontract.VrooliUserRoot(home)
		if err != nil {
			return "", err
		}
		root = filepath.Join(runtimeHome, "research")
	}
	return filepath.Join(root, "topics", "local-model-performance"), nil
}

var answerLinePattern = regexp.MustCompile(`^` + regexp.QuoteMeta(AdmissionDecisionID) + `:\s*(approve|defer|other)\b`)

const AdmissionPacketSHA256 = "0eac00d7e36ca235ea1b5bc90e13c28ad0777677701224f48ed3f11e7b17eeb4"

// TopicFileApprovalSource reads only a current operator answer from the
// research topic's INBOX.md. Historical answers and the pending decision row
// alone never grant dispatch authority.
type TopicFileApprovalSource struct {
	TopicDir string
	// expectedPacketSHA256 is set only by package tests. Production always
	// pins the packet digest reviewed with this decision.
	expectedPacketSHA256 string
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
	expectedPacketSHA := s.expectedPacketSHA256
	if expectedPacketSHA == "" {
		expectedPacketSHA = AdmissionPacketSHA256
	}
	if err != nil || hashBytes(packet) != expectedPacketSHA {
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
			// An answer is current only inside its own operator-answer section.
			// Never pair an older matching decision with a newer unrelated stamp.
			answer = ""
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
	if _, err := os.Lstat(s.dispatchMarkerPath()); err == nil {
		return Approval{}, errors.New("LM2 one-time dispatch approval was already consumed")
	} else if !errors.Is(err, os.ErrNotExist) {
		return Approval{}, errors.New("LM2 one-time dispatch marker could not be checked")
	}
	return Approval{DecisionID: id, Answer: answer, Scope: AdmissionScope, RecordedAt: recordedAt}, nil
}

type dispatchMarker struct {
	DecisionID string    `json:"decision_id"`
	Scope      string    `json:"scope"`
	CaseSetSHA string    `json:"case_set_sha256"`
	ClaimedAt  time.Time `json:"claimed_at"`
}

func (s TopicFileApprovalSource) ClaimDispatch(ctx context.Context, approval Approval, caseSetSHA string) error {
	if strings.TrimSpace(s.TopicDir) == "" || approval.DecisionID != AdmissionDecisionID || approval.Answer != "approve" || approval.Scope != AdmissionScope || approval.RecordedAt.IsZero() || !validHash(caseSetSHA) {
		return errors.New("LM2 one-time dispatch claim is invalid")
	}
	current, err := s.ReadDecision(ctx, AdmissionDecisionID)
	if err != nil || current.Answer != approval.Answer || current.Scope != approval.Scope || !current.RecordedAt.Equal(approval.RecordedAt) {
		return errors.New("current LM2 operator approval changed before dispatch")
	}
	packetDir := filepath.Join(s.TopicDir, "work", AdmissionDecisionID)
	info, err := os.Lstat(packetDir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("LM2 decision packet directory is unavailable")
	}
	marker := dispatchMarker{DecisionID: approval.DecisionID, Scope: approval.Scope, CaseSetSHA: caseSetSHA, ClaimedAt: time.Now().UTC()}
	data, err := json.Marshal(marker)
	if err != nil {
		return fmt.Errorf("encode LM2 one-time dispatch marker: %w", err)
	}
	file, err := os.OpenFile(s.dispatchMarkerPath(), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return errors.New("LM2 one-time dispatch approval was already consumed or marker is unavailable")
	}
	// Keep a partial marker on any error: once a durable create began, fail
	// closed and never risk a second live batch after an uncertain write.
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return errors.New("LM2 one-time dispatch marker write failed")
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return errors.New("LM2 one-time dispatch marker sync failed")
	}
	if err := file.Close(); err != nil {
		return errors.New("LM2 one-time dispatch marker close failed")
	}
	directory, err := os.Open(packetDir)
	if err != nil {
		return errors.New("LM2 one-time dispatch marker directory could not be opened")
	}
	syncErr := directory.Sync()
	closeErr := directory.Close()
	if syncErr != nil || closeErr != nil {
		return errors.New("LM2 one-time dispatch marker directory sync failed")
	}
	return nil
}

func (s TopicFileApprovalSource) dispatchMarkerPath() string {
	return filepath.Join(s.TopicDir, "work", AdmissionDecisionID, "dispatch-admission.json")
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
