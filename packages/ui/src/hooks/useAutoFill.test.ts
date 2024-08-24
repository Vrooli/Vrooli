import { createUpdatedTranslations, getAutoFillTranslationData } from "./useAutoFill";

describe("getAutoFillTranslationData", () => {
    it("should return correct translation data omitting specific fields", () => {
        const values = {
            translations: [
                { language: "en", content: "Hello", id: "123", __typename: "Translation", otherField: "value" },
            ],
        };
        const language = "en";

        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const result = getAutoFillTranslationData(values, language) as any;

        expect(result).toEqual({ content: "Hello", otherField: "value" });
        expect(result.id).toBeUndefined();
        expect(result.language).toBeUndefined();
        expect(result.__typename).toBeUndefined();
    });

    it("should return any translation data if the language does not match", () => {
        const values = {
            translations: [
                { language: "fr", content: "Bonjour", id: "124", __typename: "Translation" },
            ],
        };
        const language = "en";

        const result = getAutoFillTranslationData(values, language);

        expect(result).toEqual({ content: "Bonjour" });
    });

    it("should handle null translations gracefully", () => {
        const values = {
            translations: null,
        };
        const language = "en";

        const result = getAutoFillTranslationData(values, language);

        expect(result).toEqual({});
    });

    it("should handle an empty translations array", () => {
        const values = {
            translations: [],
        };
        const language = "en";

        const result = getAutoFillTranslationData(values, language);

        expect(result).toEqual({});
    });

    it("should omit the specified fields even if they are not present", () => {
        const values = {
            translations: [
                { language: "en", content: "Hello", id: "123" }, // __typename is not present
            ],
        };
        const language = "en";

        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const result = getAutoFillTranslationData(values, language) as any;

        expect(result).toEqual({ content: "Hello" });
        expect(result.id).toBeUndefined();
        expect(result.language).toBeUndefined();
        expect(result.__typename).toBeUndefined(); // Ensuring it handles missing fields properly
    });
});

// describe("createUpdatedTranslations", () => {
//     it("should return empty translations if initial translations are undefined", () => {
//         const values = {};
//         const autofillData = { name: "New Name", description: "New Description" };
//         const language = "en";
//         const nonTranslatedFields = ["name"];

//         const result = createUpdatedTranslations(values, autofillData, language, nonTranslatedFields);

//         expect(result.updatedTranslations).toEqual([]);
//         expect(result.rest).toEqual({ name: "New Name" });
//     });

//     it("should update the correct translation based on language", () => {
//         const values = {
//             translations: [
//                 { language: "en", content: "Old Content", id: "1" },
//                 { language: "fr", content: "Contenu", id: "2" },
//             ],
//         };
//         const autofillData = { content: "New Content" };
//         const language = "en";
//         const nonTranslatedFields = [];

//         const result = createUpdatedTranslations(values, autofillData, language, nonTranslatedFields);

//         expect(result.updatedTranslations).toEqual([
//             { language: "en", content: "New Content", id: "1" },
//             { language: "fr", content: "Contenu", id: "2" },
//         ]);
//         expect(result.rest).toEqual({});
//     });

//     it("should handle non-existent language by updating the first translation", () => {
//         const values = {
//             translations: [
//                 { language: "es", content: "Contenido", id: "1" },
//             ],
//         };
//         const autofillData = { content: "New Content" };
//         const language = "de"; // German is not present
//         const nonTranslatedFields = [];

//         const result = createUpdatedTranslations(values, autofillData, language, nonTranslatedFields);

//         expect(result.updatedTranslations[0]).toEqual({
//             language: "es", content: "New Content", id: "1",
//         });
//     });

//     it("should separate translated and non-translated fields appropriately", () => {
//         const values = {
//             translations: [
//                 { language: "en", content: "Old Content", id: "1" },
//             ],
//         };
//         const autofillData = { content: "New Content", name: "New Name" };
//         const language = "en";
//         const nonTranslatedFields = ["name"];

//         const result = createUpdatedTranslations(values, autofillData, language, nonTranslatedFields);

//         expect(result.updatedTranslations[0]).toEqual({
//             language: "en", content: "New Content", id: "1",
//         });
//         expect(result.rest).toEqual({ name: "New Name" });
//     });

//     it("should return empty updatedTranslations if translations array is empty", () => {
//         const values = { translations: [] };
//         const autofillData = { content: "New Content" };
//         const language = "en";
//         const nonTranslatedFields = [];

//         const result = createUpdatedTranslations(values, autofillData, language, nonTranslatedFields);

//         expect(result.updatedTranslations).toEqual([]);
//     });
// });

describe("createUpdatedTranslations", () => {
    it("should return empty translations if initial translations are undefined", () => {
        const values = {};
        const autofillData = { name: "New Name", description: "New Description" };
        const language = "en";
        const translatedFields = ["description"];

        const result = createUpdatedTranslations(values, autofillData, language, translatedFields);

        expect(result.updatedTranslations).toEqual([]);
        expect(result.rest).toEqual({ name: "New Name" });
    });

    it("should update the correct translation based on language", () => {
        const values = {
            translations: [
                { language: "en", content: "Old Content", id: "1" },
                { language: "fr", content: "Contenu", id: "2" },
            ],
        };
        const autofillData = { content: "New Content" };
        const language = "en";
        const translatedFields = ["content"];

        const result = createUpdatedTranslations(values, autofillData, language, translatedFields);

        expect(result.updatedTranslations).toEqual([
            { language: "en", content: "New Content", id: "1" },
            { language: "fr", content: "Contenu", id: "2" },
        ]);
        expect(result.rest).toEqual({});
    });

    it("should handle non-existent language by updating the first translation", () => {
        const values = {
            translations: [
                { language: "es", content: "Contenido", id: "1" },
            ],
        };
        const autofillData = { content: "New Content" };
        const language = "de"; // German is not present
        const translatedFields = ["content"];

        const result = createUpdatedTranslations(values, autofillData, language, translatedFields);

        expect(result.updatedTranslations[0]).toEqual({
            language: "es", content: "New Content", id: "1",
        });
    });

    it("should separate translated and non-translated fields appropriately", () => {
        const values = {
            translations: [
                { language: "en", content: "Old Content", id: "1" },
            ],
        };
        const autofillData = { content: "New Content", name: "New Name" };
        const language = "en";
        const translatedFields = ["content"];

        const result = createUpdatedTranslations(values, autofillData, language, translatedFields);

        expect(result.updatedTranslations[0]).toEqual({
            language: "en", content: "New Content", id: "1",
        });
        expect(result.rest).toEqual({ name: "New Name" });
    });

    it("should return empty updatedTranslations if translations array is empty", () => {
        const values = { translations: [] };
        const autofillData = { content: "New Content" };
        const language = "en";
        const translatedFields = ["content"];

        const result = createUpdatedTranslations(values, autofillData, language, translatedFields);

        expect(result.updatedTranslations).toEqual([]);
        expect(result.rest).toEqual({});
    });

    it("should only update specified translated fields", () => {
        const values = {
            translations: [
                { language: "en", title: "Old Title", content: "Old Content", id: "1" },
            ],
        };
        const autofillData = { title: "New Title", content: "New Content", extra: "Extra Info" };
        const language = "en";
        const translatedFields = ["title"];

        const result = createUpdatedTranslations(values, autofillData, language, translatedFields);

        expect(result.updatedTranslations[0]).toEqual({
            language: "en", title: "New Title", content: "Old Content", id: "1",
        });
        expect(result.rest).toEqual({ content: "New Content", extra: "Extra Info" });
    });
});
