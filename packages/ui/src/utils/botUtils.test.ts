import { User } from "@local/shared";
import { findBotData } from "./botUtils";

describe("findBotData", () => {
    beforeAll(() => {
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        jest.spyOn(console, "error").mockImplementation(() => { });
    });
    afterAll(() => {
        jest.restoreAllMocks();
    });

    // Setup a basic existing user for tests
    const existingUser = {
        translations: [{
            __typename: "UserTranslation",
            id: "translation1",
            language: "fr",
            bio: "Original Bio",
            bias: "Neutral",
        }],
        botSettings: JSON.stringify({
            creativity: "0.8", // Invalid, but should still be able to parse
            verbosity: 0.4,
            model: "gpt-4",
            translations: {
                fr: {
                    bias: "bot settings bias",
                    otherField: "Other Field",
                },
                ge: {
                    bio: "German Bio",
                },
            },
        }),
    } as unknown as Partial<User>;

    // Typical use cases
    test("should return default values for empty settings", () => {
        const result = findBotData("ge", null); // Passing in null instead of data 
        expect(result).toEqual({
            creativity: 0.5,
            verbosity: 0.5,
            model: "gpt-3.5-turbo",
            translations: [expect.objectContaining({ language: "ge" })], // checks if it contains the language
        });
    });

    test("should return parsed values from user settings", () => {
        console.log('before parsed data for user')
        const result = findBotData("fr", existingUser);
        expect(result).toEqual({
            creativity: 0.8,
            verbosity: 0.4,
            model: "gpt-4",
            translations: [expect.objectContaining({
                language: "fr",
                bias: "Neutral", // Original translations should override bot settings translations
                otherField: "Other Field", // Should still contain bot settings translations
            }), expect.objectContaining({
                language: "ge",
            })],
        });
    });
});
