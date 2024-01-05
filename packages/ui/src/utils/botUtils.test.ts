import { User } from "@local/shared";
import { findBotData, parseBotSettings } from "./botUtils";

describe("Bot Settings Tests", () => {
    beforeAll(() => {
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        jest.spyOn(console, "error").mockImplementation(() => { });
    });
    afterAll(() => {
        jest.restoreAllMocks();
    });

    // Test suite for parseBotSettings
    describe("parseBotSettings", () => {
        // Typical use cases
        test("should parse valid bot settings from User", () => {
            const user = {
                __typename: "User",
                id: "user123",
                handle: "testuser",
                isBotDepictingPerson: false,
                isPrivate: false,
                name: "Test User",
                translations: [{
                    __typename: "UserTranslation",
                    id: "translation1",
                    language: "en",
                    bio: "Original Bio",
                    bias: "Neutral",
                }],
                botSettings: JSON.stringify({
                    boop: "beep",
                    translations: {
                        en: {
                            bio: "Bot Bio",
                            otherField: "Other Field",
                        },
                    },
                }),
            } as unknown as Partial<User>;
            expect(parseBotSettings(user)).toEqual({
                boop: "beep",
                // Should have none of the non-bot translation fields
                translations: {
                    en: {
                        bio: "Bot Bio",
                        otherField: "Other Field",
                    },
                },
            });
        });

        test("should parse valid bot settings from BotShape", () => {
            const botShape = {
                __typename: "User",
                id: "bot123",
                handle: "testbot",
                isBotDepictingPerson: true,
                isBot: true,
                isPrivate: false,
                name: "Test Bot",
                translations: null,
                botSettings: JSON.stringify({ responsiveness: "high", accuracy: "medium", creativity: 0.8 }),
            } as unknown as Partial<User>;
            expect(parseBotSettings(botShape)).toEqual({ responsiveness: "high", accuracy: "medium", creativity: 0.8 });
        });

        // Edge cases and error handling
        test("should return empty object for invalid JSON", () => {
            const user = { botSettings: "not a valid json" };
            expect(parseBotSettings(user)).toEqual({});
        });

        test("should return empty object for undefined settings", () => {
            expect(parseBotSettings(undefined)).toEqual({});
        });
    });

    // Test suite for findBotData
    describe("findBotData", () => {
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
                        bio: "Bot Bio",
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
            const result = findBotData("fr", existingUser);
            expect(result).toEqual({
                creativity: 0.8,
                verbosity: 0.4,
                model: "gpt-4",
                translations: [expect.objectContaining({
                    language: "fr",
                    bio: "Original Bio", // Original translations should override bot settings translations
                    otherField: "Other Field", // Should still contain bot settings translations
                }), expect.objectContaining({
                    language: "ge",
                })],
            });
        });
    });
});
