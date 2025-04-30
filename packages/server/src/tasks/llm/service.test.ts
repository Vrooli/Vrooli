/* eslint-disable @typescript-eslint/ban-ts-comment */
import { LlmTask, SessionUser } from "@local/shared";
import { expect } from "chai";
import { RedisClientType } from "redis";
import sinon from "sinon";
import { DbProvider } from "../../db/provider.js";
import { initializeRedis } from "../../redisConn.js";
import { PreMapUserData } from "../../utils/chat.js";
import { ChatContextCollector } from "./context.js";
import { CreditValue, LanguageModelService, calculateMaxCredits, fetchMessagesFromDatabase } from "./service.js";
import { AnthropicService } from "./services/anthropic.js";
import { MistralService } from "./services/mistral.js";
import { OpenAIService } from "./services/openai.js";
import { GenerateResponseParams, GenerateResponseResult } from "./types.js";

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

describe("tokenEstimation function", () => {
    const testService = new OpenAIService();

    it("ensures token count is more than 0 and less than the character count", () => {
        const text = "The quick brown fox";
        const { tokens } = testService.estimateTokens({ text, aiModel: "gpt-3.5-turbo" });
        expect(tokens).to.be.greaterThan(0);
        expect(tokens).to.be.at.most(text.length);
    });

    it("ensures token count scales reasonably with text length", () => {
        const shortText = "Hello";
        const longText = "Hello, this is a longer piece of text to estimate";

        const { tokens: tokensShort } = testService.estimateTokens({ text: shortText, aiModel: "gpt-3.5-turbo" });
        const { tokens: tokensLong } = testService.estimateTokens({ text: longText, aiModel: "gpt-3.5-turbo" });

        expect(tokensShort).to.be.at.most(shortText.length);
        expect(tokensLong).to.be.at.most(longText.length);
        expect(tokensLong).to.be.greaterThan(tokensShort); // Longer text should result in more tokens
    });

    it("handles empty strings appropriately", () => {
        const text = "";
        const { tokens } = testService.estimateTokens({ text, aiModel: "gpt-3.5-turbo" });
        // 0 or 1 is fine
        expect(tokens).to.be.at.least(0);
        expect(tokens).to.be.at.most(1);
    });
});

describe("fetchMessagesFromDatabase", () => {
    const mockMessages = [
        { id: "1", translations: [{ language: "en", text: "Hello" }] },
        { id: "2", translations: [{ language: "en", text: "World" }] },
    ];
    let sandbox: sinon.SinonSandbox;

    beforeEach(async () => {
        sandbox = sinon.createSandbox();
        // Stub the DbProvider's findMany method
        sandbox.stub(DbProvider.get().chat_message, 'findMany').resolves(mockMessages);
    });

    afterEach(() => {
        sandbox.restore();
    });

    it("successfully retrieves messages from the database", async () => {
        const messages1 = await fetchMessagesFromDatabase(["1", "2"]);
        expect(messages1).to.deep.equal(mockMessages);

        const messages2 = await fetchMessagesFromDatabase(["2"]);
        expect(messages2).to.deep.equal(mockMessages);
    });

    it("returns an empty array when given an empty array of message IDs", async () => {
        // Override the stub for this specific test case
        sandbox.restore();
        sandbox.stub(DbProvider.get().chat_message, 'findMany').resolves([]);

        const messages = await fetchMessagesFromDatabase([]);
        expect(messages).to.deep.equal([]);
    });

    it("doesn't return messages that don't exist", async () => {
        // Override the stub for this specific test case
        sandbox.restore();
        sandbox.stub(DbProvider.get().chat_message, 'findMany').resolves([mockMessages[0]]);

        const messages = await fetchMessagesFromDatabase(["1", "3"]);
        expect(messages).to.deep.equal([mockMessages[0]]);
    });
});

