import { useMemo } from "react";

import type { FlowTransition } from "../../api/inventory";
import { useTranslation } from "../../i18n";

/** Shape of a counterexample step extracted from a failing run's
 *  counterexample blob. Quint emits ITF traces whose `states` array
 *  holds objects with at minimum a `__state` field; we accept a few
 *  shapes here defensively because counterexample writers across
 *  scenarios are not yet uniform. */
export interface CounterexampleStep {
  state: string;
  event?: string;
}

export interface CounterexampleDiffProps {
  /** Raw counterexample JSON from `runs.counterexample`. */
  counterexampleJson: string;
  expectedTransitions: FlowTransition[];
}

interface ParsedCounterexample {
  steps: CounterexampleStep[];
  parseError?: string;
}

function parseCounterexample(raw: string): ParsedCounterexample {
  if (!raw) return { steps: [] };
  try {
    const parsed = JSON.parse(raw) as unknown;
    const steps = coerceSteps(parsed);
    return { steps };
  } catch (err) {
    return { steps: [], parseError: (err as Error).message };
  }
}

function coerceSteps(value: unknown): CounterexampleStep[] {
  if (!value || typeof value !== "object") return [];
  const record = value as Record<string, unknown>;
  // ITF traces have `states: [...]`; some writers wrap as `trace`.
  const arr = record.states ?? record.trace ?? record.steps;
  if (!Array.isArray(arr)) return [];
  return arr.map((entry) => {
    if (!entry || typeof entry !== "object") return { state: String(entry) };
    const e = entry as Record<string, unknown>;
    const state =
      (e.state as string | undefined) ??
      (e.__state as string | undefined) ??
      (e.status as string | undefined) ??
      JSON.stringify(entry);
    const event =
      (e.event as string | undefined) ?? (e.action as string | undefined);
    return { state, event };
  });
}

/**
 * CounterexampleDiff renders a side-by-side table of what the model
 * expected vs what the failing run actually walked through. Expected
 * transitions are looked up in the flow's transition matrix; actual
 * transitions come from the run's counterexample blob.
 */
export function CounterexampleDiff({
  counterexampleJson,
  expectedTransitions,
}: CounterexampleDiffProps) {
  const { t } = useTranslation();
  const parsed = useMemo(
    () => parseCounterexample(counterexampleJson),
    [counterexampleJson],
  );

  if (parsed.parseError) {
    return (
      <p data-testid="ce-diff-parse-error" className="text-app-danger">
        {t("counterexample.parseError", {
          defaultValue: "Failed to parse counterexample: ",
        })}
        {parsed.parseError}
      </p>
    );
  }

  if (parsed.steps.length === 0) {
    return (
      <p data-testid="ce-diff-empty" className="text-app-foreground">
        {t("counterexample.empty", {
          defaultValue: "No counterexample on this run.",
        })}
      </p>
    );
  }

  const rows: Array<{
    idx: number;
    from: string;
    event: string;
    actual: string;
    expected: string;
    mismatch: boolean;
  }> = [];

  for (let i = 1; i < parsed.steps.length; i++) {
    const prev = parsed.steps[i - 1];
    const curr = parsed.steps[i];
    if (!prev || !curr) continue;
    const event = curr.event ?? "?";
    const expected =
      expectedTransitions.find(
        (tr) => tr.from === prev.state && tr.event === event,
      )?.to ?? "—";
    rows.push({
      idx: i,
      from: prev.state,
      event,
      actual: curr.state,
      expected,
      mismatch: expected !== "—" && expected !== curr.state,
    });
  }

  return (
    <div
      data-testid="ce-diff"
      className="overflow-x-auto rounded-panel border border-app-border bg-app-surface"
    >
      <table className="w-full text-left text-sm text-app-foreground">
        <thead className="text-xs uppercase text-app-muted-foreground">
          <tr>
            <th className="px-3 py-2">#</th>
            <th className="px-3 py-2">{t("counterexample.colFrom", { defaultValue: "From" })}</th>
            <th className="px-3 py-2">{t("counterexample.colEvent", { defaultValue: "Event" })}</th>
            <th className="px-3 py-2">{t("counterexample.colExpected", { defaultValue: "Expected →" })}</th>
            <th className="px-3 py-2">{t("counterexample.colActual", { defaultValue: "Actual →" })}</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr
              key={row.idx}
              data-testid={`ce-diff-row-${row.idx}`}
              className={`border-t border-app-border ${row.mismatch ? "bg-app-danger/10" : ""}`}
            >
              <td className="px-3 py-2 text-xs text-app-muted-foreground">{row.idx}</td>
              <td className="px-3 py-2 font-mono">{row.from}</td>
              <td className="px-3 py-2 font-mono">{row.event}</td>
              <td className="px-3 py-2 font-mono">{row.expected}</td>
              <td
                data-testid={`ce-diff-actual-${row.idx}`}
                className={`px-3 py-2 font-mono ${row.mismatch ? "text-app-danger" : ""}`}
              >
                {row.actual}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
