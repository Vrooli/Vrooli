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
	KindSecret  Kind = "secret"
	KindChoice  Kind = "choice"
	KindConfirm Kind = "confirm"
)

type Request struct {
	ID          string   `json:"id"`
	Kind        Kind     `json:"kind"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Default     string   `json:"default,omitempty"`
	Options     []string `json:"options,omitempty"`
	Unblocks    []string `json:"unblocks,omitempty"`
	Validation  string   `json:"validation,omitempty"`
	Required    bool     `json:"required"`
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
	case KindSecret, KindChoice, KindConfirm:
	default:
		return fmt.Errorf("operator input %q has unsupported kind %q", request.ID, request.Kind)
	}
	if request.Kind == KindChoice && len(request.Options) == 0 {
		return fmt.Errorf("operator input %q choice has no options", request.ID)
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
		if request.Kind == KindChoice && value != "" && !contains(request.Options, value) {
			return nil, fmt.Errorf("operator input %q has invalid choice %q", request.ID, value)
		}
		if request.Kind == KindConfirm && value != "" && value != "true" && value != "false" {
			return nil, fmt.Errorf("operator input %q must be true or false", request.ID)
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
	return values, nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
