import assert from "node:assert/strict";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement, type ReactNode } from "react";
import { test, vi } from "vitest";
import type { useQuery } from "@tanstack/react-query";
import { ModelUsageBreakdown } from "../../src/features/stats/components/breakdown/ModelUsageBreakdown.js";
import type { ModelBreakdownResponse, ModelUsageRunsResponse } from "../../src/features/stats/api/types.js";
import { useModelBreakdown, useModelUsageRuns } from "../../src/features/stats/hooks/useModelBreakdown.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import { makeModelBreakdownResponse, makeModelUsageRunsResponse } from "../testutil/stats.js";

interface ChartDatum {
  name?: string;
}

interface ChartProps {
  children?: ReactNode;
  data?: ChartDatum[];
  onClick?: (datum: ChartDatum, index: number) => void;
  content?: (props: { active?: boolean; payload?: Array<{ payload: ChartDatum }> }) => ReactNode;
  formatter?: (value: unknown, name: unknown) => unknown;
  labelFormatter?: (label: unknown) => ReactNode;
}

vi.mock("recharts", async () => {
  const React = await vi.importActual<typeof import("react")>("react");
  let currentData: ChartDatum[] = [];

  function Passthrough({ children }: ChartProps) {
    return React.createElement("div", null, children);
  }

  function BarChart({ children, data = [] }: ChartProps) {
    currentData = data;
    return React.createElement("div", { "data-testid": "bar-chart" }, children);
  }

  function Bar({ onClick }: ChartProps) {
    return React.createElement(
      "div",
      null,
      currentData.map((datum, index) =>
        React.createElement(
          "button",
          {
            key: datum.name ?? String(index),
            type: "button",
            onClick: () => onClick?.(datum, index),
          },
          datum.name ?? String(index),
        ),
      ),
    );
  }

  function Tooltip({ content, formatter, labelFormatter }: ChartProps) {
    return React.createElement(
      "div",
      { "data-testid": "model-tooltip" },
      currentData.map((datum, index) => React.createElement(
        "div",
        { key: datum.name ?? String(index) },
        labelFormatter?.(datum.name),
        String(formatter?.((datum as ChartDatum & { runs?: number }).runs, "runs") ?? ""),
        String(formatter?.("not-a-number", "cost") ?? ""),
        content?.({ active: true, payload: [{ payload: datum }] }),
      )),
    );
  }

  return {
    ResponsiveContainer: Passthrough,
    BarChart,
    Bar,
    XAxis: Passthrough,
    YAxis: Passthrough,
    CartesianGrid: Passthrough,
    Tooltip,
    Cell: () => null,
  };
});

vi.mock("../../src/features/stats/hooks/useModelBreakdown.js", () => ({
  useModelBreakdown: vi.fn(),
  useModelUsageRuns: vi.fn(),
}));

type ModelBreakdownQuery = ReturnType<typeof useQuery<ModelBreakdownResponse, Error>>;
type ModelRunsQuery = ReturnType<typeof useQuery<ModelUsageRunsResponse, Error>>;

function queryResult<T>(overrides: Partial<ReturnType<typeof useQuery<T, Error>>>): ReturnType<typeof useQuery<T, Error>> {
  return {
    data: undefined,
    isLoading: false,
    error: null,
    ...overrides,
  } as ReturnType<typeof useQuery<T, Error>>;
}

test("ModelUsageBreakdown opens selected model run details from the chart", async () => {
  const user = userEvent.setup();
  vi.mocked(useModelBreakdown).mockReturnValue(queryResult<ModelBreakdownResponse>({
    data: makeModelBreakdownResponse(),
  }) as ModelBreakdownQuery);
  vi.mocked(useModelUsageRuns).mockReturnValue(queryResult<ModelUsageRunsResponse>({
    data: makeModelUsageRunsResponse(),
  }) as ModelRunsQuery);

  renderWithProviders(createElement(ModelUsageBreakdown));

  assert.ok(screen.getByText("Click a bar to view runs"));
  await user.click(screen.getByRole("button", { name: "claude-3-opus" }));

  await waitFor(() => {
    assert.ok(screen.getByText("Audit dependency graph"));
  });

  assert.ok(screen.getAllByText("Model Usage").length > 0);
  assert.ok(screen.getByText("75.0%"));
  assert.ok(screen.getByText("$3.25"));
  assert.ok(screen.getByText("42.0K"));
  assert.ok(screen.getByRole("link", { name: "Audit dependency graph" }).getAttribute("href")?.endsWith("/runs/run-model-12345678"));
  assert.ok(
    vi.mocked(useModelUsageRuns).mock.calls.some(([options]) =>
      options?.model === "claude-3-opus" && options.enabled === true && options.limit === 25
    ),
  );
});

