import assert from "node:assert/strict";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { test, vi } from "vitest";
import { RunTimeline } from "../../src/components/RunTimeline.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import { makeRun } from "../testutil/runs.js";
import { RunStatus } from "../../src/types.js";
import {
  makeMessageEvent,
  makeRunEvent,
  RUN_EVENT_TYPE_LOG,
} from "../testutil/runEvents.js";
import { makeToolCallEvent, makeToolResultEvent } from "../testutil/runEvents.js";

const FILTER_STORAGE_KEY = "agm.runTimelineFilters.v1";

function renderTimeline(events = [
  makeMessageEvent("msg-1", 1n, "Visible answer"),
]) {
  return renderWithProviders(
    createElement(RunTimeline, {
      run: makeRun(),
      events,
      eventsLoading: false,
      onContinue: vi.fn(async () => undefined),
      onDeleteMessage: vi.fn(async () => undefined),
    }),
  );
}

test("RunTimeline all filter reveals operational log entries through the component UI", async () => {
  const user = userEvent.setup();
  renderTimeline([
    makeMessageEvent("msg-1", 1n, "Visible answer"),
    makeRunEvent({
      id: "log-1",
      sequence: 2n,
      eventType: RUN_EVENT_TYPE_LOG,
      data: { case: "log", value: { level: "info", message: "phase: background sync complete" } },
    }),
  ]);

  assert.ok(screen.getByText("Visible answer"));
  assert.equal(screen.queryByText("background sync complete"), null);

  await user.click(screen.getByRole("button", { name: /open timeline filters/i }));
  await user.click(screen.getByRole("button", { name: "All" }));

  assert.ok(await screen.findByText("background sync complete"));
  assert.equal(screen.queryByText("Visible answer"), null);

  await waitFor(() => {
    const raw = window.localStorage.getItem(FILTER_STORAGE_KEY);
    assert.ok(raw);
    assert.equal(JSON.parse(raw).mode, "events");
  });
});

test("RunTimeline restores persisted conversation filters before rendering events", () => {
  window.localStorage.setItem(
    FILTER_STORAGE_KEY,
    JSON.stringify({
      mode: "conversation",
      categories: {
        messages: true,
        reasoning: true,
        tools: true,
        errors: true,
        status: true,
        logs: true,
        artifacts: true,
        metrics: true,
        compaction: true,
        redactions: true,
      },
    }),
  );

  renderTimeline([
    makeMessageEvent("msg-1", 1n, "Persisted message view"),
    makeRunEvent({
      id: "reasoning-1",
      sequence: 2n,
      eventType: RUN_EVENT_TYPE_LOG,
      data: { case: "log", value: { level: "debug", message: "Thinking: hidden reasoning" } },
    }),
  ]);

  assert.ok(screen.getByText("Persisted message view"));
  assert.equal(screen.queryByText("hidden reasoning"), null);
  assert.match(
    screen.getByRole("button", { name: /open timeline filters/i }).getAttribute("aria-label") ?? "",
    /Conversation mode/,
  );
});

test("RunTimeline allows follow-up while showing a sandbox finalization warning", () => {
  renderWithProviders(
    createElement(RunTimeline, {
      run: makeRun({
        sessionId: "session-1",
        actions: {
          canInvestigate: false,
          canApplyInvestigation: false,
          canDelete: true,
          canStop: false,
          canRetry: true,
          canContinue: true,
          canApprove: false,
          canReject: false,
          canReview: false,
          canExtractRecommendations: false,
          canRegenerateRecommendations: false,
          canContinueReason: "",
          canResumeFromFailure: false,
          canResumeFromFailureReason: "",
          finalizationWarning: "Sandbox finalization failed: checkpoint rejected",
          canRetryFinalization: true,
        },
      }),
      events: [makeMessageEvent("msg-1", 1n, "Ready for follow-up")],
      eventsLoading: false,
      onContinue: vi.fn(async () => undefined),
      onDeleteMessage: vi.fn(async () => undefined),
    }),
  );

  assert.ok(screen.getByText("Sandbox finalization failed: checkpoint rejected"));
  assert.ok(screen.getByPlaceholderText("Type your follow-up message..."));
  assert.equal(screen.getByRole("button", { name: "Send message" }).hasAttribute("disabled"), true);
});

