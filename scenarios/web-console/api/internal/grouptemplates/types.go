// Package grouptemplates owns the group-template domain: a saved list of role
// definitions that creates a group and its roles in one action.
//
// A template carries no privilege. Shipped examples are ordinary rows written
// through the same Upsert call the UI uses, so deleting one leaves every
// capability working. There is deliberately no built-in marker anywhere in
// this package — shipped content is data, not behaviour.
//
// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: docs/internal/ROLES-AND-HANDOFFS-UX.md
package grouptemplates

import (
	"errors"
	"time"
)

// StartMode decides whether a template role starts a process when the group
// is created, or becomes a waiting placeholder.
const (
	StartModeEager   = "eager"
	StartModeWaiting = "waiting"
)

// ErrInvalidTemplate is returned for caller-visible validation failures: a
// blank name, or a role with an unknown start mode. Handlers map it to
// CodeInvalidArgument.
var ErrInvalidTemplate = errors.New("grouptemplates: invalid template")

// TemplateRole is one role definition inside a template.
//
// Command is a plain string, never an enum of known agents: a template must be
// able to describe any executable the operator can type.
type TemplateRole struct {
	Label          string `json:"label"`
	Command        string `json:"command"`
	WorkingDir     string `json:"working_dir"`
	IncomingPrompt string `json:"incoming_prompt"`
	Backend        string `json:"backend"`
	TargetID       string `json:"target_id"`
	StartMode      string `json:"start_mode"`
}

// IsEager reports whether creating a group from this template should start a
// process for this role. Anything that is not explicitly eager waits, so an
// unrecognised value can never cost the operator an unexpected process.
func (r TemplateRole) IsEager() bool { return r.StartMode == StartModeEager }

// Template is a named, colour-tagged list of roles.
//
// Roles is an ordered list of any length from zero upward. Nothing in this
// type constrains it to two: a pair-shaped template is one shape among many,
// not the model.
type Template struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Color     string         `json:"color"`
	Roles     []TemplateRole `json:"roles"`
	UseCount  int            `json:"use_count"`
	CreatedAt string         `json:"created_at,omitempty"`
	UpdatedAt string         `json:"updated_at,omitempty"`
}

// UpsertRequest is the create-or-update payload. A blank ID is assigned by
// the store.
type UpsertRequest struct {
	ID       string
	Name     string
	Color    string
	Roles    []TemplateRole
	UseCount int
	// HasUseCount distinguishes "leave the counter alone" from "set it to
	// zero", which matters because incrementing after a successful create is
	// a separate write from editing the template's content.
	HasUseCount bool
}

// Validate rejects the caller mistakes the store must not persist.
func (r UpsertRequest) Validate() error {
	if r.Name == "" {
		return ErrInvalidTemplate
	}
	for _, role := range r.Roles {
		switch role.StartMode {
		case StartModeEager, StartModeWaiting:
		default:
			return ErrInvalidTemplate
		}
	}
	return nil
}

// FormatTime returns a UTC RFC3339Nano string for the timestamp columns.
func FormatTime(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }
