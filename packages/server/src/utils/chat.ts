import { ChatUpdateInput, getTranslation, SessionUser } from "@local/shared";
import { DbProvider } from "../db/provider.js";
import { CustomError } from "../events/error.js";
import { logger } from "../events/logger.js";
import { PreShapeEmbeddableTranslatableResult } from "./shapes/preShapeEmbeddableTranslatable.js";

export type PreMapMessageDataCreate = {
    __type: "Create";
    chatId: string;
    messageId: string;
    parentId: string | null;
    translations: readonly { id: string, language: string, text: string }[];
    userId: string;
}
export type PreMapMessageDataUpdate = {
    __type: "Update";
    chatId: string;
    messageId: string;
    parentId?: string | null;
    /** Existing translations after the update */
    translations?: readonly { id: string, language: string, text: string }[];
    userId?: string;
}
export type PreMapMessageDataDelete = {
    __type: "Delete";
    chatId: string;
    messageId: string;
}

/** Information for a message, collected in mutate.shape.pre */
export type PreMapMessageData = PreMapMessageDataCreate | PreMapMessageDataUpdate | PreMapMessageDataDelete;

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

export type ChatPreBranchInfo = {
    /** The message most recently created (i.e. has the highest sequence number), or null if no messages */
    lastSequenceId: string | null,
    /** 
     * Map of message IDs to their parent ID and list of child IDs. 
     * This data should be provided for each message that's being deleted, so that we 
     * can heal the branch structure around the deleted messages.
     */
    messageTreeInfo: Record<string, { parentId: string | null; childIds: string[] }>;
};
export type ChatPre = PreShapeEmbeddableTranslatableResult & {
    /** Invited bots which you have the permission to invite */
    bots: { id: string }[];
    /** Map of chat IDs to branchable status and the last message */
    branchInfo: Record<string, ChatPreBranchInfo>;
};

type ChatMessageCreate = {
    id: string;
    parent?: {
        connect: {
            id: string;
        };
    };
    // Include other fields necessary for creating a message
    [key: string]: any;
}
type ChatMessageUpdate = Omit<ChatMessageCreate, "parent">;
type ChatMessageDelete = {
    id: string;
}
export type ChatMessageOperations = {
    create?: ChatMessageCreate[] | null;
    update?: ChatMessageUpdate[] | null;
    delete?: ChatMessageDelete[] | null;
}
export type ChatMessageCreateResult = Omit<ChatMessageCreate, "parent"> & {
    parent?: {
        create: ChatMessageCreateResult
    } | {
        connect: {
            id: string;
        }
    };
};

export type ChatMessageOperationsResult = {
    // The `create` field is a single object with all messages nested
    create?: ChatMessageCreateResult;
    // Each update can now update its parent reference or remove it
    update?: (ChatMessageUpdate & { parent?: ({ connect: { id: string } } | { disconnect: true }) })[];
    // Deletes are unchanged
    delete?: ChatMessageDelete[];
};

export type ChatMessageOperationsSummaryResult = {
    Create: { id: string, parentId: string | null }[];
    Update: { id: string, parentId: string | null }[];
}

