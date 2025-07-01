// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-19
// AI_CHECK: TYPE_SAFETY=eliminated-51-ts-ignore-comments | LAST: 2025-06-28
import { describe, it, expect, vi, beforeEach } from "vitest";

// Mock modules before importing them
vi.mock("i18next");
vi.mock("react-i18next");

import { generatePK, type CommonKey, type Session } from "@vrooli/shared";
import { type FieldInputProps, type FieldMetaProps } from "formik";
import i18next from "i18next";
import * as yup from "yup";
import { addEmptyTranslation, combineErrorsWithTranslations, getFormikErrorsWithTranslations, getLanguageSubtag, getPreferredLanguage, getShortenedLabel, getTranslationData, getUserLanguages, getUserLocale, handleTranslationChange, loadLocale, removeTranslation, translateSnackMessage, updateTranslation, updateTranslationFields, type TranslationObject } from "./translationTools.js";

// Mocks for navigator.language and navigator.languages
function mockNavigatorLanguage(language) {
    Object.defineProperty(global.navigator, "language", {
        value: language,
        writable: true,
    });
}
function mockNavigatorLanguages(languages) {
    Object.defineProperty(global.navigator, "languages", {
        value: languages,
        writable: true,
    });
}
// Cast the i18next.t function to the Jest Mock type
// const mockedTranslate = i18next.t as unknown as ReturnType<typeof vi.fn>;

// Utility function for creating a session object
function createSession(languages: string[] | null | undefined) {
    return {
        __typename: "Session",
        isLoggedIn: true,
        users: [{
            __typename: "SessionUser",
            id: generatePK().toString(),
            languages,
        }],
    } as unknown as Session;
}

describe("loadLocale", () => {
    const localeTestCases = [
        { input: "de", expected: "de", description: "load the specified valid locale" },
        { input: "unknown-locale", expected: "en-US", description: "fallback to en-US when an unknown locale is requested" },
        { input: undefined, expected: "en-US", description: "load en-US by default when no locale is specified" },
        { input: "en-GB", expected: "en-GB", description: "handle the loading of locales with region codes" },
        { input: "en-ZZ", expected: "en-US", description: "pick default region code when requested one doesn't exist" },
    ];

    it.each(localeTestCases)("should $description", async ({ input, expected }) => {
        const locale = await loadLocale(input as string | undefined);
        expect(locale.code).toEqual(expected);
    });
});

describe("updateTranslationFields", () => {
    const mockTranslations = [
        { id: generatePK().toString(), language: "en", content: "Hello", note: "Greeting" },
        { id: generatePK().toString(), language: "es", content: "Hola", note: "Saludo" },
    ];

    it("should correctly update fields for an existing translation", () => {
        const language = "en";
        const changes = { content: "Hi", additional: "Extra info" };
        const updatedTranslations = updateTranslationFields({ translations: mockTranslations }, language, changes);
        const updatedTranslation = updatedTranslations.find(t => t.language === language);
        expect(updatedTranslation).toMatchObject({ ...changes });
    });

    it("should add a new translation if the specified language is not found", () => {
        const language = "fr";
        const changes = { content: "Bonjour" };
        const updatedTranslations = updateTranslationFields({ translations: mockTranslations }, language, changes);
        const newTranslation = updatedTranslations.find(t => t.language === language);
        expect(newTranslation).toMatchObject({ language, ...changes });
        expect(newTranslation?.id).toBeDefined();
    });

    it("should preserve existing fields that are not being updated", () => {
        const language = "en";
        const changes = { additional: "Extra info" };
        const updatedTranslations = updateTranslationFields({ translations: mockTranslations }, language, changes);
        const updatedTranslation = updatedTranslations.find(t => t.language === language);
        expect(updatedTranslation).toMatchObject({ ...mockTranslations[0], ...changes });
    });

    it("should return an array with a new translation if given null or undefined object", () => {
        const language = "en";
        const changes = { content: "Hi" };
        const resultFromNull = updateTranslationFields(null, language, changes);
        const resultFromUndefined = updateTranslationFields(undefined, language, changes);

        // Check that both results are arrays with a single object
        expect(resultFromNull).toHaveLength(1);
        expect(resultFromUndefined).toHaveLength(1);

        // Check that the single object in each array has the correct language and content
        expect(resultFromNull[0]).toMatchObject({ language, ...changes });
        expect(resultFromUndefined[0]).toMatchObject({ language, ...changes });

        // Check that each new translation has a unique id
        expect(resultFromNull[0].id).toBeDefined();
        expect(resultFromUndefined[0].id).toBeDefined();
    });

    it("should handle an empty translations array", () => {
        const language = "en";
        const changes = { content: "Hi" };
        expect(updateTranslationFields({ translations: [] }, language, changes).length).toBe(1);
    });

    it("should handle null or undefined translations property", () => {
        const language = "en";
        const changes = { content: "Hi" };
        expect(updateTranslationFields({ translations: null }, language, changes).length).toBe(1);
        expect(updateTranslationFields({ translations: undefined }, language, changes).length).toBe(1);
    });

    it("should allow removal of a field by setting it to null", () => {
        const language = "en";
        const changes = { note: null };
        const updatedTranslations = updateTranslationFields({ translations: mockTranslations }, language, changes);
        const updatedTranslation = updatedTranslations.find(t => t.language === language) as Record<string, unknown>;
        expect(updatedTranslation?.note).toBeNull();
    });

    it("should not update fields if changes object is empty", () => {
        const language = "en";
        const changes = {};
        const updatedTranslations = updateTranslationFields({ translations: mockTranslations }, language, changes);
        expect(updatedTranslations).toEqual(mockTranslations);
    });

    it("should not update fields if changes object is null or undefined", () => {
        const language = "en";
        const updatedTranslationsNullChanges = updateTranslationFields({ translations: mockTranslations }, language, null as any);
        const updatedTranslationsUndefinedChanges = updateTranslationFields({ translations: mockTranslations }, language, undefined as any);
        expect(updatedTranslationsNullChanges).toEqual(mockTranslations);
        expect(updatedTranslationsUndefinedChanges).toEqual(mockTranslations);
    });
});

