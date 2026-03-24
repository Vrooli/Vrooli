import test from "node:test";
import assert from "node:assert/strict";
import { formatPricingUsdPerMillion, formatUsdFixed, } from "../../src/lib/currency.js";
import { formatChartAxisByPreset, formatDateTime, formatRelativeTimeShort, toValidDate, } from "../../src/lib/dateTime.js";
test("formatUsdFixed supports explicit precision without grouping", () => {
    assert.equal(formatUsdFixed(1234.5, 4, { useGrouping: false }), "$1234.5000");
});
test("formatPricingUsdPerMillion preserves pricing display rules", () => {
    assert.equal(formatPricingUsdPerMillion(undefined), "-");
    assert.equal(formatPricingUsdPerMillion(0), "-");
    assert.equal(formatPricingUsdPerMillion(0.0095), "$0.0095");
    assert.equal(formatPricingUsdPerMillion(1.2), "$1.20");
});
test("toValidDate rejects invalid date strings", () => {
    assert.equal(toValidDate("not-a-date"), undefined);
});
test("formatDateTime returns fallback when date is invalid", () => {
    assert.equal(formatDateTime("not-a-date", undefined, "N/A"), "N/A");
});
test("formatRelativeTimeShort uses relative labels for recent timestamps", () => {
    const now = new Date("2026-02-07T20:00:00.000Z");
    assert.equal(formatRelativeTimeShort("2026-02-07T19:59:30.000Z", { now }), "just now");
    assert.equal(formatRelativeTimeShort("2026-02-07T19:58:00.000Z", { now }), "2m ago");
    assert.equal(formatRelativeTimeShort("2026-02-07T18:00:00.000Z", { now }), "2h ago");
    assert.equal(formatRelativeTimeShort("2026-02-05T20:00:00.000Z", { now }), "2d ago");
});
test("formatRelativeTimeShort falls back after configured day threshold", () => {
    const now = new Date("2026-02-07T20:00:00.000Z");
    const result = formatRelativeTimeShort("2026-01-20T10:00:00.000Z", {
        now,
        fallbackAfterDays: 7,
        fallbackFormatter: () => "older",
    });
    assert.equal(result, "older");
});
test("formatChartAxisByPreset returns a label for each preset family", () => {
    const value = "2026-02-07T20:00:00.000Z";
    assert.notEqual(formatChartAxisByPreset(value, "30d"), "N/A");
    assert.notEqual(formatChartAxisByPreset(value, "7d"), "N/A");
    assert.notEqual(formatChartAxisByPreset(value, "24h"), "N/A");
});
