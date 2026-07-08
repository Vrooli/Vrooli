/**
 * PhaseViewer — one shared, source-agnostic viewer for a single operating-mode
 * phase. It renders four concern tabs (Instructions / Reads / Emits /
 * Transition) from a PhaseView, so the Flow tab (Simulation/Live) and the
 * Phases tab (Contract) render the exact same surface — only the data fill
 * differs. The Instructions tab lazily renders the literal agent prompt (the
 * most explanatory artifact) for the selected source, degrading to the resolved
 * prompt variables when the render seam is unavailable.
 */

import { type ReactNode, useState } from "react";
import {
  AlertCircle,
  ArrowRight,
  Ban,
  BookOpen,
  CircleDot,
  Flag,
  Info,
  Loader2,
  RotateCcw,
} from "lucide-react";
import { cn } from "../../../lib/utils";
import { selectors } from "../../../consts/selectors";
import { StatusChip } from "../../ui/status-chip";
import type {
  OperatingModeArtifactSnapshot,
  OperatingModeRound,
  OperatingModeRoundItem,
} from "../../../types/operating-mode";
import {
  describeTransition,
  formatTransition,
  PHASE_READ_CATEGORIES,
  phaseTraceEmits,
  type PhaseEmitSpec,
  type PhaseReadCategory,
  type TransitionExplanation,
} from "./phase-interpretability";
import type { PhaseView, PhaseViewReads } from "./phase-view";
import { usePhasePrompt } from "./use-phase-prompt";
import { PhaseProfilePopover } from "./phase-profile-popover";

type PhaseTab = "instructions" | "reads" | "emits" | "transition";

const TABS: Array<{ value: PhaseTab; label: string; testId: string }> = [
  { value: "instructions", label: "Instructions", testId: selectors.initiativeDetails.phaseViewerTabInstructions },
  { value: "reads", label: "Reads", testId: selectors.initiativeDetails.phaseViewerTabReads },
  { value: "emits", label: "Emits", testId: selectors.initiativeDetails.phaseViewerTabEmits },
  { value: "transition", label: "Transition", testId: selectors.initiativeDetails.phaseViewerTabTransition },
];

const TERMINAL_CHIP = {
  background: "bg-violet-500/10",
  border: "border-violet-500/30",
  text: "text-violet-300",
};

export function PhaseViewer({
  view,
  title,
  subtitle,
  controls,
  children,
  hideHeader,
}: {
  view: PhaseView;
  title?: string;
  subtitle?: string;
  controls?: ReactNode;
  children?: ReactNode;
  /** Suppress the label/id header row when the embedder already renders one. */
  hideHeader?: boolean;
}) {
  const [activeTab, setActiveTab] = useState<PhaseTab>("instructions");

  return (
    <section
      className="rounded-lg border border-slate-800 bg-slate-950/50 p-3"
      data-testid={selectors.initiativeDetails.phaseViewer}
      data-phase={view.phase}
      data-source={view.source}
    >
      {!hideHeader && (
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div className="min-w-0">
            <h3 className="flex flex-wrap items-center gap-2 text-sm font-semibold text-slate-100">
              {title ?? view.label}
              <code className="rounded bg-slate-800/80 px-1.5 py-0.5 text-[11px] font-mono text-slate-300">
                {view.phase}
              </code>
              {view.terminal && <StatusChip label="terminal" colors={TERMINAL_CHIP} />}
            </h3>
            {subtitle ? <p className="mt-0.5 truncate text-xs text-slate-500">{subtitle}</p> : null}
          </div>
          {controls}
        </div>
      )}

      {children}

      <div className="mt-3 flex gap-1 border-b border-slate-800" role="tablist" aria-label="Phase detail">
        {TABS.map((tab) => (
          <button
            key={tab.value}
            type="button"
            role="tab"
            aria-selected={activeTab === tab.value}
            data-testid={tab.testId}
            onClick={() => setActiveTab(tab.value)}
            className={cn(
              "-mb-px border-b-2 px-2.5 py-1.5 text-xs font-medium transition-colors",
              activeTab === tab.value
                ? "border-cyan-400 text-cyan-200"
                : "border-transparent text-slate-400 hover:text-slate-200",
            )}
          >
            {tab.label}
          </button>
        ))}
      </div>

      <div className="mt-3">
        {activeTab === "instructions" && <InstructionsTab view={view} />}
        {activeTab === "reads" && <ReadsTab reads={view.reads} />}
        {activeTab === "emits" && <EmitsTab view={view} />}
        {activeTab === "transition" && <TransitionTab view={view} />}
      </div>
    </section>
  );
}

