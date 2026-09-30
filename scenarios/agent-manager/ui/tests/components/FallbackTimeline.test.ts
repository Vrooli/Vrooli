import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { test, vi } from "vitest";
import { FallbackTimeline } from "../../src/components/runs/FallbackTimeline.js";
import { renderWithProviders } from "../../src/test-utils/index.js";

const query = vi.fn();
vi.mock("@tanstack/react-query", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@tanstack/react-query")>()),
  useQuery: (input: unknown) => query(input),
}));
vi.mock("../../src/features/health/api/eventsClient.js", () => ({
  eventsQueryKeys: { forRun: (runId: string) => ["events", runId] },
  fetchEventsForRun: vi.fn(),
}));

test("FallbackTimeline collapses, loads, errors, filters non-fallback events, and renders all fallback summaries", async () => {
  const user = userEvent.setup();
  query.mockReturnValue({ data: undefined, isLoading: false, error: null });
  const { rerender } = renderWithProviders(createElement(FallbackTimeline, { runId: "run-1" }));
  assert.equal(screen.queryByTestId("fallback-timeline-empty"), null);
  await user.click(screen.getByRole("button", { name: /Fallback timeline/ }));
  assert.ok(screen.getByTestId("fallback-timeline-empty"));

  query.mockReturnValue({ data: undefined, isLoading: true, error: null });
  rerender(createElement(FallbackTimeline, { runId: "run-1", defaultOpen: true }));
  assert.ok(screen.getByText("Loading…"));
  query.mockReturnValue({ data: undefined, isLoading: false, error: new Error("network") });
  rerender(createElement(FallbackTimeline, { runId: "run-1", defaultOpen: true }));
  assert.ok(screen.getByText("Failed to load fallback events: network"));

  query.mockReturnValue({ data: { events: [
    { id: "a", event_type: "runner.fallback.attempted", timestamp: "2026-07-29T00:00:00Z", payload: { from: "a", to: "b", reason: "down", attempt_no: 2, chain_position: 1, chain_length: 3 } },
    { id: "b", event_type: "runner.fallback.exhausted", timestamp: "2026-07-29T00:00:00Z", payload: { primary: "a", candidates_tried: ["a", "b"], last_reason: "down" } },
    { id: "c", event_type: "model.fallback.exhausted", timestamp: "2026-07-29T00:00:00Z", payload: { preset: "fast", chain: ["x", "y"], last_reason: "quota" } },
    { id: "d", event_type: "model.fallback.attempted", timestamp: "2026-07-29T00:00:00Z", payload: null },
    { id: "e", event_type: "runner.fallback.attempted", timestamp: "2026-07-29T00:00:00Z", payload: { from: "", to: "" } },
    { id: "f", event_type: "runner.fallback.exhausted", timestamp: "2026-07-29T00:00:00Z", payload: { primary: "fallback" } },
    { id: "g", event_type: "model.fallback.exhausted", timestamp: "2026-07-29T00:00:00Z", payload: { preset: "balanced" } },
    { id: "ignored", event_type: "run.started", timestamp: "2026-07-29T00:00:00Z", payload: {} },
  ] }, isLoading: false, error: null });
  rerender(createElement(FallbackTimeline, { runId: "run-1", defaultOpen: true }));
  assert.ok(screen.getByTestId("fallback-timeline-list"));
  assert.equal(screen.getAllByTestId(/fallback-event-/).length, 7);
  assert.ok(screen.getByText("a, b"));
  assert.ok(screen.getByText("x, y"));
});
