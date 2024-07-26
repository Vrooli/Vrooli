import { User } from "../api/generated/graphqlTypes";
import { OpenAIModel } from "../consts/llm";
import { findBotData, toBotSettings } from "./bot";

describe("toBotSettings", () => {
    beforeAll(() => {
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        jest.spyOn(console, "error").mockImplementation(() => { });
    });
    afterAll(() => {
        jest.restoreAllMocks();
    });

    it("parses valid bot settings correctly", () => {
        const bot = {
            name: "TestBot",
            botSettings: JSON.stringify({
                model: "gpt-3",
                maxTokens: 100,
                translations: {
                    en: {
                        persona: "friendly",
                    },
                },
            }),
        };
        expect(toBotSettings(bot, console)).toEqual({
            name: "TestBot",
            model: "gpt-3",
            maxTokens: 100,
            translations: {
                en: {
                    persona: "friendly",
                },
            },
        });
    });

    it("handles invalid JSON gracefully", () => {
        const bot = {
            name: "TestBot",
            botSettings: "invalid JSON",
        };
        expect(toBotSettings(bot, console)).toEqual({ name: "TestBot" });
    });

    it("returns name when botSettings is undefined", () => {
        const bot = {
            name: "TestBot",
            botSettings: undefined,
        };
        expect(toBotSettings(bot, console)).toEqual({ name: "TestBot" });
    });

    it("ignores non-string botSettings", () => {
        const bot = {
            name: "TestBot",
            botSettings: { some: "object" }, // Intentionally incorrect type
        };
        expect(toBotSettings(bot, console)).toEqual({ name: "TestBot" });
    });

    it("handles empty botSettings object", () => {
        const bot = {
            name: "TestBot",
            botSettings: "{}",
        };
        expect(toBotSettings(bot, console)).toEqual({ name: "TestBot" });
    });

    it("retains additional properties in botSettings", () => {
        const bot = {
            name: "TestBot",
            botSettings: JSON.stringify({ customProp: "customValue" }),
        };
        expect(toBotSettings(bot, console)).toEqual({
            name: "TestBot",
            customProp: "customValue"
        });
    });

    it("handles missing name field", () => {
        const bot = {
            botSettings: JSON.stringify({ model: "gpt-3" }),
        };
        expect(toBotSettings(bot, console)).toEqual({ name: "", model: "gpt-3" });
    });

    it("handles null input gracefully", () => {
        // @ts-ignore: Testing runtime scenario
        expect(toBotSettings(null, console)).toEqual({ name: "" });
    });

    it("handles undefined input gracefully", () => {
        // @ts-ignore: Testing runtime scenario
        expect(toBotSettings(undefined, console)).toEqual({ name: "" });
    });

    it("handles array input gracefully", () => {
        // @ts-ignore: Testing runtime scenario
        expect(toBotSettings([], console)).toEqual({ name: "" });
    });

    test("should parse valid bot settings from User", () => {
        const user = {
            name: "Test User",
            translations: [{
                __typename: "UserTranslation" as const,
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
        };
        expect(toBotSettings(user, console)).toEqual({
            boop: "beep",
            name: "Test User",
            // Should have none of the non-bot translation fields
            translations: {
                en: {
                    bio: "Bot Bio",
                    otherField: "Other Field",
                },
            },
        });
    });

    test("should parse valid bot settings when translations field is null", () => {
        const botShape = {
            __typename: "User",
            id: "bot123",
            handle: "testbot",
            isBotDepictingPerson: true,
            isBot: true,
            isPrivate: false,
            name: "Test Bot",
            translations: null,
            botSettings: JSON.stringify({
                translations: {
                    en: {
                        responsiveness: "high", accuracy: "medium", creativity: 0.8
                    },
                },
            }),
        };
        expect(toBotSettings(botShape, console)).toEqual({
            name: "Test Bot",
            translations: {
                en: {
                    responsiveness: "high", accuracy: "medium", creativity: 0.8
                },
            },
        });
    });

    test("should parse valid bot settings when botSettings is missing translations", () => {
        const botShape = {
            __typename: "User",
            id: "bot123",
            handle: "testbot",
            isBotDepictingPerson: true,
            isBot: true,
            isPrivate: false,
            name: "Test Bot",
            translations: [{
                __typename: "UserTranslation" as const,
                language: "fr",
                id: "translation1",
                bio: "Original Bio",
                bias: "Neutral",
                creativity: 0.8,
            }, {
                __typename: "UserTranslation" as const,
                language: "ge",
                id: "translation2",
                bio: "German Bio",
                bias: "Bunny",
                creativity: 0.7,
            }],
            botSettings: JSON.stringify({
                responsiveness: "high", accuracy: "medium", creativity: 0.8
            }),
        };
        expect(toBotSettings(botShape, console)).toEqual({
            name: "Test Bot",
            // Since translations is missing from botSettings, it uses the 
            // non-typical fields present in the top-level translations
            translations: {
                fr: {
                    bias: "Neutral",
                    creativity: 0.8,
                },
                ge: {
                    bias: "Bunny",
                    creativity: 0.7,
                },
            },
            // Doesn't move the botSettings field to a translation
            responsiveness: "high", accuracy: "medium", creativity: 0.8
        });
    });
});

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
            model: OpenAIModel.Gpt4,
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
            model: OpenAIModel.Gpt3_5Turbo,
            translations: [expect.objectContaining({ language: "ge" })], // checks if it contains the language
        });
    });

    test("should return parsed values from user settings", () => {
        const result = findBotData("fr", existingUser);
        expect(result).toEqual({
            creativity: 0.8,
            verbosity: 0.4,
            model: OpenAIModel.Gpt4,
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
