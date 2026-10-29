import { Link } from "react-router-dom";
import { Lightbulb } from "lucide-react";
import { Card } from "@vrooli/react-component-library/Card/1";
import { Skeleton } from "@vrooli/react-component-library/Skeleton/1";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import type { UsageEvidenceRef, UsageRecommendation, UsageRuleStatus } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { DataState } from "../../components/feedback/DataState";
import { useUsageRecommendations } from "./api";
import { confidenceTone, evidenceText, formatSaving, formatServerText, formatTokenCount, leverLabel, metricLabel, windowTitle } from "./format";

function metricValue(value: number, unit: string): string {
  if (unit === "tokens") return formatTokenCount(Math.round(value));
  if (unit === "ratio") return `${Math.round(value * 100)}%`;
  if (unit === "points") return `${value.toFixed(1)} pts`;
  return `${Number.isInteger(value) ? value : value.toFixed(1)} ${unit}`;
}

/** One evidence row: its kind, then text that tells it apart from its siblings (label and short id). */
function EvidenceRow({ evidence }: { evidence: UsageEvidenceRef }) {
  const text = evidenceText(evidence);
  return (
    <li className="flex min-h-11 min-w-0 items-center gap-2 border-b border-border last:border-b-0 sm:min-h-8" data-testid="usage-evidence">
      <span className="w-16 shrink-0 text-xs text-muted-foreground">{evidence.kind}</span>
      {evidence.kind === "run" ? (
        <Link className="flex min-h-11 min-w-0 flex-1 items-center truncate text-primary underline-offset-2 hover:underline sm:min-h-8" to={`/runs/${encodeURIComponent(evidence.id)}?tab=usage`}>{text}</Link>
      ) : (
        <span className="min-w-0 flex-1 truncate" title={evidence.id}>{text}</span>
      )}
    </li>
  );
}

function RecommendationRow({ recommendation, open, siblings }: { recommendation: UsageRecommendation; open: boolean; siblings: UsageRecommendation[] }) {
  return (
    <details open={open} className="group border-t border-border py-3 first:border-t-0" data-testid="usage-recommendation">
      <summary className="flex min-h-11 cursor-pointer list-none flex-wrap items-center gap-x-3 gap-y-1 sm:min-h-8">
        <span aria-hidden="true" className="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-primary/10 text-sm font-bold text-primary">{recommendation.rank}</span>
        <span className="min-w-0 flex-[1_1_16rem] font-semibold"><span className="sr-only">Recommendation {recommendation.rank}: </span>{recommendation.action}</span>
        <StatusBadge tone="neutral">{leverLabel(recommendation.lever)}</StatusBadge>
        <span className="font-bold tabular-nums text-success">{formatSaving(recommendation.estWeeklyQuotaPointsSaved)}</span>
        <StatusBadge tone={confidenceTone(recommendation.confidence)}>{recommendation.confidence} confidence</StatusBadge>
      </summary>
      <div className="mt-3 grid gap-4 text-sm sm:ml-10 md:grid-cols-2">
        <div className="space-y-2">
          <h4 className="text-xs font-bold uppercase tracking-wide text-muted-foreground">Why</h4>
          <p>{formatServerText(recommendation.rationale)}</p>
          {recommendation.metrics.length > 0 ? (
            <dl className="grid grid-cols-2 gap-x-4 gap-y-1">
              {recommendation.metrics.map((metric) => (
                <div key={metric.name} className="contents">
                  <dt className="text-muted-foreground">{metricLabel(metric.name)}</dt>
                  <dd className="tabular-nums">{metricValue(metric.value, metric.unit)}</dd>
                </div>
              ))}
            </dl>
          ) : null}
        </div>
        <div className="space-y-2">
          <h4 className="text-xs font-bold uppercase tracking-wide text-muted-foreground">Evidence</h4>
          <p className="text-muted-foreground">
            {recommendation.targetLabel || recommendation.targetId} · {windowTitle(recommendation, siblings)} · {recommendation.consumerPoints.toFixed(1)} pts this window
          </p>
          <ul className="m-0 max-h-72 list-none overflow-y-auto p-0">
            {recommendation.evidenceRefs.map((evidence) => <EvidenceRow key={`${evidence.kind}/${evidence.id}`} evidence={evidence} />)}
          </ul>
        </div>
      </div>
    </details>
  );
}

/** Ranked recommendations with their evidence; families that did not fire say why. */
export function RecommendationList({ recommendations, rules = [] }: { recommendations: UsageRecommendation[]; rules?: UsageRuleStatus[] }) {
  return (
    <div>
      {recommendations.map((recommendation, index) => <RecommendationRow key={`${recommendation.rule}/${recommendation.targetId}`} recommendation={recommendation} open={index === 0} siblings={recommendations} />)}
      {rules.filter((rule) => !rule.fired).map((rule) => (
        <p key={rule.rule} className="border-t border-border py-2 text-xs text-muted-foreground" data-testid="usage-rule-not-fired">
          {rule.rule} did not fire: {rule.reason ? formatServerText(rule.reason) : "no candidates in this window"}
        </p>
      ))}
    </div>
  );
}

/** Shaped like ranked rows: number, action line, chips. */
export function RecommendationsSkeleton({ rows = 3 }: { rows?: number }) {
  return (
    <div>
      {Array.from({ length: rows }, (_, index) => (
        <div key={index} className="flex items-center gap-3 border-t border-border py-3 first:border-t-0">
          <Skeleton className="h-7 w-7 shrink-0 rounded-full" />
          <Skeleton className="h-4 min-w-0 flex-1" />
          <Skeleton className="hidden h-5 w-20 sm:block" />
          <Skeleton className="h-5 w-16" />
        </div>
      ))}
    </div>
  );
}

export function UsageRecommendations() {
  const query = useUsageRecommendations();
  const recommendations = query.data?.recommendations ?? [];
  const rules = query.data?.rules ?? [];
  return (
    <Card className="p-4 sm:p-5">
      <DataState
        compact
        subject="Recommendations"
        loading={query.isLoading}
        error={query.error}
        isEmpty={recommendations.length === 0 && rules.length === 0}
        skeleton={<RecommendationsSkeleton />}
        onRetry={() => void query.refetch()}
        empty={<EmptyState density="compact" icon={<Lightbulb aria-hidden="true" />} title="No recommendations yet" description="Recommendations need provider readings and indexed usage for the current window." />}
      >
        {recommendations.length === 0 ? <p className="pb-2 text-sm text-muted-foreground">No rule found a saving in the current window.</p> : null}
        <RecommendationList recommendations={recommendations} rules={rules} />
      </DataState>
    </Card>
  );
}
