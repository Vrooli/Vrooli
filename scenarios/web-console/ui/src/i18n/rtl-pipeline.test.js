/**
 * RTL pipeline regression test.
 *
 * Proves that the `setLocale → languageChanged → applyDocumentLocale` chain
 * actually mutates `<html lang>` and `<html dir>`. Without this assertion the
 * rtl branch of `LOCALE_CONFIG` would be unexercised: every component test
 * runs in cimode and never round-trips through real catalogs, so a broken
 * applyDocumentLocale would ship silently.
 *
 * Direction is stateful — we explicitly cover the ltr→rtl→ltr round-trip so
 * a "set once, forget to flip back" regression in the languageChanged handler
 * fails here.
 */
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { setLocale } from "./index";
const STORAGE_KEY = "vrooli.locale";
describe("i18n RTL pipeline", () => {
    beforeEach(async () => {
        window.localStorage.clear();
        await setLocale("en");
    });
    afterEach(async () => {
        await setLocale("en");
    });
    it("starts in English with ltr direction", () => {
        expect(document.documentElement.lang).toBe("en");
        expect(document.documentElement.dir).toBe("ltr");
    });
    it("flips <html dir> to rtl when switching to Arabic", async () => {
        await setLocale("ar");
        expect(document.documentElement.lang).toBe("ar");
        expect(document.documentElement.dir).toBe("rtl");
    });
    it("flips <html dir> back to ltr when returning from an RTL locale", async () => {
        await setLocale("ar");
        expect(document.documentElement.dir).toBe("rtl");
        await setLocale("en");
        expect(document.documentElement.dir).toBe("ltr");
        await setLocale("ja");
        expect(document.documentElement.lang).toBe("ja");
        expect(document.documentElement.dir).toBe("ltr");
    });
    it("persists the chosen locale to localStorage for returning visits", async () => {
        await setLocale("ar");
        expect(window.localStorage.getItem(STORAGE_KEY)).toBe("ar");
        await setLocale("ja");
        expect(window.localStorage.getItem(STORAGE_KEY)).toBe("ja");
    });
});
