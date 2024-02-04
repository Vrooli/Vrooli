import pkg from "../../__mocks__/@prisma/client";
import { mockPrisma } from "../../__mocks__/prismaUtils";
import { ModelMap } from "../../models";
import { fetchMessagesFromDatabase, tokenEstimationDefault } from "./service";

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

// // Test each implementation of LanguageModelService to ensure they comply with the interface
// describe("LanguageModelService implementations", () => {
//     const implementations = [
//         { name: "OpenAIService", instance: new OpenAIService() },
//         // add other implementations as needed
//     ];

//     const respondingBotId1 = "bot_1";
//     const respondingBotConfig1 = {
//         name: "TestBot",
//         model: "default_model",
//         maxTokens: 100,
//     };
//     const messageContextInfo1 = [
//         { messageId: "msg_1", tokenSize: 10, userId: "user_1", language: "en" },
//     ];
//     const participantsData1 = {
//         "user_1": { botSettings: "default", id: "user_1", name: "User One" },
//     };
//     const userData1 = {
//         id: "user_1",
//         name: "User One",
//         credits: 100,
//         handle: "user1",
//         hasPremium: false,
//         languages: ["en"],
//         profileImage: "https://example.com/image.png",
//         updated_at: new Date().toISOString(),
//     };

//     const chatId1 = "chat_1";
//     const respondingToMessageId1 = "msg_1";
//     const respondingToMessageContent1 = "Hello, world!";

//     implementations.forEach(({ name, instance }) => {
//         describe(`${name} compliance with LanguageModelService interface`, () => {
//             it("estimateTokens returns a tuple with TokenNameType and number", () => {
//                 const [tokenType, count] = instance.estimateTokens("sample text");
//                 expect(tokenType).toBeDefined(); // Add more specific checks as needed
//                 expect(count).toBeDefined();
//                 expect(typeof count).toBe("number");
//             });

//             it("generateContext returns a LanguageModelContext", async () => {
//                 await expect(instance.generateContext(
//                     respondingBotId1,
//                     respondingBotConfig1,
//                     messageContextInfo1,
//                     participantsData1,
//                     userData1,
//                 )).resolves.toBeDefined();
//             });

//             it("generateResponse returns a string", async () => {
//                 const response = await instance.generateResponse(
//                     chatId1,
//                     respondingToMessageId1,
//                     respondingToMessageContent1,
//                     respondingBotId1,
//                     respondingBotConfig1,
//                     userData1,
//                 );
//                 expect(typeof response).toBe("string");
//             });

//             it("getContextSize returns a number", () => {
//                 const size = instance.getContextSize();
//                 expect(typeof size).toBe("number");
//             });

//             it("getEstimationMethod returns a TokenNameType", () => {
//                 const method = instance.getEstimationMethod();
//                 expect(method).toBeDefined(); // Add more specific checks as needed
//             });

//             it("getEstimationTypes returns an array of TokenNameType", () => {
//                 const types = instance.getEstimationTypes();
//                 expect(Array.isArray(types)).toBeTruthy();
//                 // Optionally, check if types array contains specific expected types
//             });

//             it("getModel returns a GenerateNameType", () => {
//                 const model = instance.getModel();
//                 expect(model).toBeDefined(); // Add more specific checks as needed
//             });
//         });
//     });
// });
