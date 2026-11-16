package home

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Record file names inside a commission folder (FILE-CONTRACT.md §1).
const (
	CharterFile    = "README.md"
	StanceFile     = "STANCE-MESSAGE.txt"
	KernelFile     = "KERNEL.txt"
	PacketFile     = "PACKET.json"
	LaunchFile     = "LAUNCH.json"
	OperatorFile   = "OPERATOR.md"
	StateFile      = "STATE.md"
	LedgerFile     = "LEDGER.md"
	BaselineFile   = "BASELINE.json"
	SettlementFile = "SETTLEMENT.json"
)

// LaunchSchema and PacketSchema are the schema ids of the two JSON records.
const (
	LaunchSchema     = "research-desk.launch/v1"
	PacketSchema     = "research-desk.packet/v1"
	SettlementSchema = "research-desk.settlement/v1"
)

// HistoricalSettlement records the commission's historical disposition.
// It is intentionally independent of the live worker/session state.
type HistoricalSettlement struct {
	Schema        string     `json:"schema"`
	CommissionID  string     `json:"commission_id"`
	Status        string     `json:"status"`
	AsOf          time.Time  `json:"as_of"`
	ActivityStart *time.Time `json:"activity_start,omitempty"`
	ActivityEnd   *time.Time `json:"activity_end,omitempty"`
	RecordedAt    time.Time  `json:"recorded_at"`
	RecordedBy    string     `json:"recorded_by"`
	Reason        string     `json:"reason"`
	Evidence      []string   `json:"evidence"`
}

type HistoricalStatus string

const (
	HistoricalPrepared           HistoricalStatus = "prepared"
	HistoricalClosed             HistoricalStatus = "closed"
	HistoricalClosedPartial      HistoricalStatus = "closed_partial"
	HistoricalContinuedElsewhere HistoricalStatus = "continued_elsewhere"
)

func (s HistoricalStatus) Valid() bool {
	switch s {
	case HistoricalPrepared, HistoricalClosed, HistoricalClosedPartial, HistoricalContinuedElsewhere:
		return true
	default:
		return false
	}
}

// ReadHistoricalSettlement reads SETTLEMENT.json; absence is not a settlement.
func ReadHistoricalSettlement(path string) (*HistoricalSettlement, error) {
	var record HistoricalSettlement
	if err := readJSON(path, SettlementSchema, &record, func() string { return record.Schema }); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if record.CommissionID == "" || !HistoricalStatus(record.Status).Valid() || record.AsOf.IsZero() || record.RecordedAt.IsZero() || record.RecordedBy == "" || strings.TrimSpace(record.Reason) == "" || len(record.Evidence) == 0 {
		return nil, fmt.Errorf("%s is missing required settlement fields", path)
	}
	if (record.ActivityStart == nil) != (record.ActivityEnd == nil) || (record.ActivityStart != nil && record.ActivityEnd.Before(*record.ActivityStart)) {
		return nil, fmt.Errorf("%s has an incomplete or reversed activity interval", path)
	}
	if record.Status != string(HistoricalPrepared) && record.ActivityStart == nil {
		return nil, fmt.Errorf("%s needs a cited activity interval for a completed disposition", path)
	}
	return &record, nil
}

// ValidateHistoricalEvidence verifies citations stay within the research home
// and name readable regular files.
func ValidateHistoricalEvidence(researchDir string, record *HistoricalSettlement) error {
	root, err := filepath.EvalSymlinks(researchDir)
	if err != nil {
		return fmt.Errorf("resolve research home: %w", err)
	}
	for _, citation := range record.Evidence {
		if citation == "" || filepath.IsAbs(citation) || filepath.Clean(citation) != citation {
			return fmt.Errorf("%w: evidence citations must be clean paths relative to the research home", ErrInvalid)
		}
		resolved, err := filepath.EvalSymlinks(filepath.Join(researchDir, filepath.FromSlash(citation)))
		if err != nil {
			return fmt.Errorf("%w: evidence file %q cannot be read: %v", ErrInvalid, citation, err)
		}
		rel, err := filepath.Rel(root, resolved)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("%w: evidence file %q is outside the research home", ErrInvalid, citation)
		}
		info, err := os.Stat(resolved)
		if err != nil || !info.Mode().IsRegular() {
			return fmt.Errorf("%w: evidence citation %q must name a regular file", ErrInvalid, citation)
		}
	}
	return nil
}

