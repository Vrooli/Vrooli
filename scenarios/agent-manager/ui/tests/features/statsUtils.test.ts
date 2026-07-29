import assert from "node:assert/strict";
import { test } from "vitest";
import {
  avgBucketField,
  calculateSuccessRate,
  calculateThroughput,
  getWindowHours,
  percentChange,
  sumBucketField,
} from "../../src/features/stats/utils/calculations.js";
import { getSeriesColor, getStatusColor, CHART_COLORS } from "../../src/features/stats/utils/chartConfig.js";
import { formatCompact, formatDuration, formatNumber, formatPercent, formatTokens } from "../../src/features/stats/utils/formatters.js";

const buckets = [
  { runsStarted: 4, runsCompleted: 3, runsFailed: 1, totalCostUsd: 1.5, avgDurationMs: 0 },
  { runsStarted: 6, runsCompleted: 5, runsFailed: 0, totalCostUsd: 2.5, avgDurationMs: 90_000 },
];

test("stats calculations handle completed-only rates, empty windows, presets, totals, and nonzero averages", () => {
  assert.equal(calculateSuccessRate({ complete: 3, failed: 1, pending: 2, running: 0, cancelled: 0, needsReview: 0, total: 6 }), 0.75);
  assert.equal(calculateSuccessRate({ complete: 0, failed: 0, pending: 2, running: 0, cancelled: 0, needsReview: 0, total: 2 }), 0);
  assert.equal(calculateThroughput(buckets, 4), 2);
  assert.equal(calculateThroughput(buckets, 0), 0);
  assert.equal(getWindowHours("6h"), 6);
  assert.equal(getWindowHours("12h"), 12);
  assert.equal(getWindowHours("24h"), 24);
  assert.equal(getWindowHours("7d"), 168);
  assert.equal(getWindowHours("30d"), 720);
  assert.equal(getWindowHours("unknown"), 24);
  assert.equal(percentChange(10, 0), 100);
  assert.equal(percentChange(0, 0), 0);
  assert.equal(percentChange(15, 10), 50);
  assert.equal(sumBucketField(buckets, "runsStarted"), 10);
  assert.equal(sumBucketField(buckets, "totalCostUsd"), 4);
  assert.equal(avgBucketField(buckets, "avgDurationMs"), 90_000);
  assert.equal(avgBucketField([{ ...buckets[0]!, avgDurationMs: 0 }], "avgDurationMs"), 0);
});

test("stats formatting and chart colors expose stable operator-facing values", () => {
  assert.equal(formatCompact(1_200), "1.2K");
  assert.equal(formatDuration(999), "999ms");
  assert.equal(formatDuration(1_500), "1.5s");
  assert.equal(formatDuration(90_000), "1.5m");
  assert.equal(formatDuration(7_200_000), "2.0h");
  assert.equal(formatPercent(0.1234, 2), "12.34%");
  assert.equal(formatNumber(12_345), "12,345");
  assert.equal(formatTokens(999), "999");
  assert.equal(formatTokens(12_500), "12.5K");
  assert.equal(formatTokens(1_250_000), "1.25M");
  assert.equal(getStatusColor("SUCCESS"), CHART_COLORS.complete);
  assert.equal(getStatusColor("error"), CHART_COLORS.failed);
  assert.equal(getStatusColor("needs_review"), CHART_COLORS.needsReview);
  assert.equal(getStatusColor("mystery"), CHART_COLORS.muted);
  assert.equal(getSeriesColor(0), CHART_COLORS.series[0]);
  assert.equal(getSeriesColor(CHART_COLORS.series.length), CHART_COLORS.series[0]);
});
