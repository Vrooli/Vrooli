package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	maxActionIDLength          = 96
	maxActionRunHistoryEntries = 100
)

var (
	actionIDRegex          = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*$`)
	actionPlaceholderRegex = regexp.MustCompile(`^\{\{([a-z][a-zA-Z0-9]*)\}\}$`)
	actionUnsafeTokenChars = regexp.MustCompile(`[|&;<>` + "`" + `]|\$\(`)
)

var actionRejectedExecutables = map[string]bool{
	"bash":    true,
	"cat":     true,
	"curl":    true,
	"docker":  true,
	"fish":    true,
	"git":     true,
	"grep":    true,
	"node":    true,
	"psql":    true,
	"python":  true,
	"python3": true,
	"rg":      true,
	"sh":      true,
	"zsh":     true,
}

// IsValidActionID validates Action IDs, including dotted namespaces.
func IsValidActionID(id string) bool {
	if len(id) == 0 || len(id) > maxActionIDLength {
		return false
	}
	return actionIDRegex.MatchString(id)
}

// ValidateActionContract performs store-level structural validation.
//
// This intentionally does not certify command ownership or run eligibility; that
// belongs to the Action validation service once the controlled-command resolver
// exists. Store validation rejects malformed and obviously unsafe contracts
// before persistence.
func ValidateActionContract(action *Action) error {
	if action == nil {
		return fmt.Errorf("action is required")
	}
	if !IsValidActionID(action.ID) {
		return fmt.Errorf("invalid action ID format: %s", action.ID)
	}
	if action.Kind != "" && action.Kind != KindAction {
		return fmt.Errorf("invalid action kind: %s", action.Kind)
	}
	if action.SchemaVersion != 0 && action.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("unsupported action schema version: %d", action.SchemaVersion)
	}
	if strings.TrimSpace(action.Name) == "" {
		return fmt.Errorf("action name is required")
	}
	if action.Status != "" && action.Status != StatusActive && action.Status != StatusDraft && action.Status != StatusArchived {
		return fmt.Errorf("invalid action status: %s", action.Status)
	}
	if strings.TrimSpace(action.Owner.Type) == "" || strings.TrimSpace(action.Owner.ID) == "" {
		return fmt.Errorf("action owner type and id are required")
	}
	switch action.Owner.Type {
	case "project", "scenario", "resource", "team":
	default:
		return fmt.Errorf("invalid action owner type: %s", action.Owner.Type)
	}
	if err := ValidateActionArgv(action.Command.Argv); err != nil {
		return err
	}
	if err := validateActionInputs(action.Inputs); err != nil {
		return err
	}
	if err := validateActionOutputs(action.Outputs); err != nil {
		return err
	}
	if err := validateActionRuntimeMetadata(action); err != nil {
		return err
	}
	return validateActionPlaceholders(action)
}

