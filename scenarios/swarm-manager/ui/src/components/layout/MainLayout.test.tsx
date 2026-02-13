import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { MainLayout } from "./MainLayout";
import { settingsService } from "../../services";

vi.mock("../../services", () => ({
  settingsService: {
    get: vi.fn(),
  },
}));

// [REQ:MOD-P0-008] Test tabbed navigation UI
describe("MainLayout", () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });
    vi.clearAllMocks();
    vi.mocked(settingsService.get).mockResolvedValue({
      theme: "dark",
      customFocus: "",
      insightsEnabled: false,
      insightsAutoAnalyze: false,
    });
  });

  const renderWithRouter = (initialRoute = "/backlog") => {
    return render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[initialRoute]}>
          <MainLayout />
        </MemoryRouter>
      </QueryClientProvider>
    );
  };

  // [REQ:REQ-P0-009] Test desktop tab navigation renders
  it("renders desktop tabs with all navigation items", () => {
    renderWithRouter();

    expect(screen.getByTestId("main-layout")).toBeInTheDocument();
    expect(screen.getByTestId("desktop-tabs")).toBeInTheDocument();

    // Check all tabs exist
    expect(screen.getByTestId("tab-backlog")).toBeInTheDocument();
    expect(screen.getByTestId("tab-scenarios")).toBeInTheDocument();
    expect(screen.getByTestId("tab-execution")).toBeInTheDocument();
    expect(screen.getByTestId("tab-settings")).toBeInTheDocument();
  });

  // [REQ:REQ-P0-009] Test mobile navigation renders
  it("renders mobile navigation with all tabs", () => {
    renderWithRouter();

    expect(screen.getByTestId("mobile-nav")).toBeInTheDocument();

    // Check all mobile tabs exist
    expect(screen.getByTestId("mobile-tab-backlog")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-tab-scenarios")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-tab-execution")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-tab-settings")).toBeInTheDocument();
  });

  // [REQ:REQ-P0-009] Test active tab highlighting on backlog page
  it("highlights Backlog tab when on /backlog route", () => {
    renderWithRouter("/backlog");

    const backlogTab = screen.getByTestId("tab-backlog");
    expect(backlogTab).toHaveClass("text-cyan-400");
  });

  // [REQ:REQ-P0-009] Test active tab highlighting on scenarios page
  it("highlights Scenarios tab when on /scenarios route", () => {
    renderWithRouter("/scenarios");

    const scenariosTab = screen.getByTestId("tab-scenarios");
    expect(scenariosTab).toHaveClass("text-cyan-400");
  });

  // [REQ:REQ-P0-009] Test tab click navigation
  it("navigates when clicking desktop tabs", () => {
    renderWithRouter("/backlog");

    const scenariosTab = screen.getByTestId("tab-scenarios");
    fireEvent.click(scenariosTab);

    // After clicking, the scenarios tab should be active
    expect(scenariosTab).toHaveClass("text-cyan-400");
  });

  // [REQ:REQ-P0-009] Test mobile tab click navigation
  it("navigates when clicking mobile tabs", () => {
    renderWithRouter("/backlog");

    const mobileSettingsTab = screen.getByTestId("mobile-tab-settings");
    fireEvent.click(mobileSettingsTab);

    // After clicking, the settings tab should be active
    expect(mobileSettingsTab).toHaveClass("text-cyan-400");
  });

  // [REQ:REQ-P0-009] Test header displays app name
  it("displays Swarm Manager header", () => {
    renderWithRouter();

    expect(screen.getAllByText("Swarm Manager").length).toBeGreaterThan(0);
  });
});