export type PrepareChatMessageOperationsResult = {
    /** The operations to provide with a Chat create/update Prisma query */
    operations: ChatMessageOperationsResult | undefined;
    /** 
     * A summary of parent ID changes, which can be more easily consumed if not performing 
     * a direct prisma chat mutation (e.g. updating a chat message directly)
     */
    summary: ChatMessageOperationsSummaryResult;
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

export type ChatMessagePre = {
    /** Map of chat IDs to information about the chat */
    chatData: Record<string, PreMapChatData>;
    /** Map of message IDs to information about the message */
    messageData: Record<string, PreMapMessageData>;
    /** Map of user IDs to information about the user */
    userData: Record<string, PreMapUserData>;
};

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
    /** Maps for collecting chat, message, and user data */
    preMap: ChatMessagePre,
    /** The current user's data */
    userData: SessionUser,
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
export function buildChatParticipantMessageQuery(
    messageIds: string[],
    includeMessageInfo: boolean,
    includeMessageParentInfo: boolean,
): Record<string, any> {
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
}

/**
 * Populates a map of message IDs to PreMapMessageData objects.
 * @param messageMap Map of message data, by ID
 * @param messages Array of messages fetched from the database
 * @param userData User session data containing language preferences
 * @returns Map of message IDs to PreMapMessageData objects
 */
export function populateMessageDataMap(
    messageMap: Record<string, PreMapMessageData>,
    messages: QueriedMessage[],
    userData: SessionUser,
): void {
    function populateMessage(message: QueriedMessage) {
        // If we've already populated this message, skip it
        if (messageMap[message.id]) return;

        const bestTrans = getTranslation<{ language: string, text: string }>(message, userData.languages);
        if (Object.keys(bestTrans).length === 0) {
            logger.warning("No translation found for message", { trace: "0441", message: message?.id, user: userData.id });
            return;
        }

        const messageData: PreMapMessageDataUpdate = {
            __type: "Update",
            chatId: message.chat?.id ?? null,
            messageId: message.id,
            parentId: message.parent?.id ?? undefined,
            translations: message.translations.map((t: { id: string, language: string; text: string }) => ({
                id: t.id,
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
    }
    messages.forEach(message => {
        populateMessage(message);
    });
}

/**
 * Collects chat and participant information for existing chats, and adds 
 * it to the preMapUserData and preMapChatData objects.
 */
export async function getChatParticipantData({
    includeMessageParentInfo,
    includeMessageInfo,
    preMap,
    userData,
    ...rest
}: CollectParticipantDataParams): Promise<void> {
    const chatIds = rest.chatIds ?? [];
    const messageIds = rest.messageIds ?? [];

    // Build where query
    const wheres: object[] = [];
    if (chatIds.length > 0) {
        wheres.push({ id: { in: chatIds } });
    }
    if (messageIds.length > 0) {
        wheres.push({ messages: { some: { id: { in: messageIds } } } });
    }
    // If no chat or message IDs are provided, log warning and return
    if (wheres.length === 0) {
        logger.error("No chat or message IDs provided", { trace: "0355", user: userData.id });
        return;
    }
    const chatSelect = wheres.length > 1 ? { OR: wheres } : wheres[0];

    // Build message select query
    const notSelectingLastMessage = includeMessageInfo === true || includeMessageParentInfo === true;
    const messageQuery = buildChatParticipantMessageQuery(messageIds, includeMessageInfo == true, includeMessageParentInfo === true);
    // Query chat information from database
    const chatInfo = await DbProvider.get().chat.findMany({
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
    console.log("got chat info", chatInfo.length);
    // Parse chat and bot information
    chatInfo.forEach(chat => {
        // Find bots you are allowed to talk to
        const allowedBots = chat.participants.filter(p => p.user.isBot && (!p.user.isPrivate || p.user.invitedByUser?.id === userData.id));
        // Set chat information
        preMap.chatData[chat.id] = {
            isNew: false,
            botParticipants: allowedBots.map(p => p.user.id),
            participantsCount: chat._count.participants,
            // Skip if custom message query is provided
            lastMessageId: (!notSelectingLastMessage && chat.messages.length > 0) ? chat.messages[0].id : undefined,
        };
        // Set message information
        if (notSelectingLastMessage) {
            populateMessageDataMap(preMap.messageData, chat.messages, userData);
        }
        // Set information about all participants
        chat.participants.forEach(p => {
            preMap.userData[p.user.id] = {
                botSettings: p.user.botSettings ?? JSON.stringify({}),
                id: p.user.id,
                isBot: p.user.isBot,
                name: p.user.name,
            };
        });
    });
}


/**
 * Generates a safe message structure for a Prisma chat mutation. 
 * This function restructures the 'create' field to ensure that messages are properly linked,
 * preserving the order and hierarchy dictated by the chat's branching logic, while avoiding 
 * references to uncreated messages. It leaves the 'delete' field unchanged and optionally 
 * adds or modifes the 'update' field to adjust the chat structure around deleted messages.
 * 
 * Additionally, it returns a simplified summary of the parent connections for easier consumption.
 * 
 * @param operations The proposed operations for Prisma, typically including 'create' and 'delete'.
 *                   This function modifies 'create' to nest messages according to their parent-child
 *                   relationships, adjusts 'update' to handle relational changes due to deletions,
 *                   and maintains 'delete' as provided.
 * @param branchInfo Information on the chat's branching capability and the identifier of the last
 *                   message in the sequence, used to anchor new messages if necessary.
 * @returns An object containing:
 *          - `prismaOperations`: Structured for direct use in a Prisma mutation, with nested 'create',
 *            unmodified 'delete', and newly added 'update' operations to maintain chat integrity.
 *          - `summary`: A simplified structure with `Create` and `Update`, and `Delete` arrays containing
 *           message IDs and their new parent IDs
 */
export function prepareChatMessageOperations(
    operations: ChatMessageOperations | undefined,
    branchInfo: ChatPreBranchInfo,
): PrepareChatMessageOperationsResult {
    const createOperations = operations?.create || [];
    const updateOperations = operations?.update || [];
    const deleteOperations = operations?.delete || [];
    const branchData = branchInfo || { lastSequenceId: null, messageTreeInfo: {} };
    const summary: ChatMessageOperationsSummaryResult = { Create: [], Update: [] };

    // Map of message IDs to their create data
    const createMessageMap = new Map<string, ChatMessageCreate>();
    createOperations.forEach((msg) => {
        createMessageMap.set(msg.id, msg);
    });

    // Detect branching in the new messages (i.e. a message with multiple children), 
    // which could cause issues with Prisma.
    // 
    // In practice there should never be branching, since if multiple people are trying 
    // to update a chat at the same time, it's not a branchable chat. And if it's one person sending 
    // multiple messages, they wouldn't be on different branches.
    const messagesWithChild = new Set<string>();
    for (const msg of createOperations) {
        const parentId = msg.parent?.connect?.id;
        if (parentId && createMessageMap.has(parentId)) {
            if (messagesWithChild.has(parentId)) {
                throw new Error("Cannot create nested messages in a branching chat. All messages must be sequential.");
            }
            messagesWithChild.add(parentId);
        }
    }

    /**
     * Function to recursively build nested creates from a list of messages 
     * ordered from root to leaf.
     * 
     * Result should be a nested structure where the leaf is at the top, and 
     * the root is the most deeply nested object.
     */
    function buildNestedCreate(orderedMessages: ChatMessageCreate[]): ChatMessageCreateResult {
        let nestedCreate: ChatMessageCreateResult | undefined;
        // Loop backwards through the ordered messages, building the nested structure
        for (let i = orderedMessages.length - 1; i >= 0; i--) {
            const msg = orderedMessages[i];

            // Determine the parent ID
            // Use orderedMessages[i + 1] as the parent if it exists
            let parentId = i > orderedMessages.length - 2 ? null : orderedMessages[i + 1].id;
            if (parentId) {
                nestedCreate = { ...msg, parent: { create: nestedCreate! } };
            }
            // Use the root's connect if it's not to a message being created
            if (!parentId && msg.parent?.connect?.id && !createMessageMap.has(msg.parent.connect.id)) {
                parentId = msg.parent.connect.id;
                nestedCreate = { ...msg, parent: { connect: { id: parentId } } };
            }
            // Use the lastSequenceId if it exists
            if (!parentId && branchData.lastSequenceId) {
                parentId = branchData.lastSequenceId;
                nestedCreate = { ...msg, parent: { connect: { id: parentId } } };
            }
            if (!parentId) {
                nestedCreate = { ...msg };
            }

            // Add the message to the summary
            summary.Create.push({ id: msg.id, parentId });
        }
        return nestedCreate!;
    }

    // Build the nested create structure for the root message
    let nestedCreate: ChatMessageCreateResult | undefined;
    if (createMessageMap.size > 0) {
        // Find messages with specified parents
        const parentIds: Set<string> = new Set();
        const messagesWithParents = createOperations.filter((msg) => {
            const parentId = msg.parent?.connect?.id;
            if (parentId) {
                parentIds.add(parentId);
                return true;
            }
            return false;
        });
        // Sort messages from leaf to root
        messagesWithParents.sort((a, b) => {
            const aParentId = a.parent?.connect?.id;
            const bParentId = b.parent?.connect?.id;
            if (aParentId === b.id) {
                return -1;
            }
            if (bParentId === a.id) {
                return 1;
            }
            return 0;
        });
        // Make sure that every connected parent is a new message, except at the root
        const isEveryParentNew = messagesWithParents.every((msg, i) => {
            // Ignore the root, which is the last message in the sequence
            if (i === messagesWithParents.length - 1) {
                return true;
            }
            const parentId = msg.parent?.connect?.id;
            return parentId && createMessageMap.has(parentId);
        });
        if (!isEveryParentNew) {
            throw new CustomError("0416", "InternalError", { msg: "Cannot create nested messages in a branching chat. All messages must be sequential." });
        }

        // Find messages without specified parents
        const messagesWithoutParents = createOperations.filter((msg) => {
            const parentId = msg.parent?.connect?.id;
            return !parentId;
        });
        // Sort messages without parents so that any in `parentIds` are at the end
        messagesWithoutParents.sort((a, b) => {
            const isAInParentIds = parentIds.has(a.id);
            const isBInParentIds = parentIds.has(b.id);
            if (isAInParentIds && !isBInParentIds) {
                return 1;
            }
            if (!isAInParentIds && isBInParentIds) {
                return -1;
            }
            return 0;
        }).reverse();

        // Combine the two sets of messages
        const allMessages = [...messagesWithParents, ...messagesWithoutParents];

        // Now every message (except maybe the first) should be connected to a parent. 
        // To allow Prisma to create them all in one go, we need to convert this to a nested structure.
        nestedCreate = buildNestedCreate(allMessages);
    }

    // Now handle delete operations and adjust update operations accordingly

    // Create a Set of deleted message IDs for quick lookup
    const deletedMessageIds = new Set<string>(deleteOperations.map((delOp) => delOp.id));

    // Collect additional update operations needed to reparent messages
    const additionalUpdateOperations: ChatMessageUpdate[] = [];

    // Function to find the nearest non-deleted ancestor
    function findNearestNonDeletedAncestor(startingParentId: string | null): string | null {
        let currentParentId = startingParentId;
        while (currentParentId) {
            if (!deletedMessageIds.has(currentParentId)) {
                return currentParentId;
            }
            const parentTreeInfo = branchData.messageTreeInfo[currentParentId];
            if (!parentTreeInfo) {
                // We don't have further parent info, assume null
                return null;
            }
            currentParentId = parentTreeInfo.parentId;
        }
        return null;
    }

    // Process each deleted message
    deleteOperations.forEach((deleteOp) => {
        const messageId = deleteOp.id;
        const treeInfo = branchData.messageTreeInfo[messageId];

        if (!treeInfo) {
            throw new Error(`Missing message tree info for deleted message ID: ${messageId}`);
        }

        const childIds = treeInfo.childIds || [];

        childIds.forEach((childId) => {
            if (deletedMessageIds.has(childId)) {
                // The child is also being deleted, so no need to update it
                return;
            }

            // Find the new parent ID for the child
            const newParentId = findNearestNonDeletedAncestor(treeInfo.parentId);

            // Find existing update operation that matches the child ID
            const existingUpdateOpIndex = updateOperations.findIndex((op) => op.id === childId);
            // Create or update the update operation
            const updateOp: ChatMessageUpdate = {
                ...(existingUpdateOpIndex >= 0 ? updateOperations[existingUpdateOpIndex] : {}),
                id: childId,
                parent: newParentId ? { connect: { id: newParentId } } : { disconnect: true },
            };
            // Add or replace the update operation
            if (existingUpdateOpIndex >= 0) {
                updateOperations[existingUpdateOpIndex] = updateOp;
            } else {
                additionalUpdateOperations.push(updateOp);
            }
            // Add to the summary
            summary.Update.push({ id: childId, parentId: newParentId });
        });
    });

    // Merge additionalUpdateOperations into updateOperations
    const allUpdateOperations = updateOperations.concat(additionalUpdateOperations);

    const messages = {
        create: nestedCreate,
        update: allUpdateOperations.length > 0 ? allUpdateOperations : undefined,
        delete: deleteOperations.length > 0 ? deleteOperations : undefined,
    };
    // Removed undefined and empty arrays
    const filteredMessages = Object.fromEntries(Object.entries(messages).filter(([_, v]) => v !== undefined && (Array.isArray(v) ? v.length > 0 : true)));
    // If the result is an empty object, return undefined
    const operationsResult = Object.keys(filteredMessages).length ? filteredMessages : undefined;

    return {
        operations: operationsResult,
        summary,
    };
}

/**
 * Collects preMap data for existing chats
 */
export async function populatePreMapForChatUpdates({
    updateInputs,
}: {
    updateInputs: ChatUpdateInput[],
}): Promise<{
    branchInfo: ChatPre["branchInfo"];
}> {
    // Find information needed to link new chat messages to their parent messages
    let branchInfo: ChatPre["branchInfo"] = {};
    const chatIds = updateInputs.map(u => u.id);
    if (chatIds.length === 0) {
        return { branchInfo };
    }
    // Fetch chat information
    const chats = await DbProvider.get().chat.findMany({
        where: { id: { in: chatIds } },
        select: {
            id: true,
            messages: {
                orderBy: { sequence: "desc" },
                take: 1,
                select: { id: true },
            },
        },
    });
    // Calculate branchability
    branchInfo = chats.reduce((acc, chat) => {
        acc[chat.id] = {
            lastSequenceId: chat.messages.length ? chat.messages[0].id : null,
            messageTreeInfo: {},
        };
        return acc;
    }, {} as ChatPre["branchInfo"]);

    // Collect IDs of last messages to always fetch their tree info
    const lastMessageIds = Object.values(branchInfo)
        .map(info => info.lastSequenceId)
        .filter((id): id is string => id !== null);

    // Collect all message IDs that are being deleted
    const deletedMessageIds = updateInputs.reduce((acc, input) => {
        const deleteIds = input.messagesDelete ?? [];
        acc.push(...deleteIds);
        return acc;
    }, [] as string[]);
    // If there are any messages being deleted, fetch each message's parent ID and child IDs
    if (deletedMessageIds.length > 0) {
        const messages = await DbProvider.get().chat_message.findMany({
            where: { id: { in: deletedMessageIds } },
            select: {
                id: true,
                chat: { select: { id: true } },
                parent: { select: { id: true } },
                children: { select: { id: true } },
            },
        });
        // Populate messageTreeInfo
        messages.forEach((msg) => {
            if (!msg.chat) {
                return;
            }
            const chatId = msg.chat.id;
            if (!branchInfo[chatId]) {
                branchInfo[msg.id] = {
                    lastSequenceId: null,
                    messageTreeInfo: {},
                };
            }
            branchInfo[chatId].messageTreeInfo[msg.id] = {
                parentId: msg.parent?.id ?? null,
                childIds: msg.children.map(c => c.id),
            };
        });
    }
    // Always fetch tree info for the last message in each chat, if it exists and wasn't already fetched via deletion
    const idsToFetch = lastMessageIds.filter(id => !deletedMessageIds.includes(id));
    if (idsToFetch.length > 0) {
        const lastMessages = await DbProvider.get().chat_message.findMany({
            where: { id: { in: idsToFetch } },
            select: {
                id: true,
                chat: { select: { id: true } },
                parent: { select: { id: true } },
                children: { select: { id: true } }, // Need children to confirm it's a leaf/last node? Or just parent?
            },
        });
        // Populate messageTreeInfo for these last messages
        lastMessages.forEach((msg) => {
            if (!msg.chat) {
                return;
            }
            const chatId = msg.chat.id;
            // Ensure the chat entry exists (should always exist from the first query)
            if (!branchInfo[chatId]) {
                logger.error(`BranchInfo missing for chat ${chatId} when processing last message ${msg.id}`, { trace: "0888" });
                return; // Skip if chat info is unexpectedly missing
            }
            // Add or overwrite the entry for the last message ID
            branchInfo[chatId].messageTreeInfo[msg.id] = {
                parentId: msg.parent?.id ?? null,
                childIds: msg.children.map(c => c.id), // Populate children even if empty
            };
        });
    }
    return { branchInfo };
}
