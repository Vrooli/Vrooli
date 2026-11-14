package policy

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var dailySchedulePattern = regexp.MustCompile(`^daily:([0-2][0-9]):([0-5][0-9])-([0-2][0-9]):([0-5][0-9])$`)

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

func (s *Service) ListProfiles(ctx context.Context, deviceGroup string) ([]Profile, error) {
	return s.repo.ListProfiles(ctx, normalizeDeviceGroup(deviceGroup))
}

func (s *Service) UpsertProfile(ctx context.Context, profile Profile) (Profile, error) {
	normalized, err := s.normalizeProfile(profile)
	if err != nil {
		return Profile{}, err
	}
	normalized.UpdatedAt = s.now().UTC()
	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = normalized.UpdatedAt
	}
	return s.repo.UpsertProfile(ctx, normalized)
}

func (s *Service) EvaluateSchedule(ctx context.Context, profileID, target string, now time.Time) (ScheduleEvaluation, error) {
	if strings.TrimSpace(profileID) == "" {
		return ScheduleEvaluation{}, fmt.Errorf("profile_id is required")
	}
	profile, err := s.repo.GetProfile(ctx, strings.TrimSpace(profileID))
	if err != nil {
		return ScheduleEvaluation{}, err
	}
	if now.IsZero() {
		now = s.now().UTC()
	}
	target = normalizeTarget(target)
	evaluation := ScheduleEvaluation{
		ProfileID:   profile.ID,
		ProfileName: profile.Name,
		Target:      target,
		Status:      "manual_required",
		Effects: []string{
			fmt.Sprintf("Profile %s applies %s filtering to %s.", profile.Name, profile.FilteringStrength, target),
			"Schedule evaluation is advisory until an approved policy apply is run.",
		},
	}
	if profile.Status != "enabled" {
		evaluation.Status = "disabled"
		evaluation.Effects = append(evaluation.Effects, "Profile is disabled.")
		return evaluation, nil
	}
	active, nextChangeAt, err := evaluateScheduleWindow(profile.Schedule, now.UTC())
	if err != nil {
		return ScheduleEvaluation{}, err
	}
	evaluation.Active = active
	evaluation.NextChangeAt = nextChangeAt
	if active {
		evaluation.Status = "active"
		evaluation.Effects = append(evaluation.Effects, fmt.Sprintf("Override behavior: %s.", profile.OverrideBehavior))
		return evaluation, nil
	}
	evaluation.Status = "inactive"
	evaluation.Effects = append(evaluation.Effects, "Current time is outside the configured schedule.")
	return evaluation, nil
}

func (s *Service) DiagnoseEncryptedDNSBypass(_ context.Context, target string, adapterBacked bool) GuidanceReport {
	target = normalizeTarget(target)
	report := GuidanceReport{
		ID:          uuid.NewString(),
		Target:      target,
		Profile:     "ipv6-encrypted-dns",
		Status:      "manual_required",
		GeneratedAt: s.now().UTC(),
		Checks: []GuidanceCheck{
			{
				ID:       "ipv6-dns",
				Title:    "IPv6 resolver path",
				Status:   "review_required",
				Evidence: "Host and resolver capabilities do not prove that IPv6 clients are using the managed resolver.",
				Recommendations: []string{
					"Confirm router advertisements publish the managed resolver over IPv6.",
					"Disable unmanaged ISP IPv6 DNS advertisement or add an equivalent managed IPv6 resolver.",
				},
			},
			{
				ID:       "dot-doq",
				Title:    "Encrypted DNS transport bypass",
				Status:   "manual_required",
				Evidence: "DoT and DoQ policy needs router or firewall support; Network Manager will not fake enforcement.",
				Recommendations: []string{
					"Review outbound TCP/UDP 853 and UDP 784/8853 handling on the router or firewall.",
					"Prefer adapter-backed firewall rules with rollback support before applying persistent blocks.",
				},
			},
			{
				ID:       "doh",
				Title:    "DNS over HTTPS bypass",
				Status:   "guidance_only",
				Evidence: "DoH can blend with ordinary HTTPS traffic and must be handled through endpoint/browser policy where possible.",
				Recommendations: []string{
					"Use managed browser or OS policy for enrolled endpoints.",
					"Avoid TLS interception or hidden traffic inspection.",
				},
			},
		},
		ManualSteps: []string{
			"Document current router IPv6 DNS advertisement settings before making changes.",
			"Validate managed resolver answers over both IPv4 and IPv6 after changes.",
			"Keep a manual recovery path for clients that lose name resolution.",
		},
		Guardrails: []string{
			"No router or firewall mutation is attempted by this guidance command.",
			"Unsupported adapters must return manual_required rather than success.",
			"Do not inspect or log user browsing contents to detect bypasses.",
		},
	}
	if adapterBacked {
		report.AdapterActions = []string{
			"Preview adapter rule: advertise managed IPv6 resolver when router adapter supports rollback.",
			"Preview adapter rule: block unmanaged encrypted DNS transports only when firewall adapter supports rollback.",
		}
		return report
	}
	report.AdapterActions = []string{"No adapter-backed bypass action is currently available for this target."}
	return report
}

