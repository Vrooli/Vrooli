import { TaskContextInfo } from "@local/shared";
import { RedisClientType } from "redis";
import { prismaInstance } from "../../db/instance";
import { logger } from "../../events/logger";
import { withRedis } from "../../redisConn";
import { UI_URL } from "../../server";
import { PreMapMessageDataCreate, PreMapMessageDataDelete, PreMapMessageDataUpdate, PreMapUserData } from "../../utils/chat";
import { LanguageModelService } from "./service";
import { OpenAIService } from "./services/openai";

type CachedChatMessage = {
    id: string;
    userId?: string;
    parentId?: string;
    translatedTokenCounts: string;
}
type ContextInfoBase = {
    language: string;
    tokenSize: number;
    userId: string | null;
}
export type MessageContextInfo = ContextInfoBase & {
    __type: "message";
    messageId: string;
}
export type TextContextInfo = ContextInfoBase & {
    __type: "text";
    text: string;
}
export type ContextInfo = MessageContextInfo | TextContextInfo;

type TokenCounts = Record<string, Record<string, number>>;

export type CollectMessageContextInfoParams = {
    /** The chat ID where the message was sent */
    chatId?: string | null;
    /** User's languages */
    languages: string[];
    /** 
     * The message ID to start fetching context from. 
     * The context info will be all messages leading up and including this message, 
     * which fits within the token limits of the language model. 
     * 
     * NOTE 1: If you include `taskMessage`, the context info will also include this message.
     */
    latestMessage?: string | null;
    /**
     * The name of the model to use for token estimation and context size.
     */
    model: string;
    /**
     * An additional task message to include in the context info, if needed. 
     * This is useful for things like autoFill and running routines, among other things.
     */
    taskMessage?: string | null;
};

/**
 * ChatContextManager manages the writing and updating of chat message data in Redis.
 * This class is responsible for maintaining the integrity and structure of chat messages
 * within the cache, ensuring that new messages, edits, and branching are properly handled.
 */
export class ChatContextManager {
    private languageModelService: LanguageModelService<string, string>;

    constructor(languageModelService?: LanguageModelService<string, string>) {
        this.languageModelService = languageModelService ?? new OpenAIService();
    }

    async addMessage(data: PreMapMessageDataCreate): Promise<void> {
        const { chatId, messageId, parentId, translations, userId } = data;
        await withRedis({
            process: async (redisClient) => {
                if (!redisClient) return;

                const messageData: CachedChatMessage = {
                    id: messageId,
                    translatedTokenCounts: JSON.stringify(this.calculateTokenCounts(translations, ...this.languageModelService.getEstimationTypes())),
                };
                if (parentId) {
                    messageData.parentId = parentId;
                }
                if (userId) {
                    messageData.userId = userId;
                }

                await redisClient.hSet(`message:${messageId}`, messageData);
                await redisClient.zAdd(`chat:${chatId}`, { score: Date.now(), value: messageId });
                if (parentId) {
                    await redisClient.sAdd(`children:${parentId}`, messageId);
                }
            },
            trace: "0076",
        });
    }

