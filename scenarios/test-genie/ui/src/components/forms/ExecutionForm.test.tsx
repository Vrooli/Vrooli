import { beforeEach, describe, expect, it, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { ExecutionForm } from "./ExecutionForm";
import { initialExecutionForm, useUIStore } from "../../stores/uiStore";

const fetchMock = vi.fn();

vi.stubGlobal("fetch", fetchMock);

function renderExecutionForm(props?: { scenarioName?: string }) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false
      }
    }
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <ExecutionForm
        scenarioOptions={[]}
        datalistId="execution-scenarios"
        scenarioName={props?.scenarioName}
      />
    </QueryClientProvider>
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
});
