import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { afterEach, test, vi } from "vitest";
import App from "../src/App.js";
import { renderWithProviders } from "../src/test-utils/index.js";

const state = vi.hoisted(() => ({
  api: {
    refetch: vi.fn(), createTask: vi.fn(), createRun: vi.fn(), getRun: vi.fn(), getRunEvents: vi.fn(),
    health: null as any, profiles: [] as any[], tasks: [] as any[], runs: [] as any[], snapshots: {} as Record<string, unknown>, mobile: false,
  },
  websocket: vi.fn(), events: vi.fn(),
}));
const emptyActions = { runsSnapshotLoaded: vi.fn(), runSnapshotLoaded: vi.fn(), runStatusReceived: vi.fn(), runEventReceived: vi.fn(), taskStatusReceived: vi.fn(), connected: vi.fn(), disconnected: vi.fn(), eventsGapFilled: vi.fn(), clearReconciliationIntent: vi.fn() };

vi.mock("../src/hooks/useApi.js", () => ({
  useHealth: () => ({ data: state.api.health, error: null, refetch: state.api.refetch }),
  useProfiles: () => ({ data: state.api.profiles, refetch: state.api.refetch }),
  useTasks: () => ({ data: state.api.tasks, refetch: state.api.refetch, createTask: state.api.createTask, getTask: vi.fn() }),
  useRuns: () => ({ data: state.api.runs, refetch: state.api.refetch, createRun: state.api.createRun, getRun: state.api.getRun, getRunEvents: state.api.getRunEvents }),
  useRunStatusCounts: () => ({ refetch: state.api.refetch }),
  useRolePolicyCatalog: () => ({ data: null }),
}));
vi.mock("../src/hooks/useWebSocket.js", () => ({ useWebSocket: (options: unknown) => { state.websocket(options); return { status: "connected", subscribe: vi.fn(), unsubscribe: vi.fn() }; } }));
vi.mock("../src/hooks/useRunEventStore.js", () => ({ useRunEventStore: () => { state.events(); return { state: { runsById: state.api.snapshots }, actions: emptyActions, reconciliationIntents: [] }; } }));
vi.mock("../src/hooks/useViewportSize.js", () => ({ useIsMobile: () => state.api.mobile }));
vi.mock("../src/components/layout/AppHeader.js", () => ({ AppHeader: ({ activeSection, onSectionChange, onStatusClick, onSettingsClick, onQuickRunClick }: any) => createElement("div", null, createElement("span", { "data-testid": "active" }, activeSection), ...["dashboard", "profiles", "tasks", "runs", "workflows", "stats", "health"].map((section) => createElement("button", { key: section, onClick: () => onSectionChange(section) }, `${section} nav`)), createElement("button", { onClick: onStatusClick }, "Status"), createElement("button", { onClick: onSettingsClick }, "Settings"), createElement("button", { onClick: onQuickRunClick }, "Quick run")) }));
vi.mock("../src/components/layout/MobileNav.js", () => ({ MobileNav: ({ activeSection, onSectionChange }: any) => createElement("button", { onClick: () => onSectionChange("runs") }, `mobile ${activeSection}`) }));
vi.mock("../src/pages/DashboardPage.js", () => ({ DashboardPage: ({ onRefresh, onNavigateToRun, runs }: any) => createElement("div", null, "Dashboard page", createElement("span", { "data-testid": "dashboard-runs" }, String(runs.length)), createElement("button", { onClick: onRefresh }, "Dashboard refresh"), createElement("button", { onClick: () => onNavigateToRun("run-9", "diff") }, "Dashboard run")) }));
vi.mock("../src/pages/ProfilesPage.js", () => ({ ProfilesPage: () => createElement("div", null, "Profiles page") }));
vi.mock("../src/pages/TasksPage.js", () => ({ TasksPage: () => createElement("div", null, "Tasks page") }));
vi.mock("../src/pages/RunsPage.js", () => ({ RunsPage: () => createElement("div", null, "Runs page") }));
vi.mock("../src/pages/WorkflowsPage.js", () => ({ WorkflowsPage: () => createElement("div", null, "Workflows page") }));
vi.mock("../src/pages/FindingsPage.js", () => ({ FindingsPage: () => createElement("div", null, "Findings page") }));
vi.mock("../src/features/stats/index.js", () => ({ StatsPage: () => createElement("div", null, "Stats page") }));
vi.mock("../src/features/health/index.js", () => ({ HealthPage: () => createElement("div", null, "Health page") }));
vi.mock("../src/components/dialogs/StatusDialog.js", () => ({ StatusDialog: ({ onOpenChange }: any) => createElement("div", null, "Status dialog", createElement("button", { onClick: () => onOpenChange(false) }, "Close status")) }));
vi.mock("../src/components/dialogs/SettingsDialog/index.js", () => ({ SettingsDialog: ({ onOpenChange, onPurgeComplete }: any) => createElement("div", null, "Settings dialog", createElement("button", { onClick: onPurgeComplete }, "Purge complete"), createElement("button", { onClick: () => onOpenChange(false) }, "Close settings")) }));
vi.mock("../src/components/QuickRunDialog.js", () => ({ QuickRunDialog: ({ onOpenChange, onRunCreated, defaultProjectRoot, profiles }: any) => createElement("div", null, "Quick run dialog", createElement("span", { "data-testid": "quick-root" }, `${defaultProjectRoot ?? "none"}:${profiles.length}`), createElement("button", { onClick: () => onRunCreated({ id: "created-run" }) }, "Created run"), createElement("button", { onClick: () => onOpenChange(false) }, "Close quick run")) }));

