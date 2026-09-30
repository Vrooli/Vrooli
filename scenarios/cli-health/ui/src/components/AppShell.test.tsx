/**
 * AppShell tests — focused on the shell's locale switcher and the
 * end-to-end i18n pipeline (catalogs → DOM, persistence, html attrs).
 *
 * Tests opt into real locales because they verify that the
 * `setLocale → applyDocumentLocale → languageChanged` chain flips
 * `<html lang>` and `<html dir>` correctly. cimode would short-circuit
 * the catalog lookup and leave those attributes unverified.
 */
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { AppShell } from "./AppShell";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { setLocale } from "../i18n";
import ar from "../i18n/locales/ar.json";
import en from "../i18n/locales/en.json";
import ja from "../i18n/locales/ja.json";

describe("AppShell rendering (cimode)", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders the title element via its test id", () => {
    renderWithProviders(<AppShell><div /></AppShell>);
    expect(screen.getByTestId(selectors.app.title)).toBeInTheDocument();
  });

  it("renders translation keys for the app surface (cimode echoes keys)", () => {
    renderWithProviders(<AppShell><div /></AppShell>);
    expect(screen.getByText(strings.app.eyebrow)).toBeInTheDocument();
    expect(screen.getByText(strings.app.description)).toBeInTheDocument();
  });

  it("renders the locale switcher with toggles for every supported locale", () => {
    renderWithProviders(<AppShell><div /></AppShell>);
    expect(screen.getByTestId(selectors.locale.switcher)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.locale.toggle({ code: "en" }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.locale.toggle({ code: "ja" }))).toBeInTheDocument();
  });

  it("renders children in the slot region", () => {
    renderWithProviders(
      <AppShell>
        <div data-testid="shell-child">slotted</div>
      </AppShell>,
    );
    expect(screen.getByTestId("shell-child")).toBeInTheDocument();
  });
});

describe("AppShell locale switching (real locales)", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
  });

  it("renders English copy by default and reflects it on <html>", async () => {
    renderWithProviders(<AppShell><div /></AppShell>);
    expect(await screen.findByText(en.app.eyebrow)).toBeInTheDocument();
    expect(screen.getByText(en.app.description)).toBeInTheDocument();
    expect(document.documentElement.lang).toBe("en");
    expect(document.documentElement.dir).toBe("ltr");
  });

  it("switches to Japanese when the 日本語 toggle is clicked", async () => {
    const user = userEvent.setup();
    renderWithProviders(<AppShell><div /></AppShell>);
    await user.click(screen.getByTestId(selectors.locale.toggle({ code: "ja" })));

    await waitFor(() => {
      expect(screen.getByText(ja.app.eyebrow)).toBeInTheDocument();
    });
    expect(screen.getByText(ja.app.description)).toBeInTheDocument();
    expect(document.documentElement.lang).toBe("ja");
  });

  it("flips <html dir> to rtl when an RTL locale (ar) is chosen", async () => {
    // Whole point of ar in the template: prove the LTR→RTL pipeline works
    // end-to-end. Without this assertion, the rtl branch of LOCALE_CONFIG
    // would be unexercised.
    const user = userEvent.setup();
    renderWithProviders(<AppShell><div /></AppShell>);
    await user.click(screen.getByTestId(selectors.locale.toggle({ code: "ar" })));

    await waitFor(() => {
      expect(screen.getByText(ar.app.eyebrow)).toBeInTheDocument();
    });
    expect(document.documentElement.lang).toBe("ar");
    expect(document.documentElement.dir).toBe("rtl");
  });

  it("flips <html dir> back to ltr when returning to a non-RTL locale", async () => {
    // Direction is a stateful attribute; an rtl→ltr round-trip catches the
    // failure mode where applyDocumentLocale only sets dir once.
    const user = userEvent.setup();
    renderWithProviders(<AppShell><div /></AppShell>);
    await user.click(screen.getByTestId(selectors.locale.toggle({ code: "ar" })));
    await waitFor(() => {
      expect(document.documentElement.dir).toBe("rtl");
    });

    await user.click(screen.getByTestId(selectors.locale.toggle({ code: "en" })));
    await waitFor(() => {
      expect(document.documentElement.dir).toBe("ltr");
    });
  });

  it("persists the chosen locale to localStorage so returning visits restore it", async () => {
    const user = userEvent.setup();
    renderWithProviders(<AppShell><div /></AppShell>);
    await user.click(screen.getByTestId(selectors.locale.toggle({ code: "ja" })));

    await waitFor(() => {
      expect(window.localStorage.getItem("vrooli.locale")).toBe("ja");
    });
  });

  it("marks the active locale's toggle as pressed", async () => {
    const user = userEvent.setup();
    renderWithProviders(<AppShell><div /></AppShell>);

    expect(screen.getByTestId(selectors.locale.toggle({ code: "en" }))).toHaveAttribute(
      "aria-pressed",
      "true",
    );

    await user.click(screen.getByTestId(selectors.locale.toggle({ code: "ja" })));

    await waitFor(() => {
      expect(
        screen.getByTestId(selectors.locale.toggle({ code: "ja" })),
      ).toHaveAttribute("aria-pressed", "true");
    });
  });
});
