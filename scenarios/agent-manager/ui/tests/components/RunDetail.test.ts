import assert from "node:assert/strict";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { test, vi } from "vitest";
import { RunDetail } from "../../src/components/RunDetail.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import { makeRun } from "../testutil/runs.js";
import type { RunDiff } from "../../src/types.js";

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

test("RunDetail dispatches its desktop lifecycle and investigation actions", async () => {
  const run = makeRun({ id: "action-run", actions: { canStop: true, canInvestigate: true, canApplyInvestigation: true, canRetry: true, canResumeFromFailure: true, canReview: true, canDelete: true } });
  const stop = vi.fn(async () => undefined); const investigate = vi.fn(); const apply = vi.fn();
  const retry = vi.fn(async (value) => value); const resume = vi.fn(); const remove = vi.fn();
  renderWithProviders(createElement(RunDetail, {
    run, events: [], diff: null, eventsLoading: false, diffLoading: false, task: null, taskTitle: "Action task", profileName: "Profile",
    onApprove: vi.fn(async () => undefined), onReject: vi.fn(async () => undefined), onRetry: retry, onResumeFromFailure: resume,
    onInvestigate: investigate, onApplyInvestigation: apply, onStop: stop, onDelete: remove,
    onContinue: vi.fn(async () => undefined), onDeleteMessage: vi.fn(async () => undefined), deleteLoading: false,
  }));
  fireEvent.click(screen.getByTitle("Stop run"));
  fireEvent.click(screen.getByTitle("Investigate"));
  fireEvent.click(screen.getByTitle("Apply Fixes"));
  fireEvent.click(screen.getByTitle("Re-run from scratch (fresh attempt, no prior context)"));
  fireEvent.click(screen.getByTitle("Resume: continue this task with the prior transcript + diff as context"));
  fireEvent.click(screen.getByTitle("Delete run"));
  await waitFor(() => assert.equal(stop.mock.calls.length, 1));
  assert.deepEqual(investigate.mock.calls, [["action-run"]]); assert.deepEqual(apply.mock.calls, [["action-run"]]);
  assert.equal(retry.mock.calls[0]?.[0], run); assert.equal(resume.mock.calls[0]?.[0], run); assert.equal(remove.mock.calls[0]?.[0], run);
});

test("RunDetail submits inline approval and rejection with trimmed operator attribution", async () => {
  const approve = vi.fn(async () => undefined); const reject = vi.fn(async () => undefined);
  const run = makeRun({ actions: { canApprove: true, canReject: true } });
  renderWithProviders(createElement(RunDetail, {
    run, events: [], diff: null, eventsLoading: false, diffLoading: false, initialTab: "diff" as never, task: null, taskTitle: "Review task", profileName: "Profile",
    onApprove: approve, onReject: reject, onRetry: vi.fn(async (value) => value), onResumeFromFailure: vi.fn(), onInvestigate: vi.fn(), onApplyInvestigation: vi.fn(),
    onStop: vi.fn(async () => undefined), onDelete: vi.fn(), onContinue: vi.fn(async () => undefined), onDeleteMessage: vi.fn(async () => undefined), deleteLoading: false,
  }));
  fireEvent.click(screen.getByRole("button", { name: "Approve All" }));
  fireEvent.change(screen.getByLabelText("Your Name (optional)"), { target: { value: "  operator  " } });
  fireEvent.change(screen.getByLabelText("Commit Message (optional)"), { target: { value: "apply report fix" } });
  fireEvent.click(screen.getByRole("button", { name: "Confirm Approval" }));
  await waitFor(() => assert.deepEqual(approve.mock.calls, [[{ actor: "operator", commitMsg: "apply report fix" }]]));
  fireEvent.click(screen.getByRole("button", { name: "Reject" }));
  fireEvent.change(screen.getByLabelText("Rejection Reason"), { target: { value: "needs more evidence" } });
  fireEvent.click(screen.getByRole("button", { name: "Confirm Rejection" }));
  await waitFor(() => assert.deepEqual(reject.mock.calls, [[{ actor: undefined, reason: "needs more evidence" }]]));
});

