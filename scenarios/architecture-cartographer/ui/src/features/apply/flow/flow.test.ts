import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { describe, expect, it } from "vitest";

import { assertTransitionMatrix, type MatrixRow } from "@vrooli/flow-runtime";
import { replayTraces, type Trace } from "@vrooli/flow-runtime";

import {
  APPLY_EVENTS,
  APPLY_STATES,
  INITIAL_APPLY_STATE,
  TERMINAL_APPLY_STATES,
  legalEventsFor,
  transition,
  type ApplyEvent,
  type ApplyState,
} from "./transition";

const here = dirname(fileURLToPath(import.meta.url));
const flowJsonPath = resolve(here, "./flow.json");

interface FlowJsonState {
  readonly id: string;
  readonly initial?: boolean;
}
interface FlowJsonEvent {
  readonly id: string;
}
interface FlowJsonTransition {
  readonly from: string;
  readonly event: string;
  readonly to: string;
}
interface FlowJson {
  readonly states: readonly FlowJsonState[];
  readonly events: readonly FlowJsonEvent[];
  readonly transitions: readonly FlowJsonTransition[];
}

const flowJson = JSON.parse(readFileSync(flowJsonPath, "utf8")) as FlowJson;

describe("per-domain apply flow", () => {
  it("declares the same states transition.ts knows about", () => {
    expect(flowJson.states.map((s) => s.id).sort()).toEqual([...APPLY_STATES].sort());
  });

  it("declares the same events transition.ts knows about", () => {
    expect(flowJson.events.map((e) => e.id).sort()).toEqual([...APPLY_EVENTS].sort());
  });

  it("declares baseline_captured as the only initial state", () => {
    const initials = flowJson.states.filter((s) => s.initial === true).map((s) => s.id);
    expect(initials).toEqual([INITIAL_APPLY_STATE]);
  });

  it("transition.ts agrees with flow.json across the full matrix", () => {
    const rows: MatrixRow<ApplyState, ApplyEvent>[] = flowJson.transitions.map((row) => ({
      name: `${row.from}/${row.event}`,
      from: row.from as ApplyState,
      event: row.event as ApplyEvent,
      to: row.to as ApplyState,
    }));
    assertTransitionMatrix(APPLY_STATES, APPLY_EVENTS, rows, transition);
  });

  it("committed and force_committed are terminal", () => {
    for (const terminal of TERMINAL_APPLY_STATES) {
      for (const event of APPLY_EVENTS) {
        expect(transition(terminal, event)).toBe(terminal);
      }
    }
  });

  it("force_commit is only legal from refused_build_break", () => {
    for (const state of APPLY_STATES) {
      const result = transition(state, "force_commit");
      if (state === "refused_build_break") {
        expect(result).toBe("force_committed");
      } else {
        expect(result).toBe(state);
      }
    }
  });

  it("happy path: baseline → plan → dry_run → apply → commit", () => {
    const trace: Trace<ApplyState, ApplyEvent> = {
      name: "happy path",
      initial: "baseline_captured",
      steps: [
        { event: "plan", want: "plan_generated" },
        { event: "dry_run", want: "dry_run_ok" },
        { event: "apply", want: "applied" },
        { event: "commit", want: "committed" },
      ],
    };
    replayTraces([trace], transition);
  });

  it("refuse path: applied → refused → force_commit", () => {
    const trace: Trace<ApplyState, ApplyEvent> = {
      name: "force-commit path",
      initial: "baseline_captured",
      steps: [
        { event: "plan", want: "plan_generated" },
        { event: "apply", want: "applied" },
        { event: "refuse", want: "refused_build_break" },
        { event: "force_commit", want: "force_committed" },
      ],
    };
    replayTraces([trace], transition);
  });

  it("legalEventsFor returns only events that change state", () => {
    for (const state of APPLY_STATES) {
      for (const event of legalEventsFor(state)) {
        expect(transition(state, event)).not.toBe(state);
      }
      for (const event of APPLY_EVENTS) {
        if (legalEventsFor(state).includes(event)) continue;
        expect(transition(state, event)).toBe(state);
      }
    }
  });
});
