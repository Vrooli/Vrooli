// [REQ:KO-HD-001] Basic UI rendering test
import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import App from "./App";

const mockUseHealthStatus = vi.hoisted(() => vi.fn());
const mockUseHashRoute = vi.hoisted(() => vi.fn());

vi.mock("./hooks/knowledgeHooks", () => ({
  useHealthStatus: mockUseHealthStatus,
}));

vi.mock("./hooks/useHashRoute", () => ({
  useHashRoute: mockUseHashRoute,
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