    async editMessage(data: PreMapMessageDataUpdate): Promise<void> {
        const { chatId, messageId, parentId, translations, userId } = data;
        await withRedis({
            process: async (redisClient) => {
                if (!redisClient) return;

                // Find existing message data
                let existingData = await redisClient.hGetAll(`message:${messageId}`);
                let shouldDeleteOldData = false;
                if (typeof existingData !== "object" || Object.keys(existingData).length === 0) {
                    logger.error("Failed to find existing message data", { trace: "0068", messageId });
                    existingData = {};
                    shouldDeleteOldData = true;
                }
                const existingMessageData = { ...existingData };
                // Delete invalid data if necessary
                if (shouldDeleteOldData) {
                    await redisClient.del(`message:${messageId}`);
                }
                // If we don't have any data and we weren't provided sufficient data to create a full message, return
                if (Object.keys(existingMessageData).length === 0 && (!translations || translations.length === 0 || !userId)) {
                    return;
                }
                let existingTranslations: Record<string, Record<string, number>> = {};
                try {
                    if (existingMessageData?.translatedTokenCounts) {
                        existingTranslations = JSON.parse(existingMessageData.translatedTokenCounts);
                    }
                } catch (error) {
                    logger.error("Failed to parse existing translations", { trace: "0069", error, existingMessageData });
                }
                const messageData: CachedChatMessage = {
                    id: messageId,
                    translatedTokenCounts: JSON.stringify({
                        ...existingTranslations,
                        ...this.calculateTokenCounts(translations || [], ...this.languageModelService.getEstimationTypes()),
                    }),
                };
                // hSet keeps existing fields if not provided, so we only need to update the fields that have changed
                if (parentId && parentId !== existingMessageData.parentId) {
                    messageData.parentId = parentId;
                }
                if (userId && userId !== existingMessageData.userId) {
                    messageData.userId = userId;
                }
                // Add/Update the message data
                await redisClient.hSet(`message:${messageId}`, messageData);
                // Check if parentId has changed and handle accordingly
                if (messageData.parentId) {
                    if (existingMessageData.parentId) {
                        await redisClient.sRem(`children:${existingMessageData.parentId}`, messageData.id); // Remove from old parent's children set
                    }
                    await redisClient.sAdd(`children:${messageData.parentId}`, messageData.id); // Add to new parent's children set
                }
            },
            trace: "0077",
        });
    }

    async deleteMessage(data: PreMapMessageDataDelete): Promise<void> {
        const { chatId, messageId } = data;
        await withRedis({
            process: async (redisClient) => {
                if (!redisClient) return;
                // Find existing data
                let messageToDelete = await redisClient.hGetAll(`message:${messageId}`);
                if (typeof messageToDelete !== "object" || Object.keys(messageToDelete).length === 0) {
                    logger.error("Failed to find message data to delete", { trace: "0066", messageId });
                    messageToDelete = {};
                }
                // Find children IDs of the message to delete
                let children = await redisClient.sMembers(`children:${messageId}`);
                if (!Array.isArray(children)) {
                    logger.error("Failed to find children of message to delete", { trace: "0065", messageId });
                    children = [];
                }

                // Update parent reference for each child message
                for (const childId of children) {
                    // Update parent reference to the parent of the message being deleted
                    await redisClient.hSet(`message:${childId}`, "parentId", messageToDelete.parentId);

                    // If the parent exists, add the child to its children set
                    if (messageToDelete.parentId) {
                        await redisClient.sAdd(`children:${messageToDelete.parentId}`, childId);
                    }
                }

                // Delete the message and its reference in the children set
                await redisClient.del([`message:${messageId}`, `children:${messageId}`]);
                await redisClient.zRem(`chat:${chatId}`, messageId);
            },
            trace: "0078",
        });
    }

    async deleteChat(chatId: string): Promise<void> {
        await withRedis({
            process: async (redisClient) => {
                if (!redisClient) return;
                // Retrieve all message IDs associated with the chat
                const messageIds = await redisClient.zRange(`chat:${chatId}`, 0, -1);

                // Prepare keys for deletion (messages and their child references)
                const keysToDelete = messageIds.map(messageId => [`message:${messageId}`, `children:${messageId}`]).flat();

                // Delete all messages and their child references
                if (keysToDelete.length > 0) {
                    await redisClient.del(keysToDelete);
                }

                // Remove the sorted set for the chat
                await redisClient.del(`chat:${chatId}`);
            },
            trace: "0079",
        });
    }

    /**
     * Calculates token counts for each translation of a message for multiple estimation methods.
     * 
     * @param translations Array of message translations
     * @param estimationMethods Each token estimation method to use
     * @returns An object with languages as keys and token counts for each estimation method as values
     */
    public calculateTokenCounts(translations: readonly { language: string, text: string }[], ...estimationMethods: string[]): TokenCounts {
        const translatedTokenCounts = {};

        translations.forEach(translation => {
            if (!translatedTokenCounts[translation.language]) {
                translatedTokenCounts[translation.language] = {};
            }

            estimationMethods.forEach(method => {
                // Check if the method is valid; if not, fall back to a default method
                const validMethod = method ?? "default";

                // Ensure we only proceed if there's a method to use
                if (validMethod) {
                    // Safely handle the case where estimateTokens might return undefined
                    const estimation = this.languageModelService.estimateTokens({
                        text: translation.text,
                        model: validMethod,
                    });

                    translatedTokenCounts[translation.language][validMethod] = estimation?.tokens ?? 0;
                }
            });
        });

        return translatedTokenCounts;
    }
}

