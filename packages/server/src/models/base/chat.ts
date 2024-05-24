import { ChatInviteStatus, ChatSortBy, chatValidation, MaxObjects, uuidValidate } from "@local/shared";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { prismaInstance } from "../../db/instance";
import { bestTranslation, defaultPermissions, getEmbeddableString } from "../../utils";
import { labelShapeHelper, preShapeEmbeddableTranslatable, PreShapeEmbeddableTranslatableResult, translationShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions } from "../../validators";
import { ChatFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { ChatModelLogic } from "./types";

type ChatPreBranchInfo = {
    /** True if the chat can be branched. False if messages must all be sequential */
    isBranchable: boolean,
    /** The message most recently created (i.e. has the highest sequence number), or null if no messages */
    lastSequenceId: string | null,
};
type ChatPre = PreShapeEmbeddableTranslatableResult & {
    /** Invited bots which you have the permission to invite */
    bots: { id: string }[];
    /** Map of chat IDs to branchable status and the last message */
    branchInfo: Record<string, ChatPreBranchInfo>;
};

/**
 * Generate nested chat message structure for a Prisma chat mutation. 
 * This is necessary to ensure that messages are linked together properly, 
 * while not using "connect" on messages that haven't been created yet.
 * @param messages GraphQL mutation for each message
 * @param branchInfo Information about the chat's branching status, which is needed to 
 * link the final parent in the nested create to an existing message.
 * @returns Nested chat message structure for Prisma
 */
export const chatCreateNestedMessages = (
    messages: any[],
    branchInfo: ChatPreBranchInfo,
) => {
    if (messages.length === 0) {
        return undefined;
    }

    const createObjects: Record<string, any>[] = [];
    // Store all visited message IDs
    const allVisited: Set<string> = new Set();
    // Store message IDs visited in current branch, in the order they were visited
    let currVisited: string[] = [];
    // Map message indexes by ID for easy lookup
    const messageMap: Map<string, number> = new Map(messages.map((msg, i) => [msg.id, i]));
    // Track number of times a loop has been detected (which restarts the process at the current message) 
    // to prevent infinite loops
    let restarts = 0;
    // Store current message index
    let currMessageIndex = messages.length - 1; // Start at the end of the array, as we'll be working backwards

    const canVisitMore = () => allVisited.size + currVisited.length < messages.length;
    const hasBeenVisited = (id: string | undefined) => id !== undefined && (allVisited.has(id) || currVisited.includes(id));

    // Recursive helper function to create nested message structure
    const createNested = () => {
        const message = messages[currMessageIndex];
        // Add message to visited set
        if (hasBeenVisited(message.id)) {
            throw new Error("Message visited twice - Flaw in logic");
        }
        currVisited.push(message.id);

        // Determine parent message
        // If message has a parent, use that
        let parentId: string | undefined = message.parent?.connect?.id;
        // If this is the last message to be visited, we can use lastSequenceId if it exists
        if (!parentId && !canVisitMore() && branchInfo.lastSequenceId) {
            parentId = branchInfo.lastSequenceId;
        }
        // Otherwise, use earlier message in array that hasn't been visited. 
        // For example, if we're at message 4 and message 3 was visited, use message 2.
        if (!parentId && canVisitMore()) {
            for (let i = 1; i <= messages.length; i++) {
                const index = (currMessageIndex - i + messages.length) % messages.length;
                if (!allVisited.has(messages[index].id) && !currVisited.includes(messages[index].id)) {
                    console.log("had no parent id. Looped backwards for id", messages[index].id);
                    parentId = messages[index].id;
                    break;
                }
            }
        }

        // If the parentId exists, we need to make sure we haven't visited it yet. 
        // This would indicate a loop, which requires us to restart the process at the current message
        if (hasBeenVisited(parentId)) {
            restarts++;
            // If we've restarted as many times as there are messages, we're in an infinite loop
            if (restarts >= messages.length) {
                throw new Error("Loop detected");
            }
            // Otherwise, reset and try again
            currVisited = [];
            createNested();
        }
        // If we don't have a parentId, or if the parentId does not exist in the array, that's okay. 
        // It just means we're at the end of the current branch
        const atEndOfBranch = !parentId || !messageMap.has(parentId) || !canVisitMore();
        // If we're at the end of the branch, create the nested structure and add it to the create objects
        if (atEndOfBranch) {
            // Build the nested structure from currVisited array
            let nestedCreate: Record<string, any> | null = null;
            // If there is a parent ID, we'll connect to it
            if (parentId) {
                nestedCreate = { connect: { id: parentId } };
            }
            // We reverse because we are nesting from last to first as parent to child
            for (let i = currVisited.length - 1; i >= 0; i--) {
                const id = currVisited[i];
                const msg = messages[messageMap.get(id)!];
                if (nestedCreate) {
                    nestedCreate = { create: { ...msg, parent: nestedCreate } };
                } else {
                    nestedCreate = { create: { ...msg } };
                }
            }
            if (nestedCreate !== null && nestedCreate.create) {
                createObjects.push(nestedCreate.create);
            }

            // Add current visited messages to allVisited
            currVisited.forEach(id => allVisited.add(id));
            // Clear current visited messages
            currVisited = [];

            // If there are still messages left to visit (i.e. other branches), move the current index to the previous 
            // message which hasn't been visited
            if (canVisitMore()) {
                for (let i = 1; i <= messages.length; i++) {
                    const index = (currMessageIndex - i + messages.length) % messages.length;
                    if (!hasBeenVisited(messages[index].id)) {
                        currMessageIndex = index;
                        break;
                    }
                }
            }
        }
        // Otherwise, recurse
        else {
            // Move to the parent index
            const parentIndex = messageMap.get(parentId as string)!;
            currMessageIndex = parentIndex;
            // Recursive
            createNested();
        }
    };

    // Call the recursive function until all messages have been visited
    while (canVisitMore()) {
        createNested();
    }

    // If no create objects were generated, return undefined
    if (!createObjects.length) {
        return undefined;
    }
    // If only one create object was generated, return it directly
    if (createObjects.length === 1) {
        return { create: createObjects[0] };
    }
    // Otherwise, return the array of create objects
    return { create: createObjects };
};

const __typename = "Chat" as const;
export const ChatModel: ChatModelLogic = ({
    __typename,
    dbTable: "chat",
    dbTranslationTable: "chat_translation",
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
                // Find information needed to link new chat messages to their parent messages. 
                // This is not trivial, as chats can sometimes have branching conversations. 
                // Here are the rules:
                // 1. If the chat has or will have 2 participants (where 1 is you and another is a bot), 
                // then the chat can branch (think ChatGPT response regenerations). In this case, we must 
                // rely on the parent ID being provided in the input.
                // 2. Otherwise, we'll link messages sequentially by sequence number. This means we can simply 
                // find the highest sequence number in the chat and use that as the parent.
                let branchInfo: ChatPre["branchInfo"] = {};
                // We only care about parent messages for updates, since new chats won't have any messages yet
                if (Update.length) {
                    const chatIds = Update.map(u => u.input.id);
                    // Fetch chat information
                    const chats = await prismaInstance.chat.findMany({
                        where: { id: { in: chatIds } },
                        select: {
                            id: true,
                            participants: {
                                select: {
                                    id: true,
                                    user: {
                                        select: {
                                            id: true,
                                            isBot: true,
                                        },
                                    },
                                },
                                take: 2,
                            },
                            messages: {
                                orderBy: { sequence: "desc" },
                                take: 1,
                                select: { id: true },
                            },
                        },
                    });
                    // Calculate branchability
                    branchInfo = chats.reduce((acc, chat) => {
                        const isBranchable = chat.participants.length === 2 && chat.participants.some(p => p.user.isBot);
                        acc[chat.id] = {
                            isBranchable,
                            lastSequenceId: chat.messages.length ? chat.messages[0].id : null,
                        };
                        return acc;
                    }, {} as ChatPre["branchInfo"]);
                }
                // Find translations that need text embeddings
                const embeddingMaps = preShapeEmbeddableTranslatable<"id">({ Create, Update, objectType: __typename });
                return { ...embeddingMaps, bots, branchInfo };
            },
            create: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as ChatPre;
                const messageShapes = await shapeHelper({ relation: "messages", relTypes: ["Create"], isOneToOne: false, objectType: "ChatMessage", parentRelationshipName: "chat", data, ...rest });
                const messages = chatCreateNestedMessages(messageShapes?.create ?? [], preData.branchInfo[data.id] ?? {}) as any;

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
                    translations: await translationShapeHelper({ relTypes: ["Create"], embeddingNeedsUpdate: preData.embeddingNeedsUpdateMap[data.id], data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                const preData = rest.preMap[__typename] as ChatPre;
                const messages = await shapeHelper({ relation: "messages", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "ChatMessage", parentRelationshipName: "chat", data, ...rest });
                const messagesCreate = chatCreateNestedMessages(messages?.create ?? [], preData.branchInfo[data.id] ?? {}) as any;
                if (messagesCreate) {
                    messages.create = messagesCreate.create;
                }

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
            toGraphQL: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, userData)),
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
