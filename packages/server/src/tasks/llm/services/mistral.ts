import { MistralModel } from "@local/shared";
import MistralClient, { ChatCompletionResponse } from "@mistralai/mistralai";
import { CustomError } from "../../../events/error";
import { logger } from "../../../events/logger";
import { LlmServiceErrorType, LlmServiceId, LlmServiceRegistry } from "../registry";
import { EstimateTokensParams, GenerateContextParams, GenerateResponseParams, GetConfigObjectParams, GetResponseCostParams, LanguageModelContext, LanguageModelMessage, LanguageModelService, generateDefaultContext, getDefaultConfigObject, tokenEstimationDefault } from "../service";

type MistralTokenModel = "default";

/** Cost in cents per 1_000_000 input tokens */
const inputCosts: Record<MistralModel, number> = {
    [MistralModel.Mistral8x7b]: 70,
    [MistralModel.Mistral7b]: 25,
};
/** Cost in cents per 1_000_000 output tokens */
const outputCosts: Record<MistralModel, number> = {
    [MistralModel.Mistral8x7b]: 70,
    [MistralModel.Mistral7b]: 25,
};

export class MistralService implements LanguageModelService<MistralModel, MistralTokenModel> {
    public __id = LlmServiceId.Mistral;
    private client: MistralClient;
    private defaultModel: MistralModel = MistralModel.Mistral7b;

    constructor() {
        this.client = new MistralClient(process.env.MISTRAL_API_KEY);
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

        // Ensure roles alternate between "user" and "assistant". This is a requirement of the Mistral API.
        const alternatingMessages: LanguageModelMessage[] = [];
        const messagesWithResponding = params.respondingToMessage ? [...messages, { role: "user" as const, content: params.respondingToMessage.text }] : messages;
        let lastRole: LanguageModelMessage["role"] = "assistant";
        for (const { role, content } of messagesWithResponding) {
            // Skip empty messages. This is another requirement of the Mistral API.
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

        // Ensure first message is from the user. This is another requirement of the Mistral API.
        if (alternatingMessages.length && alternatingMessages[0].role === "assistant") {
            alternatingMessages.shift();
        }

        const messagesWithContext = [
            // Add system message first
            { role: "system" as const, content: systemMessage },
            // Add other messages
            ...alternatingMessages.map(({ role, content }) => ({ role, content })),
        ] as LanguageModelMessage[];

        return { messages: messagesWithContext, systemMessage };
    }

    async generateResponse({
        messages,
        model,
        userData,
    }: GenerateResponseParams) {
        // Generate response
        const params = {
            messages,
            model,
            max_tokens: 1024, // Adjust as needed
        };
        const completion: ChatCompletionResponse = await this.client
            .chat(params)
            .catch((error) => {
                const trace = "0249";
                const errorType = this.getErrorType(error);
                LlmServiceRegistry.get().updateServiceState(this.__id, errorType);
                logger.error("Failed to call Mistral", { trace, error, errorType });
                throw new CustomError(trace, "InternalError", userData.languages, { error, errorType });
            });
        const message = completion.choices[0].message.content ?? "";
        const cost = this.getResponseCost({
            model,
            usage: {
                input: completion.usage.prompt_tokens,
                output: completion.usage?.completion_tokens,
            },
        });
        return { message, cost };
    }

    async *generateResponseStreaming({
        messages,
        model,
    }: GenerateResponseParams) {
        const params = {
            model,
            messages,
        };

        let accumulatedMessage = "";
        // NOTE: Mistral's API might not provide token usage when streaming. You'll need to estimate it yourself.
        const inputTokens = this.estimateTokens({ model, text: messages.map(m => m.content).join("\n") }).tokens;
        let accumulatedOutputTokens = 0;

        try {
            // Create the stream
            const chatStreamResponse = this.client.chatStream(params);
            for await (const chunk of chatStreamResponse) {
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
            const trace = "0324";
            const errorType = this.getErrorType(error);
            LlmServiceRegistry.get().updateServiceState(this.__id, errorType);
            logger.error("Failed to stream from Mistral", { trace, error, errorType });

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

    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    getContextSize(_model?: string | null) {
        return 8192;
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
        if (model.includes("8x7b")) return MistralModel.Mistral8x7b;
        if (model.includes("mistral-7b")) return MistralModel.Mistral7b;
        return this.defaultModel;
    }

    getErrorType(error: unknown) {
        return LlmServiceErrorType.Authentication; //TODO can't find error codes
    }
}
