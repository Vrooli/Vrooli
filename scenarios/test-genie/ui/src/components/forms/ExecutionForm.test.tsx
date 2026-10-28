import { beforeEach, describe, expect, it, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders as render } from "../../test-utils";
import { ExecutionForm } from "./ExecutionForm";
import { initialExecutionForm, useUIStore } from "../../stores/uiStore";

const fetchMock = vi.fn();

vi.stubGlobal("fetch", fetchMock);

function renderExecutionForm(props?: { scenarioName?: string }) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <ExecutionForm
      scenarioOptions={[]}
      datalistId="execution-scenarios"
      scenarioName={props?.scenarioName}
    />,
    {
      withoutQueryClient: true,
      extraProviders: (children) => <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>,
    },
  );
}

describe("ExecutionForm", () => {
  beforeEach(() => {
    fetchMock.mockReset();
    useUIStore.setState({
      executionForm: { ...initialExecutionForm },
      executionFeedback: null
    });
  });

  it("shows execution plan estimates and timeout budgets", async () => {
    fetchMock.mockResolvedValue(
      new Response(
        JSON.stringify({
          scenarioName: "demo",
          presetUsed: "quick",
          phases: [
            {
              name: "unit",
              description: "Runs unit tests",
              optional: false,
              estimatedDurationSeconds: 42,
              timeoutSeconds: 900,
              estimateSource: "scenario_history",
              estimateConfidence: "medium",
              estimateSampleSize: 6
            },
            {
              name: "playbooks",
              description: "Runs BAS playbooks",
              optional: true,
              estimatedDurationSeconds: 900,
              timeoutSeconds: 900,
              estimateSource: "timeout_fallback",
              estimateConfidence: "low",
              estimateSampleSize: 0
            }
          ],
          summary: {
            phaseCount: 2,
            estimatedDurationSeconds: 942,
            timeoutSeconds: 1800
          },
          warnings: [
            "Phase 'playbooks' is globally disabled and was skipped by default."
          ]
        }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" }
        }
      )
    );

    renderExecutionForm({ scenarioName: "demo" });

    await waitFor(() => {
      expect(screen.getByText("Execution preview")).toBeInTheDocument();
      expect(screen.getByText("15m 42s")).toBeInTheDocument();
    });

    expect(screen.getAllByText("Estimate:").length).toBeGreaterThan(0);
    expect(screen.getByText("Timeout budget:")).toBeInTheDocument();
    expect(screen.getByText("30m 0s")).toBeInTheDocument();
    expect(screen.getByText("Runs unit tests")).toBeInTheDocument();
    expect(screen.getByText("Timeout fallback")).toBeInTheDocument();
    expect(screen.getByText("medium confidence from 6 runs")).toBeInTheDocument();
    expect(screen.getByText("Phase 'playbooks' is globally disabled and was skipped by default.")).toBeInTheDocument();
  });

  it("submits the selected scenario, preset, and fail-fast option to the execution stream", async () => {
    const user = userEvent.setup();
    fetchMock.mockImplementation(async (url: RequestInfo | URL, init?: RequestInit) => {
      const requestUrl = String(url);
      if (requestUrl.includes("/executions/plan")) {
        return new Response(JSON.stringify({
          scenarioName: "demo",
          presetUsed: "smoke",
          phases: [],
          summary: { phaseCount: 0, estimatedDurationSeconds: 0, timeoutSeconds: 0 },
          warnings: [],
        }), { status: 200, headers: { "Content-Type": "application/json" } });
      }
      if (requestUrl.includes("/executions/stream")) {
        return new Response(
          `event: complete\ndata: ${JSON.stringify({
            success: true,
            presetUsed: "smoke",
            phases: [],
            phaseSummary: { total: 0, passed: 0, failed: 0, durationSeconds: 60, observationCount: 0 },
            startedAt: "2026-10-20T10:00:00Z",
            completedAt: "2026-10-20T10:01:00Z",
          })}\n\n`,
          { status: 200, headers: { "Content-Type": "text/event-stream" } },
        );
      }
      throw new Error(`Unexpected Test Genie UI request: ${init?.method ?? "GET"} ${requestUrl}`);
    });

    renderExecutionForm({ scenarioName: "demo" });
    await user.click(screen.getByRole("button", { name: /Smoke/ }));
    await user.click(screen.getByRole("checkbox", { name: "Stop on first failed phase" }));
    await user.click(screen.getByRole("button", { name: "Run tests" }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/executions/stream"),
      expect.objectContaining({
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ scenarioName: "demo", preset: "smoke", failFast: false }),
      }),
    ));
    await waitFor(() => expect(screen.getByText("Tests completed in 1.0 min")).toBeInTheDocument());
  });
});
