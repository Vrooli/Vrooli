import { DEFAULT_LANGUAGE, getTranslation } from "@local/shared";
import { prismaInstance } from "../db/instance";
import { logger } from "../events/logger";
import { SessionUserToken } from "../types";

/** Information for a message, collected in mutate.shape.pre */
export type PreMapMessageData = {
    /** ID of the chat this message belongs to */
    chatId: string | null;
    /** Content in user's preferred (or closest to preferred) language */
    content: string;
    id: string;
    isNew: boolean;
    /** Language code for message content */
    language: string;
    /** ID of the message which should appear directly before this one */
    parentId?: string;
    /** All translations */
    translations: { language: string, text: string }[];
    /** ID of the user who sent this message */
    userId: string | null;
}

/** Information for a message's corresponding chat, collected in mutate.shape.pre */
export type PreMapChatData = {
    /** User IDs for all bots in the chat */
    botParticipants?: string[],
    /** User IDs of participants being invited, which may be bots */
    potentialBotIds?: string[],
    /** User IDs of participants being deleted */
    participantsDelete?: string[],
    isNew: boolean,
    /** ID of the last message in the chat */
    lastMessageId?: string,
    /** Total number of participants in the chat, including bots, users, and the current user */
    participantsCount?: number,
    /** Determines configuration to provide for AI responses */
    task?: string, //TODO can find taask in inputsById[node.parent.id].task for creates, but need way for updates. 
    // Should be able to store not necessarily the first task, but the current one. Can add new row to chat table for this. 
    // Or better yet, make it a stack of contexts. Can use reminder list for this, since where it only focuses on one reminder at a time 
    // (and can even add multiple items to the reminder to keep track of multiple things at once). For example, could start off with 
    // no reminders, which defaults to normal AI configuration ("What would you like to do today?" type response). Then when it starts a task or routine, 
    // you add it as a reminder. When that task is complete, you remove it from the reminder list (or just mark it as complete) and move on to 
    // the highest index reminder in the list that isn't marked as complete. Will need way to add a link or resource to reminder items. For example, 
    // let's say you want to complete the routine "Start a Business". The chat will add a reminder with one item. The item will have a name of the routine,
    // a description with the context of why you want to complete it, and a link to the routine ID.
};

/** 
 * Fields that we'll use to set up context for bots and other users in the chat. 
 * Taken by combining parsed bot settings wiht other information 
 * from the user object.
 */
export type PreMapUserData = {
    botSettings: string,
    id: string,
    isBot: boolean,
    name: string;
};

export type ChatMessageBeforeDeletedData = {
    id: string,
    chatId: string | undefined,
    userId: string | undefined,
    parentId: string | undefined,
    childIds: string[],
}

export type CollectParticipantDataParams = {
    /** IDs of chats to collect information for */
    chatIds?: string[],
    /** 
     * IDs of messages in chats to collect information for, 
     * if the chat ID is not known
     */
    messageIds?: string[],
    /**
     * If true, will collect message information for each ID in messageIds.
     * 
     * NOTE: This will omit lastMessageId from the chat data.
     */
    includeMessageInfo?: boolean,
    /**
    * If true, will collect message information for the parent of each ID in messageIds.
    * 
    * NOTE: This will omit lastMessageId from the chat data.
    */
    includeMessageParentInfo?: boolean,
    /** Map of chat data, by ID */
    preMapChatData: Record<string, PreMapChatData>,
    /** Map of message data, by ID */
    preMapMessageData: Record<string, PreMapMessageData>,
    /** Map of user data, by ID */
    preMapUserData: Record<string, PreMapUserData>,
    /** The current user's data */
    userData: SessionUserToken,
};

type QueriedMessage = any;

// Fields required for basic message information
const basicMessageSelect = {
    id: true,
    chat: {
        select: {
            id: true,
        },
    },
    parent: {
        select: {
            id: true,
        },
    },
    translations: {
        select: {
            id: true,
            language: true,
            text: true,
        },
    },
    user: {
        select: {
            id: true,
            isBot: true,
        },
    },
} as Record<string, any>;

/**
 * Builds the message select query for getChatParticipantData
 * @param messageIds IDs of messages to select
 * @param includeMessageInfo If true, will include message information
 * @param includeMessageParentInfo If true, will include parent message information
 * @returns The message select query
 */
export const buildChatParticipantMessageQuery = (
    messageIds: string[],
    includeMessageInfo: boolean,
    includeMessageParentInfo: boolean,
): Record<string, any> => {
    // If we're not including message information, return a query to select the 
    // last message ID in the chat. This is used to populate PreMapChatData.lastMessageId
    if (!includeMessageInfo && !includeMessageParentInfo) {
        return {
            orderBy: { sequence: "desc" },
            take: 1,
            select: {
                id: true,
            },
        };
    }
    // Otherwise, we'll be looking for data to create PreMapMessageData.
    // Construct select query based on which messages we want to include
    let messageSelect = JSON.parse(JSON.stringify(basicMessageSelect));
    // If selecting both message and parent message information, update query 
    // to get more parent message information
    if (includeMessageParentInfo && includeMessageInfo) {
        messageSelect.parent.select = JSON.parse(JSON.stringify(basicMessageSelect));
    }
    // If selecting just parent message information, modify messageSelect 
    // so that it's selecting on the parent message
    else if (includeMessageParentInfo) {
        messageSelect = { parent: { select: JSON.parse(JSON.stringify(basicMessageSelect)) } };
    }
    // Return the message select query with the where clause
    return {
        where: { id: { in: messageIds } },
        select: messageSelect,
    };
};

