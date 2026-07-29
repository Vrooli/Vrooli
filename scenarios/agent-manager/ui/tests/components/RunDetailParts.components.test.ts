import assert from "node:assert/strict";
import { fireEvent, screen } from "@testing-library/react";
import { createElement, createRef } from "react";
import { test, vi } from "vitest";
import { CostBreakdown, MobileHeaderActions, RunDetailsContent, RunModelBadge, StatusDotWithLegend } from "../../src/components/RunDetailParts.js";
import { ExecutionMode, RunPhase, RunStatus } from "../../src/types.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import { makeRun } from "../testutil/runs.js";

test("RunDetailsContent makes run provenance, pause state, live session, failure, and costs inspectable", () => {
  const run = makeRun({
    status: RunStatus.PARKED, phase: RunPhase.AWAITING_REVIEW, executionMode: ExecutionMode.INTERACTIVE,
    errorMsg: "runner disconnected", sandboxId: "sandbox-1", changedFiles: 3, actualModel: "fallback-model", tag: "incident-1",
    webConsoleSessionUrl: "https://console.test/session", awaitHandle: { producer: "test-genie", key: "run-1" },
    resolvedConfig: { roleRef: "investigator", maxTurns: 20, effort: "high", features: { enableBrowser: true }, extraFlags: { codex: { flags: ["--json"] } } },
  });
  const totals = { inputTokens: 10, outputTokens: 20, cacheCreationTokens: 2, cacheReadTokens: 3, totalCostUsd: 0.125, webSearchRequests: 1, serverToolUseRequests: 2, models: ["fallback-model"], serviceTiers: ["priority"], events: 2 };
  renderWithProviders(createElement("div", null,
    createElement(RunDetailsContent, { run, taskTitle: "Investigate reports", profileName: "Reliability", durationMs: 1250, costTotals: totals }),
    createElement(CostBreakdown, { totals }),
  ));
  assert.ok(screen.getByText("runner disconnected")); assert.ok(screen.getByText("Parked — waiting, not hung"));
  assert.ok(screen.getByTestId("open-live-session")); assert.ok(screen.getByText("investigator")); assert.ok(screen.getByText("fallback-model"));
  assert.equal(screen.getAllByText("$0.1250").length, 2); assert.ok(screen.getByText("Web search: 1")); assert.ok(screen.getByText("Service tiers: priority"));
});

test("mobile run actions and status legend dispatch only enabled operational controls", () => {
  const calls = { info: vi.fn(), stop: vi.fn(), investigate: vi.fn(), apply: vi.fn(), retry: vi.fn(), resume: vi.fn(), review: vi.fn(), remove: vi.fn(), menu: vi.fn(), legend: vi.fn() };
  renderWithProviders(createElement("div", null,
    createElement(StatusDotWithLegend, { dotColor: "bg-warning", statusLabel: "needs_review", legendOpen: true, setLegendOpen: calls.legend }),
    createElement(MobileHeaderActions, {
      actions: { ...makeRun().actions, canStop: true, canInvestigate: true, canRetry: true, canResumeFromFailure: true, canReview: true },
      onInfoOpen: calls.info, onStop: calls.stop, actionsMenuOpen: true, setActionsMenuOpen: calls.menu, actionsMenuRef: createRef(), onInvestigate: calls.investigate, canApplyFixes: true, onApplyInvestigation: calls.apply, onRetry: calls.retry, onResumeFromFailure: calls.resume, onReview: calls.review, canDeleteRun: true, onDelete: calls.remove, deleteLoading: false,
    }),
  ));
  assert.ok(screen.getByText("Needs review"));
  fireEvent.click(screen.getByTitle("Run details")); fireEvent.click(screen.getByTitle("Stop run"));
  fireEvent.click(screen.getByRole("button", { name: "Investigate" })); fireEvent.click(screen.getByRole("button", { name: "Apply Fixes" })); fireEvent.click(screen.getByRole("button", { name: "Re-run" })); fireEvent.click(screen.getByRole("button", { name: "Resume" })); fireEvent.click(screen.getByRole("button", { name: "Review" })); fireEvent.click(screen.getByRole("button", { name: "Delete" }));
  for (const fn of [calls.info, calls.stop, calls.investigate, calls.apply, calls.retry, calls.resume, calls.review, calls.remove]) assert.equal(fn.mock.calls.length, 1);
  fireEvent.click(screen.getByTitle("needs review")); assert.deepEqual(calls.legend.mock.calls, [[false]]);
});

