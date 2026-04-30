package actions

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"prompt-manager/store"
)

type ActionStore interface {
	List(ctx context.Context) ([]store.Action, error)
	Get(ctx context.Context, id string) (*store.Action, error)
	Create(ctx context.Context, pack string, action *store.Action) error
	Update(ctx context.Context, id string, action *store.Action) error
	Archive(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
}

type Service struct {
	store    ActionStore
	resolver ControlledCommandResolver
}

func NewService(store ActionStore, resolver ControlledCommandResolver) *Service {
	return &Service{store: store, resolver: resolver}
}

func (s *Service) List(ctx context.Context, filters ListFilters) ([]store.Action, error) {
	actions, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	filtered := make([]store.Action, 0, len(actions))
	for _, action := range actions {
		if filters.Pack != "" && action.Pack != filters.Pack {
			continue
		}
		if filters.Status != "" && action.Status != filters.Status {
			continue
		}
		if filters.Owner != "" && action.Owner.Type+":"+action.Owner.ID != filters.Owner && action.Owner.ID != filters.Owner {
			continue
		}
		if filters.Tag != "" && !slices.Contains(action.Tags, filters.Tag) {
			continue
		}
		filtered = append(filtered, action)
	}
	return filtered, nil
}

func (s *Service) Get(ctx context.Context, id string) (*store.Action, error) {
	return s.store.Get(ctx, id)
}

func (s *Service) Create(ctx context.Context, pack string, action *store.Action) (*store.Action, ValidationResponse, error) {
	if pack == "" {
		pack = "drafts"
	}
	validation := s.Validate(ctx, action)
	if !validation.Valid {
		return nil, validation, nil
	}
	if err := s.store.Create(ctx, pack, action); err != nil {
		return nil, validation, err
	}
	created, err := s.store.Get(ctx, action.ID)
	if err != nil {
		return nil, validation, err
	}
	validation = s.Validate(ctx, created)
	validation.Action = created
	return created, validation, nil
}

func (s *Service) Update(ctx context.Context, id string, action *store.Action) (*store.Action, ValidationResponse, error) {
	if action.ID == "" {
		action.ID = id
	}
	if action.ID != id {
		validation := ValidationResponse{ActionID: id, Valid: false}
		validation.Checks = append(validation.Checks, Check{Code: "id", Status: CheckFailed, Path: "id", Message: "action ID cannot be changed"})
		return nil, validation, nil
	}
	current, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, ValidationResponse{}, err
	}
	candidate := *current
	applyMutation(&candidate, action)
	validation := s.Validate(ctx, &candidate)
	if !validation.Valid {
		return nil, validation, nil
	}
	if err := s.store.Update(ctx, id, action); err != nil {
		return nil, validation, err
	}
	updated, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, validation, err
	}
	validation = s.Validate(ctx, updated)
	validation.Action = updated
	return updated, validation, nil
}

