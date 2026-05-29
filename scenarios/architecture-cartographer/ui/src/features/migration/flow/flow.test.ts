import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { describe, expect, it } from "vitest";

import { assertTransitionMatrix, type MatrixRow } from "../../../test-utils/modeltest/matrix";
import { replayTraces, type Trace } from "../../../test-utils/modeltest/traces";

import {
  INITIAL_MIGRATION_STATE,
  MIGRATION_EVENTS,
  MIGRATION_STATES,
  TERMINAL_MIGRATION_STATES,
  isOpenState,
  legalActionsFor,
  transition,
  type MigrationEvent,
  type MigrationFindingState,
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

describe("migration finding-lifecycle flow", () => {
  it("declares the same states transition.ts knows about", () => {
    const declared = flowJson.states.map((state) => state.id).sort();
    const runtime = [...MIGRATION_STATES].sort();
    expect(declared).toEqual(runtime);
  });

  it("declares the same events transition.ts knows about", () => {
    const declared = flowJson.events.map((event) => event.id).sort();
    const runtime = [...MIGRATION_EVENTS].sort();
    expect(declared).toEqual(runtime);
  });

  it("declares 'detected' as the only initial state and matches INITIAL_MIGRATION_STATE", () => {
    const initials = flowJson.states.filter((state) => state.initial === true).map((state) => state.id);
    expect(initials).toEqual([INITIAL_MIGRATION_STATE]);
  });

  it("transition.ts agrees with flow.json across the full transition matrix", () => {
    const rows: MatrixRow<MigrationFindingState, MigrationEvent>[] = flowJson.transitions.map((row) => ({
      name: `${row.from}/${row.event}`,
      from: row.from as MigrationFindingState,
      event: row.event as MigrationEvent,
      to: row.to as MigrationFindingState,
    }));
    assertTransitionMatrix(MIGRATION_STATES, MIGRATION_EVENTS, rows, transition);
  });

  it("makes the sink states terminal — every event returns the same state", () => {
    for (const terminal of TERMINAL_MIGRATION_STATES) {
      for (const event of MIGRATION_EVENTS) {
        expect(transition(terminal, event)).toBe(terminal);
      }
    }
  });

  it("walks the happy path: detected → resolved → validated", () => {
    const trace: Trace<MigrationFindingState, MigrationEvent> = {
      name: "ingest → fix → re-audit",
      initial: "detected",
      steps: [
        { event: "resolve", want: "resolved" },
        { event: "validate", want: "validated" },
      ],
    };
    replayTraces([trace], transition);
  });

  it("regresses a reappeared fix back to detected", () => {
    const trace: Trace<MigrationFindingState, MigrationEvent> = {
      name: "fix didn't hold",
      initial: "detected",
      steps: [
        { event: "apply", want: "resolved" },
        { event: "validate", want: "validated" },
        { event: "regress", want: "detected" },
      ],
    };
    replayTraces([trace], transition);
  });

  it("close never changes a finding's status", () => {
    for (const state of MIGRATION_STATES) {
      expect(transition(state, "close")).toBe(state);
    }
  });

  it("legalActionsFor exposes resolve/apply only on open states", () => {
    for (const state of MIGRATION_STATES) {
      const actions = legalActionsFor(state);
      if (isOpenState(state)) {
        expect([...actions].sort()).toEqual(["apply", "resolve"]);
      } else {
        expect(actions).toEqual([]);
      }
    }
  });
});
