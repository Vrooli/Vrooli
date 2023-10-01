import OpenAI from "openai";
import { logger } from "../events";

// Define an interface for a language model service
interface LanguageModelService {
    /** Generate a message response */
    generateResponse(prompt: string, config?: any): Promise<string>;
    /** Convert a preferred model to an available one */
    getModel(requestedModel: string): string;
}

// Implement the interface for OpenAI
class OpenAIService implements LanguageModelService {
    private openai: OpenAI;

    constructor() {
        this.openai = new OpenAI({
            apiKey: process.env.OPENAI_API_KEY,
        });
    }

    async generateResponse(prompt: string, config?: { model?: string, maxTokens?: number }): Promise<string> {
        const model = this.getModel(config?.model);
        const params: OpenAI.Chat.ChatCompletionCreateParams = {
            messages: [{ role: "user", content: prompt }],
            model,
        };
        const chatCompletion: OpenAI.Chat.ChatCompletion = await this.openai.chat.completions
            .create(params)
            .catch((error) => {
                const message = "Failed to call OpenAI";
                logger.error(message, { trace: "0009", error });
                throw new Error(message);
            });
        return chatCompletion.choices[0].message.content ?? "";
    }

    getModel(requestedModel: string | null | undefined): string {
        const defaultModel = "gpt-3.5-turbo";
        if (typeof requestedModel !== "string") return defaultModel;
        if (requestedModel.startsWith("gpt-4")) return "gpt-4";
        return defaultModel;
    }
}

export const getLanguageModelService = (botSettings: Record<string, unknown>): LanguageModelService => {
    // For now, always return OpenAIService, but you could add logic to return different services based on botSettings
    return new OpenAIService();
};
