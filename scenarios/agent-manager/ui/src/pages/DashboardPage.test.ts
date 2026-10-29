import assert from "node:assert/strict";
import { fireEvent, screen } from "@testing-library/react";
import { createElement } from "react";
import { test, vi } from "vitest";
import { DashboardPage } from "./DashboardPage.js";
import { renderWithProviders } from "../test-utils/index.js";
import { HealthStatus, RunStatus } from "../types.js";
import { makeRun } from "../test-utils/fixtures/runs.js";
import { getErrorNotice } from "../test-utils/feedback.js";

const usage = vi.hoisted(() => ({
  forecast: { data: undefined as unknown },
  recommendations: { data: undefined as unknown },
}));
vi.mock("../features/usage/api.js", () => ({
  useUsageForecast: () => usage.forecast,
  useUsageRecommendations: () => usage.recommendations,
}));
vi.mock("../features/usage/DashboardUsageStrip.js", () => ({ DashboardUsageStrip: () => createElement("section", { "data-testid": "dashboard-usage-strip" }) }));
const stringValue = (value: string) => ({ kind: { case: "stringValue", value } });
const objectValue = (fields: Record<string, unknown>) => ({ kind: { case: "objectValue", value: { fields } } });

test("DashboardPage presents run counts, review navigation, run titles, and refresh", () => {
  const refresh = vi.fn();
  const navigate = vi.fn();
  renderWithProviders(createElement(DashboardPage, {
    health: null,
    counts: { active: 1, review: 1, windowTotal: 2, operatorSessions: 0 },
    runs: [makeRun({ id: "review", label: "Review task", status: RunStatus.NEEDS_REVIEW, changedFiles: 2 }), makeRun({ id: "running", label: "Running task", status: RunStatus.RUNNING })],
    onRefresh: refresh, onNavigateToRun: navigate,
  }));
  assert.ok(screen.getByText("Awaiting review (1)"));
  fireEvent.click(screen.getByText("Review task").closest("button") as Element);
  assert.deepEqual(navigate.mock.calls[0], ["review", "diff"]);
  assert.ok(screen.getByText("Running task"));
  fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
  assert.equal(refresh.mock.calls.length, 1);
});

test("DashboardPage labels its tiles, links each to its runs, and groups thousands", () => {
  const { rerender } = renderWithProviders(createElement(DashboardPage, {
    health: null,
    // 21 running and 83 awaiting approval across the fleet, while the dashboard page loaded only 2 runs.
    counts: { active: 21, review: 83, windowTotal: 11417, operatorSessions: 0 },
    runs: [makeRun({ id: "r1", status: RunStatus.RUNNING }), makeRun({ id: "rv", status: RunStatus.NEEDS_REVIEW })],
    onRefresh: vi.fn(),
  }));
  const tile = (title: string) => screen.getByText(title).closest("[data-testid='dashboard-count-tile']") as HTMLElement;
  assert.match(tile("Running now").textContent ?? "", /21/);
  assert.match(tile("Waiting for your approval").textContent ?? "", /83/);
  assert.match(tile("Agent Manager runs · 7 days").textContent ?? "", /11,417/);
  assert.equal(tile("Running now").querySelector("a")?.getAttribute("href"), `/runs?status=${RunStatus.RUNNING}`);
  assert.equal(tile("Waiting for your approval").querySelector("a")?.getAttribute("href"), `/runs?status=${RunStatus.NEEDS_REVIEW}`);
  assert.ok(screen.getByTestId("dashboard-usage-strip"), "the usage strip sits above the tiles");
  assert.ok(screen.getByRole("button", { name: "Refresh" }).className.includes("min-h-11"), "Refresh keeps a 44 px target on a phone");
  rerender(createElement(DashboardPage, { health: null, counts: null, runs: [], onRefresh: vi.fn() }));
  for (const title of ["Running now", "Waiting for your approval", "Agent Manager runs · 7 days"]) assert.match(tile(title).textContent ?? "", /—/);
});

test("DashboardPage names the biggest lever under a window that will run out and links to that window", () => {
  usage.forecast.data = { windows: [{ provider: "claude", pool: "", window: "seven_day", status: "will_exhaust" }] };
  usage.recommendations.data = { groups: [
    { provider: "codex", pool: "", window: "seven_day", targetLabel: "other", levers: [{ action: "ignored", savedPoints: 90 }] },
    { provider: "claude", pool: "", window: "seven_day", targetLabel: "Agent manager token usage analysis", levers: [{ action: "brief sub-agents", savedPoints: 46.2 }, { action: "trim context", savedPoints: 3 }] },
  ] };
  renderWithProviders(createElement(DashboardPage, { health: null, runs: [], onRefresh: vi.fn() }));
  const lever = screen.getByTestId("dashboard-usage-lever");
  assert.match(lever.textContent ?? "", /Biggest lever: brief sub-agents in "Agent manager token usage analysis", ≈ −46 pts\/wk\./);
  assert.equal(screen.getByRole("link", { name: "What to do →" }).getAttribute("href"), "/usage?window=claude%2F%2Fseven_day");
  usage.forecast.data = { windows: [{ provider: "claude", pool: "", window: "seven_day", status: "on_track" }] };
  renderWithProviders(createElement(DashboardPage, { health: null, runs: [], onRefresh: vi.fn() }));
  assert.equal(screen.getAllByTestId("dashboard-usage-lever").length, 1, "an on-track window shows no lever line");
});

