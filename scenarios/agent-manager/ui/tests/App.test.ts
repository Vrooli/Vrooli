import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement } from "react";
import { afterEach, test, vi } from "vitest";
import App from "../src/App.js";
import { renderWithProviders } from "../src/test-utils/index.js";

const state = vi.hoisted(() => ({ api: { refetch: vi.fn(), createTask: vi.fn(), createRun: vi.fn(), getRun: vi.fn(), getRunEvents: vi.fn() }, websocket: vi.fn(), events: vi.fn() }));
const emptyActions = { runsSnapshotLoaded: vi.fn(), runStatusReceived: vi.fn(), runEventReceived: vi.fn(), taskStatusReceived: vi.fn(), connected: vi.fn(), disconnected: vi.fn(), eventsGapFilled: vi.fn(), clearReconciliationIntent: vi.fn() };

vi.mock("../src/hooks/useApi.js", () => ({
  useHealth: () => ({ data: null, error: null }),
  useProfiles: () => ({ data: [], refetch: state.api.refetch }),
  useTasks: () => ({ data: [], refetch: state.api.refetch, createTask: state.api.createTask }),
  useRuns: () => ({ data: [], refetch: state.api.refetch, createRun: state.api.createRun, getRun: state.api.getRun, getRunEvents: state.api.getRunEvents }),
  useRunStatusCounts: () => ({ refetch: state.api.refetch }),
  useRolePolicyCatalog: () => ({ data: null }),
}));
vi.mock("../src/hooks/useWebSocket.js", () => ({ useWebSocket: (options: unknown) => { state.websocket(options); return { status: "connected", subscribe: vi.fn(), unsubscribe: vi.fn() }; } }));
vi.mock("../src/hooks/useRunEventStore.js", () => ({ useRunEventStore: () => { state.events(); return { state: { runsById: {} }, actions: emptyActions, reconciliationIntents: [] }; } }));
vi.mock("../src/hooks/useViewportSize.js", () => ({ useIsMobile: () => false }));
vi.mock("../src/components/layout/AppHeader.js", () => ({ AppHeader: ({ activeSection, onSectionChange, onStatusClick, onSettingsClick, onQuickRunClick }: any) => createElement("div", null, createElement("span", { "data-testid": "active" }, activeSection), createElement("button", { onClick: () => onSectionChange("health") }, "Health nav"), createElement("button", { onClick: onStatusClick }, "Status"), createElement("button", { onClick: onSettingsClick }, "Settings"), createElement("button", { onClick: onQuickRunClick }, "Quick run")) }));
vi.mock("../src/components/layout/MobileNav.js", () => ({ MobileNav: () => null }));
vi.mock("../src/pages/DashboardPage.js", () => ({ DashboardPage: () => createElement("div", null, "Dashboard page") }));
vi.mock("../src/pages/ProfilesPage.js", () => ({ ProfilesPage: () => createElement("div", null, "Profiles page") }));
vi.mock("../src/pages/TasksPage.js", () => ({ TasksPage: () => createElement("div", null, "Tasks page") }));
vi.mock("../src/pages/RunsPage.js", () => ({ RunsPage: () => createElement("div", null, "Runs page") }));
vi.mock("../src/pages/WorkflowsPage.js", () => ({ WorkflowsPage: () => createElement("div", null, "Workflows page") }));
vi.mock("../src/pages/FindingsPage.js", () => ({ FindingsPage: () => createElement("div", null, "Findings page") }));
vi.mock("../src/features/stats/index.js", () => ({ StatsPage: () => createElement("div", null, "Stats page") }));
vi.mock("../src/features/health/index.js", () => ({ HealthPage: () => createElement("div", null, "Health page") }));
vi.mock("../src/components/dialogs/StatusDialog.js", () => ({ StatusDialog: () => createElement("div", null, "Status dialog") }));
vi.mock("../src/components/dialogs/SettingsDialog.js", () => ({ SettingsDialog: () => createElement("div", null, "Settings dialog") }));
vi.mock("../src/components/QuickRunDialog.js", () => ({ QuickRunDialog: () => createElement("div", null, "Quick run dialog") }));

afterEach(() => vi.clearAllMocks());

test("App routes each primary page and maps the health section to observability", async () => {
  const user = userEvent.setup();
  renderWithProviders(createElement(App), { initialEntries: ["/"] });
  assert.ok(await screen.findByText("Dashboard page"));
  assert.equal(screen.getByTestId("active").textContent, "dashboard");
  await user.click(screen.getByRole("button", { name: "Health nav" }));
  assert.ok(await screen.findByText("Health page"));
  assert.equal(screen.getByTestId("active").textContent, "health");
});

test("App opens the status surface from the header without requiring route data", async () => {
  const user = userEvent.setup();
  renderWithProviders(createElement(App), { initialEntries: ["/runs"] });
  assert.ok(await screen.findByText("Runs page"));
  await user.click(screen.getByRole("button", { name: "Status" }));
  assert.ok(await screen.findByText("Status dialog"));
});