func (s *Service) Archive(ctx context.Context, id string) error {
	return s.store.Archive(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

func (s *Service) ValidateByID(ctx context.Context, id string) (ValidationResponse, error) {
	action, err := s.store.Get(ctx, id)
	if err != nil {
		return ValidationResponse{}, err
	}
	result := s.Validate(ctx, action)
	result.Action = action
	return result, nil
}

func (s *Service) Validate(ctx context.Context, action *store.Action) ValidationResponse {
	result := ValidationResponse{}
	if action != nil {
		result.ActionID = action.ID
		result.Status = action.Status
	}
	add := func(code string, status CheckStatus, path, message string) {
		result.Checks = append(result.Checks, Check{Code: code, Status: status, Path: path, Message: message})
	}

	if err := store.ValidateActionContract(action); err != nil {
		add("schema", CheckFailed, "", err.Error())
		result.Valid = false
		return result
	}
	add("schema", CheckPassed, "", "action contract is structurally valid")

	if action.Status == store.StatusArchived {
		add("status", CheckWarning, "status", "archived actions are discoverable but not runnable")
	} else {
		add("status", CheckPassed, "status", "action status allows validation")
	}

	if s.resolver == nil {
		add("command_ownership", CheckFailed, "command.argv", "controlled-command resolver is not configured")
	} else {
		resolution, err := s.resolver.ResolveCommand(ctx, action.Command.Argv)
		result.Command = &resolution
		if err != nil {
			add("command_ownership", CheckFailed, "command.argv", err.Error())
		} else {
			switch resolution.Certainty {
			case CertaintyOperation, CertaintyCommand:
				add("command_ownership", CheckPassed, "command.argv", resolution.Message)
				checkPermissionAlignment(action, resolution, add)
			case CertaintyOwnerOnly:
				if action.Status == store.StatusDraft {
					add("command_ownership", CheckWarning, "command.argv", resolution.Message)
				} else {
					add("command_ownership", CheckFailed, "command.argv", "active actions require cataloged command certainty; "+resolution.Message)
				}
			default:
				add("command_ownership", CheckFailed, "command.argv", resolution.Message)
			}
		}
	}

	checkPlaceholders(action, add)
	checkInputDefaults(action, add)
	checkOutputPermissions(action, add)
	checkValidationHook(ctx, action, s.resolver, add)

	result.Valid = true
	for _, check := range result.Checks {
		if check.Status == CheckFailed {
			result.Valid = false
			break
		}
	}
	result.Runnable = result.Valid && action.Status == store.StatusActive && result.Command != nil && (result.Command.Certainty == CertaintyCommand || result.Command.Certainty == CertaintyOperation)
	return result
}

func checkPermissionAlignment(action *store.Action, resolution CommandResolution, add func(string, CheckStatus, string, string)) {
	declared := permissionsToSet(action.Permissions)
	missing := []string{}
	for _, permission := range resolution.Permissions {
		switch permission {
		case "filesystem:read":
			if !declared["filesystemRead"] && !declared["filesystemWrite"] {
				missing = append(missing, permission)
			}
		case "filesystem:write":
			if !declared["filesystemWrite"] {
				missing = append(missing, permission)
			}
		case "network:localhost":
			if !declared["localhostNetwork"] {
				missing = append(missing, permission)
			}
		case "network:external":
			if !declared["externalNetwork"] {
				missing = append(missing, permission)
			}
		case "api:read":
			if !declared["apiRead"] && !declared["apiWrite"] {
				missing = append(missing, permission)
			}
		case "api:write":
			if !declared["apiWrite"] {
				missing = append(missing, permission)
			}
		case "process:start":
			if !declared["processStart"] {
				missing = append(missing, permission)
			}
		case "process:stop":
			if !declared["processStop"] {
				missing = append(missing, permission)
			}
		case "host:configure":
			if !declared["hostConfigure"] {
				missing = append(missing, permission)
			}
		case "secret:read":
			if !declared["secretRead"] && !declared["secretWrite"] {
				missing = append(missing, permission)
			}
		case "secret:write":
			if !declared["secretWrite"] {
				missing = append(missing, permission)
			}
		}
	}
	if resolution.Effect == EffectDestructive && !action.Permissions.Destructive {
		missing = append(missing, "destructive")
	}
	if len(missing) > 0 {
		add("permissions", CheckFailed, "permissions", "missing permission declarations: "+strings.Join(missing, ", "))
		return
	}
	add("permissions", CheckPassed, "permissions", "permission declarations cover the resolved command")
}

func checkPlaceholders(action *store.Action, add func(string, CheckStatus, string, string)) {
	used := map[string]bool{}
	for index, token := range action.Command.Argv {
		for _, match := range regexp.MustCompile(`\{\{([a-z][a-zA-Z0-9]*)\}\}`).FindAllStringSubmatch(token, -1) {
			if token != match[0] {
				add("placeholders", CheckFailed, fmt.Sprintf("command.argv[%d]", index), "placeholders must occupy a whole argv token")
				continue
			}
			name := match[1]
			if _, ok := action.Inputs[name]; !ok {
				add("placeholders", CheckFailed, fmt.Sprintf("command.argv[%d]", index), "placeholder references undeclared input: "+name)
				continue
			}
			used[name] = true
		}
	}
	unusedRequired := []string{}
	for name, input := range action.Inputs {
		if input.Required && !used[name] && input.Default == nil {
			unusedRequired = append(unusedRequired, name)
		}
	}
	if len(unusedRequired) > 0 {
		slices.Sort(unusedRequired)
		add("placeholders", CheckFailed, "inputs", "required inputs are not used by command and have no default: "+strings.Join(unusedRequired, ", "))
		return
	}
	add("placeholders", CheckPassed, "command.argv", "command placeholders match declared inputs")
}

func checkInputDefaults(action *store.Action, add func(string, CheckStatus, string, string)) {
	for name, input := range action.Inputs {
		if input.Default == nil {
			continue
		}
		if err := validateDefaultValue(input); err != nil {
			add("input_defaults", CheckFailed, "inputs."+name+".default", err.Error())
			return
		}
	}
	add("input_defaults", CheckPassed, "inputs", "input defaults are valid")
}

func validateDefaultValue(input store.ActionInput) error {
	switch input.Type {
	case "string", "file", "path", "scenario", "team", "action":
		value, ok := input.Default.(string)
		if !ok {
			return fmt.Errorf("default must be a string for %s inputs", input.Type)
		}
		if input.MaxLength != nil && len(value) > *input.MaxLength {
			return fmt.Errorf("default exceeds maxLength")
		}
		if !input.AllowMultiline && strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("default must be single-line")
		}
		if len(input.Enum) > 0 && !slices.Contains(input.Enum, value) {
			return fmt.Errorf("default is not in enum")
		}
		if input.Pattern != "" {
			matched, err := regexp.MatchString(input.Pattern, value)
			if err != nil {
				return err
			}
			if !matched {
				return fmt.Errorf("default does not match pattern")
			}
		}
	case "number":
		value, ok := numericDefault(input.Default)
		if !ok {
			return fmt.Errorf("default must be numeric")
		}
		if input.Min != nil && value < *input.Min {
			return fmt.Errorf("default is less than min")
		}
		if input.Max != nil && value > *input.Max {
			return fmt.Errorf("default is greater than max")
		}
	case "integer":
		value, ok := numericDefault(input.Default)
		if !ok || value != float64(int64(value)) {
			return fmt.Errorf("default must be an integer")
		}
		if input.Min != nil && value < *input.Min {
			return fmt.Errorf("default is less than min")
		}
		if input.Max != nil && value > *input.Max {
			return fmt.Errorf("default is greater than max")
		}
	case "boolean":
		if _, ok := input.Default.(bool); !ok {
			return fmt.Errorf("default must be boolean")
		}
	}
	return nil
}

func numericDefault(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case jsonNumber:
		value, err := typed.Float64()
		return value, err == nil
	default:
		return 0, false
	}
}

type jsonNumber interface {
	Float64() (float64, error)
}

func checkOutputPermissions(action *store.Action, add func(string, CheckStatus, string, string)) {
	for name, output := range action.Outputs {
		if (output.Type == "file" || output.Type == "path") && !action.Permissions.FilesystemWrite && !action.Permissions.FilesystemRead {
			add("output_permissions", CheckFailed, "outputs."+name, "file/path outputs require filesystem permission declaration")
			return
		}
	}
	add("output_permissions", CheckPassed, "outputs", "output declarations are compatible with permissions")
}

func checkValidationHook(ctx context.Context, action *store.Action, resolver ControlledCommandResolver, add func(string, CheckStatus, string, string)) {
	if action.Validation == nil || len(action.Validation.Argv) == 0 {
		add("validation_hook", CheckPassed, "validation", "no owner-specific validation hook declared")
		return
	}
	argv := action.Validation.Argv
	if err := store.ValidateActionArgv(argv); err != nil {
		add("validation_hook", CheckFailed, "validation.argv", err.Error())
		return
	}
	if isSelfRecursiveValidationHook(argv, action.ID) {
		add("validation_hook", CheckFailed, "validation.argv", "validation hook must not recursively validate the same action")
		return
	}
	if resolver == nil {
		add("validation_hook", CheckFailed, "validation.argv", "controlled-command resolver is not configured for validation hook")
		return
	}
	resolution, err := resolver.ResolveCommand(ctx, argv)
	if err != nil {
		add("validation_hook", CheckFailed, "validation.argv", err.Error())
		return
	}
	switch resolution.Certainty {
	case CertaintyOperation, CertaintyCommand:
		if resolution.Effect == EffectDestructive || resolution.Effect == EffectAdmin {
			add("validation_hook", CheckFailed, "validation.argv", "validation hook must not be destructive or admin-level")
			return
		}
		add("validation_hook", CheckPassed, "validation.argv", "validation hook is controlled and non-recursive")
	case CertaintyOwnerOnly:
		if action.Status == store.StatusDraft {
			add("validation_hook", CheckWarning, "validation.argv", "validation hook owner is known, but command path is not yet cataloged")
			return
		}
		add("validation_hook", CheckFailed, "validation.argv", "active actions require cataloged validation hook command certainty")
	default:
		add("validation_hook", CheckFailed, "validation.argv", resolution.Message)
	}
}

func isSelfRecursiveValidationHook(argv []string, actionID string) bool {
	if len(argv) < 4 || argv[0] != "prompt-manager" {
		return false
	}
	if argv[1] != "action" && argv[1] != "actions" {
		return false
	}
	if argv[2] != "validate" {
		return false
	}
	return argv[3] == actionID
}

func permissionsToSet(permissions store.ActionPermissions) map[string]bool {
	return map[string]bool{
		"filesystemRead":   permissions.FilesystemRead,
		"filesystemWrite":  permissions.FilesystemWrite,
		"localhostNetwork": permissions.LocalhostNetwork,
		"externalNetwork":  permissions.ExternalNetwork,
		"apiRead":          permissions.APIRead,
		"apiWrite":         permissions.APIWrite,
		"processStart":     permissions.ProcessStart,
		"processStop":      permissions.ProcessStop,
		"hostConfigure":    permissions.HostConfigure,
		"secretRead":       permissions.SecretRead,
		"secretWrite":      permissions.SecretWrite,
		"destructive":      permissions.Destructive,
	}
}

func applyMutation(action, updates *store.Action) {
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
	if updates.Permissions != (store.ActionPermissions{}) {
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
