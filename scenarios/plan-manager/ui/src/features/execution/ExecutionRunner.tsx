import { useState } from "react";
import { Flag, Play } from "lucide-react";

import {
  completeExecution,
  getContext,
  getStatus,
  resumeExecution,
  startExecution,
  transitionPhase,
} from "../../api/execution";
import { addBug, addDecision, addFinding, addNote, addRecord, listEntries, syncEntry } from "../../api/log";
import { PlanSelect } from "../../components/PlanSelect";
import { StatusBadge } from "../../components/StatusBadge";
import { Card, MetaRow, SectionPanel } from "../../components/Surfaces";
import { GuidedStepPanel } from "../../components/GuidedStepPanel";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Textarea } from "../../components/ui/textarea";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { type StringKey } from "../../consts/stringKey";
import { errorMessage } from "../../lib/errorMessage";
import { phaseStatusDescriptor, stalenessDescriptor, verdictDescriptor } from "../../lib/planStatus";
import { contextCommand, contextKindLabel, repeatLabel } from "../../lib/relevantContext";
import { useTranslation } from "../../i18n";
import {
  Completeness,
  LogEntryType,
  LogSyncStatus,
  PhaseStatus,
  type Handoff,
  type LogEntry,
  type LogSummary,
  type RelevantContextItem,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";
import { RelevantContextStatus } from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";
import type { GuidedStep } from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";
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
  bugs: LogEntry[];
  records: LogEntry[];
  notes: LogEntry[];
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

function commaSeparated(value: string): string[] {
  return value.split(",").map((item) => item.trim()).filter(Boolean);
}

function logEntriesByType(entries: readonly LogEntry[]): Pick<RunnerState, "decisions" | "findings" | "bugs" | "records" | "notes"> {
  const grouped: Pick<RunnerState, "decisions" | "findings" | "bugs" | "records" | "notes"> = {
    decisions: [],
    findings: [],
    bugs: [],
    records: [],
    notes: [],
  };
  for (const entry of entries) {
    switch (entry.type) {
      case LogEntryType.DECISION:
        grouped.decisions.push(entry);
        break;
      case LogEntryType.FINDING:
        grouped.findings.push(entry);
        break;
      case LogEntryType.BUG_REPORT:
        grouped.bugs.push(entry);
        break;
      case LogEntryType.RECORD:
        grouped.records.push(entry);
        break;
      case LogEntryType.NOTE:
        grouped.notes.push(entry);
        break;
    }
  }
  return grouped;
}

function syncLabel(status: LogSyncStatus): StringKey {
  switch (status) {
    case LogSyncStatus.SYNCED:
      return strings.pages.execution.captureForwarded;
    case LogSyncStatus.PENDING:
      return strings.pages.execution.captureForwardingPending;
    case LogSyncStatus.FAILED:
      return strings.pages.execution.captureForwardingFailed;
    case LogSyncStatus.LOCAL:
      return strings.pages.execution.captureLocal;
    default:
      return strings.pages.execution.captureForwardingUnknown;
  }
}

function CaptureState({ entry, onRetry }: { entry: LogEntry; onRetry?: (entry: LogEntry) => void }) {
  const { t } = useTranslation();
  // DownstreamRef carries the same producer response for older persisted rows;
  // prefer the first-class field but retain that compatibility during reload.
  const capture = entry.capture ?? entry.downstream?.capture;
  const downstream = entry.downstream;
  const retryable = entry.syncStatus === LogSyncStatus.PENDING || entry.syncStatus === LogSyncStatus.FAILED;
  if (!capture?.state && !downstream && !retryable) return null;
  return (
    <div className="mt-2 flex flex-col gap-1 rounded-control border border-app-border bg-app-surface px-2 py-1 text-xs text-app-foreground">
      {capture?.state ? (
        <p>
          {t(strings.pages.execution.captureDisposition)}: {capture.state === "published" ? t(strings.pages.execution.captureAcceptedPublished) : capture.state}
          {capture.draftId ? ` — ${t(strings.pages.execution.capturePrivateDraft, { draftId: capture.draftId })}` : ""}
        </p>
      ) : null}
      <p>{t(strings.pages.execution.captureForwarding)}: {t(syncLabel(entry.syncStatus))}</p>
      {capture?.needs.length ? <p>{t(strings.pages.execution.captureNeeds)}: {capture.needs.join(", ")}</p> : null}
      {capture?.invalid.map((invalid) => <p key={`${invalid.field}-${invalid.value}`}>{t(strings.pages.execution.captureInvalid)} {invalid.field}: {invalid.message}</p>)}
      {capture?.warnings.map((warning, index) => <p key={`${warning}-${index}`}>{t(strings.pages.execution.captureWarning)}: {warning}</p>)}
      {capture?.nextAction.length ? <code className="block break-all font-mono">{t(strings.pages.execution.captureRepair)}: {capture.nextAction.join(" ")}</code> : null}
      {downstream ? (
        <div className="border-t border-app-border pt-1">
          <p>{t(strings.pages.execution.captureProvenance)}: {downstream.system || t(strings.pages.execution.captureDownstream)}{downstream.kind ? ` / ${downstream.kind}` : ""}</p>
          {downstream.reference ? <p className="break-all">{t(strings.pages.execution.captureReference)}: {downstream.reference}</p> : null}
          {downstream.detail ? <p>{t(strings.pages.execution.captureSyncDetail)}: {downstream.detail}</p> : null}
          {downstream.syncedAt ? <p>{t(strings.pages.execution.captureSynced)}: {downstream.syncedAt}</p> : null}
        </div>
      ) : null}
      {entry.sourceCommand ? <code className="block break-all font-mono">{t(strings.pages.execution.captureCapturedBy)}: {entry.sourceCommand}</code> : null}
      {entry.attributionRunId ? <p>{t(strings.pages.execution.captureRun)}: {entry.attributionRunId}</p> : null}
      {entry.evidence.length ? <p className="break-words">{t(strings.pages.execution.captureEvidence)}: {entry.evidence.join(", ")}</p> : null}
      {retryable && onRetry ? (
        <Button type="button" size="sm" variant="outline" data-testid={selectors.execution.retrySync({ id: entry.id })} className="mt-1 w-fit" onClick={() => onRetry(entry)}>
          {t(strings.pages.execution.captureRetrySync)}
        </Button>
      ) : null}
    </div>
  );
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
                {contextKindLabel(item.kind)}
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
  const [resumeExecutionId, setResumeExecutionId] = useState("");
  const [state, setState] = useState<RunnerState>({
    decisions: [],
    findings: [],
    bugs: [],
    records: [],
    notes: [],
    nudges: [],
  });
  const [toStatus, setToStatus] = useState<PhaseStatus>(PhaseStatus.DONE);
  const [decisionSummary, setDecisionSummary] = useState("");
  const [decisionDetail, setDecisionDetail] = useState("");
  const [findingTitle, setFindingTitle] = useState("");
  const [findingDetail, setFindingDetail] = useState("");
  const [bugTitle, setBugTitle] = useState("");
  const [bugDetail, setBugDetail] = useState("");
  const [bugSignalType, setBugSignalType] = useState("");
  const [bugSeverity, setBugSeverity] = useState("");
  const [bugRepro, setBugRepro] = useState("");
  const [bugExpected, setBugExpected] = useState("");
  const [bugActual, setBugActual] = useState("");
  const [bugDescription, setBugDescription] = useState("");
  const [bugScenario, setBugScenario] = useState("");
  const [bugHonestyFlags, setBugHonestyFlags] = useState("");
  const [recordTitle, setRecordTitle] = useState("");
  const [recordDetail, setRecordDetail] = useState("");
  const [recordKind, setRecordKind] = useState("");
  const [recordScenario, setRecordScenario] = useState("");
  const [recordTrigger, setRecordTrigger] = useState("");
  const [recordApproach, setRecordApproach] = useState("");
  const [recordEvidence, setRecordEvidence] = useState("");
  const [recordOutcome, setRecordOutcome] = useState("");
  const [noteTitle, setNoteTitle] = useState("");
  const [noteDetail, setNoteDetail] = useState("");
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);

  const execution = state.execution;
  const context = state.context;
  const baselineSet = context;
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

  const hydrateLogEntries = async (executionId: string) => {
    const res = await listEntries({ planOrExecution: executionId });
    setState((prev) => ({
      ...prev,
      ...logEntriesByType(res.entries),
      step: res.step ?? prev.step,
    }));
  };

  const replaceLogEntry = (entry: LogEntry) => {
    const grouped = logEntriesByType([entry]);
    const replace = (entries: LogEntry[]) => entries.map((current) => current.id === entry.id ? entry : current);
    setState((prev) => ({
      ...prev,
      decisions: grouped.decisions.length ? replace(prev.decisions) : prev.decisions,
      findings: grouped.findings.length ? replace(prev.findings) : prev.findings,
      bugs: grouped.bugs.length ? replace(prev.bugs) : prev.bugs,
      records: grouped.records.length ? replace(prev.records) : prev.records,
      notes: grouped.notes.length ? replace(prev.notes) : prev.notes,
    }));
  };

  const loadExecution = async (
    execution: Execution,
    context: PhaseContext | undefined,
    step: GuidedStep | undefined,
  ) => {
    setState({
      execution,
      context,
      decisions: [],
      findings: [],
      bugs: [],
      records: [],
      notes: [],
      nudges: [],
      step,
    });
    if (!context) {
      await refreshStatus(execution.id);
    }
    // The execution service owns resumability. Rehydrate its durable ledger
    // after every start/resume instead of treating browser state as canonical.
    await hydrateLogEntries(execution.id);
  };

  const handleStart = (e: React.FormEvent) => {
    e.preventDefault();
    if (planId.length === 0) return;
    run(async () => {
      const res = await startExecution(planId, runId.trim());
      const exec = res.execution;
      if (exec) {
        await loadExecution(exec, res.context, res.step);
      }
    });
  };

  const handleResume = (e: React.FormEvent) => {
    e.preventDefault();
    const executionId = resumeExecutionId.trim();
    if (!executionId) return;
    run(async () => {
      const res = await resumeExecution(executionId);
      if (res.execution) {
        await loadExecution(res.execution, res.context, res.step);
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
      await hydrateLogEntries(execution.id);
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
        await refreshStatus(execution.id);
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
        await refreshStatus(execution.id);
      }
    });
  };

  const handleRecordBug = () => {
    if (!execution || bugTitle.trim().length === 0) return;
    run(async () => {
      const res = await addBug(execution.id, currentPhaseId, bugTitle.trim(), {
        detail: bugDetail.trim(),
        signalType: bugSignalType.trim(),
        reportSeverity: bugSeverity.trim(),
        repro: commaSeparated(bugRepro),
        expected: bugExpected.trim(),
        actual: bugActual.trim(),
        description: bugDescription.trim(),
        context: bugScenario.trim() ? { scenario: bugScenario.trim() } : {},
        honestyFlags: commaSeparated(bugHonestyFlags),
      });
      const bug = res.entry;
      if (bug) {
        setState((prev) => ({ ...prev, bugs: [...prev.bugs, bug], step: res.step }));
        setBugTitle("");
        setBugDetail("");
        setBugSignalType("");
        setBugSeverity("");
        setBugRepro("");
        setBugExpected("");
        setBugActual("");
        setBugDescription("");
        setBugScenario("");
        setBugHonestyFlags("");
        await refreshStatus(execution.id);
      }
    });
  };

  const handleRecordRecord = () => {
    if (!execution || recordTitle.trim().length === 0) return;
    run(async () => {
      const res = await addRecord(execution.id, currentPhaseId, recordTitle.trim(), {
        detail: recordDetail.trim(),
        kind: recordKind.trim(),
        scenario: recordScenario.trim(),
        trigger: recordTrigger.trim(),
        approach: recordApproach.trim(),
        recordEvidence: recordEvidence.trim(),
        outcome: recordOutcome.trim(),
      });
      const record = res.entry;
      if (record) {
        setState((prev) => ({ ...prev, records: [...prev.records, record], step: res.step }));
        setRecordTitle("");
        setRecordDetail("");
        setRecordKind("");
        setRecordScenario("");
        setRecordTrigger("");
        setRecordApproach("");
        setRecordEvidence("");
        setRecordOutcome("");
        await refreshStatus(execution.id);
      }
    });
  };

  const handleRecordNote = () => {
    if (!execution || noteTitle.trim().length === 0) return;
    run(async () => {
      const res = await addNote(execution.id, currentPhaseId, noteTitle.trim(), {
        detail: noteDetail.trim(),
      });
      const note = res.entry;
      if (note) {
        setState((prev) => ({ ...prev, notes: [...prev.notes, note], step: res.step }));
        setNoteTitle("");
        setNoteDetail("");
        await refreshStatus(execution.id);
      }
    });
  };

  const handleNoFeedback = () => {
    if (!execution || currentPhaseId.length === 0) return;
    const checkpoint = context?.feedbackCheckpoint;
    run(async () => {
      const res = await addNote(
        execution.id,
        currentPhaseId,
        checkpoint?.noFeedbackTitle || "Phase feedback reviewed: none",
        { detail: checkpoint?.noFeedbackDetail || "No decisions, findings, bugs, records, or reusable notes to capture for this phase." },
      );
      const note = res.entry;
      if (note) {
        setState((prev) => ({ ...prev, notes: [...prev.notes, note], step: res.step }));
        await refreshStatus(execution.id);
      }
    });
  };

  const handleRetrySync = (entry: LogEntry) => {
    if (!execution) return;
    run(async () => {
      const res = await syncEntry(entry.id);
      if (res.entry) {
        replaceLogEntry(res.entry);
      }
      setState((prev) => ({ ...prev, step: res.step ?? prev.step }));
      await refreshStatus(execution.id);
      await hydrateLogEntries(execution.id);
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
        <form
          data-testid={selectors.execution.resumeForm}
          onSubmit={handleResume}
          className="mt-4 flex flex-col gap-3 border-t border-app-border pt-4 sm:flex-row sm:items-end"
        >
          <label className="flex-1 text-sm">
            <span className="text-xs font-medium text-app-muted-foreground">
              {t(strings.pages.execution.resumeExecutionIdLabel)}
            </span>
            <Input
              data-testid={selectors.execution.resumeExecutionIdInput}
              value={resumeExecutionId}
              onChange={(e) => setResumeExecutionId(e.target.value)}
            />
          </label>
          <Button
            type="submit"
            data-testid={selectors.execution.resumeButton}
            disabled={busy || resumeExecutionId.trim().length === 0}
            className="shrink-0"
          >
            <Play aria-hidden="true" className="me-2 h-4 w-4" />
            {t(strings.pages.execution.resume)}
          </Button>
        </form>
        <p className="mt-2 text-xs text-app-muted-foreground">{t(strings.pages.execution.resumeHelp)}</p>
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

      <GuidedStepPanel
        step={state.step}
        headingId="execution-step-heading"
        testId={selectors.execution.guidedStep}
        commandPrefix={["vrooli", "scenario", "plan-manager"]}
      />

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
            <h4 className="text-sm font-semibold">{t(strings.pages.execution.changeBoundaryHeading)}</h4>
            <div className="mt-2 flex flex-col gap-2">
              {(context?.changeBoundary?.acceptanceAllow.length ?? 0) ||
              (context?.changeBoundary?.acceptanceDeny.length ?? 0) ||
              context?.changeBoundary?.operatorOnlyReason ? (
                <>
                  {context?.changeBoundary?.acceptanceAllow.length ? (
                    <div>
                      <p className="text-xs font-medium text-app-muted-foreground">
                        {t(strings.pages.execution.changeBoundaryAllow)}
                      </p>
                      <StringList items={context.changeBoundary.acceptanceAllow} empty={t(strings.common.none)} />
                    </div>
                  ) : null}
                  {context?.changeBoundary?.acceptanceDeny.length ? (
                    <div>
                      <p className="text-xs font-medium text-app-muted-foreground">
                        {t(strings.pages.execution.changeBoundaryDeny)}
                      </p>
                      <StringList items={context.changeBoundary.acceptanceDeny} empty={t(strings.common.none)} />
                    </div>
                  ) : null}
                  {context?.changeBoundary?.operatorOnlyReason ? (
                    <p className="text-xs text-app-muted-foreground">
                      {context.changeBoundary.operatorOnlyReason}
                    </p>
                  ) : null}
                </>
              ) : (
                <p className="text-sm text-app-muted-foreground">{t(strings.pages.execution.changeBoundaryNone)}</p>
              )}
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
              <MetaRow term={t(strings.pages.execution.inputsFreshenedHeading)}>
                <span data-testid={selectors.execution.freshenStatus}>
                  {context?.inputsFreshened
                    ? context.freshenStatus || t(strings.pages.execution.freshenStatusHeading)
                    : t(strings.pages.execution.freshenPending)}
                </span>
              </MetaRow>
              {context?.inputsFreshened && context.freshenDetail ? (
                <MetaRow term={t(strings.pages.execution.freshenStatusHeading)}>
                  <span className="text-xs text-app-muted-foreground">{context.freshenDetail}</span>
                </MetaRow>
              ) : null}
			  {baselineSet?.baselineSet?.name ? (
				<>
				  <MetaRow term={t(strings.pages.execution.baselineSetHeading)}>
					<span data-testid="execution-baseline-set" className="font-mono text-xs">
					  {baselineSet.baselineSet.name} ({baselineSet.baselineSet.status})
					</span>
				  </MetaRow>
				  <MetaRow term={t(strings.pages.execution.baselineSetCoverage)}>
					<span className="text-xs">
					  required {baselineSet.baselineSet.required} · ready {baselineSet.baselineSet.ready} · pending {baselineSet.baselineSet.pending} · failed {baselineSet.baselineSet.failed}
					</span>
				  </MetaRow>
				  {baselineSet.baselineSet.repoPaths.length > 0 ? (
					<MetaRow term={t(strings.pages.execution.baselineSetSourcePaths)}>
					  <span className="text-xs text-app-muted-foreground">{baselineSet.baselineSet.repoPaths.join(", ")}</span>
					</MetaRow>
				  ) : null}
				  {baselineSet.baselineSet.members && baselineSet.baselineSet.members.length > 0 ? (
					<MetaRow term={t(strings.pages.execution.baselineSetMembers)}>
					  <span className="text-xs text-app-muted-foreground">
						{baselineSet.baselineSet.members.map((member) => `${member.scenario}: ${member.status}${member.runId ? ` (${member.runId})` : ""}${member.gitSha ? ` @${member.gitSha}` : ""}`).join(" · ")}
					  </span>
					</MetaRow>
				  ) : null}
				  {baselineSet.baselineSet.pathSnapshots && baselineSet.baselineSet.pathSnapshots.length > 0 ? (
					<MetaRow term={t(strings.pages.execution.baselineSetSourceEvidence)}>
					  <span className="text-xs text-app-muted-foreground">
						{baselineSet.baselineSet.pathSnapshots.map((snapshot) => `${snapshot.name}${snapshot.branch ? ` (${snapshot.branch})` : ""}`).join(", ")}
					  </span>
					</MetaRow>
				  ) : null}
				  {execution?.scopeAmendments.length ? (
					<MetaRow term={t(strings.pages.execution.scopeAmendmentsHeading)}>
					  <span data-testid="execution-scope-amendments" className="text-xs text-app-muted-foreground">
						{execution.scopeAmendments.map((amendment) => `${amendment.phaseId}: ${amendment.oldMinimum.join(", ")} → ${amendment.newMinimum.join(", ")} (${amendment.reason})`).join(" · ")}
					  </span>
					</MetaRow>
				  ) : null}
				  {execution?.degradedReason ? (
					<MetaRow term={t(strings.pages.execution.executionStateHeading)}>
					  <span className="text-xs text-app-warning">{execution.degradedReason}</span>
					</MetaRow>
				  ) : null}
				</>
			  ) : null}
            </dl>
          </Card>

          <Card className="bg-app-surface-muted">
            <h4 className="text-sm font-semibold">{t(strings.pages.execution.logSummaryHeading)}</h4>
            <div className="mt-2">
              <LogSummaryView summary={context?.logSummary} testId={selectors.execution.logSummary} />
            </div>
          </Card>

          <Card className="bg-app-surface-muted">
            <h4 className="text-sm font-semibold">{t(strings.pages.execution.feedbackCheckpointHeading)}</h4>
            <div className="mt-2 flex flex-col gap-2 text-sm">
              <p data-testid={selectors.execution.feedbackCheckpoint} className="text-app-muted-foreground">
                {context?.feedbackCheckpoint?.summary || t(strings.pages.execution.feedbackReviewDefault)}
              </p>
              <div className="flex flex-wrap gap-2 text-xs">
                <span className="rounded-pill bg-app-info/15 px-2 py-0.5 text-app-info">
                  {t(strings.pages.execution.feedbackDecisions)} {context?.feedbackCheckpoint?.decisions ?? 0}
                </span>
                <span className="rounded-pill bg-app-info/15 px-2 py-0.5 text-app-info">
                  {t(strings.pages.execution.feedbackFindings)} {context?.feedbackCheckpoint?.findings ?? 0}
                </span>
                <span className="rounded-pill bg-app-info/15 px-2 py-0.5 text-app-info">
                  {t(strings.pages.execution.feedbackBugs)} {context?.feedbackCheckpoint?.bugReports ?? 0}
                </span>
                <span className="rounded-pill bg-app-info/15 px-2 py-0.5 text-app-info">
                  {t(strings.pages.execution.feedbackRecords)} {context?.feedbackCheckpoint?.records ?? 0}
                </span>
              </div>
              <Button
                type="button"
                size="sm"
                variant="outline"
                data-testid={selectors.execution.noFeedbackButton}
                disabled={busy || currentPhaseId.length === 0 || Boolean(context?.feedbackCheckpoint?.satisfied)}
                onClick={handleNoFeedback}
                className="w-fit"
              >
                {t(strings.pages.execution.confirmNoFeedback)}
              </Button>
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

        <SectionPanel title={t(strings.pages.execution.bugReportsHeading)} headingId="execution-bugs-heading">
          <div className="flex flex-col gap-2">
            <Input
              data-testid={selectors.execution.bugTitle}
              value={bugTitle}
              onChange={(e) => setBugTitle(e.target.value)}
              placeholder={t(strings.pages.execution.bugTitleLabel)}
              aria-label={t(strings.pages.execution.bugTitleLabel)}
            />
            <Textarea
              data-testid={selectors.execution.bugDetail}
              value={bugDetail}
              onChange={(e) => setBugDetail(e.target.value)}
              rows={2}
              aria-label={t(strings.pages.execution.bugDetailLabel)}
            />
            <div className="grid gap-2 md:grid-cols-2">
              <Input value={bugSignalType} onChange={(e) => setBugSignalType(e.target.value)} placeholder="Signal type (for example, regression)" aria-label="Bug signal type" />
              <Input value={bugSeverity} onChange={(e) => setBugSeverity(e.target.value)} placeholder="Report severity (blocker, major, minor)" aria-label="Bug report severity" />
              <Input value={bugScenario} onChange={(e) => setBugScenario(e.target.value)} placeholder="Affected scenario" aria-label="Affected scenario" />
              <Input value={bugHonestyFlags} onChange={(e) => setBugHonestyFlags(e.target.value)} placeholder="Honesty flags (comma-separated)" aria-label="Bug honesty flags" />
            </div>
            <Textarea value={bugRepro} onChange={(e) => setBugRepro(e.target.value)} rows={2} placeholder="Reproduction steps (comma-separated)" aria-label="Bug reproduction steps" />
            <Textarea value={bugExpected} onChange={(e) => setBugExpected(e.target.value)} rows={2} placeholder="Expected behavior" aria-label="Bug expected behavior" />
            <Textarea value={bugActual} onChange={(e) => setBugActual(e.target.value)} rows={2} placeholder="Actual behavior" aria-label="Bug actual behavior" />
            <Textarea value={bugDescription} onChange={(e) => setBugDescription(e.target.value)} rows={3} placeholder="Taxonomy description" aria-label="Bug taxonomy description" />
            <Button
              type="button"
              size="sm"
              data-testid={selectors.execution.recordBugButton}
              disabled={busy || bugTitle.trim().length === 0}
              onClick={handleRecordBug}
              className="w-fit"
            >
              {t(strings.pages.execution.fileBugReport)}
            </Button>
          </div>
          {state.bugs.length === 0 ? (
            <p className="text-sm text-app-muted-foreground">{t(strings.pages.execution.noBugReports)}</p>
          ) : (
            <ul className="flex flex-col gap-2">
              {state.bugs.map((bug) => (
                <li key={bug.id} data-testid={selectors.execution.logEntry({ id: bug.id })} className="rounded-control border border-app-border bg-app-surface-muted px-3 py-2 text-sm">
                  <p className="font-medium text-app-foreground">{bug.title}</p>
                  {bug.detail ? <p className="text-app-muted-foreground">{bug.detail}</p> : null}
                  <CaptureState entry={bug} onRetry={handleRetrySync} />
                </li>
              ))}
            </ul>
          )}
        </SectionPanel>

        <SectionPanel title={t(strings.pages.execution.reusableRecordsHeading)} headingId="execution-records-heading">
          <div className="flex flex-col gap-2">
            <Input
              data-testid={selectors.execution.recordTitle}
              value={recordTitle}
              onChange={(e) => setRecordTitle(e.target.value)}
              placeholder={t(strings.pages.execution.recordTitleLabel)}
              aria-label={t(strings.pages.execution.recordTitleLabel)}
            />
            <Textarea
              data-testid={selectors.execution.recordDetail}
              value={recordDetail}
              onChange={(e) => setRecordDetail(e.target.value)}
              rows={2}
              aria-label={t(strings.pages.execution.recordDetailLabel)}
            />
            <div className="grid gap-2 md:grid-cols-2">
              <Input value={recordKind} onChange={(e) => setRecordKind(e.target.value)} placeholder="Record kind" aria-label="Record kind" />
              <Input value={recordScenario} onChange={(e) => setRecordScenario(e.target.value)} placeholder="Target scenario" aria-label="Record scenario" />
              <Input value={recordOutcome} onChange={(e) => setRecordOutcome(e.target.value)} placeholder="Outcome" aria-label="Record outcome" />
            </div>
            <Textarea value={recordTrigger} onChange={(e) => setRecordTrigger(e.target.value)} rows={2} placeholder="Trigger or goal" aria-label="Record trigger" />
            <Textarea value={recordApproach} onChange={(e) => setRecordApproach(e.target.value)} rows={2} placeholder="Approach" aria-label="Record approach" />
            <Textarea value={recordEvidence} onChange={(e) => setRecordEvidence(e.target.value)} rows={2} placeholder="Validation evidence" aria-label="Record evidence" />
            <Button
              type="button"
              size="sm"
              data-testid={selectors.execution.recordRecordButton}
              disabled={busy || recordTitle.trim().length === 0}
              onClick={handleRecordRecord}
              className="w-fit"
            >
              {t(strings.pages.execution.captureRecord)}
            </Button>
          </div>
          {state.records.length === 0 ? (
            <p className="text-sm text-app-muted-foreground">{t(strings.pages.execution.noRecords)}</p>
          ) : (
            <ul className="flex flex-col gap-2">
              {state.records.map((record) => (
                <li key={record.id} data-testid={selectors.execution.logEntry({ id: record.id })} className="rounded-control border border-app-border bg-app-surface-muted px-3 py-2 text-sm">
                  <p className="font-medium text-app-foreground">{record.title}</p>
                  {record.detail ? <p className="text-app-muted-foreground">{record.detail}</p> : null}
                  <CaptureState entry={record} onRetry={handleRetrySync} />
                </li>
              ))}
            </ul>
          )}
        </SectionPanel>

        <SectionPanel title={t(strings.pages.execution.notesHeading)} headingId="execution-notes-heading">
          <div className="flex flex-col gap-2">
            <Input
              data-testid={selectors.execution.noteTitle}
              value={noteTitle}
              onChange={(e) => setNoteTitle(e.target.value)}
              placeholder={t(strings.pages.execution.noteTitleLabel)}
              aria-label={t(strings.pages.execution.noteTitleLabel)}
            />
            <Textarea
              data-testid={selectors.execution.noteDetail}
              value={noteDetail}
              onChange={(e) => setNoteDetail(e.target.value)}
              rows={2}
              aria-label={t(strings.pages.execution.noteDetailLabel)}
            />
            <Button
              type="button"
              size="sm"
              data-testid={selectors.execution.recordNoteButton}
              disabled={busy || noteTitle.trim().length === 0}
              onClick={handleRecordNote}
              className="w-fit"
            >
              {t(strings.pages.execution.recordNote)}
            </Button>
          </div>
          {state.notes.length === 0 ? (
            <p className="text-sm text-app-muted-foreground">{t(strings.pages.execution.noNotes)}</p>
          ) : (
            <ul className="flex flex-col gap-2">
              {state.notes.map((note) => (
                <li key={note.id} className="rounded-control border border-app-border bg-app-surface-muted px-3 py-2 text-sm">
                  <p className="font-medium text-app-foreground">{note.title}</p>
                  {note.detail ? <p className="text-app-muted-foreground">{note.detail}</p> : null}
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