test("RunTimeline sends a follow-up, copies and deletes a message, and expands grouped tool activity", async () => {
  window.localStorage.removeItem(FILTER_STORAGE_KEY);
  const onContinue = vi.fn(async () => undefined);
  const onDeleteMessage = vi.fn(async () => undefined);
  const clipboard = { writeText: vi.fn(async () => undefined) };
  Object.defineProperty(navigator, "clipboard", { configurable: true, value: clipboard });
  vi.stubGlobal("confirm", vi.fn(() => true));
  renderWithProviders(createElement(RunTimeline, {
    run: makeRun({ actions: { ...makeRun().actions, canContinue: true } }),
    events: [
      makeMessageEvent("message-1", 1n, "A useful response"),
      makeToolCallEvent("call-1", 2n, "Read", "one"),
      makeToolResultEvent("result-1", 3n, "Read", "one"),
      makeRunEvent({ id: "thinking-1", sequence: 4n, eventType: RUN_EVENT_TYPE_LOG, data: { case: "log", value: { message: "Reasoning: inspect the result" } } }),
      makeToolCallEvent("call-2", 5n, "Write", "two"),
      makeToolResultEvent("result-2", 6n, "Write", "two", false),
    ],
    eventsLoading: false, onContinue, onDeleteMessage,
  }));
  fireEvent.change(screen.getByPlaceholderText("Type your follow-up message..."), { target: { value: "  try the safe fix  " } });
  fireEvent.click(screen.getByRole("button", { name: "Send message" }));
  await waitFor(() => assert.deepEqual(onContinue.mock.calls, [["try the safe fix", undefined]]));
  fireEvent.click(screen.getByTitle("Copy message"));
  await waitFor(() => assert.deepEqual(clipboard.writeText.mock.calls, [["A useful response"]]));
  fireEvent.click(screen.getByTitle("Delete message"));
  await waitFor(() => assert.deepEqual(onDeleteMessage.mock.calls, [["message-1"]]));
  fireEvent.click(screen.getByRole("button", { name: /Activity/ }));
  assert.ok(screen.getByText("Read"));
  assert.ok(screen.getByText("Write"));
  fireEvent.click(screen.getByText("Write"));
  assert.ok(screen.getByText("Input"));
  assert.ok(screen.getByText("Error"));
});

test("RunTimeline gives clear loading, empty-filter, generating, and unavailable-continuation feedback", () => {
  const props = { onContinue: vi.fn(async () => undefined), onDeleteMessage: vi.fn(async () => undefined) };
  const loading = renderWithProviders(createElement(RunTimeline, { run: makeRun(), events: [], eventsLoading: true, ...props }));
  assert.ok(screen.getByText("Loading timeline..."));
  loading.unmount();
  window.localStorage.setItem(FILTER_STORAGE_KEY, JSON.stringify({ mode: "events", categories: { messages: false, reasoning: false, tools: false, errors: false, status: false, logs: false, artifacts: false, metrics: false, compaction: false, redactions: false } }));
  const empty = renderWithProviders(createElement(RunTimeline, { run: makeRun({ actions: { ...makeRun().actions, canContinueReason: "Run is archived" } }), events: [makeMessageEvent("msg", 1n, "hidden")], eventsLoading: false, ...props }));
  assert.ok(screen.getByText("No timeline entries match the current filters")); assert.ok(screen.getByText("Run is archived"));
  empty.unmount();
  window.localStorage.removeItem(FILTER_STORAGE_KEY);
  renderWithProviders(createElement(RunTimeline, { run: makeRun({ status: RunStatus.RUNNING }), events: [makeMessageEvent("live", 1n, "working")], eventsLoading: false, ...props }));
  assert.ok(screen.getByText("Run is still generating new timeline entries..."));
  assert.equal((screen.getByRole("button", { name: "Send message" }) as HTMLButtonElement).disabled, true);
  assert.ok(screen.getByPlaceholderText("Type your follow-up message while the run completes..."));
});

