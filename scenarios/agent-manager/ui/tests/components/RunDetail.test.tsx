import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import { createElement } from "react";
import { test, vi } from "vitest";
import { RunDetail } from "../../src/components/RunDetail.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import { makeRun } from "../testutil/runs.js";

vi.mock("../../src/components/RunTimeline.js", () => ({
  RunTimeline: () => createElement("div", { "data-testid": "run-timeline" }, "timeline"),
}));

vi.mock("../../src/components/runs/FallbackTimeline.js", () => ({
  FallbackTimeline: () => createElement("div", { "data-testid": "fallback-timeline" }, "fallback"),
}));

function renderRunDetail() {
  return renderWithProviders(
    createElement(RunDetail, {
      run: makeRun(),
      events: [],
      diff: null,
      eventsLoading: false,
      diffLoading: false,
      initialTab: "timeline",
      task: null,
      taskTitle: "Test task",
      profileName: "Test profile",
      onApprove: vi.fn(async () => undefined),
      onReject: vi.fn(async () => undefined),
      onRetry: vi.fn(async (run) => run),
      onResumeFromFailure: vi.fn(),
      onInvestigate: vi.fn(),
      onApplyInvestigation: vi.fn(),
      onStop: vi.fn(async () => undefined),
      onDelete: vi.fn(),
      onContinue: vi.fn(async () => undefined),
      onDeleteMessage: vi.fn(async () => undefined),
      deleteLoading: false,
    }),
  );
}

test("RunDetail gives the timeline tab a bounded flex layout for internal scrolling", () => {
  renderRunDetail();

  const layout = screen.getByTestId("run-detail-timeline-layout");
  assert.match(layout.className, /flex/);
  assert.match(layout.className, /h-full/);
  assert.match(layout.className, /min-h-0/);
  assert.match(layout.className, /flex-col/);

  const fallbackSlot = screen.getByTestId("fallback-timeline").parentElement;
  assert.ok(fallbackSlot);
  assert.match(fallbackSlot.className, /shrink-0/);

  const timelineSlot = screen.getByTestId("run-timeline").parentElement;
  assert.ok(timelineSlot);
  assert.match(timelineSlot.className, /min-h-0/);
  assert.match(timelineSlot.className, /flex-1/);
});
