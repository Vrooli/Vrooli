// Package planimport bridges authored plan-manager plans into the swarm-manager
// backlog: it fetches a plan read-only over the plan-manager Connect API and
// translates its phases into a provenance-stamped, dependency-ordered batch
// that lands via the existing atomic batch-create path. Per plan decision D3 it
// consumes plan-manager read-only — no plan-manager mutation, no proto change —
// and produces a linear depends_on chain (phases execute sequentially).
package planimport

import (
	"fmt"
	"sort"
	"strings"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

// importKind is the backlog kind imported plan phases land as. Plan phases are
// executable work, so they become execute items.
const importKind = "execute"

// BatchItem mirrors the swarm-manager batch-create item JSON so the payload
// this package produces lands unchanged through POST /api/v1/backlog/batch.
type BatchItem struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Kind        string   `json:"kind"`
	DependsOn   []string `json:"depends_on,omitempty"`
	SpawnedFrom string   `json:"spawned_from,omitempty"`
	Effort      string   `json:"effort,omitempty"`
}

// BatchPayload is the JSON body posted to the batch-create endpoint.
type BatchPayload struct {
	Items []BatchItem `json:"items"`
}

// Translate maps a plan's phases into a batch payload: one execute item per
// phase, chained by a linear depends_on in phase order (D3), each stamped with
// spawned_from = "plan-manager:<slug>/phase-<order>" and left unsized so the ETA
// global distribution applies until an operator sizes it. Phases are ordered by
// their Order field; ties keep their input order.
func Translate(plan *sharedv1.Plan) (BatchPayload, error) {
	if plan == nil {
		return BatchPayload{}, fmt.Errorf("planimport: nil plan")
	}
	slug := strings.TrimSpace(plan.GetSlug())
	if slug == "" {
		return BatchPayload{}, fmt.Errorf("planimport: plan has no slug")
	}
	phases := append([]*sharedv1.Phase(nil), plan.GetPhases()...)
	if len(phases) == 0 {
		return BatchPayload{}, fmt.Errorf("planimport: plan %q has no phases", slug)
	}
	sort.SliceStable(phases, func(i, j int) bool {
		return phases[i].GetOrder() < phases[j].GetOrder()
	})

	items := make([]BatchItem, 0, len(phases))
	var prevRef string
	for _, phase := range phases {
		order := phase.GetOrder()
		title := strings.TrimSpace(phase.GetTitle())
		if title == "" {
			title = fmt.Sprintf("Phase %d", order)
		}
		name := fmt.Sprintf("%s-phase-%d", slug, order)
		desc := strings.TrimSpace(phase.GetIntent())
		if desc == "" {
			desc = strings.TrimSpace(phase.GetAcceptance())
		}

		item := BatchItem{
			Name:        name,
			Title:       title,
			Description: desc,
			Kind:        importKind,
			SpawnedFrom: fmt.Sprintf("plan-manager:%s/phase-%d", slug, order),
		}
		if prevRef != "" {
			item.DependsOn = []string{prevRef}
		}
		items = append(items, item)
		prevRef = importKind + "/" + name
	}
	return BatchPayload{Items: items}, nil
}