test("RunTimeline handles deleted-message reveal and interactive filter category controls", async () => {
  const user = userEvent.setup();
  window.localStorage.removeItem(FILTER_STORAGE_KEY);
  renderTimeline([
    makeMessageEvent("deleted-message", 1n, "Redacted evidence"),
    makeRunEvent({ id: "redact", sequence: 2n, data: { case: "messageDeleted", value: { targetEventId: "deleted-message" } } }),
  ]);
  assert.ok(screen.getByText("Message deleted"));
  await user.click(screen.getByRole("button", { name: "Show" }));
  assert.ok(screen.getByText("Redacted evidence"));
  await user.click(screen.getByRole("button", { name: "Hide" }));
  await user.click(screen.getByRole("button", { name: /open timeline filters/i }));
  assert.ok(screen.getByText("Timeline Filters"));
  await user.click(screen.getByText("All"));
  assert.ok(screen.getByText("Redaction"));
  const messageCheckbox = screen.getAllByRole("checkbox")[0]!;
  await user.click(messageCheckbox);
  assert.match(window.localStorage.getItem(FILTER_STORAGE_KEY) ?? "", /"messages":false/);
  await user.click(screen.getByRole("button", { name: "Hybrid" }));
  await user.click(screen.getByRole("button", { name: "Done" }));
  assert.equal(screen.queryByText("Timeline Filters"), null);
});

test("RunTimeline keeps the operator draft and exposes a failed follow-up submission", async () => {
  window.localStorage.removeItem(FILTER_STORAGE_KEY);
  const onContinue = vi.fn(async () => { throw new Error("runner is unavailable"); });
  renderWithProviders(createElement(RunTimeline, {
    run: makeRun({ id: "failed-follow-up", actions: { ...makeRun().actions, canContinue: true } }),
    events: [makeMessageEvent("message-1", 1n, "Need another pass")], eventsLoading: false,
    onContinue, onDeleteMessage: vi.fn(async () => undefined),
  }));
  const input = screen.getByPlaceholderText("Type your follow-up message...");
  fireEvent.change(input, { target: { value: "retry with logs" } });
  fireEvent.keyDown(input, { key: "Enter" });
  await waitFor(() => assert.deepEqual(onContinue.mock.calls, [["retry with logs", undefined]]));
  assert.ok(screen.getByText("runner is unavailable"));
  assert.equal((input as HTMLTextAreaElement).value, "retry with logs");
});

test("RunTimeline closes its filter popover on Escape and outside interaction without changing the operator view", async () => {
  const user = userEvent.setup();
  window.localStorage.removeItem(FILTER_STORAGE_KEY);
  renderTimeline([makeMessageEvent("message-1", 1n, "Keep this visible")]);

  await user.click(screen.getByRole("button", { name: /open timeline filters/i }));
  assert.ok(screen.getByText("Timeline Filters"));
  await user.keyboard("{Escape}");
  await waitFor(() => assert.equal(screen.queryByText("Timeline Filters"), null));
  assert.ok(screen.getByText("Keep this visible"));

  await user.click(screen.getByRole("button", { name: /open timeline filters/i }));
  assert.ok(screen.getByText("Timeline Filters"));
  fireEvent.mouseDown(document.body);
  await waitFor(() => assert.equal(screen.queryByText("Timeline Filters"), null));
  assert.ok(screen.getByText("Keep this visible"));
});