describe("updateTranslation", () => {
    const mockTranslations = [
        { id: generatePK().toString(), language: "en", content: "Hello" },
        { id: generatePK().toString(), language: "es", content: "Hola" },
    ];

    it("should correctly update an existing translation", () => {
        const newTranslation = { id: mockTranslations[0].id, language: "en", content: "Hi" };
        const updatedTranslations = updateTranslation({ translations: mockTranslations }, newTranslation);
        const updatedTranslation = updatedTranslations.find(t => t.language === "en");
        expect(updatedTranslation).toEqual(newTranslation);
    });

    it("should add a new translation if the specified language is not found", () => {
        const newTranslation = { id: generatePK().toString(), language: "fr", content: "Bonjour" };
        const updatedTranslations = updateTranslation({ translations: mockTranslations }, newTranslation);
        const addedTranslation = updatedTranslations.find(t => t.language === "fr");
        expect(addedTranslation).toEqual(newTranslation);
    });

    it("should return the original translations if the translation object is empty", () => {
        const newTranslation = {} as any;
        const updatedTranslations = updateTranslation({ translations: mockTranslations }, newTranslation);
        expect(updatedTranslations).toEqual(mockTranslations);
    });

    it("should handle null or undefined translations property", () => {
        const newTranslation = { id: generatePK().toString(), language: "fr", content: "Bonjour" };
        expect(updateTranslation({ translations: null as any }, newTranslation)).toHaveLength(0);
        expect(updateTranslation({ translations: undefined as any }, newTranslation)).toHaveLength(0);
    });

    it("should handle empty translations array", () => {
        const newTranslation = { id: generatePK().toString(), language: "fr", content: "Bonjour" };
        expect(updateTranslation({ translations: [] }, newTranslation)).toEqual([newTranslation]);
    });

    it("should not update translations if the provided translation does not have a language", () => {
        const newTranslation = { id: generatePK().toString(), content: "Bonjour" } as any;
        const updatedTranslations = updateTranslation({ translations: mockTranslations }, newTranslation);
        expect(updatedTranslations).toEqual(mockTranslations);
    });

    it("should preserve other translations when updating", () => {
        const newTranslation = { id: generatePK().toString(), language: "en", content: "Hi" };
        const updatedTranslations = updateTranslation({ translations: mockTranslations }, newTranslation);
        const unchangedTranslation = updatedTranslations.find(t => t.language === "es");
        expect(unchangedTranslation).toEqual(mockTranslations[1]);
    });
});