test("RunDetail approves only explicitly selected diff files", async () => {
  const partialApprove = vi.fn(async () => undefined);
  const diff = { files: [
    { id: "file-a", path: "api/report.go", changeType: "modified", additions: 1, deletions: 0, patch: "@@ -1 +1 @@\n+report" },
    { id: "file-b", path: "ui/report.tsx", changeType: "modified", additions: 1, deletions: 0, patch: "@@ -1 +1 @@\n+report" },
  ] } as RunDiff;
  renderWithProviders(createElement(RunDetail, {
    run: makeRun({ actions: { ...makeRun().actions, canApprove: true } }), events: [], diff, eventsLoading: false, diffLoading: false, initialTab: "diff", task: null, taskTitle: "Review task", profileName: "Profile",
    onApprove: vi.fn(async () => undefined), onReject: vi.fn(async () => undefined), onPartialApprove: partialApprove,
    onRetry: vi.fn(async (value) => value), onResumeFromFailure: vi.fn(), onInvestigate: vi.fn(), onApplyInvestigation: vi.fn(), onStop: vi.fn(async () => undefined), onDelete: vi.fn(), onContinue: vi.fn(async () => undefined), onDeleteMessage: vi.fn(async () => undefined), deleteLoading: false,
  }));
  fireEvent.click(screen.getByRole("button", { name: "Partial Approve" }));
  const checkboxes = screen.getAllByRole("checkbox");
  fireEvent.click(checkboxes[0]!);
  fireEvent.change(screen.getByLabelText("Your Name (optional)"), { target: { value: "  reviewer " } });
  fireEvent.click(screen.getByRole("button", { name: "Approve Selected (1 file)" }));
  await waitFor(() => assert.deepEqual(partialApprove.mock.calls, [[ ["file-a"], "reviewer", undefined ]]));
});

test("RunDetail explains absent task/diff/cost data and exposes its collapse and identifier controls", async () => {
  const clipboard = { writeText: vi.fn().mockResolvedValue(undefined) };
  Object.defineProperty(navigator, "clipboard", { configurable: true, value: clipboard });
  const run = makeRun({ id: "no-data-run", taskId: "missing-task", sandboxId: "" });
  renderWithProviders(createElement(RunDetail, {
    run, events: [], diff: null, eventsLoading: false, diffLoading: false, initialTab: "task", task: null, taskTitle: "Missing", profileName: "Profile",
    onApprove: vi.fn(async () => undefined), onReject: vi.fn(async () => undefined), onRetry: vi.fn(async (value) => value), onResumeFromFailure: vi.fn(), onInvestigate: vi.fn(), onApplyInvestigation: vi.fn(), onStop: vi.fn(async () => undefined), onDelete: vi.fn(), onContinue: vi.fn(async () => undefined), onDeleteMessage: vi.fn(async () => undefined), deleteLoading: false,
  }));
  assert.ok(screen.getByText("Task details unavailable for missing-task"));
  fireEvent.click(screen.getByRole("button", { name: "Diff" }));
  assert.ok(screen.getByText("This run didn't use a sandbox, so no diff was collected."));
  fireEvent.click(screen.getByRole("button", { name: "Cost" }));
  assert.ok(screen.getByText("No cost data available"));
  fireEvent.click(screen.getByTitle("Copy run ID: no-data-run"));
  await waitFor(() => assert.deepEqual(clipboard.writeText.mock.calls, [["no-data-run"]]));
  fireEvent.click(screen.getByText("Details"));
  assert.equal(screen.queryByText("Run Overview"), null);
});

test("RunDetail opens sandbox review from an available sandbox and routes desktop review into the review modal", async () => {
  const open = vi.spyOn(window, "open").mockImplementation(() => null);
  vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
    if (String(input).includes("external-url")) {
      return new Response(JSON.stringify({ url: "http://sandbox.local/review" }), { status: 200 });
    }
    return new Response(JSON.stringify({}), { status: 404 });
  }));
  const run = makeRun({ id: "sandboxed-run", sandboxId: "sandbox-4", actions: { canReview: true, canApprove: true } });
  renderWithProviders(createElement(RunDetail, {
    run, events: [], diff: null, eventsLoading: false, diffLoading: false, initialTab: "diff", task: null, taskTitle: "Review task", profileName: "Profile",
    onApprove: vi.fn(async () => undefined), onReject: vi.fn(async () => undefined), onRetry: vi.fn(async (value) => value), onResumeFromFailure: vi.fn(), onInvestigate: vi.fn(), onApplyInvestigation: vi.fn(),
    onStop: vi.fn(async () => undefined), onDelete: vi.fn(), onContinue: vi.fn(async () => undefined), onDeleteMessage: vi.fn(async () => undefined), deleteLoading: false,
  }));
  await waitFor(() => assert.ok(screen.getByRole("button", { name: "Open in Workspace Sandbox" })));
  fireEvent.click(screen.getByRole("button", { name: "Open in Workspace Sandbox" }));
  assert.deepEqual(open.mock.calls, [["http://sandbox.local/review?sandbox=sandbox-4&review=true", "_blank", "noopener,noreferrer"]]);
  fireEvent.click(screen.getByTitle("Review changes"));
  await waitFor(() => assert.ok(screen.getByRole("heading", { name: "Review Changes" })));
  vi.unstubAllGlobals();
  open.mockRestore();
});

