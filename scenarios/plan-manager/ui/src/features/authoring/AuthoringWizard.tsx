import { useState } from "react";
import { Link } from "react-router-dom";
import { create } from "@bufbuild/protobuf";
import { CheckCircle2, Sparkles, Wand2 } from "lucide-react";

import {
  acceptContextCandidate,
  acceptReferenceCandidate,
  addPhase,
  autofill,
  discoverContextCandidates,
  finalize,
  getSession,
  nextPhase,
  rejectContextCandidate,
  rejectReferenceCandidate,
  removeRelevantContextItem,
  startSession,
  submitRelevantContextItem,
  submitPhaseField,
  submitSection,
  suggestReferences,
  validateStructure,
} from "../../api/authoring";
import { SectionPanel } from "../../components/Surfaces";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Textarea } from "../../components/ui/textarea";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { errorMessage } from "../../lib/errorMessage";
import { useTranslation } from "../../i18n";
import type {
  AuthoringProgress,
  AuthoringSession,
  AutofillResult,
  ContextCandidate,
  PhaseDraft,
  ReferenceCandidate,
  Section,
  StructureViolation,
} from "@vrooli/proto-types/plan-manager/v1/authoring/authoring_pb";
import {
  ReferenceKind,
  RelevantContextItemSchema,
  RelevantContextKind,
  RelevantContextRepeatPolicy,
  RelevantContextScope,
  RelevantContextSource,
  RelevantContextStatus,
  type GuidedStep,
  type RelevantContextItem,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

/**
 * Local wizard state. The session is the source of truth once started. Because
 * mutations no longer echo the full session (focused response contract), the
 * session is re-hydrated explicitly via getSession after each mutation
 * (read-after-write); progress carries the compact navigation snapshot.
 */
interface WizardState {
  session?: AuthoringSession;
  progress?: AuthoringProgress;
  violations: StructureViolation[];
  autofillResults: AutofillResult[];
  step?: GuidedStep;
  finalizedSlug?: string;
  finalizedPlanId?: string;
}

const phaseFields = [
  "references",
  "relevant_context",
  "reminders",
  "acceptance",
  "no_code_refs_reason",
];

const contextKinds = [
  RelevantContextKind.SKILL,
  RelevantContextKind.DOC,
  RelevantContextKind.COMMAND,
  RelevantContextKind.SEARCH,
  RelevantContextKind.CODE_REF,
  RelevantContextKind.REQ_REF,
  RelevantContextKind.NOTE,
];

const contextKindLabels: Record<RelevantContextKind, string> = {
  [RelevantContextKind.UNSPECIFIED]: "context",
  [RelevantContextKind.SKILL]: "skill",
  [RelevantContextKind.DOC]: "doc",
  [RelevantContextKind.COMMAND]: "command",
  [RelevantContextKind.SEARCH]: "search",
  [RelevantContextKind.CODE_REF]: "code_ref",
  [RelevantContextKind.REQ_REF]: "req_ref",
  [RelevantContextKind.NOTE]: "note",
};

function ContextList({
  items,
  onRemove,
  busy,
}: {
  items: readonly RelevantContextItem[];
  onRemove?: (item: RelevantContextItem) => void;
  busy?: boolean;
}) {
  const { t } = useTranslation();
  if (items.length === 0) {
    return <p className="text-sm text-app-muted-foreground">{t(strings.pages.authoring.contextEmpty)}</p>;
  }
  return (
    <ul data-testid={selectors.authoring.contextItems} className="flex flex-col gap-1.5">
      {items.map((item, i) => (
        <li
          key={item.id || `${item.kind}-${item.label}-${i}`}
          className="rounded-control border border-app-border bg-app-surface-muted px-3 py-2 text-sm"
        >
          <div className="flex flex-wrap items-center gap-2">
            <span className="rounded-pill bg-app-info/15 px-2 py-0.5 text-xs text-app-info">
              {contextKindLabels[item.kind]}
            </span>
            <span className="font-medium text-app-foreground">{item.label || item.target || item.command}</span>
            {onRemove && item.id ? (
              <Button
                type="button"
                size="sm"
                variant="outline"
                data-testid={selectors.authoring.contextRemoveButton}
                disabled={busy}
                onClick={() => onRemove(item)}
                className="ms-auto"
              >
                {t(strings.pages.authoring.contextRemove)}
              </Button>
            ) : null}
          </div>
          {item.reason ? <p className="mt-1 text-xs text-app-muted-foreground">{item.reason}</p> : null}
          {item.instruction ? <p className="mt-1 text-xs text-app-foreground">{item.instruction}</p> : null}
          {item.command || item.target ? (
            <code className="mt-2 block break-all rounded-control bg-app-surface px-2 py-1 font-mono text-xs">
              {item.command || item.target}
            </code>
          ) : null}
        </li>
      ))}
    </ul>
  );
}

function contextCandidateLabel(candidate: ContextCandidate) {
  const item = candidate.item;
  return item?.label || item?.target || item?.command || candidate.concept || candidate.id;
}

function ContextCandidateList({
  candidates,
  onAccept,
  onReject,
  busy,
}: {
  candidates: readonly ContextCandidate[];
  onAccept: (candidateId: string) => void;
  onReject: (candidateId: string) => void;
  busy: boolean;
}) {
  const { t } = useTranslation();
  if (candidates.length === 0) {
    return <p className="text-sm text-app-muted-foreground">{t(strings.pages.authoring.contextCandidatesEmpty)}</p>;
  }
  return (
    <ul data-testid={selectors.authoring.contextCandidates} className="flex flex-col gap-1.5">
      {candidates.map((candidate) => {
        const item = candidate.item;
        const pending = candidate.status === "" || candidate.status === "pending";
        return (
          <li
            key={candidate.id}
            className="rounded-control border border-app-border bg-app-surface-muted px-3 py-2 text-sm"
          >
            <div className="flex flex-wrap items-center gap-2">
              <span className="font-medium text-app-foreground">{contextCandidateLabel(candidate)}</span>
              <span className="rounded-pill bg-app-info/15 px-2 py-0.5 text-xs text-app-info">
                {candidate.status || "pending"}
              </span>
              {candidate.degraded ? (
                <span className="rounded-pill bg-app-warning/15 px-2 py-0.5 text-xs text-app-warning">
                  {t(strings.common.degradedBadge)}
                </span>
              ) : null}
            </div>
            {item?.reason ? <p className="mt-1 text-xs text-app-muted-foreground">{item.reason}</p> : null}
            {candidate.detail ? <p className="mt-1 text-xs text-app-muted-foreground">{candidate.detail}</p> : null}
            {item?.command || item?.target ? (
              <code className="mt-2 block break-all rounded-control bg-app-surface px-2 py-1 font-mono text-xs">
                {item.command || item.target}
              </code>
            ) : null}
            {candidate.rejectionReason ? (
              <p className="mt-1 text-xs text-app-danger">{candidate.rejectionReason}</p>
            ) : null}
            {pending ? (
              <div className="mt-2 flex flex-wrap gap-2">
                <Button
                  type="button"
                  size="sm"
                  data-testid={selectors.authoring.contextCandidateAcceptButton}
                  disabled={busy}
                  onClick={() => onAccept(candidate.id)}
                >
                  {t(strings.pages.authoring.contextAccept)}
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  data-testid={selectors.authoring.contextCandidateRejectButton}
                  disabled={busy}
                  onClick={() => onReject(candidate.id)}
                >
                  {t(strings.pages.authoring.contextReject)}
                </Button>
              </div>
            ) : null}
          </li>
        );
      })}
    </ul>
  );
}

function referenceMarker(kind: ReferenceKind): string {
  switch (kind) {
    case ReferenceKind.REQ:
      return "REQ";
    case ReferenceKind.DOC:
      return "DOC";
    default:
      return "CODE";
  }
}

function referenceLocatorLabel(candidate: ReferenceCandidate): string {
  const ref = candidate.reference;
  if (!ref) return candidate.id;
  return `[${referenceMarker(ref.kind)}: ${ref.target}]`;
}

// ReferenceCandidateList mirrors ContextCandidateList: search-hub reference
// suggestions are reviewable — only accepted candidates enter the references
// section. A raw suggestion never satisfies the references gate.
function ReferenceCandidateList({
  candidates,
  onAccept,
  onReject,
  busy,
}: {
  candidates: readonly ReferenceCandidate[];
  onAccept: (candidateId: string) => void;
  onReject: (candidateId: string) => void;
  busy: boolean;
}) {
  const { t } = useTranslation();
  if (candidates.length === 0) {
    return <p className="text-sm text-app-muted-foreground">{t(strings.pages.authoring.referenceCandidatesEmpty)}</p>;
  }
  return (
    <ul data-testid={selectors.authoring.referenceCandidates} className="flex flex-col gap-1.5">
      {candidates.map((candidate) => {
        const pending = candidate.status === "" || candidate.status === "pending";
        return (
          <li
            key={candidate.id}
            className="rounded-control border border-app-border bg-app-surface-muted px-3 py-2 text-sm"
          >
            <div className="flex flex-wrap items-center gap-2">
              <code className="break-all font-mono text-xs text-app-foreground">{referenceLocatorLabel(candidate)}</code>
              <span className="rounded-pill bg-app-info/15 px-2 py-0.5 text-xs text-app-info">
                {candidate.status || "pending"}
              </span>
              {candidate.degraded ? (
                <span className="rounded-pill bg-app-warning/15 px-2 py-0.5 text-xs text-app-warning">
                  {t(strings.common.degradedBadge)}
                </span>
              ) : null}
            </div>
            {candidate.source ? (
              <p className="mt-1 text-xs text-app-muted-foreground">{candidate.source}</p>
            ) : null}
            {candidate.detail ? <p className="mt-1 text-xs text-app-muted-foreground">{candidate.detail}</p> : null}
            {candidate.rejectionReason ? (
              <p className="mt-1 text-xs text-app-danger">{candidate.rejectionReason}</p>
            ) : null}
            {pending ? (
              <div className="mt-2 flex flex-wrap gap-2">
                <Button
                  type="button"
                  size="sm"
                  data-testid={selectors.authoring.referenceCandidateAcceptButton}
                  disabled={busy}
                  onClick={() => onAccept(candidate.id)}
                >
                  {t(strings.pages.authoring.referenceAccept)}
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  data-testid={selectors.authoring.referenceCandidateRejectButton}
                  disabled={busy}
                  onClick={() => onReject(candidate.id)}
                >
                  {t(strings.pages.authoring.referenceReject)}
                </Button>
              </div>
            ) : null}
          </li>
        );
      })}
    </ul>
  );
}

function SectionRow({
  section,
  active,
  onSelect,
}: {
  section: Section;
  active: boolean;
  onSelect: () => void;
}) {
  const { t } = useTranslation();
  return (
    <li data-testid={selectors.authoring.section({ key: section.key })}>
      <button
        type="button"
        onClick={onSelect}
        aria-current={active ? "true" : undefined}
        className={[
          "flex w-full items-center justify-between gap-2 rounded-control border px-3 py-2 text-left text-sm transition-colors",
          active
            ? "border-app-primary bg-app-primary/10 text-app-foreground"
            : "border-app-border bg-app-surface text-app-foreground hover:bg-app-surface-muted",
        ].join(" ")}
      >
        <span className="flex items-center gap-2">
          {section.filled ? (
            <CheckCircle2 aria-hidden="true" className="h-4 w-4 text-app-success" />
          ) : (
            <span
              aria-hidden="true"
              className="h-2 w-2 rounded-full bg-app-muted-foreground"
            />
          )}
          {section.label || section.key}
        </span>
        <span className="flex items-center gap-1 text-xs text-app-muted-foreground">
          {section.mandatory ? <span>{t(strings.pages.authoring.mandatory)}</span> : null}
          {section.autofilled ? (
            <span className="rounded-pill bg-app-info/15 px-1.5 text-app-info">
              {t(strings.pages.authoring.autofilled)}
            </span>
          ) : null}
        </span>
      </button>
    </li>
  );
}

function StepPanel({ step }: { step?: GuidedStep }) {
  if (!step || (!step.title && !step.summary)) return null;
  return (
    <SectionPanel title={step.title || step.stepKind} headingId="authoring-guidance-heading">
      <div data-testid={selectors.authoring.guidance} className="flex flex-col gap-3">
        {step.summary ? <p className="text-sm text-app-muted-foreground">{step.summary}</p> : null}
        {step.instructions.length > 0 ? (
          <ul className="flex flex-col gap-1 text-sm text-app-foreground">
            {step.instructions.map((item, i) => (
              <li key={`instruction-${i}`}>{item}</li>
            ))}
          </ul>
        ) : null}
        {step.requiredInputs.length > 0 ? (
          <div className="flex flex-wrap gap-1.5">
            {step.requiredInputs.map((item) => (
              <span key={item} className="rounded-pill bg-app-info/15 px-2 py-0.5 text-xs text-app-info">
                {item}
              </span>
            ))}
          </div>
        ) : null}
        {step.nextActions.length > 0 ? (
          <div className="flex flex-col gap-1 text-xs text-app-muted-foreground">
            {step.nextActions.map((action) => (
              <code key={action.id || action.label} className="rounded-control bg-app-surface-muted px-2 py-1">
                {action.argv.join(" ")}
              </code>
            ))}
          </div>
        ) : null}
      </div>
    </SectionPanel>
  );
}

function PhaseRow({
  phase,
  active,
  onSelect,
}: {
  phase: PhaseDraft;
  active: boolean;
  onSelect: () => void;
}) {
  return (
    <li>
      <button
        type="button"
        onClick={onSelect}
        aria-current={active ? "true" : undefined}
        className={[
          "flex w-full items-center justify-between gap-2 rounded-control border px-3 py-2 text-left text-sm transition-colors",
          active
            ? "border-app-primary bg-app-primary/10 text-app-foreground"
            : "border-app-border bg-app-surface text-app-foreground hover:bg-app-surface-muted",
        ].join(" ")}
      >
        <span>
          {phase.order}. {phase.title || "Untitled phase"}
        </span>
        <span className="text-xs text-app-muted-foreground">
          {phase.references.length > 0 || phase.noCodeRefsReason ? "refs" : "needs refs"}
        </span>
      </button>
    </li>
  );
}

/**
 * AuthoringWizard — the guided composer. Start a session, walk + submit
 * sections (surfacing structure violations honestly), run autofill (showing
 * which sources filled and which degraded), and finalize into a structured plan.
 */
export function AuthoringWizard() {
  const { t } = useTranslation();
  const [title, setTitle] = useState("");
  const [templateId, setTemplateId] = useState("");
  const [state, setState] = useState<WizardState>({ violations: [], autofillResults: [] });
  const [activeKey, setActiveKey] = useState("");
  const [draft, setDraft] = useState("");
  const [phaseTitle, setPhaseTitle] = useState("");
  const [phaseIntent, setPhaseIntent] = useState("");
  const [activePhaseId, setActivePhaseId] = useState("");
  const [phaseField, setPhaseField] = useState<string>("references");
  const [phaseDraft, setPhaseDraft] = useState("");
  const [contextKind, setContextKind] = useState<RelevantContextKind>(RelevantContextKind.COMMAND);
  const [contextPhaseScoped, setContextPhaseScoped] = useState(false);
  const [contextLabel, setContextLabel] = useState("");
  const [contextReason, setContextReason] = useState("");
  const [contextInstruction, setContextInstruction] = useState("");
  const [contextCommand, setContextCommand] = useState("");
  const [contextTarget, setContextTarget] = useState("");
  const [contextConcepts, setContextConcepts] = useState("");
  const [contextComplexity, setContextComplexity] = useState("architectural");
  const [contextRejectReason, setContextRejectReason] = useState("");
  const [referenceRejectReason, setReferenceRejectReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const session = state.session;
  const sections = session?.sections ?? [];
  const activeSection = sections.find((s) => s.key === activeKey);
  const phases = session?.phaseDrafts ?? [];
  const activePhase = phases.find((p) => p.id === activePhaseId) ?? phases[0];

  const run = async (fn: () => Promise<void>) => {
    setBusy(true);
    setError(null);
    try {
      await fn();
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };

  const handleStart = (e: React.FormEvent) => {
    e.preventDefault();
    if (title.trim().length === 0) return;
    void run(async () => {
      const res = await startSession(title.trim(), "", templateId.trim());
      if (res.session) {
        setState({ session: res.session, violations: [], autofillResults: [], step: res.step });
        const first = res.session.currentSectionKey || res.session.sections[0]?.key || "";
        setActiveKey(first);
        setActivePhaseId(res.session.currentPhaseId || res.session.phaseDrafts[0]?.id || "");
        setDraft(res.session.sections.find((sec) => sec.key === first)?.content ?? "");
      }
    });
  };

  const selectSection = (section: Section) => {
    setActiveKey(section.key);
    setDraft(section.content);
  };

  // hydrate re-reads the full session (read-after-write) and folds in the
  // mutation's own violations/progress/step. Mutations no longer echo the
  // session, so this explicit read is how the wizard stays current.
  const hydrate = async (mutation?: {
    violations?: StructureViolation[];
    progress?: AuthoringProgress;
    step?: GuidedStep;
  }) => {
    if (!session) return undefined;
    const fresh = await getSession(session.id);
    setState((prev) => ({
      ...prev,
      session: fresh.session ?? prev.session,
      progress: mutation?.progress ?? prev.progress,
      violations: mutation?.violations ?? prev.violations,
      step: mutation?.step ?? fresh.step ?? prev.step,
    }));
    return fresh.session;
  };

  const handleSubmitSection = () => {
    if (!session || !activeSection) return;
    void run(async () => {
      const res = await submitSection(session.id, activeSection.key, draft);
      await hydrate(res);
    });
  };

  const handleAddPhase = () => {
    if (!session || phaseTitle.trim().length === 0) return;
    void run(async () => {
      const res = await addPhase(session.id, phaseTitle.trim(), phaseIntent.trim());
      await hydrate(res);
      setActivePhaseId(res.phase?.id || res.progress?.currentPhaseId || "");
      setPhaseTitle("");
      setPhaseIntent("");
    });
  };

  const handleSubmitPhaseField = () => {
    if (!session || !activePhase) return;
    const phaseId = activePhase.id;
    void run(async () => {
      const res = await submitPhaseField(session.id, phaseId, phaseField, phaseDraft);
      await hydrate(res);
      setActivePhaseId(res.progress?.currentPhaseId || phaseId);
      setPhaseDraft("");
    });
  };

  const handleNextPhase = () => {
    if (!session) return;
    void run(async () => {
      const res = await nextPhase(session.id);
      setState((prev) => ({ ...prev, step: res.step }));
      if (res.phase) {
        setActivePhaseId(res.phase.id);
      }
    });
  };

  const handleValidate = () => {
    if (!session) return;
    void run(async () => {
      const res = await validateStructure(session.id);
      setState((prev) => ({ ...prev, violations: res.violations, step: res.step }));
    });
  };

  const handleAutofill = () => {
    if (!session) return;
    void run(async () => {
      const res = await autofill(session.id);
      await hydrate(res);
      setState((prev) => ({ ...prev, autofillResults: res.results }));
    });
  };

  const handleSuggestReferences = () => {
    if (!session) return;
    void run(async () => {
      const res = await suggestReferences(session.id);
      await hydrate(res);
    });
  };

  const handleAcceptReference = (candidateId: string) => {
    if (!session) return;
    void run(async () => {
      const res = await acceptReferenceCandidate(session.id, candidateId);
      await hydrate(res);
    });
  };

  const handleRejectReference = (candidateId: string) => {
    if (!session) return;
    void run(async () => {
      const res = await rejectReferenceCandidate(
        session.id,
        candidateId,
        referenceRejectReason.trim() || "not relevant",
      );
      await hydrate(res);
      setReferenceRejectReason("");
    });
  };

  const handleSubmitContext = () => {
    if (!session) return;
    const phaseId = contextPhaseScoped ? activePhase?.id ?? "" : "";
    if (contextPhaseScoped && phaseId.length === 0) return;
    void run(async () => {
      const item = create(RelevantContextItemSchema, {
        kind: contextKind,
        scope: contextPhaseScoped ? RelevantContextScope.PHASE : RelevantContextScope.GLOBAL,
        phaseId,
        label: contextLabel.trim(),
        reason: contextReason.trim(),
        instruction: contextInstruction.trim(),
        command: contextCommand.trim(),
        target: contextTarget.trim(),
        required: true,
        repeatPolicy: contextPhaseScoped
          ? RelevantContextRepeatPolicy.PHASE_ENTRY
          : RelevantContextRepeatPolicy.ONCE_PER_EXECUTION,
        source: RelevantContextSource.AUTHORED,
        status: RelevantContextStatus.READY,
      });
      const res = await submitRelevantContextItem(session.id, phaseId, item);
      await hydrate(res);
      // Only clear inputs when the item was accepted (no violations); a rejected
      // item keeps its draft so the operator can correct it.
      if (res.violations.length === 0) {
        setContextLabel("");
        setContextReason("");
        setContextInstruction("");
        setContextCommand("");
        setContextTarget("");
      }
    });
  };

  const handleRemoveContext = (item: RelevantContextItem) => {
    if (!session || !item.id) return;
    const phaseId = item.scope === RelevantContextScope.PHASE ? item.phaseId : "";
    void run(async () => {
      const res = await removeRelevantContextItem(session.id, phaseId, item.id);
      await hydrate(res);
    });
  };

  const handleDiscoverContext = () => {
    if (!session) return;
    const concepts = contextConcepts
      .split(",")
      .map((concept) => concept.trim())
      .filter(Boolean);
    void run(async () => {
      const res = await discoverContextCandidates(session.id, concepts, contextComplexity.trim());
      await hydrate(res);
    });
  };

  const handleAcceptCandidate = (candidateId: string) => {
    if (!session) return;
    const phaseId = contextPhaseScoped ? activePhase?.id ?? "" : "";
    if (contextPhaseScoped && phaseId.length === 0) return;
    void run(async () => {
      const res = await acceptContextCandidate(session.id, candidateId, phaseId);
      await hydrate(res);
    });
  };

  const handleRejectCandidate = (candidateId: string) => {
    if (!session) return;
    void run(async () => {
      const res = await rejectContextCandidate(
        session.id,
        candidateId,
        contextRejectReason.trim() || "not relevant",
      );
      await hydrate(res);
      setContextRejectReason("");
    });
  };

  const handleFinalize = () => {
    if (!session) return;
    void run(async () => {
      const res = await finalize(session.id);
      const plan = res.plan;
      if (plan) {
        setState((prev) => ({
          ...prev,
          step: res.step,
          finalizedSlug: plan.slug,
          finalizedPlanId: plan.id,
        }));
      }
    });
  };

  if (!session) {
    return (
      <SectionPanel title={t(strings.pages.authoring.startHeading)} headingId="authoring-start-heading">
        <form
          data-testid={selectors.authoring.startForm}
          onSubmit={handleStart}
          className="flex flex-col gap-3"
        >
          <label className="flex flex-col gap-1 text-sm">
            <span className="text-xs font-medium text-app-muted-foreground">
              {t(strings.pages.authoring.startTitleLabel)}
            </span>
            <Input
              data-testid={selectors.authoring.titleInput}
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={t(strings.pages.authoring.startTitlePlaceholder)}
            />
          </label>
          <label className="flex flex-col gap-1 text-sm">
            <span className="text-xs font-medium text-app-muted-foreground">
              {t(strings.pages.authoring.startTemplateLabel)}
            </span>
            <Input
              data-testid={selectors.authoring.templateInput}
              value={templateId}
              onChange={(e) => setTemplateId(e.target.value)}
            />
          </label>
          {error ? (
            <p role="alert" className="text-sm text-app-danger">
              {errorMessage(error, t)}
            </p>
          ) : null}
          <Button
            type="submit"
            data-testid={selectors.authoring.startButton}
            disabled={busy || title.trim().length === 0}
            className="w-fit"
          >
            {t(strings.pages.authoring.start)}
          </Button>
        </form>
      </SectionPanel>
    );
  }

  if (state.finalizedSlug !== undefined) {
    return (
      <div
        data-testid={selectors.authoring.finalizedBanner}
        role="status"
        className="flex flex-col items-start gap-3 rounded-panel border border-app-success/40 bg-app-success/10 p-6"
      >
        <p className="flex items-center gap-2 text-sm font-medium text-app-success">
          <CheckCircle2 aria-hidden="true" className="h-5 w-5" />
          {t(strings.pages.authoring.finalized, { slug: state.finalizedSlug })}
        </p>
        {state.finalizedPlanId ? (
          <Button asChild variant="outline" size="sm">
            <Link to={`/plans/${state.finalizedPlanId}`}>
              {t(strings.pages.authoring.viewFinalizedPlan)}
            </Link>
          </Button>
        ) : null}
      </div>
    );
  }

  return (
    <div className="grid gap-4 lg:grid-cols-[18rem_1fr]">
      <SectionPanel title={t(strings.pages.authoring.sectionsHeading)} headingId="authoring-sections-heading">
        <ul data-testid={selectors.authoring.sections} className="flex flex-col gap-1.5">
          {sections.map((section) => (
            <SectionRow
              key={section.key}
              section={section}
              active={section.key === activeKey}
              onSelect={() => selectSection(section)}
            />
          ))}
        </ul>
      </SectionPanel>

      <div className="flex flex-col gap-4">
        {error ? (
          <p role="alert" className="rounded-control bg-app-danger/10 px-3 py-2 text-sm text-app-danger">
            {errorMessage(error, t)}
          </p>
        ) : null}

        <StepPanel step={state.step} />

        {activeSection ? (
          <SectionPanel
            title={activeSection.label || activeSection.key}
            headingId="authoring-current-heading"
          >
            <label className="flex flex-col gap-1 text-sm">
              <span className="sr-only">{t(strings.pages.authoring.sectionContentLabel)}</span>
              <Textarea
                data-testid={selectors.authoring.contentInput}
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                placeholder={t(strings.pages.authoring.sectionContentPlaceholder)}
                rows={8}
                aria-label={t(strings.pages.authoring.sectionContentLabel)}
              />
            </label>
            <div className="flex flex-wrap gap-2">
              <Button
                type="button"
                size="sm"
                data-testid={selectors.authoring.submitButton}
                disabled={busy}
                onClick={handleSubmitSection}
              >
                {t(strings.pages.authoring.submitSection)}
              </Button>
              <Button
                type="button"
                size="sm"
                variant="outline"
                data-testid={selectors.authoring.validateButton}
                disabled={busy}
                onClick={handleValidate}
              >
                {t(strings.pages.authoring.validateStructure)}
              </Button>
            </div>
          </SectionPanel>
        ) : null}

        <SectionPanel title={t(strings.pages.authoring.phasesHeading)} headingId="authoring-phases-heading">
          <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_minmax(16rem,1fr)]">
            <div className="flex flex-col gap-3">
              <div className="grid gap-2 sm:grid-cols-2">
                <Input
                  data-testid={selectors.authoring.phaseTitleInput}
                  value={phaseTitle}
                  onChange={(e) => setPhaseTitle(e.target.value)}
                  placeholder={t(strings.pages.authoring.phaseTitlePlaceholder)}
                  aria-label={t(strings.pages.authoring.phaseTitleLabel)}
                />
                <Input
                  data-testid={selectors.authoring.phaseIntentInput}
                  value={phaseIntent}
                  onChange={(e) => setPhaseIntent(e.target.value)}
                  placeholder={t(strings.pages.authoring.phaseIntentPlaceholder)}
                  aria-label={t(strings.pages.authoring.phaseIntentLabel)}
                />
              </div>
              <div className="flex flex-wrap gap-2">
                <Button
                  type="button"
                  size="sm"
                  data-testid={selectors.authoring.phaseAddButton}
                  disabled={busy || phaseTitle.trim().length === 0}
                  onClick={handleAddPhase}
                >
                  {t(strings.pages.authoring.addPhase)}
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  data-testid={selectors.authoring.phaseNextButton}
                  disabled={busy}
                  onClick={handleNextPhase}
                >
                  {t(strings.pages.authoring.nextPhase)}
                </Button>
              </div>
              {phases.length === 0 ? (
                <p className="text-sm text-app-muted-foreground">{t(strings.pages.authoring.noPhases)}</p>
              ) : (
                <ul data-testid={selectors.authoring.phases} className="flex flex-col gap-1.5">
                  {phases.map((phase) => (
                    <PhaseRow
                      key={phase.id}
                      phase={phase}
                      active={phase.id === activePhase?.id}
                      onSelect={() => setActivePhaseId(phase.id)}
                    />
                  ))}
                </ul>
              )}
            </div>
            <div className="flex flex-col gap-2">
              <select
                data-testid={selectors.authoring.phaseFieldSelect}
                value={phaseField}
                onChange={(e) => setPhaseField(e.target.value)}
                className="h-10 rounded-control border border-app-border bg-app-surface px-3 text-sm"
                aria-label={t(strings.pages.authoring.phaseFieldLabel)}
              >
                {phaseFields.map((field) => (
                  <option key={field} value={field}>
                    {field}
                  </option>
                ))}
              </select>
              <Textarea
                data-testid={selectors.authoring.phaseFieldInput}
                value={phaseDraft}
                onChange={(e) => setPhaseDraft(e.target.value)}
                placeholder={t(strings.pages.authoring.phaseFieldContentPlaceholder)}
                rows={5}
                aria-label={t(strings.pages.authoring.phaseFieldContentLabel)}
              />
              <Button
                type="button"
                size="sm"
                data-testid={selectors.authoring.phaseSubmitButton}
                disabled={busy || !activePhase}
                onClick={handleSubmitPhaseField}
                className="w-fit"
              >
                {t(strings.pages.authoring.savePhaseField)}
              </Button>
            </div>
          </div>
        </SectionPanel>

        <SectionPanel
          title={t(strings.pages.authoring.violationsHeading)}
          headingId="authoring-violations-heading"
        >
          {state.violations.length === 0 ? (
            <p className="flex items-center gap-2 text-sm text-app-success">
              <CheckCircle2 aria-hidden="true" className="h-4 w-4" />
              {t(strings.pages.authoring.noViolations)}
            </p>
          ) : (
            <ul data-testid={selectors.authoring.violations} className="flex flex-col gap-2">
              {state.violations.map((v, i) => (
                <li
                  key={`${v.sectionKey}-${i}`}
                  className="rounded-control border border-app-danger/40 bg-app-danger/10 px-3 py-2 text-sm text-app-danger"
                >
                  <span className="font-mono text-xs">{v.sectionKey}</span>
                  <span className="ms-2">{v.message}</span>
                </li>
              ))}
            </ul>
          )}
        </SectionPanel>

        <SectionPanel
          title={t(strings.pages.authoring.autofillHeading)}
          headingId="authoring-autofill-heading"
          actions={
            <Button
              type="button"
              size="sm"
              variant="outline"
              data-testid={selectors.authoring.autofillButton}
              disabled={busy}
              onClick={handleAutofill}
            >
              <Wand2 aria-hidden="true" className="me-2 h-4 w-4" />
              {t(strings.pages.authoring.autofill)}
            </Button>
          }
        >
          {state.autofillResults.length === 0 ? (
            <p className="text-sm text-app-muted-foreground">{t(strings.pages.authoring.autofillEmpty)}</p>
          ) : (
            <ul data-testid={selectors.authoring.autofillResults} className="flex flex-col gap-1.5">
              {state.autofillResults.map((res, i) => (
                <li
                  key={`${res.source}-${i}`}
                  className="flex flex-wrap items-center justify-between gap-2 rounded-control border border-app-border bg-app-surface-muted px-3 py-2 text-sm"
                >
                  <span className="font-mono text-xs text-app-foreground">{res.source}</span>
                  <span className="flex items-center gap-2 text-xs">
                    {res.filled ? (
                      <span className="rounded-pill bg-app-success/15 px-2 py-0.5 text-app-success">
                        {t(strings.pages.authoring.filled)}
                      </span>
                    ) : null}
                    {res.degraded ? (
                      <span className="rounded-pill bg-app-warning/15 px-2 py-0.5 text-app-warning">
                        {t(strings.common.degradedBadge)}
                      </span>
                    ) : null}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </SectionPanel>

        <SectionPanel
          title={t(strings.pages.authoring.referencesHeading)}
          headingId="authoring-references-heading"
          actions={
            <Button
              type="button"
              size="sm"
              variant="outline"
              data-testid={selectors.authoring.referenceSuggestButton}
              disabled={busy}
              onClick={handleSuggestReferences}
            >
              <Sparkles aria-hidden="true" className="me-2 h-4 w-4" />
              {t(strings.pages.authoring.referenceSuggest)}
            </Button>
          }
        >
          <div className="flex flex-col gap-2">
            <Input
              data-testid={selectors.authoring.referenceRejectReasonInput}
              value={referenceRejectReason}
              onChange={(e) => setReferenceRejectReason(e.target.value)}
              placeholder={t(strings.pages.authoring.referenceRejectReasonPlaceholder)}
              aria-label={t(strings.pages.authoring.referenceRejectReasonLabel)}
            />
            <p className="text-xs font-medium uppercase tracking-wide text-app-muted-foreground">
              {t(strings.pages.authoring.referenceCandidates)}
            </p>
            <ReferenceCandidateList
              candidates={session.referenceCandidates}
              onAccept={handleAcceptReference}
              onReject={handleRejectReference}
              busy={busy}
            />
          </div>
        </SectionPanel>

        <SectionPanel title={t(strings.pages.authoring.contextHeading)} headingId="authoring-context-heading">
          <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_minmax(16rem,1fr)]">
            <div className="flex flex-col gap-2">
              <div className="grid gap-2 sm:grid-cols-[minmax(0,1fr)_12rem]">
                <Input
                  data-testid={selectors.authoring.contextConceptsInput}
                  value={contextConcepts}
                  onChange={(e) => setContextConcepts(e.target.value)}
                  placeholder={t(strings.pages.authoring.contextConceptsPlaceholder)}
                  aria-label={t(strings.pages.authoring.contextConceptsLabel)}
                />
                <Input
                  data-testid={selectors.authoring.contextComplexityInput}
                  value={contextComplexity}
                  onChange={(e) => setContextComplexity(e.target.value)}
                  placeholder={t(strings.pages.authoring.contextComplexityPlaceholder)}
                  aria-label={t(strings.pages.authoring.contextComplexityLabel)}
                />
              </div>
              <Button
                type="button"
                size="sm"
                variant="outline"
                data-testid={selectors.authoring.contextDiscoverButton}
                disabled={busy}
                onClick={handleDiscoverContext}
                className="w-fit"
              >
                {t(strings.pages.authoring.contextDiscover)}
              </Button>
              <div className="grid gap-2 sm:grid-cols-2">
                <select
                  data-testid={selectors.authoring.contextKindSelect}
                  value={String(contextKind)}
                  onChange={(e) => setContextKind(Number(e.target.value))}
                  className="h-10 rounded-control border border-app-border bg-app-surface px-3 text-sm"
                  aria-label={t(strings.pages.authoring.contextKindLabel)}
                >
                  {contextKinds.map((kind) => (
                    <option key={kind} value={String(kind)}>
                      {contextKindLabels[kind]}
                    </option>
                  ))}
                </select>
                <label className="flex h-10 items-center gap-2 rounded-control border border-app-border bg-app-surface px-3 text-sm">
                  <input
                    data-testid={selectors.authoring.contextPhaseToggle}
                    type="checkbox"
                    checked={contextPhaseScoped}
                    onChange={(e) => setContextPhaseScoped(e.target.checked)}
                  />
                  {t(strings.pages.authoring.contextPhaseScoped)}
                </label>
              </div>
              <Input
                data-testid={selectors.authoring.contextLabelInput}
                value={contextLabel}
                onChange={(e) => setContextLabel(e.target.value)}
                placeholder={t(strings.pages.authoring.contextLabelPlaceholder)}
                aria-label={t(strings.pages.authoring.contextLabelLabel)}
              />
              <Textarea
                data-testid={selectors.authoring.contextReasonInput}
                value={contextReason}
                onChange={(e) => setContextReason(e.target.value)}
                placeholder={t(strings.pages.authoring.contextReasonPlaceholder)}
                rows={2}
                aria-label={t(strings.pages.authoring.contextReasonLabel)}
              />
              <Textarea
                data-testid={selectors.authoring.contextInstructionInput}
                value={contextInstruction}
                onChange={(e) => setContextInstruction(e.target.value)}
                placeholder={t(strings.pages.authoring.contextInstructionPlaceholder)}
                rows={2}
                aria-label={t(strings.pages.authoring.contextInstructionLabel)}
              />
              <Input
                data-testid={selectors.authoring.contextCommandInput}
                value={contextCommand}
                onChange={(e) => setContextCommand(e.target.value)}
                placeholder={t(strings.pages.authoring.contextCommandPlaceholder)}
                aria-label={t(strings.pages.authoring.contextCommandLabel)}
              />
              <Input
                data-testid={selectors.authoring.contextTargetInput}
                value={contextTarget}
                onChange={(e) => setContextTarget(e.target.value)}
                placeholder={t(strings.pages.authoring.contextTargetPlaceholder)}
                aria-label={t(strings.pages.authoring.contextTargetLabel)}
              />
              <Input
                data-testid={selectors.authoring.contextRejectReasonInput}
                value={contextRejectReason}
                onChange={(e) => setContextRejectReason(e.target.value)}
                placeholder={t(strings.pages.authoring.contextRejectReasonPlaceholder)}
                aria-label={t(strings.pages.authoring.contextRejectReasonLabel)}
              />
              <Button
                type="button"
                size="sm"
                data-testid={selectors.authoring.contextSubmitButton}
                disabled={busy || contextLabel.trim().length === 0}
                onClick={handleSubmitContext}
                className="w-fit"
              >
                {t(strings.pages.authoring.contextSubmit)}
              </Button>
            </div>
            <div className="flex flex-col gap-3">
              <div>
                <p className="mb-1 text-xs font-medium uppercase tracking-wide text-app-muted-foreground">
                  {t(strings.pages.authoring.contextCandidates)}
                </p>
                <ContextCandidateList
                  candidates={session.contextCandidates}
                  onAccept={handleAcceptCandidate}
                  onReject={handleRejectCandidate}
                  busy={busy}
                />
              </div>
              <div>
                <p className="mb-1 text-xs font-medium uppercase tracking-wide text-app-muted-foreground">
                  {t(strings.pages.authoring.globalContext)}
                </p>
                <ContextList items={session.relevantContext} onRemove={handleRemoveContext} busy={busy} />
              </div>
              <div>
                <p className="mb-1 text-xs font-medium uppercase tracking-wide text-app-muted-foreground">
                  {t(strings.pages.authoring.phaseContext)}
                </p>
                <ContextList items={activePhase?.relevantContext ?? []} onRemove={handleRemoveContext} busy={busy} />
              </div>
            </div>
          </div>
        </SectionPanel>

        <div className="flex justify-end">
          <Button
            type="button"
            data-testid={selectors.authoring.finalizeButton}
            disabled={busy}
            onClick={handleFinalize}
          >
            <Sparkles aria-hidden="true" className="me-2 h-4 w-4" />
            {t(strings.pages.authoring.finalize)}
          </Button>
        </div>
      </div>
    </div>
  );
}