// ── Instructions ───────────────────────────────────────────────────────────

function InstructionsTab({ view }: { view: PhaseView }) {
  const [profileOpen, setProfileOpen] = useState(false);
  const prompt = usePhasePrompt(view.prompt);
  const showVariables = (prompt.degraded || Boolean(prompt.error)) && !prompt.isSlots;

  return (
    <div className="space-y-3" data-testid={selectors.initiativeDetails.phaseViewerInstructions}>
      <div className="flex flex-wrap items-center gap-2">
        <span className="inline-flex items-center gap-1.5 rounded-md border border-cyan-500/20 bg-cyan-500/10 px-2 py-1 text-[11px] text-cyan-100">
          <BookOpen className="h-3.5 w-3.5 text-cyan-300" aria-hidden />
          <code className="font-mono text-cyan-200/90">{prompt.skillId || view.skillId || "no skill"}</code>
        </span>
        {(prompt.profileKey || view.profileKey) && (
          <>
            <button
              type="button"
              onClick={() => setProfileOpen(true)}
              aria-label={`Explain agent profile ${prompt.profileKey || view.profileKey}`}
              data-testid={selectors.initiativeDetails.phaseViewerProfileChip}
              className="group inline-flex items-center gap-1 rounded-full border border-slate-700 bg-slate-800/80 px-2 py-0.5 text-[11px] text-slate-200 transition-colors hover:border-cyan-500/60 hover:bg-slate-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500/50"
            >
              <span className="font-medium">{prompt.profileKey || view.profileKey}</span>
              <Info className="h-3 w-3 text-slate-500 transition-colors group-hover:text-cyan-300" aria-hidden />
            </button>
            {profileOpen && (
              <PhaseProfilePopover
                profileKey={prompt.profileKey || view.profileKey}
                isOpen
                onClose={() => setProfileOpen(false)}
              />
            )}
          </>
        )}
        <SourceNote source={view.source} isSlots={prompt.isSlots} />
      </div>

      {prompt.isLoading && (
        <div className="flex items-center gap-2 rounded-md border border-slate-800 bg-slate-900/40 p-3 text-xs text-slate-400">
          <Loader2 className="h-4 w-4 animate-spin text-cyan-400" />
          Rendering agent prompt…
        </div>
      )}

      {!prompt.isLoading && prompt.error && !showVariables && (
        <div className="flex flex-wrap items-start justify-between gap-2 rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-300">
          <p className="flex min-w-0 flex-1 items-center gap-2">
            <AlertCircle className="h-4 w-4 shrink-0" />
            {prompt.error.message || "Failed to render the agent prompt."}
          </p>
          <button
            type="button"
            onClick={prompt.refetch}
            className="inline-flex items-center gap-1 rounded border border-slate-700 bg-slate-800/80 px-2 py-1 font-medium text-slate-200 hover:border-cyan-500/60"
          >
            <RotateCcw className="h-3 w-3" /> Retry
          </button>
        </div>
      )}

      {!prompt.isLoading && showVariables && (
        <div className="rounded-md border border-amber-500/30 bg-amber-500/10 p-3 text-xs text-amber-200">
          <p className="mb-2">
            The prompt renderer is unavailable, so the literal prompt can't be shown. These are the values that
            would be substituted:
          </p>
          <VariableTable variables={prompt.variables} />
        </div>
      )}

      {!prompt.isLoading && !prompt.error && !showVariables && prompt.prompt && (
        <pre
          className="max-h-[28rem] overflow-auto rounded-md border border-slate-800 bg-slate-950 p-3 text-[11px] leading-relaxed text-slate-200 whitespace-pre-wrap break-words"
          data-testid={selectors.initiativeDetails.phaseViewerPrompt}
        >
          {prompt.prompt}
        </pre>
      )}

      {!prompt.isLoading && !prompt.error && !showVariables && !prompt.prompt && (
        <p className="rounded-md border border-slate-800 bg-slate-900/40 p-3 text-xs italic text-slate-500">
          No prompt content for this phase.
        </p>
      )}
    </div>
  );
}

