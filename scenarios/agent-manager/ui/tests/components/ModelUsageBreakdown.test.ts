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

  return {
    ResponsiveContainer: Passthrough,
    BarChart,
    Bar,
    XAxis: Passthrough,
    YAxis: Passthrough,
    CartesianGrid: Passthrough,
    Tooltip: Passthrough,
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
