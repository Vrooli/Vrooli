import assert from "node:assert/strict";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createElement, type ReactNode } from "react";
import { test, vi } from "vitest";
import type { useQuery } from "@tanstack/react-query";
import { ToolUsageAnalytics } from "../../src/features/stats/components/breakdown/ToolUsageAnalytics.js";
import type {
  ToolUsageModelsResponse,
  ToolUsageResponse,
  ToolUsageRunsResponse,
} from "../../src/features/stats/api/types.js";
import {
  useToolUsage,
  useToolUsageModels,
  useToolUsageRuns,
} from "../../src/features/stats/hooks/useToolUsage.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import {
  makeToolUsageModelsResponse,
  makeToolUsageResponse,
  makeToolUsageRunsResponse,
} from "../testutil/stats.js";

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

vi.mock("../../src/features/stats/hooks/useToolUsage.js", () => ({
  useToolUsage: vi.fn(),
  useToolUsageRuns: vi.fn(),
  useToolUsageModels: vi.fn(),
}));

type ToolUsageQuery = ReturnType<typeof useQuery<ToolUsageResponse, Error>>;
type ToolRunsQuery = ReturnType<typeof useQuery<ToolUsageRunsResponse, Error>>;
type ToolModelsQuery = ReturnType<typeof useQuery<ToolUsageModelsResponse, Error>>;

function queryResult<T>(overrides: Partial<ReturnType<typeof useQuery<T, Error>>>): ReturnType<typeof useQuery<T, Error>> {
  return {
    data: undefined,
    isLoading: false,
    error: null,
    ...overrides,
  } as ReturnType<typeof useQuery<T, Error>>;
}

test("ToolUsageAnalytics opens selected tool run details from the chart", async () => {
  const user = userEvent.setup();
  vi.mocked(useToolUsage).mockReturnValue(queryResult<ToolUsageResponse>({
    data: makeToolUsageResponse(),
  }) as ToolUsageQuery);
  vi.mocked(useToolUsageRuns).mockReturnValue(queryResult<ToolUsageRunsResponse>({
    data: makeToolUsageRunsResponse(),
  }) as ToolRunsQuery);
  vi.mocked(useToolUsageModels).mockReturnValue(queryResult<ToolUsageModelsResponse>({
    data: makeToolUsageModelsResponse(),
  }) as ToolModelsQuery);

  renderWithProviders(createElement(ToolUsageAnalytics));

  assert.ok(screen.getByText("Click a bar to view runs"));
  await user.click(screen.getByRole("button", { name: "Edit" }));

  await waitFor(() => {
    assert.ok(screen.getByText("Patch orchestration tests"));
  });

  assert.ok(screen.getAllByText("Tool Usage").length > 0);
  assert.ok(screen.getByText("90.0%"));
  assert.ok(screen.getAllByText("2").length > 0);
  assert.ok(screen.getByRole("link", { name: "Patch orchestration tests" }).getAttribute("href")?.endsWith("/runs/run-tool-12345678"));
  assert.ok(
    vi.mocked(useToolUsageRuns).mock.calls.some(([options]) =>
      options?.toolName === "Edit" && options.enabled === true && options.limit === 25
    ),
  );
});

test("ToolUsageAnalytics switches selected tool details from runs to models", async () => {
  const user = userEvent.setup();
  vi.mocked(useToolUsage).mockReturnValue(queryResult<ToolUsageResponse>({
    data: makeToolUsageResponse(),
  }) as ToolUsageQuery);
  vi.mocked(useToolUsageRuns).mockReturnValue(queryResult<ToolUsageRunsResponse>({
    data: makeToolUsageRunsResponse(),
  }) as ToolRunsQuery);
  vi.mocked(useToolUsageModels).mockReturnValue(queryResult<ToolUsageModelsResponse>({
    data: makeToolUsageModelsResponse(),
  }) as ToolModelsQuery);

  renderWithProviders(createElement(ToolUsageAnalytics));

  await user.click(screen.getByRole("button", { name: "Edit" }));
  await user.click(screen.getByRole("button", { name: "Models" }));

  await waitFor(() => {
    assert.ok(screen.getByText("claude-3-opus"));
  });

  assert.ok(screen.getByText("6 runs • 12 calls"));
  assert.ok(screen.getByText("83.3% success"));
  assert.ok(
    vi.mocked(useToolUsageModels).mock.calls.some(([options]) =>
      options?.toolName === "Edit" && options.enabled === true && options.limit === 25
    ),
  );
});

test("ToolUsageAnalytics renders empty and error states", () => {
  vi.mocked(useToolUsage).mockReturnValue(queryResult<ToolUsageResponse>({
    data: makeToolUsageResponse({ tools: [] }),
  }) as ToolUsageQuery);
  vi.mocked(useToolUsageRuns).mockReturnValue(queryResult<ToolUsageRunsResponse>({}) as ToolRunsQuery);
  vi.mocked(useToolUsageModels).mockReturnValue(queryResult<ToolUsageModelsResponse>({}) as ToolModelsQuery);

  const { unmount } = renderWithProviders(createElement(ToolUsageAnalytics));

  assert.ok(screen.getByText("No tool usage data available"));

  unmount();
  vi.mocked(useToolUsage).mockReturnValue(queryResult<ToolUsageResponse>({
    error: new Error("tool stats unavailable"),
  }) as ToolUsageQuery);

  renderWithProviders(createElement(ToolUsageAnalytics));

  assert.ok(screen.getByText("Failed to load: tool stats unavailable"));
});