function SourceNote({ source, isSlots }: { source: PhaseView["source"]; isSlots: boolean }) {
  const note = isSlots
    ? "Template with unfilled {{VARIABLE}} slots"
    : source === "simulation"
      ? "Filled with the selected preset's data"
      : source === "live"
        ? "Filled with this initiative's real data"
        : "Rendered agent prompt";
  return <span className="text-[11px] text-slate-500">· {note}</span>;
}

function VariableTable({ variables }: { variables: Record<string, string> }) {
  const entries = Object.entries(variables).filter(([, value]) => value.trim() !== "");
  if (entries.length === 0) {
    return <p className="italic text-amber-300/70">No variables were resolved.</p>;
  }
  return (
    <ul className="space-y-1" data-testid={selectors.initiativeDetails.phaseViewerVariables}>
      {entries.map(([key, value]) => (
        <li key={key} className="break-words">
          <code className="rounded bg-slate-800/80 px-1.5 py-0.5 font-mono text-[11px] text-slate-100">{key}</code>
          <span className="ml-2 text-slate-300">{truncate(value, 160)}</span>
        </li>
      ))}
    </ul>
  );
}

// ── Reads ────────────────────────────────────────────────────────────────

function ReadsTab({ reads }: { reads?: PhaseViewReads }) {
  return (
    <section data-testid={selectors.initiativeDetails.flowTraceReads}>
      <p className="text-[11px] text-slate-500">
        Each card is one prompt variable this phase reads from context.
      </p>
      <div className="mt-2 grid gap-2 sm:grid-cols-2">
        {PHASE_READ_CATEGORIES.map((category) => (
          <ReadCard
            key={category.key}
            label={category.label}
            meaning={category.meaning}
            variable={category.variable}
            category={category.key}
            reads={reads}
          />
        ))}
      </div>
    </section>
  );
}

function ReadCard({
  label,
  meaning,
  variable,
  category,
  reads,
}: {
  label: string;
  meaning: string;
  variable: string;
  category: PhaseReadCategory;
  reads?: PhaseViewReads;
}) {
  const summary = reads ? readSummary(reads, category) : null;
  return (
    <div className="rounded border border-slate-800/80 bg-slate-950/50 p-2.5" title={meaning}>
      <div className="flex items-baseline justify-between gap-2">
        <span className="text-xs font-medium text-slate-200">{label}</span>
        {summary ? (
          <span className="text-xs tabular-nums text-slate-400">{summary.count}</span>
        ) : null}
      </div>
      <code className="mt-0.5 block truncate font-mono text-[10px] text-cyan-200/80">{`{{${variable}}}`}</code>
      {summary && summary.details.length > 0 ? (
        <ul className="mt-1.5 space-y-0.5">
          {summary.details.map((detail, idx) => (
            <li key={idx} className="truncate text-[11px] text-slate-400" title={detail}>
              {detail}
            </li>
          ))}
        </ul>
      ) : summary ? (
        <p className="mt-1.5 text-[11px] italic text-slate-600">none</p>
      ) : (
        <p className="mt-1.5 text-[11px] leading-relaxed text-slate-500">{meaning}</p>
      )}
    </div>
  );
}

function readSummary(
  reads: PhaseViewReads,
  category: PhaseReadCategory,
): { count: number; details: string[] } {
  switch (category) {
    case "items": {
      const items: OperatingModeRoundItem[] = reads.items;
      return {
        count: items.length,
        details: withOverflow(
          items.map((item) => (item.title ? `${item.ref} — ${item.title}` : item.ref)),
          3,
        ),
      };
    }
    case "artifacts": {
      const artifacts: OperatingModeArtifactSnapshot[] = reads.artifacts;
      return { count: artifacts.length, details: withOverflow(artifacts.map((a) => a.path), 3) };
    }
    case "priorRounds": {
      const rounds: OperatingModeRound[] = reads.priorRounds;
      const last = rounds[rounds.length - 1];
      return {
        count: rounds.length,
        details: last ? [`latest: round ${last.round} · ${last.phase}`] : [],
      };
    }
    case "acceptanceCriteria": {
      const criteria = reads.acceptanceCriteria;
      return { count: criteria.length, details: withOverflow(criteria, 2) };
    }
  }
}