test("DashboardPage does not report a zero count for a pending run without diff evidence", async () => {
  renderWithProviders(createElement(DashboardPage, {
    health: null,
    counts: { active: 0, review: 1, windowTotal: 1, operatorSessions: 0 },
    runs: [makeRun({ id: "unknown-review", label: "Unknown diff review", status: RunStatus.NEEDS_REVIEW, changedFiles: 0 })],
    onRefresh: vi.fn(),
  }));
  assert.ok(await screen.findByText(/File count unavailable/));
  assert.equal(screen.queryByText("0 files changed"), null);
});

test("DashboardPage summarises health on one line that links to the Health page, with no probing", () => {
  renderWithProviders(createElement(DashboardPage, {
    health: { status: HealthStatus.HEALTHY, dependencies: { sandbox: objectValue({ status: stringValue("healthy") }), runner_codex: objectValue({ status: stringValue("healthy") }) } } as never,
    runs: [], onRefresh: vi.fn(),
  }));
  assert.ok(screen.getByText("All systems operational"));
  assert.equal(screen.getByRole("link", { name: "Health →" }).getAttribute("href"), "/observability");
  assert.equal(screen.queryByRole("button", { name: /probe/i }), null);
});

test("DashboardPage counts unhealthy dependencies in the summary", () => {
  renderWithProviders(createElement(DashboardPage, {
    health: { status: HealthStatus.HEALTHY, dependencies: {
      sandbox: objectValue({ status: stringValue("unhealthy"), error: stringValue("sandbox stopped") }),
      runner_codex: objectValue({ status: stringValue("unhealthy"), error: stringValue("runner unavailable") }),
    } } as never,
    runs: [], onRefresh: vi.fn(),
  }));
  assert.ok(screen.getByText("2 checks need attention"));
  assert.ok(screen.getByText(/No recent Agent Manager runs/));
});

test("DashboardPage distinguishes loading, unreachable, and empty activity instead of claiming no runs", () => {
  const retry = vi.fn();
  const { rerender } = renderWithProviders(createElement(DashboardPage, { health: null, runs: [], runsLoading: true, onRefresh: retry }));
  assert.ok(screen.getByText("Checking health…"));
  assert.ok(screen.getByText(/Loading recent runs/));
  assert.equal(screen.queryByText(/No recent Agent Manager runs/), null);
  rerender(createElement(DashboardPage, { health: null, healthError: "Request failed: 503", runs: [], runsError: "Request failed: 503", onRefresh: retry }));
  const activity = getErrorNotice("Recent runs", "Request failed: 503");
  assert.match(activity.textContent ?? "", /unreachable/);
  getErrorNotice("Health");
  assert.equal(screen.queryByText(/No recent Agent Manager runs/), null);
  fireEvent.click(activity.querySelector("button") as Element);
  assert.equal(retry.mock.calls.length, 1);
});

test("DashboardPage reports a degraded API that answers with its reason, not as unreachable", () => {
  renderWithProviders(createElement(DashboardPage, {
    health: { status: HealthStatus.DEGRADED, service: "agent-manager", dependencies: {}, metrics: {} } as never,
    healthReason: "startup degraded: workflow_accounting: context deadline exceeded",
    runs: [], onRefresh: vi.fn(),
  }));
  assert.match(screen.getByTestId("dashboard-health-summary").textContent ?? "", /Agent Manager is degraded\. startup degraded: workflow_accounting: context deadline exceeded/);
  assert.equal(screen.queryByText(/unreachable/i), null);
});

test("DashboardPage lists the runs it is given as activity, titled from the run, and says how many operator sessions it leaves out", () => {
  const navigate = vi.fn();
  renderWithProviders(createElement(DashboardPage, {
    health: null,
    counts: { active: 0, review: 0, windowTotal: 1214, operatorSessions: 412 },
    runs: [
      makeRun({ id: "mine", label: "Fix the parser", labelSource: "user", status: RunStatus.FAILED }),
      makeRun({ id: "parked", status: RunStatus.PARKED }),
    ],
    onRefresh: vi.fn(), onNavigateToRun: navigate,
  }));
  assert.ok(screen.getByText("Fix the parser"));
  assert.ok(screen.getByText("Run parked"), "a run with no label falls back to its short id");
  assert.ok(screen.getByText("Failed"));
  assert.ok(screen.getByText("Parked"));
  assert.ok(screen.getByText("+ 412 operator sessions"));
  assert.equal(screen.getByText("Agent Manager runs · 7 days").closest("[data-testid='dashboard-count-tile']")?.querySelector("a")?.getAttribute("href"), "/runs?origin=agent_manager");
  fireEvent.click(screen.getByText("Parked").closest("button") as Element);
  assert.deepEqual(navigate.mock.calls, [["parked"]]);
});
