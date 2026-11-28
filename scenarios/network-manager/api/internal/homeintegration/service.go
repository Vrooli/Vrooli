package homeintegration

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo      Repository
	publisher Publisher
	now       func() time.Time
}

type Config struct {
	Repo      Repository
	Publisher Publisher
	Now       func() time.Time
}

func NewService(cfg Config) *Service {
	s := &Service{repo: cfg.Repo, publisher: cfg.Publisher, now: cfg.Now}
	if s.publisher == nil {
		s.publisher = NoopPublisher{}
	}
	if s.now == nil {
		s.now = func() time.Time { return time.Now().UTC() }
	}
	return s
}

func (s *Service) ListActions(context.Context) []Action {
	return []Action{
		{Name: "network.health.run", Description: "Run a read-only network health check for Home Automation.", Effect: "read", ApprovalRequired: false},
		{Name: "network.adblock.pause_device", Description: "Request a filtering pause for one device; returns manual-required unless a supported resolver adapter is configured.", Effect: "write", ApprovalRequired: true},
		{Name: "network.policy.apply_profile", Description: "Request a policy profile apply; returns manual-required unless a supported resolver adapter is configured.", Effect: "write", ApprovalRequired: true},
	}
}

func (s *Service) InvokeAction(ctx context.Context, name string, params map[string]string, approved bool) (Invocation, Event, error) {
	action, ok := s.actionByName(name)
	if !ok {
		return Invocation{}, Event{}, ErrUnknownAction
	}
	status, message := s.actionResult(action, approved)
	event := Event{
		ID:            uuid.NewString(),
		Type:          eventTypeForAction(action.Name, status),
		Summary:       redactedSummary(action, status, message),
		OccurredAt:    s.now().UTC(),
		PublishStatus: "pending",
	}
	if s.repo == nil {
		return Invocation{}, Event{}, fmt.Errorf("home integration repository is required")
	}
	savedEvent, err := s.repo.SaveEvent(ctx, event)
	if err != nil {
		return Invocation{}, Event{}, err
	}
	publishStatus := "published"
	publishErr := ""
	if err := s.publisher.Publish(ctx, savedEvent); err != nil {
		publishStatus = "publish_failed"
		publishErr = err.Error()
	}
	savedEvent, err = s.repo.UpdateEventPublish(ctx, savedEvent.ID, publishStatus, publishErr)
	if err != nil {
		return Invocation{}, Event{}, err
	}
	invocation, err := s.repo.SaveInvocation(ctx, Invocation{
		ID:         uuid.NewString(),
		ActionName: action.Name,
		Status:     status,
		Approved:   approved,
		Message:    message,
		Params:     sanitizeParams(params),
		EventID:    savedEvent.ID,
		CreatedAt:  s.now().UTC(),
	})
	if err != nil {
		return Invocation{}, Event{}, err
	}
	return invocation, savedEvent, nil
}

func (s *Service) ListRecentEvents(ctx context.Context, limit int) ([]Event, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("home integration repository is required")
	}
	if limit <= 0 {
		limit = 25
	}
	return s.repo.ListEvents(ctx, limit)
}

func (s *Service) actionByName(name string) (Action, bool) {
	name = strings.TrimSpace(name)
	for _, action := range s.ListActions(context.Background()) {
		if action.Name == name {
			return action, true
		}
	}
	return Action{}, false
}

func (s *Service) actionResult(action Action, approved bool) (string, string) {
	if action.ApprovalRequired && !approved {
		return "approval_required", fmt.Sprintf("%s requires explicit approval before Network Manager will consider a write action.", action.Name)
	}
	switch action.Name {
	case "network.health.run":
		return "accepted", "Read-only health action accepted; Network Manager recorded a redacted Home Automation event."
	case "network.adblock.pause_device", "network.policy.apply_profile":
		return "manual_required", "No supported Home Automation write adapter is configured; use Network Manager policy controls for preview, approval, and rollback."
	default:
		return "unsupported", "Action is unsupported."
	}
}

func eventTypeForAction(actionName, status string) string {
	if status == "manual_required" || status == "approval_required" {
		return "network.quality.degraded"
	}
	if actionName == "network.health.run" {
		return "network.quality.degraded"
	}
	return "network.outage.detected"
}

func redactedSummary(action Action, status, message string) string {
	return fmt.Sprintf("Home Automation action %s ended with %s. %s", action.Name, status, message)
}

func sanitizeParams(params map[string]string) map[string]string {
	out := make(map[string]string, len(params))
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		cleanKey := strings.TrimSpace(key)
		if cleanKey == "" {
			continue
		}
		out[cleanKey] = "[redacted]"
	}
	return out
}
