/* eslint-disable @typescript-eslint/ban-ts-comment */
import pkg from "../../__mocks__/@prisma/client";
import { mockPrisma, resetPrismaMockData } from "../../__mocks__/prismaUtils";
import { ModelMap } from "../../models";
import { fetchMessagesFromDatabase, tokenEstimationDefault } from "./service";
import { AnthropicService } from "./services/anthropic";
import { OpenAIService } from "./services/openai";

const { PrismaClient } = pkg;

const botSettings1 = {
    model: "gpt-3.5-turbo",
    maxTokens: 100,
    name: "Valyxa",
    translations: {
        en: {
            bias: "",
            creativity: 0.5,
            domainKnowledge: "Planning, scheduling, and task management",
            keyPhrases: "",
            occupation: "Vrooli assistant",
            persona: "Helpful, friendly, and professional",
            startingMessage: "Hello! How can I help you today?",
            tone: "Friendly",
            verbosity: 0.5,
        },
    },
};
const botSettings2 = {
    model: "gpt-4",
    maxTokens: 150,
    name: "Elon Musk",
    translations: {
        en: {
            bias: "Libertarian, pro-capitalism, and pro-technology",
            creativity: 1,
            domainKnowledge: "SpaceX, Tesla, and Neuralink",
            keyPhrases: "Mars, electric cars, and brain implants",
            occupation: "Entrepreneur and CEO",
            persona: "Visionary, ambitious, and eccentric",
            startingMessage: "Hello! I'm Elon Musk. Let's build the future together!",
            tone: "Confident",
            verbosity: 0.7,
        },
    },
};
const botSettings3 = {
    model: "invalid_model",
    maxTokens: 100,
    name: "Bob Ross",
};
const botSettings4 = {};
const botSettings5 = {
    model: "gpt-3.5-turbo",
    name: "Steve Jobs",
    translations: {
        en: {
            occupation: "Entrepreneur",
            persona: "Visionary",
            startingMessage: "Hello! I'm Steve Jobs. What shall we create today?",
        },
        es: {
            occupation: "Empresario",
            persona: "Visionario",
            startingMessage: "¡Hola! Soy Steve Jobs. ¿Qué crearemos hoy?",
        },
    },
};
const allBots = [botSettings1, botSettings2, botSettings3, botSettings4, botSettings5];

describe("tokenEstimationDefault function", () => {
    test("ensures token count is more than 0 and less than the character count", () => {
        const text = "The quick brown fox";
        const [model, tokens] = tokenEstimationDefault({ text, requestedModel: "default" });
        expect(model).toBe("default");
        expect(tokens).toBeGreaterThan(0);
        expect(tokens).toBeLessThanOrEqual(text.length);
    });

    test("ensures token count scales reasonably with text length", () => {
        const shortText = "Hello";
        const longText = "Hello, this is a longer piece of text to estimate";

        const [modelShort, tokensShort] = tokenEstimationDefault({ text: shortText, requestedModel: "default" });
        const [modelLong, tokensLong] = tokenEstimationDefault({ text: longText, requestedModel: "default" });

        expect(modelShort).toBe("default");
        expect(modelLong).toBe("default");
        expect(tokensShort).toBeLessThanOrEqual(shortText.length);
        expect(tokensLong).toBeLessThanOrEqual(longText.length);
        expect(tokensLong).toBeGreaterThan(tokensShort); // Longer text should result in more tokens
    });

    test("handles empty strings appropriately", () => {
        const text = "";
        const [model, tokens] = tokenEstimationDefault({ text, requestedModel: "default" });
        expect(model).toBe("default");
        // 0 or 1 is fine
        expect(tokens).toBeGreaterThanOrEqual(0);
        expect(tokens).toBeLessThanOrEqual(1);
    });
});

