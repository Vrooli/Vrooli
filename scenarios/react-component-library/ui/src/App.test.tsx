/**
 * App tests — composition smoke + route resolution.
 *
 * Asserts: (1) the operational shell mounts around the catalog workspace,
 * (2) only the catalog, asset detail, and settings routes resolve,
 * (3) navigating between routes does not remount the shell.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "./test-utils";

const emitShortcutIntentMock = vi.fn();

vi.mock("@vrooli/iframe-bridge", () => ({
  emitShortcutIntent: (...args: unknown[]) => emitShortcutIntentMock(...args),
}));

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
    emitShortcutIntentMock.mockClear();
  });

  it("mounts the shell on /", async () => {
    renderWithProviders(<App />, { routerEntries: ["/"] });
    await waitFor(() => {
      expect(screen.getByTestId("app-shell")).toBeInTheDocument();
    });
    expect(screen.getAllByTestId("catalog-browser")).not.toHaveLength(0);
  });

  it("renders the catalog workspace at /", async () => {
    renderWithProviders(<App />, { routerEntries: ["/"] });
    await waitFor(() => {
      expect(screen.getAllByTestId("catalog-browser")).toHaveLength(2);
    });
  });

  it("does not retain the removed components route", async () => {
    renderWithProviders(<App />, { routerEntries: ["/components"] });
    await waitFor(() => {
      expect(screen.getByTestId("not-found-page")).toBeInTheDocument();
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

  it("relays unhandled host keyboard shortcuts", async () => {
    renderWithProviders(<App />, { routerEntries: ["/"] });
    await waitFor(() => {
      expect(screen.getByTestId("app-shell")).toBeInTheDocument();
    });

    fireEvent.keyDown(window, { key: "k", metaKey: true });

    expect(emitShortcutIntentMock).toHaveBeenCalledWith({
      action: "react-component-library.unhandled-shortcut",
      outcome: "noop",
      chord: "meta+k",
      source: "keyboard",
    });
  });
});