func (s *Service) EndpointDoHGuidance(_ context.Context, platform, browser, managementMode string) GuidanceReport {
	platform = normalizeGuidanceField(platform, "unknown-platform")
	browser = normalizeGuidanceField(browser, "managed-browser")
	managementMode = normalizeGuidanceField(managementMode, "manual")
	report := GuidanceReport{
		ID:          uuid.NewString(),
		Target:      platform + "/" + browser,
		Profile:     "endpoint-doh",
		Status:      "guidance_only",
		GeneratedAt: s.now().UTC(),
		Checks: []GuidanceCheck{
			{
				ID:       "management-mode",
				Title:    "Endpoint management mode",
				Status:   endpointManagementStatus(managementMode),
				Evidence: fmt.Sprintf("Management mode is %s.", managementMode),
				Recommendations: []string{
					"Use MDM, group policy, or managed browser policy when available.",
					"Use manual instructions only for unmanaged personal endpoints.",
				},
			},
			{
				ID:              "browser-doh",
				Title:           "Browser DNS over HTTPS policy",
				Status:          "guidance_only",
				Evidence:        fmt.Sprintf("Generate browser policy guidance for %s without traffic interception.", browser),
				Recommendations: endpointBrowserRecommendations(browser),
			},
			{
				ID:       "privacy",
				Title:    "Privacy boundary",
				Status:   "enforced_by_design",
				Evidence: "Guidance avoids TLS interception, packet capture, and query-level surveillance.",
				Recommendations: []string{
					"Communicate endpoint policy changes to affected users.",
					"Keep query visibility disabled unless an explicit audit profile is enabled.",
				},
			},
		},
		ManualSteps: []string{
			fmt.Sprintf("Apply %s policy guidance for %s on %s.", managementMode, browser, platform),
			"Restart the browser or refresh managed policy before validating.",
			"Run a new network snapshot and resolver health check after endpoint policy changes.",
		},
		AdapterActions: []string{"Endpoint DoH guidance is advisory; Network Manager does not mutate endpoint policy directly."},
		Guardrails: []string{
			"No TLS interception.",
			"No hidden endpoint monitoring.",
			"No claim of enforcement without managed endpoint policy confirmation.",
		},
	}
	return report
}

func (s *Service) normalizeProfile(profile Profile) (Profile, error) {
	profile.ID = strings.TrimSpace(profile.ID)
	if profile.ID == "" {
		profile.ID = uuid.NewString()
	}
	profile.Name = strings.TrimSpace(profile.Name)
	if profile.Name == "" {
		return Profile{}, fmt.Errorf("profile name is required")
	}
	profile.DeviceGroup = normalizeDeviceGroup(profile.DeviceGroup)
	strength, err := normalizeFilteringStrength(profile.FilteringStrength)
	if err != nil {
		return Profile{}, err
	}
	profile.FilteringStrength = strength
	schedule, err := normalizeSchedule(profile.Schedule)
	if err != nil {
		return Profile{}, err
	}
	profile.Schedule = schedule
	profile.OverrideBehavior = normalizeOverrideBehavior(profile.OverrideBehavior)
	profile.Status = normalizeProfileStatus(profile.Status)
	profile.Effects = []string{
		fmt.Sprintf("Profile %s targets group %s.", profile.Name, profile.DeviceGroup),
		fmt.Sprintf("Filtering strength: %s.", profile.FilteringStrength),
		fmt.Sprintf("Schedule: %s.", profile.Schedule),
		"Persistent resolver changes still require preview and approval.",
	}
	return profile, nil
}

