import { ChatSortBy, chatValidation, MaxObjects, User, uuidValidate } from "@local/shared";
import { ChatInviteStatus } from "@prisma/client";
import { noNull, shapeHelper } from "../../builders";
import { bestTranslation, defaultPermissions, getEmbeddableString, labelShapeHelper, translationShapeHelper } from "../../utils";
import { preShapeEmbeddableTranslatable } from "../../utils/preShapeEmbeddableTranslatable";
import { getSingleTypePermissions } from "../../validators";
import { ChatFormat } from "../format/chat";
import { ModelLogic } from "../types";
import { ChatModelLogic } from "./types";

const __typename = "Chat" as const;
const suppFields = ["you"] as const;
export const ChatModel: ModelLogic<ChatModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.chat,
    display: {
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: ({ translations }, languages) => bestTranslation(translations, languages).name ?? "",
        },
        embed: {
            select: () => ({ id: true, translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, description: true } } }),
            get: ({ translations }, languages) => {
                const trans = bestTranslation(translations, languages);
                return getEmbeddableString({
                    description: trans.description,
                    name: trans.name,
                }, languages[0]);
            },
        },
    },
    format: ChatFormat,
    mutate: {
        shape: {
            pre: async ({ createList, updateList, prisma, userData }) => {
                // Find invited users. Any that are bots are automatically accepted.
                const invitedUsers = createList.reduce((acc, c) => [...acc, ...(c.invitesCreate?.map((i) => i.userConnect) ?? []) as string[]], [] as string[]);
                // Find all bots
                const bots = await prisma.user.findMany({ where: { id: { in: invitedUsers }, isBot: true } });
                // Find translations that need text embeddings
                const embeddingMaps = preShapeEmbeddableTranslatable({ createList, updateList, objectType: __typename });
                return { ...embeddingMaps, bots };
            },
            create: async ({ data, ...rest }) => ({
                id: data.id,
                openToAnyoneWithInvite: noNull(data.openToAnyoneWithInvite),
                // Create invite for non-bots
                invites: {
                    create: data.invitesCreate?.map((u) => ({
                        id: u.id,
                        user: { connect: { id: u.userConnect } },
                        status: rest.preMap[__typename].bots.some((b: User) => b.id === u.userConnect) ? ChatInviteStatus.Accepted : ChatInviteStatus.Pending,
                        message: noNull(u.message),
                    })),
                },
                // Handle participants
                participants: {
                    // Automatically accept bots, and add yourself
                    create: [
                        ...(rest.preMap[__typename].bots.map((u: User) => ({
                            user: { connect: { id: u.id } },
                        }))),
                        {
                            user: { connect: { id: rest.userData.id } },
                        },
                    ],
                },
                ...(await shapeHelper({ relation: "organization", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "Organization", parentRelationshipName: "chats", data, ...rest })),
                // ...(await shapeHelper({ relation: "restrictedToRoles", relTypes: ["Connect"], isOneToOne: false, isRequired: false, objectType: "Role", parentRelationshipName: "chats", data, ...rest })),
                ...(await labelShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Chat", relation: "labels", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                openToAnyoneWithInvite: noNull(data.openToAnyoneWithInvite),
                // Handle invites
                invites: {
                    create: data.invitesCreate?.map((u) => ({
                        id: u.id,
                        user: { connect: { id: u.userConnect } },
                        status: rest.preMap[__typename].bots.includes(u.userConnect) ? ChatInviteStatus.Accepted : ChatInviteStatus.Pending,
                        message: noNull(u.message),
                    })),
                    update: data.invitesUpdate?.map((u) => ({
                        where: { id: u.id },
                        data: {
                            message: noNull(u.message),
                        },
                    })),
                    delete: data.invitesDelete?.map((id) => ({ id })),
                },
                // Handle participants
                participants: {
                    // Automatically accept bots, and add yourself
                    create: [
                        ...(rest.preMap[__typename].bots.map((u: User) => ({
                            user: { connect: { id: u.id } },
                        }))),
                        {
                            user: { connect: { id: rest.userData.id } },
                        },
                    ],
                    delete: data.participantsDelete?.map((id) => ({ id })),
                },
                // ...(await shapeHelper({ relation: "restrictedToRoles", relTypes: ["Connect", "Disconnect"], isOneToOne: false, isRequired: false, objectType: "Role", parentRelationshipName: "chats", data, ...rest })),
                ...(await labelShapeHelper({ relTypes: ["Connect", "Create", "Delete", "Disconnect"], parentType: "Chat", relation: "labels", data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest })),
            }),
        },
        trigger: {
            onCreated: async ({ created, prisma, userData }) => {
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
            openToAnyoneWithInvite: true,
            labelsIds: true,
            organizationId: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        sortBy: ChatSortBy,
        searchStringQuery: () => ({
            OR: [
                "labelsWrapped",
                "tagsWrapped",
                "transNameWrapped",
                "transDescriptionWrapped",
            ],
        }),
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        // hasUnread: await ChatModel.query.getHasUnread(prisma, userData?.id, ids, __typename),
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
            Organization: data.organization,
            User: data.creator,
        }),
        permissionResolvers: ({ data, isAdmin, isDeleted, isLoggedIn, isPublic, userId }) => {
            const isInvited = uuidValidate(userId) && data.invites?.some((i) => i.userId === userId && i.status === ChatInviteStatus.Pending);
            const isParticipant = uuidValidate(userId) && data.participants?.some((p) => p.userId === userId);
            return {
                ...defaultPermissions({ isAdmin, isDeleted, isLoggedIn, isPublic }),
                canInvite: () => isLoggedIn && isAdmin,
                canRead: () => !isDeleted && (isPublic || isAdmin || isInvited || isParticipant),
            };
        },
        permissionsSelect: (userId) => ({
            id: true,
            creator: "User",
            ...(userId ? {
                participants: {
                    where: {
                        userId,
                    },
                    select: {
                        id: true,
                    },
                },
                invites: {
                    where: {
                        userId,
                    },
                    select: {
                        id: true,
                        status: true,
                    },
                },
            } : {}),
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                creator: { id: userId },
            }),
        },
    },
});
