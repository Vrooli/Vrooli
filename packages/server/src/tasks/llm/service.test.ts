/* eslint-disable @typescript-eslint/ban-ts-comment */
import { SessionUserToken } from "@local/server";
import { LlmTask } from "@local/shared";
import pkg from "../../__mocks__/@prisma/client";
import { ModelMap } from "../../models";
import { PreMapUserData } from "../../models/base/chatMessage";
import { ChatContextCollector } from "./context";
import { GenerateResponseParams, LanguageModelService, fetchMessagesFromDatabase, tokenEstimationDefault } from "./service";
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
        const { model, tokens } = tokenEstimationDefault({ text, model: "default" });
        expect(model).toBe("default");
        expect(tokens).toBeGreaterThan(0);
        expect(tokens).toBeLessThanOrEqual(text.length);
    });

    test("ensures token count scales reasonably with text length", () => {
        const shortText = "Hello";
        const longText = "Hello, this is a longer piece of text to estimate";

        const { model: modelShort, tokens: tokensShort } = tokenEstimationDefault({ text: shortText, model: "default" });
        const { model: modelLong, tokens: tokensLong } = tokenEstimationDefault({ text: longText, model: "default" });

        expect(modelShort).toBe("default");
        expect(modelLong).toBe("default");
        expect(tokensShort).toBeLessThanOrEqual(shortText.length);
        expect(tokensLong).toBeLessThanOrEqual(longText.length);
        expect(tokensLong).toBeGreaterThan(tokensShort); // Longer text should result in more tokens
    });

    test("handles empty strings appropriately", () => {
        const text = "";
        const { model, tokens } = tokenEstimationDefault({ text, model: "default" });
        expect(model).toBe("default");
        // 0 or 1 is fine
        expect(tokens).toBeGreaterThanOrEqual(0);
        expect(tokens).toBeLessThanOrEqual(1);
    });
});

describe("fetchMessagesFromDatabase", () => {
    const mockMessages = [
        { id: "1", translations: [{ language: "en", text: "Hello" }] },
        { id: "2", translations: [{ language: "en", text: "World" }] },
    ];

    beforeEach(async () => {
        await ModelMap.init();
        jest.clearAllMocks();
        PrismaClient.injectData({
            ChatMessage: JSON.parse(JSON.stringify(mockMessages)),
        });
    });

    afterEach(() => {
        PrismaClient.clearData();
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
    beforeEach(async () => {
        await ModelMap.init();
        jest.clearAllMocks();
        PrismaClient.injectData({});
    });

    afterEach(() => {
        PrismaClient.clearData();
    });

    const lmServices = [
        { name: "OpenAIService", lmService: new OpenAIService() },
        { name: "AnthropicService", lmService: new AnthropicService() },
        // { name: "MistralService", lmService: new MistralService() },
        // add other lmServices as needed
    ];

    const respondingBotId1 = "bot_1";
    const respondingBotConfig1 = {
        name: "TestBot",
        model: "default_model",
        maxTokens: 100,
    };
    const contextInfo1 = [
        {
            __type: "message" as const,
            messageId: "msg_1",
            tokenSize: 10,
            userId: "user_1",
            language: "en",
        },
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
            const model = lmService.getModel();
            const { model: tokenModel, tokens } = lmService.estimateTokens({
                text: "sample text",
                model: model as string,
            });
            expect(tokenModel).toBeDefined();
            expect(typeof tokenModel).toBe("string");
            expect(tokens).toBeDefined();
            expect(typeof tokens).toBe("number");
        });

        // Generate context
        it(`${lmServiceName}: generateContext returns a LanguageModelContext`, async () => {
            console.log("beginning of failing test");
            const model = lmService.getModel();
            await expect(lmService.generateContext({
                contextInfo: contextInfo1,
                force: true,
                model,
                participantsData: participantsData1,
                respondingBotConfig: respondingBotConfig1,
                respondingBotId: respondingBotId1,
                task: "Start",
                // @ts-ignore: Testing runtime scenario
                userData: userData1,
            })).resolves.toBeDefined();
        });

        const getResponseParams = async (
            respondingToMessageId: string | null,
            respondingToMessageContent: string | null,
            chatId: string,
            participantsData: Record<string, PreMapUserData> | null | undefined,
            task: LlmTask | `${LlmTask}`,
            userData: SessionUserToken,
            lmService: LanguageModelService<string, string>,
        ) => {
            const respondingToMessage = respondingToMessageId || respondingToMessageContent ? {
                id: respondingToMessageId,
                text: respondingToMessageContent ?? undefined,
            } : null;
            const model = lmService.getModel();
            const contextInfo = await (new ChatContextCollector(lmService)).collectMessageContextInfo(chatId, model, userData.languages, respondingToMessage);
            const { messages, systemMessage } = await lmService.generateContext({
                contextInfo,
                force: true,
                model,
                participantsData,
                respondingBotId: respondingBotId1,
                respondingBotConfig: respondingBotConfig1,
                task,
                userData,
            });
            return { messages, model, respondingToMessage, systemMessage, userData } as GenerateResponseParams;
        };

        // Generate response
        it(`${lmServiceName}: generateResponse returns a valid object - message provided`, async () => {
            const params = await getResponseParams(
                respondingToMessageId1,
                respondingToMessageContent1,
                chatId1,
                participantsData1,
                "Start",
                // @ts-ignore: Testing runtime scenario
                userData1,
                lmService,
            );
            const response = await lmService.generateResponse(params);
            expect(typeof response).toBe("object");
            expect(typeof response.message).toBe("string");
            expect(typeof response.cost).toBe("number");
            expect(response.cost).toBeGreaterThan(0);
            expect(response.cost).not.toBeNaN();
        });
        it(`${lmServiceName}: generateResponse returns a valid object - message not provided`, async () => {
            const params = await getResponseParams(
                null,
                null,
                chatId1,
                participantsData1,
                "Start",
                // @ts-ignore: Testing runtime scenario
                userData1,
                lmService,
            );
            const response = await lmService.generateResponse(params);
            expect(typeof response).toBe("object");
            expect(typeof response.message).toBe("string");
            expect(typeof response.cost).toBe("number");
            expect(response.cost).toBeGreaterThan(0);
            expect(response.cost).not.toBeNaN();
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

            // Verify that the 'personality' field contains the English translation from botSettings1, 
            // except for the 'startingMessage' field and any empty fields.
            let expectedTranslation = { ...botSettings1.translations.en } as object;
            delete (expectedTranslation as { startingMessage?: string }).startingMessage;
            expectedTranslation = Object.fromEntries(Object.entries(expectedTranslation).filter(([_, v]) => v !== ""));
            expect(yamlConfig.ai_assistant.personality).toEqual(expectedTranslation);
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
            let expectedTranslation = { ...botSettings1.translations.en } as object;
            delete (expectedTranslation as { startingMessage?: string }).startingMessage;
            expectedTranslation = Object.fromEntries(Object.entries(expectedTranslation).filter(([_, v]) => v !== ""));
            expect(yamlConfig.ai_assistant.personality).toEqual(expectedTranslation);
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

            // Verify that the 'personality' field contains the Spanish translation from botSettings5,
            // except for the 'startingMessage' field and any empty fields.
            let expectedTranslation = { ...botSettings5.translations.es } as object;
            delete (expectedTranslation as { startingMessage?: string }).startingMessage;
            expectedTranslation = Object.fromEntries(Object.entries(expectedTranslation).filter(([_, v]) => v !== ""));
            expect(yamlConfig.ai_assistant.personality).toEqual(expectedTranslation);
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