afterEach(() => {
  vi.clearAllMocks();
  state.api.health = null;
  state.api.profiles = [];
  state.api.tasks = [];
  state.api.runs = [];
  state.api.snapshots = {};
  state.api.mobile = false;
});

test("App routes each primary page and maps the health section to observability", async () => {
  const user = userEvent.setup();
  renderWithProviders(createElement(App), { initialEntries: ["/"] });
  assert.ok(await screen.findByText("Dashboard page"));
  assert.equal(screen.getByTestId("active").textContent, "dashboard");
  await user.click(screen.getByRole("button", { name: "health nav" }));
  assert.ok(await screen.findByText("Health page"));
  assert.equal(screen.getByTestId("active").textContent, "health");
});

test("App maps every header section to its lazy page", async () => {
  const user = userEvent.setup();
  renderWithProviders(createElement(App), { initialEntries: ["/"] });
  for (const [section, page] of [["profiles", "Profiles page"], ["tasks", "Tasks page"], ["runs", "Runs page"], ["workflows", "Workflows page"], ["stats", "Stats page"], ["dashboard", "Dashboard page"]] as const) {
    await user.click(screen.getByRole("button", { name: `${section} nav` }));
    assert.ok(await screen.findByText(page));
    assert.equal(screen.getByTestId("active").textContent, section);
  }
});

test("App opens the status surface from the header without requiring route data", async () => {
  const user = userEvent.setup();
  renderWithProviders(createElement(App), { initialEntries: ["/runs"] });
  assert.ok(await screen.findByText("Runs page"));
  await user.click(screen.getByRole("button", { name: "Status" }));
  assert.ok(await screen.findByText("Status dialog"));
});

test("App controls settings and quick-run lifecycle callbacks and navigates to the created run", async () => {
  const user = userEvent.setup();
  renderWithProviders(createElement(App), { initialEntries: ["/"] });
  await user.click(screen.getByRole("button", { name: "Settings" }));
  assert.ok(await screen.findByText("Settings dialog"));
  await user.click(screen.getByRole("button", { name: "Purge complete" }));
  assert.ok(state.api.refetch.mock.calls.length >= 3);
  await user.click(screen.getByRole("button", { name: "Close settings" }));
  await user.click(screen.getByRole("button", { name: "Quick run" }));
  assert.ok(await screen.findByText("Quick run dialog"));
  await user.click(screen.getByRole("button", { name: "Created run" }));
  assert.ok(await screen.findByText("Runs page"));
  assert.equal(screen.getByTestId("active").textContent, "runs");
});

