import { ChatCreateInput, ChatInviteCreateInput, chatInviteValidation, ChatMessage, ChatMessageCreateInput, ChatMessageSearchTreeInput, ChatMessageSearchTreeResult, ChatMessageSortBy, ChatMessageUpdateInput, ChatUpdateInput, MaxObjects, uuidValidate } from "@local/shared";
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
import { chatMessage_findMany } from "../../endpoints";
import { CustomError } from "../../events/error";
import { logger } from "../../events/logger";
import { Trigger } from "../../events/trigger";
import { io } from "../../io";
import { UI_URL } from "../../server";
import { ChatContextManager } from "../../tasks/llm/context";
import { requestBotResponse } from "../../tasks/llm/queue";
import { getAuthenticatedData, permissionsCheck, PrismaType } from "../../types";
import { bestTranslation, SortMap } from "../../utils";
import { translationShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions, isOwnerAdminCheck } from "../../validators";
import { ChatMessageFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { ChatMessageModelLogic, ChatModelInfo, ChatModelLogic, ReactionModelLogic, UserModelLogic } from "./types";

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
    userId: string;
}

/** Information for a message's corresponding chat, collected in mutate.shape.pre */
export type PreMapChatData = {
    botParticipants?: string[],
    potentialBotIds?: string[],
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
    name: string;
};

export type ChatMessageBeforeDeletedData = {
    id: string,
    chatId: string | undefined,
    userId: string | undefined,
    parentId: string | undefined,
    childIds: string[],
}

