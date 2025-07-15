import { ChatInviteStatus, ChatSortBy, chatValidation, generatePublicId, getTranslation, MaxObjects, validatePK } from "@vrooli/shared";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { DbProvider } from "../../db/provider.js";
import { conversationStateStore } from "../../services/response/chatStore.js";
import { BranchInfoLoader, type ChatPreBranchInfo, MessageTree } from "../../utils/messageTree.js";
import { preShapeEmbeddableTranslatable } from "../../utils/shapes/preShapeEmbeddableTranslatable.js";
import { translationShapeHelper } from "../../utils/shapes/translationShapeHelper.js";
import { defaultPermissions, getSingleTypePermissions } from "../../validators/permissions.js";
import { ChatFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { type ChatModelInfo, type ChatModelLogic } from "./types.js";

/**
 * All pre-map data that should be collected for a chat transaction.
 */
type ChatPre = {
    /** 
     * Users whose invitations are allowed to be auto-approved
     * (meaning the bot is public or you own it)
     */
    bots: { id: string }[];
    /** 
     * Information about the chat's branch structure, 
     * including where to start adding new messages (if not specified by the client) 
     * and how to patch up the tree after messages are deleted.
     */
    branchInfo: Record<string, ChatPreBranchInfo>;
    /**
     * Indicates which chat translations need their search embeddings updated.
     */
    embeddingNeedsUpdateMap: { [chatId: string]: { [language: string]: boolean } };
};

const __typename = "Chat" as const;
export const ChatModel: ChatModelLogic = ({
    __typename,
    dbTable: "chat",
    dbTranslationTable: "chat_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: ({ translations }, languages) => getTranslation({ translations }, languages).name ?? "",
        },
    }),
    format: ChatFormat,
    mutate: {
        shape: {
            // inside ChatModel
            pre: async ({ Create, Update, userData }): Promise<ChatPre> => {
                /* 1️⃣  Which chats are we about to touch? */
                const updateIds = Update.map(u => u.input.id);

                /* 2️⃣  Deleted message ids (needed for BranchInfo healing). */
                const deletedMsgIds = Update.flatMap(u => u.input.messagesDelete ?? []);

                /* 3️⃣  Load branch info in one go. */
                const branchInfo = await BranchInfoLoader.load(updateIds, deletedMsgIds);

                /* 4️⃣  Which invites are bots the caller is allowed to auto-accept? */
                const inviteUserIds = Create.flatMap(c => c.input.invitesCreate?.map(i => i.userConnect) ?? []);
                const bots = inviteUserIds.length
                    ? (await DbProvider.get().user.findMany({
                        where: {
                            id: { in: inviteUserIds.map(BigInt) },
                            isBot: true,
                            OR: [{ isPrivate: false }, { invitedByUserId: BigInt(userData.id) }],
                        },
                        select: { id: true },
                    })).map(b => ({ id: b.id.toString() }))
                    : [];

                /* 5️⃣  Nothing else is needed for Chat pre-shape. */
                const embeddingNeedsUpdateMap = preShapeEmbeddableTranslatable({ 
                    Create: Create as any, 
                    Update: Update as any, 
                    objectType: "Chat", 
                }).embeddingNeedsUpdateMap;
                return { bots, branchInfo, embeddingNeedsUpdateMap };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as ChatPre;
                let messages = await shapeHelper({ relation: "messages", relTypes: ["Create"], isOneToOne: false, objectType: "ChatMessage", parentRelationshipName: "chat", data, ...rest });
                const treeInfo = MessageTree.buildOperations({ branchInfo: preData.branchInfo[data.id], ...messages });
                messages = treeInfo.prismaOps;

                return {
                    id: BigInt(data.id),
                    publicId: generatePublicId(),
                    openToAnyoneWithInvite: noNull(data.openToAnyoneWithInvite),
                    creator: { connect: { id: BigInt(rest.userData.id) } },
                    // Create invite for non-bots and not yourself
                    invites: {
                        create: data.invitesCreate?.filter((u) => !preData.bots.some(b => b.id === u.userConnect) && u.userConnect !== rest.userData.id).map((u) => ({
                            id: BigInt(u.id),
                            user: { connect: { id: BigInt(u.userConnect) } },
                            status: preData.bots.some((b) => b.id === u.userConnect) ? ChatInviteStatus.Accepted : ChatInviteStatus.Pending,
                            message: noNull(u.message),
                        })),
                    },
                    // Handle participants
                    participants: {
                        // Automatically accept bots, and add yourself
                        create: [
                            ...(preData.bots.map((u) => ({
                                id: BigInt(u.id),
                                user: { connect: { id: BigInt(u.id) } },
                            }))),
                            {
                                id: BigInt(rest.userData.id),
                                user: { connect: { id: BigInt(rest.userData.id) } },
                            },
                        ],
                    },
                    messages,
                    team: await shapeHelper({ relation: "team", relTypes: ["Connect"], isOneToOne: true, objectType: "Team", parentRelationshipName: "chats", data, ...rest }),
                    translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as ChatPre;
                let messages = await shapeHelper({ relation: "messages", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "ChatMessage", parentRelationshipName: "chat", data, ...rest });
                const treeInfo = MessageTree.buildOperations({ branchInfo: preData.branchInfo[data.id], ...messages });
                messages = treeInfo.prismaOps;

                return {
                    openToAnyoneWithInvite: noNull(data.openToAnyoneWithInvite),
                    // Handle invites
                    invites: {
                        create: data.invitesCreate?.filter((u) => !preData.bots.some(b => b.id === u.userConnect) && u.userConnect !== rest.userData.id).map((u) => ({
                            id: BigInt(u.id),
                            user: { connect: { id: BigInt(u.userConnect) } },
                            status: ChatInviteStatus.Pending,
                            message: noNull(u.message),
                        })),
                        update: data.invitesUpdate?.map((u) => ({
                            where: { id: BigInt(u.id) },
                            data: {
                                message: noNull(u.message),
                            },
                        })),
                        delete: data.invitesDelete?.map((id) => ({ id: BigInt(id) })),
                    },
                    // Handle participants
                    participants: {
                        // Automatically accept bots. You should already be a participant, so no need to add yourself
                        create: [
                            ...(preData.bots.map((u) => ({
                                id: BigInt(u.id),
                                user: { connect: { id: BigInt(u.id) } },
                            }))),
                        ],
                        delete: data.participantsDelete?.map((id) => ({ id: BigInt(id) })),
                    },
                    messages,
                    translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
        },
        trigger: {
            afterMutations: async ({ createdIds, updatedIds, deletedIds, userData: _userData }) => {
                // Standard trigger logic (notifications, etc.) should remain here.

                // Invalidate ConversationState cache for all affected chats
                const allAffectedChatIds = [
                    ...createdIds,
                    ...updatedIds,
                    ...deletedIds,
                ];

                for (const chatId of allAffectedChatIds) {
                    if (chatId) {
                        try {
                            await conversationStateStore.invalidateDistributed(chatId.toString());
                        } catch (error) {
                            console.error(`Failed to invalidate ConversationState cache for chat ${chatId}`, { error });
                        }
                    }
                }
            },
        },
        yup: chatValidation,
    },
    search: {
        defaultSort: ChatSortBy.DateUpdatedDesc,
        searchFields: {
            createdTimeFrame: true,
            creatorId: true,
            openToAnyoneWithInvite: true,
            teamId: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        sortBy: ChatSortBy,
        searchStringQuery: () => ({
            OR: [
                "transNameWrapped",
                "transDescriptionWrapped",
            ],
        }),
        supplemental: {
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<ChatModelInfo["ApiPermission"]>(__typename, ids, userData)),
                        // hasUnread: await ModelMap.get<ChatModelLogic>("Chat").query.getHasUnread(userData?.id, ids, __typename),
                    },
                };
            },
        },
    },
    validate: () => ({
        isDeleted: () => false,
        isPublic: (data, _getParentInfo?) => data?.openToAnyoneWithInvite === true,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, _userId) => ({
            Team: data?.team,
            User: data?.creator,
        }),
        permissionResolvers: ({ data, isAdmin, isDeleted, isLoggedIn, isPublic, userId }) => {
            const isInvited = validatePK(userId) && data.invites?.some((i) => i.userId.toString() === userId && i.status === ChatInviteStatus.Pending);
            const isParticipant = validatePK(userId) && data.participants?.some((p) => {
                const participantUserId = p.userId ?? (p as any).user?.id;
                return participantUserId?.toString() === userId;
            });
            return {
                ...defaultPermissions({ isAdmin, isDeleted, isLoggedIn, isPublic }),
                canInvite: () => isLoggedIn && isAdmin,
                canRead: () => !isDeleted && (isPublic || isAdmin || isInvited || isParticipant),
            };
        },
        permissionsSelect: (userId) => ({
            id: true,
            creator: "User",
            openToAnyoneWithInvite: true,
            ...(userId ? {
                participants: {
                    where: {
                        userId: BigInt(userId),
                    },
                    select: {
                        id: true,
                        user: {
                            select: {
                                id: true,
                            },
                        },
                    },
                },
                invites: {
                    where: {
                        userId: BigInt(userId),
                    },
                    select: {
                        id: true,
                        status: true,
                        user: {
                            select: {
                                id: true,
                            },
                        },
                    },
                },
            } : {}),
        }),
        visibility: {
            // For now, this also includes if you're a participant
            own: function getOwn(data) {
                return {
                    OR: [
                        { creator: { id: BigInt(data.userId) } },
                        { team: useVisibility("Team", "Own", data) },
                        {
                            participants: {
                                some: {
                                    user: {
                                        id: BigInt(data.userId),
                                    },
                                },
                            },
                        },
                    ],
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    OR: [
                        useVisibility("Chat", "Own", data),
                        useVisibility("Chat", "Public", data),
                    ],
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return {
                    isPrivate: true,
                    ...useVisibility("Chat", "Own", data),
                };
            },
            ownPublic: function getOwnPublic(data) {
                return {
                    isPrivate: false,
                    ...useVisibility("Chat", "Own", data),
                };
            },
            public: function getPublic() {
                return {
                    isPrivate: false,
                };
            },
        },
    }),
});
