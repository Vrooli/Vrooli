import { afterEach, describe, expect, it } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { AppNavigation } from "@vrooli/react-component-library/AppNavigation/1";
import { NavigationTree } from "@vrooli/react-component-library/NavigationTree/1";
import { NavLink } from "@vrooli/react-component-library/NavLink/1";

describe("adopted navigation assets", () => {
  afterEach(() => cleanup());

  it("renders the default desktop navigation and marks the current route", () => {
    renderWithProviders(<AppNavigation />);

    expect(document.querySelector("[data-rcl-app-navigation]")).toHaveAttribute(
      "data-viewport-mode",
      "desktop",
    );
    expect(screen.getByRole("link", { name: "Home" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("link", { name: "Library" })).toHaveAttribute("href", "/library");
  });

  it("supports mobile mode and caller-owned navigation content", () => {
    renderWithProviders(
      <AppNavigation mode="mobile" brand="Workspace">
        <ul data-testid="custom-navigation">
          <li>
            <a href="/custom">Custom</a>
          </li>
        </ul>
      </AppNavigation>,
    );

    expect(screen.getByText("Workspace")).toBeInTheDocument();
    expect(screen.getByTestId("custom-navigation")).toBeInTheDocument();
    expect(document.querySelector("[data-rcl-app-navigation]")).toHaveAttribute(
      "data-viewport-mode",
      "mobile",
    );
  });

  it("renders the default tree and allows a composed tree slot", () => {
    const { rerender } = renderWithProviders(<NavigationTree title="Assets" />);

    expect(screen.getByRole("link", { name: "Overview" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("link", { name: "Activity" })).not.toHaveAttribute(
      "aria-current",
      "page",
    );

    rerender(
      <NavigationTree title="Assets" items={[]}>
        <div data-testid="custom-tree">Inventory tree</div>
      </NavigationTree>,
    );
    expect(screen.getByTestId("custom-tree")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Overview" })).not.toBeInTheDocument();
  });

  it("renders NavLink state, optional icon, and description", () => {
    renderWithProviders(
      <NavLink label="Settings" current description="Open settings" icon={<span>⚙</span>} />,
    );

    const link = screen.getByRole("link", { name: "Settings" });
    expect(link).toHaveAttribute("href", "/");
    expect(link).toHaveAttribute("aria-current", "page");
    expect(link).toHaveAttribute("title", "Open settings");
    expect(screen.getByText("⚙")).toBeInTheDocument();
  });
});
