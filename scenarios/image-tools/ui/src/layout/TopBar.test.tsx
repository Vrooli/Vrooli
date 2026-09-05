/**
 * TopBar tests — the title, the locale switcher, and the theme select. The
 * shell wires TopBar into the chrome (covered structurally in
 * `AppShell.test.tsx`); this file drives the two interactive seams directly so
 * the locale `onClick` and theme `onChange` handlers are exercised, not just
 * rendered.
 */
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { getCurrentLocale, setLocale } from "../i18n";
import { TopBar } from "./TopBar";

describe("TopBar", () => {
  beforeEach(async () => {
    await setLocale("en");
    document.documentElement.removeAttribute("data-theme");
  });

  afterEach(() => {
    cleanup();
  });

  it("renders the title and the locale + theme controls", () => {
    renderWithProviders(<TopBar />, { initialTheme: "light" });
    expect(screen.getByTestId(selectors.app.title)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.locale.switcher)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.theme.switcher)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.theme.select)).toBeInTheDocument();
  });

  it("renders a pressed-aware toggle for every supported locale", () => {
    renderWithProviders(<TopBar />, { initialTheme: "light" });
    const en = screen.getByTestId(selectors.locale.toggle({ code: "en" }));
    const ja = screen.getByTestId(selectors.locale.toggle({ code: "ja" }));
    expect(en).toBeInTheDocument();
    expect(ja).toBeInTheDocument();
    // The current locale toggle reflects aria-pressed; the others don't.
    expect(en).toHaveAttribute("aria-pressed", "true");
    expect(ja).toHaveAttribute("aria-pressed", "false");
  });

  it("switching locale via a toggle updates the active language", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TopBar />, { initialTheme: "light" });
    await user.click(screen.getByTestId(selectors.locale.toggle({ code: "ja" })));
    await waitFor(() => {
      expect(getCurrentLocale()).toBe("ja");
    });
  });

  it("choosing a theme from the select applies data-theme on <html>", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TopBar />, { initialTheme: "light" });
    const select = screen.getByTestId<HTMLSelectElement>(selectors.theme.select);
    expect(select.value).toBe("light");

    await user.selectOptions(select, "dark");
    expect(select.value).toBe("dark");
    await waitFor(() => {
      expect(document.documentElement.getAttribute("data-theme")).toBe("dark");
    });

    // `system` clears the attribute so the CSS @media fallback owns resolution.
    await user.selectOptions(select, "system");
    await waitFor(() => {
      expect(document.documentElement.hasAttribute("data-theme")).toBe(false);
    });
  });
});
