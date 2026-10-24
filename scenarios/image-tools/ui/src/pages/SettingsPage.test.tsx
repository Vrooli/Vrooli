/**
 * SettingsPage tests — the real settings surface. Covers the new
 * display/accessibility preference controls (font scale, reduced motion, text
 * direction, handedness): each renders as a radiogroup, reflects the active
 * choice, and drives the store on click (which writes to <html>). The embedded
 * ModelDefaultsCard's network calls are mocked so the page mounts cleanly.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { makeModelsMocks } from "../features/models/mocks/models";

vi.mock("../api/models", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/models")>();
  return { ...actual, ...makeModelsMocks() };
});

import { SettingsPage } from "./SettingsPage";
import { selectors } from "../consts/selectors";
import { setLocale } from "../i18n";

const root = () => document.documentElement;

const clearRootAttrs = () => {
  for (const attr of ["data-font-scale", "data-reduced-motion", "data-handedness"]) {
    root().removeAttribute(attr);
  }
};

describe("SettingsPage display preferences", () => {
  beforeEach(async () => {
    await setLocale("en");
    window.localStorage.clear();
    clearRootAttrs();
  });

  afterEach(async () => {
    cleanup();
    vi.clearAllMocks();
    clearRootAttrs();
    await setLocale("en");
  });

  it("renders the four display preference radiogroups", () => {
    renderWithProviders(<SettingsPage />);
    // The default-selected option in each new group exists and is checked.
    expect(screen.getByTestId(selectors.settingsPage.fontScaleOption({ choice: "default" }))).toHaveAttribute(
      "aria-checked",
      "true",
    );
    expect(
      screen.getByTestId(selectors.settingsPage.reducedMotionOption({ choice: "system" })),
    ).toHaveAttribute("aria-checked", "true");
    expect(
      screen.getByTestId(selectors.settingsPage.textDirectionOption({ choice: "auto" })),
    ).toHaveAttribute("aria-checked", "true");
    expect(screen.getByTestId(selectors.settingsPage.handednessOption({ choice: "right" }))).toHaveAttribute(
      "aria-checked",
      "true",
    );
  });

  it("applies a font-scale choice to <html> and persists it", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);

    await user.click(screen.getByTestId(selectors.settingsPage.fontScaleOption({ choice: "large" })));

    await waitFor(() => {
      expect(root().getAttribute("data-font-scale")).toBe("large");
    });
    expect(
      screen.getByTestId(selectors.settingsPage.fontScaleOption({ choice: "large" })),
    ).toHaveAttribute("aria-checked", "true");
  });

  it("forces reduced motion on via the Always choice", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);

    await user.click(
      screen.getByTestId(selectors.settingsPage.reducedMotionOption({ choice: "always" })),
    );

    await waitFor(() => {
      expect(root().getAttribute("data-reduced-motion")).toBe("always");
    });
  });

  it("forces RTL text direction via the explicit choice", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);

    await user.click(
      screen.getByTestId(selectors.settingsPage.textDirectionOption({ choice: "rtl" })),
    );

    await waitFor(() => {
      expect(root().dir).toBe("rtl");
    });
  });

  it("sets the handedness attribute when switching to left-handed", async () => {
    const user = userEvent.setup();
    renderWithProviders(<SettingsPage />);

    await user.click(screen.getByTestId(selectors.settingsPage.handednessOption({ choice: "left" })));

    await waitFor(() => {
      expect(root().getAttribute("data-handedness")).toBe("left");
    });
  });

  it("keeps the existing theme and locale controls", () => {
    renderWithProviders(<SettingsPage />);
    expect(screen.getByTestId(selectors.settingsPage.themeOption({ choice: "light" }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.settingsPage.localeOption({ code: "en" }))).toBeInTheDocument();
  });
});
