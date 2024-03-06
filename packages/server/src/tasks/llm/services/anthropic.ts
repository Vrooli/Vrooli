import Anthropic from "@anthropic-ai/sdk";
import { logger } from "../../../events/logger";
import { ChatContextCollector } from "../context";
import { EstimateTokensParams, GenerateContextParams, GenerateResponseParams, GetConfigObjectParams, LanguageModelContext, LanguageModelMessage, LanguageModelService, generateDefaultContext, getDefaultConfigObject, tokenEstimationDefault } from "../service";

type AnthropicGenerateModel = "claude-3-opus-20240229" | "claude-3-sonnet-20240229";
type AnthropicTokenModel = "default";

export class AnthropicService implements LanguageModelService<AnthropicGenerateModel, AnthropicTokenModel> {
    private anthropic: Anthropic;
    private defaultModel: AnthropicGenerateModel = "claude-3-opus-20240229";

    constructor() {
        this.anthropic = new Anthropic({ apiKey: process.env.ANTHROPIC_API_KEY });
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
        if (alternatingMessages[0].role === "assistant") {
            alternatingMessages.shift();
        }

        const params: Anthropic.MessageCreateParams = {
            messages: alternatingMessages.map(({ role, content }) => ({ role, content })),
            model,
            max_tokens: 1024, // Adjust as needed
            system: systemMessage,
        };

        const message: Anthropic.Message = await this.anthropic.messages
            .create(params)
            .catch((error) => {
                const message = "Failed to call Anthropic";
                logger.error(message, { trace: "0010", error });
                throw new Error(message);
            });
        const responseText = message.content?.map(block => block.text).join("") ?? "";
        return responseText;
    }

    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    getContextSize(_requestedModel?: string | null) {
        return 200000;
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
        if (requestedModel.includes("opus")) return "claude-3-opus-20240229";
        if (requestedModel.includes("sonnet")) return "claude-3-sonnet-20240229";
        return this.defaultModel;
    }
}