describe("getLanguageSubtag", () => {
    it("should return the subtag for a standard IETF language code", () => {
        expect(getLanguageSubtag("en-US")).toEqual("en");
        expect(getLanguageSubtag("fr-CA")).toEqual("fr");
    });

    it("should handle language codes without a region", () => {
        expect(getLanguageSubtag("de")).toEqual("de");
    });

    it("should handle language codes with extended tags", () => {
        expect(getLanguageSubtag("zh-Hant-HK")).toEqual("zh");
    });

    it("should return empty string for undefined input", () => {
        expect(getLanguageSubtag(undefined as any)).toEqual("");
    });

    it("should return empty string for null input", () => {
        expect(getLanguageSubtag(null as any)).toEqual("");
    });

    it("should return empty string for empty string input", () => {
        expect(getLanguageSubtag("")).toEqual("");
    });

    it("should return empty string for non-string input", () => {
        expect(getLanguageSubtag(1234 as any)).toEqual("");
        expect(getLanguageSubtag({} as any)).toEqual("");
    });

    it("should handle codes with non-standard format", () => {
        expect(getLanguageSubtag("i-klingon")).toEqual("i");
        expect(getLanguageSubtag("sgn-BE-FR")).toEqual("sgn");
    });

    it("should always lowercase input", () => {
        expect(getLanguageSubtag("EN-us")).toEqual("en");
        expect(getLanguageSubtag("Fr-cA")).toEqual("fr");
    });

    it("should handle input with extra whitespace", () => {
        expect(getLanguageSubtag(" en-US ")).toEqual("en");
        expect(getLanguageSubtag(" fr-CA ")).toEqual("fr");
    });

    it("should handle input with only region subtag", () => {
        expect(getLanguageSubtag("-US")).toEqual("");
        expect(getLanguageSubtag("-CA")).toEqual("");
    });
});

describe("getUserLanguages", () => {
    // Reset navigator.language before each test
    beforeEach(() => {
        mockNavigatorLanguage(undefined);
    });

    it("should return at least English when useDefault is true and no languages are found", () => {
        expect(getUserLanguages(null, true)).toEqual(["en"]);
        expect(getUserLanguages(undefined)).toEqual(["en"]);
    });

    it("should return user defined languages from session", () => {
        const session = createSession(["en-US", "fr"]);
        expect(getUserLanguages(session)).toEqual(["en", "fr"]);
    });

    it("should handle session with empty language array", () => {
        const session = createSession([]);
        expect(getUserLanguages(session)).toEqual(["en"]);
    });

    it("should handle session with missing languages property", () => {
        const session = createSession(undefined);
        expect(getUserLanguages(session)).toEqual(["en"]);
    });

    it("should return navigator language if no session languages", () => {
        mockNavigatorLanguage("de");
        const session = createSession([]);
        expect(getUserLanguages(session, false)).toEqual(["de"]);
    });

    it("should return empty array when useDefault is false and no languages are found", () => {
        expect(getUserLanguages(null, false)).toHaveLength(0);
        expect(getUserLanguages(undefined, false)).toHaveLength(0);
    });

    it("should handle cases where navigator does not return a language", () => {
        mockNavigatorLanguage(undefined);
        const session = createSession(undefined);
        expect(getUserLanguages(session, false)).toHaveLength(0);
    });
});

describe("getUserLocale", () => {
    // Reset navigator.languages and navigator.language before each test
    beforeEach(() => {
        mockNavigatorLanguages([]);
        mockNavigatorLanguage(undefined);
    });

    it("Prefer first language in both navigator and session", () => {
        const session = createSession(["fr", "de"]);
        mockNavigatorLanguages(["de-AT", "en-US"]);
        mockNavigatorLanguage("en-US");
        expect(getUserLocale(session)).toEqual("de-AT");
    });

    it("Prefer first language session if none of navigator languages are in the session", () => {
        const session = createSession(["af", "de"]);
        mockNavigatorLanguages(["fr-CA", "ja"]);
        mockNavigatorLanguage("zh");
        expect(getUserLocale(session)).toEqual("af");
    });

    it("Prefer navigator when session languages not present", () => {
        const session = createSession([]);
        mockNavigatorLanguage("zh");
        expect(getUserLocale(session)).toEqual("zh");
    });

    it("Prefer navigator when session languages not valid", () => {
        const session = createSession(["sheep"]);
        mockNavigatorLanguages(["fr-CA", "ja"]);
        mockNavigatorLanguage("zh");
        expect(getUserLocale(session)).toEqual("fr-CA");
    });

    it("Default to en-US when no languages valid", () => {
        const session = createSession(["chicken"]);
        mockNavigatorLanguages(["nuggets"]);
        expect(getUserLocale(session)).toEqual("en-US");
    });

    it("should handle undefined session", () => {
        mockNavigatorLanguages(["en-GB"]);
        expect(getUserLocale(undefined)).toEqual("en-GB");
    });

    it("should handle null session", () => {
        mockNavigatorLanguages(["en-GB"]);
        expect(getUserLocale(null)).toEqual("en-GB");
    });
});