test("App folds WebSocket status, run, event, task, and workflow lifecycle messages into durable UI state", async () => {
  const workflowListener = vi.fn();
  window.addEventListener("agent-manager:workflow-lifecycle", workflowListener);
  state.api.getRun.mockResolvedValue({ id: "run-1" });
  renderWithProviders(createElement(App), { initialEntries: ["/runs"] });
  const options = state.websocket.mock.calls.at(-1)![0] as any;
  options.onStatusChange("connected");
  options.onStatusChange("error");
  options.onMessage({ type: "run_status", payload: { id: "run-1", taskId: "task-1" } });
  options.onMessage({ type: "run_event", payload: { runId: "run-1" } });
  options.onMessage({ type: "task_status", payload: { id: "task-1" } });
  options.onMessage({ type: "workflow_lifecycle", payload: { workflow: "investigate" } });
  await Promise.resolve();
  assert.equal(emptyActions.connected.mock.calls.length, 1);
  assert.equal(emptyActions.disconnected.mock.calls.length, 1);
  assert.ok(emptyActions.runStatusReceived.mock.calls.length >= 1);
  assert.ok(emptyActions.runEventReceived.mock.calls.length >= 1);
  assert.ok(emptyActions.taskStatusReceived.mock.calls.length >= 1);
  assert.deepEqual(workflowListener.mock.calls[0]![0].detail, { workflow: "investigate" });
  window.removeEventListener("agent-manager:workflow-lifecycle", workflowListener);
});

test("App refreshes dashboard state, merges run snapshots, and handles dashboard WebSocket updates", async () => {
  const user = userEvent.setup();
  state.api.runs = [{ id: "run-1", title: "before" }];
  state.api.snapshots = { "run-1": { title: "after" } };
  state.api.getRun.mockResolvedValue({ id: "run-1" });
  renderWithProviders(createElement(App), { initialEntries: ["/"] });
  await screen.findByText("Dashboard page");
  assert.equal(screen.getByTestId("dashboard-runs").textContent, "1");
  await user.click(screen.getByRole("button", { name: "Dashboard refresh" }));
  assert.ok(state.api.refetch.mock.calls.length >= 3);
  const options = state.websocket.mock.calls.at(-1)![0] as any;
  options.onMessage({ type: "run_status", payload: { id: "run-1" } });
  options.onMessage({ type: "run_status", payload: {} });
  options.onMessage({ type: "task_status", payload: { id: "task-1" } });
  options.onStatusChange("disconnected");
  options.onStatusChange("idle");
  await Promise.resolve();
  assert.ok(emptyActions.runSnapshotLoaded.mock.calls.length >= 1);
  assert.ok(emptyActions.disconnected.mock.calls.length >= 1);
  await user.click(screen.getByRole("button", { name: "Dashboard run" }));
  assert.ok(await screen.findByText("Runs page"));
});

test("App supplies quick-run route data, task refreshes, mobile navigation, and unknown-route recovery", async () => {
  const user = userEvent.setup();
  state.api.mobile = true;
  state.api.profiles = [{ id: "profile-1" }];
  state.api.health = { metrics: { default_project_root: { kind: { case: "stringValue", value: "/workspace/project" } } } };
  renderWithProviders(createElement(App), { initialEntries: ["/tasks"] });
  assert.ok(await screen.findByText("Tasks page"));
  assert.ok(screen.getByRole("button", { name: "mobile tasks" }));
  await user.click(screen.getByRole("button", { name: "Quick run" }));
  assert.equal((await screen.findByTestId("quick-root")).textContent, "/workspace/project:1");
  await user.click(screen.getByRole("button", { name: "Created run" }));
  assert.ok(await screen.findByText("Runs page"));
  await user.click(screen.getByRole("button", { name: "mobile runs" }));
  assert.ok(await screen.findByText("Runs page"));
});

test("App redirects unrecognized locations to the dashboard", async () => {
  renderWithProviders(createElement(App), { initialEntries: ["/not-a-route"] });
  assert.ok(await screen.findByText("Dashboard page"));
});
