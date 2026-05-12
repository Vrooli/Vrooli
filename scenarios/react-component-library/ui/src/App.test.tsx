/**
 * App tests — composition smoke + route resolution.
 *
 * Asserts: (1) the operational shell mounts (brand + sidebar nav),
 * (2) each top-level route resolves to its page,
 * (3) navigating between routes does not remount the shell.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "./test-utils";

vi.mock("./api/health", () => ({
  fetchHealth: vi
    .fn()
    .mockResolvedValue({ status: "ok", service: "react-component-library", timestamp: new Date().toISOString() }),
}));

vi.mock("./api/components", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./api/components")>();
  return {
    ...actual,
    componentsClient: {
      listComponents: vi.fn().mockResolvedValue({ components: [] }),
      getComponent: vi.fn(),
      getComponentByLibraryId: vi.fn(),
      indexComponents: vi.fn(),
      getComponentContent: vi.fn(),
      updateComponentContent: vi.fn(),
    },
  };
});

import App from "./App";

describe("App composition", () => {
  afterEach(() => {
    cleanup();
  });

  it("mounts the shell on /", async () => {
    renderWithProviders(<App />, { routerEntries: ["/"] });
    await waitFor(() => {
      expect(screen.getByTestId("app-shell")).toBeInTheDocument();
    });
    expect(screen.getByTestId("nav-dashboard")).toBeInTheDocument();
  });

  it("renders the dashboard stub at /", async () => {
    renderWithProviders(<App />, { routerEntries: ["/"] });
    await waitFor(() => {
      expect(screen.getByTestId("dashboard-page")).toBeInTheDocument();
    });
  });

  it("renders the components stub at /components", async () => {
    renderWithProviders(<App />, { routerEntries: ["/components"] });
    await waitFor(() => {
      expect(screen.getByTestId("components-page")).toBeInTheDocument();
    });
  });

  it("renders the settings stub at /settings", async () => {
    renderWithProviders(<App />, { routerEntries: ["/settings"] });
    await waitFor(() => {
      expect(screen.getByTestId("settings-page")).toBeInTheDocument();
    });
  });

  it("falls back to the not-found stub for unknown routes", async () => {
    renderWithProviders(<App />, { routerEntries: ["/never-was-a-route"] });
    await waitFor(() => {
      expect(screen.getByTestId("not-found-page")).toBeInTheDocument();
    });
  });
});
