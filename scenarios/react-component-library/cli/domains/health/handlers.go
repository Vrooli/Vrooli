package health

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	libraryhealthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/libraryhealth"
	libraryhealthconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/libraryhealth/libraryhealth_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp + the generated
// Connect-Go client. Mirrors cli/domains/adoptions/.
type handlers struct {
	core   *cliapp.ScenarioApp
	client libraryhealthconnect.LibraryHealthServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: libraryhealthconnect.NewLibraryHealthServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) showCall(ctx cliapp.OperationContext) (*libraryhealthv1.GetLibraryHealthResponse, error) {
	days, err := parseNonNegativeInt32(ctx.Flag("activity-days"), "--activity-days")
	if err != nil {
		return nil, err
	}
	resp, err := h.client.GetLibraryHealth(context.Background(), connect.NewRequest(&libraryhealthv1.GetLibraryHealthRequest{ActivityDays: days}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get library health", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetHealth() == nil {
		return nil, fmt.Errorf("server returned no library health snapshot")
	}
	return resp.Msg, nil
}

func (h *handlers) showReport(_ cliapp.OperationContext, msg *libraryhealthv1.GetLibraryHealthResponse) cliapp.ListReport {
	health := msg.GetHealth()
	rows := make([]string, 0, 16)
	rows = appendSection(rows, "overview", health.GetOverview())
	rows = appendSection(rows, "quality", health.GetQuality())
	rows = appendSection(rows, "activity", health.GetActivity())
	rows = appendSection(rows, "services", health.GetServices())
	if counts := health.GetCounts(); counts != nil && countsAreUsable(health.GetOverview()) {
		rows = append(rows, "counts:")
		rows = append(rows, fmt.Sprintf("  assets: %d", counts.GetAssets()))
		rows = append(rows, fmt.Sprintf("  published_versions: %d", counts.GetPublished()))
		rows = append(rows, fmt.Sprintf("  workspaces: %d", counts.GetWorkspaces()))
		rows = append(rows, fmt.Sprintf("  findings_open: %d", counts.GetFindingsOpen()))
		rows = append(rows, fmt.Sprintf("  coverage_rows: %d", counts.GetCoverageRows()))
	}
	return cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Library health snapshot generated at %s.", nonEmpty(health.GetGeneratedAt(), "(unknown time)")),
			fmt.Sprintf("Sections: overview=%s quality=%s activity=%s services=%s.",
				sectionStateLabel(health.GetOverview()), sectionStateLabel(health.GetQuality()),
				sectionStateLabel(health.GetActivity()), sectionStateLabel(health.GetServices())),
		},
		ResultsHeading: "Sections",
		Results:        rows,
		RetrievalHints: []string{
			"`activity summary --days 30` — inspect the activity window directly",
			"`review summary` — inspect the review queue directly",
		},
	}
}

// appendSection renders one composed section. An unavailable or unclassified
// section prints its state and reason without its metrics: a zero-valued metric
// from a source that did not answer is not a measured zero and must never be
// shown as one.
func appendSection(rows []string, name string, section *libraryhealthv1.HealthSection) []string {
	if section == nil {
		return append(rows, fmt.Sprintf("%s: unavailable — no section returned", name))
	}
	reason := strings.TrimSpace(section.GetReason())
	switch section.GetState() {
	case libraryhealthv1.SectionState_SECTION_STATE_UNAVAILABLE:
		return append(rows, fmt.Sprintf("%s: unavailable — %s", name, nonEmpty(reason, "no reason supplied")))
	case libraryhealthv1.SectionState_SECTION_STATE_UNSPECIFIED:
		return append(rows, fmt.Sprintf("%s: unknown — %s", name, nonEmpty(reason, "server did not classify this section")))
	case libraryhealthv1.SectionState_SECTION_STATE_PARTIAL:
		rows = append(rows, fmt.Sprintf("%s: partial — %s", name, nonEmpty(reason, "no reason supplied")))
	case libraryhealthv1.SectionState_SECTION_STATE_EMPTY:
		rows = append(rows, fmt.Sprintf("%s: empty (measured zero)%s", name, reasonSuffix(reason)))
	case libraryhealthv1.SectionState_SECTION_STATE_AVAILABLE:
		rows = append(rows, fmt.Sprintf("%s: available%s", name, reasonSuffix(reason)))
	default:
		return append(rows, fmt.Sprintf("%s: unknown — %s", name, nonEmpty(reason, "unrecognized section state")))
	}
	for _, metric := range section.GetMetrics() {
		if metric == nil {
			continue
		}
		line := fmt.Sprintf("  %s: %d", nonEmpty(metric.GetLabel(), metric.GetKey()), metric.GetValue())
		if metric.GetTotal() != 0 {
			line += fmt.Sprintf("/%d", metric.GetTotal())
		}
		if detail := strings.TrimSpace(metric.GetDetail()); detail != "" {
			line += " — " + detail
		}
		rows = append(rows, line)
	}
	return rows
}

// countsAreUsable gates the composed counts block on the overview section's
// availability, so an unavailable overview never renders a row of zeros.
func countsAreUsable(overview *libraryhealthv1.HealthSection) bool {
	if overview == nil {
		return false
	}
	switch overview.GetState() {
	case libraryhealthv1.SectionState_SECTION_STATE_AVAILABLE,
		libraryhealthv1.SectionState_SECTION_STATE_EMPTY,
		libraryhealthv1.SectionState_SECTION_STATE_PARTIAL:
		return true
	default:
		return false
	}
}

func sectionStateLabel(section *libraryhealthv1.HealthSection) string {
	if section == nil {
		return "unavailable"
	}
	switch section.GetState() {
	case libraryhealthv1.SectionState_SECTION_STATE_AVAILABLE:
		return "available"
	case libraryhealthv1.SectionState_SECTION_STATE_EMPTY:
		return "empty"
	case libraryhealthv1.SectionState_SECTION_STATE_PARTIAL:
		return "partial"
	case libraryhealthv1.SectionState_SECTION_STATE_UNAVAILABLE:
		return "unavailable"
	default:
		return "unknown"
	}
}

func reasonSuffix(reason string) string {
	if reason == "" {
		return ""
	}
	return " — " + reason
}

func parseNonNegativeInt32(raw, flag string) (int32, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer (got %q)", flag, raw)
	}
	return int32(value), nil
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
