/* eslint-disable @typescript-eslint/ban-ts-comment */
import { expect } from "chai";
import sinon from "sinon";
import { OpenAIModel } from "../../ai/services.js";
import { User } from "../../api/types.js";
import { BotSettingsConfig, LlmModel, findBotDataForForm } from "./bot.js";

// Valid bot data
const validBot1Settings = {
    __version: BotSettingsConfig.LATEST_VERSION,
    schema: {
        model: "gpt-3",
        maxTokens: 100,
        translations: {
            en: {
                persona: "friendly",
            },
            fr: {
                tone: "humorous",
            },
        },
    },
};
const validBot1 = {
    name: "TestBot",
    botSettings: JSON.stringify(validBot1Settings),
};

describe("BotSettingsConfig", () => {
    let consoleErrorStub: sinon.SinonStub;

    before(() => {
        consoleErrorStub = sinon.stub(console, "error");
    });

    beforeEach(() => {
        consoleErrorStub.resetHistory();
    });

    after(() => {
        consoleErrorStub.restore();
    });

    it("parses valid bot settings correctly", () => {
        const botConfigObject = BotSettingsConfig.deserialize(validBot1, console);
        // Should add the name to the schema
        expect(botConfigObject).to.deep.equal({
            ...validBot1Settings,
            schema: {
                ...validBot1Settings.schema,
                name: validBot1.name,
            },
        });
        // Export should be the same
        expect(botConfigObject.export()).to.deep.equal({
            ...validBot1Settings,
            schema: {
                ...validBot1Settings.schema,
                name: validBot1.name,
            },
        });
        // Serialize shouldn't include the name
        expect(botConfigObject.serialize()).to.deep.equal(JSON.stringify(validBot1Settings));
    });

    describe("handles invalid bot settings", () => {
        it("non-object string", () => {
            const botData = {
                name: "TestBot",
                botSettings: "invalid string",
            };
            const botConfigObject = BotSettingsConfig.deserialize(botData, console);
            expect(botConfigObject.schema).to.deep.equal({
                ...BotSettingsConfig.defaultBotSettings(),
                name: "TestBot",
            });
        });

        it("undefined botSettings", () => {
            const botData = {
                name: "Boop",
                botSettings: undefined,
            };
            const botConfigObject = BotSettingsConfig.deserialize(botData, console);
            expect(botConfigObject.schema).to.deep.equal({
                ...BotSettingsConfig.defaultBotSettings(),
                name: "Boop",
            });
        });

        it("object", () => {
            const botData = {
                name: "Hello world",
                // While arbitrary properties are allowed, this isn't serialized
                botSettings: { __version: "1.0", schema: { some: "object" } },
            };
            // @ts-ignore: Testing runtime scenario
            const botConfigObject = BotSettingsConfig.deserialize(botData, console);
            expect(botConfigObject.schema).to.deep.equal({
                ...BotSettingsConfig.defaultBotSettings(),
                name: "Hello world",
            });
        });
    });

    describe("handles missing bot data gracefully", () => {
        it("null input", () => {
            const botData = null;
            // @ts-ignore: Testing runtime scenario
            const botConfigObject = BotSettingsConfig.deserialize(botData, console);
            expect(botConfigObject.schema).to.deep.equal(BotSettingsConfig.defaultBotSettings());
        });

        it("undefined input", () => {
            const botData = undefined;
            // @ts-ignore: Testing runtime scenario
            const botConfigObject = BotSettingsConfig.deserialize(botData, console);
            expect(botConfigObject.schema).to.deep.equal(BotSettingsConfig.defaultBotSettings());
        });

        it("array input", () => {
            const botData = [];
            // @ts-ignore: Testing runtime scenario
            const botConfigObject = BotSettingsConfig.deserialize(botData, console);
            expect(botConfigObject.schema).to.deep.equal(BotSettingsConfig.defaultBotSettings());
        });
    });

    it("handles empty botSettings object", () => {
        const botData = {
            name: "TestBot",
            botSettings: "{}",
        };
        const botConfigObject = BotSettingsConfig.deserialize(botData, console);
        expect(botConfigObject.schema).to.deep.equal({
            ...BotSettingsConfig.defaultBotSettings(),
            name: "TestBot",
        });
    });

    it("retains additional properties in botSettings", () => {
        const botSettings = { __version: "1.0", schema: { some: "object" } };
        const botData = {
            name: "TestBot",
            botSettings: JSON.stringify(botSettings),
        };
        const botConfigObject = BotSettingsConfig.deserialize(botData, console);
        expect(botConfigObject.export()).to.deep.equal({
            ...botSettings,
            schema: {
                ...botSettings.schema,
                name: botData.name,
            },
        });
    });

    it("handles missing name field", () => {
        const botData = {
            botSettings: JSON.stringify(validBot1Settings),
            translations: null,
        };
        const botConfigObject = BotSettingsConfig.deserialize(botData, console);
        expect(botConfigObject.export()).to.deep.equal({
            ...validBot1Settings,
            schema: {
                ...validBot1Settings.schema,
                name: "",
            },
        });
    });

    it("should parse valid bot settings from User", () => {
        const userSettings = {
            __version: BotSettingsConfig.LATEST_VERSION,
            schema: {
                boop: "beep",
                translations: {
                    en: {
                        tone: "humorous",
                        otherField: "Other Field",
                    },
                },
            },
        };
        const userData = {
            name: "Test User",
            translations: [{
                // Fields defined in UserTranslation
                __typename: "UserTranslation" as const,
                id: "translation1",
                language: "en",
                bio: "Original Bio",
                // Additional fields - should add to schema
                bias: "Neutral",
            }],
            botSettings: JSON.stringify(userSettings),
        };
        const userConfigObject = BotSettingsConfig.deserialize(userData, console);
        // Should add the name and additional translation fields to the schema
        expect(userConfigObject).to.deep.equal({
            ...userSettings,
            schema: {
                ...userSettings.schema,
                name: userData.name,
                translations: {
                    en: {
                        // Fields defined in botSettings schema
                        tone: "humorous",
                        otherField: "Other Field",
                        // Additional fields from UserTranslation
                        bias: "Neutral",
                    },
                },
            },
        });
        // Serialize shouldn't include the name, but SHOULD keep additional translation fields
        expect(userConfigObject.serialize()).to.deep.equal(JSON.stringify({
            ...userSettings,
            schema: {
                ...userSettings.schema,
                // Name is not included
                translations: {
                    en: {
                        tone: "humorous",
                        otherField: "Other Field",
                        bias: "Neutral",
                    },
                },
            },
        }));
    });

    it("should handle ill-formed creativty/verbosity values", () => {
        const botSettings = {
            __version: BotSettingsConfig.LATEST_VERSION,
            schema: {
                model: "gpt-4o",
                maxTokens: 1_000,
                verbosity: 0.6, // Should be fine
                creativity: "0.2", // Should convert to number
                translations: {
                    en: {
                        persona: "friendly",
                        creativity: "-1", // Should be clamped to 0
                        verbosity: 1000, // Should be clamped to 1
                    },
                },
            },
        };
        const botData = {
            name: "Mr. Bean",
            botSettings: JSON.stringify(botSettings),
        };
        const botConfigObject = BotSettingsConfig.deserialize(botData, console);
        expect(botConfigObject).to.deep.equal({
            ...botSettings,
            schema: {
                ...botSettings.schema,
                name: botData.name,
                verbosity: 0.6,
                creativity: 0.2,
                translations: {
                    en: {
                        persona: "friendly",
                        creativity: 0,
                        verbosity: 1,
                    },
                },
            },
        });
    });
});

