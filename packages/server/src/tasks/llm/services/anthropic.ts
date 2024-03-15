import Anthropic from "@anthropic-ai/sdk";
import { logger } from "../../../events/logger";
import { ChatContextCollector } from "../context";
import { EstimateTokensParams, GenerateContextParams, GenerateResponseParams, GetConfigObjectParams, GetResponseCostParams, LanguageModelContext, LanguageModelMessage, LanguageModelService, generateDefaultContext, getDefaultConfigObject, tokenEstimationDefault } from "../service";

type AnthropicGenerateModel = "claude-3-opus-20240229" | "claude-3-sonnet-20240229";
type AnthropicTokenModel = "default";

/** Cost in cents per 1_000_000 input tokens */
const inputCosts: Record<AnthropicGenerateModel, number> = {
    "claude-3-opus-20240229": 1500,
    "claude-3-sonnet-20240229": 300,
};
/** Cost in cents per 1_000_000 output tokens */
const outputCosts: Record<AnthropicGenerateModel, number> = {
    "claude-3-opus-20240229": 7500,
    "claude-3-sonnet-20240229": 1500,
};

export class AnthropicService implements LanguageModelService<AnthropicGenerateModel, AnthropicTokenModel> {
    private client: Anthropic;
    private defaultModel: AnthropicGenerateModel = "claude-3-sonnet-20240229";

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
            force,
            messageContextInfo,
            model,
            participantsData,
            respondingBotId,
            respondingBotConfig,
            task,
            userData,
        });

        // Ensure roles alternate between "user" and "assistant". This is a requirement of the Anthropic API.
        const alternatingMessages: LanguageModelMessage[] = [];
        const messagesWithResponding = respondingToMessage ? [...messages, { role: "user" as const, content: respondingToMessage.text }] : messages;
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

        const params: Anthropic.MessageCreateParams = {
            messages: alternatingMessages.map(({ role, content }) => ({ role, content })),
            model,
            max_tokens: 1024, // Adjust as needed
            system: systemMessage,
        };

        // Generate response
        const completion: Anthropic.Message = await this.client.messages
            .create(params)
            .catch((error) => {
                const message = "Failed to call Anthropic";
                logger.error(message, { trace: "0010", error });
                throw new Error(message);
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
        if (model.includes("opus")) return "claude-3-opus-20240229";
        if (model.includes("sonnet")) return "claude-3-sonnet-20240229";
        return this.defaultModel;
    }
}
