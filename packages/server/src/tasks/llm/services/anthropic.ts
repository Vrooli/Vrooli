import Anthropic from "@anthropic-ai/sdk";
import { CustomError } from "../../../events/error";
import { logger } from "../../../events/logger";
import { AnthropicModel, LlmServiceErrorType, LlmServiceId, LlmServiceRegistry } from "../registry";
import { EstimateTokensParams, GenerateContextParams, GenerateResponseParams, GetConfigObjectParams, GetResponseCostParams, LanguageModelContext, LanguageModelMessage, LanguageModelService, generateDefaultContext, getDefaultConfigObject, tokenEstimationDefault } from "../service";

type AnthropicTokenModel = "default";

/** Cost in cents per 1_000_000 input tokens */
const inputCosts: Record<AnthropicModel, number> = {
    [AnthropicModel.Opus]: 1500,
    [AnthropicModel.Sonnet]: 300,
};
/** Cost in cents per 1_000_000 output tokens */
const outputCosts: Record<AnthropicModel, number> = {
    [AnthropicModel.Opus]: 7500,
    [AnthropicModel.Sonnet]: 1500,
};

export class AnthropicService implements LanguageModelService<AnthropicModel, AnthropicTokenModel> {
    public __id = LlmServiceId.Anthropic;
    private client: Anthropic;
    private defaultModel: AnthropicModel = AnthropicModel.Sonnet;

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
        const messagesWithResponding = params.respondingToMessage
            ? [...messages, { role: "user" as const, content: params.respondingToMessage.text }]
            : messages;
        let lastRole: LanguageModelMessage["role"] = "assistant";
        for (const { role, content } of messagesWithResponding) {
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
        messages,
        model,
        systemMessage,
        userData,
    }: GenerateResponseParams) {
        const params: Anthropic.MessageCreateParams = {
            messages: messages.map(({ role, content }) => ({ role, content })),
            model,
            max_tokens: 1024, // Adjust as needed
            system: systemMessage,
        };

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
        return { message, cost };
    }

    async *generateResponseStreaming({
        messages,
        model,
        systemMessage,
        userData,
    }: GenerateResponseParams) {
        const params: Anthropic.MessageCreateParams = {
            messages: messages.map(({ role, content }) => ({ role, content })),
            model,
            max_tokens: 1024, // Adjust as needed
            system: systemMessage,
        };

        let accumulatedText: [string, number][] = [];
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
            logger.error("Failed to stream from OpenAI", { trace, error, errorType });

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

    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    getContextSize(_requestedModel?: string | null) {
        return 200000;
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
        if (model.includes("opus")) return AnthropicModel.Opus;
        if (model.includes("sonnet")) return AnthropicModel.Sonnet;
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
}
