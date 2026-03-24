import test from "node:test";
import assert from "node:assert/strict";
import { formatHyphenatedLabel, formatStatusLabel, formatUnknownLabel, statusBadgeVariant, } from "../../src/lib/display.js";
test("statusBadgeVariant returns known status variants", () => {
    assert.equal(statusBadgeVariant("running"), "running");
    assert.equal(statusBadgeVariant("needs_review"), "needs_review");
});
test("statusBadgeVariant falls back to secondary for unknown values", () => {
    assert.equal(statusBadgeVariant("paused"), "secondary");
});
test("formatStatusLabel title-cases underscore status text", () => {
    assert.equal(formatStatusLabel("needs_review"), "Needs Review");
});
test("formatHyphenatedLabel title-cases hyphenated text", () => {
    assert.equal(formatHyphenatedLabel("claude-code"), "Claude Code");
});
test("formatUnknownLabel normalizes empty and unknown values", () => {
    assert.equal(formatUnknownLabel(""), "Unknown");
    assert.equal(formatUnknownLabel("unknown"), "Unknown");
    assert.equal(formatUnknownLabel("codex"), "codex");
});
