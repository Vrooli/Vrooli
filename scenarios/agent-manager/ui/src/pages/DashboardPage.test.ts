import assert from "node:assert/strict";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement } from "react";
import { test, vi } from "vitest";
import { DashboardPage } from "./DashboardPage.js";
import { renderWithProviders } from "../test-utils/index.js";
import { HealthStatus, RunStatus } from "../types.js";
import { makeRun } from "../test-utils/fixtures/runs.js";
import { getErrorNotice } from "../test-utils/feedback.js";

const state = vi.hoisted(() => ({ probeRunner: vi.fn() }));
vi.mock("../hooks/useApi.js", () => ({ probeRunner: state.probeRunner }));
vi.mock("../features/usage/DashboardUsageStrip.js", () => ({ DashboardUsageStrip: () => createElement("section", { "data-testid": "dashboard-usage-strip" }) }));
const stringValue = (value: string) => ({ kind: { case: "stringValue", value } });
const objectValue = (fields: Record<string, unknown>) => ({ kind: { case: "objectValue", value: { fields } } });

test("DashboardPage presents run counts, review navigation, run titles, and refresh", () => {
  const refresh = vi.fn();
  const navigate = vi.fn();
  renderWithProviders(createElement(DashboardPage, {
    health: null,
    counts: { active: 1, review: 1, windowTotal: 2 },
    runs: [makeRun({ id: "review", label: "Review task", status: RunStatus.NEEDS_REVIEW, changedFiles: 2 }), makeRun({ id: "running", label: "Running task", status: RunStatus.RUNNING })],
    onRefresh: refresh, onNavigateToRun: navigate,
  }));
  assert.ok(screen.getByText("Awaiting Review (1)"));
  const reviewLabel = screen.getByText("Review task");
  assert.ok(screen.getByText("Running task"));
  fireEvent.click(reviewLabel.closest('[role="button"]') as Element);
  assert.deepEqual(navigate.mock.calls[0], ["review", "diff"]);
  fireEvent.click(screen.getByTitle("Refresh"));
  assert.equal(refresh.mock.calls.length, 1);
});

test("DashboardPage shows API-counted totals, not the length of the loaded run page", () => {
  const { rerender } = renderWithProviders(createElement(DashboardPage, {
    health: null,
    // 19 running and 83 awaiting review across the fleet, while the dashboard page loaded only 2 runs.
    counts: { active: 19, review: 83, windowTotal: 642 },
    runs: [makeRun({ id: "r1", status: RunStatus.RUNNING }), makeRun({ id: "rv", status: RunStatus.NEEDS_REVIEW })],
    onRefresh: vi.fn(),
  }));
  const card = (title: string) => screen.getByText(title).parentElement?.textContent ?? "";
  assert.match(card("Active"), /^19/);
  assert.match(card("Review"), /^83/);
  assert.match(card("Runs · 7d"), /^642/);
  assert.ok(screen.getByTestId("dashboard-usage-strip"), "the usage strip sits above the counters");
  rerender(createElement(DashboardPage, { health: null, counts: null, runs: [makeRun({ id: "r1", status: RunStatus.RUNNING })], onRefresh: vi.fn() }));
  for (const title of ["Active", "Review", "Runs · 7d"]) assert.match(card(title), /^—/);
});

test("DashboardPage does not report a zero count for a pending run without diff evidence", async () => {
  renderWithProviders(createElement(DashboardPage, {
    health: null,
    counts: { active: 0, review: 1, windowTotal: 1 },
    runs: [makeRun({ id: "unknown-review", label: "Unknown diff review", status: RunStatus.NEEDS_REVIEW, changedFiles: 0 })],
    onRefresh: vi.fn(),
  }));
  assert.ok(await screen.findByText(/File count unavailable/));
  assert.equal(screen.queryByText("0 files changed"), null);
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
    runs: [], onRefresh: vi.fn(),
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

test("DashboardPage collapses a healthy summary, titles activity from the run, and supports keyboard run navigation", () => {
  const navigate = vi.fn();
  renderWithProviders(createElement(DashboardPage, {
    health: { dependencies: { sandbox: objectValue({ status: stringValue("healthy") }), runner_codex: objectValue({ status: stringValue("healthy") }) } } as never,
    counts: { active: 0, review: 0, windowTotal: 1 }, runs: [makeRun({ id: "complete", label: "Completed evidence run", status: RunStatus.COMPLETE })],
    onRefresh: vi.fn(), onNavigateToRun: navigate,
  }));
  assert.ok(screen.getByText("Completed evidence run"));
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
    runs: [], onRefresh: vi.fn(),
  }));

  assert.ok(screen.getByText(/No runs yet/));
  assert.ok(screen.getByText("runner unavailable"));
  fireEvent.click(screen.getByText("System Health"));
  assert.ok(screen.getByText("2 issues detected"));
});

