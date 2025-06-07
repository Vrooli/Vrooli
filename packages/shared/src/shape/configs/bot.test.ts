/* eslint-disable @typescript-eslint/ban-ts-comment */
import { describe, it, expect } from "vitest";
import sinon from "sinon";
import { OpenAIModel } from "../../ai/services.js";
import { type User } from "../../api/types.js";
import { BotConfig, DEFAULT_CREATIVITY, DEFAULT_PERSONA, DEFAULT_VERBOSITY, type BotConfigObject } from "./bot.js";

const LATEST_VERSION_STRING = "1.0"; // Consistent version string

// Valid bot data for new BotConfig structure
const validBotSettingsObject: BotConfigObject = {
    __version: LATEST_VERSION_STRING,
    model: "gpt-3",
    maxTokens: 100,
    persona: {
        occupation: "friendly",
        tone: "humorous",
        // Add other default persona fields to make it a complete persona object expected by tests
        bias: "",
        creativity: DEFAULT_CREATIVITY, // Use default from bot.ts
        domainKnowledge: "",
        keyPhrases: "",
        persona: "", // field in DEFAULT_PERSONA
        startingMessage: "",
        verbosity: DEFAULT_VERBOSITY, // Use default from bot.ts
    },
    resources: [],
};

// Test data will use BotConfigObject | null | undefined for botSettings
const validUserWithBotSettings: Pick<User, "botSettings"> = {
    botSettings: validBotSettingsObject, // Direct object, no JSON.stringify
};