/**
 * Populates a map of message IDs to PreMapMessageData objects.
 * @param messageMap Map of message data, by ID
 * @param messages Array of messages fetched from the database
 * @param userData User session data containing language preferences
 * @returns Map of message IDs to PreMapMessageData objects
 */
export const populateMessageDataMap = (
    messageMap: Record<string, PreMapMessageData>,
    messages: QueriedMessage[],
    userData: SessionUserToken,
): void => {
    const populateMessage = (message: QueriedMessage) => {
        // If we've already populated this message, skip it
        if (messageMap[message.id]) return;

        const bestTrans = getTranslation<{ language: string, text: string }>(message, userData.languages);
        if (Object.keys(bestTrans).length === 0) {
            logger.warning("No translation found for message", { trace: "0441", message: message?.id, user: userData.id });
            return;
        }

        const messageData: PreMapMessageData = {
            chatId: message.chat?.id ?? null,
            content: bestTrans.text ?? "",
            id: message.id,
            isNew: false, // Any message fetched from the database is not new by definition
            language: bestTrans.language ?? DEFAULT_LANGUAGE,
            parentId: message.parent?.id ?? undefined,
            translations: message.translations.map((t: { language: string; text: string }) => ({
                language: t.language,
                text: t.text,
            })),
            userId: message.user?.id ?? null,
        };

        messageMap[message.id] = messageData;

        // If the parent object has more information, populate that as well
        if (message.parent !== null && typeof message.parent === "object" && Object.keys(message.parent).length > 1) {
            populateMessage(message.parent);
        }
    };
    messages.forEach(message => {
        populateMessage(message);
    });
};

/**
 * Collects chat and participant information for existing chats, and adds 
 * it to the preMapUserData and preMapChatData objects.
 */
export const getChatParticipantData = async ({
    includeMessageParentInfo,
    includeMessageInfo,
    preMapChatData,
    preMapMessageData,
    preMapUserData,
    userData,
    ...rest
}: CollectParticipantDataParams): Promise<void> => {
    const chatIds = rest.chatIds ?? [];
    const messageIds = rest.messageIds ?? [];
    // If no chat or message IDs are provided, log warning and return
    if (chatIds.length === 0 && messageIds.length === 0) {
        logger.error("No chat IDs or message IDs provided", { trace: "0355", user: userData.id });
        return;
    }
    // Build select query
    const chatSelect = messageIds.length > 0 ? {
        OR: [
            { id: { in: chatIds } },
            { messages: { some: { id: { in: messageIds } } } },
        ],
    } : { id: { in: chatIds } };
    // Build message select query
    const notSelectingLastMessage = includeMessageInfo === true || includeMessageParentInfo === true;
    const messageQuery = buildChatParticipantMessageQuery(messageIds, includeMessageInfo === true, includeMessageParentInfo === true);
    // Query chat information from database
    const chatInfo = await prismaInstance.chat.findMany({
        where: chatSelect,
        select: {
            id: true,
            participants: {
                select: {
                    user: {
                        select: {
                            id: true,
                            invitedByUser: {
                                select: {
                                    id: true,
                                },
                            },
                            isBot: true,
                            isPrivate: true,
                            botSettings: true,
                            name: true,
                        },
                    },
                },
            },
            messages: messageQuery,
            // Find number of participants
            _count: { select: { participants: true } },
        },
    });
    // Parse chat and bot information
    chatInfo.forEach(chat => {
        // Find bots you are allowed to talk to
        const allowedBots = chat.participants.filter(p => p.user.isBot && (!p.user.isPrivate || p.user.invitedByUser?.id === userData.id));
        // Set chat information
        preMapChatData[chat.id] = {
            isNew: false,
            botParticipants: allowedBots.map(p => p.user.id),
            participantsCount: chat._count.participants,
            // Skip if custom message query is provided
            lastMessageId: (notSelectingLastMessage && chat.messages.length > 0) ? chat.messages[0].id : undefined,
        };
        // Set message information
        if (notSelectingLastMessage) {
            populateMessageDataMap(preMapMessageData, chat.messages, userData);
        }
        // Set information about all participants
        chat.participants.forEach(p => {
            preMapUserData[p.user.id] = {
                botSettings: p.user.botSettings ?? JSON.stringify({}),
                id: p.user.id,
                isBot: p.user.isBot,
                name: p.user.name,
            };
        });
    });
};