describe("fetchMessagesFromDatabase", () => {
    let prismaMock;

    const mockMessages = [
        { id: "1", translations: [{ language: "en", text: "Hello" }] },
        { id: "2", translations: [{ language: "en", text: "World" }] },
    ];

    beforeEach(async () => {
        await ModelMap.init();
        jest.clearAllMocks();
        prismaMock = mockPrisma({
            ChatMessage: JSON.parse(JSON.stringify(mockMessages)),
        });
        PrismaClient.injectMocks(prismaMock);
    });

    afterEach(() => {
        PrismaClient.resetMocks();
        resetPrismaMockData();
    });

    test("successfully retrieves messages from the database", async () => {
        const messages1 = await fetchMessagesFromDatabase(["1", "2"]);
        expect(messages1).toEqual(mockMessages);

        const messages2 = await fetchMessagesFromDatabase(["2"]);
        expect(messages2).toEqual([mockMessages[1]]);
    });

    test("returns an empty array when given an empty array of message IDs", async () => {
        const messages = await fetchMessagesFromDatabase([]);

        expect(messages).toEqual([]);
    });

    test("doesn't return messages that don't exist", async () => {
        const messages = await fetchMessagesFromDatabase(["1", "3"]);

        expect(messages).toEqual([mockMessages[0]]);
    });
});

