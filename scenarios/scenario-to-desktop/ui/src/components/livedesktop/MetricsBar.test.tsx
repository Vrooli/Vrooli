import { screen } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { useLiveDesktopStore } from "../../store/liveDesktopStore";
import { MetricsBar } from "./MetricsBar";

describe("MetricsBar", () => {
  beforeEach(() => {
    useLiveDesktopStore.setState({
      activeSession: null,
      connectionStatus: "disconnected",
      error: null,
      isOpen: false,
      scenarioName: null,
      appPath: null,
    });
  });

  it("stays hidden until the desktop application is actually running", () => {
    renderWithProviders(<MetricsBar />);

    expect(screen.queryByText("Starting...")).not.toBeInTheDocument();
  });

  it("shows startup, CPU, and memory evidence for a running desktop session", () => {
    useLiveDesktopStore.setState({
      activeSession: {
        appRunning: true,
        metrics: {
          splashDetected: true,
          splashDurationMs: 800n,
          readyDetected: true,
          readyDurationMs: 2100n,
          sampleCount: 1,
          currentCpuPercent: 83.2,
          currentRssMb: 1536,
          peakRssMb: 2048,
        },
      } as never,
    });

    renderWithProviders(<MetricsBar />);

    expect(screen.getByText("Splash 0.8s")).toBeInTheDocument();
    expect(screen.getByText("Ready 2.1s")).toBeInTheDocument();
    expect(screen.getByText("83%")).toBeInTheDocument();
    expect(screen.getByText("1.5 GB")).toBeInTheDocument();
    expect(screen.getByTitle("Peak: 2.0 GB")).toBeInTheDocument();
  });

  it("reports a pending startup instead of inventing readiness timing", () => {
    useLiveDesktopStore.setState({
      activeSession: {
        appRunning: true,
        metrics: { sampleCount: 0 },
      } as never,
    });

    renderWithProviders(<MetricsBar />);

    expect(screen.getByText("Starting...")).toBeInTheDocument();
    expect(screen.queryByText(/Ready/)).not.toBeInTheDocument();
  });
});
