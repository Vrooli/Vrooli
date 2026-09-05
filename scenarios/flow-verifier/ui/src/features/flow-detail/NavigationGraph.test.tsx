import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen } from "@testing-library/react";

import { renderWithProviders } from "@vrooli/api-base/testing";

vi.mock("@xyflow/react/dist/style.css", () => ({}));
vi.mock("@xyflow/react", () => ({
  Background: () => null,
  Position: { Left: "left", Right: "right", Top: "top", Bottom: "bottom" },
  ReactFlow: () => <div data-testid="rf-mock" />,
}));

import { NavigationGraph } from "./NavigationGraph";
import type { NavigationStudioDescriptor } from "../../api/inventory";

const descriptor: NavigationStudioDescriptor = {
  renderer: "navigation-graph",
  routes: [
    { id: "home", path: "/", page: "HomePage", requires: "", parents: [] },
    { id: "settings", path: "/settings", page: "SettingsPage", requires: "", parents: [] },
  ],
  affordances: [
    {
      id: "nav_settings",
      to: "settings",
      showWhen: "",
      sideEffect: "",
      presentations: [{ in: "sidebar", label: "Settings", testId: "nav-settings" }],
    },
  ],
  containers: [
    { id: "sidebar", kind: "persistent", showWhen: "viewport=desktop", disclosure: "always_visible", hostRoutes: ["*"] },
    { id: "mobile_nav", kind: "persistent", showWhen: "viewport=mobile", disclosure: "always_visible", hostRoutes: ["*"] },
  ],
  contexts: [
    { name: "viewport", kind: "enum", values: ["desktop", "mobile"], defaultValue: "desktop" },
  ],
  invariants: [
    { id: "settings_reachable", passed: true, message: "settings reachable from /" },
  ],
};

afterEach(() => cleanup());

describe("NavigationGraph", () => {
  it("renders the toggle, container strip, and invariants", () => {
    renderWithProviders(<NavigationGraph descriptor={descriptor} />);
    expect(screen.getByTestId("navigation-graph")).toBeInTheDocument();
    expect(screen.getByTestId("nav-toggle-viewport")).toBeInTheDocument();
    // Default viewport=desktop shows sidebar, hides mobile_nav.
    expect(screen.getByTestId("nav-container-sidebar")).toBeInTheDocument();
    expect(screen.queryByTestId("nav-container-mobile_nav")).not.toBeInTheDocument();
    expect(screen.getByTestId("nav-invariant-settings_reachable")).toHaveAttribute(
      "data-passed",
      "true",
    );
  });

  it("filters containers when the viewport toggle changes to mobile", () => {
    renderWithProviders(<NavigationGraph descriptor={descriptor} />);
    fireEvent.change(screen.getByTestId("nav-toggle-viewport"), { target: { value: "mobile" } });
    expect(screen.queryByTestId("nav-container-sidebar")).not.toBeInTheDocument();
    expect(screen.getByTestId("nav-container-mobile_nav")).toBeInTheDocument();
  });

  it("shows the empty message when no routes are declared", () => {
    renderWithProviders(
      <NavigationGraph descriptor={{ ...descriptor, routes: [] }} />,
    );
    expect(screen.getByTestId("navigation-graph-empty")).toBeInTheDocument();
  });
});