test("RunModelBadge distinguishes requested model fallback from default execution", () => {
  const { rerender } = renderWithProviders(createElement(RunModelBadge, { requested: "primary", actual: "fallback", fallbackChain: ["primary", "fallback"] }));
  assert.ok(screen.getByText("model: fallback (fallback)"));
  assert.match(screen.getByText("model: fallback (fallback)").getAttribute("title") ?? "", /Fallback chain/);
  rerender(createElement(RunModelBadge, { requested: "", actual: "" }));
  assert.equal(screen.queryByText(/model:/), null);
});

test("RunDetailsContent distinguishes a live session that is discoverable in web-console from one still starting", () => {
  const totals = { inputTokens: 0, outputTokens: 0, cacheCreationTokens: 0, cacheReadTokens: 0, totalCostUsd: 0, webSearchRequests: 0, serverToolUseRequests: 0, models: [], serviceTiers: [], events: 0 };
  const { rerender } = renderWithProviders(createElement(RunDetailsContent, {
    run: makeRun({ executionMode: ExecutionMode.INTERACTIVE, webConsoleSessionId: "session-42", tag: "" }), taskTitle: "Run", profileName: "Profile", durationMs: null, costTotals: totals,
  }));
  assert.ok(screen.getByText("session-42"));
  assert.ok(screen.getAllByText("None").length >= 1);
  assert.ok(screen.getByText("—"));
  rerender(createElement(RunDetailsContent, {
    run: makeRun({ executionMode: ExecutionMode.INTERACTIVE, webConsoleSessionId: "", tag: "" }), taskTitle: "Run", profileName: "Profile", durationMs: null, costTotals: totals,
  }));
  assert.ok(screen.getByText("Session starting…"));
});

test("CostBreakdown omits usage context when cost events do not identify a model or service tier", () => {
  renderWithProviders(createElement(CostBreakdown, {
    totals: { inputTokens: 1, outputTokens: 2, cacheCreationTokens: 0, cacheReadTokens: 0, totalCostUsd: 0, webSearchRequests: 0, serverToolUseRequests: 0, models: [], serviceTiers: [], events: 1 },
  }));
  assert.ok(screen.getByText("Total tokens"));
  assert.equal(screen.queryByText("Usage context"), null);
});

test("StatusDotWithLegend closes only for an outside interaction", () => {
  const setLegendOpen = vi.fn();
  renderWithProviders(createElement(StatusDotWithLegend, {
    dotColor: "bg-primary", statusLabel: "running", legendOpen: true, setLegendOpen,
  }));
  assert.ok(screen.getByText("Status"));
  fireEvent.mouseDown(screen.getByText("Running"));
  assert.deepEqual(setLegendOpen.mock.calls, []);
  fireEvent.mouseDown(document.body);
  assert.deepEqual(setLegendOpen.mock.calls, [[false]]);
});

test("MobileHeaderActions surfaces review from approval capability and protects a deleting run", () => {
  const review = vi.fn();
  const remove = vi.fn();
  renderWithProviders(createElement(MobileHeaderActions, {
    actions: { ...makeRun().actions, canApprove: true, canReview: false },
    onInfoOpen: vi.fn(), onStop: vi.fn(), actionsMenuOpen: true, setActionsMenuOpen: vi.fn(),
    actionsMenuRef: createRef(), onInvestigate: vi.fn(), canApplyFixes: false,
    onApplyInvestigation: vi.fn(), onRetry: vi.fn(), onResumeFromFailure: vi.fn(), onReview: review,
    canDeleteRun: true, onDelete: remove, deleteLoading: true,
  }));
  fireEvent.click(screen.getByRole("button", { name: "Review" }));
  assert.equal(review.mock.calls.length, 1);
  assert.equal((screen.getByRole("button", { name: "Delete" }) as HTMLButtonElement).disabled, true);
  fireEvent.click(screen.getByRole("button", { name: "Run actions" }));
});
