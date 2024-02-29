import { BotSettings, BotSettingsTranslation } from "@local/shared";
import "openai/shims/node"; // NOTE: Make sure to save without formatting (use command palette for this), so that this import is above the openai import
import OpenAI from "openai";
import { logger } from "../../../events/logger";
import { SessionUserToken } from "../../../types";
import { objectToYaml } from "../../../utils";
import { bestTranslation } from "../../../utils/bestTranslation";
import { LlmTask, getStructuredTaskConfig } from "../config";
import { ChatContextCollector } from "../context";
import { EstimateTokensParams, GenerateContextParams, GenerateResponseParams, LanguageModelContext, LanguageModelService, fetchMessagesFromDatabase, tokenEstimationDefault } from "../service";

type OpenAIGenerateModel = "gpt-3.5-turbo" | "gpt-4";
type OpenAITokenModel = "default";
export class OpenAIService implements LanguageModelService<OpenAIGenerateModel, OpenAITokenModel> {
    private openai: OpenAI;
    private defaultModel: OpenAIGenerateModel = "gpt-3.5-turbo";

    constructor() {
        this.openai = new OpenAI({
            apiKey: process.env.OPENAI_API_KEY,
        });
    }

    estimateTokens({ text }: EstimateTokensParams) {
        return tokenEstimationDefault(text);
    }

    async getConfigObject(
        botSettings: BotSettings,
        userData: Pick<SessionUserToken, "languages">,
        task: LlmTask,
        force: boolean,
    ) {
        const translationsList = Object.entries(botSettings?.translations ?? {}).map(([language, translation]) => ({ language, ...translation })) as { language: string }[];
        const translation = (bestTranslation(translationsList, userData.languages) ?? {}) as BotSettingsTranslation;

        const name: string | undefined = botSettings.name;
        const initMessage = translation.startingMessage?.length ?
            translation.startingMessage :
            name ?
                `Hello👋, I'm ${name}. How can I help you today?` :
                "Hello👋, how can I help you today?";
        delete (translation as { language?: string }).language;

        const taskConfig = await getStructuredTaskConfig(task, force, userData.languages[0] ?? "en");
        const configObject = {
            ai_assistant: {
                metadata: {
                    // author: config.author ?? "Vrooli", // May add this in the future to credit the bot creator
                    name: botSettings.name ?? "Bot",
                },
                init_message: initMessage, //TODO only need for first message?
                personality: { ...translation },
                ...taskConfig,
            },
        };

        return configObject;
    }

    async generateContext({
        respondingBotConfig,
        messageContextInfo,
        participantsData,
        task,
        force,
        userData,
        requestedModel,
    }: GenerateContextParams): Promise<LanguageModelContext> {
        const messages: LanguageModelContext["messages"] = [];

        // Construct the initial YAML configuration message for relevant participants
        let systemMessage = "You are a helpful assistant for an app named Vrooli. Please follow the configuration below to best suit each user's needs:\n\n";
        const config = await this.getConfigObject(respondingBotConfig, userData, task, force);
        // Add yml for bot responding
        systemMessage += objectToYaml(config) + "\n";
        // We'll see if we need the other bot configs after testing
        // // Identify bots present in the message context, minus the one responding
        // const participantIdsInContext = new Set(messageContextInfo.filter(info => info.userId && info.userId !== respondingBotId).map(info => info.userId));
        // // Add yml for each bot in the context
        // if (participantIdsInContext.size > 0) {
        //     systemMessage += `There are other bots in this chat. Here are their configurations:\n\n`;
        // }

        // Calculate token size for the YAML configuration
        const systemMessageSize = this.estimateTokens({ text: systemMessage, requestedModel })[1];
        const maxContentSize = this.getContextSize(requestedModel);

        // Fetch messages from the database
        const messagesFromDB = await fetchMessagesFromDatabase(messageContextInfo.map(info => info.messageId));
        let currentTokenCount = systemMessageSize;

        // Add the YAML configuration as the first message if it doesn't exceed the context size
        if (currentTokenCount <= maxContentSize) {
            messages.push({ role: "system", text: systemMessage });
        }
        // Otherwise, omit context entirely
        else {
            return { messages: [], systemMessage: "" };
        }

        for (const contextInfo of messageContextInfo) {
            const messageData = messagesFromDB.find(message => message.id === contextInfo.messageId);
            if (!messageData) continue;

            const messageTranslation = messageData.translations.find(translation => translation.language === contextInfo.language);
            if (!messageTranslation) continue;

            const userName = contextInfo.userId ? participantsData[contextInfo.userId]?.name : undefined;
            const tokenSize = contextInfo.tokenSize + (userName?.length ? (userName.length / 2) : 0); // For now, add a rough buffer for displaying the user's name

            // Stop if adding this message would exceed the context size
            if (currentTokenCount + tokenSize > maxContentSize) {
                break;
            }

            // Otherwise, increment the token count and add the message
            currentTokenCount += tokenSize;
            messages.push({ role: "user", text: `${userName ? `${userName}: ` : ""}${messageTranslation.text}` });
        }

        return { messages, systemMessage };
    }

    async generateResponse({
        chatId,
        respondingToMessage,
        respondingBotId,
        respondingBotConfig,
        task,
        force,
        userData,
    }: GenerateResponseParams) {
        const model = this.getModel(respondingBotConfig?.model);
        const messageContextInfo = respondingToMessage ?
            await (new ChatContextCollector(this)).collectMessageContextInfo(chatId, model, userData.languages, respondingToMessage.id) :
            [];
        const context = await this.generateContext({
            respondingBotId,
            respondingBotConfig,
            messageContextInfo,
            participantsData: {},
            task,
            force,
            userData,
            requestedModel: model,
        });

        const params: OpenAI.Chat.ChatCompletionCreateParams = {
            messages: respondingToMessage ? [
                ...(context.messages.map(({ role, text }) => ({ role: role ?? "assistant", content: text })) as { role: "user" | "assistant", content: string }[]),
                { role: "user", content: respondingToMessage.text },
            ] : [],
            model,
            user: userData.name ?? undefined,
        };
        const chatCompletion: OpenAI.Chat.ChatCompletion = await this.openai.chat.completions
            .create(params)
            .catch((error) => {
                const message = "Failed to call OpenAI";
                logger.error(message, { trace: "0009", error });
                throw new Error(message);
            });
        return chatCompletion.choices[0].message.content ?? "";
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
