import { ChatInviteStatus, ChatSortBy, chatValidation, getTranslation, MaxObjects, uuidValidate } from "@local/shared";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { defaultPermissions, getEmbeddableString } from "../../utils";
import { labelShapeHelper, preShapeEmbeddableTranslatable, PreShapeEmbeddableTranslatableResult, translationShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions } from "../../validators";
import { ChatFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { ChatModelInfo, ChatModelLogic } from "./types";

export type ChatPreBranchInfo = {
    /** The message most recently created (i.e. has the highest sequence number), or null if no messages */
    lastSequenceId: string | null,
    /** 
     * Map of message IDs to their parent ID and list of child IDs. 
     * This data should be provided for each message that's being deleted, so that we 
     * can heal the branch structure around the deleted messages.
     */
    messageTreeInfo: Record<string, { parentId: string | null; childIds: string[] }>;
};
type ChatPre = PreShapeEmbeddableTranslatableResult & {
    /** Invited bots which you have the permission to invite */
    bots: { id: string }[];
    /** Map of chat IDs to branchable status and the last message */
    branchInfo: Record<string, ChatPreBranchInfo>;
};

type ChatMessageCreate = {
    id: string;
    parent?: {
        connect: {
            id: string;
        };
    };
    // Include other fields necessary for creating a message
    [key: string]: any;
}
type ChatMessageUpdate = Omit<ChatMessageCreate, "parent">;
type ChatMessageDelete = {
    id: string;
}
export type ChatMessageOperations = {
    create?: ChatMessageCreate[] | null;
    update?: ChatMessageUpdate[] | null;
    delete?: ChatMessageDelete[] | null;
}
export type ChatMessageCreateResult = Omit<ChatMessageCreate, "parent"> & {
    parent?: {
        create: ChatMessageCreateResult
    } | {
        connect: {
            id: string;
        }
    };
};
export type PrismaChatMutationResult = {
    // The `create` field is a single object with all messages nested
    create?: ChatMessageCreateResult;
    // Each update can now update its parent reference or remove it
    update?: (ChatMessageUpdate & { parent?: ({ connect: { id: string } } | { disconnect: true }) })[];
    // Deletes are unchanged
    delete?: ChatMessageDelete[];
} | undefined;

/**
 * Generates a safe message structure for a Prisma chat mutation. 
 * This function restructures the 'create' field to ensure that messages are properly linked,
 * preserving the order and hierarchy dictated by the chat's branching logic, while avoiding 
 * references to uncreated messages. It leaves the 'delete' field unchanged and optionally 
 * adds or modifes the 'update' field to adjust the chat structure around deleted messages.
 * 
 * @param operations The proposed operations for Prisma, typically including 'create' and 'delete'.
 *                   This function modifies 'create' to nest messages according to their parent-child
 *                   relationships, adjusts 'update' to handle relational changes due to deletions,
 *                   and maintains 'delete' as provided.
 * @param branchInfo Information on the chat's branching capability and the identifier of the last
 *                   message in the sequence, used to anchor new messages if necessary.
 * @returns An object structured for direct use in a Prisma mutation, with nested 'create',
 *          unmodified 'delete', and newly added 'update' operations to maintain chat integrity.
 */
export function prepareChatMessageOperations(
    operations: ChatMessageOperations,
    branchInfo: ChatPreBranchInfo,
): PrismaChatMutationResult {
    const createOperations = operations.create || [];
    const updateOperations = operations.update || [];
    const deleteOperations = operations.delete || [];

    // Map of message IDs to their create data
    const createMessageMap = new Map<string, ChatMessageCreate>();
    createOperations.forEach((msg) => {
        createMessageMap.set(msg.id, msg);
    });

    // Detect branching in the new messages (i.e. a message with multiple children), 
    // which could cause issues with Prisma.
    // 
    // In practice there should never be branching, since if multiple people are trying 
    // to update a chat at the same time, it's not a branchable chat. And if it's one person sending 
    // multiple messages, they wouldn't be on different branches.
    const messagesWithChild = new Set<string>();
    for (const msg of createOperations) {
        const parentId = msg.parent?.connect?.id;
        if (parentId && createMessageMap.has(parentId)) {
            if (messagesWithChild.has(parentId)) {
                throw new Error("Cannot create nested messages in a branching chat. All messages must be sequential.");
            }
            messagesWithChild.add(parentId);
        }
    }

    /**
     * Function to recursively build nested creates from a list of messages 
     * ordered from root to leaf.
     * 
     * Result should be a nested structure where the leaf is at the top, and 
     * the root is the most deeply nested object.
     */
    function buildNestedCreate(orderedMessages: ChatMessageCreate[]): ChatMessageCreateResult {
        let nestedCreate: ChatMessageCreateResult | undefined;
        // Loop backwards through the ordered messages, building the nested structure
        for (let i = orderedMessages.length - 1; i >= 0; i--) {
            const msg = orderedMessages[i];
            // Use orderedMessages[i + 1] as the parent if it exists
            const newParentId = i > orderedMessages.length - 2 ? null : orderedMessages[i + 1].id;
            if (newParentId) {
                nestedCreate = { ...msg, parent: { create: nestedCreate! } };
            }
            // At this point, we're working with the root message
            // Use the root's connect if it's not to a message being created
            else if (msg.parent?.connect?.id && !createMessageMap.has(msg.parent.connect.id)) {
                nestedCreate = { ...msg, parent: { connect: { id: msg.parent.connect.id } } };
            }
            // Use the lastSequenceId if it exists
            else if (branchInfo.lastSequenceId) {
                nestedCreate = { ...msg, parent: { connect: { id: branchInfo.lastSequenceId } } };
            }
            // Otherwise, just create the message
            else {
                nestedCreate = { ...msg };
            }
        }
        return nestedCreate!;
    }

    // Build the nested create structure for the root message
    let nestedCreate: ChatMessageCreateResult | undefined;
    if (createMessageMap.size > 0) {
        // Find messages with specified parents
        const parentIds: Set<string> = new Set();
        const messagesWithParents = createOperations.filter((msg) => {
            const parentId = msg.parent?.connect?.id;
            if (parentId) {
                parentIds.add(parentId);
                return true;
            }
            return false;
        });
        // Sort messages from leaf to root
        messagesWithParents.sort((a, b) => {
            const aParentId = a.parent?.connect?.id;
            const bParentId = b.parent?.connect?.id;
            if (aParentId === b.id) {
                return -1;
            }
            if (bParentId === a.id) {
                return 1;
            }
            return 0;
        });
        // Make sure that every connected parent is a new message, except at the root
        const isEveryParentNew = messagesWithParents.every((msg, i) => {
            // Ignore the root, which is the last message in the sequence
            if (i === messagesWithParents.length - 1) {
                return true;
            }
            const parentId = msg.parent?.connect?.id;
            return parentId && createMessageMap.has(parentId);
        });
        if (!isEveryParentNew) {
            throw new CustomError("0416", "InternalError", ["en"], { msg: "Cannot create nested messages in a branching chat. All messages must be sequential." });
        }

        // Find messages without specified parents
        const messagesWithoutParents = createOperations.filter((msg) => {
            const parentId = msg.parent?.connect?.id;
            return !parentId;
        });
        // Sort messages without parents so that any in `parentIds` are at the end
        messagesWithoutParents.sort((a, b) => {
            const isAInParentIds = parentIds.has(a.id);
            const isBInParentIds = parentIds.has(b.id);
            if (isAInParentIds && !isBInParentIds) {
                return 1;
            }
            if (!isAInParentIds && isBInParentIds) {
                return -1;
            }
            return 0;
        }).reverse();

        // Combine the two sets of messages
        const allMessages = [...messagesWithParents, ...messagesWithoutParents];

        // Now every message (except maybe the first) should be connected to a parent. 
        // To allow Prisma to create them all in one go, we need to convert this to a nested structure.
        nestedCreate = buildNestedCreate(allMessages);
    }

    // Now handle delete operations and adjust update operations accordingly

    // Create a Set of deleted message IDs for quick lookup
    const deletedMessageIds = new Set<string>(deleteOperations.map((delOp) => delOp.id));

    // Collect additional update operations needed to reparent messages
    const additionalUpdateOperations: ChatMessageUpdate[] = [];

    // Function to find the nearest non-deleted ancestor
    function findNearestNonDeletedAncestor(startingParentId: string | null): string | null {
        let currentParentId = startingParentId;
        while (currentParentId) {
            if (!deletedMessageIds.has(currentParentId)) {
                return currentParentId;
            }
            const parentTreeInfo = branchInfo.messageTreeInfo[currentParentId];
            if (!parentTreeInfo) {
                // We don't have further parent info, assume null
                return null;
            }
            currentParentId = parentTreeInfo.parentId;
        }
        return null;
    }

    // Process each deleted message
    deleteOperations.forEach((deleteOp) => {
        const messageId = deleteOp.id;
        const treeInfo = branchInfo.messageTreeInfo[messageId];

        if (!treeInfo) {
            throw new Error(`Missing message tree info for deleted message ID: ${messageId}`);
        }

        const childIds = treeInfo.childIds || [];

        childIds.forEach((childId) => {
            if (deletedMessageIds.has(childId)) {
                // The child is also being deleted, so no need to update it
                return;
            }

            // Find the new parent ID for the child
            const newParentId = findNearestNonDeletedAncestor(treeInfo.parentId);

            // Find existing update operation that matches the child ID
            const existingUpdateOpIndex = updateOperations.findIndex((op) => op.id === childId);
            // Create or update the update operation
            const updateOp: ChatMessageUpdate = {
                ...(existingUpdateOpIndex >= 0 ? updateOperations[existingUpdateOpIndex] : {}),
                id: childId,
                parent: newParentId ? { connect: { id: newParentId } } : { disconnect: true },
            };
            // Add or replace the update operation
            if (existingUpdateOpIndex >= 0) {
                updateOperations[existingUpdateOpIndex] = updateOp;
            } else {
                additionalUpdateOperations.push(updateOp);
            }
        });
    });

    // Merge additionalUpdateOperations into updateOperations
    const allUpdateOperations = updateOperations.concat(additionalUpdateOperations);

    const messages = {
        create: nestedCreate,
        update: allUpdateOperations.length > 0 ? allUpdateOperations : undefined,
        delete: deleteOperations.length > 0 ? deleteOperations : undefined,
    };
    // Removed undefined and empty arrays
    const filteredMessages = Object.fromEntries(Object.entries(messages).filter(([_, v]) => v !== undefined && (Array.isArray(v) ? v.length > 0 : true)));
    // If the result is an empty object, return undefined
    return Object.keys(filteredMessages).length ? filteredMessages : undefined;
}

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
                }, languages[0]);
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
                    bots = await prismaInstance.user.findMany({
                        where: {
                            id: { in: invitedUsers },
                            isBot: true,
                            OR: [
                                { isPrivate: false }, // Public bots
                                { invitedByUser: { id: userData.id } }, // Private bots you created
                            ],
                        },
                        select: { id: true },
                    });
                }
                // Find information needed to link new chat messages to their parent messages
                let branchInfo: ChatPre["branchInfo"] = {};
                // We only care about parent messages for updates, since new chats won't have any messages yet, 
                // and thus no existing message tree to link to.
                // We also only care about messageTreeInfo for updates, since new chats can't delete messages.
                if (Update.length) {
                    const chatIds = Update.map(u => u.input.id);
                    // Fetch chat information
                    const chats = await prismaInstance.chat.findMany({
                        where: { id: { in: chatIds } },
                        select: {
                            id: true,
                            messages: {
                                orderBy: { sequence: "desc" },
                                take: 1,
                                select: { id: true },
                            },
                        },
                    });
                    // Calculate branchability
                    branchInfo = chats.reduce((acc, chat) => {
                        acc[chat.id] = {
                            lastSequenceId: chat.messages.length ? chat.messages[0].id : null,
                            messageTreeInfo: {},
                        };
                        return acc;
                    }, {} as ChatPre["branchInfo"]);

                    // Collect all message IDs that are being deleted
                    const deletedMessageIds = Update.reduce((acc, update) => {
                        const deleteIds = update.input.messagesDelete ?? [];
                        acc.push(...deleteIds);
                        return acc;
                    }, [] as string[]);
                    // If there are any messages being deleted, fetch each message's parent ID and child IDs
                    if (deletedMessageIds.length > 0) {
                        const messages = await prismaInstance.chat_message.findMany({
                            where: { id: { in: deletedMessageIds } },
                            select: {
                                id: true,
                                chat: { select: { id: true } },
                                parent: { select: { id: true } },
                                children: { select: { id: true } },
                            },
                        });
                        // Populate messageTreeInfo
                        messages.forEach((msg) => {
                            if (!msg.chat) {
                                return;
                            }
                            const chatId = msg.chat.id;
                            if (!branchInfo[chatId]) {
                                branchInfo[msg.id] = {
                                    lastSequenceId: null,
                                    messageTreeInfo: {},
                                };
                            }
                            branchInfo[chatId].messageTreeInfo[msg.id] = {
                                parentId: msg.parent?.id ?? null,
                                childIds: msg.children.map(c => c.id),
                            };
                        });
                    }
                }
                // Find translations that need text embeddings
                const embeddingMaps = preShapeEmbeddableTranslatable<"id">({ Create, Update, objectType: __typename });
                return { ...embeddingMaps, bots, branchInfo };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as ChatPre;
                let messages = await shapeHelper({ relation: "messages", relTypes: ["Create"], isOneToOne: false, objectType: "ChatMessage", parentRelationshipName: "chat", data, ...rest });
                messages = prepareChatMessageOperations(messages, preData.branchInfo[data.id] ?? {}) as any;

                return {
                    id: data.id,
                    openToAnyoneWithInvite: noNull(data.openToAnyoneWithInvite),
                    creator: { connect: { id: rest.userData.id } },
                    // Create invite for non-bots and not yourself
                    invites: {
                        create: data.invitesCreate?.filter((u) => !preData.bots.some(b => b.id === u.userConnect) && u.userConnect !== rest.userData.id).map((u) => ({
                            id: u.id,
                            user: { connect: { id: u.userConnect } },
                            status: preData.bots.some((b) => b.id === u.userConnect) ? ChatInviteStatus.Accepted : ChatInviteStatus.Pending,
                            message: noNull(u.message),
                        })),
                    },
                    // Handle participants
                    participants: {
                        // Automatically accept bots, and add yourself
                        create: [
                            ...(preData.bots.map((u) => ({
                                user: { connect: { id: u.id } },
                            }))),
                            {
                                user: { connect: { id: rest.userData.id } },
                            },
                        ],
                    },
                    labels: await labelShapeHelper({ relTypes: ["Connect", "Create"], parentType: "Chat", data, ...rest }),
                    messages,
                    team: await shapeHelper({ relation: "team", relTypes: ["Connect"], isOneToOne: true, objectType: "Team", parentRelationshipName: "chats", data, ...rest }),
                    restrictedToRoles: await shapeHelper({
                        relation: "restrictedToRoles", relTypes: ["Connect"], isOneToOne: false, objectType: "Role", parentRelationshipName: "", joinData: {
                            fieldName: "role",
                            uniqueFieldName: "chat_roles_chatid_roleid_unique",
                            childIdFieldName: "roleId",
                            parentIdFieldName: "chatId",
                            parentId: data.id ?? null,
                        }, data, ...rest,
                    }),
                    translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as ChatPre;
                let messages = await shapeHelper({ relation: "messages", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "ChatMessage", parentRelationshipName: "chat", data, ...rest });
                messages = prepareChatMessageOperations(messages, preData.branchInfo[data.id] ?? {}) as any;

                return {
                    openToAnyoneWithInvite: noNull(data.openToAnyoneWithInvite),
                    // Handle invites
                    invites: {
                        create: data.invitesCreate?.filter((u) => !preData.bots.some(b => b.id === u.userConnect) && u.userConnect !== rest.userData.id).map((u) => ({
                            id: u.id,
                            user: { connect: { id: u.userConnect } },
                            status: ChatInviteStatus.Pending,
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
                            ...(preData.bots.map((u) => ({
                                user: { connect: { id: u.id } },
                            }))),
                        ],
                        delete: data.participantsDelete?.map((id) => ({ id })),
                    },
                    labels: await labelShapeHelper({ relTypes: ["Connect", "Create", "Delete", "Disconnect"], parentType: "Chat", data, ...rest }),
                    messages,
                    restrictedToRoles: await shapeHelper({
                        relation: "restrictedToRoles", relTypes: ["Connect", "Disconnect"], isOneToOne: false, objectType: "Role", parentRelationshipName: "", joinData: {
                            fieldName: "role",
                            uniqueFieldName: "chat_roles_chatid_roleid_unique",
                            childIdFieldName: "roleId",
                            parentIdFieldName: "chatId",
                            parentId: data.id ?? null,
                        }, data, ...rest,
                    }),
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
            labelsIds: true,
            teamId: true,
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
            toGraphQL: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<ChatModelInfo["GqlPermission"]>(__typename, ids, userData)),
                        // hasUnread: await ModelMap.get<ChatModelLogic>("Chat").query.getHasUnread(userData?.id, ids, __typename),
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
            Team: data?.team,
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
            private: function getVisibilityPrivate() {
                return {};
            },
            public: function getVisibilityPublic() {
                return {};
            },
            owner: (userId) => ({
                creator: { id: userId },
            }),
        },
    }),
});