test("RunDetail makes a failed report request actionable instead of leaving an empty panel", async () => {
  vi.stubGlobal("fetch", vi.fn(async () => new Response("not available", { status: 503 })));
  renderRunDetail("report");
  await waitFor(() => assert.ok(screen.getByText(/Report unavailable:/)));
  vi.unstubAllGlobals();
});

test("RunDetail lets desktop operators traverse every evidence tab and preserves a useful empty report", async () => {
  vi.stubGlobal("fetch", vi.fn(async (input: RequestInfo | URL) => {
    if (String(input).includes("external-url")) return new Response(JSON.stringify({}), { status: 404 });
    return new Response(JSON.stringify({
      run_id: "run-1", status: "complete", turns: 0, tokens: 0, cost_usd: 0,
      result: { selection_status: "none", candidate_count: 0 }, event_counts: {}, tools: [],
      project_owned_tool_calls: 0, external_tool_calls: 0, repeated_tool_calls: 0, files_read_more_than_once: 0,
      fallback_count: 0, diff: { files: 0, bytes: 0, available: { state: "unavailable" } },
      events_availability: { state: "unobserved" }, receipts_availability: { state: "unobserved" }, receipt_count: 0,
    }), { status: 200, headers: { "Content-Type": "application/json" } });
  }));
  const task = { id: "task-1", title: "Evidence task", description: "", scopePath: "/workspace", status: 0 } as never;
  renderWithProviders(createElement(RunDetail, {
    run: makeRun({ sandboxId: "sandbox-7" }), events: [], diff: null, eventsLoading: false, diffLoading: false, task, taskTitle: "Evidence", profileName: "Profile",
    onApprove: vi.fn(async () => undefined), onReject: vi.fn(async () => undefined), onRetry: vi.fn(async (value) => value), onResumeFromFailure: vi.fn(), onInvestigate: vi.fn(), onApplyInvestigation: vi.fn(), onStop: vi.fn(async () => undefined), onDelete: vi.fn(), onContinue: vi.fn(async () => undefined), onDeleteMessage: vi.fn(async () => undefined), deleteLoading: false,
  }));
  fireEvent.click(screen.getByRole("button", { name: "Task" }));
  assert.ok(screen.getByText("Evidence task"));
  fireEvent.click(screen.getByRole("button", { name: /Timeline/ }));
  assert.ok(screen.getByTestId("run-timeline"));
  fireEvent.click(screen.getByRole("button", { name: "Diff" }));
  assert.ok(screen.getByText("The diff hasn't been generated yet. You can review changes directly in the sandbox."));
  fireEvent.click(screen.getByRole("button", { name: "Cost" }));
  assert.ok(screen.getByText("No cost data available"));
  fireEvent.click(screen.getByRole("button", { name: "Report" }));
  await waitFor(() => assert.match(screen.getByTestId("run-report").textContent ?? "", /Status: complete/));
  const report = screen.getByTestId("run-report").textContent ?? "";
  assert.match(report, /heartbeat gap=unavailablems/);
  assert.match(report, /Final output: none \(unavailable\)/);
  assert.match(report, /Structured: unavailable/);
  assert.match(report, /requested=unavailable actual=unavailable/);
  vi.unstubAllGlobals();
});

