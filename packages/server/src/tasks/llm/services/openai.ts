import OpenAI from "openai";
import { logger } from "../../../events/logger";
import { ChatContextCollector } from "../context";
import { EstimateTokensParams, GenerateContextParams, GenerateResponseParams, GetConfigObjectParams, GetResponseCostParams, LanguageModelContext, LanguageModelService, generateDefaultContext, getDefaultConfigObject, tokenEstimationDefault } from "../service";

type OpenAIGenerateModel = "gpt-4-0125-preview" | "gpt-3.5-turbo-0125";
type OpenAITokenModel = "default";

/** Cost in cents per 1_000_000 input tokens */
const inputCosts: Record<OpenAIGenerateModel, number> = {
    "gpt-4-0125-preview": 1000,
    "gpt-3.5-turbo-0125": 50,
};
/** Cost in cents per 1_000_000 output tokens */
const outputCosts: Record<OpenAIGenerateModel, number> = {
    "gpt-4-0125-preview": 3000,
    "gpt-3.5-turbo-0125": 150,
};

export class OpenAIService implements LanguageModelService<OpenAIGenerateModel, OpenAITokenModel> {
    private client: OpenAI;
    private defaultModel: OpenAIGenerateModel = "gpt-3.5-turbo-0125";

    constructor() {
        this.client = new OpenAI({ apiKey: process.env.OPENAI_API_KEY });
    }

    estimateTokens(params: EstimateTokensParams) {
        return tokenEstimationDefault(params);
    }

    async getConfigObject(params: GetConfigObjectParams) {
        return getDefaultConfigObject(params);
    }

    async generateContext(params: GenerateContextParams): Promise<LanguageModelContext> {
        return generateDefaultContext({
            ...params,
            service: this,
        });
    }

    async generateResponse({
        chatId,
        force,
        participantsData,
        respondingBotConfig,
        respondingBotId,
        respondingToMessage,
        task,
        userData,
    }: GenerateResponseParams) {
        const model = this.getModel(respondingBotConfig?.model);
        const contextInfo = await (new ChatContextCollector(this)).collectMessageContextInfo(chatId, model, userData.languages, respondingToMessage);
        const { messages, systemMessage } = await this.generateContext({
            contextInfo,
            force,
            model,
            participantsData,
            respondingBotConfig,
            respondingBotId,
            task,
            userData,
        });

        // Generate response
        const params: OpenAI.Chat.ChatCompletionCreateParams = {
            messages: [
                // Add system message first
                { role: "system", content: systemMessage },
                // Add previous messages
                ...messages.map(({ role, content }) => ({ role, content })),
                // Add message we're responding to
                ...(respondingToMessage ? [{ role: "user" as const, content: respondingToMessage.text }] : []),
            ],
            model,
            user: userData.name ?? undefined,
        };
        const completion: OpenAI.Chat.ChatCompletion = await this.client.chat.completions
            .create(params)
            .catch((error) => {
                const message = "Failed to call OpenAI";
                logger.error(message, { trace: "0009", error });
                throw new Error(message);
            });
        const message = completion.choices[0].message.content ?? "";
        const cost = this.getResponseCost({
            model,
            usage: {
                input: completion.usage?.prompt_tokens ?? this.estimateTokens({ model, text: messages.map(m => m.content).join("\n") }).tokens,
                output: completion.usage?.completion_tokens ?? this.estimateTokens({ model, text: message }).tokens,
            },
        });
        return { message, cost };
    }

    getContextSize(requestedModel?: string | null) {
        const model = this.getModel(requestedModel);
        switch (model) {
            case "gpt-4-0125-preview":
                return 120_000;
            default:
                return 16_000;
        }
    }

    getResponseCost({ model, usage }: GetResponseCostParams) {
        const { input, output } = usage;
        return (inputCosts[model] * input) + (outputCosts[model] * output);
    }

    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    getEstimationMethod(_model?: string | null | undefined): "default" {
        return "default";
    }

    getEstimationTypes() {
        return ["default"] as const;
    }

    getModel(model?: string | null) {
        if (typeof model !== "string") return this.defaultModel;
        if (model.startsWith("gpt-4")) return "gpt-4-0125-preview";
        return this.defaultModel;
    }
}
