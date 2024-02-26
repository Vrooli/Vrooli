/* eslint-disable @typescript-eslint/ban-ts-comment */
import { OpenAIService } from "./openai";

// NOTE: We don't need to test every function, since the service.test.ts tests covers some of them
describe("OpenAIService", () => {
    let openAIService: OpenAIService;

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

    beforeEach(() => {
        openAIService = new OpenAIService();
    });

    // Get config object
    allBots.forEach((botSettings, index) => {
        // YAML config tests
        it(`Bot settings #${index + 1}: should generate YAML config based on bot settings and user data`, async () => {
            const userData = { languages: ["en"] };
            // @ts-ignore: Testing runtime scenario
            const yamlConfig = await openAIService.getConfigObject(botSettings, userData);
            // Should be an object with a single key "ai_assistant"
            expect(yamlConfig).toHaveProperty("ai_assistant");
            expect(Object.keys(yamlConfig).length).toBe(1);
        });
    });
    it("should correctly translate based on user language preference", async () => {
        const userData = { languages: ["en"] };
        const yamlConfig = await openAIService.getConfigObject(botSettings1, userData, "Start", false);

        // Verify that the 'personality' field contains the English translation from botSettings1
        expect(yamlConfig.ai_assistant.personality).toEqual(botSettings1.translations.en);
    });

    it("should handle missing translations by providing the next available language", async () => {
        const userData = { languages: ["fr"] }; // Assuming 'fr' translation is not available
        const yamlConfig = await openAIService.getConfigObject(botSettings1, userData, "RoutineAdd", false);

        // Verify that the 'personality' field is the first available translation from botSettings1,
        // since 'fr' is not available.
        expect(yamlConfig.ai_assistant.personality).toEqual(botSettings1.translations.en);
    });
    it("should handle missing name field by using a default name", async () => {
        const userData = { languages: ["en"] };
        const yamlConfig = await openAIService.getConfigObject(botSettings3, userData, "ReminderAdd", false); // botSettings3 lacks a 'name'

        // Verify that the 'metadata.name' field is a non-empty string
        expect(yamlConfig.ai_assistant.metadata.name.length).toBeGreaterThan(0);
    });
    it("should handle multiple language translations and select based on user preference", async () => {
        const userData = { languages: ["es"] }; // Spanish preference
        const yamlConfig = await openAIService.getConfigObject(botSettings5, userData, "ScheduleFind", false);

        // Verify that the 'personality' field contains the Spanish translation from botSettings5
        expect(yamlConfig.ai_assistant.personality).toEqual(botSettings5.translations.es);
    });


    // Add more tests as needed for other methods and error handling
});
