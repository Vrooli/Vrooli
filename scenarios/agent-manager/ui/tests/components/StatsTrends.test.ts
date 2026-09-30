import assert from "node:assert/strict";
import { screen } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, test, vi } from "vitest";
import type { useQuery } from "@tanstack/react-query";
import { CostDurationTrends } from "../../src/features/stats/components/trends/CostDurationTrends.js";
import { RunStatusTrends } from "../../src/features/stats/components/trends/RunStatusTrends.js";
import type { TimePreset, TimeSeriesResponse } from "../../src/features/stats/api/types.js";
import { useRunTrends } from "../../src/features/stats/hooks/useRunTrends.js";
import { useTimeWindow } from "../../src/features/stats/hooks/useTimeWindow.js";
import { renderWithProviders } from "../../src/test-utils/index.js";
import { makeTimeSeriesResponse } from "../testutil/stats.js";

interface ChartProps {
  children?: ReactNode;
  data?: unknown[];
  dataKey?: string;
  name?: string;
  tickFormatter?: (value: string | number) => string;
  labelFormatter?: (label: string | number | undefined) => string;
  formatter?: (value: string | number, name?: string | number) => [string, string];
}

vi.mock("recharts", async () => {
  const React = await vi.importActual<typeof import("react")>("react");

  function Passthrough({ children }: ChartProps) {
    return React.createElement("div", null, children);
  }

  function Chart({ children, data = [] }: ChartProps) {
    const visibleChildren = React.Children.toArray(children).filter(
      (child) => !(React.isValidElement(child) && child.type === "defs"),
    );

    return React.createElement("div", { "data-testid": "stats-chart", "data-points": data.length }, visibleChildren);
  }

  function Series({ dataKey, name }: ChartProps) {
    return React.createElement("span", { "data-testid": `series-${dataKey ?? name}` }, name ?? dataKey);
  }

  function Tooltip({ formatter, labelFormatter }: ChartProps) {
    const label = labelFormatter?.("2026-05-01T12:00:00.000Z");
    const numeric = formatter?.(4, "Cost");
    const string = formatter?.("3", "failed");
    return React.createElement("span", { "data-testid": "tooltip-preview" }, [label, numeric?.join(" "), string?.join(" ")].filter(Boolean).join(" | "));
  }

  return {
    ResponsiveContainer: Passthrough,
    AreaChart: Chart,
    LineChart: Chart,
    Area: Series,
    Line: Series,
    XAxis: Passthrough,
    YAxis: Passthrough,
    CartesianGrid: Passthrough,
    Tooltip,
    Legend: Passthrough,
  };
});

vi.mock("../../src/features/stats/hooks/useRunTrends.js", () => ({
  useRunTrends: vi.fn(),
}));

vi.mock("../../src/features/stats/hooks/useTimeWindow.js", () => ({
  useTimeWindow: vi.fn(),
}));

type QueryResult = ReturnType<typeof useQuery<TimeSeriesResponse, Error>>;

const presetOptions: readonly TimePreset[] = ["6h", "12h", "24h", "7d", "30d"];

function queryResult(overrides: Partial<QueryResult>): QueryResult {
  return {
    data: undefined,
    isLoading: false,
    error: null,
    ...overrides,
  } as QueryResult;
}

beforeEach(() => {
  vi.mocked(useTimeWindow).mockReturnValue({
    preset: "24h",
    setPreset: vi.fn(),
    filter: { preset: "24h" },
    presetOptions,
  });
});

test("RunStatusTrends renders complete and failed series from time-series buckets", () => {
  vi.mocked(useRunTrends).mockReturnValue(queryResult({
    data: makeTimeSeriesResponse(),
  }));

  renderWithProviders(createElement(RunStatusTrends));

  assert.ok(screen.getByText("Run Trends"));
  assert.equal(screen.getByTestId("stats-chart").getAttribute("data-points"), "2");
  assert.ok(screen.getByText("Completed"));
  assert.ok(screen.getByText("Failed"));
  assert.match(screen.getByTestId("tooltip-preview").textContent ?? "", /4 Cost/);
  assert.match(screen.getByTestId("tooltip-preview").textContent ?? "", /Failed/);
  assert.equal(screen.queryByText("No data available for this time period"), null);
});

test("CostDurationTrends renders cost and average-duration series from time-series buckets", () => {
  vi.mocked(useRunTrends).mockReturnValue(queryResult({
    data: makeTimeSeriesResponse(),
  }));

  renderWithProviders(createElement(CostDurationTrends));

  assert.ok(screen.getByText("Cost & Duration Trends"));
  assert.equal(screen.getByTestId("stats-chart").getAttribute("data-points"), "2");
  assert.ok(screen.getByText("Cost"));
  assert.ok(screen.getByText("Avg Duration"));
  assert.match(screen.getByTestId("tooltip-preview").textContent ?? "", /\$4\.00 Cost/);
  assert.match(screen.getByTestId("tooltip-preview").textContent ?? "", /3ms failed/);
  assert.equal(screen.queryByText("No data available for this time period"), null);
});

test("stats trend components render empty, loading, and error states", () => {
  vi.mocked(useRunTrends).mockReturnValue(queryResult({
    data: makeTimeSeriesResponse({ buckets: [] }),
  }));

  const empty = renderWithProviders(createElement(RunStatusTrends));
  assert.ok(screen.getByText("No data available for this time period"));

  empty.unmount();
  vi.mocked(useRunTrends).mockReturnValue(queryResult({
    isLoading: true,
  }));

  const loading = renderWithProviders(createElement(CostDurationTrends));
  assert.equal(document.querySelectorAll(".animate-pulse").length, 1);

  loading.unmount();
  vi.mocked(useRunTrends).mockReturnValue(queryResult({
    error: new Error("trend stats unavailable"),
  }));

  renderWithProviders(createElement(RunStatusTrends));
  assert.ok(screen.getByText("Run trends: trend stats unavailable"));
});
