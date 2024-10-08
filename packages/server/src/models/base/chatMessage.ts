import { ChatCreateInput, ChatInviteCreateInput, ChatMessage, ChatMessageCreateInput, ChatMessageSearchTreeInput, ChatMessageSearchTreeResult, ChatMessageSortBy, ChatMessageUpdateInput, chatMessageValidation, ChatUpdateInput, getTranslation, MaxObjects, uuidValidate } from "@local/shared";
import { Request } from "express";
import { InputNode } from "utils/inputNode";
import { ModelMap } from ".";
import { getUser } from "../../auth/request";
import { addSupplementalFields } from "../../builders/addSupplementalFields";
import { modelToGql } from "../../builders/modelToGql";
import { selectHelper } from "../../builders/selectHelper";
import { shapeHelper } from "../../builders/shapeHelper";
import { toPartialGqlInfo } from "../../builders/toPartialGqlInfo";
import { GraphQLInfo, PartialGraphQLInfo } from "../../builders/types";
import { useVisibility } from "../../builders/visibilityBuilder";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { logger } from "../../events/logger";
import { Trigger } from "../../events/trigger";
import { emitSocketEvent } from "../../sockets/events";
import { ChatContextManager, determineRespondingBots } from "../../tasks/llm/context";
import { requestBotResponse } from "../../tasks/llm/queue";
import { getAuthenticatedData, SortMap } from "../../utils";
import { ChatMessagePre, getChatParticipantData, populatePreMapForChatUpdates, PreMapChatData, PreMapMessageData, PreMapMessageDataCreate, PreMapMessageDataDelete, PreMapMessageDataUpdate, PreMapUserData, prepareChatMessageOperations } from "../../utils/chat";
import { translationShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions, isOwnerAdminCheck, permissionsCheck } from "../../validators";
import { ChatMessageFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { ChatMessageModelInfo, ChatMessageModelLogic, ChatModelInfo, ChatModelLogic, ReactionModelLogic, UserModelLogic } from "./types";

const DEFAULT_CHAT_TAKE = 50;

const __typename = "ChatMessage" as const;
export const ChatMessageModel: ChatMessageModelLogic = ({
    __typename,
    dbTable: "chat_message",
    dbTranslationTable: "chat_message_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, text: true } } }),
            get: (select, languages) => {
                return getTranslation(select, languages).text ?? "";
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
                const preMap: ChatMessagePre = { chatData: {}, messageData: {}, userData: {} };

                // Find stored chat and chat message information 
                await getChatParticipantData({
                    includeMessageInfo: true,
                    includeMessageParentInfo: true,
                    chatIds: [...Create].map(({ input }) => input.chatConnect).filter(Boolean) as string[],
                    messageIds: [...Update, ...Delete].map(({ node }) => node.id),
                    preMap,
                    userData,
                });
                console.log("preMap chatData here", JSON.stringify(preMap.chatData));

                // Collect information for new messages, which can't be queried for
                for (const { node, input } of Create) {
                    // Collect chat information
                    let chatId = input.chatConnect ?? null;
                    // Collect information for new chats
                    if (node.parent && node.parent.__typename === "Chat" && ["Create"].includes(node.parent.action)) {
                        const chatUpsertInfo = inputsById[node.parent.id]?.input as ChatCreateInput | undefined;
                        if (chatUpsertInfo?.id) {
                            chatId = chatUpsertInfo.id;
                            // Store all invite information. Later we'll check if any of these are bots (which are automatically accepted, 
                            // and can potentially be used for AI responses)
                            preMap.chatData[chatId] = {
                                potentialBotIds: chatUpsertInfo.invitesCreate?.map(i => typeof i === "string" ? (inputsById[i]?.input as ChatInviteCreateInput)?.userConnect : i.userConnect) ?? [],
                                participantsDelete: ((chatUpsertInfo as ChatUpdateInput).participantsDelete ?? []),
                                isNew: node.parent.action === "Create",
                            };
                        }
                    }
                    // Collect message data
                    const translations = (input.translationsCreate?.filter(t => t.text.length > 0) || []);
                    preMap.messageData[node.id] = {
                        __type: "Create",
                        chatId,
                        messageId: node.id,
                        parentId: input.parentConnect || null, // NOTE: This is overwritten later
                        translations,
                        userId: input.userConnect || userData.id,
                    };
                }

                // Update translation information for updated messages
                for (const { node, input } of Update) {
                    const messageData = preMap.messageData[node.id] as PreMapMessageDataUpdate | undefined;
                    const translationsUpdates = (input.translationsUpdate || []).filter(t => t.text && t.text.length > 0);
                    const translationsDeletes = (input.translationsDelete || []).filter(Boolean);
                    if (!messageData || (!translationsUpdates.length && !translationsDeletes.length)) {
                        continue;
                    }
                    let messageDataTranslations = messageData.translations || [];
                    // Apply updates
                    for (const { id, text, language } of translationsUpdates) {
                        const translation = messageDataTranslations.find(t => t.id === id);
                        if (translation) {
                            translation.text = text || "";
                        } else {
                            messageDataTranslations = [...messageDataTranslations, { id, language, text: text || "" }];
                        }
                    }
                    // Apply deletes
                    messageDataTranslations = messageDataTranslations.filter(t => !translationsDeletes.includes(t.id));

                    // Update messageData
                    messageData.translations = messageDataTranslations;
                }

                // Collect information for constructing the message tree, which is effected by creating and deleting messages on existing chats
                const chatsWithMessages: { [chatId: string]: { messagesCreate: ChatMessageCreateInput[]; messagesDelete: string[] } } = {};
                for (const { node, input } of [...Create, ...Delete]) {
                    // If chat is new, skip
                    const isChatNew = node.parent && node.parent.__typename === "Chat" && ["Create"].includes(node.parent.action);
                    if (isChatNew) continue;
                    const chatId = preMap.messageData[node.id]?.chatId;
                    if (!chatId) continue;
                    // Collect messages to create and delete for each chat
                    if (!chatsWithMessages[chatId]) chatsWithMessages[chatId] = { messagesCreate: [], messagesDelete: [] };
                    if (node.action === "Create") chatsWithMessages[chatId].messagesCreate.push(input as ChatMessageCreateInput);
                    else if (node.action === "Delete") chatsWithMessages[chatId].messagesDelete.push(input as string);
                }
                const chatUpdateInputs = Object.entries(chatsWithMessages).reduce((acc, [chatId, { messagesCreate, messagesDelete }]) => {
                    acc.push({
                        id: chatId,
                        messagesCreate,
                        messagesDelete,
                    });
                    return acc;
                }, [] as ChatUpdateInput[]);
                // Collect db information for constructing the message tree
                const { branchInfo } = await populatePreMapForChatUpdates({ updateInputs: chatUpdateInputs });
                for (const data of chatUpdateInputs) {
                    // Update Creates and Updates (and add new Updates if needed) to include parent and children information
                    const rest = { idsCreateToConnect: {}, preMap: {}, userData };
                    const operations = await shapeHelper({ relation: "messages", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "ChatMessage", parentRelationshipName: "chat", data, ...rest });
                    const { summary } = prepareChatMessageOperations(operations, branchInfo[data.id]);
                    for (const { id: messageId, parentId } of summary.Create) {
                        // Update the corresponding message in preMapMessageData
                        const messageData = preMap.messageData[messageId] as PreMapMessageDataCreate | PreMapMessageDataUpdate | undefined;
                        if (messageData) {
                            messageData.parentId = parentId;
                        }
                    }
                    for (const { id: messageId, parentId } of summary.Update) {
                        // If it exists, update the corresponding message in preMapMessageData
                        const messageData = preMap.messageData[messageId] as PreMapMessageDataCreate | PreMapMessageDataUpdate | undefined;
                        if (messageData) {
                            messageData.parentId = parentId;
                        }
                        // If it doesn't exist yet, this indicates the update is due to healing the tree 
                        // (i.e. a message was deleted, so its children need to be updated to point to the new parent).
                        // Add a new entry to preMapMessageData, and push new input to the Update array.
                        else {
                            const updateInput: { node: InputNode; input: ChatMessageUpdateInput; } = {
                                node: { __typename: "ChatMessage", id: messageId, action: "Update", children: [], parent: null },
                                input: { id: messageId }, // No updates needed. We'll get the parent ID from the preMap data
                            };
                            Update.push(updateInput);
                            preMap.messageData[messageId] = {
                                __type: "Update",
                                chatId: data.id,
                                messageId,
                                parentId,
                            };
                        }
                    }
                }

                // Parse potential bots and bots being removed
                const potentialBotIds: string[] = [];
                const participantsBeingRemovedIds: string[] = [];
                Object.entries(preMap.chatData).forEach(([id, chat]) => {
                    if (!chat.isNew) return;
                    if (chat.potentialBotIds) {
                        chat.potentialBotIds.forEach(botId => {
                            // Only add to potentialBotIds if the user ID is not already a key in botData (and also not your ID)
                            if (!preMap.userData[botId] && botId !== userData.id) potentialBotIds.push(botId);
                            // If it is already a key in bot data, update chatData.botParticipants
                            else if (preMap.userData[botId]) {
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
                        preMap.userData[user.id] = {
                            botSettings: user.botSettings ?? JSON.stringify({}),
                            id: user.id,
                            isBot: true,
                            name: user.name,
                        };
                        // Add any bot that is public or invited by you to participants
                        if (!user.isPrivate || user.invitedByUser?.id === userData.id) {
                            Object.entries(preMap.chatData).forEach(([id, chat]) => {
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
                        Object.values(preMap.chatData).forEach((chat) => {
                            if (chat.botParticipants?.includes(participant.user.id)) {
                                chat.botParticipants = chat.botParticipants.filter(id => id !== participant.user.id);
                            }
                        });
                        // NOTE: Don't remove from preMapUserData, since their messages may still be in the context for AI responses
                    });
                }
                // Messages can be created for you or bots. Make sure all new messages meet this criteria.
                for (const { node } of Create) {
                    const message = preMap.messageData[node.id] as PreMapMessageDataCreate | undefined;
                    if (message && message.userId !== userData.id && !Object.keys(preMap.userData).includes(message.userId ?? "")) {
                        throw new CustomError("0526", "Unauthorized", userData.languages, { message });
                    }
                }

                // Return data
                return preMap;
            },
            create: async ({ data, ...rest }) => {
                const preMap = rest.preMap[__typename] as ChatMessagePre;
                // Prefer parent ID from preMap data over what's provided by the client
                const messageData = preMap?.messageData[data.id] as PreMapMessageDataCreate | undefined;
                const parentId = messageData?.parentId !== undefined ? messageData.parentId : data.parentConnect;
                return {
                    id: data.id,
                    parent: parentId ? { connect: { id: parentId } } : undefined,
                    user: { connect: { id: data.userConnect ?? rest.userData.id } }, // Can create messages for bots. This is authenticated in the "pre" function.
                    versionIndex: data.versionIndex,
                    chat: await shapeHelper({ relation: "chat", relTypes: ["Connect"], isOneToOne: true, objectType: "Chat", parentRelationshipName: "messages", data, ...rest }),
                    translations: await translationShapeHelper({ relTypes: ["Create"], data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                const preMap = rest.preMap[__typename] as ChatMessagePre;
                // Allow parentId updates, but only from the pre function
                const messageData = preMap?.messageData[data.id] as PreMapMessageDataUpdate | undefined;
                const parentId = messageData?.parentId;
                return {
                    parent: parentId ? { connect: { id: parentId } } : undefined,
                    translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], data, ...rest }),
                };
            },
        },
        trigger: {
            afterMutations: async ({ additionalData, createdIds, deletedIds, updatedIds, preMap, resultsById, userData }) => {
                const preMapUserData: Record<string, PreMapUserData> = preMap[__typename]?.userData ?? {};
                const preMapChatData: Record<string, PreMapChatData> = preMap[__typename]?.chatData ?? {};
                const preMapMessageData: Record<string, PreMapMessageData> = preMap[__typename]?.messageData ?? {};
                // Call triggers
                for (const objectId of createdIds) {
                    const messageData = preMapMessageData[objectId] as PreMapMessageDataCreate | undefined;
                    if (!messageData) {
                        logger.error("Message data not found", { trace: "0238", user: userData.id, message: objectId });
                        continue;
                    }
                    const chatId = messageData.chatId;
                    const chatMessage = resultsById[objectId] as ChatMessage | undefined;
                    const senderId = messageData.userId;
                    if (!chatMessage || !chatId || !senderId) {
                        logger.error("Message, message sender, or chat not found", { trace: "0363", user: userData.id, messageId: objectId, chatId });
                        continue;
                    }
                    const messageText = getTranslation({ translations: messageData.translations }, userData.languages).text || null;
                    // Add message to cache
                    await (new ChatContextManager()).addMessage(messageData);
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
                    const botsToRespond = determineRespondingBots(messageText, messageData.userId, chat, bots, userData.id);
                    if (botsToRespond.length) {
                        // Send typing indicator while bots are responding
                        emitSocketEvent("typing", chatId, { starting: botsToRespond });
                        const task = additionalData.task || "Start";
                        const taskContexts = Array.isArray(additionalData.taskContexts) ? additionalData.taskContexts : [];
                        // For each bot that should respond, request bot response
                        for (const botId of botsToRespond) {
                            // Call LLM for bot response
                            requestBotResponse({
                                chatId,
                                mode: "text",
                                parentId: messageData.messageId,
                                parentMessage: messageText,
                                respondingBotId: botId,
                                shouldNotRunTasks: false,
                                task,
                                taskContexts,
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
                    // Update message in cache
                    const messageData = preMapMessageData[objectId] as PreMapMessageDataUpdate | undefined;
                    if (messageData) {
                        await (new ChatContextManager()).editMessage(messageData);
                    }
                    //TODO should probably call determineRespondingBots and requestBotResponse here as well
                    const chatMessage = resultsById[objectId] as ChatMessage | undefined;
                    if (!chatMessage) {
                        logger.error("Result message not found", { trace: "0365", user: userData.id, messageId: objectId });
                        continue;
                    }
                    await Trigger(userData.languages).chatMessageUpdated({
                        data: preMapMessageData[objectId],
                        message: chatMessage,
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
                    const messageData = preMapMessageData[objectId] as PreMapMessageDataDelete | undefined;
                    if (messageData) {
                        await Trigger(userData.languages).chatMessageDeleted({
                            data: messageData,
                        });
                    } else {
                        logger.error("Message data not found", { trace: "0067", user: userData.id, message: objectId });
                    }
                }
            },
        },
        yup: chatMessageValidation,
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
            const take = input.take ?? DEFAULT_CHAT_TAKE;
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
                        ...(await getSingleTypePermissions<ChatMessageModelInfo["GqlPermission"]>(__typename, ids, userData)),
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
            const isParticipant = uuidValidate(userId) && (data.chat as Record<string, any>).participants?.some((p) => p.user?.id === userId || p.userId === userId);
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
            own: function getOwn(data) {
                return { // If you own the chat or created the message
                    OR: [
                        { chat: useVisibility("Chat", "Own", data) },
                        { user: { id: data.userId } },
                    ],
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    OR: [
                        { chat: useVisibility("Chat", "Own", data) },
                        { chat: useVisibility("Chat", "Public", data) },
                    ],
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return { chat: useVisibility("Chat", "OwnPrivate", data) };
            },
            ownPublic: function getOwnPublic(data) {
                return { chat: useVisibility("Chat", "OwnPublic", data) };
            },
            public: function getPublic(data) {
                return {
                    chat: useVisibility("Chat", "Public", data),
                };
            },
        },
    }),
});
