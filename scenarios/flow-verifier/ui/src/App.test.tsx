/**
 * App tests — composition smoke + route resolution.
 *
 * Asserts: (1) the operational shell mounts (brand + sidebar nav),
 * (2) each top-level route resolves to its page,
 * (3) navigating between routes does not remount the shell.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "./test-utils";

vi.mock("./api/inventory", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./api/inventory")>();
  return {
    ...actual,
    fetchFlows: vi.fn().mockResolvedValue([]),
    fetchRuns: vi.fn().mockResolvedValue([]),
    fetchFlowDetail: vi.fn().mockResolvedValue({
      flowId: "example.workflow.api",
      contractPath: "api/example/flow/flow.json",
      language: "go",
      schemaVersion: 6,
      initialState: "draft",
      states: [{ id: "draft", quint: "Draft", initial: true }],
      events: [],
      transitions: [],
      traces: [],
      invariants: [],
      report: "",
    }),
    fetchRun: vi.fn().mockResolvedValue({
      id: "abc-123",
      flowId: "example.workflow.api",
      flowPath: "api/example/flow/flow.json",
      root: ".",
      mode: "check",
      status: "passed",
      startedAt: "2026-05-10T11:59:58Z",
      finishedAt: "2026-05-10T12:00:00Z",
      durationMs: 2000,
    }),
    verifyFlow: vi.fn().mockResolvedValue({ status: "passed", runs: [] }),
  };
});

vi.mock("./api/health", () => ({
  fetchHealth: vi.fn().mockResolvedValue({ status: "ok", service: "flow-verifier", timestamp: new Date().toISOString() }),
}));

vi.mock("./features/flow-detail/StateGraph", () => ({
  StateGraph: () => <div data-testid="state-graph-stub" />,
}));

import App from "./App";

describe("App composition", () => {
  afterEach(() => {
    cleanup();
  });

  it("mounts the operational shell with brand", async () => {
    renderWithProviders(<App />);
    expect(await screen.findByTestId("app-shell")).toBeInTheDocument();
    expect(screen.getByTestId("app-brand")).toBeInTheDocument();
  });

  it("lands on the Dashboard at /", async () => {
    renderWithProviders(<App />, { routerEntries: ["/"] });
    expect(await screen.findByTestId("dashboard-page", undefined, { timeout: 2000 })).toBeInTheDocument();
  });

  it("resolves /flows to the Inventory page", async () => {
    renderWithProviders(<App />, { routerEntries: ["/flows"] });
    expect(await screen.findByTestId("inventory-page", undefined, { timeout: 2000 })).toBeInTheDocument();
  });

  it("resolves /flows/:flowId to FlowDetailPage and surfaces the flowId", async () => {
    renderWithProviders(<App />, { routerEntries: ["/flows/example.workflow.api"] });
    expect(
      await screen.findByTestId("flow-detail-page", undefined, { timeout: 2000 }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("flow-detail-id")).toHaveTextContent("example.workflow.api");
  });

  it("resolves /runs/:runId to RunDetailPage and surfaces the runId", async () => {
    renderWithProviders(<App />, { routerEntries: ["/runs/abc-123"] });
    expect(
      await screen.findByTestId("run-detail-page", undefined, { timeout: 2000 }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("run-detail-id")).toHaveTextContent("abc-123");
  });

  it("resolves /settings to the Settings page", async () => {
    renderWithProviders(<App />, { routerEntries: ["/settings"] });
    expect(await screen.findByTestId("settings-page", undefined, { timeout: 2000 })).toBeInTheDocument();
  });

  it("resolves an unknown path to the NotFound page", async () => {
    renderWithProviders(<App />, { routerEntries: ["/totally-not-a-route"] });
    expect(await screen.findByTestId("not-found-page", undefined, { timeout: 2000 })).toBeInTheDocument();
  });
});

describe("App nav navigation", () => {
  afterEach(() => {
    cleanup();
  });

  it("clicking the Flows nav link navigates to /flows without remounting the shell", async () => {
    const user = userEvent.setup();
    renderWithProviders(<App />, { routerEntries: ["/"] });
    expect(await screen.findByTestId("dashboard-page", undefined, { timeout: 2000 })).toBeInTheDocument();

    await user.click(screen.getByTestId("nav-flows"));

    await waitFor(() => expect(screen.getByTestId("inventory-page")).toBeInTheDocument());
    expect(screen.queryByTestId("dashboard-page")).not.toBeInTheDocument();
    // Shell + brand persist across navigation.
    expect(screen.getByTestId("app-shell")).toBeInTheDocument();
    expect(screen.getByTestId("app-brand")).toBeInTheDocument();
  });
});
