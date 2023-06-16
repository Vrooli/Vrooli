import { chatInviteValidation, ChatMessageCreateInput, ChatMessageSortBy, MaxObjects, uuidValidate } from "@local/shared";
import { shapeHelper } from "../../builders";
import { Trigger } from "../../events";
import { bestTranslation, defaultPermissions, translationShapeHelper } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { ChatMessageFormat } from "../format/chatMessage";
import { ModelLogic } from "../types";
import { ChatModel } from "./chat";
import { ReactionModel } from "./reaction";
import { ChatMessageModelLogic } from "./types";
import { UserModel } from "./user";

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
                // Collect data to ensure that trigger has chat ID and user ID 
                // for each message, whether it's a create, update, or delete
                const messageData: Record<string, { chatId: string; userId: string }> = {};
                // Loop through createList and updateList. Collect IDs with missing data
                const missingDataIds = new Set<string>();
                for (const d of [...createList, ...updateList.map(({ where, data }) => ({ ...data, id: where.id }))]) {
                    if ((d as ChatMessageCreateInput).chatConnect) {
                        messageData[d.id] = { chatId: (d as ChatMessageCreateInput).chatConnect, userId: userData.id };
                    } else {
                        missingDataIds.add(d.id);
                    }
                }
                // Add every delete ID to missingDataIds
                for (const d of deleteList) {
                    missingDataIds.add(d);
                }
                // If there are any missing data IDs, query the database for them
                if (missingDataIds.size > 0) {
                    const messages = await prisma.chat_message.findMany({
                        where: { id: { in: [...missingDataIds] } },
                        select: { id: true, chatId: true, userId: true },
                    });
                    for (const { id, chatId, userId } of messages) {
                        messageData[id] = { chatId: chatId ?? "", userId: userId ?? "" };
                    }
                }
                // Return the message data
                return { messageData };
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
                { user: UserModel.search!.searchStringQuery() },
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
            Organization: (data.chat as any).organization,
            User: (data.chat as any).user,
        }),
        permissionResolvers: ({ data, isAdmin, isDeleted, isLoggedIn, isPublic, userId }) => {
            const isParticipant = uuidValidate(userId) && (data.chat as any).participants?.some((p) => p.userId === userId);
            return {
                ...defaultPermissions({ isAdmin, isDeleted, isLoggedIn, isPublic }),
                canReply: () => isLoggedIn && !isDeleted && (isAdmin || isParticipant),
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
                chat: ChatModel.validate!.visibility.owner(userId),
            }),
        },
    },
});
