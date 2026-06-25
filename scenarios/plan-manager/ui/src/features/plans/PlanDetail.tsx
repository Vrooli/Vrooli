import { useState } from "react";
import { Archive } from "lucide-react";

import { AsyncBoundary } from "../../components/AsyncBoundary";
import { StatusBadge } from "../../components/StatusBadge";
import { Card, MetaRow, SectionPanel } from "../../components/Surfaces";
import { Button } from "../../components/ui/button";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { phaseStatusDescriptor, planStatusDescriptor, stalenessDescriptor } from "../../lib/planStatus";
import type {
  Phase,
  Plan,
  PlanEdge,
  RegressionAnchor,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";
import { useArchivePlan, usePlanDetail, usePlanGraph, usePlanMarkdown } from "./usePlans";

/** Render a list of strings as a bulleted block, or the empty fallback. */
function StringList({ items, empty }: { items: readonly string[]; empty: string }) {
  if (items.length === 0) {
    return <p className="text-xs text-app-muted-foreground">{empty}</p>;
  }
  return (
    <ul className="flex flex-col gap-1">
      {items.map((item, i) => (
        <li key={`${item}-${i}`} className="break-words font-mono text-xs text-app-foreground">
          {item}
        </li>
      ))}
    </ul>
  );
}

function PhaseCard({ phase }: { phase: Phase }) {
  const { t } = useTranslation();
  return (
    <li
      data-testid={selectors.plans.phase({ id: phase.id })}
      className="rounded-panel border border-app-border bg-app-surface-muted p-3"
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h4 className="text-sm font-semibold text-app-foreground">
          {phase.order}. {phase.title}
        </h4>
        <StatusBadge descriptor={phaseStatusDescriptor(phase.status)} />
      </div>
      {phase.intent ? (
        <p className="mt-2 text-sm text-app-muted-foreground">{phase.intent}</p>
      ) : null}
      <dl className="mt-3 flex flex-col gap-2">
        {phase.acceptance ? (
          <MetaRow term={t(strings.pages.plans.detail.phaseAcceptance)}>{phase.acceptance}</MetaRow>
        ) : null}
        {phase.requiredReading.length > 0 ? (
          <MetaRow term={t(strings.pages.plans.detail.phaseRequiredReading)}>
            <StringList items={phase.requiredReading} empty={t(strings.common.none)} />
          </MetaRow>
        ) : null}
        {phase.reminders.length > 0 ? (
          <MetaRow term={t(strings.pages.plans.detail.phaseReminders)}>
            <StringList items={phase.reminders} empty={t(strings.common.none)} />
          </MetaRow>
        ) : null}
        {phase.baselineScope.length > 0 ? (
          <MetaRow term={t(strings.pages.plans.detail.phaseBaselineScope)}>
            <StringList items={phase.baselineScope} empty={t(strings.common.none)} />
          </MetaRow>
        ) : null}
      </dl>
    </li>
  );
}

function AnchorView({ anchor }: { anchor: RegressionAnchor }) {
  const { t } = useTranslation();
  return (
    <dl className="flex flex-col gap-2">
      {anchor.unavailable ? (
        <p className="rounded-control bg-app-warning/10 px-3 py-2 text-xs text-app-warning">
          {t(strings.pages.plans.detail.anchorUnavailable)}
        </p>
      ) : null}
      {anchor.strategy ? (
        <MetaRow term={t(strings.pages.plans.detail.anchorStrategy)}>{anchor.strategy}</MetaRow>
      ) : null}
      {anchor.scenario ? (
        <MetaRow term={t(strings.pages.plans.detail.anchorScenario)}>{anchor.scenario}</MetaRow>
      ) : null}
      {anchor.baselineName ? (
        <MetaRow term={t(strings.pages.plans.detail.anchorBaseline)}>{anchor.baselineName}</MetaRow>
      ) : null}
      {anchor.headSha ? (
        <MetaRow term={t(strings.pages.plans.detail.anchorHeadSha)}>
          <span className="font-mono">{anchor.headSha}</span>
        </MetaRow>
      ) : null}
      {anchor.allowlistPaths.length > 0 ? (
        <MetaRow term={t(strings.pages.plans.detail.anchorAllowlist)}>
          <StringList items={anchor.allowlistPaths} empty={t(strings.common.none)} />
        </MetaRow>
      ) : null}
      {anchor.commands.length > 0 ? (
        <MetaRow term={t(strings.pages.plans.detail.anchorCommands)}>
          <StringList items={anchor.commands} empty={t(strings.common.none)} />
        </MetaRow>
      ) : null}
    </dl>
  );
}

function GraphView({ planId, edges }: { planId: string; edges: PlanEdge[] }) {
  const { t } = useTranslation();
  const supersedes = edges.filter((e) => e.fromPlanId === planId && e.kind === "supersedes");
  const supersededBy = edges.filter((e) => e.toPlanId === planId && e.kind === "supersedes");

  if (edges.length === 0) {
    return <p className="text-xs text-app-muted-foreground">{t(strings.pages.plans.detail.edgeNone)}</p>;
  }
  return (
    <div data-testid={selectors.plans.detailGraph} className="flex flex-col gap-3 text-sm">
      {supersedes.length > 0 ? (
        <div>
          <p className="text-xs uppercase tracking-wide text-app-muted-foreground">
            {t(strings.pages.plans.detail.supersedes)}
          </p>
          <StringList items={supersedes.map((e) => e.toPlanId)} empty={t(strings.common.none)} />
        </div>
      ) : null}
      {supersededBy.length > 0 ? (
        <div>
          <p className="text-xs uppercase tracking-wide text-app-muted-foreground">
            {t(strings.pages.plans.detail.supersededBy)}
          </p>
          <StringList items={supersededBy.map((e) => e.fromPlanId)} empty={t(strings.common.none)} />
        </div>
      ) : null}
    </div>
  );
}

function PlanBody({ plan }: { plan: Plan }) {
  const { t } = useTranslation();
  const [showMarkdown, setShowMarkdown] = useState(false);
  const markdown = usePlanMarkdown(plan.id, showMarkdown);
  const graph = usePlanGraph(plan.id);
  const archive = useArchivePlan();

  return (
    <section
      data-testid={selectors.pages.planDetail}
      aria-labelledby="plan-detail-heading"
      className="flex flex-col gap-4"
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex flex-col gap-1">
          <h2 id="plan-detail-heading" className="text-2xl font-semibold text-app-foreground">
            {plan.title}
          </h2>
          <div className="flex flex-wrap items-center gap-2 text-xs text-app-muted-foreground">
            <StatusBadge descriptor={planStatusDescriptor(plan.status)} />
            {plan.slug ? <span className="font-mono">{plan.slug}</span> : null}
            {plan.updatedAt ? (
              <span>
                {t(strings.pages.plans.updatedLabel)}{" "}
                {formatDate(new Date(plan.updatedAt), { dateStyle: "medium", timeStyle: "short" })}
              </span>
            ) : null}
          </div>
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          data-testid={selectors.plans.archiveButton}
          disabled={archive.isPending}
          onClick={() => archive.mutate(plan.id)}
        >
          <Archive aria-hidden="true" className="me-2 h-4 w-4" />
          {t(strings.pages.plans.archive)}
        </Button>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        {plan.purpose ? (
          <SectionPanel title={t(strings.pages.plans.detail.purposeHeading)} headingId="plan-purpose">
            <p className="whitespace-pre-wrap text-sm text-app-foreground">{plan.purpose}</p>
          </SectionPanel>
        ) : null}
        {plan.scope ? (
          <SectionPanel title={t(strings.pages.plans.detail.scopeHeading)} headingId="plan-scope">
            <p className="whitespace-pre-wrap text-sm text-app-foreground">{plan.scope}</p>
          </SectionPanel>
        ) : null}
        {plan.constraints ? (
          <SectionPanel title={t(strings.pages.plans.detail.constraintsHeading)} headingId="plan-constraints">
            <p className="whitespace-pre-wrap text-sm text-app-foreground">{plan.constraints}</p>
          </SectionPanel>
        ) : null}
        {plan.nonGoals ? (
          <SectionPanel title={t(strings.pages.plans.detail.nonGoalsHeading)} headingId="plan-nongoals">
            <p className="whitespace-pre-wrap text-sm text-app-foreground">{plan.nonGoals}</p>
          </SectionPanel>
        ) : null}
      </div>

      {plan.definitionOfDone ? (
        <SectionPanel title={t(strings.pages.plans.detail.dodHeading)} headingId="plan-dod">
          <p className="whitespace-pre-wrap text-sm text-app-foreground">{plan.definitionOfDone}</p>
        </SectionPanel>
      ) : null}

      <SectionPanel title={t(strings.pages.plans.detail.phasesHeading)} headingId="plan-phases">
        {plan.phases.length === 0 ? (
          <p className="text-sm text-app-muted-foreground">{t(strings.pages.plans.detail.noPhases)}</p>
        ) : (
          <ol data-testid={selectors.plans.detailPhases} className="flex flex-col gap-2">
            {plan.phases.map((phase) => (
              <PhaseCard key={phase.id} phase={phase} />
            ))}
          </ol>
        )}
      </SectionPanel>

      <div className="grid gap-4 lg:grid-cols-2">
        <SectionPanel title={t(strings.pages.plans.detail.referencesHeading)} headingId="plan-references">
          {plan.references.length === 0 ? (
            <p className="text-sm text-app-muted-foreground">
              {t(strings.pages.plans.detail.noReferences)}
            </p>
          ) : (
            <ul className="flex flex-col gap-2">
              {plan.references.map((ref) => (
                <li key={ref.id} className="flex flex-wrap items-center gap-2 text-sm">
                  <span className="break-all font-mono text-xs text-app-foreground">{ref.target}</span>
                  <StatusBadge descriptor={stalenessDescriptor(ref.staleness)} />
                </li>
              ))}
            </ul>
          )}
        </SectionPanel>

        <SectionPanel title={t(strings.pages.plans.detail.anchorHeading)} headingId="plan-anchor">
          {plan.regressionAnchor ? (
            <AnchorView anchor={plan.regressionAnchor} />
          ) : (
            <p className="text-sm text-app-muted-foreground">
              {t(strings.pages.plans.detail.anchorNone)}
            </p>
          )}
        </SectionPanel>
      </div>

      <SectionPanel title={t(strings.pages.plans.detail.graphHeading)} headingId="plan-graph">
        <AsyncBoundary
          isLoading={graph.isLoading}
          error={graph.error}
          testIdPrefix={selectors.plans.detailGraph}
        >
          <GraphView planId={plan.id} edges={graph.data ?? []} />
        </AsyncBoundary>
      </SectionPanel>

      <SectionPanel
        title={t(strings.pages.plans.detail.markdownHeading)}
        headingId="plan-markdown"
        actions={
          <Button
            type="button"
            variant="outline"
            size="sm"
            data-testid={selectors.plans.detailMarkdownToggle}
            aria-expanded={showMarkdown}
            onClick={() => setShowMarkdown((v) => !v)}
          >
            {showMarkdown
              ? t(strings.pages.plans.detail.hideMarkdown)
              : t(strings.pages.plans.detail.showMarkdown)}
          </Button>
        }
      >
        {showMarkdown ? (
          <AsyncBoundary
            isLoading={markdown.isLoading}
            error={markdown.error}
            testIdPrefix={selectors.plans.detailMarkdown}
          >
            <Card className="bg-app-surface-muted p-0">
              <pre
                data-testid={selectors.plans.detailMarkdown}
                className="max-h-[40rem] overflow-auto whitespace-pre-wrap break-words p-4 font-mono text-xs text-app-foreground"
              >
                {markdown.data ?? ""}
              </pre>
            </Card>
          </AsyncBoundary>
        ) : null}
      </SectionPanel>
    </section>
  );
}

/**
 * PlanDetail — the read view for a single plan: metadata sections, phases with
 * status + references + reminders, the regression anchor, the supersession
 * graph, and a lazy rendered-markdown view (fetched only when expanded).
 */
export function PlanDetail({ planId }: { planId: string }) {
  const { t } = useTranslation();
  const plan = usePlanDetail(planId);

  return (
    <AsyncBoundary
      isLoading={plan.isLoading}
      error={plan.error}
      isEmpty={!plan.isLoading && !plan.error && !plan.data}
      testIdPrefix={selectors.pages.planDetail}
      emptyLabel={t(strings.pages.plans.detail.notFound)}
    >
      {plan.data ? <PlanBody plan={plan.data} /> : null}
    </AsyncBoundary>
  );
}
