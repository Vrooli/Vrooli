import OpenAI from "openai";
import { logger } from "../../../events/logger";
import { ChatContextCollector } from "../context";
import { EstimateTokensParams, GenerateContextParams, GenerateResponseParams, GetConfigObjectParams, LanguageModelContext, LanguageModelService, generateDefaultContext, getDefaultConfigObject, tokenEstimationDefault } from "../service";

type OpenAIGenerateModel = "gpt-3.5-turbo" | "gpt-4";
type OpenAITokenModel = "default";
export class OpenAIService implements LanguageModelService<OpenAIGenerateModel, OpenAITokenModel> {
    private client: OpenAI;
    private defaultModel: OpenAIGenerateModel = "gpt-3.5-turbo";

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
        const messageContextInfo = respondingToMessage ?
            await (new ChatContextCollector(this)).collectMessageContextInfo(chatId, model, userData.languages, respondingToMessage.id) :
            [];
        const { messages, systemMessage } = await this.generateContext({
            respondingBotId,
            respondingBotConfig,
            messageContextInfo,
            participantsData,
            task,
            force,
            userData,
            requestedModel: model,
        });

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
        return completion.choices[0].message.content ?? "";
    }

    getContextSize(requestedModel?: string | null) {
        const model = this.getModel(requestedModel);
        switch (model) {
            case "gpt-3.5-turbo":
                return 4096;
            case "gpt-4":
                return 8192;
            default:
                return 4096;
        }
    }

    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    getEstimationMethod(_requestedModel?: string | null | undefined): "default" {
        return "default";
    }

    getEstimationTypes() {
        return ["default"] as const;
    }

    getModel(requestedModel?: string | null) {
        if (typeof requestedModel !== "string") return this.defaultModel;
        if (requestedModel.startsWith("gpt-4")) return "gpt-4";
        return this.defaultModel;
    }
}
