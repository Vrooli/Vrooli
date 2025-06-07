import Anthropic from "@anthropic-ai/sdk";
import { AnthropicModel, anthropicServiceInfo, LlmServiceId } from "@vrooli/shared";
import { CustomError } from "../../../events/error.js";
import { logger } from "../../../events/logger.js";
import { AIServiceErrorType, AIServiceRegistry } from "../../../services/conversation/registry.js";
import { TokenEstimationRegistry } from "../../../services/conversation/tokens.js";
import { generateDefaultContext, getDefaultMaxOutputTokensRestrained, getDefaultResponseCost } from "../service.js";
import { type EstimateTokensParams, type EstimateTokensResult, type GenerateContextParams, type GenerateResponseParams, type GetOutputTokenLimitParams, type GetOutputTokenLimitResult, type GetResponseCostParams, type LanguageModelContext, type LanguageModelMessage, type LanguageModelService, TokenEstimatorType } from "../types.js";

export class AnthropicService implements LanguageModelService<AnthropicModel> {
    public __id = LlmServiceId.Anthropic;
    private client: Anthropic;
    private defaultModel: AnthropicModel = AnthropicModel.Sonnet3_5;

    constructor() {
        this.client = new Anthropic({ apiKey: process.env.ANTHROPIC_API_KEY });
    }

    estimateTokens(params: EstimateTokensParams) {
        return TokenEstimationRegistry.get().estimateTokens(TokenEstimatorType.Tiktoken, params);
    }

    async generateContext(params: GenerateContextParams): Promise<LanguageModelContext> {
        const { messages, systemMessage } = await generateDefaultContext({
            ...params,
            service: this,
        });

        // Ensure roles alternate between "user" and "assistant". This is a requirement of the Anthropic API.
        const alternatingMessages: LanguageModelMessage[] = [];
        let lastRole: LanguageModelMessage["role"] = "assistant";
        for (const { role, content } of messages) {
            // Skip empty messages. This is another requirement of the Anthropic API.
            if (content.trim() === "") {
                continue;
            }
            if (role !== lastRole) {
                alternatingMessages.push({ role, content });
                lastRole = role;
            } else {
                // Merge consecutive messages with the same role
                if (alternatingMessages.length > 0) {
                    alternatingMessages[alternatingMessages.length - 1].content += "\n" + content;
                } else {
                    alternatingMessages.push({ role, content });
                }
            }
        }

        // Ensure first message is from the user. This is another requirement of the Anthropic API.
        if (alternatingMessages.length && alternatingMessages[0].role === "assistant") {
            alternatingMessages.shift();
        }

        return { messages: alternatingMessages, systemMessage };
    }

    async *generateResponseStreaming({
        maxTokens,
        messages,
        model,
        systemMessage,
    }: GenerateResponseParams) {
        const params: Anthropic.MessageCreateParams = {
            messages: messages.map(({ role, content }) => ({ role, content })),
            model,
            max_tokens: maxTokens ?? anthropicServiceInfo.fallbackMaxTokens,
            system: systemMessage,
        } as const;

        const accumulatedText: [string, number][] = [];
        let totalInputTokens = this.estimateTokens({ aiModel: model, text: messages.map(m => m.content).join("\n") }).tokens;
        let totalOutputTokens = 0;

        try {
            // Setup the stream
            const stream = this.client.messages.stream(params);
            for await (const chunk of stream) {
                switch (chunk.type) {
                    case "message_start": {
                        totalInputTokens = chunk.message.usage.input_tokens;
                        totalOutputTokens = chunk.message.usage.output_tokens;
                        break;
                    }
                    case "content_block_delta": {
                        accumulatedText.push([chunk.delta.text, chunk.index]);
                        const cost = this.getResponseCost({
                            model,
                            usage: {
                                input: totalInputTokens,
                                output: totalOutputTokens,
                            },
                        });
                        yield { __type: "stream" as const, message: chunk.delta.text, cost };
                        break;
                    }
                    case "message_delta": {
                        totalOutputTokens = chunk.usage.output_tokens;
                        break;
                    }
                }
            }
            const message = accumulatedText.sort((a, b) => a[1] - b[1]).map(([text]) => text).join("");
            const cost = this.getResponseCost({
                model,
                usage: {
                    input: totalInputTokens,
                    output: totalOutputTokens,
                },
            });
            yield { __type: "end" as const, message, cost };
        } catch (error) {
            const trace = "0421";
            const errorType = this.getErrorType(error);
            AIServiceRegistry.get().updateServiceState(this.__id, errorType);
            logger.error("Failed to stream from Anthropic", { trace, error, errorType });

            const message = accumulatedText.sort((a, b) => a[1] - b[1]).map(([text]) => text).join("");
            const cost = this.getResponseCost({
                model,
                usage: {
                    input: totalInputTokens,
                    // We don't get the token usage until the end, so our best bet it to estimate the output tokens
                    output: this.estimateTokens({ aiModel: model, text: message }).tokens,
                },
            });
            yield { __type: "error" as const, message, cost };
        }
    }

