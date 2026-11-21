import { fireEvent, screen } from "@testing-library/react";
import { createElement } from "react";
import { expect, test, vi } from "vitest";
import { MobileInfoDialog } from "./MobileInfoDialog.js";
import { makeRun } from "../../test-utils/fixtures/runs.js";
import { renderWithProviders } from "../../test-utils/index.js";

const costTotals = { inputTokens: 0, outputTokens: 0, cacheCreationTokens: 0, cacheReadTokens: 0, totalCostUsd: 0, webSearchRequests: 0, serverToolUseRequests: 0, models: [], serviceTiers: [], events: 0 };

function open(runIdCopyStatus: string, handlers = { onCopyRunId: vi.fn(), onClose: vi.fn() }) {
  renderWithProviders(createElement(MobileInfoDialog, {
    run: makeRun({ id: "run-abcdef123456" }), diff: null, taskTitle: "Fix the parser", profileName: "Reliability",
    durationMs: 479_000, costTotals, runIdCopyStatus, ...handlers,
  }));
  return handlers;
}

test("shows the run's details with its short id and lets the operator copy it", () => {
  const handlers = open("idle");
  expect(screen.getByRole("heading", { name: "Run Details" })).toBeTruthy();
  expect(screen.getByText("run-abcd")).toBeTruthy();
  fireEvent.click(screen.getByTitle("Copy run ID: run-abcdef123456"));
  expect(handlers.onCopyRunId).toHaveBeenCalledTimes(1);
});

test("confirms a copy with a check mark instead of the copy icon", () => {
  open("copied");
  expect(screen.getByTitle("Copy run ID: run-abcdef123456").querySelector(".text-emerald-500")).toBeTruthy();
});

test("closes from the header button and from the Escape key", () => {
  const handlers = open("idle");
  fireEvent.click(screen.getByRole("button", { name: "Close" }));
  fireEvent.keyDown(document, { key: "Escape" });
  expect(handlers.onClose).toHaveBeenCalledTimes(2);
});
