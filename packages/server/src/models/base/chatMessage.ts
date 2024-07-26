import { ChatCreateInput, ChatInviteCreateInput, chatInviteValidation, ChatMessage, ChatMessageCreateInput, ChatMessageSearchTreeInput, ChatMessageSearchTreeResult, ChatMessageSortBy, ChatMessageUpdateInput, ChatUpdateInput, getTranslation, MaxObjects, uuidValidate } from "@local/shared";
import { Request } from "express";
import { ModelMap } from ".";
import { readManyHelper } from "../../actions/reads";
import { getUser } from "../../auth/request";
import { addSupplementalFields } from "../../builders/addSupplementalFields";
import { modelToGql } from "../../builders/modelToGql";
import { selectHelper } from "../../builders/selectHelper";
import { shapeHelper } from "../../builders/shapeHelper";
import { toPartialGqlInfo } from "../../builders/toPartialGqlInfo";
import { GraphQLInfo, PartialGraphQLInfo } from "../../builders/types";
import { prismaInstance } from "../../db/instance";
import { chatMessage_findMany } from "../../endpoints/generated/chatMessage_findMany";
import { CustomError } from "../../events/error";
import { logger } from "../../events/logger";
import { Trigger } from "../../events/trigger";
import { emitSocketEvent } from "../../sockets/events";
import { ChatContextManager, determineRespondingBots } from "../../tasks/llm/context";
import { requestBotResponse } from "../../tasks/llm/queue";
import { getAuthenticatedData, SortMap } from "../../utils";
import { ChatMessageBeforeDeletedData, getChatParticipantData, PreMapChatData, PreMapMessageData, PreMapUserData } from "../../utils/chat";
import { translationShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions, isOwnerAdminCheck, permissionsCheck } from "../../validators";
import { ChatMessageFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { ChatMessageModelLogic, ChatModelInfo, ChatModelLogic, ReactionModelLogic, UserModelLogic } from "./types";

type ChatMessagePre = {
    /** Map of chat IDs to information about the chat */
    chatData: Record<string, PreMapChatData>;
    /** Map of message IDs to information about the message */
    messageData: Record<string, PreMapMessageData>;
    /** Map of user IDs to information about the user */
    userData: Record<string, PreMapUserData>;
};

const __typename = "ChatMessage" as const;
export const ChatMessageModel: ChatMessageModelLogic = ({
    __typename,
    dbTable: "chat_message",
    dbTranslationTable: "chat_message_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, text: true } } }),
            get: (select, languages) => {
                const best = getTranslation(select, languages).text ?? "";
                return best.length > 30 ? best.slice(0, 30) + "..." : best;
            },
        },
    }),
    format: ChatMessageFormat,
    mutate: {
        shape: {
            /**
             * Collects bot information, message information, and chat information for AI responses, 
             * notifications, and web socket events.
             * 
             * NOTE: Updated messages don't trigger AI responses. Instead, you must create a new message 
             * with versionIndex set to the previous version's index + 1.
             */
            pre: async ({ Create, Update, Delete, userData, inputsById }): Promise<ChatMessagePre> => {
                // Initialize objects to store bot, message, and chat information
                const preMapChatData: Record<string, PreMapChatData> = {};
                const preMapMessageData: Record<string, PreMapMessageData> = {};
                const preMapUserData: Record<string, PreMapUserData> = {};
                // Find known create/update information. We'll query for the rest later.
                for (const { node, input } of [...Create, ...Update]) {
                    const translations = [...((input as ChatMessageCreateInput).translationsCreate ?? []), ...((input as ChatMessageUpdateInput).translationsUpdate ?? [])].filter(t => t.text) as { language: string, text: string }[];
                    // While messages can technically be created in multiple languages, we'll only worry about one
                    const best = getTranslation({ translations }, userData.languages);
                    // Collect chat information
                    let chatId = (input as ChatMessageCreateInput).chatConnect ?? null;
                    // If chat is being created or updated, we may not need to query for it later
                    if (node.parent && node.parent.__typename === "Chat" && ["Create", "Update"].includes(node.parent.action)) {
                        const chatUpsertInfo = inputsById[node.parent.id]?.input as ChatCreateInput | ChatUpdateInput | undefined;
                        if (chatUpsertInfo?.id) {
                            chatId = chatUpsertInfo.id;
                            // Store all invite information. Later we'll check if any of these are bots (which are automatically accepted, 
                            // and can potentially be used for AI responses)
                            preMapChatData[chatId] = {
                                potentialBotIds: chatUpsertInfo.invitesCreate?.map(i => typeof i === "string" ? (inputsById[i]?.input as ChatInviteCreateInput)?.userConnect : i.userConnect) ?? [],
                                participantsDelete: ((chatUpsertInfo as ChatUpdateInput).participantsDelete ?? []),
                                isNew: node.parent.action === "Create",
                            };
                        }
                    }
                    // Collect message data
                    preMapMessageData[input.id] = {
                        chatId,
                        content: best?.text ?? "",
                        id: input.id,
                        isNew: node.action === "Create",
                        language: best?.language ?? userData.languages[0],
                        translations,
                        userId: (input as ChatMessageCreateInput).userConnect ?? userData.id,
                    };
                }
                // Find known delete information
                for (const { node, input: id } of Delete) {
                    // Collect chat information
                    let chatId: string | null = null;
                    if (node.parent && node.parent.__typename === "Chat" && node.parent.action === "Update") {
                        const chatUpdateInfo = inputsById[node.parent.id]?.input as ChatUpdateInput | undefined;
                        if (chatUpdateInfo?.id) {
                            chatId = chatUpdateInfo.id;
                            preMapChatData[chatId] = {
                                isNew: false,
                            };
                        }
                    }
                    // Collect message data
                    preMapMessageData[id] = {
                        chatId,
                        content: "",
                        id,
                        isNew: false,
                        language: "",
                        userId: "",
                        translations: [], // Not needed for deletes
                    };
                }
                // Find chat information of every chat in chatData and every chat missing from messageData
                const messageIdsForMissingChats = Object.entries(preMapMessageData).filter(([, data]) => !data.chatId).map(([id]) => id);
                const knownExistingChatIds = Object.entries(preMapChatData).filter(([, data]) => !data.isNew).map(([id]) => id);
                await getChatParticipantData({
                    chatIds: knownExistingChatIds,
                    messageIds: messageIdsForMissingChats,
                    preMapChatData,
                    preMapMessageData,
                    preMapUserData,
                    userData,
                });
                const lastParentId: Record<string, string> = {};
                // Loop through all new messages to find the parent ID of each one
                for (const [messageId, message] of Object.entries(preMapMessageData)) {
                    if (message.parentId || !message.isNew || !message.chatId) continue;
                    const chat = preMapChatData[message.chatId];
                    // If the parent ID was already updated from the loop (i.e. a new last message was created), use that
                    if (lastParentId[message.chatId]) {
                        message.parentId = lastParentId[message.chatId];
                        lastParentId[message.chatId] = messageId;
                    }
                    // Otherwise, use the last message currently in the database
                    else {
                        message.parentId = chat.lastMessageId;
                        lastParentId[message.chatId] = messageId;
                    }
                }
                // Parse potential bots and bots being removed
                const potentialBotIds: string[] = [];
                const participantsBeingRemovedIds: string[] = [];
                Object.entries(preMapChatData).forEach(([id, chat]) => {
                    if (!chat.isNew) return;
                    if (chat.potentialBotIds) {
                        chat.potentialBotIds.forEach(botId => {
                            // Only add to potentialBotIds if the user ID is not already a key in botData (and also not your ID)
                            if (!preMapUserData[botId] && botId !== userData.id) potentialBotIds.push(botId);
                            // If it is already a key in bot data, update chatData.botParticipants
                            else if (preMapUserData[botId]) {
                                if (!chat.botParticipants) chat.botParticipants = [];
                                chat.botParticipants.push(botId);
                            }
                        });
                    }
                    // Add participants being removed to participantsBeingRemovedIds. 
                    // Since this is the ID of the participant object and not the actual user, we'll have to query for the user later.
                    if (chat.participantsDelete) participantsBeingRemovedIds.push(...chat.participantsDelete);
                });
                // Query potential bot IDs. Any found to be bots will automatically be accepted, 
                // and can potentially be used for AI responses.
                if (potentialBotIds.length) {
                    const potentialBots = await prismaInstance.user.findMany({
                        where: {
                            id: { in: potentialBotIds },
                            isBot: true,
                        },
                        select: {
                            id: true,
                            invitedByUser: {
                                select: {
                                    id: true,
                                },
                            },
                            isPrivate: true,
                            botSettings: true,
                            name: true,
                        },
                    });
                    potentialBots.forEach(user => {
                        // Any participant (even not bots) can be added to preMapUserData
                        preMapUserData[user.id] = {
                            botSettings: user.botSettings ?? JSON.stringify({}),
                            id: user.id,
                            isBot: true,
                            name: user.name,
                        };
                        // Add any bot that is public or invited by you to participants
                        if (!user.isPrivate || user.invitedByUser?.id === userData.id) {
                            Object.entries(preMapChatData).forEach(([id, chat]) => {
                                if (chat.potentialBotIds?.includes(user.id) && !chat.botParticipants?.includes(user.id)) {
                                    if (!chat.botParticipants) chat.botParticipants = [];
                                    chat.botParticipants.push(user.id);
                                }
                            });
                        }
                    });
                }
                // Query participants being deleted and remove them from botData and chatData.botParticipants
                if (participantsBeingRemovedIds.length) {
                    const participantsBeingRemoved = await prismaInstance.chat_participants.findMany({
                        where: {
                            id: { in: participantsBeingRemovedIds },
                        },
                        select: {
                            id: true,
                            user: {
                                select: {
                                    id: true,
                                },
                            },
                        },
                    });
                    participantsBeingRemoved.forEach(participant => {
                        // Remove from chatData.botParticipants
                        Object.values(preMapChatData).forEach((chat) => {
                            if (chat.botParticipants?.includes(participant.user.id)) {
                                chat.botParticipants = chat.botParticipants.filter(id => id !== participant.user.id);
                            }
                        });
                        // NOTE: Don't remove from preMapUserData, since their messages may still be in the context for AI responses
                    });
                }
                // Messages can be created for you or bots. Make sure all new messages meet this criteria.
                Object.values(preMapMessageData).forEach((message) => {
                    // If the message is new, but the user ID is not yours or a bot, throw an error
                    if (message.isNew && message.userId !== userData.id && !Object.keys(preMapUserData).includes(message.userId ?? "")) {
                        throw new CustomError("0526", "Unauthorized", ["en"], { message });
                    }
                });
                // Return data
                return { userData: preMapUserData, chatData: preMapChatData, messageData: preMapMessageData };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as ChatMessagePre;
                const parentId = preData.messageData[data.id]?.parentId;
                return {
                    id: data.id,
                    user: { connect: { id: data.userConnect ?? rest.userData.id } }, // Can create messages for bots. This is authenticated in the "pre" function.
                    ...(parentId ? { parent: { connect: { id: parentId } } } : {}),
                    versionIndex: data.versionIndex,
                    chat: await shapeHelper({ relation: "chat", relTypes: ["Connect"], isOneToOne: true, objectType: "Chat", parentRelationshipName: "messages", data, ...rest }),
                    translations: await translationShapeHelper({ relTypes: ["Create"], data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => ({
                translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], data, ...rest }),
            }),
        },
        trigger: {
            beforeDeleted: async ({ beforeDeletedData, deletingIds }) => {
                // Find the chat user, parent, and children for each message being deleted
                const deleting = await prismaInstance.chat_message.findMany({
                    where: { id: { in: deletingIds } },
                    select: {
                        id: true,
                        chat: { select: { id: true } },
                        user: { select: { id: true } },
                        parent: { select: { id: true } },
                        children: { select: { id: true } },
                    },
                });
                // Add data to beforeDeletedData
                const messageData: Record<string, ChatMessageBeforeDeletedData> = {};
                for (const m of deleting) {
                    messageData[m.id] = {
                        id: m.id,
                        chatId: m.chat?.id,
                        userId: m.user?.id,
                        parentId: m.parent?.id,
                        childIds: m.children.map(c => c.id),
                    };
                }
                if (beforeDeletedData[__typename]) beforeDeletedData[__typename] = { ...beforeDeletedData[__typename], ...messageData };
                else beforeDeletedData[__typename] = messageData;
            },
            afterMutations: async ({ beforeDeletedData, createdIds, deletedIds, updatedIds, preMap, userData }) => {
                const preMapUserData: Record<string, PreMapUserData> = preMap[__typename]?.userData ?? {};
                const preMapChatData: Record<string, PreMapChatData> = preMap[__typename]?.chatData ?? {};
                const preMapMessageData: Record<string, PreMapMessageData> = preMap[__typename]?.messageData ?? {};
                let messages: ChatMessage[] = [];
                if (createdIds.length > 0 || updatedIds.length > 0) {
                    const paginatedMessages = await readManyHelper({
                        info: chatMessage_findMany,
                        input: { ids: [...createdIds, ...updatedIds], take: createdIds.length + updatedIds.length },
                        objectType: __typename,
                        req: { session: { languages: userData.languages, users: [userData] } },
                    });
                    messages = paginatedMessages.edges.map(e => e.node);
                }
                // Call triggers
                for (const objectId of createdIds) {
                    const messageData: PreMapMessageData = preMapMessageData[objectId];
                    const chatId = messageData.chatId;
                    const chatMessage = messages.find(m => m.id === objectId) as ChatMessage;
                    const senderId = messageData.userId;
                    if (!chatMessage || !chatId || !senderId) {
                        logger.error("Message, message sender, or chat not found", { trace: "0363", user: userData.id, messageId: objectId, chatId });
                        continue;
                    }
                    // Add message to cache
                    await (new ChatContextManager()).addMessage(chatId, messageData);
                    // Common trigger logic
                    await Trigger(userData.languages).objectCreated({
                        createdById: userData.id,
                        hasCompleteAndPublic: true, // N/A
                        hasParent: true, // N/A
                        owner: { id: userData.id, __typename: "User" },
                        objectId,
                        objectType: __typename,
                    });
                    await Trigger(userData.languages).chatMessageCreated({
                        excludeUserId: userData.id,
                        chatId,
                        messageId: objectId,
                        senderId,
                        message: chatMessage,
                    });
                    // Determine which bots should respond, if any.
                    const chat: PreMapChatData | undefined = preMapChatData[chatId];
                    const bots: PreMapUserData[] = chat?.botParticipants?.map(id => preMapUserData[id]).filter(b => b) ?? [];
                    const botsToRespond = determineRespondingBots(messageData, chat, bots, userData.id);
                    if (botsToRespond.length) {
                        // Send typing indicator while bots are responding
                        emitSocketEvent("typing", chatId, { starting: botsToRespond });
                        // For each bot that should respond, request bot response
                        for (const botId of botsToRespond) {
                            // Call LLM for bot response
                            requestBotResponse({
                                chatId,
                                parent: messageData,
                                respondingBotId: botId,
                                task: "Start", //TODO need task queue for chats, so we can enter an exit tasks
                                participantsData: preMapUserData,
                                userData,
                            });
                        }
                    }
                }
                for (const objectId of updatedIds) {
                    await Trigger(userData.languages).objectUpdated({
                        updatedById: userData.id,
                        hasCompleteAndPublic: true, // N/A
                        hasParent: true, // N/A
                        owner: { id: userData.id, __typename: "User" },
                        objectId,
                        objectType: __typename,
                        wasCompleteAndPublic: true, // N/A
                    });
                    await Trigger(userData.languages).chatMessageUpdated({
                        data: preMapMessageData[objectId],
                        message: messages.find(m => m.id === objectId) as ChatMessage,
                    });
                }
                for (const objectId of deletedIds) {
                    await Trigger(userData.languages).objectDeleted({
                        deletedById: userData.id,
                        hasBeenTransferred: false, // N/A
                        hasParent: true, // N/A
                        objectId,
                        objectType: __typename,
                        wasCompleteAndPublic: true, // N/A
                    });
                    const messageData = beforeDeletedData[__typename]?.[objectId] as ChatMessageBeforeDeletedData | undefined;
                    if (messageData) {
                        // Update the children of each message being deleted
                        // to point to the parent of the message being deleted. You can think of this as a 
                        // linked list, where each message has a pointer to the next message.
                        if (messageData.parentId) {
                            await prismaInstance.chat_message.updateMany({
                                where: { id: { in: messageData.childIds ?? [] } },
                                data: { parentId: messageData.parentId },
                            });
                        }
                        await Trigger(userData.languages).chatMessageDeleted({
                            data: messageData,
                            messageId: objectId,
                        });
                    } else {
                        logger.error("Message data not found", { trace: "0067", user: userData.id, message: objectId });
                    }
                }
            },
        },
        yup: chatInviteValidation,
    },
    query: {
        /**
         * Custom search query for chat messages. Starts either at the most recent 
         * message or specified, and traverses up and down the chat tree to find
         * surrounding messages.
         */
        async searchTree(
            req: Request,
            input: ChatMessageSearchTreeInput,
            info: GraphQLInfo | PartialGraphQLInfo,
        ): Promise<ChatMessageSearchTreeResult> {
            if (!input.chatId) throw new CustomError("0531", "InvalidArgs", getUser(req.session)?.languages ?? ["en"], { input });
            // Query for all authentication data
            const userData = getUser(req.session);
            const authDataById = await getAuthenticatedData({ "Chat": [input.chatId] }, userData ?? null);
            if (Object.keys(authDataById).length === 0) {
                throw new CustomError("0016", "NotFound", userData?.languages ?? req.session.languages, { input, userId: userData?.id });
            }
            await permissionsCheck(authDataById, { ["Read"]: [input.chatId] }, {}, userData);
            // Partially convert info type
            const partial = toPartialGqlInfo(info, {
                __typename: "ChatMessageSearchTreeResult",
                messages: "ChatMessage",
            }, req.session.languages, true);
            // Determine sort order. This is only used if startId is not provided, since the sort is used 
            // to determine the starting point of the search.
            const orderByField = input.sortBy ?? ModelMap.get<ChatMessageModelLogic>("ChatMessage").search.defaultSort;
            const orderBy = !input.startId && orderByField in SortMap ? SortMap[orderByField] : undefined;
            // First, find the total number of messages in the chat
            const totalInThread = await prismaInstance.chat_message.count({
                where: { chatId: input.chatId },
            });
            // If it's less than or equal to the take amount, we can just return all messages. 
            const take = input.take ?? 50;
            if (totalInThread <= take) {
                let messages: any[] = await prismaInstance.chat_message.findMany({
                    where: { chatId: input.chatId },
                    orderBy,
                    take,
                    ...selectHelper(partial.messages as PartialGraphQLInfo),
                });
                messages = messages.map((c: any) => modelToGql(c, partial.messages as PartialGraphQLInfo));
                messages = await addSupplementalFields(getUser(req.session), messages, partial.messages as PartialGraphQLInfo);
                return {
                    __typename: "ChatMessageSearchTreeResult" as const,
                    hasMoreDown: false,
                    hasMoreUp: false,
                    messages,
                };
            }
            // Otherwise, we need to traverse up and/or down the chat tree to find the messages
            //TODO
            return {} as any;
        },
    },
    search: {
        defaultSort: ChatMessageSortBy.DateCreatedDesc,
        searchFields: {
            createdTimeFrame: true,
            minScore: true,
            chatId: true,
            userId: true,
            updatedTimeFrame: true,
            translationLanguages: true,
        },
        sortBy: ChatMessageSortBy,
        searchStringQuery: () => ({
            OR: [
                "transTextWrapped",
                { user: ModelMap.get<UserModelLogic>("User").search.searchStringQuery() },
            ],
        }),
        supplemental: {
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, userData)),
                        reaction: await ModelMap.get<ReactionModelLogic>("Reaction").query.getReactions(userData?.id, ids, __typename),
                    },
                };
            },
        },
    },
    validate: () => ({
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data?.user,
        }),
        permissionResolvers: ({ data, isAdmin: isMessageOwner, isDeleted, isLoggedIn, userId }) => {
            const isChatAdmin = userId ? isOwnerAdminCheck(ModelMap.get<ChatModelLogic>("Chat").validate().owner(data.chat as ChatModelInfo["PrismaModel"], userId), userId) : false;
            const isParticipant = uuidValidate(userId) && (data.chat as ChatModelInfo["PrismaModel"]).participants?.some((p) => p.userId === userId);
            return {
                canConnect: () => isLoggedIn && !isDeleted && isParticipant,
                canDelete: () => isLoggedIn && !isDeleted && (isMessageOwner || isChatAdmin),
                canDisconnect: () => isLoggedIn,
                canUpdate: () => isLoggedIn && !isDeleted && isMessageOwner,
                canRead: () => !isDeleted && isParticipant,
                canReply: () => isLoggedIn && !isDeleted && isParticipant,
                canReport: () => isLoggedIn && !isDeleted && isParticipant,
                canReact: () => isLoggedIn && !isDeleted && isParticipant,
            };
        },
        permissionsSelect: () => ({
            id: true,
            chat: ["Chat", ["messages"]],
            user: "User",
        }),
        visibility: {
            private: function getVisibilityPrivate() {
                return {};
            },
            public: function getVisibilityPublic() {
                return {};
            },
            owner: (userId) => ({
                chat: ModelMap.get<ChatModelLogic>("Chat").validate().visibility.owner(userId),
            }),
        },
    }),
});