test("RunDetail permits cancelling a partial review before selected files are submitted", () => {
  const diff = { files: [
    { id: "one", path: "api/one.go", changeType: "modified", additions: 1, deletions: 0, patch: "" },
    { id: "two", path: "api/two.go", changeType: "modified", additions: 1, deletions: 0, patch: "" },
  ] } as RunDiff;
  renderWithProviders(createElement(RunDetail, {
    run: makeRun({ actions: { ...makeRun().actions, canApprove: true } }), events: [], diff, eventsLoading: false, diffLoading: false, initialTab: "diff", task: null, taskTitle: "Review", profileName: "Profile",
    onApprove: vi.fn(async () => undefined), onReject: vi.fn(async () => undefined), onPartialApprove: vi.fn(async () => undefined), onRetry: vi.fn(async (value) => value), onResumeFromFailure: vi.fn(), onInvestigate: vi.fn(), onApplyInvestigation: vi.fn(), onStop: vi.fn(async () => undefined), onDelete: vi.fn(), onContinue: vi.fn(async () => undefined), onDeleteMessage: vi.fn(async () => undefined), deleteLoading: false,
  }));
  fireEvent.click(screen.getByRole("button", { name: "Partial Approve" }));
  assert.ok(screen.getByText("Select files above, then approve the selected subset."));
  assert.equal((screen.getByRole("button", { name: /Approve Selected/ }) as HTMLButtonElement).disabled, true);
  fireEvent.click(screen.getByRole("button", { name: "Partial Approve" }));
  assert.equal(screen.queryByText("Select files above, then approve the selected subset."), null);
});

test("RunDetail keeps the partial approval selection accurate when an operator deselects a file", () => {
  const diff = { files: [
    { id: "one", path: "api/one.go", changeType: "modified", additions: 1, deletions: 0, patch: "" },
    { id: "two", path: "api/two.go", changeType: "modified", additions: 1, deletions: 0, patch: "" },
  ] } as RunDiff;
  renderWithProviders(createElement(RunDetail, {
    run: makeRun({ actions: { ...makeRun().actions, canApprove: true } }), events: [], diff, eventsLoading: false, diffLoading: false, initialTab: "diff", task: null, taskTitle: "Review", profileName: "Profile",
    onApprove: vi.fn(async () => undefined), onReject: vi.fn(async () => undefined), onPartialApprove: vi.fn(async () => undefined), onRetry: vi.fn(async (value) => value), onResumeFromFailure: vi.fn(), onInvestigate: vi.fn(), onApplyInvestigation: vi.fn(), onStop: vi.fn(async () => undefined), onDelete: vi.fn(), onContinue: vi.fn(async () => undefined), onDeleteMessage: vi.fn(async () => undefined), deleteLoading: false,
  }));
  fireEvent.click(screen.getByRole("button", { name: "Partial Approve" }));
  const firstFile = screen.getAllByRole("checkbox")[0]!;
  fireEvent.click(firstFile);
  assert.equal((screen.getByRole("button", { name: "Approve Selected (1 file)" }) as HTMLButtonElement).disabled, false);
  fireEvent.click(firstFile);
  assert.equal((screen.getByRole("button", { name: /Approve Selected \(0 files\)/ }) as HTMLButtonElement).disabled, true);
});

test("RunDetail submits only the selected files with the operator's partial-approval metadata", async () => {
  const partialApprove = vi.fn(async () => undefined);
  const diff = { files: [
    { id: "one", path: "api/one.go", changeType: "modified", additions: 1, deletions: 0, patch: "" },
    { id: "two", path: "api/two.go", changeType: "modified", additions: 1, deletions: 0, patch: "" },
  ] } as RunDiff;
  renderWithProviders(createElement(RunDetail, {
    run: makeRun({ actions: { ...makeRun().actions, canApprove: true } }), events: [], diff, eventsLoading: false, diffLoading: false, initialTab: "diff", task: null, taskTitle: "Review", profileName: "Profile",
    onApprove: vi.fn(async () => undefined), onReject: vi.fn(async () => undefined), onPartialApprove: partialApprove,
    onRetry: vi.fn(async (value) => value), onResumeFromFailure: vi.fn(), onInvestigate: vi.fn(), onApplyInvestigation: vi.fn(), onStop: vi.fn(async () => undefined), onDelete: vi.fn(), onContinue: vi.fn(async () => undefined), onDeleteMessage: vi.fn(async () => undefined), deleteLoading: false,
  }));

  fireEvent.click(screen.getByRole("button", { name: "Partial Approve" }));
  fireEvent.click(screen.getAllByRole("checkbox")[1]!);
  fireEvent.change(screen.getByLabelText("Your Name (optional)"), { target: { value: "  Ada  " } });
  fireEvent.change(screen.getByLabelText("Commit Message (optional)"), { target: { value: "Ship safe subset" } });
  fireEvent.click(screen.getByRole("button", { name: "Approve Selected (1 file)" }));

  await waitFor(() => assert.deepEqual(partialApprove.mock.calls, [[ ["two"], "Ada", "Ship safe subset" ]]));
  assert.equal(screen.queryByText("Select files above, then approve the selected subset."), null);
});

