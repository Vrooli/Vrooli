package policy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo    Repository
	adapter ResolverPolicyAdapter
	now     func() time.Time
}

type Config struct {
	Repo    Repository
	Adapter ResolverPolicyAdapter
	Now     func() time.Time
}

func NewService(cfg Config) *Service {
	s := &Service{repo: cfg.Repo, adapter: cfg.Adapter, now: cfg.Now}
	if s.adapter == nil {
		s.adapter = ConservativeResolverPolicyAdapter{}
	}
	if s.now == nil {
		s.now = func() time.Time { return time.Now().UTC() }
	}
	return s
}

func (s *Service) Preview(ctx context.Context, target, action string, values []string) (Change, error) {
	target = normalizeTarget(target)
	action, err := normalizeAction(action)
	if err != nil {
		return Change{}, err
	}
	change := Change{
		ID:        uuid.NewString(),
		Target:    target,
		Action:    action,
		Status:    "previewed",
		Values:    cleanValues(values),
		CreatedAt: s.now().UTC(),
		UpdatedAt: s.now().UTC(),
	}
	plan, err := s.adapter.Preview(ctx, change)
	if err != nil {
		return Change{}, err
	}
	change.Effects = plan.Effects
	if len(change.Effects) == 0 {
		change.Effects = []string{fmt.Sprintf("Previewed %s for %s.", action, target)}
	}
	change.RollbackSupported = plan.RollbackSupported
	return s.repo.SaveChange(ctx, change)
}

func (s *Service) Apply(ctx context.Context, previewID string, approved bool) (Change, error) {
	if strings.TrimSpace(previewID) == "" {
		return Change{}, fmt.Errorf("preview_id is required")
	}
	change, err := s.repo.GetChange(ctx, previewID)
	if err != nil {
		return Change{}, err
	}
	if !approved {
		change.Status = "approval_required"
		change.Effects = append(change.Effects, "Persistent policy changes require --approved acknowledgement.")
		change.UpdatedAt = s.now().UTC()
		return s.repo.UpdateChange(ctx, change)
	}
	approval, err := s.repo.SaveApproval(ctx, ApprovalRecord{
		ID:        uuid.NewString(),
		ChangeID:  change.ID,
		Approved:  true,
		Note:      "operator approved persistent policy apply",
		CreatedAt: s.now().UTC(),
	})
	if err != nil {
		return Change{}, err
	}
	change.ApprovalID = approval.ID
	result, err := s.adapter.Apply(ctx, change)
	if err != nil {
		if errors.Is(err, ErrUnsupported) {
			change.Status = "unsupported"
			change.Effects = append(change.Effects, "Configured resolver adapter does not support live policy writes.")
			change.RollbackSupported = false
			change.UpdatedAt = s.now().UTC()
			return s.repo.UpdateChange(ctx, change)
		}
		change.Status = "apply_failed"
		change.Effects = append(change.Effects, err.Error())
		change.UpdatedAt = s.now().UTC()
		return s.repo.UpdateChange(ctx, change)
	}
	change.Status = "applied"
	change.Effects = result.Effects
	if len(change.Effects) == 0 {
		change.Effects = []string{"Policy change applied by resolver adapter."}
	}
	change.RollbackSupported = result.RollbackSupported
	change.RollbackHandle = result.RollbackHandle
	change.UpdatedAt = s.now().UTC()
	return s.repo.UpdateChange(ctx, change)
}

func (s *Service) Rollback(ctx context.Context, id string) (Change, error) {
	if strings.TrimSpace(id) == "" {
		return Change{}, fmt.Errorf("id is required")
	}
	change, err := s.repo.GetChange(ctx, id)
	if err != nil {
		return Change{}, err
	}
	if !change.RollbackSupported {
		change.Status = "unsupported"
		change.Effects = append(change.Effects, "No rollback handle is available for this policy change.")
		change.UpdatedAt = s.now().UTC()
		return s.repo.UpdateChange(ctx, change)
	}
	result, err := s.adapter.Rollback(ctx, change)
	if err != nil {
		change.Status = "rollback_failed"
		change.Effects = append(change.Effects, err.Error())
		change.UpdatedAt = s.now().UTC()
		updated, updateErr := s.repo.UpdateChange(ctx, change)
		if updateErr != nil {
			return Change{}, updateErr
		}
		_, _ = s.repo.SaveRollback(ctx, RollbackRecord{ID: uuid.NewString(), ChangeID: change.ID, Status: "failed", Details: []string{err.Error()}, CreatedAt: s.now().UTC()})
		return updated, nil
	}
	change.Status = "rolled_back"
	change.Effects = result.Effects
	if len(change.Effects) == 0 {
		change.Effects = []string{"Policy change rolled back by resolver adapter."}
	}
	change.UpdatedAt = s.now().UTC()
	updated, err := s.repo.UpdateChange(ctx, change)
	if err != nil {
		return Change{}, err
	}
	if _, err := s.repo.SaveRollback(ctx, RollbackRecord{ID: uuid.NewString(), ChangeID: change.ID, Status: "rolled_back", Details: change.Effects, CreatedAt: s.now().UTC()}); err != nil {
		return Change{}, err
	}
	return updated, nil
}

func (s *Service) Pause(ctx context.Context, target, duration string) (Change, error) {
	values := []string{}
	if strings.TrimSpace(duration) != "" {
		values = append(values, "duration="+strings.TrimSpace(duration))
	}
	return s.Preview(ctx, target, "pause_filtering", values)
}

func (s *Service) Resume(ctx context.Context, target string) (Change, error) {
	return s.Preview(ctx, target, "resume_filtering", nil)
}

func normalizeTarget(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return "network"
	}
	return target
}

func normalizeAction(action string) (string, error) {
	action = strings.TrimSpace(strings.ToLower(action))
	action = strings.ReplaceAll(action, "-", "_")
	if action == "" {
		action = "inspect"
	}
	switch action {
	case "allowlist", "denylist", "blocklist", "pause_filtering", "resume_filtering", "apply_profile", "inspect":
		return action, nil
	default:
		return "", fmt.Errorf("unsupported policy action %q", action)
	}
}

func cleanValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}
