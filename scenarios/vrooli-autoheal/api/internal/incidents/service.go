package incidents

import (
	"context"
	"fmt"
	"strings"
	"time"
	"vrooli-autoheal/internal/checks"
)

type Store interface {
	UpsertIncident(ctx context.Context, input UpsertInput) (*Incident, error)
	ListIncidents(ctx context.Context, filters ListFilters) (*ListResponse, error)
	GetIncident(ctx context.Context, id string) (*Incident, error)
	ListIncidentObservations(ctx context.Context, incidentID string, limit int) ([]Observation, error)
	UpdateIncidentStatus(ctx context.Context, incidentID string, status Status, note string) (*Incident, error)
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	return &Service{store: store, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) UpsertFromCheckResult(ctx context.Context, result checks.Result) (*Incident, bool, error) {
	rule, ok := classifyResult(result)
	if !ok {
		return nil, false, nil
	}
	observedAt := result.Timestamp
	if observedAt.IsZero() {
		observedAt = s.now()
	}
	input := UpsertInput{
		Fingerprint:     rule.fingerprint,
		Type:            rule.incidentType,
		Severity:        rule.severity,
		Title:           rule.title,
		Summary:         result.Message,
		ObservedAt:      observedAt.UTC(),
		BootID:          stringDetail(result.Details, "bootId"),
		SourceCheckID:   result.CheckID,
		Evidence:        boundedEvidence(result.Details),
		Recommendations: stringSliceDetail(result.Details, "recommendations"),
	}
	incident, err := s.store.UpsertIncident(ctx, input)
	if err != nil {
		return nil, true, err
	}
	return incident, true, nil
}

func (s *Service) UpsertFromCheckResults(ctx context.Context, results []checks.Result) (int, error) {
	count := 0
	for _, result := range results {
		_, created, err := s.UpsertFromCheckResult(ctx, result)
		if err != nil {
			return count, err
		}
		if created {
			count++
		}
	}
	return count, nil
}

type incidentRule struct {
	incidentType Type
	severity     Severity
	title        string
	fingerprint  string
}

func classifyResult(result checks.Result) (incidentRule, bool) {
	if result.Status == checks.StatusOK {
		return incidentRule{}, false
	}
	severity := SeverityWarning
	if result.Status == checks.StatusCritical {
		severity = SeverityCritical
	}
	if strings.HasPrefix(result.CheckID, "host-") {
		return incidentRule{
			incidentType: TypeHostIntegrity,
			severity:     severity,
			title:        "Host integrity issue detected",
			fingerprint:  Fingerprint(string(TypeHostIntegrity), result.CheckID, evidenceDimension(result.Details)),
		}, true
	}
	switch result.CheckID {
	case "system-boot-history":
		return incidentRule{
			incidentType: TypeUncleanBoot,
			severity:     severity,
			title:        "Unclean boot history detected",
			fingerprint:  Fingerprint(string(TypeUncleanBoot), result.CheckID, stringDetail(result.Details, "latestUncleanBootId")),
		}, true
	case "system-pstore-evidence":
		return incidentRule{incidentType: TypeUncleanBoot, severity: severity, title: "Kernel crash evidence detected", fingerprint: Fingerprint(string(TypeUncleanBoot), result.CheckID)}, true
	case "system-mce-recent":
		return incidentRule{incidentType: TypeHostIntegrity, severity: severity, title: "Recent machine-check evidence detected", fingerprint: Fingerprint(string(TypeHostIntegrity), result.CheckID)}, true
	default:
		return incidentRule{}, false
	}
}

func boundedEvidence(details map[string]any) map[string]any {
	if details == nil {
		return nil
	}
	out := map[string]any{}
	for key, value := range details {
		if key == "recommendations" {
			continue
		}
		out[key] = value
	}
	return out
}

func stringDetail(details map[string]any, key string) string {
	if details == nil {
		return ""
	}
	if value, ok := details[key].(string); ok {
		return value
	}
	return ""
}

func stringSliceDetail(details map[string]any, key string) []string {
	if details == nil {
		return nil
	}
	raw, ok := details[key]
	if !ok {
		return nil
	}
	switch values := raw.(type) {
	case []string:
		return values
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if s, ok := value.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func evidenceDimension(details map[string]any) string {
	if details == nil {
		return ""
	}
	if fp := stringDetail(details, "inventoryFingerprint"); fp != "" {
		return fp
	}
	if kernel, ok := details["kernel"]; ok {
		return fmt.Sprintf("%v", kernel)
	}
	return ""
}
