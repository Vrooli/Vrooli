/**
 * Settings surface — controls toggle preferences store, locale switcher
 * cycles supported locales, and the deferred backend sections render
 * their pending placeholders.
 */
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { selectors } from "../../../consts/selectors";
import { renderWithProviders } from "../../../test-utils";
import { Settings } from "../Settings";
import { usePreferencesStore } from "../../../shared/stores/preferencesStore";

beforeEach(() => {
  // Reset prefs to defaults before each test.
  usePreferencesStore.setState({
    theme: "dark",
    density: "comfortable",
    sidebarCollapsed: false,
    lastVisitedGoldenSlug: null,
  });
});

afterEach(() => {
  cleanup();
});

describe("Settings", () => {
  it("renders the surface heading", () => {
    renderWithProviders(<Settings />);
    expect(screen.getByTestId(selectors.settings.surface)).toBeInTheDocument();
  });

  it("toggles the theme via the dark/light buttons", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Settings />);
    expect(usePreferencesStore.getState().theme).toBe("dark");
    await user.click(screen.getByTestId(selectors.settings.themeLight));
    await waitFor(() => {
      expect(usePreferencesStore.getState().theme).toBe("light");
    });
  });

  it("toggles the density via the comfortable/compact buttons", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Settings />);
    await user.click(screen.getByTestId(selectors.settings.densityCompact));
    await waitFor(() => {
      expect(usePreferencesStore.getState().density).toBe("compact");
    });
  });

  it("toggles the sidebar-collapsed preference via the checkbox", async () => {
    const user = userEvent.setup();
    renderWithProviders(<Settings />);
    await user.click(screen.getByTestId(selectors.settings.sidebarCollapsed));
    await waitFor(() => {
      expect(usePreferencesStore.getState().sidebarCollapsed).toBe(true);
    });
  });

  it("renders the deferred-backend placeholder sections", () => {
    renderWithProviders(<Settings />);
    expect(screen.getByTestId(selectors.settings.catalogSyncPending)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.settings.watcherPending)).toBeInTheDocument();
  });
});
