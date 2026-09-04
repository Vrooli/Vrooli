// Friction routing turns recurring, bounded evidence into durable
// meta-optimization findings with explicit honesty and idempotency rules.
package orchestration

import (
	"context"
	"fmt"
	"strings"

	"agent-manager/internal/findings"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/promptmanager"
	"agent-manager/internal/runsignal"

	"github.com/google/uuid"
)

type RecurringFrictionSummary struct {
	Candidates int `json:"candidates"`
	Filed      int `json:"filed"`
	Skipped    int `json:"skipped"`
	Withheld   int `json:"withheld"`
	Cap        int `json:"cap"`
}

// PublishRecurringFriction files only fingerprints seen in three distinct
// runs. Findings provide the durable idempotency marker; the caller can run
// this operation from a scheduler without duplicating inbox entries.
func (o *Orchestrator) PublishRecurringFriction(ctx context.Context, filter invocationreadmodel.Filter, cap int) (RecurringFrictionSummary, error) {
	if cap <= 0 {
		cap = 25
	}
	projection, ok := o.invocationReadModel.(invocationreadmodel.ProjectionStore)
	if !ok {
		return RecurringFrictionSummary{}, fmt.Errorf("recurring friction requires a projection store")
	}
	publisher, ok := o.promptClient.(promptmanager.FrictionIntakeClient)
	if !ok || publisher == nil {
		return RecurringFrictionSummary{}, fmt.Errorf("friction intake is unavailable")
	}
	if o.findings == nil {
		return RecurringFrictionSummary{}, fmt.Errorf("finding store is unavailable")
	}
	episodes, err := projection.Episodes(ctx, filter, invocationEvidenceLimit)
	if err != nil {
		return RecurringFrictionSummary{}, err
	}
	type recurrence struct {
		runs    map[string]bool
		count   int
		cost    int64
		episode runsignal.FrictionEpisode
	}
	groups := map[string]*recurrence{}
	for _, item := range episodes {
		group := groups[item.Fingerprint]
		if group == nil {
			fact := item.FrictionEpisode
			fact.RunID = item.RunID
			group = &recurrence{runs: map[string]bool{}, episode: fact}
			groups[item.Fingerprint] = group
		}
		group.runs[item.RunID] = true
		group.count++
		group.cost += item.WallClockMS
	}
	result := RecurringFrictionSummary{Cap: cap}
	for fingerprint, group := range groups {
		if len(group.runs) < 3 {
			continue
		}
		result.Candidates++
		prior, listErr := o.findings.List(ctx, findings.Filter{Fingerprint: fingerprint, Limit: 10})
		if listErr != nil {
			return result, listErr
		}
		alreadyFiled := false
		for _, item := range prior {
			if strings.TrimSpace(item.FrictionTopic) != "" {
				alreadyFiled = true
				break
			}
		}
		if alreadyFiled {
			result.Skipped++
			continue
		}
		if result.Filed >= cap {
			result.Withheld++
			continue
		}
		runID, parseErr := uuid.Parse(group.episode.RunID)
		if parseErr != nil {
			result.Skipped++
			continue
		}
		investigationID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("agent-manager/recurring/"+fingerprint))
		evidence := fmt.Sprintf("Fingerprint %s appeared %d times across %d distinct runs and cost %d ms.", fingerprint, group.count, len(group.runs), group.cost)
		finding := &findings.Finding{RunID: runID, InvestigationRunID: investigationID, Category: group.episode.CauseScope, Severity: "recurring", Recommendation: "Review the recurring deterministic friction fingerprint and its suspected owner.", Evidence: evidence, TargetPath: group.episode.SuspectedOwnerCommand, Fingerprint: fingerprint}
		if err := o.findings.Create(ctx, finding); err != nil {
			return result, err
		}
		topic, publishErr := publisher.PublishFriction(ctx, promptmanager.FrictionReport{InvestigationRunID: investigationID.String(), Fingerprint: fingerprint, Category: group.episode.CauseScope, Severity: "recurring", Occurrences: len(group.runs), Recommendation: finding.Recommendation, Evidence: evidence, TargetPath: group.episode.SuspectedOwnerCommand, HonestyFlags: []string{"auto-generated"}, RecurrenceEvidence: fmt.Sprintf("distinct_runs=%d", len(group.runs))})
		if publishErr != nil {
			result.Skipped++
			continue
		}
		before := float64(len(group.runs))
		if err := o.findings.SetEffectiveness(ctx, finding.ID, &before, nil, "not_yet_measurable", topic); err != nil {
			return result, err
		}
		result.Filed++
	}
	return result, nil
}
