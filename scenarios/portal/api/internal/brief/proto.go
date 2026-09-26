package brief

import (
	agentbrief "github.com/vrooli/agentbrief-go"
	briefv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/brief"
)

func ToProto(record Record) *briefv1.Brief {
	items := make([]*briefv1.BriefItem, 0, len(record.Items))
	for _, item := range record.Items {
		items = append(items, &briefv1.BriefItem{ProviderId: item.ProviderID, Type: item.Type, Title: item.Title, Snippet: item.Snippet, Path: item.Path, Score: item.Score, RerankScore: item.RerankScore, TrustClass: trustProto(item.TrustClass), SuggestedCommand: item.SuggestedCommand})
	}
	return &briefv1.Brief{Id: record.ID, Consumer: consumerProto(record.Consumer), Verdict: verdictProto(record.Verdict), Reason: record.Reason, Items: items, Rendered: record.Rendered, LatencyMs: record.LatencyMS, Degraded: record.Degraded, MaxTrustClass: trustProto(record.MaxTrustClass), CreatedAt: record.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"), EffectiveQuery: record.EffectiveQuery, QueriedProviders: append([]string(nil), record.QueriedProviders...)}
}

func StatsToProto(rows []StatsRow) []*briefv1.BriefStatsRow {
	out := make([]*briefv1.BriefStatsRow, 0, len(rows))
	for _, row := range rows {
		withheld := map[string]int64{}
		for verdict, count := range row.WithheldByVerdict {
			withheld[verdictProto(verdict).String()] = count
		}
		out = append(out, &briefv1.BriefStatsRow{Consumer: consumerProto(row.Consumer), BriefsBuilt: row.BriefsBuilt, BriefsDelivered: row.BriefsDelivered, ItemsDelivered: row.ItemsDelivered, ItemsUsed: row.ItemsUsed, UsageRate: row.UsageRate, WithheldRate: row.WithheldRate, WithheldByVerdict: withheld})
	}
	return out
}

func ConsumerFromProto(value briefv1.BriefConsumer) agentbrief.Consumer {
	switch value {
	case briefv1.BriefConsumer_BRIEF_CONSUMER_PORTAL_LLM:
		return agentbrief.ConsumerPortalLLM
	case briefv1.BriefConsumer_BRIEF_CONSUMER_PORTAL_AGENT:
		return agentbrief.ConsumerPortalAgent
	default:
		return agentbrief.ConsumerExternalHarness
	}
}

func ConsumerFilterFromProto(value briefv1.BriefConsumer) agentbrief.Consumer {
	switch value {
	case briefv1.BriefConsumer_BRIEF_CONSUMER_PORTAL_LLM:
		return agentbrief.ConsumerPortalLLM
	case briefv1.BriefConsumer_BRIEF_CONSUMER_PORTAL_AGENT:
		return agentbrief.ConsumerPortalAgent
	case briefv1.BriefConsumer_BRIEF_CONSUMER_EXTERNAL_HARNESS:
		return agentbrief.ConsumerExternalHarness
	default:
		return ""
	}
}
func consumerProto(value agentbrief.Consumer) briefv1.BriefConsumer {
	switch value {
	case agentbrief.ConsumerPortalLLM:
		return briefv1.BriefConsumer_BRIEF_CONSUMER_PORTAL_LLM
	case agentbrief.ConsumerPortalAgent:
		return briefv1.BriefConsumer_BRIEF_CONSUMER_PORTAL_AGENT
	default:
		return briefv1.BriefConsumer_BRIEF_CONSUMER_EXTERNAL_HARNESS
	}
}
func verdictProto(value agentbrief.Verdict) briefv1.BriefVerdict {
	switch value {
	case agentbrief.VerdictDeliver:
		return briefv1.BriefVerdict_BRIEF_VERDICT_DELIVER
	case agentbrief.VerdictWithheldLowConfidence:
		return briefv1.BriefVerdict_BRIEF_VERDICT_WITHHELD_LOW_CONFIDENCE
	case agentbrief.VerdictWithheldModeOff:
		return briefv1.BriefVerdict_BRIEF_VERDICT_WITHHELD_MODE_OFF
	case agentbrief.VerdictWithheldBudget:
		return briefv1.BriefVerdict_BRIEF_VERDICT_WITHHELD_BUDGET
	case agentbrief.VerdictWithheldDegraded:
		return briefv1.BriefVerdict_BRIEF_VERDICT_WITHHELD_DEGRADED
	default:
		return briefv1.BriefVerdict_BRIEF_VERDICT_WITHHELD_NOT_APPLICABLE
	}
}
func trustProto(value agentbrief.Class) briefv1.TrustClass {
	switch value {
	case agentbrief.ClassFirstParty:
		return briefv1.TrustClass_TRUST_CLASS_FIRST_PARTY
	case agentbrief.ClassQuoted:
		return briefv1.TrustClass_TRUST_CLASS_QUOTED
	case agentbrief.ClassExternal:
		return briefv1.TrustClass_TRUST_CLASS_EXTERNAL
	default:
		return briefv1.TrustClass_TRUST_CLASS_UNSPECIFIED
	}
}