describe("getPreferredLanguage", () => {
    it("should return the user's most preferred available language", () => {
        const available = ["en", "fr", "es"];
        const userPref = ["es", "de", "en"];
        expect(getPreferredLanguage(available, userPref)).toEqual("es");
    });

    it("should return the first available language if none of the user's preferred languages are available", () => {
        const available = ["it", "pt"];
        const userPref = ["es", "de", "en"];
        expect(getPreferredLanguage(available, userPref)).toEqual("it");
    });

    it("should handle empty list of user languages", () => {
        const available = ["zh", "fr"];
        const userPref = [];
        expect(getPreferredLanguage(available, userPref)).toEqual("zh");
    });

    it("should handle empty list of available languages", () => {
        const available = [];
        const userPref = ["zh", "fr"];
        expect(getPreferredLanguage(available, userPref)).toEqual("zh");
    });

    it("should handle both lists being empty", () => {
        const available = [];
        const userPref = [];
        expect(getPreferredLanguage(available, userPref)).toEqual("en");
    });

    it("should return the first language if multiple user preferences are available", () => {
        const available = ["fr", "de", "en"];
        const userPref = ["de", "en"];
        expect(getPreferredLanguage(available, userPref)).toEqual("de");
    });

    it("should handle case sensitivity", () => {
        const available = ["EN", "FR"];
        const userPref = ["en", "fr"];
        expect(getPreferredLanguage(available, userPref)).toEqual("EN");
    });

    it("should handle null and undefined inputs", () => {
        expect(getPreferredLanguage(null as any, ["fr"])).toEqual("fr");
        expect(getPreferredLanguage(["fr"], null as any)).toEqual("fr");
        expect(getPreferredLanguage(undefined as any, ["fr"])).toEqual("fr");
        expect(getPreferredLanguage(["fr"], undefined as any)).toEqual("fr");
    });
});

describe("getShortenedLabel", () => {
    it("should shorten Latin script words to a maximum of 3 characters", () => {
        expect(getShortenedLabel("Apple")).toEqual("App");
        expect(getShortenedLabel("Banana")).toEqual("Ban");
    });

    it("should return the word as is if it's 3 characters or less", () => {
        expect(getShortenedLabel("Car")).toEqual("Car");
        expect(getShortenedLabel("Do")).toEqual("Do");
    });

    it("should shorten words with Han characters to 1 character", () => {
        expect(getShortenedLabel("苹果")).toEqual("苹");
        expect(getShortenedLabel("香蕉好吃")).toEqual("香");
    });

    it("should handle words with mixed scripts, prioritizing Han shortening", () => {
        expect(getShortenedLabel("苹apple")).toEqual("苹");
        expect(getShortenedLabel("香banana")).toEqual("香");
    });

    it("should return an empty string when given an empty string", () => {
        expect(getShortenedLabel("")).toEqual("");
    });

    it("should handle non-string inputs gracefully", () => {
        expect(getShortenedLabel(null as any)).toEqual("");
        expect(getShortenedLabel(undefined as any)).toEqual("");
        expect(getShortenedLabel(123 as any)).toEqual("");
    });

    it("should handle single-character inputs correctly", () => {
        expect(getShortenedLabel("A")).toEqual("A");
        expect(getShortenedLabel("苹")).toEqual("苹");
    });

    it("should handle special characters and numbers", () => {
        expect(getShortenedLabel("12345")).toEqual("123");
        expect(getShortenedLabel("!@#")).toEqual("!@#");
    });
});

