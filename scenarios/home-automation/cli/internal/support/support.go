package support

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func NewFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func ParseFlags(fs *flag.FlagSet, args []string) error {
	return cliutil.ParseInterspersed(fs, args)
}

func Decode(body []byte, dest interface{}) error {
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	return nil
}

func BuildQuery(params map[string]string) url.Values {
	values := url.Values{}
	for key, value := range params {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		values.Set(key, value)
	}
	return values
}

func FormatMapInline(values map[string]interface{}) string {
	if len(values) == 0 {
		return "(none)"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", key, values[key]))
	}
	return strings.Join(parts, ", ")
}

func BoolStatus(ok bool, okText, failText string) string {
	if ok {
		return okText
	}
	return failText
}

func PrintJSONReport(enabled bool, report interface{}) error {
	if !enabled {
		return nil
	}
	return cliapp.PrintReportJSON(os.Stdout, report)
}

type DeviceListResponse struct {
	Devices    []DeviceStatus `json:"devices"`
	Count      int            `json:"count"`
	DataSource string         `json:"data_source"`
	MockData   bool           `json:"mock_data"`
}

type DeviceStatus struct {
	DeviceID    string                 `json:"device_id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	State       map[string]interface{} `json:"state"`
	Available   bool                   `json:"available"`
	LastUpdated string                 `json:"last_updated"`
	Attributes  map[string]interface{} `json:"attributes"`
}

type DeviceControlResponse struct {
	Success         bool                   `json:"success"`
	DeviceID        string                 `json:"device_id"`
	Action          string                 `json:"action"`
	DeviceState     map[string]interface{} `json:"device_state"`
	Message         string                 `json:"message"`
	Error           string                 `json:"error"`
	RequestID       string                 `json:"request_id"`
	Timestamp       string                 `json:"timestamp"`
	ExecutionTimeMS int                    `json:"execution_time_ms"`
}

type AutomationListResponse struct {
	Automations []Automation `json:"automations"`
	Total       int          `json:"total"`
}

type Automation struct {
	ID             string                   `json:"id"`
	Name           string                   `json:"name"`
	Description    string                   `json:"description"`
	CreatedBy      string                   `json:"created_by"`
	TriggerType    string                   `json:"trigger_type"`
	TriggerConfig  map[string]interface{}   `json:"trigger_config"`
	Conditions     []map[string]interface{} `json:"conditions"`
	Actions        []map[string]interface{} `json:"actions"`
	Active         bool                     `json:"active"`
	GeneratedByAI  bool                     `json:"generated_by_ai"`
	ExecutionCount int                      `json:"execution_count"`
	LastExecuted   string                   `json:"last_executed"`
	CreatedAt      string                   `json:"created_at"`
	UpdatedAt      string                   `json:"updated_at"`
	SourceCode     string                   `json:"source_code"`
}

type AutomationGenerationResponse struct {
	AutomationID          string                       `json:"automation_id"`
	GeneratedCode         string                       `json:"generated_code"`
	Explanation           string                       `json:"explanation"`
	EstimatedEnergyImpact string                       `json:"estimated_energy_impact"`
	Conflicts             []string                     `json:"conflicts"`
	Validation            AutomationValidationResponse `json:"validation"`
	ReadyToDeploy         bool                         `json:"ready_to_deploy"`
}

type AutomationValidationResponse struct {
	AutomationID         string               `json:"automation_id"`
	ValidationPassed     bool                 `json:"validation_passed"`
	SecurityValidation   SecurityValidation   `json:"security_validation"`
	PermissionValidation PermissionValidation `json:"permission_validation"`
	LogicValidation      LogicValidation      `json:"logic_validation"`
	OverallRiskLevel     string               `json:"overall_risk_level"`
	Recommendations      []string             `json:"recommendations"`
	ValidationTimestamp  string               `json:"validation_timestamp"`
}

type SecurityValidation struct {
	Passed         bool            `json:"passed"`
	SecurityIssues []ValidationMsg `json:"security_issues"`
	Warnings       []ValidationMsg `json:"warnings"`
	RiskLevel      string          `json:"risk_level"`
}

type PermissionValidation struct {
	Passed           bool            `json:"passed"`
	PermissionIssues []ValidationMsg `json:"permission_issues"`
	UserPermissions  UserPermissions `json:"user_permissions"`
}

type LogicValidation struct {
	Passed              bool            `json:"passed"`
	LogicIssues         []ValidationMsg `json:"logic_issues"`
	Suggestions         []ValidationMsg `json:"suggestions"`
	DeviceCompatibility string          `json:"device_compatibility"`
	ScheduleSafety      string          `json:"schedule_safety"`
}

type ValidationMsg struct {
	Type     string `json:"type"`
	Pattern  string `json:"pattern"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	DeviceID string `json:"device_id"`
	Action   string `json:"action"`
}

