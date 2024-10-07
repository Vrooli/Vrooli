import { OpenAIModel, openAIServiceInfo } from "@local/shared";
import OpenAI from "openai";
import { CustomError } from "../../../events/error";
import { logger } from "../../../events/logger";
import { LlmServiceErrorType, LlmServiceId, LlmServiceRegistry } from "../registry";
import { EstimateTokensParams, GenerateContextParams, GenerateResponseParams, GetConfigObjectParams, GetOutputTokenLimitParams, GetOutputTokenLimitResult, GetResponseCostParams, LanguageModelContext, LanguageModelMessage, LanguageModelService, generateDefaultContext, getDefaultConfigObject, getDefaultMaxOutputTokensRestrained, getDefaultResponseCost, tokenEstimationDefault } from "../service";

type OpenAITokenModel = "default";

export class OpenAIService implements LanguageModelService<OpenAIModel, OpenAITokenModel> {
    public __id = LlmServiceId.OpenAI;
    private client: OpenAI;
    private defaultModel: OpenAIModel = OpenAIModel.Gpt4o_Mini;

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
        const { messages, systemMessage } = await generateDefaultContext({
            ...params,
            service: this,
        });

        const messagesWithContext = [
            // Add system message first
            { role: "system" as const, content: systemMessage },
            // Add previous messages
            ...messages.map(({ role, content }) => ({ role, content })),
        ] as LanguageModelMessage[];

        return { messages: messagesWithContext, systemMessage };
    }

    async generateResponse({
        maxTokens,
        messages,
        mode,
        model,
        userData,
    }: GenerateResponseParams) {
        // Generate response
        const params: OpenAI.Chat.ChatCompletionCreateParams = {
            max_tokens: maxTokens ?? openAIServiceInfo.fallbackMaxTokens,
            messages,
            model,
            response_format: { type: mode === "json" ? "json_object" : "text" },
            user: userData.name ?? undefined,
        } as const;
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
        return { attempts: 1, cost, message };
    }

    async *generateResponseStreaming({
        maxTokens,
        messages,
        mode,
        model,
        userData,
    }: GenerateResponseParams) {
        const params: OpenAI.Chat.ChatCompletionCreateParams = {
            max_tokens: maxTokens ?? openAIServiceInfo.fallbackMaxTokens,
            messages,
            model,
            response_format: { type: mode === "json" ? "json_object" : "text" },
            user: userData.name ?? undefined,
        } as const;

        let accumulatedMessage = "";
        // NOTE: OpenAI currently does not provide token usage when streaming. We'll have to estimate it ourselves.
        const inputTokens = this.estimateTokens({ model, text: messages.map(m => m.content).join("\n") }).tokens;
        let accumulatedOutputTokens = 0;

        try {
            // Create the stream
            const stream = await this.client.chat.completions.create({ ...params, stream: true });
            for await (const chunk of stream) {
                const content = chunk.choices[0]?.delta?.content || "";
                accumulatedMessage += content;
                const outputTokens = this.estimateTokens({ model, text: content }).tokens;
                accumulatedOutputTokens += outputTokens;
                const cost = this.getResponseCost({
                    model,
                    usage: {
                        input: inputTokens,
                        output: accumulatedOutputTokens, // Update this as you go
                    },
                });
                yield { __type: "stream" as const, message: content, cost };
            }
            // Return the final message
            const cost = this.getResponseCost({
                model,
                usage: {
                    input: inputTokens,
                    output: this.estimateTokens({ model, text: accumulatedMessage }).tokens,
                },
            });
            yield { __type: "end" as const, message: accumulatedMessage, cost };
        } catch (error) {
            const trace = "0420";
            const errorType = this.getErrorType(error);
            LlmServiceRegistry.get().updateServiceState(this.__id, errorType);
            logger.error("Failed to stream from OpenAI", { trace, error, errorType });

            const cost = this.getResponseCost({
                model,
                usage: {
                    input: inputTokens,
                    output: this.estimateTokens({ model, text: accumulatedMessage }).tokens,
                },
            });
            yield { __type: "error" as const, message: accumulatedMessage, cost };
        }
    }

    getContextSize(requestedModel?: string | null) {
        const model = this.getModel(requestedModel);
        return openAIServiceInfo.models[model].contextWindow;
    }

    getModelInfo() {
        return openAIServiceInfo.models;
    }

    getMaxOutputTokens(requestedModel?: string | null | undefined): number {
        const model = this.getModel(requestedModel);
        return openAIServiceInfo.models[model].maxOutputTokens;
    }

    getMaxOutputTokensRestrained(params: GetOutputTokenLimitParams): number {
        return getDefaultMaxOutputTokensRestrained(params, this);
    }

    getResponseCost(params: GetResponseCostParams) {
        return getDefaultResponseCost(params, this);
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
        if (model.startsWith("gpt-4o-mini")) return OpenAIModel.Gpt4o_Mini;
        if (model.startsWith("gpt-4o")) return OpenAIModel.Gpt4o;
        if (model.startsWith("gpt-4-turbo")) return OpenAIModel.Gpt4_Turbo;
        if (model.startsWith("gpt-4")) return OpenAIModel.Gpt4;
        if (model.startsWith("o1-mini")) return OpenAIModel.o1_Mini;
        if (model.startsWith("o1-preview")) return OpenAIModel.o1_Preview;
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

    async safeInputCheck(input: string): Promise<GetOutputTokenLimitResult> {
        try {
            const response = await this.client.moderations.create({
                input,
            });

            // The response contains an array of results, but we're only checking one input
            const result = response.results[0];

            // If the content is flagged, it's not safe
            const isSafe = !result.flagged;
            // The moderation check for OpenAI is free
            const cost = 0;
            return { cost, isSafe };
        } catch (error) {
            const trace = "0606";
            const errorType = this.getErrorType(error);
            LlmServiceRegistry.get().updateServiceState(this.__id, errorType);
            logger.error("Failed to call OpenAI moderation", { trace, error, errorType });

            // In case of an error, we assume the input is not safe
            return { cost: 0, isSafe: false };
        }
    }
}