describe("getTranslationData", () => {
    const mockField = {
        value: [
            { language: "en", content: "Hello" },
            { language: "es", content: "Hola" },
        ],
    } as unknown as FieldInputProps<Array<TranslationObject>>;

    const mockMeta = {
        touched: [true, false],
        error: [{ content: "Required" }, {}],
    } as unknown as FieldMetaProps<unknown>;

    it("should correctly retrieve translation data for a given language", () => {
        let language = "en";
        let result = getTranslationData(mockField, mockMeta, language);
        expect(result.index).toBe(0);
        expect(result.value).toEqual(mockField.value[0]);
        expect(result.touched).toBe(true);
        expect(result.error).toEqual({ content: "Required" });

        language = "es";
        result = getTranslationData(mockField, mockMeta, language);
        expect(result.index).toBe(1);
        expect(result.value).toEqual(mockField.value[1]);
        expect(result.touched).toBe(false);
        expect(result.error).toEqual({});
    });

    it("should return undefined values when the language is not found", () => {
        const language = "fr";
        const result = getTranslationData(mockField, mockMeta, language);
        expect(result.index).toBe(-1);
        expect(result.value).toBeUndefined();
        expect(result.touched).toBeUndefined();
        expect(result.error).toBeUndefined();
    });

    it("should handle null or undefined field values", () => {
        const language = "en";
        const resultFromNull = getTranslationData({ ...mockField, value: null as any }, mockMeta, language);
        const resultFromUndefined = getTranslationData({ ...mockField, value: undefined as any }, mockMeta, language);
        expect(resultFromNull).toEqual({ error: undefined, index: -1, touched: undefined, value: undefined });
        expect(resultFromUndefined).toEqual({ error: undefined, index: -1, touched: undefined, value: undefined });
    });

    it("should handle non-array field values", () => {
        const language = "en";
        const result = getTranslationData({ ...mockField, value: "not an array" as any }, mockMeta, language);
        expect(result).toEqual({ error: undefined, index: -1, touched: undefined, value: undefined });
    });

    it("should handle missing meta properties", () => {
        const language = "en";
        const resultNoTouched = getTranslationData(mockField, { ...mockMeta, touched: undefined as any }, language);
        const resultNoError = getTranslationData(mockField, { ...mockMeta, error: undefined }, language);
        expect(resultNoTouched.touched).toBeUndefined();
        expect(resultNoError.error).toBeUndefined();
    });

    it("should handle empty meta properties", () => {
        const language = "en";
        const resultEmptyTouched = getTranslationData(mockField, { ...mockMeta, touched: [] as any }, language);
        const resultEmptyError = getTranslationData(mockField, { ...mockMeta, error: [] as any }, language);
        expect(resultEmptyTouched.touched).toBeUndefined();
        expect(resultEmptyError.error).toBeUndefined();
    });
});

describe("handleTranslationChange", () => {
    const mockField = {
        value: [
            { language: "en", content: "Hello" },
            { language: "es", content: "Hola" },
        ],
        name: "translations",
        onChange: vi.fn(),
        onBlur: vi.fn(),
    } as unknown as FieldInputProps<Array<TranslationObject>>;

    const mockMeta = {
        touched: [false, false],
        error: [undefined, undefined],
    } as unknown as FieldMetaProps<unknown>;

    const mockHelpers = {
        setValue: vi.fn(),
    };

    it("should correctly update the translation for the given language", () => {
        const event = { target: { name: "content", value: "Hi" } };
        const language = "en";
        handleTranslationChange(mockField, mockMeta, mockHelpers, event as any, language);
        expect(mockHelpers.setValue).toHaveBeenCalledWith([
            { language: "en", content: "Hi" },
            { language: "es", content: "Hola" },
        ]);
    });

    it("should not update other translations", () => {
        const event = { target: { name: "content", value: "Hi" } };
        const language = "en";
        handleTranslationChange(mockField, mockMeta, mockHelpers, event as any, language);
        const secondTranslation = mockHelpers.setValue.mock.calls[0][0][1];
        expect(secondTranslation).toEqual(mockField.value[1]);
    });

    it("should add translation when specified language is not found", () => {
        const event = { target: { name: "content", value: "Bonjour" } };
        const language = "fr";
        handleTranslationChange(mockField, mockMeta, mockHelpers, event as any, language);
        expect(mockHelpers.setValue).toHaveBeenCalledWith([
            ...mockField.value,
            { id: expect.any(String), language: "fr", content: "Bonjour" },
        ]);
    });

    it("should add field when it's missing from the translation object", () => {
        const event = { target: { name: "note", value: "Note" } };
        const language = "en";
        handleTranslationChange(mockField, mockMeta, mockHelpers, event as any, language);
        const expectedTranslation = { language: "en", content: "Hello", note: "Note" };
        expect(mockHelpers.setValue).toHaveBeenCalledWith([expectedTranslation, mockField.value[1]]);
    });

    it("should recover from null values", () => {
        const event = { target: { name: "content", value: "Hi" } };
        const language = "en";
        handleTranslationChange({ ...mockField, value: null as any }, mockMeta, mockHelpers, event as any, language);
        expect(mockHelpers.setValue).toHaveBeenCalledWith([{ id: expect.any(String), language: "en", content: "Hi" }]);
    });

    it("should recover from undefined values", () => {
        const event = { target: { name: "content", value: "Hi" } };
        const language = "en";
        handleTranslationChange({ ...mockField, value: undefined as any }, mockMeta, mockHelpers, event as any, language);
        expect(mockHelpers.setValue).toHaveBeenCalledWith([{ id: expect.any(String), language: "en", content: "Hi" }]);
    });

    it("should recover from non-array field values", () => {
        const event = { target: { name: "content", value: "Hi" } };
        const language = "en";
        handleTranslationChange({ ...mockField, value: "not an array" as any }, mockMeta, mockHelpers, event as any, language);
        expect(mockHelpers.setValue).toHaveBeenCalledWith([{ id: expect.any(String), language: "en", content: "Hi" }]);
    });
});

