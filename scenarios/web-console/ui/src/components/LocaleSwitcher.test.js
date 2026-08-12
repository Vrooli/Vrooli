import { jsx as _jsx } from "react/jsx-runtime";
/**
 * LocaleSwitcher tests — verifies the switcher both renders a toggle for
 * every supported locale and drives the full pipeline (setLocale →
 * languageChanged → applyDocumentLocale → <html dir>) end-to-end.
 *
 * Tests opt into real catalogs (not cimode) because cimode short-circuits
 * the catalog lookup and leaves applyDocumentLocale unexercised — the
 * whole point of having an `ar` catalog is to prove the rtl branch fires.
 */
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { screen, waitFor, cleanup } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import LocaleSwitcher from "./LocaleSwitcher";
import { renderWithProviders } from "../test-utils";
import { setLocale, LOCALE_CODES } from "../i18n";
import { selectors } from "../consts/selectors";
// See LocaleSwitcher.tsx for the rationale. Selectors registry's widened
// index-signature types trip `noUncheckedIndexedAccess`; narrow once.
const localeSelectors = selectors.locale;
describe("LocaleSwitcher", () => {
    beforeEach(async () => {
        window.localStorage.clear();
        await setLocale("en");
    });
    afterEach(() => {
        cleanup();
    });
    it("renders a toggle for every supported locale with its native label", () => {
        renderWithProviders(_jsx(LocaleSwitcher, {}));
        expect(screen.getByTestId(localeSelectors.switcher)).toBeInTheDocument();
        for (const code of LOCALE_CODES) {
            expect(screen.getByTestId(localeSelectors.toggle({ code }))).toBeInTheDocument();
        }
        // Native labels are rendered untranslated so users can self-recover from
        // landing in a locale they cannot read.
        expect(screen.getByText("English")).toBeInTheDocument();
        expect(screen.getByText("日本語")).toBeInTheDocument();
        expect(screen.getByText("العربية")).toBeInTheDocument();
    });
    it("marks the active locale's toggle as pressed", () => {
        renderWithProviders(_jsx(LocaleSwitcher, {}));
        expect(screen.getByTestId(localeSelectors.toggle({ code: "en" }))).toHaveAttribute("aria-pressed", "true");
        expect(screen.getByTestId(localeSelectors.toggle({ code: "ja" }))).toHaveAttribute("aria-pressed", "false");
    });
    it("switches locale and flips <html lang> when a toggle is clicked", async () => {
        const user = userEvent.setup();
        renderWithProviders(_jsx(LocaleSwitcher, {}));
        await user.click(screen.getByTestId(localeSelectors.toggle({ code: "ja" })));
        await waitFor(() => {
            expect(document.documentElement.lang).toBe("ja");
        });
        expect(document.documentElement.dir).toBe("ltr");
        expect(window.localStorage.getItem("vrooli.locale")).toBe("ja");
    });
    it("flips <html dir> to rtl when the Arabic toggle is clicked", async () => {
        const user = userEvent.setup();
        renderWithProviders(_jsx(LocaleSwitcher, {}));
        await user.click(screen.getByTestId(localeSelectors.toggle({ code: "ar" })));
        await waitFor(() => {
            expect(document.documentElement.dir).toBe("rtl");
        });
        expect(document.documentElement.lang).toBe("ar");
    });
    it("flips <html dir> back to ltr when returning from an RTL locale", async () => {
        const user = userEvent.setup();
        renderWithProviders(_jsx(LocaleSwitcher, {}));
        await user.click(screen.getByTestId(localeSelectors.toggle({ code: "ar" })));
        await waitFor(() => expect(document.documentElement.dir).toBe("rtl"));
        await user.click(screen.getByTestId(localeSelectors.toggle({ code: "en" })));
        await waitFor(() => expect(document.documentElement.dir).toBe("ltr"));
    });
});
