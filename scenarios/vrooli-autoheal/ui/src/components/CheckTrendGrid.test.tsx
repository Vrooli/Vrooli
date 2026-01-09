// CheckTrendGrid component tests
// [REQ:UI-EVENTS-001] [REQ:PERSIST-HISTORY-001]
import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { CheckTrendGrid } from "./CheckTrendGrid";
import type { CheckTrend, TimelineEvent } from "../lib/api";

describe("[REQ:UI-EVENTS-001] CheckTrendGrid", () => {
  describe("with backend trends data", () => {
    const mockTrends: CheckTrend[] = [
      {
        checkId: "check-1",
        total: 100,
        ok: 95,
        warning: 3,
        critical: 2,
        uptimePercent: 95,
        currentStatus: "ok",
        recentStatuses: ["ok", "ok", "warning", "ok"],
        lastChecked: "2024-01-01T12:00:00Z",
      },
      {
        checkId: "check-2",
        total: 50,
        ok: 25,
        warning: 15,
        critical: 10,
        uptimePercent: 50,
        currentStatus: "critical",
        recentStatuses: ["critical", "warning", "ok"],
        lastChecked: "2024-01-01T11:00:00Z",
      },
    ];

    it("renders check trends", () => {
      render(<CheckTrendGrid trends={mockTrends} />);

      expect(screen.getByTestId("autoheal-trends-check-grid")).toBeInTheDocument();
      expect(screen.getByText("check-1")).toBeInTheDocument();
      expect(screen.getByText("check-2")).toBeInTheDocument();
    });

    it("displays total check counts", () => {
      render(<CheckTrendGrid trends={mockTrends} />);

      expect(screen.getByText("100 checks")).toBeInTheDocument();
      expect(screen.getByText("50 checks")).toBeInTheDocument();
    });

    it("displays uptime percentages", () => {
      render(<CheckTrendGrid trends={mockTrends} />);

      expect(screen.getByText("95%")).toBeInTheDocument();
      expect(screen.getByText("50%")).toBeInTheDocument();
    });

    it("applies correct color classes for uptime", () => {
      render(<CheckTrendGrid trends={mockTrends} />);

      // 95% should be green (>= 90)
      const uptimeElements = screen.getAllByText(/\d+%/);
      expect(uptimeElements[0]).toHaveClass("text-amber-400"); // 95% is >= 90, but < 99
      expect(uptimeElements[1]).toHaveClass("text-red-400"); // 50% is < 90
    });

    it("calls onCheckClick when row is clicked", () => {
      const onCheckClick = vi.fn();
      render(<CheckTrendGrid trends={mockTrends} onCheckClick={onCheckClick} />);

      fireEvent.click(screen.getByText("check-1"));
      expect(onCheckClick).toHaveBeenCalledWith("check-1");
    });

    it("adds cursor-pointer class when onCheckClick provided", () => {
      const onCheckClick = vi.fn();
      render(<CheckTrendGrid trends={mockTrends} onCheckClick={onCheckClick} />);

      const rows = screen.getAllByRole("button");
      expect(rows[0]).toHaveClass("cursor-pointer");
    });

    it("handles keyboard navigation", () => {
      const onCheckClick = vi.fn();
      render(<CheckTrendGrid trends={mockTrends} onCheckClick={onCheckClick} />);

      const row = screen.getAllByRole("button")[0];
      fireEvent.keyDown(row, { key: "Enter" });
      expect(onCheckClick).toHaveBeenCalledWith("check-1");

      fireEvent.keyDown(row, { key: " " });
      expect(onCheckClick).toHaveBeenCalledTimes(2);
    });
  });

  describe("with timeline events (fallback)", () => {
    const mockEvents: TimelineEvent[] = [
      {
        checkId: "event-check",
        status: "ok",
        message: "All good",
        timestamp: "2024-01-01T12:00:00Z",
      },
      {
        checkId: "event-check",
        status: "ok",
        message: "Still good",
        timestamp: "2024-01-01T11:00:00Z",
      },
      {
        checkId: "event-check",
        status: "warning",
        message: "Getting warm",
        timestamp: "2024-01-01T10:00:00Z",
      },
    ];

    it("computes trends from events when no backend trends", () => {
      render(<CheckTrendGrid events={mockEvents} />);

      expect(screen.getByText("event-check")).toBeInTheDocument();
      expect(screen.getByText("3 checks")).toBeInTheDocument();
    });

    it("calculates correct uptime from events", () => {
      render(<CheckTrendGrid events={mockEvents} />);

      // 2 ok out of 3 = 67%
      expect(screen.getByText("67%")).toBeInTheDocument();
    });
  });

  describe("empty state", () => {
    it("shows empty message when no data", () => {
      render(<CheckTrendGrid trends={[]} events={[]} />);

      expect(screen.getByText(/no check data available/i)).toBeInTheDocument();
    });

    it("shows instruction to run tick", () => {
      render(<CheckTrendGrid />);

      expect(screen.getByText(/run a health check tick/i)).toBeInTheDocument();
    });
  });

  describe("preference for backend data", () => {
    it("uses backend trends over events when both provided", () => {
      const trends: CheckTrend[] = [
        {
          checkId: "backend-check",
          total: 100,
          ok: 100,
          warning: 0,
          critical: 0,
          uptimePercent: 100,
          currentStatus: "ok",
          recentStatuses: ["ok"],
          lastChecked: "2024-01-01T12:00:00Z",
        },
      ];

      const events: TimelineEvent[] = [
        {
          checkId: "event-check",
          status: "critical",
          message: "Bad",
          timestamp: "2024-01-01T12:00:00Z",
        },
      ];

      render(<CheckTrendGrid trends={trends} events={events} />);

      // Should show backend check, not event check
      expect(screen.getByText("backend-check")).toBeInTheDocument();
      expect(screen.queryByText("event-check")).not.toBeInTheDocument();
    });
  });
});
