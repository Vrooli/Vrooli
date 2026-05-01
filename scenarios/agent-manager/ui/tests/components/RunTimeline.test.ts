import assert from "node:assert/strict";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { test, vi } from "vitest";
import { RunTimeline } from "../../src/components/RunTimeline.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import { makeRun } from "../testutil/runs.js";
import {
  makeMessageEvent,
  makeRunEvent,
  RUN_EVENT_TYPE_LOG,
} from "../testutil/runEvents.js";

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