// Test each implementation of LanguageModelService to ensure they comply with the interface
describe("LanguageModelService lmServices", () => {
    let prismaMock;

    beforeEach(async () => {
        await ModelMap.init();
        jest.clearAllMocks();
        prismaMock = mockPrisma({});
        PrismaClient.injectMocks(prismaMock);
    });

    afterEach(() => {
        PrismaClient.resetMocks();
        resetPrismaMockData();
    });

    const lmServices = [
        { name: "OpenAIService", lmService: new OpenAIService() },
        { name: "AnthropicService", lmService: new AnthropicService() },
        // add other lmServices as needed
    ];

    const respondingBotId1 = "bot_1";
    const respondingBotConfig1 = {
        name: "TestBot",
        model: "default_model",
        maxTokens: 100,
    };
    const messageContextInfo1 = [
        { messageId: "msg_1", tokenSize: 10, userId: "user_1", language: "en" },
    ];
    const participantsData1 = {
        "user_1": { botSettings: "default", id: "user_1", name: "User One", isBot: true },
    };
    const userData1 = {
        id: "user_1",
        name: "User One",
        credits: 100,
        handle: "user1",
        hasPremium: false,
        languages: ["en"],
        profileImage: "https://example.com/image.png",
        updated_at: new Date().toISOString(),
    };

    const chatId1 = "chat_1";
    const respondingToMessageId1 = "msg_1";
    const respondingToMessageContent1 = "Hello, world!";

    lmServices.forEach(({ name: lmServiceName, lmService: lmService }) => {
        // Estimate tokens
        it(`${lmServiceName}: estimateTokens returns a tuple with TokenNameType and number`, () => {
            console.log("estimate tokens start");
            const model = lmService.getModel();
            const [tokenType, count] = lmService.estimateTokens({
                text: "sample text",
                requestedModel: model,
            });
            console.log("estimate tokens end");
            expect(tokenType).toBeDefined(); // Add more specific checks as needed
            expect(count).toBeDefined();
            expect(typeof count).toBe("number");
        });

        // Generate context
        it(`${lmServiceName}: generateContext returns a LanguageModelContext`, async () => {
            console.log("generate context returns LanguageModelContext start");
            const model = lmService.getModel();
            await expect(lmService.generateContext({
                respondingBotId: respondingBotId1,
                respondingBotConfig: respondingBotConfig1,
                messageContextInfo: messageContextInfo1,
                participantsData: participantsData1,
                task: "Start",
                force: true,
                userData: userData1,
                requestedModel: model,
            })).resolves.toBeDefined();
            console.log("generate context returns LanguageModelContext end");
        });

        // Generate response
        it(`${lmServiceName}: generateResponse returns a string - message provided`, async () => {
            console.log("generate response returns a string - message provided start");
            const response = await lmService.generateResponse({
                chatId: chatId1,
                participantsData: participantsData1,
                respondingToMessage: {
                    id: respondingToMessageId1,
                    text: respondingToMessageContent1,
                },
                respondingBotId: respondingBotId1,
                respondingBotConfig: respondingBotConfig1,
                task: "Start",
                force: true,
                userData: userData1,
            });
            console.log("generate response returns a string - message provided end");
            expect(typeof response).toBe("string");
        });
        it(`${lmServiceName}: generateResponse returns a string - message not provided`, async () => {
            console.log("generate response returns a string - message not provided start");
            const response = await lmService.generateResponse({
                chatId: chatId1,
                participantsData: participantsData1,
                respondingToMessage: null,
                respondingBotId: respondingBotId1,
                respondingBotConfig: respondingBotConfig1,
                task: "Start",
                force: true,
                userData: userData1,
            });
            console.log("generate response returns a string - message not provided end");
            expect(typeof response).toBe("string");
        });

        // Get config object
        allBots.forEach((botSettings, index) => {
            // YAML config tests
            it(`Bot settings #${index + 1}: should generate YAML config based on bot settings and user data`, async () => {
                const userData = { languages: ["en"] };
                const yamlConfig = await lmService.getConfigObject({
                    // @ts-ignore: Testing runtime scenario
                    botSettings,
                    force: false,
                    task: "Start",
                    userData,
                });
                // Should be an object with a single key "ai_assistant"
                expect(yamlConfig).toHaveProperty("ai_assistant");
                expect(Object.keys(yamlConfig).length).toBe(1);
            });
        });
        it("should correctly translate based on user language preference", async () => {
            const userData = { languages: ["en"] };
            const yamlConfig = await lmService.getConfigObject({
                botSettings: botSettings1,
                includeInitMessage: true,
                userData,
                task: "Start",
                force: false,
            });

            // Verify that the 'personality' field contains the English translation from botSettings1
            expect(yamlConfig.ai_assistant.personality).toEqual(botSettings1.translations.en);
        });
        it("should handle missing translations by providing the next available language", async () => {
            const userData = { languages: ["fr"] }; // Assuming 'fr' translation is not available
            const yamlConfig = await lmService.getConfigObject({
                botSettings: botSettings1,
                includeInitMessage: true,
                userData,
                task: "RoutineAdd",
                force: false,
            });

            // Verify that the 'personality' field is the first available translation from botSettings1,
            // since 'fr' is not available.
            expect(yamlConfig.ai_assistant.personality).toEqual(botSettings1.translations.en);
        });
        it("should handle missing name field by using a default name", async () => {
            const userData = { languages: ["en"] };
            const yamlConfig = await lmService.getConfigObject({
                botSettings: botSettings3,
                includeInitMessage: true,
                userData,
                task: "ReminderAdd",
                force: false,
            }); // botSettings3 lacks a 'name'

            // Verify that the 'metadata.name' field is a non-empty string
            expect(yamlConfig.ai_assistant.metadata.name.length).toBeGreaterThan(0);
        });
        it("should handle multiple language translations and select based on user preference", async () => {
            const userData = { languages: ["es"] }; // Spanish preference
            const yamlConfig = await lmService.getConfigObject({
                botSettings: botSettings5,
                includeInitMessage: true,
                userData,
                task: "ScheduleFind",
                force: false,
            });

            // Verify that the 'personality' field contains the Spanish translation from botSettings5
            expect(yamlConfig.ai_assistant.personality).toEqual(botSettings5.translations.es);
        });
        it("should omit init_message if includeInitMessage is false", async () => {
            const userData = { languages: ["en"] };
            const yamlConfig = await lmService.getConfigObject({
                botSettings: botSettings1,
                includeInitMessage: false,
                userData,
                task: "Start",
                force: false,
            });

            expect(yamlConfig.ai_assistant).not.toHaveProperty("init_message");
        });
        it("should include init_message if includeInitMessage is true", async () => {
            const userData = { languages: ["en"] };
            const yamlConfig = await lmService.getConfigObject({
                botSettings: botSettings1,
                includeInitMessage: true,
                userData,
                task: "Start",
                force: false,
            });

            expect(yamlConfig.ai_assistant).toHaveProperty("init_message");
        });

        // Get context size
        it(`${lmServiceName}: getContextSize returns a number`, () => {
            const size = lmService.getContextSize();
            expect(typeof size).toBe("number");
        });

        // Get estimation method
        it(`${lmServiceName}: getEstimationMethod returns a TokenNameType`, () => {
            const method = lmService.getEstimationMethod();
            expect(method).toBeDefined(); // Add more specific checks as needed
        });

        // Get estimation types
        it(`${lmServiceName}: getEstimationTypes returns an array of TokenNameType`, () => {
            const types = lmService.getEstimationTypes();
            expect(Array.isArray(types)).toBeTruthy();
            // Optionally, check if types array contains specific expected types
        });

        // Get model
        it(`${lmServiceName}: getModel returns a GenerateNameType`, () => {
            const model = lmService.getModel();
            expect(model).toBeDefined(); // Add more specific checks as needed
        });
    });
});
