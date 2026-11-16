import { afterEach, describe, it, expect } from "vitest";
import { cleanup, fireEvent, screen } from "@testing-library/react";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { Layout } from "./Layout";

afterEach(() => cleanup());

describe("Layout", () => {
  it("renders the layout with header", () => {
    renderWithProviders(<Layout><div>Test Content</div></Layout>);
    expect(screen.getAllByText("Deployment Manager").length).toBeGreaterThan(0);
  });

  it("renders navigation items", () => {
    renderWithProviders(<Layout><div>Test Content</div></Layout>);
    expect(screen.getAllByText("Dashboard").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Profiles").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Analyze").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Deployments").length).toBeGreaterThan(0);
  });

  it("keeps phone navigation to four primary tabs and preserves all routes in the menu", () => {
    renderWithProviders(<Layout><div>Test Content</div></Layout>);

    const shell = screen.getByTestId("deployment-manager-app-shell");
    expect(shell).toHaveAttribute("data-mobile-nav", "menu-tabs");
    const tabs = screen.getByTestId("deployment-manager-app-shell-tabs");
    expect(tabs.querySelectorAll("[data-rcl-bottom-nav-item]")).toHaveLength(4);

    fireEvent.click(screen.getByTestId("deployment-manager-app-shell-menu"));
    const menu = screen.getByTestId("deployment-manager-app-shell-menu-panel");
    expect(menu.querySelectorAll("[data-rcl-app-shell-link]")).toHaveLength(8);
    expect(menu).toHaveTextContent("DashboardProfilesAnalyzeDeploymentsApprovalsEvidenceReleasesTelemetry");
  });

  it("renders children content", () => {
    renderWithProviders(<Layout><div>Test Content</div></Layout>);
    expect(screen.getByText("Test Content")).toBeDefined();
  });
});
