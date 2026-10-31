import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { runInNewContext } from "node:vm";
import { describe, expect, it, vi } from "vitest";

type WorkflowNode = {
  id: string;
  action?: { evaluate?: { expression?: string } };
};

const casePath = resolve(
  process.cwd(),
  "../bas/cases/performance/p1-mobile-shared-route-responsiveness.json",
);
const workflow = JSON.parse(readFileSync(casePath, "utf8")) as { nodes: WorkflowNode[] };
function requireFinalizeExpression(expression: string | undefined): string {
  if (typeof expression !== "string" || expression.length === 0) {
    throw new Error("Planner responsiveness case has no finalize-sequence expression");
  }
  return expression;
}

const finalizeExpression = requireFinalizeExpression(
  workflow.nodes.find((node) => node.id === "finalize-sequence")?.action?.evaluate?.expression,
);

type Transition = {
  key: string;
  route: string;
  status: string;
  ready_condition: string;
  click_ms: number;
  click_event_timestamp_ms: number;
  click_target_testid: string;
  ready_ms: number;
};

type EventTimingEntry = {
  name: "click";
  start_ms: number;
  duration_ms: number;
  processing_start_ms: number;
  processing_end_ms: number;
  interaction_id: number;
  target_testid: string;
};

async function finalize(
  transition: Transition,
  eventTimings: EventTimingEntry[],
  eventTimingSupported: boolean,
  measures: Array<{ name: string; startTime: number; duration: number }>,
) {
  const clickListener = vi.fn();
  const disconnect = vi.fn();
  const probe = {
    transitions: [transition],
    longTasks: [],
    eventTimings,
    eventTimingSupported,
    observerAvailable: true,
    clickListener,
    eventObserver: { disconnect },
    longObserver: { takeRecords: () => [], disconnect },
    serviceWorker: { status: "unavailable" },
  };
  const removeEventListener = vi.fn();
  const cancelAnimationFrame = vi.fn();
  const result = await runInNewContext(finalizeExpression, {
    window: { __plannerRouteProbe: probe },
    document: { removeEventListener },
    performance: { getEntriesByType: () => measures },
    cancelAnimationFrame,
    setTimeout,
    clearTimeout,
  });

  expect(removeEventListener).toHaveBeenCalledWith("click", clickListener, true);
  expect(disconnect).toHaveBeenCalledTimes(2);
  return result;
}

describe("Planner BAS responsiveness measurement expression", () => {
  it("matches a delayed input by its event timestamp and preserves both clocks", async () => {
    const result = await finalize(
      {
        key: "plan",
        route: "plan",
        status: "ready",
        ready_condition: "seeded plan data is ready",
        click_event_timestamp_ms: 1000,
        click_ms: 2605,
        click_target_testid: "layout-nav-plan-tab",
        ready_ms: 80,
      },
      [{
        name: "click",
        start_ms: 1000,
        duration_ms: 1632,
        processing_start_ms: 2600,
        processing_end_ms: 2615,
        interaction_id: 17,
        target_testid: "layout-nav-plan-tab",
      }],
      true,
      [{ name: "⚛ Plan (update)", startTime: 2640, duration: 7.5 }],
    );

    const transition = result.transitions[0];
    expect(transition.event_timing.status).toBe("observed");
    expect(transition.event_timing.entry.start_after_input_timestamp_ms).toBe(0);
    expect(transition.event_timing.entry.handler_minus_input_timestamp_ms).toBe(1605);
    expect(transition.event_timing.entry.input_delay_ms).toBe(1600);
    expect(transition.event_timing.entry.start_after_handler_ms).toBe(-1605);
    expect(transition.ready_ms).toBe(80);
    expect(transition.react_profiler.status).toBe("observed");
    expect(transition.react_profiler.observed_subtree_duration_sum_ms).toBe(7.5);
  });

  it("keeps unsupported event timing and absent profiler measures unavailable", async () => {
    const settingsTransition = {
      key: "settings",
      route: "settings",
      status: "ready",
      ready_condition: "preferences surface is ready",
      click_event_timestamp_ms: 9000,
      click_ms: 9012,
      click_target_testid: "layout-nav-settings-tab",
      ready_ms: 42,
    };
    const [unsupported, supportedWithoutEntry] = await Promise.all([
      finalize(settingsTransition, [], false, []),
      finalize(settingsTransition, [], true, []),
    ]);

    for (const result of [unsupported, supportedWithoutEntry]) {
      const transition = result.transitions[0];
      expect(transition.event_timing.entry).toBeNull();
      expect(transition.react_profiler.status).toBe("unavailable_no_commit_measure_in_window");
      expect(transition.react_profiler.observed_subtree_duration_sum_ms).toBeNull();
      expect(result.react_profiler_observed_subtree_duration_sum_ms).toBeNull();
    }
    expect(unsupported.transitions[0].event_timing.status).toBe("unavailable_unsupported");
    expect(supportedWithoutEntry.transitions[0].event_timing.status).toBe(
      "unavailable_no_entry_at_16ms_threshold",
    );
  });
});