// LaunchRecord is LAUNCH.json (FILE-CONTRACT.md §3): only what cannot be
// derived from other files.
type LaunchRecord struct {
	Schema              string `json:"schema"`
	CommissionID        string `json:"commission_id"`
	LaunchedBy          string `json:"launched_by"`
	IdempotencyKey      string `json:"idempotency_key,omitempty"`
	WebConsoleSessionID string `json:"web_console_session_id,omitempty"`
	ThreadID            string `json:"thread_id,omitempty"`
	// WebConsoleAccess is how the desk reached Web Console: "service
	// credential" or "local availability policy (no desk credential)".
	WebConsoleAccess string         `json:"web_console_access,omitempty"`
	Steps            []LaunchStep   `json:"steps"`
	Failed           *LaunchFailure `json:"failed"`
	Discarded        *Discard       `json:"discarded"`
	Controls         []ControlEntry `json:"controls"`
	// Supersedes are the commissions this launch stopped and replaced, by the
	// operator's choice; RunAlongside those it runs beside.
	Supersedes   []string `json:"supersedes,omitempty"`
	RunAlongside []string `json:"run_alongside,omitempty"`
	// SupersededBy is set on a commission a later launch replaced.
	SupersededBy string `json:"superseded_by,omitempty"`
	// NeverStarted is the operator's assertion that this commission, which
	// the desk could not prove unstarted, never started.
	NeverStarted *Assertion `json:"never_started,omitempty"`
}

// Assertion is the operator's word on a fact the desk cannot know, with
// what the desk searched.
type Assertion struct {
	At       time.Time `json:"at"`
	Evidence string    `json:"evidence"`
}

// LaunchStep is one completed step of an automatic launch.
type LaunchStep struct {
	Step string    `json:"step"`
	At   time.Time `json:"at"`
}

// LaunchFailure records the step an automatic launch stopped at.
type LaunchFailure struct {
	Step   string `json:"step"`
	Reason string `json:"reason"`
}

// Discard records the operator discarding a commission.
type Discard struct {
	At     time.Time `json:"at"`
	Reason string    `json:"reason"`
}

// ControlEntry records one control action typed into the session.
type ControlEntry struct {
	Action string    `json:"action"`
	At     time.Time `json:"at"`
	// Outcome says what a stop or supersede found and left: "window
	// stopped" or "no window found", and whether anything still runs.
	Outcome string `json:"outcome,omitempty"`
}

// PacketRecord is PACKET.json, written by packet.py build.
type PacketRecord struct {
	Schema          string          `json:"schema"`
	CommissionID    string          `json:"commission_id"`
	BuiltAt         string          `json:"built_at"`
	TemplateVersion string          `json:"template_version"`
	Spec            json.RawMessage `json:"spec"`
}

// PacketSpec is the subset of the packet spec the desk reads. The full spec
// is packet.py's input (FILE-CONTRACT.md §2) and is kept verbatim in
// PacketRecord.Spec.
type PacketSpec struct {
	Topics           []string `json:"topics"`
	StanceQuotes     []string `json:"stance_quotes"`
	AllowanceHours   int      `json:"allowance_hours"`
	SupervisorModel  string   `json:"supervisor_model"`
	SupervisorEffort string   `json:"supervisor_effort"`
	RunAlongside     []string `json:"run_alongside"`
	Supersedes       IDs      `json:"supersedes"`
}

// IDs is a spec's supersedes: a list of commission ids, read also from the
// single id or null that packets built before lists were allowed carry.
type IDs []string

// UnmarshalJSON reads null, one id or a list.
func (ids *IDs) UnmarshalJSON(data []byte) error {
	var one *string
	if err := json.Unmarshal(data, &one); err == nil {
		*ids = nil
		if one != nil && *one != "" {
			*ids = IDs{*one}
		}
		return nil
	}
	var list []string
	if err := json.Unmarshal(data, &list); err != nil {
		return fmt.Errorf("supersedes is neither a commission id nor a list of them: %w", err)
	}
	*ids = list
	return nil
}

// readJSON decodes a JSON file and checks its schema id.
func readJSON(path, schema string, into any, schemaOf func() string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, into); err != nil {
		return fmt.Errorf("%s is not valid JSON: %w", path, err)
	}
	if got := schemaOf(); got != schema {
		return fmt.Errorf("%s has schema %q; expected %q", path, got, schema)
	}
	return nil
}

// ReadLaunch reads a commission's LAUNCH.json. It returns (nil, nil) when
// the file does not exist.
func ReadLaunch(path string) (*LaunchRecord, error) {
	var record LaunchRecord
	if err := readJSON(path, LaunchSchema, &record, func() string { return record.Schema }); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

// ReadPacket reads a commission's PACKET.json and its spec subset. It returns
// (nil, nil, nil) when the file does not exist.
func ReadPacket(path string) (*PacketRecord, *PacketSpec, error) {
	var record PacketRecord
	if err := readJSON(path, PacketSchema, &record, func() string { return record.Schema }); err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	var spec PacketSpec
	if err := json.Unmarshal(record.Spec, &spec); err != nil {
		return nil, nil, fmt.Errorf("%s spec is not valid: %w", path, err)
	}
	return &record, &spec, nil
}
