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
  RelevantContextItem,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";
import {
  RelevantContextKind,
  RelevantContextRepeatPolicy,
  RelevantContextStatus,
  WorkPosture,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";
import { useArchivePlan, usePlanDetail, usePlanGraph, usePlanMarkdown } from "./usePlans";

/** Map the autofilled work posture enum to a human label. */
function workPostureLabel(posture: WorkPosture): string {
  switch (posture) {
    case WorkPosture.GREENFIELD:
      return "greenfield";
    case WorkPosture.BROWNFIELD:
      return "brownfield";
    default:
      return "greenfield";
  }
}

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

const contextKindLabels: Record<RelevantContextKind, string> = {
  [RelevantContextKind.UNSPECIFIED]: "Context",
  [RelevantContextKind.SKILL]: "Skill",
  [RelevantContextKind.DOC]: "Doc",
  [RelevantContextKind.COMMAND]: "Command",
  [RelevantContextKind.SEARCH]: "Search",
  [RelevantContextKind.CODE_REF]: "Code",
  [RelevantContextKind.REQ_REF]: "Requirement",
  [RelevantContextKind.NOTE]: "Note",
};

function repeatLabel(policy: RelevantContextRepeatPolicy) {
  switch (policy) {
    case RelevantContextRepeatPolicy.ON_RESUME:
      return "on resume";
    case RelevantContextRepeatPolicy.EVERY_PHASE:
      return "every phase";
    case RelevantContextRepeatPolicy.PHASE_ENTRY:
      return "phase entry";
    case RelevantContextRepeatPolicy.AS_NEEDED:
      return "as needed";
    case RelevantContextRepeatPolicy.ONCE_PER_EXECUTION:
      return "once";
    default:
      return "";
  }
}

function contextCommand(item: RelevantContextItem) {
  if (item.argv.length > 0) return item.argv.join(" ");
  return item.command;
}

function RelevantContextList({ items }: { items: readonly RelevantContextItem[] }) {
  const { t } = useTranslation();
  if (items.length === 0) {
    return <p className="text-sm text-app-muted-foreground">{t(strings.pages.plans.detail.noContext)}</p>;
  }
  return (
    <ul data-testid={selectors.plans.relevantContext} className="flex flex-col gap-2">
      {items.map((item, i) => {
        const command = contextCommand(item);
        const repeat = repeatLabel(item.repeatPolicy);
        return (
          <li
            key={item.id || `${item.kind}-${item.label}-${i}`}
            className="rounded-control border border-app-border bg-app-surface-muted px-3 py-2 text-sm"
          >
            <div className="flex flex-wrap items-center gap-2">
              <span className="rounded-pill bg-app-info/15 px-2 py-0.5 text-xs text-app-info">
                {contextKindLabels[item.kind]}
              </span>
              <span className="font-medium text-app-foreground">{item.label || item.target || command}</span>
              {repeat ? <span className="text-xs text-app-muted-foreground">{repeat}</span> : null}
              {item.status === RelevantContextStatus.DEGRADED ? (
                <span className="rounded-pill bg-app-warning/15 px-2 py-0.5 text-xs text-app-warning">
                  {t(strings.common.degradedBadge)}
                </span>
              ) : null}
            </div>
            {item.reason ? <p className="mt-1 text-xs text-app-muted-foreground">{item.reason}</p> : null}
            {item.instruction ? <p className="mt-1 text-xs text-app-foreground">{item.instruction}</p> : null}
            {command ? (
              <code className="mt-2 block break-all rounded-control bg-app-surface px-2 py-1 font-mono text-xs text-app-foreground">
                {command}
              </code>
            ) : item.target ? (
              <code className="mt-2 block break-all rounded-control bg-app-surface px-2 py-1 font-mono text-xs text-app-foreground">
                {item.target}
              </code>
            ) : null}
          </li>
        );
      })}
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
        {phase.affectedAreas.length > 0 ? (
          <MetaRow term={t(strings.pages.plans.detail.phaseAffectedAreas)}>
            <StringList items={phase.affectedAreas} empty={t(strings.common.none)} />
          </MetaRow>
        ) : null}
        {phase.steps.length > 0 ? (
          <MetaRow term={t(strings.pages.plans.detail.phaseSteps)}>
            <ol className="ms-4 list-decimal text-xs text-app-foreground">
              {phase.steps.map((s, i) => (
                <li key={i}>{s}</li>
              ))}
            </ol>
          </MetaRow>
        ) : null}
        {phase.expectedOutputs.length > 0 ? (
          <MetaRow term={t(strings.pages.plans.detail.phaseExpectedOutputs)}>
            <StringList items={phase.expectedOutputs} empty={t(strings.common.none)} />
          </MetaRow>
        ) : null}
        {phase.validation ? (
          <MetaRow term={t(strings.pages.plans.detail.phaseValidation)}>{phase.validation}</MetaRow>
        ) : null}
        {phase.acceptance ? (
          <MetaRow term={t(strings.pages.plans.detail.phaseAcceptance)}>{phase.acceptance}</MetaRow>
        ) : null}
        {phase.risksHazards.length > 0 ? (
          <MetaRow term={t(strings.pages.plans.detail.phaseRisks)}>
            <StringList items={phase.risksHazards} empty={t(strings.common.none)} />
          </MetaRow>
        ) : null}
        {phase.handoffNotes ? (
          <MetaRow term={t(strings.pages.plans.detail.phaseHandoff)}>{phase.handoffNotes}</MetaRow>
        ) : null}
        {phase.relevantContext.length > 0 ? (
          <MetaRow term={t(strings.pages.plans.detail.phaseContext)}>
            <RelevantContextList items={phase.relevantContext} />
          </MetaRow>
        ) : phase.requiredReading.length > 0 ? (
          <MetaRow term={t(strings.pages.plans.detail.phaseLegacyRequiredReading)}>
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
        {plan.problemStatement ? (
          <SectionPanel title={t(strings.pages.plans.detail.problemHeading)} headingId="plan-problem">
            <p className="whitespace-pre-wrap text-sm text-app-foreground">{plan.problemStatement}</p>
          </SectionPanel>
        ) : null}
        {plan.targetOutcome ? (
          <SectionPanel title={t(strings.pages.plans.detail.targetOutcomeHeading)} headingId="plan-target-outcome">
            <p className="whitespace-pre-wrap text-sm text-app-foreground">{plan.targetOutcome}</p>
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
        {plan.assumptions ? (
          <SectionPanel title={t(strings.pages.plans.detail.assumptionsHeading)} headingId="plan-assumptions">
            <p className="whitespace-pre-wrap text-sm text-app-foreground">{plan.assumptions}</p>
          </SectionPanel>
        ) : null}
      </div>

      <SectionPanel title={t(strings.pages.plans.detail.workPostureHeading)} headingId="plan-work-posture">
        <dl className="flex flex-col gap-2">
          <MetaRow term={t(strings.pages.plans.detail.workPostureHeading)}>
            <span className="font-mono">{workPostureLabel(plan.workPosture)}</span>
          </MetaRow>
          {plan.workPostureDetail ? (
            <MetaRow term={t(strings.pages.plans.detail.workPostureDetail)}>{plan.workPostureDetail}</MetaRow>
          ) : null}
        </dl>
      </SectionPanel>

      <div className="grid gap-4 lg:grid-cols-2">
        {plan.technicalApproach ? (
          <SectionPanel title={t(strings.pages.plans.detail.technicalApproachHeading)} headingId="plan-technical-approach">
            <p className="whitespace-pre-wrap text-sm text-app-foreground">{plan.technicalApproach}</p>
          </SectionPanel>
        ) : null}
        {plan.prohibitedApproaches ? (
          <SectionPanel title={t(strings.pages.plans.detail.prohibitedApproachesHeading)} headingId="plan-prohibited">
            <p className="whitespace-pre-wrap text-sm text-app-foreground">{plan.prohibitedApproaches}</p>
          </SectionPanel>
        ) : null}
        {plan.validationStrategy || plan.finalValidationCommands.length > 0 ? (
          <SectionPanel title={t(strings.pages.plans.detail.validationStrategyHeading)} headingId="plan-validation-strategy">
            {plan.validationStrategy ? (
              <p className="whitespace-pre-wrap text-sm text-app-foreground">{plan.validationStrategy}</p>
            ) : null}
            {plan.finalValidationCommands.length > 0 ? (
              <div className="mt-2">
                <p className="text-xs uppercase tracking-wide text-app-muted-foreground">
                  {t(strings.pages.plans.detail.finalValidationCommands)}
                </p>
                <StringList items={plan.finalValidationCommands} empty={t(strings.common.none)} />
              </div>
            ) : null}
          </SectionPanel>
        ) : null}
        {plan.risksHazards ? (
          <SectionPanel title={t(strings.pages.plans.detail.risksHeading)} headingId="plan-risks">
            <p className="whitespace-pre-wrap text-sm text-app-foreground">{plan.risksHazards}</p>
          </SectionPanel>
        ) : null}
      </div>

      {plan.definitionOfDone ? (
        <SectionPanel title={t(strings.pages.plans.detail.dodHeading)} headingId="plan-dod">
          <p className="whitespace-pre-wrap text-sm text-app-foreground">{plan.definitionOfDone}</p>
        </SectionPanel>
      ) : null}

      {plan.relevantContext.length > 0 ? (
        <SectionPanel title={t(strings.pages.plans.detail.globalContextHeading)} headingId="plan-global-context">
          <RelevantContextList items={plan.relevantContext} />
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

      {plan.importProvenance || plan.preservedLegacySections.length > 0 ? (
        <SectionPanel
          title={t(strings.pages.plans.detail.importProvenanceHeading)}
          headingId="plan-import-provenance"
        >
          {plan.importProvenance ? (
            <MetaRow term={t(strings.pages.plans.detail.importSource)}>
              <span className="break-all font-mono text-xs">{plan.importProvenance.sourcePath}</span>
            </MetaRow>
          ) : null}
          {plan.preservedLegacySections.length > 0 ? (
            <div className="mt-3">
              <p className="text-xs uppercase tracking-wide text-app-muted-foreground">
                {t(strings.pages.plans.detail.preservedLegacyHeading)}
              </p>
              <p className="text-xs text-app-muted-foreground">
                {t(strings.pages.plans.detail.preservedLegacyNote)}
              </p>
              <ul className="mt-2 flex flex-col gap-2">
                {plan.preservedLegacySections.map((sec, i) => (
                  <li key={i} className="rounded-control bg-app-surface-muted p-2">
                    <p className="text-sm font-semibold text-app-foreground">{sec.heading}</p>
                    <p className="whitespace-pre-wrap text-xs text-app-muted-foreground">{sec.content}</p>
                  </li>
                ))}
              </ul>
            </div>
          ) : null}
        </SectionPanel>
      ) : null}

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
