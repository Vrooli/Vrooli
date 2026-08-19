import { describe, it, expect, vi, beforeAll, beforeEach, afterEach } from "vitest";
import { cleanup, screen, act, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

import { renderWithProviders } from "../test-utils";
import { strings } from "../consts/strings";
import { AppShell } from "./AppShell";
import { PreferencesProvider } from "../hooks/usePreferences";

vi.mock("../api/health", () => ({
  fetchHealth: vi.fn(),
}));

vi.mock("../api/healthStatus", () => ({
  getProviderHealth: vi.fn(),
}));

import { fetchHealth } from "../api/health";
import { getProviderHealth } from "../api/healthStatus";

// jsdom doesn't implement window.matchMedia — PreferencesProvider needs it
beforeAll(() => {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockReturnValue({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }),
  });
});

const routerFuture = { v7_startTransition: true, v7_relativeSplatPath: true } as const;

beforeEach(() => {
  vi.mocked(fetchHealth).mockImplementation(() => new Promise(() => {}));
  vi.mocked(getProviderHealth).mockImplementation(() => new Promise(() => {}));
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function renderShell(initialPath = "/") {
  return renderWithProviders(
    <MemoryRouter initialEntries={[initialPath]} future={routerFuture}>
      <PreferencesProvider>
        <AppShell />
      </PreferencesProvider>
    </MemoryRouter>,
    { withoutRouter: true },
  );
}

describe("AppShell", () => {
  it("renders the main content area", () => {
    renderShell();
    expect(screen.getByRole("main", { name: strings.shell.mainContent })).toBeInTheDocument();
  });

  it("renders the primary nav (sidebar)", () => {
    renderShell();
    expect(screen.getByRole("complementary", { name: strings.shell.primaryNav })).toBeInTheDocument();
  });

  it("renders the top bar header", () => {
    renderShell();
    expect(screen.getByRole("banner")).toBeInTheDocument();
  });

  it("renders the settings button in the top bar", () => {
    renderShell();
    expect(screen.getByRole("button", { name: strings.shell.openSettings })).toBeInTheDocument();
  });

  it("opens settings drawer when settings button is clicked", () => {
    renderShell();
    act(() => {
      fireEvent.click(screen.getByRole("button", { name: strings.shell.openSettings }));
    });
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("closes settings drawer when Escape is pressed while open", () => {
    renderShell();
    act(() => {
      fireEvent.click(screen.getByRole("button", { name: strings.shell.openSettings }));
    });
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    act(() => {
      window.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));
    });
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("toggles settings drawer with Ctrl+, keyboard shortcut", () => {
    renderShell();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    act(() => {
      window.dispatchEvent(new KeyboardEvent("keydown", { key: ",", ctrlKey: true, bubbles: true }));
    });
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    act(() => {
      window.dispatchEvent(new KeyboardEvent("keydown", { key: ",", ctrlKey: true, bubbles: true }));
    });
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
});