describe("findBotDataForForm", () => {
    let consoleErrorStub: sinon.SinonStub;

    before(() => {
        consoleErrorStub = sinon.stub(console, "error");
    });

    beforeEach(() => {
        consoleErrorStub.resetHistory();
    });

    after(() => {
        consoleErrorStub.restore();
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
            model: OpenAIModel.Gpt4_Turbo,
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

    const availableModels = [
        { name: "", description: "", value: OpenAIModel.Gpt4o_Mini },
        { name: "", description: "", value: OpenAIModel.Gpt4_Turbo },
        { name: "", description: "", value: OpenAIModel.Gpt4o_Mini },
    ] as unknown as LlmModel[];

    // Typical use cases
    it("should return default values for empty settings", () => {
        const result = findBotDataForForm("ge", availableModels, null); // Passing in null instead of data 
        expect(result).to.deep.equal({
            creativity: 0.5,
            verbosity: 0.5,
            model: OpenAIModel.Gpt4o_Mini,
            translations: [expect.objectContaining({ language: "ge" })], // checks if it contains the language
        });
    });

    it("should return parsed values from user settings", () => {
        const result = findBotDataForForm("fr", availableModels, existingUser);
        expect(result).to.deep.equal({
            creativity: 0.8,
            verbosity: 0.4,
            model: OpenAIModel.Gpt4_Turbo,
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
