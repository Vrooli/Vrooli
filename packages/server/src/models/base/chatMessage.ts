import { ChatCreateInput, ChatInviteCreateInput, ChatMessage, ChatMessageCreateInput, ChatMessageSearchTreeInput, ChatMessageSearchTreeResult, ChatMessageSortBy, ChatMessageUpdateInput, chatMessageValidation, ChatUpdateInput, getTranslation, MaxObjects, openAIServiceInfo, uuidValidate } from "@local/shared";
import { Request } from "express";
import { SessionService } from "../../auth/session.js";
import { addSupplementalFields, InfoConverter } from "../../builders/infoConverter.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { PartialApiInfo } from "../../builders/types.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { Trigger } from "../../events/trigger.js";
import { emitSocketEvent } from "../../sockets/events.js";
import { ChatContextManager, determineRespondingBots } from "../../tasks/llm/context.js";
import { requestBotResponse } from "../../tasks/llm/queue.js";
import { ChatMessagePre, getChatParticipantData, populatePreMapForChatUpdates, PreMapChatData, PreMapMessageData, PreMapMessageDataCreate, PreMapMessageDataDelete, PreMapMessageDataUpdate, PreMapUserData, prepareChatMessageOperations } from "../../utils/chat.js";
import { getAuthenticatedData } from "../../utils/getAuthenticatedData.js";
import { InputNode } from "../../utils/inputNode.js";
import { translationShapeHelper } from "../../utils/shapes/translationShapeHelper.js";
import { isOwnerAdminCheck } from "../../validators/isOwnerAdminCheck.js";
import { getSingleTypePermissions, permissionsCheck } from "../../validators/permissions.js";
import { ChatMessageFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { ChatMessageModelInfo, ChatMessageModelLogic, ChatModelInfo, ChatModelLogic, ReactionModelLogic, UserModelLogic } from "./types.js";

const DEFAULT_CHAT_TAKE = 25;
const MAX_CHAT_TAKE = DEFAULT_CHAT_TAKE;
const MIN_CHAT_TAKE = 1;

const __typename = "ChatMessage" as const;
export const ChatMessageModel: ChatMessageModelLogic = ({
    __typename,
    dbTable: "chat_message",
    dbTranslationTable: "chat_message_translation",
    display: () => ({
        label: {
            select: () => ({ id: true, translations: { select: { language: true, text: true } } }),
            get: (select, languages) => {
                return getTranslation(select, languages).text ?? "";
            },
        },
    }),
    format: ChatMessageFormat,
    mutate: {
        shape: {
            /**
             * Collects bot information, message information, and chat information for AI responses, 
             * notifications, and web socket events.
             * 
             * NOTE: Updated messages don't trigger AI responses. Instead, you must create a new message 
             * with versionIndex set to the previous version's index + 1.
             */
            pre: async ({ Create, Update, Delete, userData, inputsById, ...rest }): Promise<ChatMessagePre> => {
                // Initialize objects to store bot, message, and chat information
                const preMap: ChatMessagePre = { chatData: {}, messageData: {}, userData: {} };

                // Find stored chat and chat message information 
                await getChatParticipantData({
                    includeMessageInfo: true,
                    includeMessageParentInfo: true,
                    chatIds: [...Create].map(({ input }) => input.chatConnect).filter(Boolean) as string[],
                    messageIds: [...Update, ...Delete].map(({ node }) => node.id),
                    preMap,
                    userData,
                });

                // Collect information for new messages, which can't be queried for
                for (const { node, input } of Create) {
                    // Collect chat information
                    let chatId = input.chatConnect ?? null;
                    // Collect information for new chats
                    if (node.parent && node.parent.__typename === "Chat" && ["Create"].includes(node.parent.action)) {
                        const chatUpsertInfo = inputsById[node.parent.id]?.input as ChatCreateInput | undefined;
                        if (chatUpsertInfo?.id) {
                            chatId = chatUpsertInfo.id;
                            // Store all invite information. Later we'll check if any of these are bots (which are automatically accepted, 
                            // and can potentially be used for AI responses)
                            preMap.chatData[chatId] = {
                                potentialBotIds: chatUpsertInfo.invitesCreate?.map(i => typeof i === "string" ? (inputsById[i]?.input as ChatInviteCreateInput)?.userConnect : i.userConnect) ?? [],
                                participantsDelete: ((chatUpsertInfo as ChatUpdateInput).participantsDelete ?? []),
                                isNew: node.parent.action === "Create",
                            };
                        }
                    }
                    // Collect message data
                    const translations = (input.translationsCreate?.filter(t => t.text.length > 0) || []);
                    preMap.messageData[node.id] = {
                        __type: "Create",
                        chatId,
                        messageId: node.id,
                        parentId: input.parentConnect || null, // NOTE: This is overwritten later
                        translations,
                        userId: input.userConnect || userData.id,
                    };
                }

                // Update translation information for updated messages
                for (const { node, input } of Update) {
                    const messageData = preMap.messageData[node.id] as PreMapMessageDataUpdate | undefined;
                    const translationsUpdates = (input.translationsUpdate || []).filter(t => t.text && t.text.length > 0);
                    const translationsDeletes = (input.translationsDelete || []).filter(Boolean);
                    if (!messageData || (!translationsUpdates.length && !translationsDeletes.length)) {
                        continue;
                    }
                    let messageDataTranslations = messageData.translations || [];
                    // Apply updates
                    for (const { id, text, language } of translationsUpdates) {
                        const translation = messageDataTranslations.find(t => t.id === id);
                        if (translation) {
                            translation.text = text || "";
                        } else {
                            messageDataTranslations = [...messageDataTranslations, { id, language, text: text || "" }];
                        }
                    }
                    // Apply deletes
                    messageDataTranslations = messageDataTranslations.filter(t => !translationsDeletes.includes(t.id));

                    // Update messageData
                    messageData.translations = messageDataTranslations;
                }

                // Collect information for constructing the message tree, which is effected by creating and deleting messages on existing chats
                const chatsWithMessages: { [chatId: string]: { messagesCreate: ChatMessageCreateInput[]; messagesDelete: string[] } } = {};
                for (const { node, input } of [...Create, ...Delete]) {
                    // If chat is new, skip
                    const isChatNew = node.parent && node.parent.__typename === "Chat" && ["Create"].includes(node.parent.action);
                    if (isChatNew) continue;
                    const chatId = preMap.messageData[node.id]?.chatId;
                    if (!chatId) continue;
                    // Collect messages to create and delete for each chat
                    if (!chatsWithMessages[chatId]) chatsWithMessages[chatId] = { messagesCreate: [], messagesDelete: [] };
                    if (node.action === "Create") chatsWithMessages[chatId].messagesCreate.push(input as ChatMessageCreateInput);
                    else if (node.action === "Delete") chatsWithMessages[chatId].messagesDelete.push(input as string);
                }
                const chatUpdateInputs = Object.entries(chatsWithMessages).reduce((acc, [chatId, { messagesCreate, messagesDelete }]) => {
                    acc.push({
                        id: chatId,
                        messagesCreate,
                        messagesDelete,
                    });
                    return acc;
                }, [] as ChatUpdateInput[]);
                // Collect db information for constructing the message tree
                const { branchInfo } = await populatePreMapForChatUpdates({ updateInputs: chatUpdateInputs });
                for (const data of chatUpdateInputs) {
                    // Update Creates and Updates (and add new Updates if needed) to include parent and children information
                    const operations = await shapeHelper({ relation: "messages", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "ChatMessage", parentRelationshipName: "chat", data, idsCreateToConnect: {}, preMap: {}, userData, ...rest });
                    const { summary } = prepareChatMessageOperations(operations, branchInfo[data.id]);
                    for (const { id: messageId, parentId } of summary.Create) {
                        // Update the corresponding message in preMapMessageData
                        const messageData = preMap.messageData[messageId] as PreMapMessageDataCreate | PreMapMessageDataUpdate | undefined;
                        if (messageData) {
                            messageData.parentId = parentId;
                        }
                    }
                    for (const { id: messageId, parentId } of summary.Update) {
                        // If it exists, update the corresponding message in preMapMessageData
                        const messageData = preMap.messageData[messageId] as PreMapMessageDataCreate | PreMapMessageDataUpdate | undefined;
                        if (messageData) {
                            messageData.parentId = parentId;
                        }
                        // If it doesn't exist yet, this indicates the update is due to healing the tree 
                        // (i.e. a message was deleted, so its children need to be updated to point to the new parent).
                        // Add a new entry to preMapMessageData, and push new input to the Update array.
                        else {
                            const updateInput: { node: InputNode; input: ChatMessageUpdateInput; } = {
                                node: { __typename: "ChatMessage", id: messageId, action: "Update", children: [], parent: null },
                                input: { id: messageId }, // No updates needed. We'll get the parent ID from the preMap data
                            };
                            Update.push(updateInput);
                            preMap.messageData[messageId] = {
                                __type: "Update",
                                chatId: data.id,
                                messageId,
                                parentId,
                            };
                        }
                    }
                }

                // Parse potential bots and bots being removed
                const potentialBotIds: string[] = [];
                const participantsBeingRemovedIds: string[] = [];
                Object.entries(preMap.chatData).forEach(([id, chat]) => {
                    if (!chat.isNew) return;
                    if (chat.potentialBotIds) {
                        chat.potentialBotIds.forEach(botId => {
                            // Only add to potentialBotIds if the user ID is not already a key in botData (and also not your ID)
                            if (!preMap.userData[botId] && botId !== userData.id) potentialBotIds.push(botId);
                            // If it is already a key in bot data, update chatData.botParticipants
                            else if (preMap.userData[botId]) {
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
                    const potentialBots = await DbProvider.get().user.findMany({
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
                    potentialBots.forEach(user => {
                        // Any participant (even not bots) can be added to preMapUserData
                        preMap.userData[user.id] = {
                            botSettings: user.botSettings ?? JSON.stringify({}),
                            id: user.id,
                            isBot: true,
                            name: user.name,
                        };
                        // Add any bot that is public or invited by you to participants
                        if (!user.isPrivate || user.invitedByUser?.id === userData.id) {
                            Object.entries(preMap.chatData).forEach(([id, chat]) => {
                                if (chat.potentialBotIds?.includes(user.id) && !chat.botParticipants?.includes(user.id)) {
                                    if (!chat.botParticipants) chat.botParticipants = [];
                                    chat.botParticipants.push(user.id);
                                }
                            });
                        }
                    });
                }
                // Query participants being deleted and remove them from botData and chatData.botParticipants
                if (participantsBeingRemovedIds.length) {
                    const participantsBeingRemoved = await DbProvider.get().chat_participants.findMany({
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
                        // Remove from chatData.botParticipants
                        Object.values(preMap.chatData).forEach((chat) => {
                            if (chat.botParticipants?.includes(participant.user.id)) {
                                chat.botParticipants = chat.botParticipants.filter(id => id !== participant.user.id);
                            }
                        });
                        // NOTE: Don't remove from preMapUserData, since their messages may still be in the context for AI responses
                    });
                }
                // Messages can be created for you or bots. Make sure all new messages meet this criteria.
                for (const { node } of Create) {
                    const message = preMap.messageData[node.id] as PreMapMessageDataCreate | undefined;
                    if (message && message.userId !== userData.id && !Object.keys(preMap.userData).includes(message.userId ?? "")) {
                        throw new CustomError("0526", "Unauthorized", { message });
                    }
                }

                // Return data
                return preMap;
            },
            create: async ({ data, ...rest }) => {
                const preMap = rest.preMap[__typename] as ChatMessagePre;
                // Prefer parent ID from preMap data over what's provided by the client
                const messageData = preMap?.messageData[data.id] as PreMapMessageDataCreate | undefined;
                const parentId = messageData?.parentId !== undefined ? messageData.parentId : data.parentConnect;
                return {
                    id: data.id,
                    parent: parentId ? { connect: { id: parentId } } : undefined,
                    user: { connect: { id: data.userConnect ?? rest.userData.id } }, // Can create messages for bots. This is authenticated in the "pre" function.
                    versionIndex: data.versionIndex,
                    chat: await shapeHelper({ relation: "chat", relTypes: ["Connect"], isOneToOne: true, objectType: "Chat", parentRelationshipName: "messages", data, ...rest }),
                    translations: await translationShapeHelper({ relTypes: ["Create"], data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                const preMap = rest.preMap[__typename] as ChatMessagePre;
                // Allow parentId updates, but only from the pre function
                const messageData = preMap?.messageData[data.id] as PreMapMessageDataUpdate | undefined;
                const parentId = messageData?.parentId;
                return {
                    parent: parentId ? { connect: { id: parentId } } : undefined,
                    translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], data, ...rest }),
                };
            },
        },
        trigger: {
            afterMutations: async ({ additionalData, createdIds, deletedIds, updatedIds, preMap, resultsById, userData }) => {
                const preMapUserData: Record<string, PreMapUserData> = preMap[__typename]?.userData ?? {};
                const preMapChatData: Record<string, PreMapChatData> = preMap[__typename]?.chatData ?? {};
                const preMapMessageData: Record<string, PreMapMessageData> = preMap[__typename]?.messageData ?? {};
                return;
                // Call triggers
                for (const objectId of createdIds) {
                    const messageData = preMapMessageData[objectId] as PreMapMessageDataCreate | undefined;
                    if (!messageData) {
                        logger.error("Message data not found", { trace: "0238", user: userData.id, message: objectId });
                        continue;
                    }
                    const chatId = messageData.chatId;
                    const chatMessage = resultsById[objectId] as ChatMessage | undefined;
                    const senderId = messageData.userId;
                    if (!chatMessage || !chatId || !senderId) {
                        logger.error("Message, message sender, or chat not found", { trace: "0363", user: userData.id, messageId: objectId, chatId });
                        continue;
                    }
                    const messageText = getTranslation({ translations: messageData.translations }, userData.languages).text || null;
                    // Add message to cache
                    const model = additionalData.model || openAIServiceInfo.defaultModel;
                    await (new ChatContextManager(model, userData.languages)).addMessage(messageData);
                    // Common trigger logic
                    await Trigger(userData.languages).objectCreated({
                        createdById: userData.id,
                        hasCompleteAndPublic: true, // N/A
                        hasParent: true, // N/A
                        owner: { id: userData.id, __typename: "User" },
                        objectId,
                        objectType: __typename,
                    });
                    await Trigger(userData.languages).chatMessageCreated({
                        excludeUserId: userData.id,
                        chatId,
                        messageId: objectId,
                        senderId,
                        message: chatMessage,
                    });
                    // Determine which bots should respond, if any.
                    const chat: PreMapChatData | undefined = preMapChatData[chatId];
                    const bots: PreMapUserData[] = chat?.botParticipants?.map(id => preMapUserData[id]).filter(b => b) ?? [];
                    const botsToRespond = determineRespondingBots(messageText, messageData.userId, chat, bots, userData.id);
                    if (botsToRespond.length) {
                        // Send typing indicator while bots are responding
                        emitSocketEvent("typing", chatId, { starting: botsToRespond });
                        const task = additionalData.task || "Start";
                        const taskContexts = Array.isArray(additionalData.taskContexts) ? additionalData.taskContexts : [];
                        // For each bot that should respond, request bot response
                        for (const botId of botsToRespond) {
                            // Call LLM for bot response
                            requestBotResponse({
                                chatId,
                                mode: "text",
                                model,
                                parentId: messageData.messageId,
                                parentMessage: messageText,
                                respondingBotId: botId,
                                shouldNotRunTasks: false,
                                task,
                                taskContexts,
                                participantsData: preMapUserData,
                                userData,
                            });
                        }
                    }
                }
                for (const objectId of updatedIds) {
                    await Trigger(userData.languages).objectUpdated({
                        updatedById: userData.id,
                        hasCompleteAndPublic: true, // N/A
                        hasParent: true, // N/A
                        owner: { id: userData.id, __typename: "User" },
                        objectId,
                        objectType: __typename,
                        wasCompleteAndPublic: true, // N/A
                    });
                    // Update message in cache
                    const messageData = preMapMessageData[objectId] as PreMapMessageDataUpdate | undefined;
                    if (messageData) {
                        const model = additionalData.model || openAIServiceInfo.defaultModel;
                        await (new ChatContextManager(model, userData.languages)).editMessage(messageData);
                    }
                    //TODO should probably call determineRespondingBots and requestBotResponse here as well
                    const chatMessage = resultsById[objectId] as ChatMessage | undefined;
                    if (!chatMessage) {
                        logger.error("Result message not found", { trace: "0365", user: userData.id, messageId: objectId });
                        continue;
                    }
                    await Trigger(userData.languages).chatMessageUpdated({
                        data: preMapMessageData[objectId],
                        message: chatMessage,
                    });
                }
                for (const objectId of deletedIds) {
                    await Trigger(userData.languages).objectDeleted({
                        deletedById: userData.id,
                        hasBeenTransferred: false, // N/A
                        hasParent: true, // N/A
                        objectId,
                        objectType: __typename,
                        wasCompleteAndPublic: true, // N/A
                    });
                    const messageData = preMapMessageData[objectId] as PreMapMessageDataDelete | undefined;
                    if (messageData) {
                        await Trigger(userData.languages).chatMessageDeleted({
                            data: messageData,
                        });
                    } else {
                        logger.error("Message data not found", { trace: "0067", user: userData.id, message: objectId });
                    }
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
            if (!chatId || !uuidValidate(chatId)) { // Validate chatId
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

            if (inputStartId && uuidValidate(inputStartId)) {
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
                    where: { chatId },
                    orderBy: { sequence: "desc" },
                    select: { id: true },
                });

                if (!highestSeqMessage) {
                    return { __typename: "ChatMessageSearchTreeResult", messages: [], hasMoreUp: false, hasMoreDown: false };
                }
                startId = highestSeqMessage.id;

                // Check permission on the found start message
                const startMsgAuthDataById = await getAuthenticatedData({ "ChatMessage": [startId] }, userData ?? null);
                startMessageAuthData = startMsgAuthDataById[startId];
                if (!startMessageAuthData) {
                    // This case should theoretically not happen if highestSeqMessage was found and chat permission passed,
                    // but adding check for robustness.
                    throw new CustomError("0561", "InternalError", { startId, chatId, userId: userData?.id });
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
                where: { id: startId },
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
                    node.children.forEach((child: any) => flattenAndCheckDepth(child, depthUp, depthDown + 1, node.id));
                }
            }

            // Start flattening from the fetched root node (startId)
            flattenAndCheckDepth(tree, 0, 0);

            const rawMessages = Array.from(allNodesMap.values());

            // --- Convert & Supplement --- 
            const partialMessages = rawMessages.map((c: any) => InfoConverter.get().fromDbToApi(c, partialInfo)); // Use the already derived partialInfo
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
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data?.user,
        }),
        permissionResolvers: ({ data, isAdmin: isMessageOwner, isDeleted, isLoggedIn, userId }) => {
            const isChatAdmin = userId ? isOwnerAdminCheck(ModelMap.get<ChatModelLogic>("Chat").validate().owner(data.chat as ChatModelInfo["DbModel"], userId), userId) : false;
            const isParticipant = uuidValidate(userId) && (data.chat as Record<string, any>).participants?.some((p) => p.user?.id === userId || p.userId === userId);
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
                        { user: { id: data.userId } },
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