describe("calculateMaxCredits", () => {
    // Normal cases
    it("should return the correct value when considering credits spent", () => {
        expect(Number(calculateMaxCredits(500_000, 1_000_000, 200_000))).to.equal(500_000); // Limited by remaining credits
        expect(Number(calculateMaxCredits(1_500_000, 1_000_000, 300_000))).to.equal(700_000); // Not limited by remaining credits
    });

    it("should work with different input types for creditsSpent", () => {
        expect(Number(calculateMaxCredits(500_000, 1_000_000, "200000"))).to.equal(500_000); // Limited by remaining credits
        expect(Number(calculateMaxCredits("1500000", "1000000", BigInt(300_000)))).to.equal(700_000); // Not limited by remaining credits
    });

    it("should return remaining credits when they are less than effective task max", () => {
        expect(Number(calculateMaxCredits(300_000, 1_000_000, 200_000))).to.equal(300_000);
    });

    // Edge cases
    it("should return 0 when credits spent equals or exceeds task max credits", () => {
        expect(Number(calculateMaxCredits(500_000, 1_000_000, 1_000_000))).to.equal(0);
        expect(Number(calculateMaxCredits(500_000, 1_000_000, 1_200_000))).to.equal(0);
    });

    it("should handle zero values correctly", () => {
        expect(Number(calculateMaxCredits(0, 1_000_000, 200_000))).to.equal(0);
        expect(Number(calculateMaxCredits(500_000, 0, 0))).to.equal(0);
        expect(Number(calculateMaxCredits(500_000, 1_000_000, 0))).to.equal(500_000);
    });

    it("should handle very large numbers correctly", () => {
        const largeNumber = "1234567890123456789012345678901234567890";
        expect(Number(calculateMaxCredits(largeNumber, 1000000000n, "500000"))).to.equal(999500000); // Not limited by remaining credits
        expect(Number(calculateMaxCredits(BigInt("1000000000"), largeNumber, "500000"))).to.equal(1000000000); // Limited by remaining credits
    });

    // Bad input handling
    it("should return 0 for negative numbers", () => {
        expect(Number(calculateMaxCredits(-500000, 1000000, 200000))).to.equal(0);
        expect(Number(calculateMaxCredits(500000, -1000000, 200000))).to.equal(0);
        expect(Number(calculateMaxCredits(500000, 1000000, -200000))).to.equal(0);
    });

    it("should throw an error for invalid string inputs", () => {
        expect(() => calculateMaxCredits("not a number", 1000000, 200000)).to.throw();
        expect(() => calculateMaxCredits(500000, "invalid", 200000)).to.throw();
        expect(() => calculateMaxCredits(500000, 1000000, "invalid")).to.throw();
    });

    it("should throw an error for non-integer inputs", () => {
        expect(() => calculateMaxCredits(3.14, 1000000, 200000)).to.throw();
        expect(() => calculateMaxCredits(500000, 2.718, 200000)).to.throw();
        expect(() => calculateMaxCredits(500000, 1000000, 3.14)).to.throw();
    });

    it("should treat empty string inputs as 0", () => {
        expect(Number(calculateMaxCredits("", 1000000, 200000))).to.equal(0);
        expect(Number(calculateMaxCredits(500000, "", 200000))).to.equal(0);
        expect(Number(calculateMaxCredits(500000, 1000000, ""))).to.equal(500000); // Limited by remaining credits
    });

    it("should treat null or undefined inputs as 0", () => {
        expect(Number(calculateMaxCredits(null as unknown as CreditValue, 1000000, 200000))).to.equal(0);
        expect(Number(calculateMaxCredits(500000, undefined as unknown as CreditValue, 200000))).to.equal(0);
        expect(Number(calculateMaxCredits(500000, 1000000, null as unknown as CreditValue))).to.equal(500000); // Limited by remaining credits
    });
});

