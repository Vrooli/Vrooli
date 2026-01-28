import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { MainLayout } from "./MainLayout";

// [REQ:MOD-P0-008] Test tabbed navigation UI
describe("MainLayout", () => {
  const renderWithRouter = (initialRoute = "/ideas") => {
    return render(
      <MemoryRouter initialEntries={[initialRoute]}>
        <MainLayout />
      </MemoryRouter>
    );
  };

  // [REQ:REQ-P0-009] Test desktop tab navigation renders
  it("renders desktop tabs with all navigation items", () => {
    renderWithRouter();

    expect(screen.getByTestId("main-layout")).toBeInTheDocument();
    expect(screen.getByTestId("desktop-tabs")).toBeInTheDocument();

    // Check all tabs exist
    expect(screen.getByTestId("tab-ideas")).toBeInTheDocument();
    expect(screen.getByTestId("tab-scenarios")).toBeInTheDocument();
    expect(screen.getByTestId("tab-recommendations")).toBeInTheDocument();
    expect(screen.getByTestId("tab-settings")).toBeInTheDocument();
  });

  // [REQ:REQ-P0-009] Test mobile navigation renders
  it("renders mobile navigation with all tabs", () => {
    renderWithRouter();

    expect(screen.getByTestId("mobile-nav")).toBeInTheDocument();

    // Check all mobile tabs exist
    expect(screen.getByTestId("mobile-tab-ideas")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-tab-scenarios")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-tab-recommendations")).toBeInTheDocument();
    expect(screen.getByTestId("mobile-tab-settings")).toBeInTheDocument();
  });

  // [REQ:REQ-P0-009] Test active tab highlighting on ideas page
  it("highlights Ideas tab when on /ideas route", () => {
    renderWithRouter("/ideas");

    const ideasTab = screen.getByTestId("tab-ideas");
    expect(ideasTab).toHaveClass("text-cyan-400");
  });

  // [REQ:REQ-P0-009] Test active tab highlighting on scenarios page
  it("highlights Scenarios tab when on /scenarios route", () => {
    renderWithRouter("/scenarios");

    const scenariosTab = screen.getByTestId("tab-scenarios");
    expect(scenariosTab).toHaveClass("text-cyan-400");
  });

  // [REQ:REQ-P0-009] Test tab click navigation
  it("navigates when clicking desktop tabs", () => {
    renderWithRouter("/ideas");

    const scenariosTab = screen.getByTestId("tab-scenarios");
    fireEvent.click(scenariosTab);

    // After clicking, the scenarios tab should be active
    expect(scenariosTab).toHaveClass("text-cyan-400");
  });

  // [REQ:REQ-P0-009] Test mobile tab click navigation
  it("navigates when clicking mobile tabs", () => {
    renderWithRouter("/ideas");

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
