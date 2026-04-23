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

	"swarm-manager/internal/backlogstatus"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

const (
	updateFieldTitle           = "title"
	updateFieldDescription     = "description"
	updateFieldStatus          = "status"
	updateFieldPriority        = "priority"
	updateFieldTags            = "tags"
	updateFieldDependsOn       = "depends_on"
	updateFieldInitiative      = "initiative"
	updateFieldEffort          = "effort"
	updateFieldAcceptanceAllow = "acceptance_allow"
	updateFieldAcceptanceDeny  = "acceptance_deny"
	updateFieldSpawnedFrom     = "spawned_from"
	updateFieldNote            = "note"
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
	if fields.Has(updateFieldInitiative) && req.Initiative != nil {
		trimmed := strings.TrimSpace(*req.Initiative)
		req.Initiative = &trimmed
	}
	if fields.Has(updateFieldEffort) && req.Effort != nil {
		normalized := strings.ToUpper(strings.TrimSpace(*req.Effort))
		req.Effort = &normalized
	}
	if fields.Has(updateFieldNote) && req.Note != nil {
		trimmed := strings.TrimSpace(*req.Note)
		req.Note = &trimmed
	}
}

func validateUpdateBacklogItemRequest(req *apipb.UpdateBacklogItemRequest, fields backlogUpdateFieldSet, kind BacklogKind, existingStatus BacklogStatus) string {
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
		// Whitelist: only backlog, researching, ready are user-settable via
		// PATCH. Everything else is owned by execution, the review agent,
		// the review gate, or review-decide. New statuses added to the enum
		// default to NOT user-settable (enforced by backlogstatus.IsUserSettable).
		if !backlogstatus.IsUserSettable(status) {
			switch status {
			case string(StatusQueued), string(StatusInProgress):
				return "status 'queued' and 'in_progress' can only be set by the execution system"
			case string(StatusInReview):
				return "status 'in_review' is set by the execution/review system; use the review-decide endpoint to complete review"
			case string(StatusReviewPending):
				return "status 'review_pending' is set by the review system; use the review-decide endpoint to accept or reject the review"
			case string(StatusCompleted), string(StatusFailed), string(StatusNeedsFollowup):
				return fmt.Sprintf("status %q is a terminal state; set it via the review-decide endpoint so the decision is audited", status)
			}
			return fmt.Sprintf("status %q is not user-settable via PATCH", status)
		}
		// Force review-gated items through the review-decide endpoint. PATCH
		// cannot short-circuit the review flow because review-decide is the
		// audit trail for the terminal decision (records rationale, decider,
		// and fires the itemTerminalHandler for downstream work).
		if IsReviewStatus(existingStatus) && BacklogStatus(status) != existingStatus {
			return fmt.Sprintf("item is in status %q; use the review-decide endpoint to change status", existingStatus)
		}
		// Defense-in-depth: reject transitions the state machine considers
		// nonsensical regardless of caller (terminal → anything, etc.).
		// The whitelist above handles most cases, but IsValidTransition
		// catches the rare edge (e.g., if a terminal were ever added to
		// IsUserSettable by mistake).
		if !IsValidTransition(existingStatus, BacklogStatus(status)) {
			return fmt.Sprintf("status transition %q → %q is not allowed", existingStatus, status)
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

// applyUpdateBacklogPatch adapts a proto-shaped PATCH request into the
// struct-based ItemPatch, then delegates the actual mutation to the
// shared ApplyItemPatch helper. The handler path retains its proto wire
// format; the field-assignment semantics are defined once.
//
// The proto request does not natively distinguish "unset" from "empty",
// which is why fields+getters combine to reconstruct presence. Callers
// must have already normalized the request (normalizeUpdateBacklogPatch)
// so trim/case rules don't apply twice when ApplyItemPatch re-normalizes
// on the struct side.
func applyUpdateBacklogPatch(item *BacklogItem, req *apipb.UpdateBacklogItemRequest, fields backlogUpdateFieldSet) {
	patch := ItemPatch{}
	if fields.Has(updateFieldTitle) {
		v := req.GetTitle()
		patch.Title = &v
	}
	if fields.Has(updateFieldDescription) {
		v := req.GetDescription()
		patch.Description = &v
	}
	if fields.Has(updateFieldStatus) {
		v := req.GetStatus()
		patch.Status = &v
	}
	if fields.Has(updateFieldPriority) {
		v := int(req.GetPriority())
		patch.Priority = &v
	}
	if fields.Has(updateFieldTags) {
		v := cloneStringsOrEmpty(req.Tags)
		patch.Tags = &v
	}
	if fields.Has(updateFieldDependsOn) {
		v := cloneStrings(req.DependsOn)
		patch.DependsOn = &v
	}
	if fields.Has(updateFieldInitiative) {
		v := req.GetInitiative()
		patch.Initiative = &v
	}
	if fields.Has(updateFieldEffort) {
		v := req.GetEffort()
		patch.Effort = &v
	}
	if fields.Has(updateFieldAcceptanceAllow) {
		v := cloneStrings(req.AcceptanceAllow)
		patch.AcceptanceAllow = &v
	}
	if fields.Has(updateFieldAcceptanceDeny) {
		v := cloneStrings(req.AcceptanceDeny)
		patch.AcceptanceDeny = &v
	}
	if fields.Has(updateFieldSpawnedFrom) {
		v := req.GetSpawnedFrom()
		patch.SpawnedFrom = &v
	}
	if fields.Has(updateFieldNote) {
		v := req.GetNote()
		patch.Note = &v
	}
	ApplyItemPatch(item, patch)
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
