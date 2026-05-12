import { useEffect, useReducer } from "react";

import { Button } from "../../components/ui/button";
import type { FlowTrace } from "../../api/inventory";
import { useTranslation } from "../../i18n";

import { StateGraph } from "./StateGraph";
import type { StateGraphProps } from "./StateGraph";

interface PlayerState {
  traceIdx: number;
  step: number;
}

type Action =
  | { type: "selectTrace"; idx: number }
  | { type: "next" }
  | { type: "prev" }
  | { type: "reset" };

function reducer(state: PlayerState, action: Action, traces: FlowTrace[]): PlayerState {
  switch (action.type) {
    case "selectTrace":
      return { traceIdx: action.idx, step: 0 };
    case "next": {
      const trace = traces[state.traceIdx];
      if (!trace) return state;
      const max = trace.steps.length;
      return { ...state, step: Math.min(state.step + 1, max) };
    }
    case "prev":
      return { ...state, step: Math.max(state.step - 1, 0) };
    case "reset":
      return { ...state, step: 0 };
    default:
      return state;
  }
}

export interface TracePlayerProps {
  traces: FlowTrace[];
  graphProps: Omit<StateGraphProps, "activeState">;
}

/**
 * TracePlayer walks the user through a named ITF trace one step at a
 * time. The current state is fed into the shared StateGraph so the
 * highlighted node moves with the player.
 *
 * Pure useReducer (no Context) — scope discipline per react-coherence.
 */
export function TracePlayer({ traces, graphProps }: TracePlayerProps) {
  const { t } = useTranslation();
  const [state, dispatch] = useReducer(
    (s: PlayerState, a: Action) => reducer(s, a, traces),
    { traceIdx: 0, step: 0 },
  );

  // If the trace list changes (e.g., flow swap), reset selection.
  useEffect(() => {
    dispatch({ type: "selectTrace", idx: 0 });
  }, [traces]);

  if (traces.length === 0) {
    return (
      <div data-testid="trace-player-empty" className="text-app-foreground">
        {t("tracePlayer.empty", { defaultValue: "No named traces in this flow." })}
      </div>
    );
  }

  const trace = traces[Math.min(state.traceIdx, traces.length - 1)];
  if (!trace) {
    return (
      <div data-testid="trace-player-empty" className="text-app-foreground">
        {t("tracePlayer.empty", { defaultValue: "No named traces in this flow." })}
      </div>
    );
  }
  const stepIdx = Math.min(state.step, trace.steps.length);
  const previousStep = stepIdx === 0 ? undefined : trace.steps[stepIdx - 1];
  const activeState = previousStep ? previousStep.want : trace.initial;
  const lastEvent = previousStep?.event;

  return (
    <div data-testid="trace-player" className="space-y-3">
      <div className="flex flex-wrap items-end gap-3">
        <label className="flex flex-col gap-1 text-xs text-app-muted-foreground">
          <span>{t("tracePlayer.selectLabel", { defaultValue: "Trace" })}</span>
          <select
            data-testid="trace-player-select"
            value={state.traceIdx}
            onChange={(e) =>
              dispatch({ type: "selectTrace", idx: Number(e.target.value) })
            }
            className="rounded-control border border-app-border bg-app-surface-muted px-2 py-1 text-sm text-app-foreground"
          >
            {traces.map((tr, i) => (
              <option key={tr.name} value={i}>
                {tr.name}
              </option>
            ))}
          </select>
        </label>
        <Button
          data-testid="trace-player-prev"
          size="sm"
          variant="outline"
          onClick={() => dispatch({ type: "prev" })}
          disabled={stepIdx === 0}
        >
          {t("tracePlayer.prev", { defaultValue: "Prev" })}
        </Button>
        <Button
          data-testid="trace-player-next"
          size="sm"
          variant="outline"
          onClick={() => dispatch({ type: "next" })}
          disabled={stepIdx >= trace.steps.length}
        >
          {t("tracePlayer.next", { defaultValue: "Next" })}
        </Button>
        <Button
          data-testid="trace-player-reset"
          size="sm"
          variant="outline"
          onClick={() => dispatch({ type: "reset" })}
          disabled={stepIdx === 0}
        >
          {t("tracePlayer.reset", { defaultValue: "Reset" })}
        </Button>
        <p data-testid="trace-player-progress" className="text-xs text-app-muted-foreground">
          {t("tracePlayer.progress", {
            defaultValue: `Step ${stepIdx} / ${trace.steps.length}`,
          })}
        </p>
      </div>

      <p data-testid="trace-player-active" className="font-mono text-sm text-app-foreground">
        {t("tracePlayer.activeLabel", { defaultValue: "Current state" })}
        : <span data-testid="trace-player-active-state">{activeState}</span>
        {lastEvent && (
          <>
            {" "}
            <span className="text-app-muted-foreground">
              ({t("tracePlayer.viaEvent", { defaultValue: "via" })}{" "}
              <span data-testid="trace-player-last-event">{lastEvent}</span>)
            </span>
          </>
        )}
      </p>

      <StateGraph {...graphProps} activeState={activeState} />
    </div>
  );
}
