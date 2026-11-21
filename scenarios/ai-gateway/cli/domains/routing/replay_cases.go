package routing

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

const (
	maxReplayInputBytes = 10 << 20
	maxReplayCases      = 1000 // accommodates overlapping role/task-role/consumer floors while keeping replay bounded.
)

var replayUUIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ReplayCase is one real caller request selected for a bounded mixed-load
// replay. The source references are mandatory so a fixture cannot silently
// turn generated prompts into acceptance evidence.
type ReplayCase struct {
	CaseID                      string `json:"case_id"`
	SourceRunID                 string `json:"source_run_id"`
	SourceEventID               string `json:"source_event_id"`
	SourceSequence              int64  `json:"source_sequence"`
	ConversationID              string `json:"conversation_id,omitempty"`
	Role                        string `json:"role"`
	TaskRole                    string `json:"task_role"`
	Profile                     string `json:"profile"`
	PrivacyClass                string `json:"privacy_class"`
	PrivacyProvenance           string `json:"privacy_provenance"`
	Operation                   string `json:"operation"`
	Scenario                    string `json:"scenario"`
	TimeoutMS                   int32  `json:"timeout_ms,omitempty"`
	RequiredVRAMBytes           int64  `json:"required_vram_bytes,omitempty"`
	CapacityEstimateEvidenceURI string `json:"capacity_estimate_evidence_uri,omitempty"`
	InputText                   string `json:"input_text"`
}

// ReplayReceipt intentionally excludes prompts and model output. It retains
// source provenance, route evidence and timing for the comparison record.
type ReplayReceipt struct {
	CaseID                      string                       `json:"case_id"`
	Mode                        string                       `json:"mode"`
	SourceRunID                 string                       `json:"source_run_id"`
	SourceEventID               string                       `json:"source_event_id"`
	SourceSequence              int64                        `json:"source_sequence"`
	ConversationID              string                       `json:"conversation_id,omitempty"`
	PrivacyProvenance           string                       `json:"privacy_provenance"`
	Scenario                    string                       `json:"scenario"`
	Role                        string                       `json:"role"`
	TaskRole                    string                       `json:"task_role"`
	SelectedResourceRole        string                       `json:"selected_resource_role,omitempty"`
	Status                      string                       `json:"status"`
	SelectedProvider            string                       `json:"selected_provider,omitempty"`
	GatewayEventID              string                       `json:"gateway_event_id,omitempty"`
	GatewayRequestID            string                       `json:"gateway_request_id,omitempty"`
	GatewayRoleEvidenceURI      string                       `json:"gateway_role_evidence_uri,omitempty"`
	CapacityClaimID             string                       `json:"capacity_claim_id,omitempty"`
	CapacityRequestedBytes      int64                        `json:"capacity_requested_bytes,omitempty"`
	CapacityRequiredBytes       int64                        `json:"capacity_required_bytes,omitempty"`
	CapacityGrantedBytes        int64                        `json:"capacity_granted_bytes,omitempty"`
	CapacityVerdict             string                       `json:"capacity_verdict,omitempty"`
	CapacityEstimateEvidenceURI string                       `json:"capacity_estimate_evidence_uri,omitempty"`
	RoutePlanID                 string                       `json:"route_plan_id,omitempty"`
	SelectedModel               string                       `json:"selected_model,omitempty"`
	Locality                    string                       `json:"locality,omitempty"`
	Profile                     string                       `json:"profile"`
	CandidateEligibility        []ReplayCandidateEligibility `json:"candidate_eligibility,omitempty"`
	LatencyMS                   int64                        `json:"latency_ms"`
	LatencyKind                 string                       `json:"latency_kind"`
	Valid                       *bool                        `json:"valid,omitempty"`
	Error                       string                       `json:"error,omitempty"`
}

// ReplayCandidateEligibility is a redacted snapshot of the generic router's
// candidate decision. It keeps role/locality and stable rejection codes while
// omitting input, prompts, and provider-specific metadata.
type ReplayCandidateEligibility struct {
	Provider        string `json:"provider"`
	Role            string `json:"role"`
	Locality        string `json:"locality"`
	Selected        bool   `json:"selected"`
	Fallback        bool   `json:"fallback_eligible"`
	RejectionCode   string `json:"rejection_code,omitempty"`
	CapacityVerdict string `json:"capacity_verdict,omitempty"`
}

