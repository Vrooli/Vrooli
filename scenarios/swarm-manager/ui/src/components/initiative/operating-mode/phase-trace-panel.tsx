import type { ReactNode } from "react";
import type {
  OperatingModeArtifactSnapshot,
  OperatingModePhaseTransition,
  OperatingModeRound,
  OperatingModeRoundItem,
} from "../../../types/operating-mode";

export interface PhaseTraceReads {
  items: OperatingModeRoundItem[];
  artifacts: OperatingModeArtifactSnapshot[];
  priorRounds: OperatingModeRound[];
  acceptanceCriteria: string[];
}

export interface PhaseTraceData {
  phase: string;
  status?: string;
  reads: PhaseTraceReads;
  output?: unknown;
  transition?: Omit<OperatingModePhaseTransition, "to"> & { to?: string };
  terminal?: boolean;
}

export function PhaseTracePanel({
  title,
  subtitle,
  trace,
  controls,
  children,
  testId,
}: {
  title: string;
  subtitle: string;
  trace: PhaseTraceData | null;
  controls?: ReactNode;
  children?: ReactNode;
  testId: string;
}) {
  return (
    <section
      className="rounded-lg border border-slate-800 bg-slate-950/50 p-3"
      data-testid={testId}
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h3 className="text-sm font-semibold text-slate-100">{title}</h3>
          <p className="mt-0.5 text-xs text-slate-500">{subtitle}</p>
        </div>
        {controls}
      </div>

      {children}
      {trace ? <PhaseTraceDetails trace={trace} /> : null}
    </section>
  );
}

export function PhaseTraceDetails({ trace }: { trace: PhaseTraceData }) {
  return (
    <div className="mt-3 grid gap-3 lg:grid-cols-[0.8fr_1.2fr]">
      <div className="rounded-md border border-slate-800 bg-slate-900/40 p-3">
        <h4 className="text-[11px] font-semibold uppercase tracking-wide text-slate-400">Reads</h4>
        <dl className="mt-2 grid grid-cols-2 gap-2 text-xs">
          <TraceMetric label="Items" value={trace.reads.items.length} />
          <TraceMetric label="Artifacts" value={trace.reads.artifacts.length} />
          <TraceMetric label="Prior rounds" value={trace.reads.priorRounds.length} />
          <TraceMetric label="Criteria" value={trace.reads.acceptanceCriteria.length} />
        </dl>
        <div className="mt-3">
          <p className="text-[11px] uppercase tracking-wide text-slate-500">Transition</p>
          <p className="mt-1 text-xs text-slate-300">
            {trace.transition
              ? `${trace.transition.from} -> ${trace.transition.to || "stop"} (${trace.transition.label})`
              : trace.terminal
                ? "terminal"
                : "pending"}
          </p>
        </div>
      </div>
      <div className="rounded-md border border-slate-800 bg-slate-900/40 p-3">
        <h4 className="text-[11px] font-semibold uppercase tracking-wide text-slate-400">Emits</h4>
        {trace.output && Object.keys(objectOutput(trace.output)).length > 0 ? (
          <pre className="mt-2 max-h-64 overflow-auto rounded bg-slate-950 p-2 text-[11px] leading-relaxed text-slate-300">
            {JSON.stringify(trace.output, null, 2)}
          </pre>
        ) : (
          <p className="mt-2 text-sm italic text-slate-500">No structured output yet.</p>
        )}
      </div>
    </div>
  );
}

function TraceMetric({ label, value }: { label: string; value: number }) {
  return (
    <div>
      <dt className="text-slate-500">{label}</dt>
      <dd className="text-slate-200">{value}</dd>
    </div>
  );
}

function objectOutput(value: unknown): Record<string, unknown> {
  return value && typeof value === "object" && !Array.isArray(value)
    ? value as Record<string, unknown>
    : {};
}