describe("getFormikErrorsWithTranslations", () => {
    // Define a mock validation schema
    const validationSchema = yup.object().shape({
        description: yup.string().max(50, "description is too long"),
        pages: yup.array().of(
            yup.object().shape({
                text: yup.string().max(50, "text is too long"),
            }),
        ),
    });

    const mockField = {
        name: "translations",
        value: [
            {
                id: "1",
                language: "en",
                description: "This is a long description. It is very long and yes it is very long",
                pages: [
                    { id: "1234", text: "This text is also very long, so that it exceeds the validation limit" },
                ],
            },
        ],
    } as unknown as FieldInputProps<TranslationObject[]>;

    it("should correctly convert formik errors to GridSubmitButtons errors", () => {
        const errors = getFormikErrorsWithTranslations(mockField, validationSchema);
        expect(errors).toEqual({
            "English description": ["description is too long"],
            "English pages[0].text": ["text is too long"],
        });
    });

    it("should return an empty error object when no validation errors are present", () => {
        const validField = {
            ...mockField,
            value: [{
                ...mockField.value[0],
                description: "Short description",
                pages: [{ id: "1234", text: "Short text" }],
            }],
        } as unknown as FieldInputProps<TranslationObject[]>;
        const errors = getFormikErrorsWithTranslations(validField, validationSchema);
        expect(errors).toEqual({});
    });

    it("should handle null or undefined field values", () => {
        const nullField = { ...mockField, value: null };
        const undefinedField = { ...mockField, value: undefined };
        expect(getFormikErrorsWithTranslations(nullField as any, validationSchema)).toEqual({});
        expect(getFormikErrorsWithTranslations(undefinedField as any, validationSchema)).toEqual({});
    });

    it("should handle non-array field values", () => {
        const nonArrayField = { ...mockField, value: "not an array" };
        expect(getFormikErrorsWithTranslations(nonArrayField as any, validationSchema)).toEqual({});
    });

    it("should handle empty translations array", () => {
        const emptyField = { ...mockField, value: [] };
        expect(getFormikErrorsWithTranslations(emptyField, validationSchema)).toEqual({});
    });
});

describe("combineErrorsWithTranslations", () => {
    const normalErrors = {
        name: "Name is required",
        translations: "Translations field is invalid",
        "translations[0].name": "Name too long",
        otherField: "Other error",
    };

    const translationErrors = {
        "English description": "Description is too long",
        "Spanish description": "Description is too long",
    };

    it("should correctly combine normal and translation errors", () => {
        const combinedErrors = combineErrorsWithTranslations(normalErrors, translationErrors);
        expect(combinedErrors).toEqual({
            name: "Name is required",
            otherField: "Other error",
            "English description": "Description is too long",
            "Spanish description": "Description is too long",
        });
    });

    it("should filter out normal errors that start with 'translations'", () => {
        const combinedErrors = combineErrorsWithTranslations(normalErrors, translationErrors);
        expect(combinedErrors.translations).toBeUndefined();
    });

    it("should handle empty error objects", () => {
        const combinedErrors = combineErrorsWithTranslations({}, {});
        expect(combinedErrors).toEqual({});
    });

    it("should handle only normal errors being present", () => {
        const combinedErrors = combineErrorsWithTranslations(normalErrors, {});
        expect(combinedErrors).toEqual({
            name: "Name is required",
            otherField: "Other error",
        });
    });

    it("should handle only translation errors being present", () => {
        const combinedErrors = combineErrorsWithTranslations({}, translationErrors);
        expect(combinedErrors).toEqual(translationErrors);
    });

    it("should handle null or undefined error objects", () => {
        const combinedErrorsWithNull = combineErrorsWithTranslations(null as any, translationErrors);
        const combinedErrorsWithUndefined = combineErrorsWithTranslations(undefined as any, translationErrors);
        expect(combinedErrorsWithNull).toEqual(translationErrors);
        expect(combinedErrorsWithUndefined).toEqual(translationErrors);
    });
});

