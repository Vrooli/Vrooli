import assert from "node:assert/strict";
import { test } from "vitest";
import { create } from "@bufbuild/protobuf";
import { UsageWindowForecastSchema, UsageWindowOutcomeSchema } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { consequence, pickWindow } from "./usageFocus.js";
import { windowKey } from "./format.js";

const HOUR = 3600_000;
const NOW = Date.UTC(2026, 9, 29, 12);
const at = (hours: number) => new Date(NOW + hours * HOUR).toISOString();

/** 34% used, burning 1.4 %/h with 120 h left: it hits 100% in 47 h, 73 h before the reset. */
function exhausting(overrides: Record<string, unknown> = {}) {
  return create(UsageWindowForecastSchema, {
    provider: "anthropic", pool: "seven_day", window: "10080m", status: "will_exhaust", hasPercent: true, usedPercent: 34, centralPercentPerHour: 1.4,
    lastingPercentPerHour: 0.55, windowStart: at(-48), resetsAt: at(120), observedAt: at(-0.1), sampleCount: 20n, projectedExhaustionAt: at(47),
    burn: [{ horizon: "24h", available: true, percentPerHour: 1.4 }], ...overrides,
  });
}

const outcome = (fields: Record<string, unknown>) => create(UsageWindowOutcomeSchema, { projected: true, ...fields });

test("the consequence states the API's run-out and shortfall when the levers still miss the reset", () => {
  const text = consequence(exhausting(), outcome({ recoverablePoints: 60, endPercent: 142, runOutAt: at(70), shortfallSeconds: 50n * 3600n }));
  assert.match(text, /move the run-out to around .* still 2 days short of the reset/);
});

test("the consequence states the API's end percent when the levers make the window last", () => {
  assert.match(consequence(exhausting(), outcome({ recoverablePoints: 110, endPercent: 92.4, reachesReset: true })), /projected to end at 92% and last until it resets/);
});

test("no recoverable saving never claims the run-out moves", () => {
  assert.match(consequence(exhausting(), outcome({ recoverablePoints: 0, endPercent: 202, runOutAt: at(47), shortfallSeconds: 73n * 3600n })), /None of the actions below recovers enough.*0\.55%\/h/);
});

test("without a projected outcome the band only states the window's status", () => {
  assert.match(consequence(exhausting({ status: "idle" }), undefined), /at the current burn\.$/);
  assert.match(consequence(exhausting(), outcome({ projected: false })), /at the current burn\.$/);
});

test("the window in the URL wins; otherwise the window the API marks as the focus", () => {
  const soon = exhausting({ pool: "a", projectedExhaustionAt: at(20), focus: true });
  const later = exhausting({ pool: "b", projectedExhaustionAt: at(60) });
  const dormant = exhausting({ pool: "d", resetsAt: at(-1), observedAt: at(-2), projectedExhaustionAt: at(1), dormant: true });
  assert.equal(pickWindow([later, dormant, soon], ""), soon);
  assert.equal(pickWindow([later, soon], windowKey(later)), later);
  assert.equal(pickWindow([later, dormant], ""), undefined, "no focus mark: nothing is chosen in its place");
  assert.equal(pickWindow([], ""), undefined);
});