/**
 * ChatContextCollector manages chat message contexts for generating chatbot responses.
 * It uses Redis to retrieve chat messages and a LanguageModelService to handle token estimation.
 *
 * This class is essential for ensuring the chatbot's response generation is based on
 * an appropriate context window, considering the token limits of different language models.
 * It assumes a specific Redis schema with messages stored in hashes and ordered in sorted sets.
 */
export class ChatContextCollector {
    private languageModelService: LanguageModelService<string, string>;

    constructor(languageModelService: LanguageModelService<string, string>) {
        this.languageModelService = languageModelService;
    }

    /**
     * Collects messages to include in the context info for generating a response. 
     * Messages are collected starting from the latest message and going back, 
     * until the context size is reached or no more messages are found.
     * 
     * NOTE: This also adds the taskMessage to the context info if provided, which 
     * basically simulates a message in the chat history to tell the bot how to respond.
     */
    async collectMessageContextInfo({
        chatId,
        languages,
        model,
        latestMessage,
        taskMessage,
    }: CollectMessageContextInfoParams): Promise<ContextInfo[]> {
        const contextSize = this.languageModelService.getContextSize(model);
        let currentTokenCount = 0;
        const contextInfo: ContextInfo[] = [];

        // Add task message to context info if provided
        if (taskMessage) {
            const estimationMethod = this.languageModelService.getEstimationMethod(model);
            const translations = [{
                language: languages.length ? languages[0] : "en",
                text: taskMessage,
            }];
            const translatedTokenCounts = (new ChatContextManager(this.languageModelService)).calculateTokenCounts(translations, estimationMethod);
            const [estimatedTokens, language] = this.getTokenCountFromTranslatedCounts(translatedTokenCounts, estimationMethod, languages);

            if (estimatedTokens >= 0 && estimatedTokens <= contextSize) {
                contextInfo.push({
                    __type: "text",
                    tokenSize: estimatedTokens,
                    userId: null,
                    language,
                    text: taskMessage,
                });
            } else {
                logger.warning("Failed to estimate tokens for provided text", { trace: "0075", text: taskMessage });
            }
        }

        // Collect chat messages for context info, starting at `latestMessage` and going back
        if (chatId) {
            await withRedis({
                process: async (redisClient) => {
                    // Get provided message ID, or fetch if not provided
                    let currentMessageId: string | null = latestMessage ?? await this.getLatestMessageId(redisClient, chatId);

                    // If message ID is found, loop up message tree to collect context, 
                    // until context size is reached or no more messages are found
                    while (currentMessageId && currentTokenCount < contextSize) {
                        const messageDetails = await this.getMessageDetails(redisClient, currentMessageId);
                        if (!messageDetails) {
                            logger.warning("Failed to find message details", { trace: "0080", currentMessageId });
                            break; // Break the loop if message details are not found
                        }
                        const estimationMethod = this.languageModelService.getEstimationMethod(model);
                        const [estimatedTokens, language] = await this.getTokenCountForLanguages(redisClient, currentMessageId, estimationMethod, languages);

                        if (estimatedTokens >= 0) {
                            currentTokenCount += estimatedTokens;
                            if (currentTokenCount > contextSize) {
                                break; // Break the loop if the context size is exceeded
                            }

                            contextInfo.push({
                                __type: "message",
                                messageId: currentMessageId,
                                tokenSize: estimatedTokens,
                                userId: messageDetails.userId ?? null,
                                language,
                            });

                            currentMessageId = messageDetails.parentId ?? null;
                        } else {
                            logger.warning("Failed to estimate tokens for message", { trace: "0083", messageDetails });
                            break; // Break the loop if token estimation fails
                        }
                    }
                },
                trace: "0072",
            });
        }

        // Reverse the array to get the messages in chronological order
        contextInfo.reverse();

        // If there are no messages, log warning. This may indicate an problem, 
        // and may break some llm services
        if (contextInfo.length === 0) {
            logger.warning("No messages found in context. This may cause an error with response generation", { trace: "0239" });
        }

        return contextInfo;
    }

