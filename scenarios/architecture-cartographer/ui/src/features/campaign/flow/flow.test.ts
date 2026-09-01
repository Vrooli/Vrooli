import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { describe, expect, it } from "vitest";

import { assertTransitionMatrix, type MatrixRow } from "@vrooli/flow-runtime";
import { replayTraces, type Trace } from "@vrooli/flow-runtime";

import {
  INITIAL_CAMPAIGN_STATE,
  CAMPAIGN_EVENTS,
  CAMPAIGN_STATES,
  TERMINAL_CAMPAIGN_STATES,
  isOpenState,
  legalActionsFor,
  transition,
  type CampaignEvent,
  type CampaignItemState,
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

describe("campaign item-lifecycle flow", () => {
  it("declares the same states transition.ts knows about", () => {
    const declared = flowJson.states.map((state) => state.id).sort();
    const runtime = [...CAMPAIGN_STATES].sort();
    expect(declared).toEqual(runtime);
  });

  it("declares the same events transition.ts knows about", () => {
    const declared = flowJson.events.map((event) => event.id).sort();
    const runtime = [...CAMPAIGN_EVENTS].sort();
    expect(declared).toEqual(runtime);
  });

  it("declares 'detected' as the only initial state and matches INITIAL_CAMPAIGN_STATE", () => {
    const initials = flowJson.states.filter((state) => state.initial === true).map((state) => state.id);
    expect(initials).toEqual([INITIAL_CAMPAIGN_STATE]);
  });

  it("transition.ts agrees with flow.json across the full transition matrix", () => {
    const rows: MatrixRow<CampaignItemState, CampaignEvent>[] = flowJson.transitions.map((row) => ({
      name: `${row.from}/${row.event}`,
      from: row.from as CampaignItemState,
      event: row.event as CampaignEvent,
      to: row.to as CampaignItemState,
    }));
    assertTransitionMatrix(CAMPAIGN_STATES, CAMPAIGN_EVENTS, rows, transition);
  });

  it("makes the sink states terminal — every event returns the same state", () => {
    for (const terminal of TERMINAL_CAMPAIGN_STATES) {
      for (const event of CAMPAIGN_EVENTS) {
        expect(transition(terminal, event)).toBe(terminal);
      }
    }
  });

  it("walks the happy path: detected → resolved → validated", () => {
    const trace: Trace<CampaignItemState, CampaignEvent> = {
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
    const trace: Trace<CampaignItemState, CampaignEvent> = {
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

  it("close never changes an item's status", () => {
    for (const state of CAMPAIGN_STATES) {
      expect(transition(state, "close")).toBe(state);
    }
  });

  it("legalActionsFor exposes resolve/apply only on open states", () => {
    for (const state of CAMPAIGN_STATES) {
      const actions = legalActionsFor(state);
      if (isOpenState(state)) {
        expect([...actions].sort()).toEqual(["apply", "resolve"]);
      } else {
        expect(actions).toEqual([]);
      }
    }
  });
});