test("RunTimeline exposes every operational event category with inspectable payloads", async () => {
  const user = userEvent.setup();
  window.localStorage.setItem(FILTER_STORAGE_KEY, JSON.stringify({
    mode: "events",
    categories: {
      messages: true, reasoning: true, tools: true, errors: true, status: true,
      logs: true, artifacts: true, metrics: true, compaction: true, redactions: true,
    },
  }));
  renderTimeline([
    makeRunEvent({ id: "reason", sequence: 1n, eventType: RUN_EVENT_TYPE_LOG, data: { case: "log", value: { message: "Reasoning: assess tool failures" } } }),
    makeRunEvent({ id: "error", sequence: 2n, data: { case: "error", value: { message: "Error failure" } } }),
    makeRunEvent({ id: "status", sequence: 3n, eventType: 5, data: { case: "status", value: { oldStatus: "running", newStatus: "failed" } } }),
    makeRunEvent({ id: "artifact", sequence: 4n, data: { case: "artifact", value: { path: "reports/run.json" } } }),
    makeRunEvent({ id: "metric", sequence: 5n, data: { case: "metric", value: { name: "tokens", value: 42 } } }),
    makeRunEvent({ id: "compact", sequence: 6n, data: { case: "compaction", value: { summary: "Context compacted" } } }),
    makeRunEvent({ id: "redact", sequence: 7n, data: { case: "messageDeleted", value: { targetEventId: "message-1" } } }),
    makeRunEvent({ id: "log", sequence: 8n, data: { case: "log", value: { message: "phase: persisted audit" } } }),
  ]);

  for (const summary of [
    "assess tool failures", "Error failure", "running -> failed", "reports/run.json",
    "tokens: 42", "Context compacted", "Message message- redacted", "persisted audit",
  ]) {
    const button = screen.getByText(summary).closest("button");
    assert.ok(button, `expected expandable event for ${summary}`);
    await user.click(button);
  }

  assert.match(document.body.textContent ?? "", /Error failure/);
  assert.match(document.body.textContent ?? "", /reports\/run\.json/);
});

test("RunTimeline presents attached user evidence and retains a revealed message when deletion is declined", async () => {
  window.localStorage.removeItem(FILTER_STORAGE_KEY);
  vi.stubGlobal("confirm", vi.fn(() => false));
  renderTimeline([
    makeRunEvent({
      id: "user-evidence", sequence: 1n,
      data: { case: "message", value: {
        role: "user", content: "Screenshot evidence",
        attachments: [{ id: "att-1", fileName: "evidence.png", url: "https://example.test/evidence.png" }],
      } },
    }),
  ]);

  const image = screen.getByRole("img", { name: "evidence.png" });
  assert.equal(image.getAttribute("src"), "https://example.test/evidence.png");
  fireEvent.click(screen.getByTitle("Delete message"));
  assert.ok(screen.getByText("Screenshot evidence"));
  assert.equal((window.confirm as ReturnType<typeof vi.fn>).mock.calls.length, 1);
  vi.unstubAllGlobals();
});

test("RunTimeline keeps a multiline draft when Shift+Enter is used and resets empty filters", async () => {
  const user = userEvent.setup();
  const onContinue = vi.fn(async () => undefined);
  window.localStorage.setItem(FILTER_STORAGE_KEY, JSON.stringify({
    mode: "events",
    categories: { messages: false, reasoning: false, tools: false, errors: false, status: false, logs: false, artifacts: false, metrics: false, compaction: false, redactions: false },
  }));
  renderWithProviders(createElement(RunTimeline, {
    run: makeRun({ actions: { ...makeRun().actions, canContinue: true } }),
    events: [makeMessageEvent("draft", 1n, "Restored after reset")], eventsLoading: false,
    onContinue, onDeleteMessage: vi.fn(async () => undefined),
  }));

  await user.click(screen.getByRole("button", { name: "Reset Filters" }));
  const input = screen.getByPlaceholderText("Type your follow-up message...");
  await user.type(input, "first line");
  fireEvent.keyDown(input, { key: "Enter", shiftKey: true });
  assert.deepEqual(onContinue.mock.calls, []);
  assert.equal((input as HTMLTextAreaElement).value, "first line");
  assert.ok(screen.getByText("Restored after reset"));
});