// ── Emits ────────────────────────────────────────────────────────────────

function EmitsTab({ view }: { view: PhaseView }) {
  if (view.emitSchema) {
    return <EmitSchemaList schema={view.emitSchema} />;
  }
  return <EmitActual output={view.output} />;
}

function EmitSchemaList({ schema }: { schema: PhaseEmitSpec[] }) {
  return (
    <section data-testid={selectors.initiativeDetails.flowTraceEmits}>
      <p className="text-[11px] text-slate-500">The structured result this phase is contracted to produce.</p>
      <ul className="mt-2 space-y-2">
        {schema.map((emit) => (
          <li key={emit.field} className="text-xs text-slate-300">
            <div className="flex flex-wrap items-center gap-2">
              <code className="rounded bg-slate-800/80 px-1.5 py-0.5 font-mono text-[11px] text-slate-100">
                {emit.label}
              </code>
              {emit.required && (
                <StatusChip
                  label="required"
                  colors={{ background: "bg-amber-500/10", border: "border-amber-500/30", text: "text-amber-300" }}
                />
              )}
            </div>
            <p className="mt-1 leading-relaxed text-slate-500">{emit.meaning}</p>
          </li>
        ))}
      </ul>
    </section>
  );
}

function EmitActual({ output }: { output: unknown }) {
  const emits = phaseTraceEmits(output);
  return (
    <section data-testid={selectors.initiativeDetails.flowTraceEmits}>
      <p className="text-[11px] text-slate-500">What this phase actually produced.</p>
      {emits.hasContent ? (
        <ul className="mt-2 space-y-2">
          {emits.handoffs.map((handoff, idx) => (
            <EmitRow key={`handoff-${idx}`} label="handoff">
              <span className="text-slate-300">{handoff.summary || "Execution handoff recorded."}</span>
              {handoff.nextStep && <span className="text-slate-500"> · next: {handoff.nextStep}</span>}
            </EmitRow>
          ))}
          {emits.progress && (
            <EmitRow label="progress">
              <span className="font-medium text-slate-200">{emits.progress.decision}</span>
              {emits.progress.rationale && <span className="text-slate-500"> — {emits.progress.rationale}</span>}
            </EmitRow>
          )}
          {emits.verdict && (
            <EmitRow label="verdict">
              <span className="font-medium text-slate-200">{emits.verdict}</span>
            </EmitRow>
          )}
          {emits.replanNeeded && (
            <EmitRow label="replan_needed">
              <span className="inline-flex items-center gap-1 text-amber-300">
                <RotateCcw className="h-3 w-3" aria-hidden="true" /> replan requested
              </span>
            </EmitRow>
          )}
          {emits.artifacts.length > 0 && (
            <EmitRow label="artifacts">
              <ul className="space-y-0.5">
                {withOverflow(emits.artifacts.map((a) => a.path), 3).map((path, idx) => (
                  <li key={idx} className="break-all font-mono text-[11px] text-slate-400">
                    {path}
                  </li>
                ))}
              </ul>
            </EmitRow>
          )}
          {emits.backlogSync && (
            <EmitRow label="backlog_sync">
              <BacklogSyncSummary sync={emits.backlogSync} />
            </EmitRow>
          )}
          {emits.readiness && (
            <EmitRow label="readiness">
              <span className="text-slate-400">Scored readiness report</span>
            </EmitRow>
          )}
        </ul>
      ) : (
        <p className="mt-2 text-xs italic text-slate-500">No structured output for this phase.</p>
      )}
      {output !== undefined && output !== null && Object.keys(objectOutput(output)).length > 0 && (
        <details className="mt-2 text-xs" data-testid={selectors.initiativeDetails.flowTraceRawToggle}>
          <summary className="cursor-pointer select-none text-slate-500 hover:text-slate-300">
            View raw payload
          </summary>
          <pre className="mt-2 max-h-64 overflow-auto rounded bg-slate-950 p-2 text-[11px] leading-relaxed text-slate-300">
            {JSON.stringify(output, null, 2)}
          </pre>
        </details>
      )}
    </section>
  );
}

