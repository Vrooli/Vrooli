import { chatInviteValidation, ChatMessageCreateInput, ChatMessageSortBy, ChatMessageUpdateInput, MaxObjects, uuidValidate } from "@local/shared";
import { shapeHelper } from "../../builders";
import { Trigger } from "../../events";
import { SERVER_URL } from "../../server";
import { bestTranslation, translationShapeHelper } from "../../utils";
import { getSingleTypePermissions, isOwnerAdminCheck } from "../../validators";
import { ChatMessageFormat } from "../format/chatMessage";
import { ModelLogic } from "../types";
import { ChatModel } from "./chat";
import { ReactionModel } from "./reaction";
import { ChatMessageModelLogic, ChatModelLogic, UserModelLogic } from "./types";
import { UserModel } from "./user";

/** Information for a message, collected in mutate.shape.pre */
type MessageData = {
    /** IDs of bots in this message's chat */
    botIds: string[];
    /** ID of the chat this message belongs to */
    chatId: string | null;
    /** Content in user's preferred (or closest to preferred) language */
    content: string;
    /** Language code for message content */
    language: string;
    /** Total number of participants in the chat, including bots, users, and the current user */
    participantsCount: number | null;
    /** ID of the user who sent this message */
    userId: string;
}

/** 
 * Fields that we'll use to set up bot context. 
 * Taken by combining parsed bot settings wiht other information 
 * from the user object.
 * */
type BotData = {
    name: string;
    /** Taken from bio */
    description: string;
    /** 
     * If bot data has translated information.
     * Numbers represent scales from 0 to 1 (e.g. creativity, verbosity)
     **/
    translations: { [languageCode: string]: string | number };
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
            pre: async ({ createList, updateList, deleteList, prisma, userData }) => {
                // Collect bot information for bots to respond to messages.
                // Collect chatIds and userIds to send notifications and trigger web socket events.
                const botData: Record<string, string> = {};
                const messageData: Record<string, MessageData> = {};
                // Find known information. We'll query for the rest later.
                for (const d of [...createList, ...updateList]) {
                    const best = bestTranslation([...((d as ChatMessageCreateInput).translationsCreate ?? []), ...((d as ChatMessageUpdateInput).translationsUpdate ?? [])], userData.languages);
                    messageData[d.id] = {
                        botIds: [],
                        chatId: (d as ChatMessageCreateInput).chatConnect ?? null,
                        content: best?.text ?? "",
                        language: best?.language ?? userData.languages[0],
                        participantsCount: null,
                        userId: (d as ChatMessageCreateInput).userConnect ?? userData.id,
                    };
                }
                for (const d of deleteList) {
                    messageData[d] = {
                        botIds: [],
                        chatId: null,
                        content: "",
                        language: userData.languages[0],
                        participantsCount: null,
                        userId: userData.id,
                    };
                }
                // Query chat information of new messages
                const chatIdsForNewMessages = createList.map(c => c.chatConnect);
                const chatInfoForNewMessages = await prisma.chat.findMany({
                    where: { id: { in: chatIdsForNewMessages } },
                    select: {
                        id: true,
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
                                        botSettings: true,
                                    },
                                },
                            },
                        },
                        _count: { select: { participants: true } },
                    },
                });
                // Add chat information to message data
                chatInfoForNewMessages.forEach(chat => {
                    const message = createList.find(m => m.chatConnect === chat.id);
                    if (message) {
                        messageData[message.id] = {
                            ...messageData[message.id],
                            botIds: chat.participants.map(p => {
                                botData[p.user.id] = p.user.botSettings ?? JSON.stringify({});
                                return p.user.id;
                            }),
                            chatId: chat.id,
                            participantsCount: chat._count.participants,
                            userId: userData.id,
                        };
                    }
                });
                // Query message and chat information for updated and deleted messages
                const queriedData = await prisma.chat_message.findMany({
                    where: { id: { in: [...updateList.map(u => u.id), ...deleteList] } },
                    select: {
                        id: true,
                        chat: {
                            select: {
                                id: true,
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
                                                botSettings: true,
                                            },
                                        },
                                    },
                                },
                                _count: { select: { participants: true } },
                            },
                        },
                        user: { select: { id: true } },
                    },
                });
                // Add queriedData into messageData
                queriedData.forEach(d => {
                    if (!messageData[d.id]) {
                        messageData[d.id] = {
                            ...messageData[d.id],
                            botIds: d.chat?.participants.map(p => {
                                botData[p.user.id] = p.user.botSettings ?? JSON.stringify({});
                                return p.user.id;
                            }) ?? [],
                            chatId: d.chat?.id ?? null,
                            participantsCount: d.chat?._count?.participants ?? null,
                            userId: d.user?.id ?? "",
                        };
                    }
                });
                // Return data
                return { botData, messageData };
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
                    // 1. If there are no bots in the chat, no bots should respond.
                    // 2. If there is one bot in the chat and two participants (i.e. just you and the bot), the bot should respond.
                    // 3. Otherwise, we must check the message to see if any bots were mentioned
                    // Get message and bot data
                    const message: MessageData = preMap[__typename].messageData[c.id];
                    const botIds = preMap[__typename].botData;
                    // Check condition 1
                    if (botIds.length === 0) continue;
                    // Check condition 2
                    if (botIds.length === 1 && message.participantsCount === 2) {
                        //TODO
                    }
                    // Check condition 3
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
                            botsToRespond = botIds;
                        }
                        // Otherwise, find the bots that were mentioned by name (e.g. "@BotName")
                        else {
                            botsToRespond = links.map(l => {
                                const botId = Object.keys(botIds).find(id => botIds[id].name === l.label.slice(1));
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
        permissionResolvers: ({ data, isAdmin: isMessageOwner, isDeleted, isLoggedIn, isPublic, userId }) => {
            const isChatAdmin = userId ? isOwnerAdminCheck(ChatModel.validate.owner(data.chat as ChatModelLogic["PrismaModel"], userId), userId) : false, ;
            const isParticipant = uuidValidate(userId) && (data.chat as ChatModelLogic["PrismaModel"]).participants?.some((p) => p.userId === userId);
            const isPublicBot = (data.user as UserModelLogic["PrismaModel"]).isBot && (data.user as UserModelLogic["PrismaModel"]).isPrivate === false;
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