    async getLatestMessageId(redisClient: RedisClientType | null, chatId: string): Promise<string | null> {
        if (!redisClient) return null;
        // Retrieve the last element in the sorted set (the most recent message)
        const latestMessages = await redisClient.zRange(`chat:${chatId}`, -1, -1);
        return latestMessages.length > 0 ? latestMessages[0] : null;
    }

    async getMessageDetails(redisClient: RedisClientType | null, messageId: string): Promise<CachedChatMessage | null> {
        let messageData: CachedChatMessage | null = redisClient ? await redisClient.hGetAll(`message:${messageId}`) as CachedChatMessage : null;

        if (messageData === null || messageData === undefined || typeof messageData !== "object" || Object.keys(messageData).length === 0) {
            // Query from Prisma if not found in Redis
            try {
                const message = await prismaInstance.chat_message.findUnique({
                    where: { id: messageId },
                    select: {
                        id: true,
                        parent: {
                            select: {
                                id: true,
                            },
                        },
                        translations: {
                            select: {
                                language: true,
                                text: true,
                            },
                        },
                        user: {
                            select: {
                                id: true,
                            },
                        },
                    },
                });

                if (message) {
                    messageData = {
                        id: message.id,
                        translatedTokenCounts: JSON.stringify((new ChatContextManager(this.languageModelService)).calculateTokenCounts(message.translations, ...this.languageModelService.getEstimationTypes())),
                    };
                    if (message.parent) {
                        messageData.parentId = message.parent.id;
                    }
                    if (message.user) {
                        messageData.userId = message.user.id;
                    }

                    // Store the fetched data in Redis for future use
                    if (redisClient) {
                        await redisClient.hSet(`message:${messageId}`, messageData);
                    }
                } else {
                    throw new Error(`Failed to fetch message details for ID: ${messageId}`);
                }
            } catch (error) {
                logger.error("Caught error in getMessageDetails", { trace: "0073", error });
                return null;
            }
        }

        return {
            id: messageId,
            parentId: messageData.parentId,
            translatedTokenCounts: JSON.parse(messageData.translatedTokenCounts ?? "{}"),
            userId: messageData.userId,
        };
    }

    /**
     * Converts translated token counts into a tuple of [token count, language].
     * @param translatedTokenCounts The object containing token counts per language and estimation method.
     * @param estimationMethod The token estimation method.
     * @param preferredLanguages User's preferred languages.
     * @returns A tuple of the token count for the message and the language used.
     */
    getTokenCountFromTranslatedCounts(
        translatedTokenCounts: Record<string, Record<string, number>>,
        estimationMethod: string,
        preferredLanguages: string[],
    ): [number, string] {
        const languagesWithDefault = preferredLanguages.length === 0 ? ["en"] : preferredLanguages;

        // Check if token counts for preferred languages are available
        for (const language of languagesWithDefault) {
            if (translatedTokenCounts[language] && translatedTokenCounts[language][estimationMethod] !== undefined) {
                return [translatedTokenCounts[language][estimationMethod], language];
            }
        }

        // Fallback to the first available language if preferred languages are not available
        for (const language in translatedTokenCounts) {
            if (translatedTokenCounts[language][estimationMethod] !== undefined) {
                return [translatedTokenCounts[language][estimationMethod], language];
            }
        }

        return [-1, ""]; // Return -1 if no valid token count is found
    }