type UserPermissions struct {
	CanCreate       bool `json:"can_create"`
	AllowedDevices  int  `json:"allowed_devices"`
	AutomationCount int  `json:"automation_count"`
	AutomationLimit int  `json:"automation_limit"`
}

type SafetyStatus struct {
	AutomationID   string                       `json:"automation_id"`
	CurrentStatus  string                       `json:"current_status"`
	RiskLevel      string                       `json:"risk_level"`
	LastValidated  string                       `json:"last_validated"`
	ValidationInfo AutomationValidationResponse `json:"validation_info"`
}

type CurrentContext struct {
	ContextName   string                 `json:"context_name"`
	ActiveSince   string                 `json:"active_since"`
	TriggeredBy   string                 `json:"triggered_by"`
	Configuration ContextConfig          `json:"configuration"`
	ActiveDevices map[string]interface{} `json:"active_devices"`
}

type ContextConfig struct {
	SceneID             string                            `json:"scene_id"`
	AutomationOverrides map[string]map[string]interface{} `json:"automation_overrides"`
	Description         string                            `json:"description"`
}

type CalendarEventResponse struct {
	Success             bool           `json:"success"`
	EventID             string         `json:"event_id"`
	DetectedContext     string         `json:"detected_context"`
	ContextActivated    bool           `json:"context_activated"`
	DeviceChanges       []DeviceChange `json:"device_changes"`
	Message             string         `json:"message"`
	ProcessingTimestamp string         `json:"processing_timestamp"`
}

type DeviceChange struct {
	DeviceID   string                 `json:"device_id"`
	Action     string                 `json:"action"`
	Parameters map[string]interface{} `json:"parameters"`
	Success    bool                   `json:"success"`
	Message    string                 `json:"message"`
}

type ProfilesResponse struct {
	Profiles []Profile `json:"profiles"`
	Count    int       `json:"count"`
}

type Profile struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Permissions map[string]interface{} `json:"permissions"`
	CreatedAt   string                 `json:"created_at"`
}

type HomeAssistantConfigResponse struct {
	BaseURL              string `json:"base_url"`
	TokenConfigured      bool   `json:"token_configured"`
	MockMode             bool   `json:"mock_mode"`
	Status               string `json:"status"`
	TokenType            string `json:"token_type"`
	AccessTokenExpiresAt string `json:"access_token_expires_at"`
	Message              string `json:"message"`
	Error                string `json:"error"`
	ActionRequired       string `json:"action_required"`
	Target               string `json:"target"`
	StatusCheckedAt      string `json:"status_checked_at"`
	LastCheckedAt        string `json:"last_checked_at"`
	UpdatedAt            string `json:"updated_at"`
	UpdatedBy            string `json:"updated_by"`
	Saved                bool   `json:"saved"`
	AutoProvisioned      bool   `json:"auto_provisioned"`
}

func TimestampOrUnknown(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return value
}

func HumanTime(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return parsed.UTC().Format(time.RFC3339)
}
