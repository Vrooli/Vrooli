import { ChatInviteStatus, ChatSortBy, chatValidation, MaxObjects, User, uuidValidate } from "@local/shared";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { bestTranslation, defaultPermissions, getEmbeddableString } from "../../utils";
import { labelShapeHelper, preShapeEmbeddableTranslatable, translationShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions } from "../../validators";
import { ChatFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { ChatModelLogic } from "./types";

const __typename = "Chat" as const;
export const ChatModel: ChatModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.chat,
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
            get: ({ translations }, languages) => bestTranslation(translations, languages)?.name ?? "",
        },
        embed: {
            select: () => ({ id: true, translations: { select: { id: true, embeddingNeedsUpdate: true, language: true, name: true, description: true } } }),
            get: ({ translations }, languages) => {
                const trans = bestTranslation(translations, languages);
                return getEmbeddableString({
                    description: trans?.description,
                    name: trans?.name,
                }, languages[0]);
            },
        },
    }),
    format: ChatFormat,
    mutate: {
        shape: {
            pre: async ({ Create, Update, prisma, userData, inputsById }) => {
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
                const bots = await prisma.user.findMany({
                    where: {
                        id: { in: invitedUsers },
                        isBot: true,
                        OR: [
                            { isPrivate: false }, // Public bots
                            { invitedByUser: { id: userData.id } }, // Private bots you created
                        ],
                    },
                });
                // Find translations that need text embeddings
                const embeddingMaps = preShapeEmbeddableTranslatable<"id">({ Create, Update, objectType: __typename });
                return { ...embeddingMaps, bots };
            },
            create: async ({ data, ...rest }) => {
                // Due to the way Prisma handles connections, we must be careful with how messages are inserted. 
                // When a message has a "parentConnect" relation pointing to a new message (i.e. is also in the "create" array), 
                // we must convert it to a "parentCreate", and remove the parent message from the array. Otherwise, Prisma will 
                // throw an error about not being able to find the parent message.
                let messages = await shapeHelper({ relation: "messages", relTypes: ["Create"], isOneToOne: false, objectType: "ChatMessage", parentRelationshipName: "chat", data, ...rest });
                if (messages && Object.prototype.hasOwnProperty.call(messages, "messages") && Array.isArray(messages.create)) {
                    let newMessages: unknown[] = [];
                    const parentIdsToRemove: string[] = [];
                    for (const message of messages.create) {
                        if (!Object.prototype.hasOwnProperty.call(message, "parent") || !Object.prototype.hasOwnProperty.call(message.parent, "connect")) {
                            newMessages.push(message);
                            continue;
                        }
                        // Check if the parent message is in the create array
                        const parentMessage = messages.create.find((m) => m.id === message.parent.connect.id);
                        // If it is, convert the parentConnect to a parentCreate
                        if (parentMessage) {
                            newMessages.push({
                                ...message,
                                parent: {
                                    create: {
                                        ...parentMessage,
                                        chat: { connect: { id: data.id } },
                                    },
                                },
                            });
                            parentIdsToRemove.push(parentMessage.id);
                            continue;
                        }
                        // Otherwise, leave it as a parentConnect
                        newMessages.push(message);
                    }
                    // Remove the parent messages from the create array
                    newMessages = newMessages.filter((m) => !parentIdsToRemove.includes((m as { id: string }).id));
                    // Replace the old messages with the new ones
                    messages = { ...messages, create: newMessages };
                }
                return {
                    id: data.id,
                    openToAnyoneWithInvite: noNull(data.openToAnyoneWithInvite),
                    creator: { connect: { id: rest.userData.id } },
                    // Create invite for non-bots and not yourself
                    invites: {
                        create: data.invitesCreate?.filter((u) => !rest.preMap[__typename].bots.includes(u.userConnect) && u.userConnect !== rest.userData.id).map((u) => ({
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
                    labels: await labelShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Chat", data, ...rest }),
                    messages,
                    organization: await shapeHelper({ relation: "organization", relTypes: ["Connect"], isOneToOne: true, objectType: "Organization", parentRelationshipName: "chats", data, ...rest }),
                    restrictedToRoles: await shapeHelper({
                        relation: "restrictedToRoles", relTypes: ["Connect"], isOneToOne: false, objectType: "Role", parentRelationshipName: "", joinData: {
                            fieldName: "role",
                            uniqueFieldName: "chat_roles_chatid_roleid_unique",
                            childIdFieldName: "roleId",
                            parentIdFieldName: "chatId",
                            parentId: data.id ?? null,
                        }, data, ...rest,
                    }),
                    translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => ({
                openToAnyoneWithInvite: noNull(data.openToAnyoneWithInvite),
                // Handle invites
                invites: {
                    create: data.invitesCreate?.filter((u) => !rest.preMap[__typename].bots.includes(u.userConnect) && u.userConnect !== rest.userData.id).map((u) => ({
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
                    // Automatically accept bots. You should already be a participant, so no need to add yourself
                    create: [
                        ...(rest.preMap[__typename].bots.map((u: User) => ({
                            user: { connect: { id: u.id } },
                        }))),
                    ],
                    delete: data.participantsDelete?.map((id) => ({ id })),
                },
                labels: await labelShapeHelper({ relTypes: ["Connect", "Create", "Delete", "Disconnect"], parentType: "Chat", data, ...rest }),
                messages: await shapeHelper({ relation: "messages", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "ChatMessage", parentRelationshipName: "chat", data, ...rest }),
                restrictedToRoles: await shapeHelper({
                    relation: "restrictedToRoles", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "Role", parentRelationshipName: "", joinData: {
                        fieldName: "role",
                        uniqueFieldName: "chat_roles_chatid_roleid_unique",
                        childIdFieldName: "roleId",
                        parentIdFieldName: "chatId",
                        parentId: data.id ?? null,
                    }, data, ...rest,
                }),
                translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], embeddingNeedsUpdate: rest.preMap[__typename].embeddingNeedsUpdateMap[data.id], data, ...rest }),
            }),
        },
        trigger: {
            afterMutations: async ({ createdIds, prisma, userData }) => {
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
            labelsIds: true,
            organizationId: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        sortBy: ChatSortBy,
        searchStringQuery: () => ({
            OR: [
                "labelsWrapped",
                "transNameWrapped",
                "transDescriptionWrapped",
            ],
        }),
        supplemental: {
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        // hasUnread: await ModelMap.get<ChatModelLogic>("Chat").query.getHasUnread(prisma, userData?.id, ids, __typename),
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
            Organization: data?.organization,
            User: data?.creator,
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
    }),
});
