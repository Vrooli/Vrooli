import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";

vi.mock("../../api/inventory", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/inventory")>();
  return {
    ...actual,
    fetchRuns: vi.fn(),
  };
});

vi.mock("recharts", () => ({
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  BarChart: ({ children }: { children: React.ReactNode }) => (
    <div role="img" aria-label="run-outcomes-chart">
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

describe("TimelineCard accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
    const { fetchRuns } = await import("../../api/inventory");
    vi.mocked(fetchRuns).mockResolvedValue([
      {
        id: "a",
        flowId: "f",
        flowPath: "p",
        root: ".",
        mode: "check",
        status: "passed",
        startedAt: "2026-05-10T12:00:00Z",
        finishedAt: "2026-05-10T12:00:00Z",
        durationMs: 1,
      },
      {
        id: "b",
        flowId: "f",
        flowPath: "p",
        root: ".",
        mode: "check",
        status: "failed",
        startedAt: "2026-05-11T12:00:00Z",
        finishedAt: "2026-05-11T12:00:00Z",
        durationMs: 1,
      },
    ]);
  });

  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(<TimelineCard />);
    await waitFor(() =>
      expect(screen.getByTestId("timeline-chart")).toBeInTheDocument(),
    );
    await expectNoA11yViolations(container);
  });
});