    getContextSize(requestedModel?: string | null) {
        const model = this.getModel(requestedModel);
        return anthropicServiceInfo.models[model].contextWindow;
    }

    getModelInfo() {
        return anthropicServiceInfo.models;
    }

    getMaxOutputTokens(requestedModel?: string | null | undefined): number {
        const model = this.getModel(requestedModel);
        return anthropicServiceInfo.models[model].maxOutputTokens;
    }

    getMaxOutputTokensRestrained(params: GetOutputTokenLimitParams): number {
        return getDefaultMaxOutputTokensRestrained(params, this);
    }

    getResponseCost(params: GetResponseCostParams) {
        return getDefaultResponseCost(params, this);
    }

    getEstimationInfo(model?: string | null): Pick<EstimateTokensResult, "estimationModel" | "encoding"> {
        return TokenEstimationRegistry.get().getEstimationInfo(TokenEstimatorType.Tiktoken, model);
    }

    getModel(model?: string | null) {
        if (typeof model !== "string") return this.defaultModel;
        if (model.includes("haiku")) return AnthropicModel.Haiku3;
        if (model.includes("opus")) return AnthropicModel.Opus3;
        if (model.includes("sonnet-3-5")) return AnthropicModel.Sonnet3_5;
        if (model.includes("sonnet-3")) return AnthropicModel.Sonnet3;
        return this.defaultModel;
    }

    getErrorType(error: unknown): AIServiceErrorType {
        if (!error || typeof error !== "object") return AIServiceErrorType.ApiError;
        const errorType = (error as { error?: { type?: string } }).error?.type;

        switch (errorType) {
            case "invalid_request_error":
            case "not_found_error":
                return AIServiceErrorType.InvalidRequest;
            case "authentication_error":
            case "permission_error":
                return AIServiceErrorType.Authentication;
            case "rate_limit_error":
                return AIServiceErrorType.RateLimit;
            case "api_error":
                return AIServiceErrorType.ApiError;
            case "overloaded_error":
                return AIServiceErrorType.Overloaded;
            default:
                return AIServiceErrorType.ApiError;
        }
    }

    async safeInputCheck(input: string): Promise<GetOutputTokenLimitResult> {
        try {
            const moderationPrompt = `
                A human user is in dialogue with an AI. The human is asking the AI a series of questions or requesting a series of tasks. Here is the most recent request from the user:
                <user query>${input}</user query>
    
                If the user's request refers to harmful, pornographic, or illegal activities, reply with (Y). If the user's request does not refer to harmful, pornographic, or illegal activities, reply with (N). Reply with nothing else other than (Y) or (N).
                `;
            const moderationModel = AnthropicModel.Haiku3;

            const params: Anthropic.MessageCreateParams = {
                messages: [{ role: "user", content: moderationPrompt }],
                model: moderationModel,
                max_tokens: 10,
                temperature: 0,
            };

            const completion: Anthropic.Message = await this.client.messages
                .create(params)
                .catch((error) => {
                    const trace = "0422";
                    const errorType = this.getErrorType(error);
                    AIServiceRegistry.get().updateServiceState(this.__id, errorType);
                    logger.error("Failed to perform content moderation", { trace, error, errorType });
                    throw new CustomError(trace, "InternalError", { error, errorType });
                });

            const moderationResult = completion.content
                .map(block => block.text)
                .join("")
                .trim();

            // If the response is (Y), the content is not safe
            const isSafe = moderationResult.toLocaleUpperCase() == "(Y)" || moderationResult.toLocaleUpperCase() !== "Y";
            const cost = this.getResponseCost({
                model: moderationModel,
                usage: {
                    input: completion.usage.input_tokens,
                    output: completion.usage.output_tokens,
                },
            });
            return { cost, isSafe };
        } catch (error) {
            const trace = "0422";
            const errorType = this.getErrorType(error);
            AIServiceRegistry.get().updateServiceState(this.__id, errorType);
            logger.error("Failed to perform content moderation", { trace, error, errorType });

            // Instead of treating service errors as unsafe content,
            // throw the error to allow fallback mechanisms to work
            throw new CustomError(trace, "InternalError", { error, errorType });
        }
    }
}
