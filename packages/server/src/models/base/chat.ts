import { ChatInviteStatus, ChatSortBy, chatValidation, generatePublicId, getTranslation, MaxObjects } from "@local/shared";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { DbProvider } from "../../db/provider.js";
import { ChatPre, populatePreMapForChatUpdates, prepareChatMessageOperations } from "../../utils/chat.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { getEmbeddableString } from "../../utils/embeddings/getEmbeddableString.js";
import { preShapeEmbeddableTranslatable } from "../../utils/shapes/preShapeEmbeddableTranslatable.js";
import { translationShapeHelper } from "../../utils/shapes/translationShapeHelper.js";
import { getSingleTypePermissions } from "../../validators/permissions.js";
import { ChatFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ChatModelInfo, ChatModelLogic } from "./types.js";

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
        embed: {
            select: () => ({ id: true, translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, description: true } } }),
            get: ({ translations }, languages) => {
                const trans = getTranslation({ translations }, languages);
                return getEmbeddableString({
                    description: trans.description,
                    name: trans.name,
                }, languages?.[0]);
            },
        },
    }),
    format: ChatFormat,
    mutate: {
        shape: {
            pre: async ({ Create, Update, userData, inputsById }): Promise<ChatPre> => {
                // Find invited users. Any that are bots are automatically accepted.
                const invitedUsers = Create.reduce((acc, createObject) => {
                    const invites = createObject.input.invitesCreate ?? [];
                    invites.forEach(invite => {
                        if (typeof invite === "string") {
                            // If invite is a string, find the corresponding object in `inputsById` and extract `userConnect`
                            const inviteObject: { input?: { userConnect?: string } } = inputsById[invite] as object;
                            if (inviteObject && inviteObject.input && inviteObject.input.userConnect) {
                                acc.push(inviteObject.input.userConnect);
                            }
                        } else if (invite && typeof invite === "object" && invite.userConnect) {
                            // If invite is an object, use `userConnect` directly
                            acc.push(invite.userConnect);
                        }
                    });
                    return acc;
                }, [] as string[]);
                // Find all bots
                let bots: ChatPre["bots"] = [];
                if (invitedUsers.length) {
                    const botData = await DbProvider.get().user.findMany({
                        where: {
                            id: { in: invitedUsers.map(id => BigInt(id)) },
                            isBot: true,
                            OR: [
                                { isPrivate: false }, // Public bots
                                { invitedByUser: { id: BigInt(userData.id) } }, // Private bots you created
                            ],
                        },
                        select: { id: true },
                    });
                    bots = botData.map(b => ({ id: b.id.toString() }));
                }
                const updateInputs = Update.map(u => u.input);
                const { branchInfo } = await populatePreMapForChatUpdates({ updateInputs });
                // Find translations that need text embeddings
                const embeddingMaps = preShapeEmbeddableTranslatable<"id">({ Create, Update, objectType: __typename });
                return { ...embeddingMaps, bots, branchInfo };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as ChatPre;
                let messages = await shapeHelper({ relation: "messages", relTypes: ["Create"], isOneToOne: false, objectType: "ChatMessage", parentRelationshipName: "chat", data, ...rest });
                messages = prepareChatMessageOperations(messages, preData.branchInfo[data.id]).operations;

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
                                user: { connect: { id: BigInt(u.id) } },
                            }))),
                            {
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
                messages = prepareChatMessageOperations(messages, preData.branchInfo[data.id] ?? {}).operations;

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
            afterMutations: async ({ createdIds, userData }) => {
                //TODO If starting a chat with a bot (not Valyxa, since we create an initial message in the 
                // UI for speed), allow the bot to send a message to the chat
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
        isPublic: (data) => data?.openToAnyoneWithInvite === true,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            Team: data?.team,
            User: data?.creator,
        }),
        permissionResolvers: ({ data, isAdmin, isDeleted, isLoggedIn, isPublic, userId }) => {
            const isInvited = uuidValidate(userId) && data.invites?.some((i) => i.userId === userId && i.status === ChatInviteStatus.Pending);
            const isParticipant = uuidValidate(userId) && data.participants?.some((p) => p.user?.id === userId);
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
