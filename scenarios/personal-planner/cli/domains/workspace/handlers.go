package workspace

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/workspace"
	vc "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/workspace/workspace_v1connect"
)

type handlers struct{ client vc.WorkspaceServiceClient }

func newHandlers(c *cliapp.ScenarioApp) *handlers {
	httpClient, base := cliapp.NewConnectHTTPClient(c)
	return &handlers{client: vc.NewWorkspaceServiceClient(httpClient, base)}
}

func (h *handlers) profileCall(_ cliapp.OperationContext) (*v.GetProfileResponse, error) {
	r, err := h.client.GetProfile(context.Background(), connect.NewRequest(&v.GetProfileRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get planning profile", err, nil)
	}
	if r == nil || r.Msg == nil || r.Msg.Profile == nil {
		return nil, fmt.Errorf("server returned no planning profile")
	}
	return r.Msg, nil
}

func (h *handlers) profileReport(_ cliapp.OperationContext, m *v.GetProfileResponse) cliapp.ListReport {
	p := m.Profile
	return cliapp.ListReport{Summary: []string{"Planning profile"}, ResultsHeading: "Profile", Results: []string{fmt.Sprintf("timezone=%s week_start=%s capacity=%dm reserve=%dm focus=%dm revision=%d", p.Timezone, p.WeekStart, p.DailyCapacityMinutes, p.ReserveMinutes, p.FocusSessionMinutes, p.Revision)}}
}

func (h *handlers) updateCall(c cliapp.OperationContext) (*v.UpdateProfileResponse, error) {
	capacity, err := parseInt32Flag(c.Flag("daily-capacity-minutes"), "daily-capacity-minutes")
	if err != nil {
		return nil, err
	}
	reserve, err := parseInt32Flag(c.Flag("reserve-minutes"), "reserve-minutes")
	if err != nil {
		return nil, err
	}
	focus, err := parseInt32Flag(c.Flag("focus-session-minutes"), "focus-session-minutes")
	if err != nil {
		return nil, err
	}
	revision, err := strconv.ParseInt(c.Flag("revision"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("revision must be an integer: %w", err)
	}
	r, err := h.client.UpdateProfile(context.Background(), connect.NewRequest(&v.UpdateProfileRequest{Timezone: c.Flag("timezone"), WeekStart: c.Flag("week-start"), DailyCapacityMinutes: capacity, ReserveMinutes: reserve, FocusSessionMinutes: focus, ExpectedRevision: revision}))
	if err != nil {
		return nil, cliapp.WrapAPIError("update planning profile", err, nil)
	}
	if r == nil || r.Msg == nil || r.Msg.Profile == nil {
		return nil, fmt.Errorf("server returned no planning profile")
	}
	return r.Msg, nil
}

func parseInt32Flag(raw, name string) (int32, error) {
	n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}
	return int32(n), nil
}

func (h *handlers) updateReport(_ cliapp.OperationContext, m *v.UpdateProfileResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Updated planning profile to revision %d.", m.Profile.Revision)}, NextCommand: []string{"`workspace profile` — inspect the current settings"}}
}

func (h *handlers) availabilityCall(_ cliapp.OperationContext) (*v.ListAvailabilityResponse, error) {
	r, err := h.client.ListAvailability(context.Background(), connect.NewRequest(&v.ListAvailabilityRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list availability", err, nil)
	}
	return r.Msg, nil
}

func (h *handlers) availabilityReport(_ cliapp.OperationContext, m *v.ListAvailabilityResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Availability revision %d", m.Revision)}, ResultsHeading: "Configured windows and exceptions", Results: []string{fmt.Sprintf("windows=%d exceptions=%d", len(m.Windows), len(m.Exceptions))}}
}

func (h *handlers) replaceAvailabilityCall(c cliapp.OperationContext) (*v.ReplaceAvailabilityResponse, error) {
	var windows []*v.AvailabilityWindow
	if err := json.Unmarshal([]byte(c.Flag("windows-json")), &windows); err != nil {
		return nil, fmt.Errorf("windows-json must be a JSON array: %w", err)
	}
	var exceptions []*v.AvailabilityException
	if err := json.Unmarshal([]byte(c.Flag("exceptions-json")), &exceptions); err != nil {
		return nil, fmt.Errorf("exceptions-json must be a JSON array: %w", err)
	}
	revision, err := strconv.ParseInt(c.Flag("revision"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("revision must be an integer: %w", err)
	}
	r, err := h.client.ReplaceAvailability(context.Background(), connect.NewRequest(&v.ReplaceAvailabilityRequest{Windows: windows, Exceptions: exceptions, ExpectedRevision: revision}))
	if err != nil {
		return nil, cliapp.WrapAPIError("replace availability", err, nil)
	}
	return r.Msg, nil
}

func (h *handlers) replaceAvailabilityReport(_ cliapp.OperationContext, m *v.ReplaceAvailabilityResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Updated availability to revision %d (%d windows, %d exceptions).", m.Revision, len(m.Windows), len(m.Exceptions))}, NextCommand: []string{"`workspace availability` — inspect the canonical schedule"}}
}
