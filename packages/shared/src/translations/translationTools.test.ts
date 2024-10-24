import { getTranslation } from "./translationTools";

describe("getTranslation", () => {
    const mockTranslations = [
        { language: "en", content: "Hello" },
        { language: "es", content: "Hola" },
        { language: "fr", content: "Bonjour" },
    ];

    it("should return the correct translation based on user language preference", () => {
        const obj = { translations: mockTranslations };
        const languages = ["es", "en"];
        expect(getTranslation(obj, languages)).toEqual({ language: "es", content: "Hola" });
    });

    it("should return the first translation if preferred language is not available and showAny is true", () => {
        const obj = { translations: mockTranslations };
        const languages = ["de"];
        expect(getTranslation(obj, languages)).toEqual({ language: "en", content: "Hello" });
    });

    it("should return an empty object if preferred language is not available and showAny is false", () => {
        const obj = { translations: mockTranslations };
        const languages = ["de"];
        expect(getTranslation(obj, languages, false)).toEqual({});
    });

    it("should handle null or undefined object input", () => {
        const languages = ["en"];
        expect(getTranslation(null, languages)).toEqual({});
        expect(getTranslation(undefined, languages)).toEqual({});
    });

    it("should handle null or undefined translations property", () => {
        const obj = { translations: null };
        const languages = ["en"];
        expect(getTranslation(obj, languages)).toEqual({});
    });

    it("should return an empty object for empty translations array", () => {
        const obj = { translations: [] };
        const languages = ["en"];
        expect(getTranslation(obj, languages)).toEqual({});
    });

    it("should handle case sensitivity in language codes", () => {
        const obj = { translations: mockTranslations };
        const languages = ["EN"];
        expect(getTranslation(obj, languages)).toEqual({ language: "en", content: "Hello" });
    });

    it("should handle empty languages array by returning first translation", () => {
        const obj = { translations: mockTranslations };
        const languages = [];
        expect(getTranslation(obj, languages)).toEqual({ language: "en", content: "Hello" });
    });

    it("should handle non-array translations", () => {
        const obj = { translations: "not an array" };
        const languages = ["en"];
        // @ts-ignore: Testing runtime scenario
        expect(getTranslation(obj, languages)).toEqual({});
    });
});