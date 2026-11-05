import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { UptimeStats } from "./UptimeStats";
import * as api from "../../../lib/api";
import { createTimelineResponse, createUptimeStatsResponse, renderWithProviders } from "../../../test-utils";

vi.mock("../../../lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../../lib/api")>();
  return { ...actual, fetchUptimeStats: vi.fn(), fetchTimeline: vi.fn() };
});

describe("UptimeStats", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.fetchUptimeStats).mockResolvedValue(createUptimeStatsResponse());
    vi.mocked(api.fetchTimeline).mockResolvedValue(createTimelineResponse({
      events: [
        { checkId: "dns", status: "ok", message: "ok", timestamp: "2024-01-01T12:00:00Z" },
        { checkId: "db", status: "warning", message: "slow", timestamp: "2024-01-01T11:59:00Z" },
        { checkId: "api", status: "critical", message: "down", timestamp: "2024-01-01T11:58:00Z" },
      ],
    }));
  });

  it("renders uptime bands, status history, and opens trends", async () => {
    const onShowTrends = vi.fn();
    renderWithProviders(<UptimeStats onShowTrends={onShowTrends} />);
    expect(await screen.findByText("Good")).toBeInTheDocument();
    expect(screen.getByText("95.0%")).toBeInTheDocument();
    expect(screen.getByText("100 checks in 24h")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button"));
    fireEvent.keyDown(screen.getByRole("button"), { key: "Enter" });
    fireEvent.keyDown(screen.getByRole("button"), { key: " " });
    expect(onShowTrends).toHaveBeenCalledTimes(3);
  });

  it("renders no-data and error states", async () => {
    vi.mocked(api.fetchTimeline).mockResolvedValue(createTimelineResponse());
    renderWithProviders(<UptimeStats />);
    expect(await screen.findByText("No data")).toBeInTheDocument();

    vi.mocked(api.fetchUptimeStats).mockRejectedValue(new Error("uptime unavailable"));
    renderWithProviders(<UptimeStats />);
    await waitFor(() => expect(screen.getAllByText(/retry/i).length).toBeGreaterThan(0), { timeout: 5000 });
    for (const retry of screen.getAllByText(/retry/i)) fireEvent.click(retry);
  });

  it("supports a critical uptime result", async () => {
    vi.mocked(api.fetchUptimeStats).mockResolvedValue(createUptimeStatsResponse({ uptimePercentage: 70 }));
    renderWithProviders(<UptimeStats />);
    await waitFor(() => expect(screen.getByText("Critical")).toBeInTheDocument());
  });

  it("labels excellent, fair, and poor uptime bands", async () => {
    for (const [percentage, label] of [[99, "Excellent"], [92, "Fair"], [85, "Poor"]] as const) {
      vi.mocked(api.fetchUptimeStats).mockResolvedValueOnce(createUptimeStatsResponse({ uptimePercentage: percentage }));
      const { unmount } = renderWithProviders(<UptimeStats />);
      await waitFor(() => expect(screen.getAllByText(label).length).toBeGreaterThan(0));
      unmount();
    }
  });
});
