import pkg from "../../__mocks__/@prisma/client";
import { mockPrisma, resetPrismaMockData } from "../../__mocks__/prismaUtils";
import { ModelMap } from "../../models";
import { fetchMessagesFromDatabase, tokenEstimationDefault } from "./service";
import { OpenAIService } from "./services/openai";

const { PrismaClient } = pkg;

jest.mock("@prisma/client");

describe("tokenEstimationDefault function", () => {
    test("ensures token count is more than 0 and less than the character count", () => {
        const text = "The quick brown fox";
        const [model, tokens] = tokenEstimationDefault(text);
        expect(model).toBe("default");
        expect(tokens).toBeGreaterThan(0);
        expect(tokens).toBeLessThanOrEqual(text.length);
    });

    test("ensures token count scales reasonably with text length", () => {
        const shortText = "Hello";
        const longText = "Hello, this is a longer piece of text to estimate";

        const [modelShort, tokensShort] = tokenEstimationDefault(shortText);
        const [modelLong, tokensLong] = tokenEstimationDefault(longText);

        expect(modelShort).toBe("default");
        expect(modelLong).toBe("default");
        expect(tokensShort).toBeLessThanOrEqual(shortText.length);
        expect(tokensLong).toBeLessThanOrEqual(longText.length);
        expect(tokensLong).toBeGreaterThan(tokensShort); // Longer text should result in more tokens
    });

    test("handles empty strings appropriately", () => {
        const text = "";
        const [model, tokens] = tokenEstimationDefault(text);
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
        // Initialize the ModelMap, which is used in fetchAndMapPlaceholder
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
    const lmServices = [
        { name: "OpenAIService", lmService: new OpenAIService() },
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
        "user_1": { botSettings: "default", id: "user_1", name: "User One" },
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
            const [tokenType, count] = lmService.estimateTokens({
                text: "sample text",
                requestedModel: model,
            });
            expect(tokenType).toBeDefined(); // Add more specific checks as needed
            expect(count).toBeDefined();
            expect(typeof count).toBe("number");
        });

        // Generate context
        it(`${lmServiceName}: generateContext returns a LanguageModelContext`, async () => {
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
        });

        // Generate response
        it(`${lmServiceName}: generateResponse returns a string - message provided`, async () => {
            const response = await lmService.generateResponse({
                chatId: chatId1,
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
            expect(typeof response).toBe("string");
        });
        it(`${lmServiceName}: generateResponse returns a string - message not provided`, async () => {
            const response = await lmService.generateResponse({
                chatId: chatId1,
                respondingToMessage: null,
                respondingBotId: respondingBotId1,
                respondingBotConfig: respondingBotConfig1,
                task: "Start",
                force: true,
                userData: userData1,
            });
            expect(typeof response).toBe("string");
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