test("DashboardPage distinguishes loading, unreachable, and empty activity instead of claiming no runs", () => {
  const retry = vi.fn();
  const { rerender } = renderWithProviders(createElement(DashboardPage, { health: null, runs: [], runsLoading: true, onRefresh: retry }));
  assert.ok(screen.getByText("Loading health status…"));
  assert.ok(screen.getByText(/Loading recent runs/));
  assert.equal(screen.queryByText(/No runs yet/), null);

  rerender(createElement(DashboardPage, { health: null, healthError: "Request failed: 503", runs: [], runsError: "Request failed: 503", onRefresh: retry }));
  const activity = getErrorNotice("Recent runs", "Request failed: 503");
  assert.match(activity.textContent ?? "", /unreachable/);
  getErrorNotice("Health");
  assert.equal(screen.queryByText(/No runs yet/), null);
  assert.equal(screen.queryByText("Workspace Sandbox"), null, "no dependency rows are invented without a health reading");
  fireEvent.click(activity.querySelector("button") as Element);
  assert.equal(retry.mock.calls.length, 1);
});

test("DashboardPage reports a degraded API that answers with its reason, not as unreachable", () => {
  renderWithProviders(createElement(DashboardPage, {
    health: { status: HealthStatus.DEGRADED, service: "agent-manager", dependencies: {}, metrics: {} } as never,
    healthReason: "startup degraded: workflow_accounting: context deadline exceeded",
    runs: [], onRefresh: vi.fn(),
  }));
  const notice = screen.getByTestId("health-degraded");
  assert.match(notice.textContent ?? "", /Agent Manager is degraded\. startup degraded: workflow_accounting: context deadline exceeded/);
  assert.equal(screen.queryByText(/unreachable/i), null);
});

test("DashboardPage names attached and workflow runs from run fields, keeping machine ids secondary", () => {
  const navigate = vi.fn();
  renderWithProviders(createElement(DashboardPage, {
    health: { dependencies: { sandbox: objectValue({ status: stringValue("healthy") }) } } as never,
    runs: [
      makeRun({ id: "failed", taskId: "", label: "codex session codex-dbbbdcd1a36820581225517b14d85532", labelSource: "harness", harnessKind: "codex", harnessSessionId: "codex-dbbbdcd1a36820581225517b14d85532", status: RunStatus.FAILED }),
      makeRun({ id: "cancelled", label: "Workflow e6b4c7c2-da57-416c-b2c2-8c0b5e942874 node draft", status: RunStatus.CANCELLED }),
      makeRun({ id: "parked", status: RunStatus.PARKED }),
    ],
    onRefresh: vi.fn(), onNavigateToRun: navigate,
  }));

  assert.equal(screen.queryByText(/Loading task/), null);
  assert.ok(screen.getByText("Codex session"));
  assert.ok(screen.getByText("dbbbdcd1"));
  assert.ok(screen.getByText("Workflow step · draft"));
  assert.ok(screen.getByText("e6b4c7c2"));
  assert.ok(screen.getByText("Run parked"), "a run with no label falls back to its short id");
  assert.ok(screen.getByText("Failed"));
  assert.ok(screen.getByText("Cancelled"));
  fireEvent.keyDown(screen.getByText("Parked").closest('[role="button"]') as Element, { key: "Enter" });
  assert.deepEqual(navigate.mock.calls, [["parked"]]);
});
