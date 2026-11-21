import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import { renderWithProviders } from "./test-utils";

vi.mock("./app/routes", async () => {
  const { useTheme } = await vi.importActual<typeof import("./theme/ThemeProvider")>("./theme/ThemeProvider");
  return {
    AppRouter: function AppRouterProbe() {
      const { resolved } = useTheme();
      return <p role="status">Active theme: {resolved}</p>;
    },
  };
});

import App from "./App";

describe("App composition", () => {
  afterEach(() => {
    cleanup();
    localStorage.clear();
    document.documentElement.removeAttribute("data-resolved-theme");
    document.documentElement.removeAttribute("data-theme");
  });

  it("mounts the router inside the app providers", () => {
    localStorage.setItem("vrooli.theme", "dark");

    renderWithProviders(<App />);

    expect(screen.getByText("Active theme: dark")).toHaveAttribute("role", "status");
    expect(document.documentElement).toHaveAttribute("data-resolved-theme", "dark");
  });
});