describe("addEmptyTranslation", () => {
    let mockField: any;
    let mockMeta: any;
    let mockHelpers: any;

    beforeEach(() => {
        mockField = {
            value: [
                { id: "1234", language: "en", content: "Hello" },
            ],
            name: "translations",
            onChange: vi.fn(),
            onBlur: vi.fn(),
        } as unknown as FieldInputProps<Array<TranslationObject>>;

        mockMeta = {
            initialValue: [
                { id: "1234", language: "en", content: "" },
            ],
        } as unknown as FieldMetaProps<unknown>;

        mockHelpers = {
            setValue: vi.fn(),
        };
    });

    it("should correctly add an empty translation with determined fields", () => {
        const language = "es";
        addEmptyTranslation(mockField as any, mockMeta, mockHelpers, language);
        const newValue = mockHelpers.setValue.mock.calls[0][0][1];
        expect(newValue).toHaveProperty("language", language);
        expect(newValue).toHaveProperty("content", "");
        expect(newValue).toHaveProperty("id");
        expect(typeof newValue.id).toBe("string");
    });

    it("should not add a translation if initial values are not an array", () => {
        console.error = vi.fn(); // Mock console.error to check if it was called
        const language = "es";
        addEmptyTranslation(mockField as any, { ...mockMeta, initialValue: {} }, mockHelpers, language);
        expect(console.error).toHaveBeenCalledWith("Could not determine fields in translation object");
        expect(mockHelpers.setValue).not.toHaveBeenCalled();
    });

    it("should handle null or undefined field values", () => {
        const language = "es";
        addEmptyTranslation({ ...mockField, value: null as any }, mockMeta, mockHelpers, language);
        addEmptyTranslation({ ...mockField, value: undefined as any }, mockMeta, mockHelpers, language);
        expect(mockHelpers.setValue).toHaveBeenCalledTimes(2);
        const firstCallNewValue = mockHelpers.setValue.mock.calls[0][0][0];
        expect(firstCallNewValue).toHaveProperty("language", language);
        expect(firstCallNewValue).toHaveProperty("content", "");
        expect(firstCallNewValue).toHaveProperty("id");
    });

    it("should handle non-array field values", () => {
        const language = "es";
        addEmptyTranslation({ ...mockField, value: "not an array" as any }, mockMeta, mockHelpers, language);
        const newValue = mockHelpers.setValue.mock.calls[0][0][0];
        expect(newValue).toHaveProperty("language", language);
        expect(newValue).toHaveProperty("content", "");
        expect(newValue).toHaveProperty("id");
    });

    it("should handle empty translations array", () => {
        const language = "es";
        addEmptyTranslation({ ...mockField, value: [] as any }, mockMeta, mockHelpers, language);
        const newValue = mockHelpers.setValue.mock.calls[0][0][0];
        expect(newValue).toHaveProperty("language", language);
        expect(newValue).toHaveProperty("content", "");
        expect(newValue).toHaveProperty("id");
    });
});

