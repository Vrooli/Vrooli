import OpenAI from "openai";
import { CustomError } from "../../../events/error";
import { logger } from "../../../events/logger";
import { ChatContextCollector } from "../context";
import { LlmServiceErrorType, LlmServiceId, LlmServiceRegistry, LlmServiceState, OpenAIModel } from "../registry";
import { EstimateTokensParams, GenerateContextParams, GenerateResponseParams, GetConfigObjectParams, GetResponseCostParams, LanguageModelContext, LanguageModelService, generateDefaultContext, getDefaultConfigObject, tokenEstimationDefault } from "../service";

type OpenAITokenModel = "default";

/** Cost in cents per 1_000_000 input tokens */
const inputCosts: Record<OpenAIModel, number> = {
    [OpenAIModel.Gpt4]: 1000,
    [OpenAIModel.Gpt3_5Turbo]: 50,
};
/** Cost in cents per 1_000_000 output tokens */
const outputCosts: Record<OpenAIModel, number> = {
    [OpenAIModel.Gpt4]: 3000,
    [OpenAIModel.Gpt3_5Turbo]: 150,
};

export class OpenAIService implements LanguageModelService<OpenAIModel, OpenAITokenModel> {
    public __id = LlmServiceId.OpenAI;
    private client: OpenAI;
    private defaultModel: OpenAIModel = OpenAIModel.Gpt3_5Turbo;

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
        // Check if service is active
        if (LlmServiceRegistry.get().getServiceState(this.__id) !== LlmServiceState.Active) {
            throw new CustomError("0242", "InternalError", userData.languages);
        }

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
                const trace = "0009";
                const errorType = this.getErrorType(error);
                LlmServiceRegistry.get().updateServiceState(this.__id, errorType);
                logger.error("Failed to call OpenAI", { trace, error, errorType });
                throw new CustomError(trace, "InternalError", userData.languages, { error, errorType });
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
        if (model.startsWith("gpt-4")) return OpenAIModel.Gpt4;
        if (model.startsWith("gpt-3.5")) return OpenAIModel.Gpt3_5Turbo;
        return this.defaultModel;
    }

    getErrorType(error: unknown): LlmServiceErrorType {
        if (!error || typeof error !== "object" || typeof (error as { error?: unknown }).error !== "object") return LlmServiceErrorType.ApiError;
        const { type, code } = (error as { error: { type?: string, code?: string } }).error;

        // Handle the special case where an invalid API key results in an authentication error type
        if (code === "invalid_api_key") {
            return LlmServiceErrorType.Authentication;
        }

        // Map OpenAI specific error types to LlmServiceErrorType
        switch (type) {
            case "invalid_request_error":
            case "not_found_error":
            case "tokens_exceeded_error":
                return LlmServiceErrorType.InvalidRequest;
            case "authentication_error":
            case "permission_error":
                return LlmServiceErrorType.Authentication;
            case "rate_limit_error":
                return LlmServiceErrorType.RateLimit;
            case "server_error":
                return LlmServiceErrorType.ApiError;
            default:
                return LlmServiceErrorType.ApiError;
        }
    }
}