    /**
     * Tries to find the token count for the given estimation method and preferred languages.
     * If not found, uses the first available language. If none exist, returns -1.
     * Calculates and updates the token count if necessary.
     *
     * @param redisClient Redis client instance
     * @param messageId The ID of the message
     * @param estimationMethod The token estimation method
     * @param languages User's preferred languages
     * @returns The token count for the message and the language used
     */
    async getTokenCountForLanguages(
        redisClient: RedisClientType | null,
        messageId: string,
        estimationMethod: string,
        languages: string[],
    ): Promise<[number, string]> {
        const messageData = redisClient ? await redisClient.hGetAll(`message:${messageId}`) : null;
        let translatedTokenCounts = messageData ? JSON.parse(messageData.translatedTokenCounts ?? "{}") : {};
        const languagesWithDefault = languages.length === 0 ? ["en"] : languages;

        // Check if token counts for preferred languages are available in cache
        for (const language of languagesWithDefault) {
            if (translatedTokenCounts[language] && translatedTokenCounts[language][estimationMethod] !== undefined) {
                return [translatedTokenCounts[language][estimationMethod], language];
            }
        }

        // If token counts are not in cache, query database and update cache
        if (Object.keys(translatedTokenCounts).length === 0) {
            try {
                const message = await prismaInstance.chat_message.findUnique({
                    where: { id: messageId },
                    include: { translations: true },
                });
                if (message?.translations) {
                    translatedTokenCounts = (new ChatContextManager(this.languageModelService)).calculateTokenCounts(message.translations, estimationMethod);
                    if (redisClient) {
                        await redisClient.hSet(`message:${messageId}`, "translatedTokenCounts", JSON.stringify(translatedTokenCounts));
                    }
                } else {
                    return [-1, ""]; // Return -1 if no translations are found or on failure
                }
            } catch (error) {
                logger.error("Caught error in getTokenCountForLanguages", { trace: "0074", error });
            }
        }

        return this.getTokenCountFromTranslatedCounts(translatedTokenCounts, estimationMethod, languages);
    }
}

/**
 * Finds bot information for a given bot ID. 
 * First checks the redis cache, then falls back to the database.
 */
export async function getBotInfo(botId: string): Promise<PreMapUserData | null> {
    let botInfo: PreMapUserData | null = null;

    // Check Redis cache first
    await withRedis({
        process: async (redisClient) => {
            if (!redisClient) return;
            const rawData = await redisClient.hGetAll(`bot:${botId}`);
            if (rawData && Object.keys(rawData).length >= 4) {
                // Convert isBot back to boolean
                botInfo = {
                    ...rawData,
                    isBot: rawData.isBot === "true",
                } as unknown as PreMapUserData;
            }
        },
        trace: "0233",
    });

    // If not found in cache, query the database
    if (!botInfo || Object.keys(botInfo).length < 4) { // There are 4 fields in PreMapUserData. Can add better check if needed
        botInfo = await prismaInstance.user.findUnique({
            where: { id: botId },
            select: {
                id: true,
                name: true,
                isBot: true,
                botSettings: true,
            },
        }) as PreMapUserData;
        // Store the fetched data in Redis for future use
        await withRedis({
            process: async (redisClient) => {
                if (!redisClient) return;
                // Convert isBot to string, since Redis only accepts string values
                const botInfoForRedis = {
                    ...botInfo,
                    isBot: botInfo!.isBot.toString(),
                };
                await redisClient.hSet(`bot:${botId}`, botInfoForRedis);
                await redisClient.expire(`bot:${botId}`, 60 * 60 * 24); // Expire in 24 hours
            },
            trace: "0235",
        });
    }

    return botInfo;
}

/**
 * @returns Valid mentions in a message
 */
