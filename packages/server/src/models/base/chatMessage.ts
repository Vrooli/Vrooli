import type { Prisma } from "@prisma/client";
import { type ChatMessage, type ChatMessageSearchTreeInput, type ChatMessageSearchTreeResult, ChatMessageSortBy, chatMessageValidation, DEFAULT_LANGUAGE, generatePK, MaxObjects, type TaskContextInfo, validatePK } from "@vrooli/shared";
import { type Request } from "express";
import { SessionService } from "../../auth/session.js";
import { addSupplementalFields, InfoConverter } from "../../builders/infoConverter.js";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { type PartialApiInfo } from "../../builders/types.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { Trigger } from "../../events/trigger.js";
import { RedisMessageStore } from "../../services/response/messageStore.js";
import { QueueService } from "../../tasks/queues.js";
import { type LLMCompletionTask, QueueTaskType } from "../../tasks/taskTypes.js";
import { getAuthenticatedData } from "../../utils/getAuthenticatedData.js";
import { type ChatMessagePre, MessageInfoCollector, type PreMapChatData, type PreMapMessageData } from "../../utils/messageTree.js";
import { getSingleTypePermissions, isOwnerAdminCheck, permissionsCheck } from "../../validators/permissions.js";
import { ChatMessageFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { type ChatMessageModelInfo, type ChatMessageModelLogic, type ChatModelInfo, type ChatModelLogic, type ReactionModelLogic, type UserModelLogic } from "./types.js";

const DEFAULT_CHAT_TAKE = 25;
const MAX_CHAT_TAKE = DEFAULT_CHAT_TAKE;
const MIN_CHAT_TAKE = 1;

const __typename = "ChatMessage" as const;
export const ChatMessageModel: ChatMessageModelLogic = ({
    __typename,
    dbTable: "chat_message",
    display: () => ({
        label: {
            select: () => ({ id: true, text: true }),
            get: (select) => {
                return select.text ?? "";
            },
        },
    }),
    format: ChatMessageFormat,
    mutate: {
        shape: {
            /**
             * Collects message and chat information for AI responses, 
             * notifications, and web socket events.
             * 
             * NOTE: Updated messages don't trigger AI responses. Instead, you must create a new message 
             * with versionIndex set to the previous version's index + 1.
             */
            pre: async ({ Create, Update, Delete, userData }): Promise<ChatMessagePre> => {
                /* Gather ids ---------------------------------------------------------- */
                const createChatIds = Create.map(c => c.input.chatConnect).filter(Boolean) as string[];
                const updateDeleteIds = [...Update, ...Delete].map(x => x.node.id);

                /* 1️⃣  Fast meta from DB */
                const pre = await MessageInfoCollector.collect(createChatIds, updateDeleteIds);

                /* 2️⃣  Add placeholder records for brand-new messages we just received */
                for (const { node, input } of Create) {
                    pre.chatData[input.chatConnect] ??= { hasBotParticipants: true, isNew: false };
                    pre.messageData[node.id] = {
                        __type: "Create",
                        chatId: input.chatConnect,
                        messageId: node.id,
                        parentId: input.parentConnect ?? null,
                        text: input.text,
                        userId: input.userConnect ?? userData.id,
                    };
                }

                return pre;
            },
            create: async ({ data, userData, ...rest }) => {
                const preMap = rest.preMap[__typename] as ChatMessagePre;
                // Prefer parent ID from preMap data over what's provided by the client
                const messageData = preMap?.messageData[data.id];
                let parentId = data.parentConnect;
                if (messageData?.__type === "Create" && messageData.parentId !== undefined) {
                    parentId = messageData.parentId;
                }
                return {
                    id: BigInt(data.id),
                    config: data.config as unknown as Prisma.InputJsonValue,
                    language: userData.languages[0] ?? DEFAULT_LANGUAGE,
                    parent: parentId ? { connect: { id: BigInt(parentId) } } : undefined,
                    user: { connect: { id: BigInt(data.userConnect ?? userData.id) } }, // Can create messages for bots. This is authenticated in the "pre" function.
                    text: data.text,
                    versionIndex: data.versionIndex,
                    chat: await shapeHelper({ relation: "chat", relTypes: ["Connect"], isOneToOne: true, objectType: "Chat", parentRelationshipName: "messages", data, userData, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                const preMap = rest.preMap[__typename] as ChatMessagePre;
                // Allow parentId updates, but only from the pre function
                const messageData = preMap?.messageData[data.id];
                let parentId: string | undefined = undefined;
                if (messageData?.__type === "Update" && messageData.parentId) {
                    parentId = messageData.parentId;
                }
                return {
                    config: noNull(data.config as unknown as Prisma.InputJsonValue),
                    parent: parentId ? { connect: { id: BigInt(parentId) } } : undefined,
                    text: noNull(data.text),
                };
            },
        },
        trigger: {
            afterMutations: async ({ createdIds, deletedIds, updatedIds, preMap, resultsById, userData, additionalData }) => {
                const preMapChatData: Record<string, PreMapChatData> = preMap[__typename]?.chatData ?? {};
                const preMapMessageData: Record<string, PreMapMessageData> = preMap[__typename]?.messageData ?? {};

                // Handle created messages
                for (const objectId of createdIds) {
                    const messageData = preMapMessageData[objectId];
                    if (!messageData || messageData.__type !== "Create") {
                        logger.error("Message data not found or not 'Create' type for LLM task creation", { trace: "0238", user: userData.id, messageId: objectId });
                        continue;
                    }
                    const chatId = messageData.chatId;

                    // Extract additional data passed into endpoint
                    const modelForTask = additionalData?.model as string | undefined;
                    const taskContexts = additionalData?.taskContexts as TaskContextInfo[] | undefined;

                    await Trigger(userData.languages).objectCreated({
                        createdById: userData.id,
                        hasCompleteAndPublic: true, // N/A
                        hasParent: true, // N/A
                        owner: { id: userData.id, __typename: "User" },
                        objectId,
                        objectType: __typename,
                    });
                    const chatMessageForResult = resultsById[objectId] as ChatMessage | undefined;
                    if (!chatMessageForResult) {
                        logger.error("ChatMessage result not found for chatMessageCreated trigger", { trace: "0363a", user: userData.id, messageId: objectId });
                        continue;
                    }
                    await Trigger(userData.languages).chatMessageCreated({
                        excludeUserId: userData.id,
                        chatId,
                        messageId: objectId,
                        senderId: messageData.userId,
                        message: chatMessageForResult,
                    });

                    const chat: PreMapChatData | undefined = preMapChatData[chatId];
                    if (chat?.hasBotParticipants) {
                        const llmTaskPayload: LLMCompletionTask = {
                            id: generatePK().toString(),
                            type: QueueTaskType.LLM_COMPLETION,
                            chatId: messageData.chatId,
                            messageId: objectId,
                            userData,
                            taskContexts: taskContexts ?? [],
                            model: modelForTask,
                            allocation: {
                                maxCredits: userData.hasPremium ? "200" : "100",
                                maxDurationMs: 60000,
                                maxMemoryMB: 256,
                                maxConcurrentSteps: 1,
                            },
                            options: {
                                priority: userData.hasPremium ? "high" : "medium",
                                timeout: 60000,
                                retryPolicy: {
                                    maxRetries: 3,
                                    backoffMs: 1000,
                                    backoffMultiplier: 2,
                                    maxBackoffMs: 30000,
                                },
                            },
                        };
                        QueueService.get().swarm.addTask(llmTaskPayload);
                    }
                }

                // Handle updated messages
                for (const objectId of updatedIds) {
                    const messageData = preMapMessageData[objectId];
                    if (!messageData) {
                        logger.error("Message data not found for update operations", { trace: "chatMessage_update_no_preMapData", objectId });
                        continue;
                    }

                    await Trigger(userData.languages).objectUpdated({
                        updatedById: userData.id,
                        hasCompleteAndPublic: true, // N/A
                        hasParent: true, // N/A
                        owner: { id: userData.id, __typename: "User" },
                        objectId,
                        objectType: __typename,
                        wasCompleteAndPublic: true, // N/A
                    });

                    const updatedDbMessage = resultsById[objectId] as ChatMessage | undefined;
                    if (updatedDbMessage) {
                        await RedisMessageStore.get().updateMessage(objectId, updatedDbMessage);
                    } else {
                        logger.error("Updated message not found in resultsById for cache update", { trace: "chatMessage_update_cache_no_result", objectId });
                    }

                    const chatMessage = resultsById[objectId] as ChatMessage | undefined;
                    if (!chatMessage) {
                        logger.error("Result message not found for chatMessageUpdated trigger", { trace: "0365", user: userData.id, messageId: objectId });
                        continue;
                    }
                    await Trigger(userData.languages).chatMessageUpdated({
                        data: messageData,
                        message: chatMessage,
                    });
                }

                // Handle deleted messages
                for (const objectId of deletedIds) {
                    const messageData = preMapMessageData[objectId];
                    if (!messageData) {
                        logger.error("Message data not found for delete operations", { trace: "chatMessage_delete_no_preMapData", objectId });
                        continue;
                    }

                    await Trigger(userData.languages).objectDeleted({
                        deletedById: userData.id,
                        transferredAt: null, // N/A
                        hasParent: true, // N/A
                        objectId,
                        objectType: __typename,
                        wasCompleteAndPublic: true, // N/A
                    });

                    await RedisMessageStore.get().deleteMessage(objectId);

                    await Trigger(userData.languages).chatMessageDeleted({
                        data: messageData,
                    });
                }
            },
        },
        yup: chatMessageValidation,
    },
    query: {
        /**
         * Custom search query for chat messages. Starts either at the most recent 
         * message or specified, and traverses up and down the chat tree to find
         * surrounding messages.
         */
        async searchTree(
            req: Request,
            input: ChatMessageSearchTreeInput,
            info: PartialApiInfo,
        ): Promise<ChatMessageSearchTreeResult> {
            const { chatId, startId: inputStartId, take: takeInput, excludeUp, excludeDown } = input;

            // --- Input Validation --- 
            if (!chatId || !validatePK(chatId)) { // Validate chatId
                throw new CustomError("0531", "InvalidArgs", { input, reason: "Invalid or missing chatId" });
            }

            const take = Math.min(Math.max(MIN_CHAT_TAKE, takeInput ?? DEFAULT_CHAT_TAKE), MAX_CHAT_TAKE);
            const userData = SessionService.getUser(req);

            // --- Determine Start ID & Initial Permission Check --- 
            let startId: string;
            let startMessageAuthData: Awaited<ReturnType<typeof getAuthenticatedData>>[string] | undefined;

            // Check chat read permission first
            const chatAuthDataById = await getAuthenticatedData({ "Chat": [chatId] }, userData ?? null);
            if (!chatAuthDataById[chatId]) {
                throw new CustomError("0016", "ChatNotFoundOrUnauthorized", { chatId, userId: userData?.id });
            }
            await permissionsCheck({ [chatId]: chatAuthDataById[chatId] }, { ["Read"]: [chatId] }, {}, userData);

            if (inputStartId && validatePK(inputStartId)) {
                startId = inputStartId;
                // Check permission on the provided start message
                const startMsgAuthDataById = await getAuthenticatedData({ "ChatMessage": [startId] }, userData ?? null);
                startMessageAuthData = startMsgAuthDataById[startId];
                if (!startMessageAuthData) {
                    throw new CustomError("0017", "NotFound", { startId, chatId, userId: userData?.id });
                }
                await permissionsCheck({ [startId]: startMessageAuthData }, { ["Read"]: [startId] }, {}, userData);
            } else {
                // Find the message with the highest sequence in the chat
                const highestSeqMessage = await DbProvider.get().chat_message.findFirst({
                    where: { chatId: BigInt(chatId) },
                    orderBy: { createdAt: "desc" },
                    select: { id: true },
                });

                if (!highestSeqMessage) {
                    return { __typename: "ChatMessageSearchTreeResult", messages: [], hasMoreUp: false, hasMoreDown: false };
                }
                startId = highestSeqMessage.id.toString();

                // Check permission on the found start message
                const startMsgAuthDataById = await getAuthenticatedData({ "ChatMessage": [startId] }, userData ?? null);
                startMessageAuthData = startMsgAuthDataById[startId];
                if (!startMessageAuthData) {
                    // This case should theoretically not happen if highestSeqMessage was found and chat permission passed,
                    // but adding check for robustness.
                    throw new CustomError("0961", "InternalError", { startId, chatId, userId: userData?.id });
                }
            }

            // --- Prepare Select ---
            // Convert the incoming GraphQL info into a PartialApiInfo object
            const partialInfo = InfoConverter.get().fromApiToPartialApi(info.messages as PartialApiInfo, ChatMessageFormat.apiRelMap, true);
            // Convert the PartialApiInfo into a Prisma select object
            const baseSelect = InfoConverter.get().fromPartialApiToPrismaSelect(partialInfo);
            if (!baseSelect?.select) { // Check if baseSelect and baseSelect.select are valid
                throw new CustomError("0532", "InternalError", { info });
            }

            // Helper function to recursively build select object for parent/child branches
            // Accepts baseSelectFields to merge requested fields at each level
            function buildBranchSelect(depth: number, selectUp: boolean, selectDown: boolean, baseSelectFields: Record<string, any>): Record<string, any> {
                // Base case: Merge base fields with id
                if (depth <= 0) {
                    return {
                        ...baseSelectFields,
                        id: true,
                    };
                }

                // Recursive step: Start with base fields + id
                const select: Record<string, any> = { ...baseSelectFields, id: true };

                if (selectUp) {
                    // Continue selecting parents up the chain, passing baseSelectFields
                    select.parent = {
                        select: buildBranchSelect(depth - 1, true, false, baseSelectFields), // Keep going up, don't select children from this path
                    };
                }

                if (selectDown) {
                    // Continue selecting children down the chain, passing baseSelectFields
                    select.children = {
                        select: buildBranchSelect(depth - 1, false, true, baseSelectFields), // Keep going down, don't select parents from this path
                    };
                }

                return select;
            }

            // Start with the base fields requested by the client for the root node
            const finalSelect: Record<string, any> = { ...baseSelect.select, id: true };

            // Add the parent branch select if not excluded, passing baseSelect.select
            if (!excludeUp) {
                // Fetch 'take' levels up (one more than needed for the result) to check for hasMoreUp.
                finalSelect.parent = {
                    select: buildBranchSelect(take, true, false, baseSelect.select),
                };
            }

            // Add the children branch select if not excluded, passing baseSelect.select
            if (!excludeDown) {
                // Create a copy of baseSelect.select and remove the parent field for child recursion
                const childBaseSelect = { ...baseSelect.select };
                delete childBaseSelect.parent;

                // Fetch 'take' levels down, passing the modified base fields.
                const childrenSelect = buildBranchSelect(take, false, true, childBaseSelect);

                finalSelect.children = {
                    select: childrenSelect,
                };
            }

            // Generate the full select object for the findUnique query
            const treeSelect = {
                select: finalSelect,
            };

            // --- Fetch message tree ---
            const tree = await DbProvider.get().chat_message.findUnique({
                where: { id: BigInt(startId) },
                select: treeSelect.select,
            });

            if (!tree) {
                throw new CustomError("0018", "NotFound", { startId, chatId });
            }

            // --- Flatten Tree & Determine hasMore --- 
            let hasMoreUp = false;
            let hasMoreDown = false;
            const allNodesMap = new Map<string, any>();

            // New flatten function to track depth, determine hasMore flags, and add parentId linkage
            function flattenAndCheckDepth(node: any, depthUp: number, depthDown: number, parentId: string | null = null) {
                if (!node || allNodesMap.has(node.id)) {
                    return; // Already processed or null
                }

                // Check depth limits before processing/adding
                if (!excludeUp && depthUp >= take) {
                    hasMoreUp = true;
                    return; // Reached limit upwards, don't add or recurse further up
                }
                if (!excludeDown && depthDown >= take) {
                    hasMoreDown = true;
                    return; // Reached limit downwards on this branch, don't add or recurse further down
                }

                // Store node (shallow copy)
                const shallowNode = { ...node };

                // Set parentId:
                // 1. If explicitly passed (meaning we traversed down to this node)
                // 2. Or if the node fetched from DB has a parent object with an id (meaning we traversed up or this is the start node)
                if (parentId) {
                    shallowNode.parentId = parentId;
                    shallowNode.parent = { __typename: "ChatMessage", id: parentId };
                } else if (node.parent?.id) {
                    shallowNode.parentId = node.parent.id;
                    shallowNode.parent = { __typename: "ChatMessage", id: node.parent.id };
                }

                // Clean up relations to avoid circular refs and redundant data
                delete shallowNode.parent;
                delete shallowNode.children;

                allNodesMap.set(node.id, shallowNode);

                // Recurse Up (Parent)
                if (!excludeUp && node.parent) {
                    // Pass null for parentId when going up (parentId will be derived from node.parent.parent inside the recursive call)
                    flattenAndCheckDepth(node.parent, depthUp + 1, depthDown, null);
                }

                // Recurse Down (Children)
                if (!excludeDown && node.children) {
                    // Pass current node's id as parentId when going down
                    node.children.forEach((child) => flattenAndCheckDepth(child, depthUp, depthDown + 1, node.id));
                }
            }

            // Start flattening from the fetched root node (startId)
            flattenAndCheckDepth(tree, 0, 0);

            const rawMessages = Array.from(allNodesMap.values());

            // --- Convert & Supplement --- 
            const partialMessages = rawMessages.map((c) => InfoConverter.get().fromDbToApi(c, partialInfo)); // Use the already derived partialInfo
            const messagesWithSupplements = await addSupplementalFields(userData, partialMessages, partialInfo); // Use partialInfo here too

            // --- Return Result --- 
            return {
                __typename: "ChatMessageSearchTreeResult" as const,
                hasMoreDown,
                hasMoreUp,
                messages: messagesWithSupplements as ChatMessage[], // Cast to expected type
            };
        },
    },
    search: {
        defaultSort: ChatMessageSortBy.DateCreatedDesc,
        searchFields: {
            createdTimeFrame: true,
            languages: true,
            minScore: true,
            chatId: true,
            userId: true,
            updatedTimeFrame: true,
        },
        sortBy: ChatMessageSortBy,
        searchStringQuery: () => ({
            OR: [
                "textWrapped",
                { user: ModelMap.get<UserModelLogic>("User").search.searchStringQuery() },
            ],
        }),
        supplemental: {
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<ChatMessageModelInfo["ApiPermission"]>(__typename, ids, userData)),
                        reaction: await ModelMap.get<ReactionModelLogic>("Reaction").query.getReactions(userData?.id, ids, __typename),
                    },
                };
            },
        },
    },
    validate: () => ({
        isDeleted: () => false,
        isPublic: (_data, _getParentInfo?) => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, _userId) => ({
            User: data?.user,
        }),
        permissionResolvers: ({ data, isAdmin: isMessageOwner, isDeleted, isLoggedIn, userId }) => {
            const chatOwnerData = ModelMap.get<ChatModelLogic>("Chat").validate().owner(data.chat as ChatModelInfo["DbModel"], userId ?? null);
            const ownerDataWithStringIds = {
                Team: chatOwnerData.Team ? { ...chatOwnerData.Team, id: chatOwnerData.Team.id.toString() } : null,
                User: chatOwnerData.User ? { ...chatOwnerData.User, id: chatOwnerData.User.id.toString() } : null,
            };
            const isChatAdmin = userId ? isOwnerAdminCheck(ownerDataWithStringIds, userId) : false;
            const isParticipant = validatePK(userId) && (data.chat as Record<string, any>).participants?.some((p) => p.user?.id.toString() === userId || p.userId.toString() === userId);
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
            own: function getOwn(data) {
                return { // If you own the chat or created the message
                    OR: [
                        { chat: useVisibility("Chat", "Own", data) },
                        { user: { id: BigInt(data.userId) } },
                    ],
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    OR: [
                        { chat: useVisibility("Chat", "Own", data) },
                        { chat: useVisibility("Chat", "Public", data) },
                    ],
                };
            },
            ownPrivate: function getOwnPrivate(data) {
                return { chat: useVisibility("Chat", "OwnPrivate", data) };
            },
            ownPublic: function getOwnPublic(data) {
                return { chat: useVisibility("Chat", "OwnPublic", data) };
            },
            public: function getPublic(data) {
                return {
                    chat: useVisibility("Chat", "Public", data),
                };
            },
        },
    }),
});