test("ModelUsageBreakdown renders empty and error states", () => {
  vi.mocked(useModelBreakdown).mockReturnValue(queryResult<ModelBreakdownResponse>({
    data: makeModelBreakdownResponse({ models: [] }),
  }) as ModelBreakdownQuery);
  vi.mocked(useModelUsageRuns).mockReturnValue(queryResult<ModelUsageRunsResponse>({}) as ModelRunsQuery);

  const { unmount } = renderWithProviders(createElement(ModelUsageBreakdown));

  assert.ok(screen.getByText("No model data available"));

  unmount();
  vi.mocked(useModelBreakdown).mockReturnValue(queryResult<ModelBreakdownResponse>({
    error: new Error("model stats unavailable"),
  }) as ModelBreakdownQuery);

  renderWithProviders(createElement(ModelUsageBreakdown));

  assert.ok(screen.getByText("Failed to load: model stats unavailable"));
});

test("ModelUsageBreakdown keeps the selected-model state honest while its run detail is loading, unavailable, or empty", async () => {
  const user = userEvent.setup();
  vi.mocked(useModelBreakdown).mockReturnValue(queryResult<ModelBreakdownResponse>({
    data: makeModelBreakdownResponse(),
  }) as ModelBreakdownQuery);
  vi.mocked(useModelUsageRuns).mockReturnValue(queryResult<ModelUsageRunsResponse>({
    isLoading: true,
  }) as ModelRunsQuery);

  const loading = renderWithProviders(createElement(ModelUsageBreakdown));
  await user.click(screen.getByRole("button", { name: "claude-3-opus" }));
  assert.equal(document.querySelectorAll(".animate-pulse").length, 1);
  loading.unmount();

  vi.mocked(useModelUsageRuns).mockReturnValue(queryResult<ModelUsageRunsResponse>({
    error: new Error("selected model unavailable"),
  }) as ModelRunsQuery);
  const failed = renderWithProviders(createElement(ModelUsageBreakdown));
  await user.click(screen.getByRole("button", { name: "claude-3-opus" }));
  assert.ok(screen.getByText("Failed to load runs: selected model unavailable"));
  failed.unmount();

  vi.mocked(useModelUsageRuns).mockReturnValue(queryResult<ModelUsageRunsResponse>({
    data: makeModelUsageRunsResponse({ runs: [] }),
  }) as ModelRunsQuery);
  renderWithProviders(createElement(ModelUsageBreakdown));
  await user.click(screen.getByRole("button", { name: "claude-3-opus" }));
  assert.ok(screen.getByText("No runs found for this model in the selected window"));
  await user.click(screen.getByRole("button", { name: "Back to model usage chart" }));
  assert.ok(screen.getByText("Click a bar to view runs"));
});

test("ModelUsageBreakdown presents informative chart tooltip states for unknown, perfect, and failed models", () => {
  vi.mocked(useModelBreakdown).mockReturnValue(queryResult<ModelBreakdownResponse>({
    data: makeModelBreakdownResponse({
      models: [
        { model: "", runCount: 0, successCount: 0, totalCostUsd: 0, totalTokens: 0 },
        { model: "perfect-model", runCount: 10, successCount: 10, totalCostUsd: 1.25, totalTokens: 1000 },
        { model: "failed-model", runCount: 3, successCount: 0, totalCostUsd: 0.5, totalTokens: 500 },
      ],
    }),
  }) as ModelBreakdownQuery);
  vi.mocked(useModelUsageRuns).mockReturnValue(queryResult<ModelUsageRunsResponse>({}) as ModelRunsQuery);

  renderWithProviders(createElement(ModelUsageBreakdown));

  const tooltip = screen.getByTestId("model-tooltip");
  assert.match(tooltip.textContent ?? "", /Unknown/);
  assert.match(tooltip.textContent ?? "", /Runs:/);
  assert.match(tooltip.textContent ?? "", /0 \(0\.0%\)/);
  assert.ok(document.querySelector(".text-red-500"));
});

test("ModelUsageBreakdown handles missing run labels and returns to the chart", async () => {
  const user = userEvent.setup();
  vi.mocked(useModelBreakdown).mockReturnValue(queryResult<ModelBreakdownResponse>({
    data: makeModelBreakdownResponse({
      models: [{ model: "minimal-model", runCount: 1, successCount: 1, totalCostUsd: 0, totalTokens: 0 }],
    }),
  }) as ModelBreakdownQuery);
  vi.mocked(useModelUsageRuns).mockReturnValue(queryResult<ModelUsageRunsResponse>({
    data: makeModelUsageRunsResponse({
      runs: [{
        runId: "run-minimal-1234",
        taskId: "task-minimal",
        taskTitle: "",
        profileId: "profile-minimal",
        profileName: "",
        status: "cancelled",
        createdAt: "2026-05-01T14:00:00.000Z",
        totalCostUsd: 0,
        totalTokens: 0,
      }],
    }),
  }) as ModelRunsQuery);

  renderWithProviders(createElement(ModelUsageBreakdown));
  await user.click(screen.getByRole("button", { name: "minimal-model" }));
  assert.ok(screen.getByRole("link", { name: "Untitled Task" }));
  assert.ok(screen.getByText("cancelled", { exact: false }));
  await user.click(screen.getByRole("button", { name: "Back to model usage chart" }));
  assert.ok(screen.getByText("Click a bar to view runs"));
});
