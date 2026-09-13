package heartbeat

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	ampb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// SupervisionState contains admission evidence only. It is not an effort ledger.
type SupervisionState struct {
	AccountingRef      string                         `json:"accountingRef"`
	WindowStartedAt    time.Time                      `json:"windowStartedAt"`
	WakesInWindow      int                            `json:"wakesInWindow"`
	AllowanceResumesAt time.Time                      `json:"allowanceResumesAt"`
	DiscoveryReads     uint64                         `json:"discoveryReads"`
	OwnerRunReads      uint64                         `json:"ownerRunReads"`
	AssessmentReads    uint64                         `json:"assessmentReads"`
	WakeAttempts       uint64                         `json:"wakeAttempts"`
	LastWake           *SupervisionWake               `json:"lastWake,omitempty"`
	Version            int                            `json:"version"`
	Status             string                         `json:"status"`
	LastScanAt         time.Time                      `json:"lastScanAt"`
	LastSuccessAt      time.Time                      `json:"lastSuccessAt"`
	LastWakeAt         time.Time                      `json:"lastWakeAt"`
	NextCursor         string                         `json:"nextCursor,omitempty"`
	Coverage           string                         `json:"coverage,omitempty"`
	Error              string                         `json:"error,omitempty"`
	QuotaObservations  []*ampb.EffortQuotaObservation `json:"quotaObservations,omitempty"`
	Sequence           uint64                         `json:"sequence"`
	Efforts            map[string]SupervisedCut       `json:"efforts"`
	Pending            *SupervisionWake               `json:"pending,omitempty"`
}

type SupervisedCut struct {
	EffortObservation
	ServedRevision      string    `json:"servedRevision,omitempty"`
	LastServed          uint64    `json:"lastServed"`
	LastSampleAt        time.Time `json:"lastSampleAt"`
	LastSampleAttemptAt time.Time `json:"lastSampleAttemptAt,omitempty"`
	LastAssessedAt      time.Time `json:"lastAssessedAt"`
	LastAttempt         uint64    `json:"lastAttempt"`
	RetryAfter          time.Time `json:"retryAfter"`
	AssessmentID        string    `json:"assessmentId,omitempty"`
	Disposition         string    `json:"disposition,omitempty"`
}

type SupervisionWake struct {
	AccountingRef           string                         `json:"accountingRef"`
	ID                      string                         `json:"id"`
	Efforts                 []EffortObservation            `json:"efforts"`
	ProfileKey              string                         `json:"profileKey"`
	TaskID                  string                         `json:"taskId,omitempty"`
	RunID                   string                         `json:"runId,omitempty"`
	DispatchStarted         bool                           `json:"dispatchStarted"`
	DispatchMode            string                         `json:"dispatchMode,omitempty"`
	DispatchEffortRef       string                         `json:"dispatchEffortRef,omitempty"`
	DispatchAuthorizationID string                         `json:"dispatchAuthorizationId,omitempty"`
	DispatchReplayAttempts  uint8                          `json:"dispatchReplayAttempts,omitempty"`
	DispatchReplayError     string                         `json:"dispatchReplayError,omitempty"`
	CreatedAt               time.Time                      `json:"createdAt"`
	DiscoveryCursor         string                         `json:"discoveryCursor,omitempty"`
	QuotaObservations       []*ampb.EffortQuotaObservation `json:"quotaObservations,omitempty"`
	SampledEffortIDs        []string                       `json:"sampledEffortIds,omitempty"`
	Disposition             string                         `json:"disposition,omitempty"`
	AssessmentError         string                         `json:"assessmentError,omitempty"`
	TerminalStatus          string                         `json:"terminalStatus,omitempty"`
	TerminalError           string                         `json:"terminalError,omitempty"`
}

type SupervisionStateStore interface {
	Load(teamID, agentID string) (*SupervisionState, error)
	Save(teamID, agentID string, state *SupervisionState) error
}

// FileSupervisionStateStore uses the heartbeat runtime root, never authored team
// files. Hashing the identity keeps arbitrary IDs out of filesystem paths.
type FileSupervisionStateStore struct{ Root string }

func (s FileSupervisionStateStore) path(teamID, agentID string) string {
	id := sha256.Sum256([]byte(teamID + "\x00" + agentID))
	return filepath.Join(s.Root, "effort-supervision", hex.EncodeToString(id[:])+".json")
}

func (s FileSupervisionStateStore) Load(teamID, agentID string) (*SupervisionState, error) {
	b, err := os.ReadFile(s.path(teamID, agentID))
	if os.IsNotExist(err) {
		return &SupervisionState{Version: 1, Status: "idle", Efforts: map[string]SupervisedCut{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var state SupervisionState
	if err := json.Unmarshal(b, &state); err != nil {
		return nil, fmt.Errorf("read supervision reservation: %w", err)
	}
	if state.Version != 1 || state.Efforts == nil {
		return nil, fmt.Errorf("unsupported supervision state; preserve reservation for owner recovery")
	}
	return &state, nil
}

func (s FileSupervisionStateStore) Save(teamID, agentID string, state *SupervisionState) error {
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	path := s.path(teamID, agentID)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".wake-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
