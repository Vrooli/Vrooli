import assert from "node:assert/strict";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { test, vi } from "vitest";
import { DashboardPage } from "../../src/pages/DashboardPage.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import { RunStatus } from "../../src/types.js";
import { makeRun } from "../testutil/runs.js";

const state = vi.hoisted(() => ({ probeRunner: vi.fn() }));
vi.mock("../../src/hooks/useApi.js", () => ({ probeRunner: state.probeRunner }));
const stringValue = (value: string) => ({ kind: { case: "stringValue", value } });
const objectValue = (fields: Record<string, unknown>) => ({ kind: { case: "objectValue", value: { fields } } });

test("DashboardPage presents run counts, review navigation, task names, and refresh", async () => {
  const refresh = vi.fn();
  const navigate = vi.fn();
  const getTask = vi.fn(async () => ({ id: "task-1", title: "Review task" }));
  renderWithProviders(createElement(DashboardPage, {
    health: null,
    statusCounts: { running: 1, needsReview: 1, total: 2 },
    runs: [makeRun({ id: "review", taskId: "task-1", status: RunStatus.NEEDS_REVIEW, changedFiles: 2 }), makeRun({ id: "running", status: RunStatus.RUNNING })],
    onRefresh: refresh, onNavigateToRun: navigate, onGetTask: getTask,
  }));
  assert.ok(screen.getByText("Awaiting Review (1)"));
  const taskLabels = await screen.findAllByText("Review task");
  assert.equal(taskLabels.length, 2);
  fireEvent.click(taskLabels[0]?.closest('[role="button"]') as Element);
  assert.deepEqual(navigate.mock.calls[0], ["review", "diff"]);
  fireEvent.click(screen.getByTitle("Refresh"));
  assert.equal(refresh.mock.calls.length, 1);
});

test("DashboardPage shows dependency failures and runs a copyable runner probe", async () => {
  state.probeRunner.mockResolvedValue({ success: true, latencyMs: 17n, details: { response: "pong" } });
  const clipboard = { writeText: vi.fn().mockResolvedValue(undefined) };
  Object.defineProperty(navigator, "clipboard", { configurable: true, value: clipboard });
  renderWithProviders(createElement(DashboardPage, {
    health: { dependencies: {
      sandbox: objectValue({ status: stringValue("unhealthy"), error: stringValue("sandbox unreachable") }),
      runner_codex: objectValue({ status: stringValue("healthy") }),
    } } as never,
    runs: [], onRefresh: vi.fn(), onGetTask: vi.fn(),
  }));
  assert.ok(screen.getByText("Workspace Sandbox"));
  assert.ok(screen.getByText("sandbox unreachable"));
  fireEvent.click(screen.getByRole("button", { name: "Copy error message" }));
  await waitFor(() => assert.equal(clipboard.writeText.mock.calls[0]?.[0], "sandbox unreachable"));
  fireEvent.click(screen.getByRole("button", { name: /probe/i }));
  await waitFor(() => assert.ok(screen.getByText("✓ Probe successful")));
  fireEvent.click(screen.getByRole("button", { name: "Copy probe result" }));
  await waitFor(() => assert.match(clipboard.writeText.mock.calls[1]?.[0] ?? "", /pong/));
  fireEvent.click(screen.getByRole("button", { name: "Dismiss probe result" }));
  assert.equal(screen.queryByText("✓ Probe successful"), null);
});

test("DashboardPage collapses a healthy summary, loads activity titles, and supports keyboard run navigation", async () => {
  const navigate = vi.fn();
  renderWithProviders(createElement(DashboardPage, {
    health: { dependencies: { sandbox: objectValue({ status: stringValue("healthy") }), runner_codex: objectValue({ status: stringValue("healthy") }) } } as never,
    statusCounts: { running: 0, needsReview: 0, total: 1 }, runs: [makeRun({ id: "complete", taskId: "task-complete", status: RunStatus.COMPLETE })],
    onRefresh: vi.fn(), onGetTask: vi.fn(async () => ({ id: "task-complete", title: "Completed evidence run" })), onNavigateToRun: navigate,
  }));
  assert.ok(await screen.findByText("Completed evidence run"));
  fireEvent.keyDown(screen.getByText("Completed evidence run").closest('[role="button"]') as Element, { key: "Enter" });
  assert.deepEqual(navigate.mock.calls, [["complete"]]);
  fireEvent.click(screen.getByText("System Health"));
  assert.ok(screen.getByText("All systems operational"));
});