test("RunTimeline uploads selected evidence and continues with the server attachment id", async () => {
  const user = userEvent.setup();
  const onContinue = vi.fn(async () => undefined);
  const fetch = vi.fn(async () => new Response(JSON.stringify({
    id: "attachment-server-id",
    file_name: "evidence.png",
    content_type: "image/png",
    file_size: 7,
    storage_path: "/attachments/evidence.png",
    url: "https://files.example.test/evidence.png",
  }), { status: 200, headers: { "content-type": "application/json" } }));
  vi.stubGlobal("fetch", fetch);
  window.localStorage.removeItem(FILTER_STORAGE_KEY);

  renderWithProviders(createElement(RunTimeline, {
    run: makeRun({ id: "attachment-follow-up", actions: { ...makeRun().actions, canContinue: true } }),
    events: [makeMessageEvent("message-1", 1n, "Attach supporting evidence")],
    eventsLoading: false,
    onContinue,
    onDeleteMessage: vi.fn(async () => undefined),
  }));

  const input = document.querySelector<HTMLInputElement>('input[type="file"]');
  assert.ok(input);
  await user.upload(input, new File(["evidence"], "evidence.png", { type: "image/png" }));
  await waitFor(() => assert.equal(fetch.mock.calls.length, 1));
  await waitFor(() => assert.ok(screen.getByTestId("attachment-preview-container")));
  await waitFor(() => assert.equal((screen.getByRole("button", { name: "Send message" }) as HTMLButtonElement).disabled, false));

  await user.click(screen.getByRole("button", { name: "Send message" }));
  await waitFor(() => assert.deepEqual(onContinue.mock.calls, [["", ["attachment-server-id"]]]));
  await waitFor(() => assert.equal(screen.queryByTestId("attachment-preview-container"), null));
  assert.equal((fetch.mock.calls[0]?.[1] as RequestInit).method, "POST");
  vi.unstubAllGlobals();
});

test("RunTimeline shows pending grouped tool work without inventing a result", async () => {
  const user = userEvent.setup();
  window.localStorage.removeItem(FILTER_STORAGE_KEY);
  renderTimeline([
    makeToolCallEvent("pending-call", 1n, "workspace-sandbox inspect", "pending-tool"),
    makeToolCallEvent("completed-call", 2n, "prompt-manager search", "completed-tool"),
    makeToolResultEvent("completed-result", 3n, "prompt-manager search", "completed-tool"),
  ]);

  const activity = screen.getByRole("button", { name: /Activity/ });
  assert.match(activity.textContent ?? "", /workspace-sandbox inspect/);
  await user.click(activity);
  await user.click(screen.getAllByRole("button", { name: /workspace-sandbox inspect/ })[1]!);
  assert.ok(screen.getByText("Input"));
  assert.equal(screen.queryByText("Output"), null);
  assert.equal(screen.queryByText("Error"), null);
});

test("RunTimeline recovers from malformed stored filters and represents system messages", () => {
  window.localStorage.setItem(FILTER_STORAGE_KEY, "{not-json");
  renderTimeline([
    makeRunEvent({
      id: "system-note", sequence: 1n,
      data: { case: "message", value: { role: "system", content: "Policy updated" } },
    }),
  ]);

  assert.ok(screen.getByText("Policy updated"));
  assert.ok(screen.getByText("System"));
  assert.match(
    screen.getByRole("button", { name: /open timeline filters/i }).getAttribute("aria-label") ?? "",
    /Combined mode/,
  );
});

test("RunTimeline normalizes invalid persisted filters and preserves mobile drafts until an explicit send", async () => {
  const user = userEvent.setup();
  const originalWidth = window.innerWidth;
  Object.defineProperty(window, "innerWidth", { configurable: true, value: 600 });
  window.localStorage.setItem(FILTER_STORAGE_KEY, JSON.stringify({
    mode: "unsupported-mode",
    categories: { logs: false },
  }));
  const onContinue = vi.fn(async () => undefined);

  renderWithProviders(createElement(RunTimeline, {
    run: makeRun({ actions: { ...makeRun().actions, canContinue: true } }),
    events: [
      makeMessageEvent("mobile-message", 1n, "Keep the mobile draft"),
      makeRunEvent({
        id: "hidden-log", sequence: 2n, eventType: RUN_EVENT_TYPE_LOG,
        data: { case: "log", value: { message: "This log remains filtered" } },
      }),
    ],
    eventsLoading: false,
    onContinue,
    onDeleteMessage: vi.fn(async () => undefined),
  }));

  assert.match(
    screen.getByRole("button", { name: /open timeline filters/i }).getAttribute("aria-label") ?? "",
    /Combined mode/,
  );
  assert.ok(screen.getByText("Keep the mobile draft"));
  assert.equal(screen.queryByText("This log remains filtered"), null);
  assert.match(window.localStorage.getItem(FILTER_STORAGE_KEY) ?? "", /"mode":"combined"/);

  const input = screen.getByPlaceholderText("Type your follow-up message...");
  await user.type(input, "do not submit on mobile");
  fireEvent.keyDown(input, { key: "Enter" });
  assert.deepEqual(onContinue.mock.calls, []);
  assert.equal((input as HTMLTextAreaElement).value, "do not submit on mobile");
  assert.equal(screen.queryByText(/Press Enter to send/), null);

  Object.defineProperty(window, "innerWidth", { configurable: true, value: originalWidth });
});