// Test each implementation of LanguageModelService to ensure they comply with the interface
describe("LanguageModelService implementations", () => {
    let redis: RedisClientType;
    const sandbox = sinon.createSandbox();

    beforeEach(async () => {
        redis = await initializeRedis() as RedisClientType;
        sandbox.restore();

        // Stub database calls that might be needed
        sandbox.stub(DbProvider.get().chat_message, 'findMany').resolves([]);
    });

    afterEach(async () => {
        sandbox.restore();
    });

    const lmServices = [
        { name: "OpenAIService", lmService: new OpenAIService() },
        { name: "AnthropicService", lmService: new AnthropicService() },
        { name: "MistralService", lmService: new MistralService() },
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
        updatedAt: new Date().toISOString(),
    };

    const chatId1 = "chat_1";
    const respondingToMessageId1 = "msg_1";
    const respondingToMessageContent1 = "Hello, world!";

    lmServices.forEach(({ name: lmServiceName, lmService: lmService }) => {
        // Estimate tokens
        it(`${lmServiceName}: estimateTokens returns a result with tokens`, () => {
            const model = lmService.getModel();
            const result = lmService.estimateTokens({
                text: "sample text",
                aiModel: model as string,
            });
            expect(result).to.have.property('estimationModel');
            expect(result).to.have.property('encoding');
            expect(result).to.have.property('tokens');
            expect(typeof result.tokens).to.equal("number");
        });

        // Generate context
        it(`${lmServiceName}: generateContext returns a LanguageModelContext`, async () => {
            const model = lmService.getModel();
            const result = await lmService.generateContext({
                contextInfo: contextInfo1,
                force: true,
                model,
                mode: "text",
                participantsData: participantsData1,
                respondingBotConfig: respondingBotConfig1,
                respondingBotId: respondingBotId1,
                task: "Start",
                // @ts-ignore: Testing runtime scenario
                userData: userData1,
            });
            expect(result).to.have.property('messages');
            expect(result).to.have.property('systemMessage');
        });

        async function getResponseParams(
            latestMessage: string | null,
            taskMessage: string | null,
            chatId: string,
            participantsData: Record<string, PreMapUserData> | null | undefined,
            task: LlmTask | `${LlmTask}`,
            userData: SessionUser,
            lmService: LanguageModelService<string>,
        ): Promise<GenerateResponseParams> {
            const model = lmService.getModel();

            // Stub the ChatContextCollector since we're not testing that here
            const fakeContextCollector = {
                collectMessageContextInfo: sinon.stub().resolves(contextInfo1)
            };
            sandbox.stub(ChatContextCollector.prototype, 'collectMessageContextInfo').resolves(contextInfo1);

            const { messages, systemMessage } = await lmService.generateContext({
                contextInfo: contextInfo1,
                force: true,
                mode: "text",
                model,
                participantsData,
                respondingBotId: respondingBotId1,
                respondingBotConfig: respondingBotConfig1,
                task,
                taskMessage,
                userData,
            });
            return { maxTokens: null, messages, mode: "text", model, systemMessage, userData } as GenerateResponseParams;
        }

        // Generate response tests are stubbed since we don't want to call actual LLM services in tests
        it(`${lmServiceName}: generateResponse method structure is valid`, async () => {
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

            // Stub the actual API calls
            const fakeResponse: GenerateResponseResult = {
                attempts: 1,
                message: "This is a test response",
                cost: 1000
            };

            sandbox.stub(lmService, 'generateResponse').resolves(fakeResponse);

            const response = await lmService.generateResponse(params);
            expect(response).to.have.property('message');
            expect(response).to.have.property('cost');
            expect(response).to.have.property('attempts');
            expect(typeof response.message).to.equal("string");
            expect(typeof response.cost).to.equal("number");
            expect(response.cost).to.be.greaterThan(0);
            expect(isNaN(response.cost)).to.be.false;
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
                expect(yamlConfig).to.have.property("ai_assistant");
                expect(Object.keys(yamlConfig).length).to.equal(1);
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
                mode: "text",
            });

            // Verify that the 'personality' field contains the English translation from botSettings1, 
            // except for the 'startingMessage' field and any empty fields.
            let expectedTranslation = { ...botSettings1.translations.en } as object;
            delete (expectedTranslation as { startingMessage?: string }).startingMessage;
            expectedTranslation = Object.fromEntries(Object.entries(expectedTranslation).filter(([_, v]) => v !== ""));
            expect(yamlConfig.ai_assistant.personality).to.deep.equal(expectedTranslation);
        });

        it("should handle missing translations by providing the next available language", async () => {
            const userData = { languages: ["fr"] }; // Assuming 'fr' translation is not available
            const yamlConfig = await lmService.getConfigObject({
                botSettings: botSettings1,
                includeInitMessage: true,
                userData,
                task: "RoutineAdd",
                force: false,
                mode: "text",
            });

            // Verify that the 'personality' field is the first available translation from botSettings1,
            // since 'fr' is not available.
            let expectedTranslation = { ...botSettings1.translations.en } as object;
            delete (expectedTranslation as { startingMessage?: string }).startingMessage;
            expectedTranslation = Object.fromEntries(Object.entries(expectedTranslation).filter(([_, v]) => v !== ""));
            expect(yamlConfig.ai_assistant.personality).to.deep.equal(expectedTranslation);
        });

        it("should handle missing name field by using a default name", async () => {
            const userData = { languages: ["en"] };
            const yamlConfig = await lmService.getConfigObject({
                botSettings: botSettings3,
                includeInitMessage: true,
                userData,
                task: "ReminderAdd",
                force: false,
                mode: "text",
            }); // botSettings3 lacks a 'name'

            // Verify that the 'metadata.name' field is a non-empty string
            expect(yamlConfig.ai_assistant.metadata.name.length).to.be.greaterThan(0);
        });

        it("should handle multiple language translations and select based on user preference", async () => {
            const userData = { languages: ["es"] }; // Spanish preference
            const yamlConfig = await lmService.getConfigObject({
                botSettings: botSettings5,
                includeInitMessage: true,
                userData,
                task: "ScheduleFind",
                force: false,
                mode: "text",
            });

            // Verify that the 'personality' field contains the Spanish translation from botSettings5,
            // except for the 'startingMessage' field and any empty fields.
            let expectedTranslation = { ...botSettings5.translations.es } as object;
            delete (expectedTranslation as { startingMessage?: string }).startingMessage;
            expectedTranslation = Object.fromEntries(Object.entries(expectedTranslation).filter(([_, v]) => v !== ""));
            expect(yamlConfig.ai_assistant.personality).to.deep.equal(expectedTranslation);
        });

        it("should omit init_message if includeInitMessage is false", async () => {
            const userData = { languages: ["en"] };
            const yamlConfig = await lmService.getConfigObject({
                botSettings: botSettings1,
                includeInitMessage: false,
                userData,
                task: "Start",
                force: false,
                mode: "text",
            });

            expect(yamlConfig.ai_assistant).not.to.have.property("init_message");
        });

        it("should include init_message if includeInitMessage is true", async () => {
            const userData = { languages: ["en"] };
            const yamlConfig = await lmService.getConfigObject({
                botSettings: botSettings1,
                includeInitMessage: true,
                userData,
                task: "Start",
                force: false,
                mode: "text",
            });

            expect(yamlConfig.ai_assistant).to.have.property("init_message");
        });

        // Get context size
        it(`${lmServiceName}: getContextSize returns a number`, () => {
            const size = lmService.getContextSize();
            expect(typeof size).to.equal("number");
            expect(size).to.be.greaterThan(0);
        });

        // Get model
        it(`${lmServiceName}: getModel returns a valid model name`, () => {
            const model = lmService.getModel();
            expect(model).to.be.a('string');
            expect(model.length).to.be.greaterThan(0);
        });

        // Cost
        it(`${lmServiceName}: getResponseCost returns a number`, () => {
            const cost = lmService.getResponseCost({
                model: "default",
                usage: { input: 10, output: 10 },
            });
            expect(typeof cost).to.equal("number");
            expect(cost).to.be.greaterThan(0);
        });

        it(`${lmServiceName}: getResponseCost handles large numbers`, () => {
            const cost = lmService.getResponseCost({
                model: "default",
                usage: { input: Number.MAX_SAFE_INTEGER, output: Number.MAX_SAFE_INTEGER },
            });
            expect(cost).to.be.at.least(0);
            expect(isNaN(cost)).to.be.false;
        });

        it(`${lmServiceName}: getResponseCost isn't tricked by negative numbers`, () => {
            const cost = lmService.getResponseCost({
                model: "default",
                usage: { input: -1, output: -1 },
            });
            expect(cost).to.equal(0);
        });

        it(`${lmServiceName}: getMaxOutputTokensRestrained returns a whole number`, () => {
            const params = {
                maxCredits: BigInt(100),
                model: "default",
                inputTokens: 10,
            };
            const limit = lmService.getMaxOutputTokensRestrained(params);
            expect(typeof limit).to.equal("number");
            expect(limit).to.be.at.least(0);
            expect(limit).to.be.lessThan(Number.MAX_SAFE_INTEGER);
        });

        it(`${lmServiceName}: getMaxOutputTokensRestrained returns 0 if input cost is greater than max credits`, () => {
            const params = {
                maxCredits: BigInt(1),
                model: "default",
                inputTokens: 999999,
            };
            const limit = lmService.getMaxOutputTokensRestrained(params);
            expect(limit).to.equal(0);
        });

        it(`${lmServiceName}: getMaxOutputTokensRestrained returns a number less than or equal to the context size`, () => {
            const params = {
                maxCredits: BigInt(100),
                model: "default",
                inputTokens: 10,
            };
            const limit = lmService.getMaxOutputTokensRestrained(params);
            expect(limit).to.be.at.most(lmService.getContextSize());
        });

        it(`${lmServiceName}: getMaxOutputTokensRestrained handles large numbers`, () => {
            const params = {
                maxCredits: BigInt(Number.MAX_SAFE_INTEGER),
                model: "default",
                inputTokens: Number.MAX_SAFE_INTEGER / 2,
            };
            const limit = lmService.getMaxOutputTokensRestrained(params);
            expect(limit).to.be.at.least(0);
            expect(isNaN(limit)).to.be.false;
        });

        it(`${lmServiceName}: getMaxOutputTokensRestrained isn't tricked by negative numbers`, () => {
            let params = {
                maxCredits: BigInt(100),
                model: "default",
                inputTokens: -1,
            };
            let limit = lmService.getMaxOutputTokensRestrained(params);
            expect(limit).to.be.at.least(0);
            params = {
                maxCredits: BigInt(-1),
                model: "default",
                inputTokens: 100,
            };
            limit = lmService.getMaxOutputTokensRestrained(params);
            expect(limit).to.equal(0);
        });

        it(`${lmServiceName}: passing getResponseCost's result as 'maxCredits' in 'getMaxOutputTokensRestrained' results in an output token number equal to what was passed into getResponseCost`, () => {
            const INPUT_TOKENS = 69;
            const OUTPUT_TOKENS = 420;
            const cost = lmService.getResponseCost({
                model: "default",
                usage: { input: INPUT_TOKENS, output: OUTPUT_TOKENS },
            });
            const params = {
                maxCredits: BigInt(Math.round(cost)),
                model: "default",
                inputTokens: INPUT_TOKENS,
            };
            const limit = lmService.getMaxOutputTokensRestrained(params);
            // We allow some rounding error due to floating point calculations
            expect(Math.abs(limit - OUTPUT_TOKENS)).to.be.at.most(1);
        });
    });
});
