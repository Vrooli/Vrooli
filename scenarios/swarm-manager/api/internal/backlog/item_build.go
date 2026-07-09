package backlog

import (
	"strings"
	"time"

	"swarm-manager/internal/httputil"
	"swarm-manager/internal/identity"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// CreateValidationError is a caller-facing validation failure from building a
// BacklogItem out of a CreateBacklogItemRequest. The HTTP path maps it to a
// 400; the Connect path maps it to CodeInvalidArgument. Anything that is NOT a
// *CreateValidationError is an internal error.
type CreateValidationError struct{ Msg string }

func (e *CreateValidationError) Error() string { return e.Msg }

func badCreate(msg string) error { return &CreateValidationError{Msg: msg} }

// buildItemFromCreateRequest validates and normalizes a CreateBacklogItemRequest
// into a BacklogItem. It is pure (no HTTP), so both the REST handler and the
// Connect BacklogService share the exact same validation + construction logic,
// which is the whole point of the shared typed contract. validateInitiative is
// injected because initiative-reference validation needs the store (a Handler
// method); pass a no-op-tolerant closure.
func buildItemFromCreateRequest(req *apipb.CreateBacklogItemRequest, prov identity.Provenance, validateInitiative func(string) error) (BacklogItem, error) {
	normalizeCreateBacklogItemRequest(req)

	if err := httputil.ValidateProto(req); err != nil {
		return BacklogItem{}, badCreate("invalid request body")
	}
	if validationErr := validateCreateBacklogItemRequest(req); validationErr != "" {
		return BacklogItem{}, badCreate(validationErr)
	}

	kind, err := ParseBacklogKind(req.Kind)
	if err != nil {
		return BacklogItem{}, badCreate(err.Error())
	}

	// Sanitize name (folder-safe). Allow title fallback.
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = req.Title
	}
	name = sanitizeName(name)
	if name == "" {
		return BacklogItem{}, badCreate("name is required")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	priority := 5
	if req.Priority != nil {
		priority = int(*req.Priority)
	}
	description := ""
	if req.Description != nil {
		description = *req.Description
	}
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	dependsOn := req.DependsOn
	if dependsOn == nil {
		dependsOn = []string{}
	}
	initiative := ""
	if req.Initiative != nil {
		initiative = strings.TrimSpace(*req.Initiative)
	}
	if validateInitiative != nil {
		if err := validateInitiative(initiative); err != nil {
			return BacklogItem{}, badCreate(err.Error())
		}
	}

	effort := ""
	if req.Effort != nil {
		normalized, err := validateEffort(*req.Effort)
		if err != nil {
			return BacklogItem{}, badCreate(err.Error())
		}
		effort = normalized
	}

	if err := validateGlobs(req.AcceptanceAllow); err != nil {
		return BacklogItem{}, badCreate("acceptance_allow: " + err.Error())
	}
	if err := validateGlobs(req.AcceptanceDeny); err != nil {
		return BacklogItem{}, badCreate("acceptance_deny: " + err.Error())
	}
	if err := validateGlobs(req.Creates); err != nil {
		return BacklogItem{}, badCreate("creates: " + err.Error())
	}

	spawnedFrom := ""
	if req.SpawnedFrom != nil {
		spawnedFrom = strings.TrimSpace(*req.SpawnedFrom)
	}
	planRef := planRefFromProto(req.PlanRef)
	planRef = normalizePlanRef(planRef)
	if err := validatePlanRef(planRef, PlanRefRoleExecutionSpec); err != nil {
		return BacklogItem{}, badCreate(err.Error())
	}
	note := ""
	if req.Note != nil {
		note = strings.TrimSpace(*req.Note)
	}

	item := BacklogItem{
		Name:            name,
		Title:           req.Title,
		Description:     description,
		Status:          StatusBacklog,
		Priority:        priority,
		Tags:            tags,
		Created:         now,
		Updated:         now,
		Kind:            kind,
		DependsOn:       dependsOn,
		Initiative:      initiative,
		Effort:          effort,
		AcceptanceAllow: req.AcceptanceAllow,
		AcceptanceDeny:  req.AcceptanceDeny,
		Creates:         req.Creates,
		SpawnedFrom:     spawnedFrom,
		PlanRef:         planRef,
		Note:            note,
		CreatedBy:       &prov,
	}
	return item, nil
}
