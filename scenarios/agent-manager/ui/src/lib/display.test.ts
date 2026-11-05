import { test } from "vitest";
import assert from "node:assert/strict";
import {
  formatHyphenatedLabel,
  formatStatusLabel,
  formatUnknownLabel,
  statusTone,
} from "./display.js";

test("statusTone maps run, task and check statuses to the shared badge tones", () => {
  assert.equal(statusTone("running"), "info");
  assert.equal(statusTone("needs_review"), "warning");
  assert.equal(statusTone("complete"), "success");
  assert.equal(statusTone("failed"), "danger");
});

test("statusTone falls back to neutral for unknown values", () => {
  assert.equal(statusTone("paused"), "neutral");
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
