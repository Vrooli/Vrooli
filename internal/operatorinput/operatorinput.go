// Package operatorinput is the durable, typed handoff between non-interactive
// setup and vrooli-onboarding. Producers describe a decision; onboarding is
// the only component that collects its value.
package operatorinput

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
)

type Kind string

const (
	KindSecret       Kind = "secret"
	KindChoice       Kind = "choice"
	KindConfirm      Kind = "confirm"
	KindPath         Kind = "path"
	KindEnum         Kind = "enum"
	KindBoolean      Kind = "boolean"
	KindDuration     Kind = "duration"
	KindConfirmation Kind = "confirmation"
)

// Candidate is metadata only. It is safe to persist in the operator-input
// queue because it identifies a possible destination without carrying a
// credential, token, or passphrase.
type Candidate struct {
	ID             string            `json:"id"`
	Kind           string            `json:"kind"`
	Label          string            `json:"label"`
	Location       string            `json:"location,omitempty"`
	StableIdentity string            `json:"stable_identity,omitempty"`
	DeviceIdentity string            `json:"device_identity,omitempty"`
	Writable       bool              `json:"writable"`
	Status         string            `json:"status"`
	Risk           string            `json:"risk,omitempty"`
	Remediation    string            `json:"remediation,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type Request struct {
	ID              string      `json:"id"`
	Kind            Kind        `json:"kind"`
	ContractVersion string      `json:"contract_version,omitempty"`
	Owner           string      `json:"owner,omitempty"`
	CapabilityID    string      `json:"capability_id,omitempty"`
	ActionID        string      `json:"action_id,omitempty"`
	InputID         string      `json:"input_id,omitempty"`
	Title           string      `json:"title"`
	Description     string      `json:"description,omitempty"`
	Default         string      `json:"default,omitempty"`
	Options         []string    `json:"options,omitempty"`
	Candidates      []Candidate `json:"candidates,omitempty"`
	Remediation     string      `json:"remediation,omitempty"`
	Unblocks        []string    `json:"unblocks,omitempty"`
	Validation      string      `json:"validation,omitempty"`
	Required        bool        `json:"required"`
}

type Pending struct {
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
	Requests  []Request `json:"requests"`
}

type Answer struct {
	RequestID string `json:"request_id"`
	Value     string `json:"value"`
}

var (
	mu          sync.Mutex
	queuePathFn = func() (string, error) {
		return config.VrooliPath(repocontract.HomeKeyState, "operator-input.json")
	}
)

func QueuePath() (string, error) { return queuePathFn() }

func Validate(request Request) error {
	if strings.TrimSpace(request.ID) == "" || strings.TrimSpace(request.Title) == "" {
		return errors.New("operator input id and title are required")
	}
	switch request.Kind {
	case KindSecret, KindChoice, KindConfirm, KindPath, KindEnum, KindBoolean, KindDuration, KindConfirmation:
	default:
		return fmt.Errorf("operator input %q has unsupported kind %q", request.ID, request.Kind)
	}
	if (request.Kind == KindChoice || request.Kind == KindEnum) && len(request.Options) == 0 {
		return fmt.Errorf("operator input %q choice has no options", request.ID)
	}
	if (request.Kind == KindConfirm || request.Kind == KindConfirmation) && !request.Required {
		return fmt.Errorf("operator input %q confirmation must be required", request.ID)
	}
	return nil
}

func Replace(requests []Request) error {
	for _, request := range requests {
		if err := Validate(request); err != nil {
			return err
		}
	}
	mu.Lock()
	defer mu.Unlock()
	path, err := QueuePath()
	if err != nil {
		return err
	}
	if len(requests) == 0 {
		if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
			return removeErr
		}
		return nil
	}
	queue := Pending{Version: 1, UpdatedAt: time.Now().UTC(), Requests: append([]Request(nil), requests...)}
	data, err := json.MarshalIndent(queue, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func Enqueue(request Request) error {
	queue, err := Load()
	if err != nil {
		return err
	}
	for _, existing := range queue.Requests {
		if existing.ID == request.ID {
			return Replace(queue.Requests)
		}
	}
	return Replace(append(queue.Requests, request))
}

// RemoveCapability removes only metadata requests owned by capabilityID after
// the generic provider has returned verified success. It never accepts or
// receives an answer value, so queue reconciliation cannot persist secrets.
func RemoveCapability(capabilityID string) error {
	capabilityID = strings.TrimSpace(capabilityID)
	if capabilityID == "" {
		return errors.New("capability ID is required")
	}
	queue, err := Load()
	if err != nil {
		return err
	}
	remaining := queue.Requests[:0]
	for _, request := range queue.Requests {
		if request.CapabilityID != capabilityID {
			remaining = append(remaining, request)
		}
	}
	if len(remaining) == len(queue.Requests) {
		return nil
	}
	return Replace(remaining)
}

func Load() (Pending, error) {
	path, err := QueuePath()
	if err != nil {
		return Pending{}, err
	}
	mu.Lock()
	defer mu.Unlock()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Pending{Version: 1, Requests: []Request{}}, nil
		}
		return Pending{}, err
	}
	var queue Pending
	if err := json.Unmarshal(data, &queue); err != nil {
		return Pending{}, fmt.Errorf("decode operator input queue: %w", err)
	}
	return queue, nil
}

func Resolve(answers []Answer) error {
	_, err := ResolveWith(answers, nil)
	return err
}

// ResolveWith validates answers and invokes apply before removing the queue.
// The callback receives values only in memory; callers must not persist the
// map or include it in diagnostics. If apply fails, the request remains
// pending so the operator can retry without losing the decision.
func ResolveWith(answers []Answer, apply func(map[string]string) error) (map[string]string, error) {
	queue, err := Load()
	if err != nil {
		return nil, err
	}
	byID := make(map[string]string, len(answers))
	for _, answer := range answers {
		if strings.TrimSpace(answer.RequestID) == "" {
			return nil, errors.New("operator input answer request_id is required")
		}
		if _, exists := byID[answer.RequestID]; exists {
			return nil, fmt.Errorf("operator input %q was answered more than once", answer.RequestID)
		}
		byID[answer.RequestID] = answer.Value
	}
	known := make(map[string]struct{}, len(queue.Requests))
	values := make(map[string]string, len(queue.Requests))
	for _, request := range queue.Requests {
		known[request.ID] = struct{}{}
		value, ok := byID[request.ID]
		if !ok {
			value = request.Default
		}
		if request.Required && strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("operator input %q is required", request.ID)
		}
		if (request.Kind == KindChoice || request.Kind == KindEnum) && value != "" && !contains(request.Options, value) {
			return nil, fmt.Errorf("operator input %q has invalid choice %q", request.ID, value)
		}
		if (request.Kind == KindConfirm || request.Kind == KindConfirmation) && value != "" && value != "true" && value != "false" {
			return nil, fmt.Errorf("operator input %q must be true or false", request.ID)
		}
		if request.Kind == KindConfirmation && value != "true" {
			return nil, fmt.Errorf("operator input %q must be true", request.ID)
		}
		if request.Kind == KindDuration && value != "" {
			if duration, durationErr := time.ParseDuration(value); durationErr != nil || duration <= 0 {
				return nil, fmt.Errorf("operator input %q must be a positive duration", request.ID)
			}
		}
		values[request.ID] = value
	}
	for id := range byID {
		if _, ok := known[id]; !ok {
			return nil, fmt.Errorf("operator input %q is not pending", id)
		}
	}
	if apply != nil {
		if err := apply(values); err != nil {
			return nil, err
		}
	}
	if err := Replace(nil); err != nil {
		return nil, err
	}
	// Values are returned only for backwards-compatible callers that need the
	// callback result during this invocation. Clear the map before returning so
	// a passphrase cannot survive in a caller-owned result after resolution.
	clear(values)
	return nil, nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
