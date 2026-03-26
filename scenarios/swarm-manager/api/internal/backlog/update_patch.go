package backlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"google.golang.org/protobuf/encoding/protojson"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

const (
	updateFieldTitle           = "title"
	updateFieldDescription     = "description"
	updateFieldStatus          = "status"
	updateFieldPriority        = "priority"
	updateFieldTags            = "tags"
	updateFieldResearchTarget  = "research_target"
	updateFieldDependsOn       = "depends_on"
	updateFieldInitiative      = "initiative"
	updateFieldEffort          = "effort"
	updateFieldAcceptanceAllow = "acceptance_allow"
	updateFieldAcceptanceDeny  = "acceptance_deny"
)

type backlogUpdateFieldSet map[string]struct{}

func (f backlogUpdateFieldSet) Has(field string) bool {
	_, ok := f[field]
	return ok
}

func (f backlogUpdateFieldSet) Empty() bool {
	return len(f) == 0
}

var (
	updateFieldAliasesOnce sync.Once
	updateFieldAliases     map[string]string
)

func getUpdateFieldAliases() map[string]string {
	updateFieldAliasesOnce.Do(func() {
		aliases := make(map[string]string)
		fields := (&apipb.UpdateBacklogItemRequest{}).ProtoReflect().Descriptor().Fields()
		for i := 0; i < fields.Len(); i++ {
			field := fields.Get(i)
			canonical := string(field.Name())
			aliases[canonical] = canonical
			if jsonName := field.JSONName(); jsonName != "" {
				aliases[jsonName] = canonical
			}
		}
		updateFieldAliases = aliases
	})
	return updateFieldAliases
}

func decodeUpdateBacklogPatch(r *http.Request) (*apipb.UpdateBacklogItemRequest, backlogUpdateFieldSet, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read request body: %w", err)
	}
	defer r.Body.Close()

	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil, nil, fmt.Errorf("request body is required")
	}

	fields, err := parseUpdateBacklogFields(body)
	if err != nil {
		return nil, nil, err
	}

	var req apipb.UpdateBacklogItemRequest
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(body, &req); err != nil {
		return nil, nil, err
	}

	normalizeUpdateBacklogPatch(&req, fields)
	return &req, fields, nil
}

func parseUpdateBacklogFields(body []byte) (backlogUpdateFieldSet, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	var raw map[string]json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, fmt.Errorf("request body must be a JSON object")
	}

	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("invalid trailing JSON content")
		}
		return nil, err
	}

	fields := make(backlogUpdateFieldSet, len(raw))
	aliases := getUpdateFieldAliases()
	for key, value := range raw {
		canonical, ok := aliases[key]
		if !ok {
			continue
		}
		if _, exists := fields[canonical]; exists {
			return nil, fmt.Errorf("field %q provided more than once", canonical)
		}
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, fmt.Errorf("%s cannot be null; omit it to leave it unchanged or use an empty value to clear it", canonical)
		}
		fields[canonical] = struct{}{}
	}

	return fields, nil
}

func normalizeUpdateBacklogPatch(req *apipb.UpdateBacklogItemRequest, fields backlogUpdateFieldSet) {
	if fields.Has(updateFieldTitle) && req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		req.Title = &trimmed
	}
	if fields.Has(updateFieldStatus) && req.Status != nil {
		normalized := strings.ToLower(strings.TrimSpace(*req.Status))
		req.Status = &normalized
	}
	if fields.Has(updateFieldResearchTarget) && req.ResearchTarget != nil {
		normalized := strings.ToLower(strings.TrimSpace(*req.ResearchTarget))
		req.ResearchTarget = &normalized
	}
	if fields.Has(updateFieldInitiative) && req.Initiative != nil {
		trimmed := strings.TrimSpace(*req.Initiative)
		req.Initiative = &trimmed
	}
	if fields.Has(updateFieldEffort) && req.Effort != nil {
		normalized := strings.ToUpper(strings.TrimSpace(*req.Effort))
		req.Effort = &normalized
	}
}

func validateUpdateBacklogItemRequest(req *apipb.UpdateBacklogItemRequest, fields backlogUpdateFieldSet, kind BacklogKind) string {
	if fields.Empty() {
		return "at least one field is required"
	}
	if fields.Has(updateFieldTitle) && strings.TrimSpace(req.GetTitle()) == "" {
		return "title is required"
	}
	if fields.Has(updateFieldStatus) {
		status := req.GetStatus()
		if !validateBacklogStatus(status) {
			return "status must be a valid backlog status"
		}
		if status == string(StatusQueued) || status == string(StatusInProgress) {
			return "status 'queued' and 'in_progress' can only be set by the execution system"
		}
	}
	if fields.Has(updateFieldPriority) {
		priority := req.GetPriority()
		if priority < 1 || priority > 10 {
			return "priority must be between 1 and 10"
		}
	}
	if fields.Has(updateFieldTags) {
		if err := validateUniqueStrings("tags", req.Tags); err != nil {
			return err.Error()
		}
	}
	if fields.Has(updateFieldResearchTarget) {
		if kind != KindResearch {
			return "research_target can only be set on research backlog items"
		}
		if _, err := normalizeResearchTarget(req.GetResearchTarget()); err != nil {
			return err.Error()
		}
	}
	if fields.Has(updateFieldEffort) {
		if _, err := validateEffort(req.GetEffort()); err != nil {
			return err.Error()
		}
	}
	if fields.Has(updateFieldAcceptanceAllow) {
		if err := validateGlobs(req.AcceptanceAllow); err != nil {
			return "acceptance_allow: " + err.Error()
		}
	}
	if fields.Has(updateFieldAcceptanceDeny) {
		if err := validateGlobs(req.AcceptanceDeny); err != nil {
			return "acceptance_deny: " + err.Error()
		}
	}

	return ""
}

func applyUpdateBacklogPatch(item *BacklogItem, req *apipb.UpdateBacklogItemRequest, fields backlogUpdateFieldSet) {
	if fields.Has(updateFieldTitle) {
		item.Title = req.GetTitle()
	}
	if fields.Has(updateFieldDescription) {
		item.Description = req.GetDescription()
	}
	if fields.Has(updateFieldStatus) {
		item.Status = BacklogStatus(req.GetStatus())
	}
	if fields.Has(updateFieldPriority) {
		item.Priority = int(req.GetPriority())
	}
	if fields.Has(updateFieldTags) {
		item.Tags = cloneStringsOrEmpty(req.Tags)
	}
	if fields.Has(updateFieldResearchTarget) {
		item.ResearchTarget = req.GetResearchTarget()
	}
	if fields.Has(updateFieldDependsOn) {
		item.DependsOn = cloneStrings(req.DependsOn)
	}
	if fields.Has(updateFieldInitiative) {
		item.Initiative = strings.TrimSpace(req.GetInitiative())
	}
	if fields.Has(updateFieldEffort) {
		item.Effort = req.GetEffort()
	}
	if fields.Has(updateFieldAcceptanceAllow) {
		item.AcceptanceAllow = cloneStrings(req.AcceptanceAllow)
	}
	if fields.Has(updateFieldAcceptanceDeny) {
		item.AcceptanceDeny = cloneStrings(req.AcceptanceDeny)
	}
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string(nil), values...)
}

func cloneStringsOrEmpty(values []string) []string {
	if values == nil {
		return []string{}
	}
	return append([]string(nil), values...)
}

func validateUniqueStrings(field string, values []string) error {
	seen := make(map[string]struct{}, len(values))
	for i, value := range values {
		if _, exists := seen[value]; exists {
			return fmt.Errorf("%s[%d]: duplicate value %q", field, i, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}