test("DashboardPage reports an unsuccessful runner probe without losing its diagnostic", async () => {
  state.probeRunner.mockRejectedValue(new Error("runner endpoint timed out"));
  renderWithProviders(createElement(DashboardPage, {
    health: { dependencies: { sandbox: objectValue({ status: stringValue("healthy") }), runner_codex: objectValue({ status: stringValue("unhealthy") }) } } as never,
    runs: [], onRefresh: vi.fn(),
  }));
  fireEvent.click(screen.getByRole("button", { name: /probe/i }));
  assert.ok(await screen.findByText("✗ Probe failed"));
  assert.ok(screen.getByText("runner endpoint timed out"));
});

test("DashboardPage gives a useful empty-state and collapsed unhealthy dependency summary", () => {
  renderWithProviders(createElement(DashboardPage, {
    health: { dependencies: {
      sandbox: objectValue({ status: stringValue("unhealthy"), error: stringValue("sandbox stopped") }),
      runner_custom_worker: objectValue({ status: stringValue("unhealthy"), error: stringValue("runner unavailable") }),
    } } as never,
    runs: [], onRefresh: vi.fn(), onGetTask: vi.fn(),
  }));

  assert.ok(screen.getByText("No runs yet"));
  assert.ok(screen.getByText("runner unavailable"));
  fireEvent.click(screen.getByText("System Health"));
  assert.ok(screen.getByText("2 issues detected"));
});

test("DashboardPage falls back safely while health is loading and handles a pending activity run", async () => {
  const navigate = vi.fn();
  renderWithProviders(createElement(DashboardPage, {
    health: null,
    runs: [makeRun({ id: "pending", taskId: "pending-task", status: RunStatus.PENDING })],
    onRefresh: vi.fn(),
    onGetTask: vi.fn(async () => ({ id: "pending-task", title: "Queued investigation" })),
    onNavigateToRun: navigate,
  }));

  assert.ok(screen.getByText("Loading health status..."));
  const activity = await screen.findByText("Queued investigation");
  fireEvent.click(activity.closest('[role="button"]') as Element);
  assert.deepEqual(navigate.mock.calls, [["pending"]]);
});

test("DashboardPage surfaces terminal and parked activity states while task lookups degrade gracefully", async () => {
  const navigate = vi.fn();
  const failedLookup = vi.fn(async () => { throw new Error("task catalog temporarily unavailable"); });
  renderWithProviders(createElement(DashboardPage, {
    health: { dependencies: { sandbox: objectValue({ status: stringValue("healthy") }) } } as never,
    runs: [
      makeRun({ id: "failed", taskId: "failed-task", status: RunStatus.FAILED }),
      makeRun({ id: "cancelled", taskId: "cancelled-task", status: RunStatus.CANCELLED }),
      makeRun({ id: "parked", taskId: "parked-task", status: RunStatus.PARKED }),
    ],
    onRefresh: vi.fn(), onGetTask: failedLookup, onNavigateToRun: navigate,
  }));

  const loadingLabels = await screen.findAllByText("Loading task...");
  assert.equal(loadingLabels.length, 3);
  await waitFor(() => assert.equal(failedLookup.mock.calls.length, 3));
  await waitFor(() => assert.equal(screen.getAllByText("Unknown Task").length, 3));
  assert.ok(screen.getByText("Failed"));
  assert.ok(screen.getByText("Cancelled"));
  assert.ok(screen.getByText("Parked"));
  fireEvent.keyDown(screen.getByText("Parked").closest('[role="button"]') as Element, { key: "Enter" });
  assert.deepEqual(navigate.mock.calls, [["parked"]]);
});