export function processMentions(
    messageContent: string,
    chat: { botParticipants?: string[] },
    bots: Pick<PreMapUserData, "id" | "name">[],
): string[] {
    // Find markdown links in the message
    const linkStrings = messageContent.match(/\[([^\]]+)\]\(([^)]+)\)/g);
    // Get the label and link for each link
    let links: { label: string, link: string }[] = linkStrings?.map(s => {
        const [label, link] = s.slice(1, -1).split("](");
        return { label, link };
    }) ?? [];

    // Filter out links where the that aren't a mention. Rules:
    // 1. Label must start with @
    // 2. Link must be to this site
    links = links.filter(l => {
        if (!l.label.startsWith("@")) return false;
        try {
            const url = new URL(l.link);
            return url.origin === UI_URL;
        } catch (e) {
            return false;
        }
    });

    let botsToRespond: string[] = [];
    // If one of the links is "@Everyone", all bots should respond
    if (links.some(l => l.label === "@Everyone")) {
        botsToRespond = chat?.botParticipants ?? [];
    }
    // Otherwise, find the bots that were mentioned by name (e.g. "@BotName")
    else {
        botsToRespond = links.map(l => {
            const botId = bots.find(b => b.name === l.label.slice(1))?.id;
            if (!botId) return null;
            return botId;
        }).filter(id => id !== null) as string[];

        botsToRespond = [...new Set(botsToRespond)];
    }
    return botsToRespond;
}

/**
 * Determines which bots should respond based on the message content and chat context.
 * 
 * Conditions:
 * 1. If the message content is blank (likely meaning the message was updated but not its actual content), then no bots should respond.
 * 2. If the message is not associated with your user ID, no bots should respond.
 * 3. If there are no bots in the chat, no bots should respond.
 * 4. If there is one bot in the chat and two participants (i.e., just you and the bot), the bot should respond.
 * 5. Otherwise, check the message to see if any bots were mentioned.
 * 
 * @param message - The message that might trigger bot responses.
 * @param messageFromUserId - The ID of the user who sent the message.
 * @param chat - Information about the chat where the message was sent.
 * @param bots - Information about the bots in the chat.
 * @param userId - The ID of the user sending the message.
 * @returns An array of botIds that should respond to the message.
 */
export function determineRespondingBots(
    message: string | null,
    messageFromUserId: string,
    chat: { botParticipants?: string[], participantsCount?: number },
    bots: Pick<PreMapUserData, "id" | "name">[],
    userId: string,
): string[] {
    if (
        !chat ||
        !message ||
        message.trim() === "" ||
        !messageFromUserId ||
        messageFromUserId !== userId ||
        bots.length === 0
    ) {
        return [];
    }

    if (bots.length === 1 && chat.participantsCount === 2) {
        return [bots[0].id];
    } else {
        return processMentions(message, chat, bots);
    }
}

/**
 * Stringifies a list of task context objects. These are used to provide context to the LLM,
 * typically when referencing a form or other client-side data.
 * @param taskLabel The label for the current task type, which may be used in the template 
 * @param taskContexts The list of task context objects to stringify
 * @param contextTemplateDefault The default template to use for stringifying the data
 * @returns A stringified list of task context objects
 */
export function stringifyTaskContexts(
    taskLabel: string,
    taskContexts: TaskContextInfo[],
    contextTemplateDefault?: string,
): string {
    let result = "";

    for (let i = 0; i < taskContexts.length; i++) {
        const { template, templateVariables, data } = taskContexts[i];

        let contextString = "";
        const stringifiedData = typeof data === "string" ? data : JSON.stringify(data, null, 2);

        // If the template is defined, replace template variables with actual data
        if (template || contextTemplateDefault) {
            contextString = template || contextTemplateDefault || "";

            // Define template variables
            const variables = {
                [templateVariables?.task || '<TASK>']: taskLabel,
                [templateVariables?.data || '<DATA>']: stringifiedData,
            };
            // Sort variables from longest to shortest, to avoid issues with overlapping variable names
            const sortedVariables = Object.entries(variables).sort(
                ([varNameA], [varNameB]) => varNameB.length - varNameA.length
            );
            // Replace each variable in the template
            for (const [varName, value] of sortedVariables) {
                // Escape regex special characters in variable names
                const escapedVarName = varName.replace(/[-\/\\^$*+?.()|[\]{}]/g, '\\$&');
                const regex = new RegExp(escapedVarName, 'g');
                contextString = contextString.replace(regex, value);
            }
        }
        // Otherwise, default to displaying the data
        else {
            contextString = stringifiedData;
        }

        result += contextString;

        // Add spacing between contexts
        if (i < taskContexts.length - 1) {
            result += "\n\n";
        }
    }

    return result;
}