const __typename = "ChatMessage" as const;
export const ChatMessageModel: ChatMessageModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.chat_message,
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, text: true } } }),
            get: (select, languages) => {
                const best = bestTranslation(select.translations, languages)?.text ?? "";
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
            pre: async ({ Create, Update, Delete, prisma, userData, inputsById }) => {
                // Initialize objects to store bot, message, and chat information
                const preMapUserData: Record<string, PreMapUserData> = {};
                const preMapChatData: Record<string, PreMapChatData> = {};
                const preMapMessageData: Record<string, PreMapMessageData> = {};
                // Find known create/update information. We'll query for the rest later.
                for (const { node, input } of [...Create, ...Update]) {
                    const translations = [...((input as ChatMessageCreateInput).translationsCreate ?? []), ...((input as ChatMessageUpdateInput).translationsUpdate ?? [])].filter(t => t.text) as { language: string, text: string }[];
                    // While messages can technically be created in multiple languages, we'll only worry about one
                    const best = bestTranslation(translations, userData.languages);
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
                const chatSelect = messageIdsForMissingChats.length > 0 ? {
                    OR: [
                        { id: { in: knownExistingChatIds } },
                        { messages: { some: { id: { in: messageIdsForMissingChats } } } },
                    ],
                } : { id: { in: knownExistingChatIds } };
                // Query existing chat information
                const existingChatInfo = await prisma.chat.findMany({
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
                        // Find most recent message
                        messages: {
                            orderBy: { sequence: "desc" },
                            take: 1,
                            select: {
                                id: true,
                            },
                        },
                        // Find number of participants
                        _count: { select: { participants: true } },
                    },
                });
                // Parse chat and bot information
                existingChatInfo.forEach(chat => {
                    // Find bots you are allowed to talk to
                    const allowedBots = chat.participants.filter(p => p.user.isBot && (!p.user.isPrivate || p.user.invitedByUser?.id === userData.id));
                    // Set chat information
                    preMapChatData[chat.id] = {
                        isNew: false,
                        botParticipants: allowedBots.map(p => p.user.id),
                        participantsCount: chat._count.participants,
                        lastMessageId: chat.messages.length > 0 ? chat.messages[0].id : undefined,
                    };
                    // Set information about all participants
                    chat.participants.forEach(p => {
                        preMapUserData[p.user.id] = {
                            botSettings: p.user.botSettings ?? JSON.stringify({}),
                            id: p.user.id,
                            name: p.user.name,
                        };
                    });
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
                    const potentialBots = await prisma.user.findMany({
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
                    const participantsBeingRemoved = await prisma.chat_participants.findMany({
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
                    if (message.isNew && message.userId !== userData.id && !Object.keys(preMapUserData).includes(message.userId)) {
                        throw new CustomError("0526", "Unauthorized", ["en"], { message });
                    }
                });
                // Return data
                return { userData: preMapUserData, chatData: preMapChatData, messageData: preMapMessageData };
            },
            create: async ({ data, ...rest }) => {
                const parentId = rest.preMap[__typename]?.messageData?.[data.id]?.parentId;
                return {
                    id: data.id,
                    user: { connect: { id: data.userConnect ?? rest.userData.id } }, // Can create messages for bots. This is authenticated in the "pre" function.
                    ...(parentId ? { parent: { connect: { id: parentId } } } : {}),
                    ...(await shapeHelper({ relation: "chat", relTypes: ["Connect"], isOneToOne: true, objectType: "Chat", parentRelationshipName: "messages", data, ...rest })),
                    ...(await translationShapeHelper({ relTypes: ["Create"], data, ...rest })),
                };
            },
            update: async ({ data, ...rest }) => ({
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], data, ...rest })),
            }),
        },
        trigger: {
            beforeDeleted: async ({ beforeDeletedData, deletingIds, prisma }) => {
                // Find the chat user, parent, and children for each message being deleted
                const deleting = await prisma.chat_message.findMany({
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
            afterMutations: async ({ beforeDeletedData, createdIds, deletedIds, updatedIds, preMap, prisma, userData }) => {
                const preMapUserData: Record<string, PreMapUserData> = preMap[__typename]?.userData ?? {};
                const preMapChatData: Record<string, PreMapChatData> = preMap[__typename]?.chatData ?? {};
                const preMapMessageData: Record<string, PreMapMessageData> = preMap[__typename]?.messageData ?? {};
                let messages: ChatMessage[] = [];
                if (createdIds.length > 0 || updatedIds.length > 0) {
                    const paginatedMessages = await readManyHelper({
                        info: chatMessage_findMany,
                        input: { ids: [...createdIds, ...updatedIds], take: createdIds.length + updatedIds.length },
                        objectType: __typename,
                        prisma,
                        req: { session: { languages: userData.languages, users: [userData] } },
                    });
                    messages = paginatedMessages.edges.map(e => e.node);
                }
                // Call triggers
                for (const objectId of createdIds) {
                    const message: PreMapMessageData = preMapMessageData[objectId];
                    // Add message to cache
                    await (new ChatContextManager()).addMessage(message.chatId as string, message);
                    // Common trigger logic
                    await Trigger(prisma, userData.languages).objectCreated({
                        createdById: userData.id,
                        hasCompleteAndPublic: true, // N/A
                        hasParent: true, // N/A
                        owner: { id: userData.id, __typename: "User" },
                        objectId,
                        objectType: __typename,
                    });
                    await Trigger(prisma, userData.languages).chatMessageCreated({
                        createdById: userData.id,
                        data: preMapMessageData[objectId],
                        message: messages.find(m => m.id === objectId) as ChatMessage,
                    });
                    // Determine which bots should respond, if any.
                    // Here are the conditions:
                    // 1. If the message content is blank (likely meaning the message was updated but not its actual content), then no bots should respond.
                    // 2. If the message is not associated with your user ID, no bots should respond.
                    // 3. If there are no bots in the chat, no bots should respond.
                    // 4. If there is one bot in the chat and two participants (i.e. just you and the bot), the bot should respond.
                    // 5. Otherwise, we must check the message to see if any bots were mentioned
                    // Get message and bot data
                    const chat: PreMapChatData | undefined = message.chatId ? preMapChatData[message.chatId] : undefined;
                    const bots: PreMapUserData[] = chat?.botParticipants?.map(id => preMapUserData[id]).filter(b => b) ?? [];
                    if (!chat) continue;
                    // Check condition 1
                    if (message.content.trim() === "") continue;
                    // Check condition 2
                    if (message.userId !== userData.id) continue;
                    // Check condition 3
                    if (bots.length === 0) continue;
                    // Check condition 4
                    if (bots.length === 1 && chat.participantsCount === 2) {
                        const botId = bots[0].id;
                        // Call LLM for bot response
                        requestBotResponse({
                            chatId: message.chatId as string,
                            messageId: objectId,
                            message,
                            respondingBotId: botId,
                            participantsData: preMapUserData,
                            userData,
                        });
                    }
                    // Check condition 5
                    else {
                        // Find markdown links in the message
                        const linkStrings = message.content.match(/\[([^\]]+)\]\(([^)]+)\)/g);
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
                            botsToRespond = chat.botParticipants ?? [];
                        }
                        // Otherwise, find the bots that were mentioned by name (e.g. "@BotName")
                        else {
                            botsToRespond = links.map(l => {
                                const botId = bots.find(b => b.name === l.label.slice(1))?.id;
                                if (!botId) return null;
                                return botId;
                            }).filter(id => id !== null) as string[];
                            // Remove duplicates
                            botsToRespond = [...new Set(botsToRespond)];
                        }
                        // Send typing indicator while bots are responding
                        io.to(message.chatId as string).emit("typing", { starting: botsToRespond });
                        // For each bot that should respond, request bot response
                        for (const botId of botsToRespond) {
                            // Call LLM for bot response
                            requestBotResponse({
                                chatId: message.chatId as string,
                                messageId: objectId,
                                message,
                                respondingBotId: botId,
                                participantsData: preMapUserData,
                                userData,
                            });
                        }
                    }
                }
                for (const objectId of updatedIds) {
                    await Trigger(prisma, userData.languages).objectUpdated({
                        updatedById: userData.id,
                        hasCompleteAndPublic: true, // N/A
                        hasParent: true, // N/A
                        owner: { id: userData.id, __typename: "User" },
                        objectId,
                        objectType: __typename,
                        wasCompleteAndPublic: true, // N/A
                    });
                    await Trigger(prisma, userData.languages).chatMessageUpdated({
                        data: preMapMessageData[objectId],
                        message: messages.find(m => m.id === objectId) as ChatMessage,
                    });
                }
                for (const objectId of deletedIds) {
                    await Trigger(prisma, userData.languages).objectDeleted({
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
                            await prisma.chat_message.updateMany({
                                where: { id: { in: messageData.childIds ?? [] } },
                                data: { parentId: messageData.parentId },
                            });
                        }
                        await Trigger(prisma, userData.languages).chatMessageDeleted({
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
            prisma: PrismaType,
            req: Request,
            input: ChatMessageSearchTreeInput,
            info: GraphQLInfo | PartialGraphQLInfo,
        ): Promise<ChatMessageSearchTreeResult> {
            if (!input.chatId) throw new CustomError("0531", "InvalidArgs", getUser(req.session)?.languages ?? ["en"], { input });
            // Query for all authentication data
            const userData = getUser(req.session);
            const authDataById = await getAuthenticatedData({ "Chat": [input.chatId] }, prisma, userData ?? null);
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
            const totalInThread = await prisma.chat_message.count({
                where: { chatId: input.chatId },
            });
            // If it's less than or equal to the take amount, we can just return all messages. 
            const take = input.take ?? 50;
            if (totalInThread <= take) {
                let messages: any[] = await prisma.chat_message.findMany({
                    where: { chatId: input.chatId },
                    orderBy,
                    take,
                    ...selectHelper(partial.messages as PartialGraphQLInfo),
                });
                messages = messages.map((c: any) => modelToGql(c, partial.messages as PartialGraphQLInfo));
                messages = await addSupplementalFields(prisma, getUser(req.session), messages, partial.messages as PartialGraphQLInfo);
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
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        reaction: await ModelMap.get<ReactionModelLogic>("Reaction").query.getReactions(prisma, userData?.id, ids, __typename),
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
            private: {},
            public: {},
            owner: (userId) => ({
                chat: ModelMap.get<ChatModelLogic>("Chat").validate().visibility.owner(userId),
            }),
        },
    }),
});
