import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../test-utils";
import type { SuiteExecutionResult } from "../../lib/api";
import { initialExecutionForm, useUIStore } from "../../stores/uiStore";
import { HistoryTab } from "./HistoryTab";

const fetchMock = vi.fn();

function execution(overrides: Partial<SuiteExecutionResult>): SuiteExecutionResult {
  return {
    executionId: "run-failed",
    scenarioName: "api-health",
    startedAt: "2026-11-05T12:00:00Z",
    completedAt: "2026-11-05T12:00:13Z",
    success: false,
    phases: [{ name: "unit", status: "failed", durationSeconds: 13, logPath: "", error: "assertion failed" }],
    phaseSummary: { total: 1, passed: 0, failed: 1, durationSeconds: 13, observationCount: 0 },
    ...overrides,
  };
}

function renderHistoryTab() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <HistoryTab />
    </QueryClientProvider>,
  );
}

describe("HistoryTab", () => {
  beforeEach(() => {
    fetchMock.mockReset();
    vi.stubGlobal("fetch", fetchMock);
    window.history.replaceState(null, "", "/");
    useUIStore.setState({
      activeTab: "dashboard",
      selectedScenario: null,
      focusScenario: "",
      executionForm: { ...initialExecutionForm },
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads, filters, and reruns a failed scenario from shared history", async () => {
    const user = userEvent.setup();
    const runs = [
      execution({
        executionId: "run-failed",
        preset: undefined,
      }),
      execution({
        executionId: "run-payments",
        scenarioName: "payments",
      }),
      execution({
        executionId: "run-passed",
        scenarioName: "docs-check",
        success: true,
        preset: "smoke",
        phases: [{ name: "docs", status: "passed", durationSeconds: 4, logPath: "" }],
        phaseSummary: { total: 1, passed: 1, failed: 0, durationSeconds: 4, observationCount: 0 },
      }),
    ];
    let resolveHistory: ((response: Response) => void) | undefined;
    fetchMock.mockImplementation(
      () => new Promise<Response>((resolve) => { resolveHistory = resolve; }),
    );

    renderHistoryTab();

    expect(screen.getByText("Loading test history...")).toBeInTheDocument();
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    const firstRequest = fetchMock.mock.calls[0];
    expect(String(firstRequest?.[0])).toContain("limit=20");
    resolveHistory?.(new Response(JSON.stringify({ items: runs, count: runs.length }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }));

    expect(await screen.findByRole("heading", { name: "api-health" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "docs-check" })).toBeInTheDocument();
    expect(screen.getAllByText("assertion failed")).toHaveLength(2);

    await user.click(screen.getByRole("button", { name: "failed" }));
    expect(screen.getByRole("heading", { name: "api-health" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "payments" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "docs-check" })).not.toBeInTheDocument();

    await user.type(screen.getByPlaceholderText("Search by scenario..."), "API");
    expect(screen.getByRole("heading", { name: "api-health" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "payments" })).not.toBeInTheDocument();
    expect(screen.queryByText("No runs match \"API\"")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Rerun tests" }));

    const state = useUIStore.getState();
    expect(state.focusScenario).toBe("api-health");
    expect(state.executionForm).toEqual({ scenarioName: "api-health", preset: "quick", failFast: true });
    expect(state.activeTab).toBe("runs");
    expect(state.selectedScenario).toBe("api-health");
  });
});
