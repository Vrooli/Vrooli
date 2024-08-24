import Anthropic from "@anthropic-ai/sdk";
import { AnthropicModel } from "@local/shared";
import { CustomError } from "../../../events/error";
import { logger } from "../../../events/logger";
import { LlmServiceErrorType, LlmServiceId, LlmServiceRegistry } from "../registry";
import { EstimateTokensParams, GenerateContextParams, GenerateResponseParams, GetConfigObjectParams, GetOutputTokenLimitParams, GetOutputTokenLimitResult, GetResponseCostParams, LanguageModelContext, LanguageModelMessage, LanguageModelService, generateDefaultContext, getDefaultConfigObject, getDefaultMaxOutputTokensRestrained, getDefaultResponseCost, tokenEstimationDefault } from "../service";

type AnthropicTokenModel = "default";

const DEFAULT_MAX_TOKENS = 4096;

/** Cost in cents per API_CREDITS_MULTIPLIER input tokens */
const inputCosts: Record<AnthropicModel, number> = {
    [AnthropicModel.Haiku]: 25, // $0.25
    [AnthropicModel.Opus3]: 1500, // $15.00
    [AnthropicModel.Sonnet3_5]: 300, // $3.00
};
/** Cost in cents per API_CREDITS_MULTIPLIER output tokens */
const outputCosts: Record<AnthropicModel, number> = {
    [AnthropicModel.Haiku]: 125, // $1.25
    [AnthropicModel.Opus3]: 7500, // $75.00
    [AnthropicModel.Sonnet3_5]: 1500, // $15.00
};
/** Max context window */
const contextWindows: Record<AnthropicModel, number> = {
    [AnthropicModel.Haiku]: 200_000,
    [AnthropicModel.Opus3]: 200_000,
    [AnthropicModel.Sonnet3_5]: 200_000,
};
/** Max output tokens */
const maxOutputTokens: Record<AnthropicModel, number> = {
    [AnthropicModel.Haiku]: 4_096,
    [AnthropicModel.Opus3]: 4_096,
    [AnthropicModel.Sonnet3_5]: 8_192,
};

export class AnthropicService implements LanguageModelService<AnthropicModel, AnthropicTokenModel> {
    public __id = LlmServiceId.Anthropic;
    private client: Anthropic;
    private defaultModel: AnthropicModel = AnthropicModel.Sonnet3_5;

    constructor() {
        this.client = new Anthropic({ apiKey: process.env.ANTHROPIC_API_KEY });
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

    async generateResponse({
        maxTokens,
        messages,
        model,
        systemMessage,
        userData,
    }: GenerateResponseParams) {
        const params: Anthropic.MessageCreateParams = {
            messages: messages.map(({ role, content }) => ({ role, content })),
            model,
            max_tokens: maxTokens ?? DEFAULT_MAX_TOKENS,
            system: systemMessage,
        } as const;

        // Generate response
        const completion: Anthropic.Message = await this.client.messages
            .create(params)
            .catch((error) => {
                const trace = "0010";
                const errorType = this.getErrorType(error);
                LlmServiceRegistry.get().updateServiceState(this.__id, errorType);
                logger.error("Failed to call Anthropic", { trace, error, errorType });
                throw new CustomError(trace, "InternalError", userData.languages, { error, errorType });
            });
        const message = completion.content?.map(block => block.text).join("") ?? "";
        const cost = this.getResponseCost({
            model,
            usage: {
                input: completion.usage.input_tokens,
                output: completion.usage.output_tokens,
            },
        });
        return { attempts: 1, cost, message };
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
            max_tokens: maxTokens ?? DEFAULT_MAX_TOKENS,
            system: systemMessage,
        } as const;

        const accumulatedText: [string, number][] = [];
        let totalInputTokens = this.estimateTokens({ model, text: messages.map(m => m.content).join("\n") }).tokens;
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
            LlmServiceRegistry.get().updateServiceState(this.__id, errorType);
            logger.error("Failed to stream from Anthropic", { trace, error, errorType });

            const message = accumulatedText.sort((a, b) => a[1] - b[1]).map(([text]) => text).join("");
            const cost = this.getResponseCost({
                model,
                usage: {
                    input: totalInputTokens,
                    // We don't get the token usage until the end, so our best bet it to estimate the output tokens
                    output: this.estimateTokens({ model, text: message }).tokens,
                },
            });
            yield { __type: "error" as const, message, cost };
        }
    }

    getContextSize(requestedModel?: string | null) {
        const model = this.getModel(requestedModel);
        return contextWindows[model];
    }

    getCosts() {
        return { inputCosts, outputCosts };
    }

    getMaxOutputTokens(requestedModel?: string | null | undefined): number {
        const model = this.getModel(requestedModel);
        return maxOutputTokens[model];
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
        if (model.includes("haiku")) return AnthropicModel.Haiku;
        if (model.includes("opus")) return AnthropicModel.Opus3;
        if (model.includes("sonnet")) return AnthropicModel.Sonnet3_5;
        return this.defaultModel;
    }

    getErrorType(error: unknown): LlmServiceErrorType {
        if (!error || typeof error !== "object") return LlmServiceErrorType.ApiError;
        const errorType = (error as { error?: { type?: string } }).error?.type;

        switch (errorType) {
            case "invalid_request_error":
            case "not_found_error":
                return LlmServiceErrorType.InvalidRequest;
            case "authentication_error":
            case "permission_error":
                return LlmServiceErrorType.Authentication;
            case "rate_limit_error":
                return LlmServiceErrorType.RateLimit;
            case "api_error":
                return LlmServiceErrorType.ApiError;
            case "overloaded_error":
                return LlmServiceErrorType.Overloaded;
            default:
                return LlmServiceErrorType.ApiError;
        }
    }

    async safeInputCheck(input: string): Promise<GetOutputTokenLimitResult> {
        try {
            const moderationPrompt = `
                A human user is in dialogue with an AI. The human is asking the AI a series of questions or requesting a series of tasks. Here is the most recent request from the user:
                <user query>${input}</user query>
    
                If the user's request refers to harmful, pornographic, or illegal activities, reply with (Y). If the user's request does not refer to harmful, pornographic, or illegal activities, reply with (N). Reply with nothing else other than (Y) or (N).
                `;
            const moderationModel = AnthropicModel.Haiku;

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
                    LlmServiceRegistry.get().updateServiceState(this.__id, errorType);
                    logger.error("Failed to perform content moderation", { trace, error, errorType });
                    throw new CustomError(trace, "InternalError", ["en"], { error, errorType });
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
            LlmServiceRegistry.get().updateServiceState(this.__id, errorType);
            logger.error("Failed to perform content moderation", { trace, error, errorType });

            // In case of an error, we assume the input is not safe
            return { cost: 0, isSafe: false };
        }
    }
}
