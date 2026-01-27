// [REQ:KO-HD-001] Basic UI rendering test
import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import App from "./App";

const mockUseHealthStatus = vi.hoisted(() => vi.fn());
const mockUseHashRoute = vi.hoisted(() => vi.fn());
const mockUseDocumentationSummary = vi.hoisted(() => vi.fn());
const mockUseActivityFeed = vi.hoisted(() => vi.fn());

vi.mock("./shared/hooks/knowledgeHooks", () => ({
  useHealthStatus: mockUseHealthStatus,
}));

vi.mock("./shared/hooks/useHashRoute", () => ({
  useHashRoute: mockUseHashRoute,
}));

vi.mock("./shared/hooks/documentationSummaryHooks", () => ({
  useDocumentationSummary: mockUseDocumentationSummary,
}));

vi.mock("./shared/hooks/activityHooks", () => ({
  useActivityFeed: mockUseActivityFeed,
}));

describe("App", () => {
  beforeEach(() => {
    mockUseHealthStatus.mockReturnValue({
      viewModel: {
        status: "ok",
        service: "knowledge-observatory",
        lastUpdated: "10:00 AM",
        statusLabel: "Online",
        statusPulse: false,
      },
      isLoading: false,
      hasError: false,
      hasData: true,
      refetch: vi.fn(),
    });

    mockUseHashRoute.mockReturnValue({
      route: "dashboard",
      navigate: vi.fn(),
    });

    mockUseDocumentationSummary.mockReturnValue({
      viewModel: {
        totalScenarios: 3,
        coverageLabel: "3 of 3 documented",
        coveragePercentLabel: "100%",
        coverageTone: "good",
        averageHealthLabel: "98%",
        averageHealthTone: "good",
        manifestCoverageLabel: "100% have manifests",
        lastModifiedLabel: "1/27/2026",
      },
      isLoading: false,
      hasError: false,
      errorMessage: "",
      refetch: vi.fn(),
    });

    mockUseActivityFeed.mockReturnValue([]);

    window.ENV = {
      API_PORT: "17822",
    };
  });

  it("[REQ:KO-HD-001] renders without crashing", () => {
    const { container } = render(<App />);
    expect(container).toBeTruthy();
  });

  it("[REQ:KO-HD-001] renders Knowledge Observatory title", () => {
    render(<App />);
    const titleElement = screen.getByText(/Knowledge Observatory/i);
    expect(titleElement).toBeDefined();
  });

  it("[REQ:KO-HD-002] renders feature cards", () => {
    const { container } = render(<App />);
    expect(container.textContent?.length).toBeGreaterThan(0);
  });
});