describe("BotConfig", () => {
    let consoleErrorStub: sinon.SinonStub;

    beforeAll(() => {
        consoleErrorStub = sinon.stub(console, "error");
    });

    beforeEach(() => {
        consoleErrorStub.resetHistory();
    });

    afterAll(() => {
        consoleErrorStub.restore();
    });

    it("parses valid bot settings object correctly", () => {
        const botConfigInstance = BotConfig.parse(validUserWithBotSettings, console);
        expect(botConfigInstance.export()).to.deep.equal(validBotSettingsObject);
        expect(consoleErrorStub.called).to.be.false;
    });

    describe("handles botSettings property being null or undefined", () => {
        it("botSettings is null", () => {
            const userData: Pick<User, "botSettings"> = {
                botSettings: null,
            };
            const botConfigInstance = BotConfig.parse(userData, console);
            // Should return default config because botSettings is null
            // parseBase handles null data by defaulting fields like __version and resources,
            // and BotConfig factory defaults persona.
            expect(botConfigInstance.export()).to.deep.equal(BotConfig.default().export());
            // No error expected as null is a permissible value for botSettings (Maybe<BotConfigObject>)
            // and parseBase handles it gracefully by falling back to defaults.
            expect(consoleErrorStub.called).to.be.false;
        });

        it("botSettings is undefined", () => {
            const userData: Pick<User, "botSettings"> = {
                botSettings: undefined,
            };
            const botConfigInstance = BotConfig.parse(userData, console);
            // Should return default config because botSettings is undefined
            expect(botConfigInstance.export()).to.deep.equal(BotConfig.default().export());
            // No error expected as undefined is permissible and handled by parseBase.
            expect(consoleErrorStub.called).to.be.false;
        });
    });

    describe("handles invalid overall input for the User-like object passed to parse", () => {
        it("User-like object is null", () => {
            const nullUserData = null as unknown as Pick<User, "botSettings">;
            const botConfigInstance = BotConfig.parse(nullUserData, console);
            expect(botConfigInstance.export()).to.deep.equal(BotConfig.default().export());
            // If bot is null, bot?.botSettings is undefined. parseBase receives undefined and defaults.
            // No error is logged in this path.
            expect(consoleErrorStub.called).to.be.false;
        });

        it("User-like object is undefined", () => {
            const undefinedUserData = undefined as unknown as Pick<User, "botSettings">;
            const botConfigInstance = BotConfig.parse(undefinedUserData, console);
            expect(botConfigInstance.export()).to.deep.equal(BotConfig.default().export());
            // If bot is undefined, bot?.botSettings is undefined. parseBase receives undefined and defaults.
            // No error is logged in this path.
            expect(consoleErrorStub.called).to.be.false;
        });

        // Removed test for "array input for User object" as it's less a direct concern of BotConfig.parse 
        // and more about how Pick<User, "botSettings"> would be constructed or type-checked upstream.
        // BotConfig.parse expects an object (or null/undefined for the object itself based on tests above).
    });

    it("handles botSettings being an empty object literal '{}'", () => {
        const userData: Pick<User, "botSettings"> = {
            botSettings: {} as BotConfigObject, // Valid: empty object, but missing __version
        };
        const botConfigInstance = BotConfig.parse(userData, console);
        // parseBase will get this empty object. `data.__version` will be undefined.
        // So, `__version` will be LATEST_CONFIG_VERSION.
        // `resources` will be []. `persona` will be DEFAULT_PERSONA.
        // Effectively, an empty object becomes a default config.
        expect(botConfigInstance.export()).to.deep.equal(BotConfig.default().export());
        expect(consoleErrorStub.called).to.be.false; // No error, as it's gracefully defaulted.
    });

    it("retains valid fields from botSettings object and fills defaults for missing ones", () => {
        const partialBotSettingsObject: Partial<BotConfigObject> = {
            // __version is missing, should be defaulted by parseBase
            model: "retained-model",
            // maxTokens is missing, should be undefined in BotConfig constructor
            persona: { occupation: "tester" }, // Partial persona
            // resources are missing, should be defaulted by parseBase to []
        };
        const userData: Pick<User, "botSettings"> = {
            botSettings: partialBotSettingsObject as BotConfigObject, // Cast because it's partial
        };
        const botConfigInstance = BotConfig.parse(userData, console);

        const expectedFullPersona = {
            ...DEFAULT_PERSONA,
            occupation: "tester",
        };
        const expectedExport: BotConfigObject = {
            __version: LATEST_VERSION_STRING, // Defaulted by parseBase
            model: "retained-model",
            maxTokens: undefined, // BotConfig constructor specific
            persona: expectedFullPersona, // Defaulted by BotConfig factory
            resources: [], // Defaulted by parseBase
        };
        expect(botConfigInstance.export()).to.deep.equal(expectedExport);
        expect(consoleErrorStub.called).to.be.false;
    });

    it("parses a complete botSettings object correctly", () => {
        // This is similar to the first test but re-confirms with a full object.
        const fullBotSettings: BotConfigObject = {
            __version: LATEST_VERSION_STRING,
            model: "gpt-4-turbo",
            maxTokens: 2048,
            persona: {
                ...DEFAULT_PERSONA,
                bias: "Slightly positive",
                domainKnowledge: "Quantum Physics",
            },
            resources: [{ link: "http://example.com/qp.pdf", translations: [{ language: "en", name: "QP Doc" }] }],
        };
        const userData: Pick<User, "botSettings"> = {
            botSettings: fullBotSettings,
        };
        const botConfigInstance = BotConfig.parse(userData, console);
        expect(botConfigInstance.export()).to.deep.equal(fullBotSettings);
        expect(consoleErrorStub.called).to.be.false;
    });

    describe("persona handling with useFallbacks option", () => {
        it("should use fallbacks for persona when useFallbacks is true (default) and persona is missing in settings object", () => {
            const settingsNoPersona: Omit<BotConfigObject, "persona"> = {
                __version: LATEST_VERSION_STRING,
                model: "test-model-fallback",
                maxTokens: 300,
                resources: [],
            };
            const userData: Pick<User, "botSettings"> = {
                botSettings: settingsNoPersona as BotConfigObject, // Cast because Omit<> doesn't guarantee all BaseConfigObject fields
            };

            const configWithFallback = BotConfig.parse(userData, console, { useFallbacks: true });
            expect(configWithFallback.persona).to.deep.equal(DEFAULT_PERSONA);

            const configWithDefaultFallback = BotConfig.parse(userData, console); // useFallbacks is true by default
            expect(configWithDefaultFallback.persona).to.deep.equal(DEFAULT_PERSONA);
            expect(consoleErrorStub.called).to.be.false;
        });

        it("should NOT use fallbacks for persona (leaving it undefined) when useFallbacks is false and persona is missing", () => {
            const settingsNoPersona: Omit<BotConfigObject, "persona"> = {
                __version: LATEST_VERSION_STRING,
                model: "test-model-no-fallback",
                maxTokens: 350,
                resources: [],
            };
            const userData: Pick<User, "botSettings"> = {
                botSettings: settingsNoPersona as BotConfigObject,
            };
            const configWithoutFallback = BotConfig.parse(userData, console, { useFallbacks: false });
            expect(configWithoutFallback.persona).to.be.undefined;
            expect(consoleErrorStub.called).to.be.false;
        });

        it("should correctly fill default persona fields if a partial persona is provided (with fallbacks enabled by default)", () => {
            const settingsWithPartialPersona: BotConfigObject = {
                __version: LATEST_VERSION_STRING,
                model: "test-model-partial-persona",
                maxTokens: 400,
                persona: { // Only some persona fields provided, rest should default
                    occupation: "Engineer In Test",
                    tone: "Sarcastic",
                } as Partial<typeof DEFAULT_PERSONA> as Record<string, unknown>, // Cast to satisfy BotConfigObject's persona type
                resources: [],
            };
            const userData: Pick<User, "botSettings"> = {
                botSettings: settingsWithPartialPersona,
            };
            const botConfig = BotConfig.parse(userData, console); // useFallbacks is true by default
            const expectedPersona = {
                ...DEFAULT_PERSONA,
                occupation: "Engineer In Test",
                tone: "Sarcastic",
            };
            expect(botConfig.persona).to.deep.equal(expectedPersona);
            expect(consoleErrorStub.called).to.be.false;
        });
    });

    it("default method should return a BotConfig instance with the latest version and default values", () => {
        const defaultConfig = BotConfig.default();
        expect(defaultConfig).to.be.instanceOf(BotConfig);
        const exportedDefault = defaultConfig.export();
        expect(exportedDefault.__version).to.equal(LATEST_VERSION_STRING);
        expect(exportedDefault.model).to.be.undefined;
        expect(exportedDefault.maxTokens).to.be.undefined;
        expect(exportedDefault.persona).to.deep.equal(DEFAULT_PERSONA);
        expect(exportedDefault.resources).to.deep.equal([]);
        expect(consoleErrorStub.called).to.be.false;
    });
});

describe("OpenAIModel related tests", () => {
    it("should exist (placeholder)", () => {
        expect(OpenAIModel).to.exist;
    });
});