func normalizeGuidanceField(value, fallback string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, " ", "-")
	if value == "" {
		return fallback
	}
	return value
}

func endpointManagementStatus(mode string) string {
	switch mode {
	case "mdm", "group-policy", "managed-browser":
		return "policy_available"
	default:
		return "manual_required"
	}
}

func endpointBrowserRecommendations(browser string) []string {
	switch browser {
	case "chrome", "chromium", "edge":
		return []string{
			"Use managed browser DNS-over-HTTPS mode policy to disable automatic DoH or force approved templates.",
			"Confirm the policy source is managed before relying on enforcement.",
		}
	case "firefox":
		return []string{
			"Use enterprise policies to set DNSOverHTTPS mode and provider behavior.",
			"Confirm user-level settings cannot override the managed policy.",
		}
	case "safari":
		return []string{
			"Use Apple platform profiles or network extension guidance where available.",
			"Validate iCloud Private Relay and per-network DNS settings separately.",
		}
	default:
		return []string{
			"Use vendor-supported managed policy for DoH behavior when available.",
			"Document manual settings for unmanaged endpoints.",
		}
	}
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

func normalizeDeviceGroup(group string) string {
	group = strings.TrimSpace(strings.ToLower(group))
	group = strings.ReplaceAll(group, " ", "-")
	if group == "" {
		return "trusted"
	}
	return group
}

func normalizeFilteringStrength(strength string) (string, error) {
	strength = strings.TrimSpace(strings.ToLower(strength))
	strength = strings.ReplaceAll(strength, " ", "-")
	if strength == "" {
		strength = "standard"
	}
	switch strength {
	case "off", "light", "standard", "strict":
		return strength, nil
	default:
		return "", fmt.Errorf("unsupported filtering strength %q", strength)
	}
}

func normalizeSchedule(schedule string) (string, error) {
	schedule = strings.TrimSpace(strings.ToLower(schedule))
	if schedule == "" {
		return "always", nil
	}
	if schedule == "always" || schedule == "disabled" {
		return schedule, nil
	}
	matches := dailySchedulePattern.FindStringSubmatch(schedule)
	if matches == nil {
		return "", fmt.Errorf("unsupported schedule %q; use always, disabled, or daily:HH:MM-HH:MM", schedule)
	}
	if matches[1] > "23" || matches[3] > "23" {
		return "", fmt.Errorf("schedule hour must be 00 through 23")
	}
	return schedule, nil
}

func normalizeOverrideBehavior(behavior string) string {
	behavior = strings.TrimSpace(strings.ToLower(behavior))
	behavior = strings.ReplaceAll(behavior, " ", "_")
	if behavior == "" {
		return "manual_required"
	}
	switch behavior {
	case "manual_required", "parent_override", "temporary_pause":
		return behavior
	default:
		return "manual_required"
	}
}

func normalizeProfileStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return "enabled"
	}
	switch status {
	case "enabled", "disabled":
		return status
	default:
		return "enabled"
	}
}

func evaluateScheduleWindow(schedule string, now time.Time) (bool, time.Time, error) {
	switch schedule {
	case "always":
		return true, time.Time{}, nil
	case "disabled":
		return false, time.Time{}, nil
	}
	matches := dailySchedulePattern.FindStringSubmatch(schedule)
	if matches == nil {
		return false, time.Time{}, fmt.Errorf("unsupported schedule %q", schedule)
	}
	startHour, startMinute := parseHourMinute(matches[1], matches[2])
	endHour, endMinute := parseHourMinute(matches[3], matches[4])
	start := time.Date(now.Year(), now.Month(), now.Day(), startHour, startMinute, 0, 0, now.Location())
	end := time.Date(now.Year(), now.Month(), now.Day(), endHour, endMinute, 0, 0, now.Location())
	if !end.After(start) {
		end = end.Add(24 * time.Hour)
		if now.Before(start) {
			start = start.Add(-24 * time.Hour)
			end = end.Add(-24 * time.Hour)
		}
	}
	active := (now.Equal(start) || now.After(start)) && now.Before(end)
	if active {
		return true, end, nil
	}
	if now.Before(start) {
		return false, start, nil
	}
	return false, start.Add(24 * time.Hour), nil
}

func parseHourMinute(hour, minute string) (int, int) {
	h := int(hour[0]-'0')*10 + int(hour[1]-'0')
	m := int(minute[0]-'0')*10 + int(minute[1]-'0')
	return h, m
}
