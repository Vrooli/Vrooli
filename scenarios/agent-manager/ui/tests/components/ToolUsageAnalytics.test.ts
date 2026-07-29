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
  content?: (props: { active?: boolean; payload?: Array<{ payload: ChartDatum & { calls: number; successRate: number; failedCount: number } }> }) => ReactNode;
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
      React.createElement(
        "button",
        { type: "button", onClick: () => onClick?.({}, undefined as unknown as number) },
        "Ignore malformed chart point",
      ),
      React.createElement(
        "button",
        { type: "button", onClick: () => onClick?.({}, 999) },
        "Ignore missing chart point",
      ),
    );
  }

  function Tooltip({ content }: ChartProps) {
    const [activeIndex, setActiveIndex] = React.useState<number | null>(null);
    return React.createElement(
      "div",
      null,
      currentData.map((datum, index) => React.createElement(
        "button",
        { key: datum.name, type: "button", onClick: () => setActiveIndex(index) },
        `Show chart tooltip for ${datum.name}`,
      )),
      content?.({
        active: activeIndex !== null,
        payload: activeIndex !== null && currentData[activeIndex]
          ? [{ payload: currentData[activeIndex] as ChartDatum & { calls: number; successRate: number; failedCount: number } }]
          : [],
      }),
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

test("ToolUsageAnalytics exposes selected tool loading, failure, empty, and reset states", async () => {
  const user = userEvent.setup();
  vi.mocked(useToolUsage).mockReturnValue(queryResult<ToolUsageResponse>({
    data: makeToolUsageResponse(),
  }) as ToolUsageQuery);
  vi.mocked(useToolUsageRuns).mockReturnValue(queryResult<ToolUsageRunsResponse>({
    isLoading: true,
  }) as ToolRunsQuery);
  vi.mocked(useToolUsageModels).mockReturnValue(queryResult<ToolUsageModelsResponse>({}) as ToolModelsQuery);

  const loading = renderWithProviders(createElement(ToolUsageAnalytics));
  await user.click(screen.getByRole("button", { name: "Edit" }));
  assert.equal(document.querySelectorAll(".animate-pulse").length, 1);
  loading.unmount();

  vi.mocked(useToolUsageRuns).mockReturnValue(queryResult<ToolUsageRunsResponse>({
    error: new Error("tool detail unavailable"),
  }) as ToolRunsQuery);
  const failed = renderWithProviders(createElement(ToolUsageAnalytics));
  await user.click(screen.getByRole("button", { name: "Edit" }));
  assert.ok(screen.getByText("Failed to load runs: tool detail unavailable"));
  failed.unmount();

  vi.mocked(useToolUsageRuns).mockReturnValue(queryResult<ToolUsageRunsResponse>({
    data: makeToolUsageRunsResponse({ runs: [] }),
  }) as ToolRunsQuery);
  renderWithProviders(createElement(ToolUsageAnalytics));
  await user.click(screen.getByRole("button", { name: "Edit" }));
  assert.ok(screen.getByText("No runs found for this tool in the selected window"));
  await user.click(screen.getByRole("button", { name: "Back to tool usage chart" }));
  assert.ok(screen.getByText("Click a bar to view runs"));
});

test("ToolUsageAnalytics reports model-detail loading, errors, and no-model evidence after switching tabs", async () => {
  const user = userEvent.setup();
  vi.mocked(useToolUsage).mockReturnValue(queryResult<ToolUsageResponse>({ data: makeToolUsageResponse() }) as ToolUsageQuery);
  vi.mocked(useToolUsageRuns).mockReturnValue(queryResult<ToolUsageRunsResponse>({ data: makeToolUsageRunsResponse() }) as ToolRunsQuery);
  vi.mocked(useToolUsageModels).mockReturnValue(queryResult<ToolUsageModelsResponse>({ isLoading: true }) as ToolModelsQuery);
  const loading = renderWithProviders(createElement(ToolUsageAnalytics));
  await user.click(screen.getByRole("button", { name: "Edit" }));
  await user.click(screen.getByRole("button", { name: "Models" }));
  assert.equal(document.querySelectorAll(".animate-pulse").length, 1);
  loading.unmount();

  vi.mocked(useToolUsageModels).mockReturnValue(queryResult<ToolUsageModelsResponse>({ error: new Error("model detail unavailable") }) as ToolModelsQuery);
  const failed = renderWithProviders(createElement(ToolUsageAnalytics));
  await user.click(screen.getByRole("button", { name: "Edit" }));
  await user.click(screen.getByRole("button", { name: "Models" }));
  assert.ok(screen.getByText("Failed to load models: model detail unavailable"));
  failed.unmount();

  vi.mocked(useToolUsageModels).mockReturnValue(queryResult<ToolUsageModelsResponse>({ data: makeToolUsageModelsResponse({ models: [] }) }) as ToolModelsQuery);
  renderWithProviders(createElement(ToolUsageAnalytics));
  await user.click(screen.getByRole("button", { name: "Edit" }));
  await user.click(screen.getByRole("button", { name: "Models" }));
  assert.ok(screen.getByText("No model usage found for this tool in the selected window"));
});

test("ToolUsageAnalytics makes unknown zero-call telemetry and incomplete run metadata inspectable", async () => {
  const user = userEvent.setup();
  vi.mocked(useToolUsage).mockReturnValue(queryResult<ToolUsageResponse>({
    data: makeToolUsageResponse({
      tools: [{ toolName: "", callCount: 0, successCount: 0, failedCount: 0 }],
    }),
  }) as ToolUsageQuery);
  vi.mocked(useToolUsageRuns).mockReturnValue(queryResult<ToolUsageRunsResponse>({
    data: makeToolUsageRunsResponse({
      runs: [{
        runId: "run-unknown-12345678",
        taskId: "task-unknown",
        taskTitle: "",
        profileId: "profile-unknown",
        profileName: "Unknown profile",
        status: "pending",
        createdAt: "2026-05-01T15:00:00.000Z",
        model: "unknown",
        callCount: 0,
        successCount: 0,
        failedCount: 0,
      }],
    }),
  }) as ToolRunsQuery);
  vi.mocked(useToolUsageModels).mockReturnValue(queryResult<ToolUsageModelsResponse>({}) as ToolModelsQuery);

  renderWithProviders(createElement(ToolUsageAnalytics));
  await user.click(screen.getByRole("button", { name: "unknown" }));

  assert.ok(screen.getAllByText("Unknown").length >= 2);
  assert.ok(screen.getByText("0.0%"));
  assert.ok(screen.getByRole("link", { name: "Untitled Task" }));
  assert.ok(screen.getAllByText("Unknown").length >= 2);
});

test("ToolUsageAnalytics renders a chart skeleton while aggregate telemetry is loading", () => {
  vi.mocked(useToolUsage).mockReturnValue(queryResult<ToolUsageResponse>({ isLoading: true }) as ToolUsageQuery);
  vi.mocked(useToolUsageRuns).mockReturnValue(queryResult<ToolUsageRunsResponse>({}) as ToolRunsQuery);
  vi.mocked(useToolUsageModels).mockReturnValue(queryResult<ToolUsageModelsResponse>({}) as ToolModelsQuery);

  renderWithProviders(createElement(ToolUsageAnalytics));

  assert.equal(document.querySelectorAll(".animate-pulse").length, 2);
});

test("ToolUsageAnalytics explains aggregate chart values and success-rate severity in its tooltip", async () => {
  const user = userEvent.setup();
  vi.mocked(useToolUsage).mockReturnValue(queryResult<ToolUsageResponse>({
    data: makeToolUsageResponse({
      tools: [
        { toolName: "Reliable", callCount: 10, successCount: 10, failedCount: 0 },
        { toolName: "Warning", callCount: 5, successCount: 4, failedCount: 1 },
        { toolName: "Failing", callCount: 1, successCount: 0, failedCount: 1 },
      ],
    }),
  }) as ToolUsageQuery);
  vi.mocked(useToolUsageRuns).mockReturnValue(queryResult<ToolUsageRunsResponse>({}) as ToolRunsQuery);
  vi.mocked(useToolUsageModels).mockReturnValue(queryResult<ToolUsageModelsResponse>({}) as ToolModelsQuery);

  renderWithProviders(createElement(ToolUsageAnalytics));
  await user.click(screen.getByRole("button", { name: "Show chart tooltip for Reliable" }));

  assert.equal(screen.getAllByText("Reliable").length, 2);
  assert.match(screen.getByText("Calls:").parentElement?.textContent ?? "", /10.*62\.5%/);
  assert.ok(screen.getByText("100.0%"));
  assert.ok(screen.getByText("0"));
});

test("ToolUsageAnalytics keeps malformed chart events inert and explains warning, failure, and zero-call tool telemetry", async () => {
  const user = userEvent.setup();
  vi.mocked(useToolUsage).mockReturnValue(queryResult<ToolUsageResponse>({
    data: makeToolUsageResponse({
      tools: [
        { toolName: "Warning", callCount: 5, successCount: 4, failedCount: 1 },
        { toolName: "Failing", callCount: 1, successCount: 0, failedCount: 1 },
        { toolName: "Idle", callCount: 0, successCount: 0, failedCount: 0 },
      ],
    }),
  }) as ToolUsageQuery);
  vi.mocked(useToolUsageRuns).mockReturnValue(queryResult<ToolUsageRunsResponse>({}) as ToolRunsQuery);
  vi.mocked(useToolUsageModels).mockReturnValue(queryResult<ToolUsageModelsResponse>({}) as ToolModelsQuery);
  const telemetry = renderWithProviders(createElement(ToolUsageAnalytics));

  await user.click(screen.getByRole("button", { name: "Ignore malformed chart point" }));
  await user.click(screen.getByRole("button", { name: "Ignore missing chart point" }));
  assert.ok(screen.getByText("Click a bar to view runs"));

  for (const [tool, rate] of [["Warning", "80.0%"], ["Failing", "0.0%"], ["Idle", "0.0%"]] as const) {
    await user.click(screen.getByRole("button", { name: `Show chart tooltip for ${tool}` }));
    assert.ok(screen.getByText(rate));
  }
  telemetry.unmount();
  vi.mocked(useToolUsage).mockReturnValue(queryResult<ToolUsageResponse>({
    data: makeToolUsageResponse({ tools: [{ toolName: "Idle", callCount: 0, successCount: 0, failedCount: 0 }] }),
  }) as ToolUsageQuery);
  renderWithProviders(createElement(ToolUsageAnalytics));
  await user.click(screen.getByRole("button", { name: "Show chart tooltip for Idle" }));
  assert.equal(screen.queryByText(/\(0\.0%\)/), null);
});
