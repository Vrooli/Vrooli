import { ChatCreateInput, chatInviteValidation, ChatMessageCreateInput, ChatMessageSortBy, ChatMessageUpdateInput, ChatUpdateInput, MaxObjects, uuidValidate } from "@local/shared";
import { shapeHelper } from "../../builders";
import { CustomError, Trigger } from "../../events";
import { SERVER_URL } from "../../server";
import { bestTranslation, translationShapeHelper } from "../../utils";
import { getSingleTypePermissions, isOwnerAdminCheck } from "../../validators";
import { ChatMessageFormat } from "../formats";
import { ModelLogic } from "../types";
import { ChatModel } from "./chat";
import { ReactionModel } from "./reaction";
import { ChatMessageModelLogic, ChatModelLogic } from "./types";
import { UserModel } from "./user";

/** Information for a message, collected in mutate.shape.pre */
type MessageData = {
    /** ID of the chat this message belongs to */
    chatId: string | null;
    /** Content in user's preferred (or closest to preferred) language */
    content: string;
    isNew: boolean;
    /** Language code for message content */
    language: string;
    /** ID of the user who sent this message */
    userId: string;
}

type ChatData = {
    botParticipants?: string[],
    potentialBotIds?: string[],
    participantsDelete?: string[],
    isNew: boolean,
    /** Total number of participants in the chat, including bots, users, and the current user */
    participantsCount?: number,
    /** Determines configuration to provide for AI responses */
    task?: string, //TODO can find taask in idsToInputs[node.parent.id].task for creates, but need way for updates. 
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
 * Fields that we'll use to set up bot context. 
 * Taken by combining parsed bot settings wiht other information 
 * from the user object.
 * */
type BotData = {
    botSettings: string,
    name: string;
} & { [key: string]: string };

const __typename = "ChatMessage" as const;
const suppFields = ["you"] as const;
export const ChatMessageModel: ModelLogic<ChatMessageModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.chat_message,
    display: {
        label: {
            select: () => ({ id: true, translations: { select: { language: true, text: true } } }),
            get: (select, languages) => {
                const best = bestTranslation(select.translations, languages)?.text ?? "";
                return best.length > 30 ? best.slice(0, 30) + "..." : best;
            },
        },
    },
    format: ChatMessageFormat,
    mutate: {
        shape: {
            /**
             * Collects bot information, message information, and chat information for AI responses, 
             * notifications, and web socket events.
             * 
             * NOTE: Updated messages don't trigger AI responses. Instead, you must create a new message where 
             * "isFork" is true, and "forkId" is the ID of the message you want to update. This means we only need 
             * to collect the chat ID for updated messages (to emit to web sockets).
             */
            pre: async ({ Create, Update, Delete, prisma, userData, inputsById }) => {
                // Initialize objects to store bot, message, and chat information
                const botData: Record<string, BotData> = {};
                const chatData: Record<string, ChatData> = {};
                const messageData: Record<string, MessageData> = {};
                // Find known create/update information. We'll query for the rest later.
                for (const { node, input } of [...Create, ...Update]) {
                    // While messages can technically be created in multiple languages, we'll only worry about one
                    const best = bestTranslation([...((input as ChatMessageCreateInput).translationsCreate ?? []), ...((input as ChatMessageUpdateInput).translationsUpdate ?? [])], userData.languages);
                    // Collect chat information
                    let chatId = (input as ChatMessageCreateInput).chatConnect ?? null;
                    // If chat is being created or updated, we may not need to query for it later
                    if (node.parent && node.parent.__typename === "Chat" && ["Create", "Update"].includes(node.parent.action)) {
                        const chatUpsertInfo = inputsById[node.parent.id]?.input as ChatCreateInput | ChatUpdateInput | undefined;
                        if (chatUpsertInfo?.id) {
                            chatId = chatUpsertInfo.id;
                            // Store all invite information. Later we'll check if any of these are bots (which are automatically accepted, 
                            // and can potentially be used for AI responses)
                            chatData[chatId] = {
                                potentialBotIds: chatUpsertInfo.invitesCreate?.map(i => i.userConnect) ?? [],
                                participantsDelete: ((chatUpsertInfo as ChatUpdateInput).participantsDelete ?? []),
                                isNew: node.parent.action === "Create",
                            };
                        }
                    }
                    // Collect message data
                    messageData[input.id] = {
                        chatId,
                        content: best?.text ?? "",
                        isNew: node.action === "Create",
                        language: best?.language ?? userData.languages[0],
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
                            chatData[chatId] = {
                                isNew: false,
                            };
                        }
                    }
                    // Collect message data
                    messageData[id] = {
                        chatId,
                        content: "",
                        isNew: false,
                        language: "",
                        userId: "",
                    };
                }
                // Find chat information of every chat in chatData and every chat missing from messageData
                const messageIdsForMissingChats = Object.entries(messageData).filter(([, data]) => !data.chatId).map(([id]) => id);
                const knownExistingChatIds = Object.entries(chatData).filter(([, data]) => !data.isNew).map(([id]) => id);
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
                        // Find all bots
                        participants: {
                            where: {
                                user: {
                                    AND: [
                                        { id: { not: userData.id } },
                                        { isBot: true },
                                    ],
                                },
                            },
                            select: {
                                user: {
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
                                },
                            },
                        },
                        // Find number of participants
                        _count: { select: { participants: true } },
                    },
                });
                // Parse chat and bot information
                existingChatInfo.forEach(chat => {
                    // Filter out bots that are private and not invited by the current user
                    const allowedBots = chat.participants.filter(p => !p.user.isPrivate || p.user.invitedByUser?.id === userData.id);
                    chatData[chat.id] = {
                        isNew: false,
                        botParticipants: allowedBots.map(p => p.user.id),
                        participantsCount: chat._count.participants,
                    };
                    allowedBots.forEach(p => {
                        botData[p.user.id] = {
                            botSettings: p.user.botSettings ?? JSON.stringify({}),
                            name: p.user.name,
                        };
                    });
                });
                // Parse potential bots and bots being removed
                const potentialBotIds: string[] = [];
                const participantsBeingRemovedIds: string[] = [];
                Object.entries(chatData).forEach(([id, chat]) => {
                    if (!chat.isNew) return;
                    if (chat.potentialBotIds) {
                        chat.potentialBotIds.forEach(botId => {
                            // Only add to potentialBotIds if the user ID is not already a key in botData (and also not your ID)
                            if (!botData[botId] && botId !== userData.id) potentialBotIds.push(botId);
                            // If it is already a key in bot data, update chatData.botParticipants
                            else if (botData[botId]) {
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
                    potentialBots.forEach(bot => {
                        // Make sure we're only adding bots that are public or invited by you
                        if (!bot.isPrivate || bot.invitedByUser?.id === userData.id) {
                            botData[bot.id] = {
                                botSettings: bot.botSettings ?? JSON.stringify({}),
                                name: bot.name,
                            };
                            // Also add to chatData.botParticipants
                            Object.entries(chatData).forEach(([id, chat]) => {
                                if (chat.potentialBotIds?.includes(bot.id) && !chat.botParticipants?.includes(bot.id)) {
                                    if (!chat.botParticipants) chat.botParticipants = [];
                                    chat.botParticipants.push(bot.id);
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
                        // Remove from botData
                        if (botData[participant.user.id]) delete botData[participant.user.id];
                        // Remove from chatData.botParticipants
                        Object.values(chatData).forEach((chat) => {
                            if (chat.botParticipants?.includes(participant.user.id)) {
                                chat.botParticipants = chat.botParticipants.filter(id => id !== participant.user.id);
                            }
                        });
                    });
                }
                // Messages can be created for you or bots. Make sure all new messages meet this criteria.
                Object.values(messageData).forEach((message) => {
                    // If the message is new, but the user ID is not yours or a bot, throw an error
                    if (message.isNew && message.userId !== userData.id && !Object.keys(botData).includes(message.userId)) {
                        throw new CustomError("0526", "InternalError", ["en"], { message });
                    }
                });
                // Return data
                return { botData, chatData, messageData };
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                isFork: data.isFork,
                user: { connect: { id: rest.userData.id } },
                ...(data.forkId ? { fork: { connect: { id: data.forkId } } } : {}),
                ...(await shapeHelper({ relation: "chat", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "Chat", parentRelationshipName: "messages", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
            }),
            post: async ({ created, deletedIds, updated, preMap, prisma, userData }) => {
                const messageData = preMap[__typename].messageData;
                const botData = preMap[__typename].botData;
                const chatData = preMap[__typename].chatData;
                // Call triggers
                for (const c of created) {
                    // Common trigger logic
                    await Trigger(prisma, userData.languages).objectCreated({
                        createdById: userData.id,
                        hasCompleteAndPublic: true, // N/A
                        hasParent: true, // N/A
                        owner: { id: userData.id, __typename: "User" },
                        object: {
                            ...c,
                            ...preMap[__typename].messageData[c.id],
                        },
                        objectType: __typename,
                    });
                    // Determine which bots should respond, if any.
                    // Here are the conditions:
                    // 1. If the message content is blank (likely meaning the message was updated but not its actual content), then no bots should respond.
                    // 2. If the message is not associated with your user ID, no bots should respond.
                    // 3. If there are no bots in the chat, no bots should respond.
                    // 4. If there is one bot in the chat and two participants (i.e. just you and the bot), the bot should respond.
                    // 5. Otherwise, we must check the message to see if any bots were mentioned
                    // Get message and bot data
                    const message: MessageData = messageData[c.id];
                    const chat: ChatData | undefined = message.chatId ? chatData[message.chatId] : undefined;
                    const bots: BotData[] = chat?.botParticipants?.map(id => botData[id]).filter(b => b) ?? [];
                    if (!chat) continue;
                    // Check condition 1
                    if (message.content.trim() === "") continue;
                    // Check condition 2
                    if (message.userId !== userData.id) continue;
                    // Check condition 3
                    if (bots.length === 0) continue;
                    // Check condition 4
                    if (bots.length === 1 && chat.participantsCount === 2) {
                        //TODO
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
                        const correctOrigin = new URL(SERVER_URL).origin;
                        links = links.filter(l => {
                            if (!l.label.startsWith("@")) return false;
                            try {
                                const url = new URL(l.link);
                                return url.origin === correctOrigin;
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
                        // For each bot that should respond, request bot response
                        for (const botId of botsToRespond) {
                            //TODO
                        }
                    }
                }
                for (const u of updated) {
                    await Trigger(prisma, userData.languages).objectUpdated({
                        updatedById: userData.id,
                        hasCompleteAndPublic: true, // N/A
                        hasParent: true, // N/A
                        owner: { id: userData.id, __typename: "User" },
                        object: {
                            ...u,
                            ...preMap[__typename].messageData[u.id],
                        },
                        objectType: __typename,
                        wasCompleteAndPublic: true, // N/A
                    });
                }
                for (const d of deletedIds) {
                    await Trigger(prisma, userData.languages).objectDeleted({
                        deletedById: userData.id,
                        hasBeenTransferred: false, // N/A
                        hasParent: true, // N/A
                        object: {
                            id: d,
                            ...preMap[__typename].messageData[d],
                        },
                        objectType: __typename,
                        wasCompleteAndPublic: true, // N/A
                    });
                }
            },
        },
        yup: chatInviteValidation,
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
                { user: UserModel.search.searchStringQuery() },
            ],
        }),
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        reaction: await ReactionModel.query.getReactions(prisma, userData?.id, ids, __typename),
                    },
                };
            },
        },
    },
    validate: {
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data.user,
        }),
        permissionResolvers: ({ data, isAdmin: isMessageOwner, isDeleted, isLoggedIn, userId }) => {
            const isChatAdmin = userId ? isOwnerAdminCheck(ChatModel.validate.owner(data.chat as ChatModelLogic["PrismaModel"], userId), userId) : false;
            const isParticipant = uuidValidate(userId) && (data.chat as ChatModelLogic["PrismaModel"]).participants?.some((p) => p.userId === userId);
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
            chat: ["Chat", ["invites"]],
            user: "User",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                chat: ChatModel.validate.visibility.owner(userId),
            }),
        },
    },
});
