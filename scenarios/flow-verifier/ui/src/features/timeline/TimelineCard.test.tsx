import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";

vi.mock("../../api/inventory", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/inventory")>();
  return {
    ...actual,
    fetchRuns: vi.fn(),
  };
});

vi.mock("recharts", () => ({
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="rc-responsive">{children}</div>
  ),
  BarChart: ({
    data,
    children,
  }: {
    data: Array<{ day: string; passed: number; failed: number; error: number }>;
    children: React.ReactNode;
  }) => (
    <div data-testid="rc-barchart">
      <ul data-testid="rc-data">
        {data.map((d) => (
          <li
            key={d.day}
            data-testid={`rc-day-${d.day}`}
            data-passed={d.passed}
            data-failed={d.failed}
            data-error={d.error}
          >
            {d.day}
          </li>
        ))}
      </ul>
      {children}
    </div>
  ),
  Bar: () => null,
  XAxis: () => null,
  YAxis: () => null,
  CartesianGrid: () => null,
  Tooltip: () => null,
  Legend: () => null,
}));

import { TimelineCard } from "./TimelineCard";
import type { RunRow } from "../../api/inventory";

const makeRun = (overrides: Partial<RunRow>): RunRow => ({
  id: "r",
  flowId: "f",
  flowPath: "p",
  root: ".",
  mode: "check",
  status: "passed",
  startedAt: "2026-05-10T12:00:00Z",
  finishedAt: "2026-05-10T12:00:00Z",
  durationMs: 1,
  ...overrides,
});

describe("TimelineCard", () => {
  beforeEach(async () => {
    const { fetchRuns } = await import("../../api/inventory");
    vi.mocked(fetchRuns).mockReset();
  });
  afterEach(() => cleanup());

  it("renders the card root", async () => {
    const { fetchRuns } = await import("../../api/inventory");
    vi.mocked(fetchRuns).mockResolvedValue([]);
    renderWithProviders(<TimelineCard />);
    expect(screen.getByTestId("timeline-card")).toBeInTheDocument();
  });

  it("renders the loading state while in flight", async () => {
    const { fetchRuns } = await import("../../api/inventory");
    vi.mocked(fetchRuns).mockReturnValue(new Promise(() => {}));
    renderWithProviders(<TimelineCard />);
    expect(screen.getByTestId("timeline-loading")).toBeInTheDocument();
  });

  it("renders an empty state when there is fewer than 2 distinct days of data", async () => {
    const { fetchRuns } = await import("../../api/inventory");
    vi.mocked(fetchRuns).mockResolvedValue([
      makeRun({ id: "a", finishedAt: "2026-05-10T12:00:00Z" }),
      makeRun({ id: "b", finishedAt: "2026-05-10T13:00:00Z" }),
    ]);
    renderWithProviders(<TimelineCard />);
    await waitFor(() =>
      expect(screen.getByTestId("timeline-empty")).toBeInTheDocument(),
    );
    expect(screen.queryByTestId("timeline-chart")).not.toBeInTheDocument();
  });

  it("renders the chart when there are >=2 distinct days, bucketed by status", async () => {
    const { fetchRuns } = await import("../../api/inventory");
    vi.mocked(fetchRuns).mockResolvedValue([
      makeRun({ id: "a", status: "passed", finishedAt: "2026-05-10T12:00:00Z" }),
      makeRun({ id: "b", status: "failed", finishedAt: "2026-05-10T13:00:00Z" }),
      makeRun({ id: "c", status: "error", finishedAt: "2026-05-11T08:00:00Z" }),
      makeRun({ id: "d", status: "passed", finishedAt: "2026-05-11T09:00:00Z" }),
    ]);
    renderWithProviders(<TimelineCard />);
    await waitFor(() =>
      expect(screen.getByTestId("timeline-chart")).toBeInTheDocument(),
    );
    const day0 = screen.getByTestId("rc-day-2026-05-10");
    const day1 = screen.getByTestId("rc-day-2026-05-11");
    expect(day0).toHaveAttribute("data-passed", "1");
    expect(day0).toHaveAttribute("data-failed", "1");
    expect(day0).toHaveAttribute("data-error", "0");
    expect(day1).toHaveAttribute("data-passed", "1");
    expect(day1).toHaveAttribute("data-error", "1");
  });

  it("renders an error state when fetchRuns rejects", async () => {
    const { fetchRuns } = await import("../../api/inventory");
    vi.mocked(fetchRuns).mockRejectedValue(new Error("boom"));
    renderWithProviders(<TimelineCard />);
    await waitFor(() =>
      expect(screen.getByTestId("timeline-error")).toBeInTheDocument(),
    );
  });

  it("scopes by flowId when the prop is set", async () => {
    const { fetchRuns } = await import("../../api/inventory");
    vi.mocked(fetchRuns).mockResolvedValue([]);
    renderWithProviders(<TimelineCard flowId="example.flow" />);
    await waitFor(() => {
      expect(vi.mocked(fetchRuns)).toHaveBeenCalled();
    });
    const lastCall = vi.mocked(fetchRuns).mock.calls.at(-1)?.[0];
    expect(lastCall).toEqual(expect.objectContaining({ flowId: "example.flow" }));
  });
});
