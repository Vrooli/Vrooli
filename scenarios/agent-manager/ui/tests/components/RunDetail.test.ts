import assert from "node:assert/strict";
import { screen, waitFor } from "@testing-library/react";
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

function renderRunDetail(initialTab: "timeline" | "report" = "timeline") {
  return renderWithProviders(createElement(RunDetail, {
    run: makeRun(), events: [], diff: null, eventsLoading: false, diffLoading: false, initialTab, task: null,
    taskTitle: "Test task", profileName: "Test profile", onApprove: vi.fn(async () => undefined), onReject: vi.fn(async () => undefined),
    onRetry: vi.fn(async (run) => run), onResumeFromFailure: vi.fn(), onInvestigate: vi.fn(), onApplyInvestigation: vi.fn(),
    onStop: vi.fn(async () => undefined), onDelete: vi.fn(), onContinue: vi.fn(async () => undefined), onDeleteMessage: vi.fn(async () => undefined), deleteLoading: false,
  }));
}

test("RunDetail gives the timeline tab a bounded flex layout for internal scrolling", () => {
  renderRunDetail();
  const layout = screen.getByTestId("run-detail-timeline-layout");
  assert.match(layout.className, /flex/); assert.match(layout.className, /h-full/); assert.match(layout.className, /min-h-0/); assert.match(layout.className, /flex-col/);
  const fallbackSlot = screen.getByTestId("fallback-timeline").parentElement;
  assert.ok(fallbackSlot); assert.match(fallbackSlot.className, /shrink-0/);
  const timelineSlot = screen.getByTestId("run-timeline").parentElement;
  assert.ok(timelineSlot); assert.match(timelineSlot.className, /min-h-0/); assert.match(timelineSlot.className, /flex-1/);
});

test("RunDetail renders the shared report projection including event histogram", async () => {
  vi.stubGlobal("fetch", vi.fn(async () => new Response(JSON.stringify({
    run_id: "run-1", status: "failed", exit_code: 1, error: "tool failed", duration_ms: "1250", heartbeat_gap_ms: "500", turns: 2, tokens: 42, cost_usd: 0.25,
    result: { selection_status: "ambiguous", selection_rule: "multiple", candidate_count: 2, structured_status: "invalid", diagnostic_codes: ["schema_invalid"] },
    event_counts: { tool_result: 1, "model.fallback.attempted": 1 }, tools: [{ name: "bash", calls: 1, successes: 0, failures: 1 }],
    project_owned_tool_calls: 1, external_tool_calls: 0, requested_model: "primary", actual_model: "fallback", fallback_count: 1, repeated_tool_calls: 1, files_read_more_than_once: 1, longest_event_gap_ms: "500",
    diff: { files: 1, bytes: 8, available: { state: "available" } }, events_availability: { state: "available" }, receipts_availability: { state: "unobserved" }, receipt_count: 0,
  }), { status: 200, headers: { "Content-Type": "application/json" } })));
  renderRunDetail("report");
  await waitFor(() => assert.ok(screen.getByTestId("run-report")));
  const report = screen.getByTestId("run-report");
  assert.match(report.textContent ?? "", /tool_result: 1/); assert.match(report.textContent ?? "", /model\.fallback\.attempted: 1/);
  assert.match(report.textContent ?? "", /schema_invalid/); assert.match(report.textContent ?? "", /project-owned=1/);
  vi.unstubAllGlobals();
});
