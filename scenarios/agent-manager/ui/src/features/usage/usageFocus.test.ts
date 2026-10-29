import assert from "node:assert/strict";
import { test } from "vitest";
import { create } from "@bufbuild/protobuf";
import { UsageWindowForecastSchema } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { focusFigures, pickWindow } from "./usageFocus.js";
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

test("focus figures: burn, affordable burn and the cut the window needs", () => {
  const figures = focusFigures(exhausting(), 0, NOW);
  assert.equal(figures.burn, 1.4);
  assert.equal(figures.affordable, 0.55);
  assert.ok(Math.abs((figures.cutNeeded ?? 0) - (1 - 0.55 / 1.4)) < 1e-9);
});

test("actions that recover less than the overshoot move the run-out but say it still misses the reset", () => {
  // 34 + 1.4 * 120 = 202% at the reset; recovering 60 pts leaves 142%.
  const figures = focusFigures(exhausting(), 60, NOW);
  assert.equal(figures.reachesReset, false);
  assert.match(figures.consequence, /move the run-out to around .* still \d+ (days|hours) short of the reset/);
});

test("actions that recover the whole overshoot say the window lasts", () => {
  const figures = focusFigures(exhausting(), 110, NOW);
  assert.equal(figures.reachesReset, true);
  assert.match(figures.consequence, /projected to end at 92% and last until it resets/);
});

test("no recoverable saving never claims the run-out moves", () => {
  const figures = focusFigures(exhausting(), 0, NOW);
  assert.equal(figures.reachesReset, false);
  assert.match(figures.consequence, /None of the actions below recovers enough/);
});

test("a window that already lasts has no cut and says so", () => {
  const figures = focusFigures(exhausting({ status: "idle" }), 0, NOW);
  assert.equal(figures.cutNeeded, undefined);
  assert.equal(figures.reachesReset, true);
});

test("the window in the URL wins; otherwise the soonest run-out; dormant windows are never selected", () => {
  const soon = exhausting({ pool: "a", projectedExhaustionAt: at(20) });
  const later = exhausting({ pool: "b", projectedExhaustionAt: at(60) });
  const quiet = exhausting({ pool: "c", status: "on_track", usedPercent: 90 });
  const dormant = exhausting({ pool: "d", resetsAt: at(-1), observedAt: at(-2), status: "will_exhaust", projectedExhaustionAt: at(1) });
  assert.equal(pickWindow([later, quiet, dormant, soon], "", NOW), soon);
  assert.equal(pickWindow([later, soon], windowKey(later), NOW), later);
  assert.equal(pickWindow([quiet, exhausting({ pool: "e", status: "on_track", usedPercent: 10 })], "", NOW), quiet, "otherwise the most used");
  assert.equal(pickWindow([], "", NOW), undefined);
});

test("the burn is labelled measured only when a 24 h reading exists; otherwise it is the projection's rate", () => {
  assert.equal(focusFigures(exhausting(), 0, NOW).burnMeasured, true);
  const noRate = focusFigures(exhausting({ burn: [{ horizon: "24h", available: false }] }), 0, NOW);
  assert.equal(noRate.burnMeasured, false);
  assert.equal(noRate.burn, 1.4);
});

test("late in the window only what is still recoverable counts: a full-week saving would wrongly claim the window lasts", () => {
  // 168 h window, 16 h left: 60% used and 3 %/h projects 108% at the reset.
  const late = exhausting({ usedPercent: 60, centralPercentPerHour: 3, resetsAt: at(16), windowStart: at(-152), projectedExhaustionAt: at(13) });
  assert.equal(focusFigures(late, 60, NOW).reachesReset, true, "the full-horizon saving would say it lasts");
  const honest = focusFigures(late, 5, NOW);
  assert.equal(honest.reachesReset, false);
  assert.match(honest.consequence, /still \d+ hours short of the reset/);
});
