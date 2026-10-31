package coverage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/incidents"
)

// IncidentLister is the slice of the incident store the reader needs.
type IncidentLister interface {
	ListIncidents(ctx context.Context, filters incidents.ListFilters) (*incidents.ListResponse, error)
}

// BaseURLResolver answers notification-hub's API base at read time, so a hub
// that starts after autoheal is found on the next run.
type BaseURLResolver func(context.Context) (string, error)

const (
	deliveryProjectionPath   = "/api/v1/integrations/deliveries"
	deliveryProjectionPrefix = "incident."
	deliveryProjectionLimit  = 200
	deliveryReadTimeout      = 10 * time.Second
)

// NotificationHubDeliveryReader joins autoheal's open critical incidents to
// notification-hub's durable delivery projection. The join key is the dedupe
// key the hub records for every incident event: "<event type>:<incident id>".
// An unreachable hub or store is an error, which the check reports as
// undetermined; it never reads as "everything delivered".
func NotificationHubDeliveryReader(lister IncidentLister, resolve BaseURLResolver, client *http.Client) DeliveryReader {
	if client == nil {
		client = &http.Client{Timeout: deliveryReadTimeout}
	}
	return func(ctx context.Context) (DeliverySnapshot, error) {
		if lister == nil {
			return DeliverySnapshot{}, fmt.Errorf("incident store is not configured")
		}
		if resolve == nil {
			return DeliverySnapshot{}, fmt.Errorf("notification-hub base URL resolver is not configured")
		}
		listed, err := lister.ListIncidents(ctx, incidents.ListFilters{Status: incidents.StatusOpen, Severity: incidents.SeverityCritical, Limit: deliveryProjectionLimit})
		if err != nil {
			return DeliverySnapshot{}, fmt.Errorf("list critical incidents: %w", err)
		}
		snapshot := DeliverySnapshot{}
		incidentIDs := make([]string, 0, len(listed.Incidents))
		incidentSet := make(map[string]struct{}, len(listed.Incidents))
		for _, incident := range listed.Incidents {
			snapshot.Incidents = append(snapshot.Incidents, CriticalFinding{ID: incident.ID, Check: firstCheck(incident), Title: incident.Title})
			if id := strings.TrimSpace(incident.ID); id != "" {
				if _, exists := incidentSet[id]; !exists {
					incidentSet[id] = struct{}{}
					incidentIDs = append(incidentIDs, id)
				}
			}
		}
		if len(listed.Incidents) >= deliveryProjectionLimit {
			snapshot.Incomplete = true
			snapshot.IncompleteReasons = append(snapshot.IncompleteReasons, "critical incident sample reached its 200-item limit")
		}
		base, err := resolve(ctx)
		if err != nil {
			return DeliverySnapshot{}, fmt.Errorf("resolve notification-hub: %w", err)
		}
		query := url.Values{"prefix": {deliveryProjectionPrefix}, "limit": {fmt.Sprint(deliveryProjectionLimit)}}
		for _, id := range incidentIDs {
			query.Add("incident_id", id)
		}
		endpoint := strings.TrimRight(base, "/") + deliveryProjectionPath + "?" + query.Encode()
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return DeliverySnapshot{}, err
		}
		response, err := client.Do(request)
		if err != nil {
			return DeliverySnapshot{}, fmt.Errorf("read notification-hub delivery projection: %w", err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return DeliverySnapshot{}, fmt.Errorf("notification-hub delivery projection returned %s", response.Status)
		}
		var body struct {
			Truncated     *bool `json:"truncated"`
			Notifications []struct {
				DedupeKey          string `json:"dedupe_key"`
				AutohealIncidentID string `json:"autoheal_incident_id"`
				Attempts           []struct {
					Channel string `json:"channel"`
					Outcome string `json:"outcome"`
				} `json:"attempts"`
			} `json:"notifications"`
		}
		if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
			return DeliverySnapshot{}, fmt.Errorf("decode notification-hub delivery projection: %w", err)
		}
		if body.Truncated == nil {
			snapshot.Incomplete = true
			snapshot.IncompleteReasons = append(snapshot.IncompleteReasons, "notification-hub delivery projection did not report completeness")
		} else if *body.Truncated {
			snapshot.Incomplete = true
			snapshot.IncompleteReasons = append(snapshot.IncompleteReasons, "notification-hub delivery projection reached its 200-item limit")
		}
		for _, notification := range body.Notifications {
			ids := []string{IncidentIDFromDedupeKey(notification.DedupeKey), strings.TrimSpace(notification.AutohealIncidentID)}
			seen := make(map[string]struct{}, len(ids))
			for _, incidentID := range ids {
				if incidentID == "" {
					continue
				}
				if _, requested := incidentSet[incidentID]; !requested {
					continue
				}
				if _, duplicate := seen[incidentID]; duplicate {
					continue
				}
				seen[incidentID] = struct{}{}
				for _, attempt := range notification.Attempts {
					snapshot.Attempts = append(snapshot.Attempts, DeliveryAttempt{IncidentID: incidentID, Outcome: attempt.Outcome, Channel: attempt.Channel})
				}
			}
		}
		return snapshot, nil
	}
}

// IncidentIDFromDedupeKey extracts the incident id notification-hub keeps in
// its dedupe key ("incident.opened.v1:<id>"); "" when the key is not one.
func IncidentIDFromDedupeKey(key string) string {
	if !strings.HasPrefix(key, deliveryProjectionPrefix) {
		return ""
	}
	_, id, ok := strings.Cut(key, ":")
	if !ok {
		return ""
	}
	return strings.TrimSpace(id)
}

func firstCheck(incident incidents.Incident) string {
	if len(incident.SourceCheckIDs) > 0 {
		return incident.SourceCheckIDs[0]
	}
	return ""
}
