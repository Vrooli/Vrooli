import { useState } from "react";
import { Flag, Play } from "lucide-react";

import {
  completeExecution,
  getContext,
  getStatus,
  startExecution,
  transitionPhase,
} from "../../api/execution";
import { addDecision, addFinding } from "../../api/log";
import { PlanSelect } from "../../components/PlanSelect";
import { StatusBadge } from "../../components/StatusBadge";
import { Card, MetaRow, SectionPanel } from "../../components/Surfaces";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Textarea } from "../../components/ui/textarea";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { type StringKey } from "../../consts/stringKey";
import { errorMessage } from "../../lib/errorMessage";
import { phaseStatusDescriptor, stalenessDescriptor, verdictDescriptor } from "../../lib/planStatus";
import { useTranslation } from "../../i18n";
import {
  Completeness,
  PhaseStatus,
  type GuidedStep,
  type Handoff,
  type LogEntry,
  type LogSummary,
  type RelevantContextItem,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";
import {
  RelevantContextKind,
  RelevantContextRepeatPolicy,
  RelevantContextStatus,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";
import type {
  CompletionNudge,
  Execution,
  PhaseContext,
} from "@vrooli/proto-types/plan-manager/v1/execution/execution_pb";

interface RunnerState {
  execution?: Execution;
  context?: PhaseContext;
  decisions: LogEntry[];
  findings: LogEntry[];
  handoff?: Handoff;
  nudges: CompletionNudge[];
  step?: GuidedStep;
}

const TRANSITION_TARGETS: { value: PhaseStatus; labelKey: StringKey }[] = [
  { value: PhaseStatus.ACTIVE, labelKey: strings.phaseStatus.active },
  { value: PhaseStatus.DONE, labelKey: strings.phaseStatus.done },
  { value: PhaseStatus.BLOCKED, labelKey: strings.phaseStatus.blocked },
  { value: PhaseStatus.TODO, labelKey: strings.phaseStatus.todo },
];

const COMPLETENESS_LABELS: Record<Completeness, StringKey> = {
  [Completeness.UNSPECIFIED]: strings.pages.execution.completenessUnspecified,
  [Completeness.FULL]: strings.pages.execution.completenessFull,
  [Completeness.PARTIAL]: strings.pages.execution.completenessPartial,
};

function StringList({ items, empty }: { items: readonly string[]; empty: string }) {
  if (items.length === 0) return <p className="text-xs text-app-muted-foreground">{empty}</p>;
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

function shellQuote(arg: string) {
  if (/^[A-Za-z0-9_./:=@+-]+$/.test(arg)) return arg;
  return `'${arg.replace(/'/g, "'\\''")}'`;
}

function shellCommand(argv: readonly string[]) {
  return ["vrooli", "scenario", "plan-manager", ...argv].map(shellQuote).join(" ");
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
    return <p className="text-sm text-app-muted-foreground">{t(strings.pages.execution.noSetupContext)}</p>;
  }
  return (
    <ul data-testid={selectors.execution.setupContext} className="flex flex-col gap-2">
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

function StepPanel({ step }: { step?: GuidedStep }) {
  if (!step || (!step.title && !step.summary)) return null;
  return (
    <SectionPanel title={step.title || step.stepKind} headingId="execution-step-heading">
      <div data-testid={selectors.execution.guidedStep} className="flex flex-col gap-3">
        {step.summary ? <p className="text-sm text-app-muted-foreground">{step.summary}</p> : null}
        {step.instructions.length > 0 ? (
          <ul className="flex flex-col gap-1">
            {step.instructions.map((item, i) => (
              <li key={`${item}-${i}`} className="text-sm text-app-foreground">
                {item}
              </li>
            ))}
          </ul>
        ) : null}
        {step.requiredInputs.length > 0 ? (
          <div className="flex flex-wrap gap-2">
            {step.requiredInputs.map((item) => (
              <span
                key={item}
                className="rounded-control border border-app-border bg-app-surface-muted px-2 py-1 font-mono text-xs"
              >
                {item}
              </span>
            ))}
          </div>
        ) : null}
        {step.nextActions.length > 0 ? (
          <div className="flex flex-col gap-2">
            {step.nextActions.map((action) => (
              <code
                key={`${action.label}-${action.argv.join(" ")}`}
                className="break-all rounded-control bg-app-surface-muted px-3 py-2 text-xs text-app-foreground"
              >
                {shellCommand(action.argv)}
              </code>
            ))}
          </div>
        ) : null}
      </div>
    </SectionPanel>
  );
}

/**
 * LogSummaryView — a restrained roll-up of the log ledger (counts plus any
 * pending/failed downstream sync). Surfaced from the phase context and the
 * handoff so an operator reorients without reading every entry.
 */
function LogSummaryView({ summary, testId }: { summary?: LogSummary; testId?: string }) {
  const { t } = useTranslation();
  if (!summary || summary.total === 0) {
    return <p className="text-sm text-app-muted-foreground">{t(strings.pages.execution.logSummaryEmpty)}</p>;
  }
  return (
    <dl data-testid={testId} className="flex flex-col gap-2 text-sm">
      <MetaRow term={t(strings.pages.execution.logTotal)}>{summary.total}</MetaRow>
      <MetaRow term={t(strings.pages.execution.logDecisions)}>{summary.decisions}</MetaRow>
      <MetaRow term={t(strings.pages.execution.logFindings)}>{summary.findings}</MetaRow>
      <MetaRow term={t(strings.pages.execution.logCandidates)}>{summary.candidateFindings}</MetaRow>
      {summary.pendingSync > 0 ? (
        <MetaRow term={t(strings.pages.execution.logPendingSync)}>
          <span className="text-app-warning">{summary.pendingSync}</span>
        </MetaRow>
      ) : null}
      {summary.failedSync > 0 ? (
        <MetaRow term={t(strings.pages.execution.logFailedSync)}>
          <span className="text-app-danger">{summary.failedSync}</span>
        </MetaRow>
      ) : null}
    </dl>
  );
}

/**
 * ExecutionRunner — the guided runner. Start a run for a plan, read the
 * just-in-time context for the current phase, transition phases, capture
 * decisions and candidate findings in-flow, then complete and read the
 * canonical structured handoff (plus any completion nudges).
 */
export function ExecutionRunner() {
  const { t } = useTranslation();
  const [planId, setPlanId] = useState("");
  const [runId, setRunId] = useState("");
  const [state, setState] = useState<RunnerState>({ decisions: [], findings: [], nudges: [] });
  const [toStatus, setToStatus] = useState<PhaseStatus>(PhaseStatus.DONE);
  const [decisionSummary, setDecisionSummary] = useState("");
  const [decisionDetail, setDecisionDetail] = useState("");
  const [findingTitle, setFindingTitle] = useState("");
  const [findingDetail, setFindingDetail] = useState("");
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);

  const execution = state.execution;
  const context = state.context;
  const currentPhaseId = context?.currentPhase?.id ?? execution?.currentPhaseId ?? "";

  const run = (fn: () => Promise<void>) => {
    setBusy(true);
    setError(null);
    void (async () => {
      try {
        await fn();
      } catch (e) {
        setError(e);
      } finally {
        setBusy(false);
      }
    })();
  };

  const refreshStatus = async (executionId: string) => {
    const res = await getStatus(executionId);
    setState((prev) => ({ ...prev, execution: res.execution, context: res.context, step: res.step }));
  };

  const handleStart = (e: React.FormEvent) => {
    e.preventDefault();
    if (planId.length === 0) return;
    run(async () => {
      const res = await startExecution(planId, runId.trim());
      const exec = res.execution;
      if (exec) {
        setState({
          execution: exec,
          context: res.context,
          decisions: [],
          findings: [],
          nudges: [],
          step: res.step,
        });
        if (!res.context) {
          await refreshStatus(exec.id);
        }
      }
    });
  };

  const handleRefreshContext = () => {
    if (!execution) return;
    run(async () => {
      const res = await getContext(execution.id);
      setState((prev) => ({
        ...prev,
        execution: res.execution ?? prev.execution,
        context: res.context,
        step: res.step,
      }));
    });
  };

  const handleTransition = () => {
    if (!execution || currentPhaseId.length === 0) return;
    run(async () => {
      const res = await transitionPhase(execution.id, currentPhaseId, toStatus);
      setState((prev) => ({ ...prev, execution: res.execution ?? prev.execution, step: res.step }));
      await refreshStatus(execution.id);
    });
  };

  const handleRecordDecision = () => {
    if (!execution || decisionSummary.trim().length === 0) return;
    run(async () => {
      const res = await addDecision(execution.id, currentPhaseId, decisionSummary.trim(), {
        detail: decisionDetail.trim(),
      });
      const dec = res.entry;
      if (dec) {
        setState((prev) => ({ ...prev, decisions: [...prev.decisions, dec], step: res.step }));
        setDecisionSummary("");
        setDecisionDetail("");
      }
    });
  };

  const handleRecordFinding = () => {
    if (!execution || findingTitle.trim().length === 0) return;
    run(async () => {
      const res = await addFinding(execution.id, currentPhaseId, findingTitle.trim(), {
        detail: findingDetail.trim(),
      });
      const finding = res.entry;
      if (finding) {
        setState((prev) => ({ ...prev, findings: [...prev.findings, finding], step: res.step }));
        setFindingTitle("");
        setFindingDetail("");
      }
    });
  };

  const handleComplete = () => {
    if (!execution) return;
    run(async () => {
      const res = await completeExecution(execution.id);
      setState((prev) => ({ ...prev, handoff: res.handoff, nudges: res.nudges, step: res.step }));
      await refreshStatus(execution.id);
    });
  };

  if (!execution) {
    return (
      <SectionPanel title={t(strings.pages.execution.startHeading)} headingId="execution-start-heading">
        <form
          data-testid={selectors.execution.startForm}
          onSubmit={handleStart}
          className="flex flex-col gap-3 sm:flex-row sm:items-end"
        >
          <div className="flex-1">
            <PlanSelect
              value={planId}
              onChange={setPlanId}
              label={t(strings.pages.execution.planLabel)}
              testId={selectors.execution.planSelect}
            />
          </div>
          <label className="flex flex-1 flex-col gap-1 text-sm">
            <span className="text-xs font-medium text-app-muted-foreground">
              {t(strings.pages.execution.runIdLabel)}
            </span>
            <Input
              data-testid={selectors.execution.runIdInput}
              value={runId}
              onChange={(e) => setRunId(e.target.value)}
            />
          </label>
          <Button
            type="submit"
            data-testid={selectors.execution.startButton}
            disabled={busy || planId.length === 0}
            className="shrink-0"
          >
            <Play aria-hidden="true" className="me-2 h-4 w-4" />
            {t(strings.pages.execution.start)}
          </Button>
        </form>
        {error ? (
          <p role="alert" className="text-sm text-app-danger">
            {errorMessage(error, t)}
          </p>
        ) : null}
      </SectionPanel>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      {error ? (
        <p role="alert" className="rounded-control bg-app-danger/10 px-3 py-2 text-sm text-app-danger">
          {errorMessage(error, t)}
        </p>
      ) : null}

      <StepPanel step={state.step} />

      <SectionPanel
        title={t(strings.pages.execution.contextHeading)}
        headingId="execution-context-heading"
        actions={
          <Button
            type="button"
            size="sm"
            variant="outline"
            data-testid={selectors.execution.contextButton}
            disabled={busy || !execution}
            onClick={handleRefreshContext}
          >
            {t(strings.pages.execution.refreshContext)}
          </Button>
        }
      >
        <div data-testid={selectors.execution.context} className="grid gap-4 lg:grid-cols-2">
          <Card className="bg-app-surface-muted">
            <div className="flex items-center justify-between gap-2">
              <h4 className="text-sm font-semibold">{t(strings.pages.execution.currentPhaseHeading)}</h4>
              {context?.currentPhase ? (
                <StatusBadge descriptor={phaseStatusDescriptor(context.currentPhase.status)} />
              ) : null}
            </div>
            {context?.currentPhase ? (
              <div className="mt-2 flex flex-col gap-1">
                <p className="text-sm font-medium text-app-foreground">
                  {context.currentPhase.order}. {context.currentPhase.title}
                </p>
                {context.currentPhase.intent ? (
                  <p className="text-sm text-app-muted-foreground">{context.currentPhase.intent}</p>
                ) : null}
              </div>
            ) : (
              <p className="mt-2 text-sm text-app-muted-foreground">
                {t(strings.pages.execution.noCurrentPhase)}
              </p>
            )}
          </Card>

          <Card className="bg-app-surface-muted">
            <h4 className="text-sm font-semibold">{t(strings.pages.execution.nextPhaseHeading)}</h4>
            {context?.nextPhase ? (
              <p className="mt-2 text-sm text-app-foreground">
                {context.nextPhase.order}. {context.nextPhase.title}
              </p>
            ) : (
              <p className="mt-2 text-sm text-app-muted-foreground">
                {t(strings.pages.execution.noNextPhase)}
              </p>
            )}
          </Card>

          <Card className="bg-app-surface-muted lg:col-span-2">
            <h4 className="text-sm font-semibold">{t(strings.pages.execution.setupContextHeading)}</h4>
            <div className="mt-2">
              <RelevantContextList items={context?.relevantContext ?? []} />
            </div>
          </Card>

          {context?.relevantContext.length ? null : (
            <Card className="bg-app-surface-muted">
              <h4 className="text-sm font-semibold">{t(strings.pages.execution.legacyRequiredReadingHeading)}</h4>
              <div className="mt-2">
                <StringList items={context?.requiredReading ?? []} empty={t(strings.common.none)} />
              </div>
            </Card>
          )}

          <Card className="bg-app-surface-muted">
            <h4 className="text-sm font-semibold">{t(strings.pages.execution.remindersHeading)}</h4>
            <div className="mt-2">
              <StringList items={context?.reminders ?? []} empty={t(strings.common.none)} />
            </div>
          </Card>

          <Card className="bg-app-surface-muted">
            <dl className="flex flex-col gap-2">
              <MetaRow term={t(strings.pages.execution.stalenessHeading)}>
                {context ? <StatusBadge descriptor={stalenessDescriptor(context.staleness)} /> : "—"}
              </MetaRow>
              <MetaRow term={t(strings.pages.execution.lastValidationHeading)}>
                {context?.lastValidation ? (
                  <StatusBadge descriptor={verdictDescriptor(context.lastValidation.verdict)} />
                ) : (
                  t(strings.common.none)
                )}
              </MetaRow>
            </dl>
          </Card>

          <Card className="bg-app-surface-muted">
            <h4 className="text-sm font-semibold">{t(strings.pages.execution.logSummaryHeading)}</h4>
            <div className="mt-2">
              <LogSummaryView summary={context?.logSummary} testId={selectors.execution.logSummary} />
            </div>
          </Card>

          <Card className="bg-app-surface-muted">
            <h4 className="text-sm font-semibold">{t(strings.pages.execution.transitionHeading)}</h4>
            <div className="mt-2 flex flex-wrap items-end gap-2">
              <label className="flex flex-col gap-1 text-sm">
                <span className="text-xs text-app-muted-foreground">
                  {t(strings.pages.execution.transitionTo)}
                </span>
                <select
                  data-testid={selectors.execution.transitionSelect}
                  value={String(toStatus)}
                  onChange={(e) => setToStatus(Number(e.target.value))}
                  className="h-10 rounded-control border border-app-border bg-app-surface px-3 text-app-foreground"
                >
                  {TRANSITION_TARGETS.map((target) => (
                    <option key={target.value} value={String(target.value)}>
                      {t(target.labelKey)}
                    </option>
                  ))}
                </select>
              </label>
              <Button
                type="button"
                size="sm"
                data-testid={selectors.execution.transitionButton}
                disabled={busy || currentPhaseId.length === 0}
                onClick={handleTransition}
              >
                {t(strings.pages.execution.transition)}
              </Button>
            </div>
          </Card>
        </div>
      </SectionPanel>

      <div className="grid gap-4 lg:grid-cols-2">
        <SectionPanel title={t(strings.pages.execution.decisionsHeading)} headingId="execution-decisions-heading">
          <div className="flex flex-col gap-2">
            <Input
              data-testid={selectors.execution.decisionSummary}
              value={decisionSummary}
              onChange={(e) => setDecisionSummary(e.target.value)}
              placeholder={t(strings.pages.execution.decisionSummaryPlaceholder)}
              aria-label={t(strings.pages.execution.decisionSummaryLabel)}
            />
            <Textarea
              data-testid={selectors.execution.decisionDetail}
              value={decisionDetail}
              onChange={(e) => setDecisionDetail(e.target.value)}
              rows={2}
              aria-label={t(strings.pages.execution.decisionDetailLabel)}
            />
            <Button
              type="button"
              size="sm"
              data-testid={selectors.execution.recordDecisionButton}
              disabled={busy || decisionSummary.trim().length === 0}
              onClick={handleRecordDecision}
              className="w-fit"
            >
              {t(strings.pages.execution.recordDecision)}
            </Button>
          </div>
          {state.decisions.length === 0 ? (
            <p className="text-sm text-app-muted-foreground">{t(strings.pages.execution.noDecisions)}</p>
          ) : (
            <ul className="flex flex-col gap-2">
              {state.decisions.map((dec) => (
                <li
                  key={dec.id}
                  className="rounded-control border border-app-border bg-app-surface-muted px-3 py-2 text-sm"
                >
                  <p className="font-medium text-app-foreground">{dec.title}</p>
                  {dec.detail ? <p className="text-app-muted-foreground">{dec.detail}</p> : null}
                </li>
              ))}
            </ul>
          )}
        </SectionPanel>

        <SectionPanel title={t(strings.pages.execution.findingsHeading)} headingId="execution-findings-heading">
          <div className="flex flex-col gap-2">
            <Input
              data-testid={selectors.execution.findingTitle}
              value={findingTitle}
              onChange={(e) => setFindingTitle(e.target.value)}
              placeholder={t(strings.pages.execution.findingTitlePlaceholder)}
              aria-label={t(strings.pages.execution.findingTitleLabel)}
            />
            <Textarea
              data-testid={selectors.execution.findingDetail}
              value={findingDetail}
              onChange={(e) => setFindingDetail(e.target.value)}
              rows={2}
              aria-label={t(strings.pages.execution.findingDetailLabel)}
            />
            <Button
              type="button"
              size="sm"
              data-testid={selectors.execution.recordFindingButton}
              disabled={busy || findingTitle.trim().length === 0}
              onClick={handleRecordFinding}
              className="w-fit"
            >
              {t(strings.pages.execution.recordFinding)}
            </Button>
          </div>
          {state.findings.length === 0 ? (
            <p className="text-sm text-app-muted-foreground">{t(strings.pages.execution.noFindings)}</p>
          ) : (
            <ul className="flex flex-col gap-2">
              {state.findings.map((finding) => (
                <li
                  key={finding.id}
                  className="rounded-control border border-app-border bg-app-surface-muted px-3 py-2 text-sm"
                >
                  <p className="font-medium text-app-foreground">{finding.title}</p>
                  {finding.detail ? <p className="text-app-muted-foreground">{finding.detail}</p> : null}
                </li>
              ))}
            </ul>
          )}
        </SectionPanel>
      </div>

      <SectionPanel
        title={t(strings.pages.execution.completeHeading)}
        headingId="execution-complete-heading"
        actions={
          <Button
            type="button"
            size="sm"
            data-testid={selectors.execution.completeButton}
            disabled={busy}
            onClick={handleComplete}
          >
            <Flag aria-hidden="true" className="me-2 h-4 w-4" />
            {t(strings.pages.execution.complete)}
          </Button>
        }
      >
        {state.nudges.length > 0 ? (
          <div className="mb-3">
            <p className="text-xs uppercase tracking-wide text-app-muted-foreground">
              {t(strings.pages.execution.nudgesHeading)}
            </p>
            <ul className="mt-1 flex flex-col gap-1">
              {state.nudges.map((nudge, i) => (
                <li
                  key={`${nudge.kind}-${i}`}
                  className="rounded-control bg-app-warning/10 px-3 py-1.5 text-sm text-app-warning"
                >
                  {nudge.message}
                </li>
              ))}
            </ul>
          </div>
        ) : null}

        {state.handoff ? (
          <dl data-testid={selectors.execution.handoff} className="flex flex-col gap-2 text-sm">
            <MetaRow term={t(strings.pages.execution.handoffCompleteness)}>
              {t(COMPLETENESS_LABELS[state.handoff.completeness])}
            </MetaRow>
            <MetaRow term={t(strings.pages.execution.handoffStaleness)}>
              <StatusBadge descriptor={stalenessDescriptor(state.handoff.staleness)} />
            </MetaRow>
            {state.handoff.resumePhaseId ? (
              <MetaRow term={t(strings.pages.execution.handoffResume)}>
                <span className="font-mono text-xs">{state.handoff.resumePhaseId}</span>
              </MetaRow>
            ) : null}
            {state.handoff.proseHandoffRef ? (
              <MetaRow term={t(strings.pages.execution.handoffProseRef)}>
                <span className="break-all font-mono text-xs">{state.handoff.proseHandoffRef}</span>
              </MetaRow>
            ) : null}
            <MetaRow term={t(strings.pages.execution.handoffLogEntries)}>
              {state.handoff.logEntries.length}
            </MetaRow>
          </dl>
        ) : null}

        {state.handoff?.logSummary ? (
          <div className="mt-3">
            <p className="text-xs uppercase tracking-wide text-app-muted-foreground">
              {t(strings.pages.execution.logSummaryHeading)}
            </p>
            <div className="mt-1">
              <LogSummaryView summary={state.handoff.logSummary} />
            </div>
          </div>
        ) : null}

        {!state.handoff ? (
          <p className="text-sm text-app-muted-foreground">{t(strings.pages.execution.handoffNone)}</p>
        ) : null}
      </SectionPanel>
    </div>
  );
}
