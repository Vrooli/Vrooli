import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { TestsTab } from "./ScenarioReviewPanelTests";
import { renderWithQueryClient, jsonResponse } from "../test-utils";
import type { TestExecutionResult } from "../lib/api";

function testExecution(overrides: Partial<TestExecutionResult> = {}): TestExecutionResult {
  return {
    executionId: "exec-latest",
    scenarioName: "git-control-tower",
    success: false,
    startedAt: "2026-05-01T12:00:00Z",
    completedAt: "2026-05-01T12:01:20Z",
    preset: "comprehensive",
    phases: [
      {
        name: "unit",
        status: "passed",
        durationSeconds: 12,
      },
      {
        name: "smoke",
        status: "failed",
        durationSeconds: 8,
        error: "Iframe bridge never signaled ready",
        classification: "environment",
        remediation: "Confirm the UI server is reachable before smoke.",
        observations: [
          { prefix: "ERROR", text: "GET http://localhost:21400/ failed" },
        ],
      },
    ],
    phaseSummary: {
      total: 2,
      passed: 1,
      failed: 1,
      durationSeconds: 20,
      observationCount: 1,
    },
    warnings: ["Lighthouse performance was below target"],
    ...overrides,
  };
}

function requestUrl(input: RequestInfo | URL) {
  if (input instanceof Request) return input.url;
  if (input instanceof URL) return input.toString();
  return input;
}

describe("TestsTab", () => {
  it("shows a service unavailable state when test-genie is not available", () => {
    renderWithQueryClient(
      <TestsTab
        scenarioSlug="git-control-tower"
        testGenieAvailable={false}
      />,
    );

    expect(screen.getByText(/test genie is not available/i)).toBeInTheDocument();
    expect(screen.getByText(/start the test-genie scenario/i)).toBeInTheDocument();
  });

  it("renders the latest failed execution, expandable diagnostics, and recent history", async () => {
    const latest = testExecution();
    const previous = testExecution({
      executionId: "exec-previous",
      success: true,
      completedAt: "2026-05-01T11:00:00Z",
      phaseSummary: {
        total: 2,
        passed: 2,
        failed: 0,
        durationSeconds: 18,
        observationCount: 0,
      },
    });
    const fetchMock = vi.fn(async () => jsonResponse({ items: [latest, previous], count: 2 }));
    globalThis.fetch = fetchMock as unknown as typeof fetch;

    renderWithQueryClient(
      <TestsTab
        scenarioSlug="git-control-tower"
        repoId="repo-1"
        testGenieAvailable
      />,
    );

    expect(await screen.findByText("Failed")).toBeInTheDocument();
    expect(screen.getByText("1 passed")).toBeInTheDocument();
    expect(screen.getByText("1 failed")).toBeInTheDocument();
    expect(screen.getByText(/lighthouse performance/i)).toBeInTheDocument();
    expect(screen.getByText("Recent History")).toBeInTheDocument();
    expect(screen.getByText("smoke")).toBeInTheDocument();

    fireEvent.click(screen.getByText("smoke"));

    expect(screen.getByText("Iframe bridge never signaled ready")).toBeInTheDocument();
    expect(screen.getByText("Confirm the UI server is reachable before smoke.")).toBeInTheDocument();
    expect(screen.getByText(/GET http:\/\/localhost:21400\/ failed/)).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith(
      "https://git-control-tower.test/api/v1/repo/test-executions?scenarioName=git-control-tower&limit=10",
      expect.objectContaining({
        cache: "no-store",
        headers: expect.objectContaining({ "X-Repo-Id": "repo-1" }),
      }),
    );
  });

  it("starts a test execution and invalidates the execution list", async () => {
    const latest = testExecution();
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = requestUrl(input);
      if (init?.method === "POST" && url.endsWith("/repo/test-execution")) {
        return jsonResponse(testExecution({ executionId: "exec-new", success: true }));
      }
      return jsonResponse({ items: [latest], count: 1 });
    });
    globalThis.fetch = fetchMock as unknown as typeof fetch;

    renderWithQueryClient(
      <TestsTab
        scenarioSlug="git-control-tower"
        repoId="repo-1"
        testGenieAvailable
      />,
    );

    fireEvent.click(await screen.findByRole("button", { name: /run tests/i }));

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledWith(
        "https://git-control-tower.test/api/v1/repo/test-execution",
        expect.objectContaining({
          method: "POST",
          headers: expect.objectContaining({ "X-Repo-Id": "repo-1" }),
          body: JSON.stringify({ scenarioName: "git-control-tower" }),
        }),
      );
    });

    await waitFor(() => {
      const listFetches = fetchMock.mock.calls.filter(([input]) =>
        requestUrl(input).includes("/repo/test-executions?"),
      );
      expect(listFetches.length).toBeGreaterThanOrEqual(2);
    });
  });
});
