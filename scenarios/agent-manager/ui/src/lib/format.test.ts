import assert from "node:assert/strict";
import { test } from "vitest";
import { RunnerType } from "../types.js";
import {
  MISSING, formatCount, formatDuration, formatPercent, formatTokens, pluralize, shortenIds, runnerLabel, subscriptionLabel, windowLabel,
} from "./format.js";

test("durations follow one rule across magnitudes and read missing as a dash", () => {
  const cases: Array<[number | undefined, string]> = [
    [999, "999 ms"], [1500, "1.5 s"], [12_400, "12 s"], [61_000, "1 m 1 s"],
    [479_000, "7 m 59 s"], [3_660_000, "1 h 1 m"], [undefined, MISSING], [-1, MISSING],
  ];
  for (const [ms, want] of cases) assert.equal(formatDuration(ms), want);
});

test("token counts abbreviate at B, M and K and accept bigint", () => {
  assert.equal(formatTokens(31_040_000_000n), "31.0 B");
  assert.equal(formatTokens(4_600_000), "4.6 M");
  assert.equal(formatTokens(850_000), "850 K");
  assert.equal(formatTokens(1234), "1,234");
  assert.equal(formatTokens(undefined), MISSING);
});

test("counts and percents use separators and the missing dash", () => {
  assert.equal(formatCount(1234567), "1,234,567");
  assert.equal(formatCount(null), MISSING);
  assert.equal(formatPercent(0.125), "12.5%");
  assert.equal(formatPercent(Number.NaN), MISSING);
});

test("providers read as tools for runners and as plans for subscriptions", () => {
  assert.equal(runnerLabel(RunnerType.CLAUDE_CODE), "Claude Code");
  assert.equal(runnerLabel("codex"), "Codex");
  assert.equal(runnerLabel("opencode"), "OpenCode");
  assert.equal(runnerLabel("command-session"), "Command Session");
  assert.equal(runnerLabel(undefined), MISSING);
  assert.equal(subscriptionLabel("anthropic", "seven_day"), "Claude · weekly");
  assert.equal(subscriptionLabel("codex", "300m"), "Codex · 5-hour");
  assert.equal(windowLabel("1440m"), "daily");
});

test("ids are cut to their short form and counts agree with their noun", () => {
  assert.equal(shortenIds("run 6d9754e9-804a-48a9-8477-7cb972b960f7 failed"), "run 6d9754e9 failed");
  assert.equal(shortenIds(undefined), "");
  assert.equal(pluralize(1, "run"), "1 run");
  assert.equal(pluralize(2, "run"), "2 runs");
  assert.equal(pluralize(1200, "request"), "1,200 requests");
});
