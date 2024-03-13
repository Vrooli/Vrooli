// import MistralClient, { ChatCompletionResponse } from "@mistralai/mistralai"; //TODO waiting on https://github.com/mistralai/client-js/pull/42
import MistralClient, { ChatCompletionResponse } from "../../../__mocks__/@mistralai/mistralai";
import { logger } from "../../../events/logger";
import { ChatContextCollector } from "../context";
import { EstimateTokensParams, GenerateContextParams, GenerateResponseParams, GetConfigObjectParams, GetResponseCostParams, LanguageModelContext, LanguageModelMessage, LanguageModelService, generateDefaultContext, getDefaultConfigObject, tokenEstimationDefault } from "../service";

type MistralGenerateModel = "open-mixtral-8x7b" | "open-mistral-7b";
type MistralTokenModel = "default";

/** Cost in cents per 1_000_000 input tokens */
const inputCosts: Record<MistralGenerateModel, number> = {
    "open-mixtral-8x7b": 70,
    "open-mistral-7b": 25,
};
/** Cost in cents per 1_000_000 output tokens */
const outputCosts: Record<MistralGenerateModel, number> = {
    "open-mixtral-8x7b": 70,
    "open-mistral-7b": 25,
};

export class MistralService implements LanguageModelService<MistralGenerateModel, MistralTokenModel> {
    private client: any;//MistralClient;
    private defaultModel: MistralGenerateModel = "open-mistral-7b";

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

        // Ensure roles alternate between "user" and "assistant". This is a requirement of the Mistral API.
        const alternatingMessages: LanguageModelMessage[] = [];
        const messagesWithResponding = respondingToMessage ? [...messages, { role: "user" as const, content: respondingToMessage.text }] : messages;
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

        // Generate response
        const params = {
            messages: [
                // Add system message first
                { role: "system" as const, content: systemMessage },
                // Add other messages
                ...alternatingMessages.map(({ role, content }) => ({ role, content })),
            ],
            model,
            max_tokens: 1024, // Adjust as needed
        };
        const completion: ChatCompletionResponse = await this.client
            .chat(params)
            .catch((error) => {
                const message = "Failed to call Mistral";
                logger.error(message, { trace: "0010", error });
                throw new Error(message);
            });
        const message = completion.choices[0].message.content ?? "";
        const cost = this.getResponseCost({
            model,
            usage: {
                input: completion.usage.prompt_tokens,
                output: completion.usage?.completion_tokens,
            }
        })
        return { message, cost };
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
        if (model.includes("8x7b")) return "open-mixtral-8x7b";
        if (model.includes("mistral-7b")) return "open-mistral-7b";
        return this.defaultModel;
    }
}