test("RunTimeline preserves visible history on failed removal and ignores clipboard failures", async () => {
  window.localStorage.removeItem(FILTER_STORAGE_KEY);
  vi.stubGlobal("confirm", vi.fn(() => true));
  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: { writeText: vi.fn(async () => { throw new Error("clipboard blocked"); }) },
  });
  const onDeleteMessage = vi.fn(async () => { throw new Error("history is read-only"); });
  renderWithProviders(createElement(RunTimeline, {
    run: makeRun(),
    events: [makeMessageEvent("removed", 1n, "Retained in history")],
    eventsLoading: false,
    onContinue: vi.fn(async () => undefined),
    onDeleteMessage,
  }));

  assert.ok(screen.getByText("Retained in history"));
  const user = userEvent.setup();
  await user.click(screen.getByTitle("Copy message"));
  await user.click(screen.getByTitle("Delete message"));
  await waitFor(() => assert.deepEqual(onDeleteMessage.mock.calls, [["removed"]]));
  assert.ok(screen.getByText("Retained in history"));
  vi.unstubAllGlobals();
});

test("RunTimeline clears a failed submission notice when the operator edits the retained draft", async () => {
  window.localStorage.removeItem(FILTER_STORAGE_KEY);
  const onContinue = vi.fn(async () => { throw new Error("runner is unavailable"); });
  renderWithProviders(createElement(RunTimeline, {
    run: makeRun({ id: "clear-follow-up-error", actions: { ...makeRun().actions, canContinue: true } }),
    events: [makeMessageEvent("message-1", 1n, "Need another pass")],
    eventsLoading: false,
    onContinue,
    onDeleteMessage: vi.fn(async () => undefined),
  }));

  const input = screen.getByPlaceholderText("Type your follow-up message...");
  fireEvent.change(input, { target: { value: "retry with logs" } });
  fireEvent.click(screen.getByRole("button", { name: "Send message" }));
  await waitFor(() => assert.ok(screen.getByText("runner is unavailable")));

  fireEvent.change(input, { target: { value: "retry with the raw event payload" } });
  assert.equal(screen.queryByText("runner is unavailable"), null);
  assert.equal((input as HTMLTextAreaElement).value, "retry with the raw event payload");
});

test("RunTimeline lets an operator inspect grouped reasoning and each tool outcome", async () => {
  const user = userEvent.setup();
  window.localStorage.removeItem(FILTER_STORAGE_KEY);
  renderTimeline([
    makeRunEvent({
      id: "grouped-reasoning", sequence: 1n, eventType: RUN_EVENT_TYPE_LOG,
      data: { case: "log", value: { message: "Reasoning: compare the API receipt" } },
    }),
    makeToolCallEvent("pending-tool", 2n, "workspace-sandbox inspect", "pending"),
    makeToolCallEvent("successful-tool", 3n, "prompt-manager query", "success"),
    makeToolResultEvent("successful-result", 4n, "prompt-manager query", "success"),
    makeToolCallEvent("failed-tool", 5n, "test-genie runs wait", "failed"),
    makeToolResultEvent("failed-result", 6n, "test-genie runs wait", "failed", false),
  ]);

  await user.click(screen.getByRole("button", { name: /Activity/ }));
  await user.click(screen.getByRole("button", { name: /compare the API receipt/ }));
  assert.ok(screen.getByText("compare the API receipt"));

  for (const name of ["workspace-sandbox inspect", "prompt-manager query", "test-genie runs wait"]) {
    const matches = screen.getAllByRole("button", { name: new RegExp(name) });
    await user.click(matches[matches.length - 1]!);
  }
  assert.equal(screen.getAllByText("Input").length, 3);
  assert.ok(screen.getByText("Output"));
  assert.ok(screen.getByText("Error"));
});