describe("removeTranslation", () => {
    let mockField: any;
    let mockMeta: any;
    let mockHelpers: any;

    beforeEach(() => {
        mockField = {
            value: [
                { language: "en", content: "Hello" },
                { language: "es", content: "Hola" },
                { language: "fr", content: "Bonjour" },
            ],
            name: "translations",
            onChange: vi.fn(),
            onBlur: vi.fn(),
        } as unknown as FieldInputProps<Array<TranslationObject>>;

        mockMeta = {
            touched: [false, false, false],
            error: [undefined, undefined, undefined],
        } as unknown as FieldMetaProps<unknown>;

        mockHelpers = {
            setValue: vi.fn(),
        };
    });

    it("should correctly remove the translation for the given language", () => {
        const language = "es";
        removeTranslation(mockField as any, mockMeta, mockHelpers, language);
        expect(mockHelpers.setValue).toHaveBeenCalledWith([
            { language: "en", content: "Hello" },
            { language: "fr", content: "Bonjour" },
        ]);
    });

    it("should not remove any translations when the language is not found", () => {
        const language = "de";
        removeTranslation(mockField as any, mockMeta, mockHelpers, language);
        expect(mockHelpers.setValue).toHaveBeenCalledWith(mockField.value);
    });

    it("should handle null or undefined field values", () => {
        const language = "en";
        removeTranslation({ ...mockField, value: null as any }, mockMeta, mockHelpers, language);
        removeTranslation({ ...mockField, value: undefined as any }, mockMeta, mockHelpers, language);
        expect(mockHelpers.setValue).toHaveBeenCalledTimes(2);
        expect(mockHelpers.setValue).toHaveBeenCalledWith([]);
    });

    it("should handle non-array field values", () => {
        const language = "en";
        removeTranslation({ ...mockField, value: "not an array" as any }, mockMeta, mockHelpers, language);
        expect(mockHelpers.setValue).toHaveBeenCalledWith([]);
    });

    it("should handle empty translations array", () => {
        const language = "en";
        removeTranslation({ ...mockField, value: [] as any }, mockMeta, mockHelpers, language);
        expect(mockHelpers.setValue).toHaveBeenCalledWith([]);
    });

    it("should handle the only translation being removed", () => {
        const language = "en";
        removeTranslation({ ...mockField, value: [{ language: "en", content: "Hello" }] as any }, mockMeta, mockHelpers, language);
        expect(mockHelpers.setValue).toHaveBeenCalledWith([]);
    });
});

describe("translateSnackMessage", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        // Configure the i18next.t mock for these tests
        vi.mocked(i18next.t).mockImplementation((key: string, options?: Record<string, unknown>) => {
            // Mock translations for the tests
            const translations: Record<string, string> = {
                "CannotConnectToServer": "Cannot connect to server",
                "CannotConnectToServerDetails": "The details of cannot connect to server",
                "ChangePassword": "Change password",
            };
            
            // Check for defaultValue in options
            if (options?.defaultValue) {
                // If a translation exists, return it; otherwise return the defaultValue
                return translations[key] || options.defaultValue;
            }
            
            // Return the translation if it exists, otherwise return the key
            if (translations[key]) {
                let result = translations[key];
                // Handle interpolation
                if (options && typeof options === "object") {
                    Object.entries(options).forEach(([k, v]) => {
                        if (k !== "defaultValue") {
                            result = result.replace(`{{${k}}}`, String(v));
                        }
                    });
                }
                return result;
            }
            return key;
        });
    });

    afterEach(() => {
        vi.resetAllMocks();
    });

    it("should return message with details if both are in i18next dictionary", () => {
        const result = translateSnackMessage("CannotConnectToServer", {});
        expect(result).toEqual({ message: "Cannot connect to server", details: "The details of cannot connect to server" });
    });

    it("should return message without details if only message is in i18next dictionary", () => {
        const result = translateSnackMessage("ChangePassword", {});
        expect(result).toEqual({ message: "Change password", details: undefined });
    });

    it("should handle undefined variables - test 1", () => {
        const result = translateSnackMessage("CannotConnectToServer", undefined);
        expect(result).toEqual({ message: "Cannot connect to server", details: "The details of cannot connect to server" });
    });

    it("should handle undefined variables - test 2", () => {
        (vi.mocked(i18next.t) as any).mockImplementation((key: any, options: any) => {
            return `Message with variable ${options?.variable}`;
        });
        const result = translateSnackMessage("CannotConnectToServer", undefined);
        expect(result).toEqual({ message: "Message with variable undefined", details: "Message with variable undefined" }); // Limitation of i18next, so make sure your variables are defined
    });

    it("should interpolate variables into the message", () => {
        (vi.mocked(i18next.t) as any).mockImplementation((key: any, options: any) => {
            return `Message with variable ${options?.variable}`;
        });
        const result = translateSnackMessage("CannotConnectToServer", { variable: "value" });
        expect(result.message).toBe("Message with variable value");
    });

    it("should return default key as message if translation is missing", () => {
        const result = translateSnackMessage("qwerqwerqwer" as CommonKey, {});
        expect(result.message).toEqual("qwerqwerqwer");
        expect(result.details).toEqual(undefined);
    });
});
