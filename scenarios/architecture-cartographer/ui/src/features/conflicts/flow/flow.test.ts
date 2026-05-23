import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { describe, expect, it } from "vitest";

import { assertTransitionMatrix, type MatrixRow } from "../../../test-utils/modeltest/matrix";
import { replayTraces, type Trace } from "../../../test-utils/modeltest/traces";

import {
  CONFLICT_EVENTS,
  CONFLICT_STATES,
  INITIAL_CONFLICT_STATE,
  TERMINAL_CONFLICT_STATES,
  legalEventsFor,
  transition,
  type ConflictEvent,
  type ConflictState,
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

describe("conflict-resolution flow", () => {
  it("declares the same states transition.ts knows about", () => {
    const declared = flowJson.states.map((state) => state.id).sort();
    const runtime = [...CONFLICT_STATES].sort();
    expect(declared).toEqual(runtime);
  });

  it("declares the same events transition.ts knows about", () => {
    const declared = flowJson.events.map((event) => event.id).sort();
    const runtime = [...CONFLICT_EVENTS].sort();
    expect(declared).toEqual(runtime);
  });

  it("declares 'detected' as the only initial state and matches INITIAL_CONFLICT_STATE", () => {
    const initials = flowJson.states.filter((state) => state.initial === true).map((state) => state.id);
    expect(initials).toEqual([INITIAL_CONFLICT_STATE]);
  });

  it("transition.ts agrees with flow.json across the full transition matrix", () => {
    const rows: MatrixRow<ConflictState, ConflictEvent>[] = flowJson.transitions.map((row) => ({
      name: `${row.from}/${row.event}`,
      from: row.from as ConflictState,
      event: row.event as ConflictEvent,
      to: row.to as ConflictState,
    }));
    assertTransitionMatrix(CONFLICT_STATES, CONFLICT_EVENTS, rows, transition);
  });

  it("makes 'committed' terminal — every event from 'committed' returns 'committed'", () => {
    for (const terminal of TERMINAL_CONFLICT_STATES) {
      for (const event of CONFLICT_EVENTS) {
        expect(transition(terminal, event)).toBe(terminal);
      }
    }
  });

  it("forbids the 'detected → committed' shortcut: must pass through 'validated'", () => {
    const trace: Trace<ConflictState, ConflictEvent> = {
      name: "happy path",
      initial: "detected",
      steps: [
        { event: "assign", want: "assigned" },
        { event: "resolve", want: "resolved" },
        { event: "validate", want: "validated" },
        { event: "commit", want: "committed" },
      ],
    };
    replayTraces([trace], transition);
  });

  it("reopen from any non-initial state returns to 'detected', except from terminal", () => {
    for (const state of CONFLICT_STATES) {
      const want = state === INITIAL_CONFLICT_STATE || TERMINAL_CONFLICT_STATES.includes(state)
        ? state
        : "detected";
      expect(transition(state, "reopen")).toBe(want);
    }
  });

  it("legalEventsFor returns only events that change state", () => {
    for (const state of CONFLICT_STATES) {
      for (const event of legalEventsFor(state)) {
        expect(transition(state, event)).not.toBe(state);
      }
      // And no other event is legal:
      for (const event of CONFLICT_EVENTS) {
        if (legalEventsFor(state).includes(event)) continue;
        expect(transition(state, event)).toBe(state);
      }
    }
  });
});