test("RunDetail lets desktop operators close an inline approval or rejection form without submitting", () => {
  renderWithProviders(createElement(RunDetail, {
    run: makeRun({ actions: { canApprove: true, canReject: true } }), events: [], diff: null, eventsLoading: false, diffLoading: false, initialTab: "diff", task: null, taskTitle: "Review", profileName: "Profile",
    onApprove: vi.fn(async () => undefined), onReject: vi.fn(async () => undefined), onRetry: vi.fn(async (value) => value), onResumeFromFailure: vi.fn(), onInvestigate: vi.fn(), onApplyInvestigation: vi.fn(), onStop: vi.fn(async () => undefined), onDelete: vi.fn(), onContinue: vi.fn(async () => undefined), onDeleteMessage: vi.fn(async () => undefined), deleteLoading: false,
  }));
  fireEvent.click(screen.getByRole("button", { name: "Approve All" }));
  assert.ok(screen.getByRole("button", { name: "Confirm Approval" }));
  fireEvent.click(screen.getByRole("button", { name: "Approve All" }));
  assert.equal(screen.queryByRole("button", { name: "Confirm Approval" }), null);
  fireEvent.click(screen.getByRole("button", { name: "Reject" }));
  assert.ok(screen.getByRole("button", { name: "Confirm Rejection" }));
  fireEvent.click(screen.getByRole("button", { name: "Reject" }));
  assert.equal(screen.queryByRole("button", { name: "Confirm Rejection" }), null);
});

test("RunDetail distinguishes loading diff and cost evidence from completed evidence", () => {
  const props = {
    run: makeRun(), events: [], diff: null, eventsLoading: false, diffLoading: true, initialTab: "diff" as const, task: null, taskTitle: "Evidence", profileName: "Profile",
    onApprove: vi.fn(async () => undefined), onReject: vi.fn(async () => undefined), onRetry: vi.fn(async (value) => value), onResumeFromFailure: vi.fn(), onInvestigate: vi.fn(), onApplyInvestigation: vi.fn(), onStop: vi.fn(async () => undefined), onDelete: vi.fn(), onContinue: vi.fn(async () => undefined), onDeleteMessage: vi.fn(async () => undefined), deleteLoading: false,
  };
  const { rerender } = renderWithProviders(createElement(RunDetail, props));
  assert.ok(screen.getByText("Loading diff..."));
  rerender(createElement(RunDetail, { ...props, diffLoading: false, eventsLoading: true, initialTab: "cost" }));
  assert.ok(screen.getByText("Loading cost..."));
  rerender(createElement(RunDetail, {
    ...props,
    diffLoading: false,
    eventsLoading: false,
    initialTab: "cost",
    events: [{ data: { case: "cost", value: { inputTokens: 10, outputTokens: 5, totalCostUsd: 0.125, model: "gpt-5", serviceTier: "priority" } } }] as never,
  }));
  assert.ok(screen.getAllByText("$0.1250").length >= 1);
  assert.ok(screen.getAllByText("15").length >= 1);
});

test("RunDetail leaves the review controls usable after an approval operation fails", async () => {
  const approve = vi.fn(async () => { throw new Error("backend unavailable"); });
  const log = vi.spyOn(console, "error").mockImplementation(() => undefined);
  renderWithProviders(createElement(RunDetail, {
    run: makeRun({ actions: { canApprove: true } }), events: [], diff: null, eventsLoading: false, diffLoading: false, initialTab: "diff", task: null, taskTitle: "Review", profileName: "Profile",
    onApprove: approve, onReject: vi.fn(async () => undefined), onRetry: vi.fn(async (value) => value), onResumeFromFailure: vi.fn(), onInvestigate: vi.fn(), onApplyInvestigation: vi.fn(), onStop: vi.fn(async () => undefined), onDelete: vi.fn(), onContinue: vi.fn(async () => undefined), onDeleteMessage: vi.fn(async () => undefined), deleteLoading: false,
  }));
  fireEvent.click(screen.getByRole("button", { name: "Approve All" }));
  fireEvent.click(screen.getByRole("button", { name: "Confirm Approval" }));
  await waitFor(() => assert.equal(log.mock.calls.length, 1));
  assert.ok(screen.getByRole("button", { name: "Confirm Approval" }));
  log.mockRestore();
});
