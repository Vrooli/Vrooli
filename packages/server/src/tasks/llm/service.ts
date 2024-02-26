import { BotSettings } from "@local/shared";
import { PreMapUserData } from "../../models/base/chatMessage";
import { SessionUserToken } from "../../types";
import { withPrisma } from "../../utils/withPrisma";
import { LlmTask } from "./config";
import { MessageContextInfo } from "./context";
import { OpenAIService } from "./services/openai";

type SimpleChatMessageData = {
    id: string;
    translations: { language: string, text: string }[];
}

/**
 * Basic token estimation, which can be used as a placeholder 
 * until a more advanced one is implemented
 * @param text The text to estimate the amount of tokens for
 * @returns The name of the token estimation model, and the estimated amount of tokens
 */
export const tokenEstimationDefault = (text: string) => {
    const words = text.split(" ");
    let tokens = 0;
    for (const word of words) {
        // Add token for each 5 characters
        tokens += Math.floor(word.length / 5);
        // Also increment for whitespace
        tokens++;
    }
    return ["default", tokens] as const;
};

/**
 * Fetches messages from the database for given message IDs.
 * 
 * @param messageIds Array of message IDs
 * @returns An array of message objects
 */
export const fetchMessagesFromDatabase = async (messageIds: string[]): Promise<SimpleChatMessageData[]> => {
    let messages: SimpleChatMessageData[] = [];

    await withPrisma({
        process: async (prisma) => {
            messages = await prisma.chat_message.findMany({
                where: {
                    id: { in: messageIds },
                },
                select: {
                    id: true,
                    translations: {
                        select: {
                            language: true,
                            text: true,
                        },
                    },
                },
            });
        },
        trace: "0080",
    });

    return messages;
};

type LanguageModelMessage = Partial<MessageContextInfo> & {
    role?: string;
    text: string;
}
export type LanguageModelContext = {
    messages: LanguageModelMessage[];
    systemMessage: string;
}

export type EstimateTokensParams = {
    text: string;
    requestedModel: string;
}
export type GenerateContextParams = {
    respondingBotId: string;
    respondingBotConfig: BotSettings;
    messageContextInfo: MessageContextInfo[];
    participantsData: Record<string, PreMapUserData>;
    task: LlmTask;
    /** 
     * Determines if context should be written to force the response to be a command 
     */
    force: boolean;
    userData: SessionUserToken;
    requestedModel: string;
}
export type GenerateResponseParams = {
    chatId: string;
    respondingToMessageId: string;
    respondingToMessageContent: string;
    respondingBotId: string;
    respondingBotConfig: BotSettings;
    task: LlmTask;
    /** 
     * Determines if context should be written to force the response to be a command 
     */
    force: boolean;
    userData: SessionUserToken;
}

export interface LanguageModelService<GenerateNameType extends string, TokenNameType> {
    /** Estimate the amount of tokens a string is */
    estimateTokens(params: EstimateTokensParams): readonly [TokenNameType, number];
    /** Generates context prompt from messages */
    generateContext(params: GenerateContextParams): Promise<LanguageModelContext>;
    /** Generate a message response */
    generateResponse(params: GenerateResponseParams): Promise<string>;
    /** @returns the context size of the model */
    getContextSize(requestedModel?: string | null): number;
    /** @returns the estimation method for the model */
    getEstimationMethod(requestedModel?: string | null): TokenNameType;
    /** @returns a list of token estimation types used by this service */
    getEstimationTypes(): readonly TokenNameType[];
    /** Convert a preferred model to an available one */
    getModel(requestedModel?: string | null): GenerateNameType;
}

export const getLanguageModelService = (botSettings: BotSettings): LanguageModelService<any, any> => {
    switch (botSettings.model) {
        default:
            return new OpenAIService();
    }
};