// ReadReplayCases reads a bounded JSONL file. It rejects missing provenance,
// duplicate IDs, empty caller inputs and oversized files before any inference
// can be dispatched.
func ReadReplayCases(r io.Reader) ([]ReplayCase, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxReplayInputBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read replay input: %w", err)
	}
	if len(data) > maxReplayInputBytes {
		return nil, fmt.Errorf("replay input exceeds %d bytes", maxReplayInputBytes)
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	seen := map[string]struct{}{}
	cases := make([]ReplayCase, 0, 32)
	for scanner.Scan() {
		if err := appendReplayLine(scanner.Text(), &cases, seen); err != nil {
			return nil, err
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read replay input: %w", err)
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("replay input contains no cases")
	}
	return cases, nil
}

func appendReplayLine(raw string, cases *[]ReplayCase, seen map[string]struct{}) error {
	line := strings.TrimSpace(raw)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil
	}
	if len(*cases) >= maxReplayCases {
		return fmt.Errorf("replay input exceeds %d cases", maxReplayCases)
	}
	number := len(*cases) + 1
	c, err := decodeReplayCase(line, number)
	if err != nil {
		return err
	}
	if err := validateReplayCase(c, seen, number); err != nil {
		return err
	}
	seen[c.CaseID] = struct{}{}
	*cases = append(*cases, c)
	return nil
}

func decodeReplayCase(line string, number int) (ReplayCase, error) {
	var c ReplayCase
	dec := json.NewDecoder(strings.NewReader(line))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return ReplayCase{}, fmt.Errorf("replay case line %d: %w", number, err)
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return ReplayCase{}, fmt.Errorf("replay case line %d contains more than one JSON value", number)
		}
		return ReplayCase{}, fmt.Errorf("replay case line %d: %w", number, err)
	}
	c.CaseID, c.SourceRunID, c.SourceEventID = strings.TrimSpace(c.CaseID), strings.TrimSpace(c.SourceRunID), strings.TrimSpace(c.SourceEventID)
	c.Role, c.TaskRole = strings.TrimSpace(c.Role), strings.TrimSpace(c.TaskRole)
	if c.TaskRole == "" {
		c.TaskRole = c.Role
	}
	if c.Profile == "" {
		c.Profile = "local-only"
	}
	return c, nil
}

func validateReplayCase(c ReplayCase, seen map[string]struct{}, number int) error {
	if err := validateReplayCaseIdentity(c, number); err != nil {
		return err
	}
	if err := validateReplayCaseOptions(c); err != nil {
		return err
	}
	if _, exists := seen[c.CaseID]; exists {
		return fmt.Errorf("duplicate replay case_id %q", c.CaseID)
	}
	return nil
}

func validateReplayCaseIdentity(c ReplayCase, number int) error {
	if err := validateReplayRequiredFields(c, number); err != nil {
		return err
	}
	if err := validateReplaySourceIDs(c); err != nil {
		return err
	}
	if len(c.InputText) > 64<<10 {
		return fmt.Errorf("replay case %q input_text exceeds 64 KiB", c.CaseID)
	}
	return nil
}

func validateReplayRequiredFields(c ReplayCase, number int) error {
	if c.CaseID == "" || c.SourceRunID == "" || c.SourceEventID == "" || c.SourceSequence < 0 || c.Role == "" || strings.TrimSpace(c.Scenario) == "" || strings.TrimSpace(c.InputText) == "" {
		return fmt.Errorf("replay case line %d requires case_id, source_run_id, source_event_id, nonnegative source_sequence, role, scenario, input_text and privacy_provenance", number)
	}
	if strings.TrimSpace(c.PrivacyProvenance) == "" {
		return fmt.Errorf("replay case %q requires explicit privacy_provenance", c.CaseID)
	}
	return nil
}

func validateReplaySourceIDs(c ReplayCase) error {
	if !replayUUIDPattern.MatchString(c.SourceRunID) {
		return fmt.Errorf("replay case %q source_run_id must be a UUID", c.CaseID)
	}
	if !replayUUIDPattern.MatchString(c.SourceEventID) {
		return fmt.Errorf("replay case %q source_event_id must be a UUID", c.CaseID)
	}
	return nil
}

func validateReplayCaseOptions(c ReplayCase) error {
	profile := replayProfile(c.Profile)
	if profile != sharedv1.Profile_PROFILE_LOCAL_ONLY && profile != sharedv1.Profile_PROFILE_REMOTE_ONLY {
		return fmt.Errorf("replay case %q profile must be local-only or remote-only", c.CaseID)
	}
	if replayPrivacy(c.PrivacyClass) == sharedv1.PrivacyClass_PRIVACY_CLASS_UNSPECIFIED {
		return fmt.Errorf("replay case %q has invalid privacy_class %q", c.CaseID, c.PrivacyClass)
	}
	if c.TimeoutMS < 0 || c.TimeoutMS > 300000 {
		return fmt.Errorf("replay case %q timeout_ms must be between 0 and 300000", c.CaseID)
	}
	return nil
}
