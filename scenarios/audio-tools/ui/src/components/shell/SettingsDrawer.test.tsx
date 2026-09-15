import { describe, it, expect, vi, beforeAll, afterEach } from "vitest";
import { cleanup, screen, act, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

import { renderWithProviders } from "../../test-utils";
import { strings } from "../../consts/strings";
import { SettingsDrawer } from "./SettingsDrawer";
import { PreferencesProvider } from "../../hooks/usePreferences";

// jsdom doesn't implement window.matchMedia — PreferencesProvider calls it on init
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

afterEach(cleanup);

const routerFuture = { v7_startTransition: true, v7_relativeSplatPath: true } as const;

function renderDrawer(open: boolean, onClose = vi.fn()) {
  return renderWithProviders(
    <MemoryRouter future={routerFuture}>
      <PreferencesProvider>
        <SettingsDrawer open={open} onClose={onClose} />
      </PreferencesProvider>
    </MemoryRouter>,
    { withoutRouter: true },
  );
}

describe("SettingsDrawer", () => {
  describe("when closed", () => {
    it("renders nothing when open=false", () => {
      const { container } = renderDrawer(false);
      expect(container).toBeEmptyDOMElement();
    });

    it("does not register Escape listener when closed", () => {
      const onClose = vi.fn();
      renderDrawer(false, onClose);
      window.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));
      expect(onClose).not.toHaveBeenCalled();
    });
  });

  describe("when open", () => {
    it("renders the dialog when open=true", () => {
      renderDrawer(true);
      expect(screen.getByRole("dialog")).toBeInTheDocument();
    });

    it("renders the settings title", () => {
      renderDrawer(true);
      expect(screen.getAllByText(strings.shell.settingsTitle).length).toBeGreaterThan(0);
    });

    it("has aria-modal=true", () => {
      renderDrawer(true);
      expect(screen.getByRole("dialog")).toHaveAttribute("aria-modal", "true");
    });

    it("renders close button", () => {
      renderDrawer(true);
      expect(screen.getByRole("button", { name: strings.shell.closeSettings })).toBeInTheDocument();
    });

    it("calls onClose when close button is clicked", () => {
      const onClose = vi.fn();
      renderDrawer(true, onClose);
      act(() => {
        fireEvent.click(screen.getByRole("button", { name: strings.shell.closeSettings }));
      });
      expect(onClose).toHaveBeenCalledTimes(1);
    });

    it("calls onClose when backdrop is clicked", () => {
      const onClose = vi.fn();
      renderDrawer(true, onClose);
      const backdrop = document.querySelector("[aria-hidden='true']");
      expect(backdrop).toBeTruthy();
      act(() => {
        fireEvent.click(backdrop as Element);
      });
      expect(onClose).toHaveBeenCalledTimes(1);
    });

    it("calls onClose when Escape key is pressed", () => {
      const onClose = vi.fn();
      renderDrawer(true, onClose);
      window.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));
      expect(onClose).toHaveBeenCalledTimes(1);
    });

    it("renders theme select field", () => {
      renderDrawer(true);
      expect(screen.getByRole("combobox", { name: strings.settings.themeLabel })).toBeInTheDocument();
    });

    it("renders font scale select field", () => {
      renderDrawer(true);
      expect(screen.getByRole("combobox", { name: strings.settings.fontScaleLabel })).toBeInTheDocument();
    });

    it("renders reduced motion checkbox", () => {
      renderDrawer(true);
      expect(screen.getByRole("checkbox")).toBeInTheDocument();
    });

    it("renders locale switcher group", () => {
      renderDrawer(true);
      expect(screen.getByRole("group", { name: strings.locale.switcherLabel })).toBeInTheDocument();
    });

    it("changes theme when select is changed", () => {
      renderDrawer(true);
      const select = screen.getByRole("combobox", { name: strings.settings.themeLabel });
      act(() => {
        fireEvent.change(select, { target: { value: "dark" } });
      });
      expect((select as HTMLSelectElement).value).toBe("dark");
    });
  });
});