function BacklogSyncSummary({
  sync,
}: {
  sync: NonNullable<ReturnType<typeof phaseTraceEmits>["backlogSync"]>;
}) {
  const parts: string[] = [];
  if (sync.completedItems.length) parts.push(`${sync.completedItems.length} completed`);
  if (sync.createdItems.length) parts.push(`${sync.createdItems.length} created`);
  if (sync.updatedItems.length) parts.push(`${sync.updatedItems.length} updated`);
  const refs = withOverflow([...sync.completedItems, ...sync.createdItems, ...sync.updatedItems], 3);
  return (
    <div>
      <span className="text-slate-300">{parts.length ? parts.join(" · ") : "Proposed backlog alignment"}</span>
      {sync.rationale && <p className="text-slate-500">{sync.rationale}</p>}
      {refs.length > 0 && (
        <ul className="mt-0.5 space-y-0.5">
          {refs.map((ref, idx) => (
            <li key={idx} className="break-all font-mono text-[11px] text-slate-400">
              {ref}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function EmitRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <li className="text-xs text-slate-300">
      <code className="rounded bg-slate-800/80 px-1.5 py-0.5 font-mono text-[11px] text-slate-100">{label}</code>
      <span className="ml-2 align-middle text-[11px] leading-relaxed">{children}</span>
    </li>
  );
}

// ── Transition ─────────────────────────────────────────────────────────────

const TRANSITION_TONES: Record<
  TransitionExplanation["tone"],
  { icon: typeof ArrowRight; color: string }
> = {
  route: { icon: ArrowRight, color: "text-cyan-300" },
  terminal: { icon: Flag, color: "text-violet-300" },
  blocked: { icon: Ban, color: "text-amber-300" },
  pending: { icon: CircleDot, color: "text-slate-400" },
};

function TransitionTab({ view }: { view: PhaseView }) {
  if (view.declaredTransitions) {
    return (
      <section data-testid={selectors.initiativeDetails.flowTraceTransition}>
        <p className="text-[11px] text-slate-500">Every outgoing route this phase can take.</p>
        {view.declaredTransitions.length > 0 ? (
          <ul className="mt-2 space-y-1">
            {view.declaredTransitions.map((transition) => (
              <li
                key={`${transition.from}-${transition.to}-${transition.label}`}
                className="text-xs text-slate-300"
              >
                {formatTransition(transition)}
              </li>
            ))}
          </ul>
        ) : (
          <p className="mt-2 text-xs text-slate-500">No outgoing transition; this phase is terminal.</p>
        )}
      </section>
    );
  }
  const explanation = describeTransition(view.firedTransition, view.terminal);
  const tone = TRANSITION_TONES[explanation.tone];
  const Icon = tone.icon;
  return (
    <section data-testid={selectors.initiativeDetails.flowTraceTransition}>
      <p className="text-[11px] text-slate-500">Why the next phase was chosen.</p>
      <div className="mt-2 flex items-start gap-2">
        <Icon className={`mt-0.5 h-4 w-4 shrink-0 ${tone.color}`} aria-hidden="true" />
        <div className="min-w-0">
          <p className={`text-xs font-medium ${tone.color}`}>{explanation.headline}</p>
          <p className="mt-0.5 text-[11px] leading-relaxed text-slate-400">{explanation.reason}</p>
        </div>
      </div>
    </section>
  );
}

// ── Shared ───────────────────────────────────────────────────────────────

function withOverflow(values: string[], limit: number): string[] {
  if (values.length <= limit) return values;
  const shown = values.slice(0, limit);
  shown.push(`+${values.length - limit} more`);
  return shown;
}

function truncate(value: string, limit: number): string {
  return value.length <= limit ? value : `${value.slice(0, limit)}…`;
}

function objectOutput(value: unknown): Record<string, unknown> {
  return value && typeof value === "object" && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : {};
}
