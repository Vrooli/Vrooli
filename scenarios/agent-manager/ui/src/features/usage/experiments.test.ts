import assert from "node:assert/strict";
import { create } from "@bufbuild/protobuf";
import { UsageExperimentPlanSchema, UsageOutcomeRowSchema, ListUsageModesResponseSchema } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { test } from "vitest";
import { assumedAcceptanceText, experimentSentence, headline, horizonText, humanDays, isExemptRole, learnRoles, paceText, ratioText, shareWords, verdictOf } from "./experiments.js";
import { modeSpans } from "./UsageWindows.js";

const plan = (detect: { reached: boolean; days: number; lowDays: number; highDays: number; perArm?: bigint }) =>
  create(UsageExperimentPlanSchema, { detect: { perArm: 70n, ...detect } });

test("days read as an operator plans: days, weeks, months, years", () => {
  assert.deepEqual([0, 3.2, 20, 100, 1200].map(humanDays), ["not reached", "4 days", "3 weeks", "4 months", "over 3 years"]);
});

test("a horizon states its likely range, or that the upper end is beyond the simulation, or that it was not reached", () => {
  assert.equal(horizonText({ reached: true, days: 35, lowDays: 15, highDays: 60, perArm: 40n, $typeName: "agent_manager.v1.measures.UsageExperimentHorizon" } as never), "about 5 weeks, likely 3 weeks to 9 weeks");
  assert.match(horizonText({ reached: true, days: 185, lowDays: 92, highDays: 0, perArm: 210n, $typeName: "agent_manager.v1.measures.UsageExperimentHorizon" } as never), /about 7 months, at least 4 months and the upper end is beyond/);
  assert.match(horizonText(undefined), /not within the range simulated/);
});

test("the headline is decided by the slowest chosen role and never promises", () => {
  const quick = headline([plan({ reached: true, days: 35, lowDays: 21, highDays: 42 })]);
  assert.equal(quick.text, "Probably 3 weeks to 6 weeks, and it could take longer.");
  assert.ok(quick.bar && quick.bar.from < quick.bar.to && quick.bar.to <= 1);
  const slowest = headline([plan({ reached: true, days: 35, lowDays: 21, highDays: 42 }), plan({ reached: true, days: 100, lowDays: 60, highDays: 200 })]);
  assert.match(slowest.text, /9 weeks to 7 months/);
  const years = headline([plan({ reached: true, days: 1200, lowDays: 460, highDays: 3300 })]);
  assert.match(years.text, /over 3 years, so a large effect is the only thing this could show/);
  assert.equal(years.open, true);
  assert.match(headline([plan({ reached: false, days: 0, lowDays: 0, highDays: 0 })]).text, /too quiet to compare/);
  assert.match(headline([]).text, /Choose a role/);
});

test("roles come from managed runs, never from sessions, with a weekly pace and exempt roles last", () => {
  const row = (key: string, runner: string, units: number) => create(UsageOutcomeRowSchema, { key, runner, units: BigInt(units) });
  const roles = learnRoles([row("operator_session", "codex", 30), row("sub_agent", "claude-code", 9), row("code.review", "codex", 12), row("code.default", "codex", 68), row("code.economy.delivery", "codex", 9), row("empty", "codex", 0), row("x", "grok", 5)], 30);
  assert.deepEqual(roles.map((r) => [r.key, r.runner, r.exempt]), [["code.default", "codex", false], ["code.economy.delivery", "codex", false], ["code.review", "codex", true]]);
  assert.equal(Math.round(roles[0]?.perWeek ?? 0), 16);
  assert.equal(paceText(0.3), "under 1 a week");
  assert.equal(paceText(15.9), "about 16 a week");
  assert.equal(isExemptRole("code.review"), true);
  assert.equal(isExemptRole("code.default"), false);
});

test("split words and verdicts", () => {
  assert.deepEqual([50, 20, 30].map(shareWords), ["about half", "about a fifth", "about 30%"]);
  assert.equal(verdictOf("nonsense"), "not_enough_data");
  assert.match(experimentSentence({ role: "code.default", verdict: "about_the_same" } as never), /look about the same/);
  assert.match(experimentSentence({ role: "code.default", verdict: "higher_looks_better" } as never), /higher model looks better/);
  assert.match(experimentSentence({ role: "code.default", verdict: "inconclusive" } as never), /does not show whether the two differ/);
  assert.match(experimentSentence({ role: "code.default", verdict: "withheld" } as never), /withheld/);
  assert.equal(verdictOf("withheld"), "withheld");
  assert.equal(assumedAcceptanceText({ incumbent: { accepted: 5n, assumedAccepted: 0n }, challenger: { accepted: 5n, assumedAccepted: 0n } } as never), "");
});

test("window spans include ended periods and let an active mode win for the same id", () => {
  const response = create(ListUsageModesResponseSchema, {
    periods: [
      { id: "ended", label: "Old stretch", runner: "codex", startedAt: "2026-10-01T00:00:00Z", endedAt: "2026-10-02T00:00:00Z" },
      { id: "live", label: "", runner: "codex", startedAt: "2026-10-03T00:00:00Z", expiresAt: "2026-10-09T00:00:00Z" },
      { id: "other", label: "Claude", runner: "claude", startedAt: "2026-10-01T00:00:00Z", endedAt: "2026-10-02T00:00:00Z" },
    ],
    modes: [{ mode: { id: "live", label: "Lower tier", scope: { runner: "codex" }, startedAt: "2026-10-03T00:00:00Z", expiresAt: "2026-10-08T00:00:00Z" } }],
  });
  const spans = modeSpans(response, "codex");
  assert.deepEqual(spans.map((s) => [s.id, s.label, s.endsAt]), [["ended", "Old stretch", "2026-10-02T00:00:00Z"], ["live", "Lower tier", "2026-10-08T00:00:00Z"]]);
  assert.deepEqual(modeSpans(undefined, "codex"), []);
});

test("an estimate with no stated upper end is doubled for the bar, and a quiet plan is not a bar at all", () => {
  const open = headline([plan({ reached: true, days: 40, lowDays: 20, highDays: 0 })]);
  assert.match(open.text, /3 weeks to 12 weeks/);
  assert.ok(open.bar && open.bar.to <= 1 && open.bar.from < open.bar.to);
  assert.equal(horizonText({ reached: true, days: 2000, lowDays: 900, highDays: 0, perArm: 5n, $typeName: "agent_manager.v1.measures.UsageExperimentHorizon" } as never).includes("at least 2 years"), true);
  assert.equal(ratioText(undefined), "unknown");
  assert.equal(ratioText(Number.NaN), "unknown");
  assert.equal(ratioText(0.5), "0.50x");
  assert.equal(experimentSentence({ role: "r", verdict: "cheaper_looks_better" } as never).startsWith("For r, the cheaper model looks better"), true);
  assert.match(experimentSentence({ role: "r", verdict: "" } as never), /not enough data yet/);
});