test("RunTimeline keeps its filter panel open for panel and trigger interactions, then repositions it", async () => {
  const user = userEvent.setup();
  window.localStorage.removeItem(FILTER_STORAGE_KEY);
  renderTimeline([makeMessageEvent("filter-panel", 1n, "Keep filters deliberate")]);

  const trigger = screen.getByRole("button", { name: /open timeline filters/i });
  Object.defineProperty(trigger, "getBoundingClientRect", {
    configurable: true,
    value: () => ({ top: 740, bottom: 776, left: 1000, right: 1040, width: 40, height: 36 }),
  });

  await user.click(trigger);
  const panel = screen.getByText("Timeline Filters").closest("div.fixed");
  assert.ok(panel);
  fireEvent.mouseDown(panel);
  assert.ok(screen.getByText("Timeline Filters"));
  fireEvent.mouseDown(trigger);
  assert.ok(screen.getByText("Timeline Filters"));
  fireEvent.resize(window);
  assert.match(panel.className, /origin-bottom-right/);
  await user.click(screen.getByRole("button", { name: "Done" }));
});

test("RunTimeline handles empty file selection, non-Error continuation failures, and a default unavailable reason", async () => {
  window.localStorage.removeItem(FILTER_STORAGE_KEY);
  const onContinue = vi.fn(async () => { throw "transport disconnected"; });
  const active = renderWithProviders(createElement(RunTimeline, {
    run: makeRun({ id: "non-error-send", actions: { ...makeRun().actions, canContinue: true } }),
    events: [makeMessageEvent("message-1", 1n, "Try another continuation")],
    eventsLoading: false,
    onContinue,
    onDeleteMessage: vi.fn(async () => undefined),
  }));

  const fileInput = document.querySelector<HTMLInputElement>('input[type="file"]');
  assert.ok(fileInput);
  fireEvent.change(fileInput, { target: { files: [] } });
  fireEvent.change(screen.getByPlaceholderText("Type your follow-up message..."), { target: { value: "retry" } });
  fireEvent.click(screen.getByRole("button", { name: "Send message" }));
  await waitFor(() => assert.ok(screen.getByText("Failed to continue run")));
  active.unmount();

  renderWithProviders(createElement(RunTimeline, {
    run: { ...makeRun(), actions: undefined },
    events: [makeMessageEvent("unavailable", 1n, "Archived history")],
    eventsLoading: false,
    onContinue: vi.fn(async () => undefined),
    onDeleteMessage: vi.fn(async () => undefined),
  }));
  assert.ok(screen.getByText("Continuation not available"));
});

test("RunTimeline shows successful grouped activity and exposes every completed tool output", async () => {
  const user = userEvent.setup();
  window.localStorage.removeItem(FILTER_STORAGE_KEY);
  renderWithProviders(createElement(RunTimeline, {
    run: makeRun({ actions: { ...makeRun().actions, canContinue: true } }),
    events: [
      makeToolCallEvent("success-call-1", 1n, "prompt-manager query", "success-1"),
      makeToolResultEvent("success-result-1", 2n, "prompt-manager query", "success-1"),
      makeToolCallEvent("success-call-2", 3n, "workspace-sandbox inspect", "success-2"),
      makeToolResultEvent("success-result-2", 4n, "workspace-sandbox inspect", "success-2"),
    ],
    eventsLoading: false,
    onContinue: vi.fn(async () => undefined),
    onDeleteMessage: vi.fn(async () => undefined),
  }));

  const activity = screen.getByRole("button", { name: /Activity/ });
  assert.match(activity.innerHTML, /text-green-600/);
  await user.click(activity);
  const queryButtons = screen.getAllByRole("button", { name: /prompt-manager query/ });
  const sandboxButtons = screen.getAllByRole("button", { name: /workspace-sandbox inspect/ });
  await user.click(queryButtons[queryButtons.length - 1]!);
  await user.click(sandboxButtons[sandboxButtons.length - 1]!);
  assert.equal(screen.getAllByText("Output").length, 2);
});