// ValidateActionArgv performs structural safety validation for an argv-shaped
// Action command. It does not certify command ownership or run eligibility.
func ValidateActionArgv(argv []string) error {
	if len(argv) == 0 {
		return fmt.Errorf("action command argv is required")
	}
	for i, token := range argv {
		if token == "" {
			return fmt.Errorf("action command argv[%d] is empty", i)
		}
		if strings.ContainsAny(token, "\r\n") {
			return fmt.Errorf("action command argv[%d] must be single-line", i)
		}
		if i == 0 {
			if actionPlaceholderRegex.MatchString(token) {
				return fmt.Errorf("action command executable must be static")
			}
			if strings.ContainsAny(token, `/\`) {
				return fmt.Errorf("action command executable must not contain path separators")
			}
			if actionRejectedExecutables[token] {
				return fmt.Errorf("action command executable is not Vrooli-controlled: %s", token)
			}
		}
		if actionPlaceholderRegex.MatchString(token) {
			continue
		}
		if actionUnsafeTokenChars.MatchString(token) {
			return fmt.Errorf("action command argv[%d] contains shell syntax", i)
		}
	}
	return nil
}

func validateActionInputs(inputs map[string]ActionInput) error {
	for name, input := range inputs {
		if !isValidActionInputName(name) {
			return fmt.Errorf("invalid action input name: %s", name)
		}
		switch input.Type {
		case "string", "number", "integer", "boolean", "file", "path", "scenario", "team", "action":
		default:
			return fmt.Errorf("invalid action input type for %s: %s", name, input.Type)
		}
		if len(input.Enum) > 0 && input.Type != "string" {
			return fmt.Errorf("action input %s enum is only supported for string inputs", name)
		}
		if input.Pattern != "" {
			if _, err := regexp.Compile(input.Pattern); err != nil {
				return fmt.Errorf("invalid action input pattern for %s: %w", name, err)
			}
		}
	}
	return nil
}

func validateActionOutputs(outputs map[string]ActionOutput) error {
	for name, output := range outputs {
		if !isValidActionInputName(name) {
			return fmt.Errorf("invalid action output name: %s", name)
		}
		switch output.Type {
		case "string", "number", "integer", "boolean", "file", "path", "json":
		default:
			return fmt.Errorf("invalid action output type for %s: %s", name, output.Type)
		}
	}
	return nil
}

func validateActionRuntimeMetadata(action *Action) error {
	if action.Execution != nil {
		if action.Execution.OutputMode != "" && action.Execution.OutputMode != "stdout" && action.Execution.OutputMode != "json" {
			return fmt.Errorf("invalid action execution outputMode: %s", action.Execution.OutputMode)
		}
	}
	if action.Validation != nil {
		if action.Validation.Mode != "" && action.Validation.Mode != "contract" && action.Validation.Mode != "owner" {
			return fmt.Errorf("invalid action validation mode: %s", action.Validation.Mode)
		}
		if len(action.Validation.Argv) > 0 {
			if err := ValidateActionArgv(action.Validation.Argv); err != nil {
				return fmt.Errorf("invalid action validation argv: %w", err)
			}
		}
	}
	return nil
}

func validateActionPlaceholders(action *Action) error {
	seen := map[string]bool{}
	for _, token := range action.Command.Argv {
		match := actionPlaceholderRegex.FindStringSubmatch(token)
		if len(match) != 2 {
			continue
		}
		name := match[1]
		if _, ok := action.Inputs[name]; !ok {
			return fmt.Errorf("action command references undeclared input: %s", name)
		}
		seen[name] = true
	}
	for name, input := range action.Inputs {
		if input.Required && !seen[name] && input.Default == nil {
			return fmt.Errorf("required action input is not used by command and has no default: %s", name)
		}
	}
	return nil
}

func isValidActionInputName(name string) bool {
	if len(name) == 0 || len(name) > 64 {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if r < 'a' || r > 'z' {
				return false
			}
			continue
		}
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

// FileActionStore implements ActionStore using the file system.
type FileActionStore struct {
	configDir string
}

// NewFileActionStore creates a new file-based Action store.
func NewFileActionStore(configDir string) *FileActionStore {
	return &FileActionStore{configDir: configDir}
}

func (s *FileActionStore) actionsDir() string {
	return filepath.Join(s.configDir, "actions")
}

func (s *FileActionStore) packsDir() string {
	return filepath.Join(s.actionsDir(), "packs")
}

func (s *FileActionStore) getPackOrder() (*PackOrder, error) {
	path := filepath.Join(s.actionsDir(), "_pack-order.json")
	return LoadJSON[PackOrder](path)
}

func (s *FileActionStore) getActivePacks() ([]string, error) {
	order, err := s.getPackOrder()
	if err != nil {
		return ListDirectories(s.packsDir())
	}
	return order.ActivePacks, nil
}

// List returns all Actions from active packs.
func (s *FileActionStore) List(ctx context.Context) ([]Action, error) {
	packs, err := s.getActivePacks()
	if err != nil {
		return nil, fmt.Errorf("getting active packs: %w", err)
	}

	seen := make(map[string]bool)
	var actions []Action
	for _, pack := range packs {
		actionDirs, err := ListDirectories(filepath.Join(s.packsDir(), pack))
		if err != nil {
			continue
		}
		for _, actionID := range actionDirs {
			if seen[actionID] {
				continue
			}
			action, err := s.loadAction(pack, actionID)
			if err != nil {
				continue
			}
			if err := ValidateActionContract(action); err != nil {
				continue
			}
			action.Pack = pack
			actions = append(actions, *action)
			seen[actionID] = true
		}
	}
	return actions, nil
}

// Get retrieves an Action by ID, searching through active packs.
func (s *FileActionStore) Get(ctx context.Context, id string) (*Action, error) {
	if !IsValidActionID(id) {
		return nil, fmt.Errorf("invalid action ID format: %s", id)
	}
	packs, err := s.getActivePacks()
	if err != nil {
		return nil, fmt.Errorf("getting active packs: %w", err)
	}
	for _, pack := range packs {
		action, err := s.loadAction(pack, id)
		if err != nil {
			continue
		}
		if err := ValidateActionContract(action); err != nil {
			continue
		}
		action.Pack = pack
		return action, nil
	}
	return nil, fmt.Errorf("action not found: %s", id)
}

// Create creates a new Action in the specified pack.
func (s *FileActionStore) Create(ctx context.Context, pack string, action *Action) error {
	if err := s.validatePack(pack); err != nil {
		return err
	}
	if err := ValidateActionContract(action); err != nil {
		return err
	}
	if _, err := s.Get(ctx, action.ID); err == nil {
		return fmt.Errorf("action already exists: %s", action.ID)
	}

	action.Kind = KindAction
	action.SchemaVersion = CurrentSchemaVersion
	if action.Status == "" {
		action.Status = StatusDraft
	}
	action.Timestamps = NewTimestamps()
	action.Pack = pack

	actionDir := filepath.Join(s.packsDir(), pack, action.ID)
	if err := os.MkdirAll(actionDir, 0o755); err != nil {
		return fmt.Errorf("creating action directory: %w", err)
	}
	if err := SaveJSON(filepath.Join(actionDir, "action.json"), action); err != nil {
		return fmt.Errorf("writing action.json: %w", err)
	}
	return s.appendHistory(actionDir, action.Revision, "created", "Initial version")
}

// Update updates an existing Action.
func (s *FileActionStore) Update(ctx context.Context, id string, updates *Action) error {
	action, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if updates.ID != "" && updates.ID != id {
		return fmt.Errorf("action ID cannot be changed")
	}

	applyActionUpdates(action, updates)
	if err := ValidateActionContract(action); err != nil {
		return err
	}
	action.UpdateTimestamp()

	actionDir := filepath.Join(s.packsDir(), action.Pack, id)
	if err := SaveJSON(filepath.Join(actionDir, "action.json"), action); err != nil {
		return fmt.Errorf("writing action.json: %w", err)
	}
	return s.appendHistory(actionDir, action.Revision, "updated", "Updated action")
}

// Archive marks an Action archived without deleting its files.
func (s *FileActionStore) Archive(ctx context.Context, id string) error {
	return s.Update(ctx, id, &Action{Status: StatusArchived})
}

// Delete removes an Action.
func (s *FileActionStore) Delete(ctx context.Context, id string) error {
	action, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	return DeleteDirectory(filepath.Join(s.packsDir(), action.Pack, id))
}

// AppendRunHistory appends a bounded Action execution audit entry.
func (s *FileActionStore) AppendRunHistory(ctx context.Context, id string, entry ActionRunHistoryEntry) error {
	action, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if entry.ActionID == "" {
		entry.ActionID = id
	}
	actionDir := filepath.Join(s.packsDir(), action.Pack, id)
	path := filepath.Join(actionDir, "runs.jsonl")
	if err := AppendJSONL(path, entry); err != nil {
		return fmt.Errorf("writing action run history: %w", err)
	}
	return trimJSONLLines(path, maxActionRunHistoryEntries)
}

func (s *FileActionStore) validatePack(pack string) error {
	packs, err := s.getActivePacks()
	if err != nil {
		return fmt.Errorf("getting active packs: %w", err)
	}
	for _, p := range packs {
		if p == pack {
			return nil
		}
	}
	return fmt.Errorf("invalid pack: %s", pack)
}

func (s *FileActionStore) loadAction(pack, actionID string) (*Action, error) {
	actionPath := filepath.Join(s.packsDir(), pack, actionID, "action.json")
	return LoadJSON[Action](actionPath)
}

func (s *FileActionStore) appendHistory(actionDir string, revision int, action, summary string) error {
	entry := HistoryEntry{
		Version:   revision,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Action:    action,
		Summary:   summary,
	}
	if err := AppendJSONL(filepath.Join(actionDir, "history.jsonl"), entry); err != nil {
		return fmt.Errorf("writing history: %w", err)
	}
	return nil
}

func trimJSONLLines(path string, maxLines int) error {
	if maxLines <= 0 {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := strings.TrimRight(string(data), "\n")
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	if len(lines) <= maxLines {
		return nil
	}
	lines = lines[len(lines)-maxLines:]
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

func applyActionUpdates(action, updates *Action) {
	if updates.Name != "" {
		action.Name = updates.Name
	}
	if updates.Description != "" {
		action.Description = updates.Description
	}
	if updates.Status != "" {
		action.Status = updates.Status
	}
	if updates.Owner.Type != "" || updates.Owner.ID != "" {
		action.Owner = updates.Owner
	}
	if len(updates.Command.Argv) > 0 {
		action.Command = updates.Command
	}
	if updates.Inputs != nil {
		action.Inputs = updates.Inputs
	}
	if updates.Outputs != nil {
		action.Outputs = updates.Outputs
	}
	if updates.Permissions != (ActionPermissions{}) {
		action.Permissions = updates.Permissions
	}
	if updates.Examples != nil {
		action.Examples = updates.Examples
	}
	if updates.Tags != nil {
		action.Tags = updates.Tags
	}
	if updates.Execution != nil {
		action.Execution = updates.Execution
	}
	if updates.Validation != nil {
		action.Validation = updates.Validation
	}
}
