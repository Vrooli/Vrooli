import { ChatMessage, ChatMessageCreateInput, ChatMessageCreateWithTaskInfoInput, ChatMessageSearchTreeInput, ChatMessageSearchTreeResult, ChatMessageShape, ChatMessageUpdateInput, ChatMessageUpdateWithTaskInfoInput, ChatParticipant, ChatShape, DUMMY_ID, LlmTask, LlmTaskInfo, RegenerateResponseInput, Session, Success, TaskContextInfo, endpointGetChatMessageTree, endpointPostChatMessage, endpointPostRegenerateResponse, endpointPutChatMessage, getTranslation, noop, uuid } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { SessionContext } from "contexts";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { getCurrentUser } from "utils/authentication/session";
import { getUserLanguages } from "utils/display/translationTools";
import { BranchMap, getCookieMessageTree, setCookieMessageTree } from "utils/localStorage";
import { PubSub } from "utils/pubsub";
import { UseChatTaskReturn } from "./tasks";
import { useHistoryState } from "./useHistoryState";
import { useLazyFetch } from "./useLazyFetch";

export type MinimumChatMessage = {
    id: string;
    created_at?: string;
    parent?: {
        id: string;
        parent?: {
            id: string;
        } | null;
    } | null;
    user?: {
        id: string;
    } | null;
    versionIndex: number;
    sequence?: number;
    translations: ChatMessageShape["translations"];
};

/** Tree structure for displaying chat messages in the correct order */
export type MessageNode<T extends MinimumChatMessage> = {
    message: T;
    /** When there is more than one child, there are multiple versions (edits) of the message */
    children: string[];
};

export type MessageTree<T extends MinimumChatMessage> = {
    /** Data for each message, including parent and children */
    map: Map<string, MessageNode<T>>;
    /** 
     * Messages without a parent, meaning they're either 
     * one of the start messages (there may be multiple versions), 
     * or they're orphaned messages (e.g. when a parent is deleted), 
     * and we haven't repaired the tree yet.
     */
    roots: string[];
};

/**
 * Finds the best message to edit when starting to reply to a message. 
 * 
 * This function searches down the message tree, starting at the replied message, 
 * and finds the first message that you've sent, which either: 
 * - Doesn't mention any other participants, or
 * - Mentions the user you're replying to.
 * 
 * If no messages are found, it returns null. This indicates that the reply will 
 * be a brand new message, rather than an edit.
 * 
 * NOTE: Only checks children that are currently in view
 * 
 * @returns {string | null} - The ID of the target message if found, or null if no suitable message is found.
 */
export function findReplyMessage(
    tree: MessageTree<MinimumChatMessage>,
    branches: BranchMap,
    replyingMessage: Pick<ChatMessageShape, "id" | "translations" | "user">,
    participants: Pick<ChatParticipant, "id" | "user">[],
    session: Session | undefined,
): string | null {
    const userData = getCurrentUser(session);
    const userLanguages = getUserLanguages(session);
    const { map } = tree;
    const otherParticipants = participants.filter(p => p.id !== userData.id);

    let currentNode = map.get(replyingMessage.id);
    if (!currentNode || !userData.id) return null;
    const replyingToUserHandle = replyingMessage.user?.handle ?? replyingMessage.user?.name ?? replyingMessage.user?.id;

    // Traverse down the tree to find the best message to edit
    const MAX_DEPTH = 10;
    let depth = 0;
    while (currentNode && depth < MAX_DEPTH) {
        depth++;
        // Pick the visible child
        if (!currentNode.children.length) return null;
        const selectedChild = currentNode.children.length > 1 && branches[currentNode.message.id]
            ? branches[currentNode.message.id]
            : currentNode.children[0];
        if (!selectedChild) return null;
        const nextNode = map.get(selectedChild);
        if (!nextNode) return null;

        // If the message isn't yours, continue to the next child
        const isOwnMessage = currentNode.message.user?.id === userData.id;
        if (!isOwnMessage) {
            currentNode = nextNode;
            continue;
        }

        // If there's only one other participant, return the node
        if (otherParticipants.length <= 1) {
            return currentNode.message.id;
        }

        // Otherwise, look for mentions (e.g. @user or @everyone)
        const messageText = getTranslation(currentNode.message, userLanguages, true)?.text ?? "";
        const handleRegex = /\b(?=\w{3,16}\b)[a-zA-Z0-9_]{3,16}/g;
        const matches = messageText.match(handleRegex);

        // If there are no mentions, return the node
        if (!matches) {
            return currentNode.message.id;
        }

        // If there are only mentions to the user you're replying to, return the node
        if (matches.includes(replyingToUserHandle)) {
            return currentNode.message.id;
        }

        currentNode = nextNode;
    }

    // No suitable message was found
    return null;
}

/**
 * Attempts to find a suitable parent or sibling for an orphaned node.
 * 
 * This function first tries to attach the orphaned node to its direct parent if possible.
 * If the direct parent does not exist or is not suitable, it then attempts to attach to the grandparent.
 * If that fails, it attempts to find the closest node by sequence number, and if that fails, by timestamp.
 * If no suitable node is found by any of these criteria, the orphaned node will remain a root.
 */
function findSuitableParentOrSibling(
    messageMap: Map<string, MessageNode<MinimumChatMessage>>,
    orphanId: string,
): string | null {
    const orphan = messageMap.get(orphanId);
    if (!orphan) return null;

    // First, try to attach to the direct parent if possible
    const parentId = orphan.message.parent?.id;
    const parentNode = parentId ? messageMap.get(parentId) : null;
    if (parentNode) return parentId || null;

    // Then, try to attach to the grandparent if possible
    const grandparentId = orphan.message.parent?.parent?.id;
    const grandparent = grandparentId ? messageMap.get(grandparentId) : null;
    if (grandparent) return grandparentId || null;

    // If no parent or grandparent, find the closest lesser sequence
    const closestBySequence = findClosestBySequence(messageMap, orphan.message.sequence);
    if (closestBySequence) return closestBySequence;

    // If no sequence match, find the closest lesser timestamp
    const closestByTimestamp = findClosestByTimestamp(messageMap, orphan.message.created_at);
    if (closestByTimestamp) return closestByTimestamp;

    // If all else fails, the node will remain a root
    return null;
}

/**
 * Finds the closest node by sequence number that is less than the sequence number of the given node.
 * 
 * This function iterates through all nodes in the message map and identifies the node with the closest lesser
 * sequence number compared to the provided sequence number. This is used to place orphaned nodes in a logical
 * position within the message tree based on their sequence.
 */
function findClosestBySequence(
    messageMap: Map<string, MessageNode<MinimumChatMessage>>,
    sequence?: number,
): string | null {
    let closestNode: MessageNode<MinimumChatMessage> | null = null;
    let closestSequence = -Infinity;
    if (!sequence) return null;
    messageMap.forEach(node => {
        if (node.message.sequence && node.message.sequence < sequence && node.message.sequence > closestSequence) {
            closestSequence = node.message.sequence;
            closestNode = node;
        }
    });
    return closestNode;
}

/**
 * Finds the closest node by timestamp that is earlier than the timestamp of the given node.
 * 
 * This function searches through the message map to find the node with the closest timestamp that is still
 * earlier than the given node's timestamp. This helps in placing orphaned nodes in a logical position within
 * the message tree when sequence numbers are not available or applicable.
 * 
 * @param {Map<string, MessageNode<MinimumChatMessage>>} newMessageMap - The current map of message nodes.
 * @param {string | undefined} timestamp - The timestamp of the node to find a position for.
 * @returns {MessageNode<MinimumChatMessage> | null} The closest node by timestamp, or null if no suitable node is found.
 */
function findClosestByTimestamp(
    messageMap: Map<string, MessageNode<MinimumChatMessage>>,
    timestamp?: string,
): string | null {
    let closestNode: MessageNode<MinimumChatMessage> | null = null;
    let closestTimestamp = -Infinity;
    if (!timestamp) return null;
    messageMap.forEach(node => {
        if (!node.message.created_at) return;
        const nodeTimestamp = new Date(node.message.created_at).getTime();
        const orphanTimestamp = new Date(timestamp).getTime();
        if (nodeTimestamp < orphanTimestamp && nodeTimestamp > closestTimestamp) {
            closestTimestamp = nodeTimestamp;
            closestNode = node;
        }
    });
    return closestNode;
}

/**
 * Processes and attempts to reattach orphaned nodes within the message tree.
 * 
 * This function iterates over all orphaned nodes and attempts to find a suitable parent or sibling for each,
 * based on a set of criteria including grandparent attachment, sequence number, and timestamp. If no suitable
 * location is found within the existing tree, the orphaned node is added as a new root node.
 */
function handleOrphanedNodes(
    messageMap: Map<string, MessageNode<MinimumChatMessage>>,
    roots: string[],
    orphanIds: string[],
) {
    if (orphanIds.length === 0) return;

    console.warn(`Found ${orphanIds.length} orphaned nodes. Attempting to reattach.`);

    const reattachedOrphans = new Set<string>();

    orphanIds.forEach(orphanId => {
        const suitableParentId = findSuitableParentOrSibling(messageMap, orphanId);
        const suitableParent = suitableParentId ? messageMap.get(suitableParentId) : null;
        if (suitableParent) {
            suitableParent.children.push(orphanId);
            sortSiblings(messageMap, suitableParent.children);
            reattachedOrphans.add(orphanId);
        } else {
            const orphanNode = messageMap.get(orphanId);
            if (orphanNode && !orphanNode.message.parent) {
                // Only add to roots if message has no parent
                if (!roots.includes(orphanId)) {
                    roots.push(orphanId);
                    sortSiblings(messageMap, roots);
                }
            } else {
                console.error(`Cannot reattach orphaned node ${orphanId} and it has a parent id ${orphanNode?.message.parent?.id}`);
            }
        }
    });

    // Remove reattached orphans from the list of orphanIds and rootIds if necessary
    reattachedOrphans.forEach(id => {
        const index = orphanIds.indexOf(id);
        if (index > -1) orphanIds.splice(index, 1);

        const rootIndex = roots.indexOf(id);
        if (rootIndex > -1) roots.splice(rootIndex, 1);
    });
}

/**
 * Sorts sibling nodes in the message tree based on their version index or, if not available, their sequence number.
 * 
 * This function is used to ensure that sibling nodes are displayed in the correct order within the message tree.
 * It first attempts to sort siblings based on their version index. If the version index is not available for comparison,
 * it falls back to sorting by sequence number.
 */
function sortSiblings(
    map: Map<string, MessageNode<MinimumChatMessage>>,
    siblings: string[],
) {
    siblings.sort((a, b) => {
        const aNode = map.get(a);
        const bNode = map.get(b);
        if (!aNode || !bNode) return 0; // If either node is not found, return 0 (no change)

        const aVersionIndex = aNode.message.versionIndex ?? 0;
        const bVersionIndex = bNode.message.versionIndex ?? 0;

        if (aVersionIndex !== bVersionIndex) {
            return aVersionIndex - bVersionIndex;
        }

        // If versionIndex is the same, sort by sequence
        const aSequence = aNode.message.sequence ?? 0;
        const bSequence = bNode.message.sequence ?? 0;
        return aSequence - bSequence;
    });
}


function removeMessageFromTree(
    messageId: string,
    map: Map<string, MessageNode<ChatMessageShape>>,
    roots: string[],
) {
    const nodeToRemove = map.get(messageId);
    if (!nodeToRemove) {
        console.error(`Message ${messageId} not found in messageMap`);
        return;
    }
    // Remove from parent's children
    const parentId = nodeToRemove.message.parent?.id;
    if (parentId) {
        const parentNode = map.get(parentId);
        if (parentNode) {
            parentNode.children = parentNode.children.filter(childId => childId !== messageId);
        }
    } else {
        // Remove from roots
        const rootIndex = roots.indexOf(messageId);
        if (rootIndex !== -1) {
            roots.splice(rootIndex, 1);
        }
    }
    // Remove from map
    map.delete(messageId);
}

function addMessageToTree(
    message: ChatMessageShape,
    map: Map<string, MessageNode<ChatMessageShape>>,
    roots: string[],
    orphans: string[],
) {
    const messageId = message.id;
    // Add to map
    if (!map.has(messageId)) {
        map.set(messageId, { message, children: [] });
    } else {
        // Update existing message data
        const existingNode = map.get(messageId);
        if (existingNode) {
            existingNode.message = message;
        }
    }
    const parentId = message.parent?.id;
    if (parentId) {
        const parentNode = map.get(parentId);
        if (parentNode) {
            // Avoid duplicates
            if (!parentNode.children.includes(messageId)) {
                parentNode.children.push(messageId);
                sortSiblings(map, parentNode.children);
            }
        } else {
            // Parent not found, add to orphans
            if (!orphans.includes(messageId)) {
                orphans.push(messageId);
            }
        }
    } else {
        // Message is a root
        if (!roots.includes(messageId)) {
            roots.push(messageId);
            sortSiblings(map, roots);
        }
    }
}

/**
 * Manages the chat message tree for a chat.
 * 
 * NOTE: This only updates the UI state. It does not update the server. 
 * If you want to update the server and UI, use `useMessageActions`.
 */
export function useMessageTree(chatId?: string | null) {
    // We query messages separate from the chat, since we must traverse the message tree
    const [getTreeData, { data: searchTreeData, loading: isTreeLoading }] = useLazyFetch<ChatMessageSearchTreeInput, ChatMessageSearchTreeResult>(endpointGetChatMessageTree);

    // The message tree structure
    const [tree, setTree] = useState<MessageTree<ChatMessageShape>>({ map: new Map(), roots: [] });
    const messagesCount = useMemo(() => tree.map.size, [tree.map.size]);

    /**
     * Clears all messages from the tree, resetting the state to its initial state.
     */
    const clearMessages = useCallback(() => {
        setTree({ map: new Map(), roots: [] });
    }, []);

    // Which branches of the tree are in view
    const [branches, setBranches] = useState<BranchMap>(chatId ? (getCookieMessageTree(chatId)?.branches ?? {}) : {});

    useEffect(() => {
        if (!chatId) return;
        // Update the cookie with current branches
        setCookieMessageTree(chatId, { branches, locationId: "someLocationId" }); // TODO locationId should be last chat message in view
    }, [branches, chatId]);

    const addMessages = useCallback((newMessages: ChatMessageShape[]) => {
        if (!newMessages || newMessages.length === 0) return;
        setTree(({ map: prevMessageMap, roots: prevRoots }) => {
            const map = new Map(prevMessageMap);
            const roots = [...prevRoots];
            const orphans: string[] = [];

            newMessages.forEach(message => {
                const existingNode = map.get(message.id);
                if (existingNode) {
                    // Remove message if it's a duplicate, so we can re-add it with the new data
                    removeMessageFromTree(message.id, map, roots);
                }
                // Add the message to the tree
                addMessageToTree(message, map, roots, orphans);
            });

            // Attempt to reattach orphaned nodes.
            handleOrphanedNodes(map, roots, orphans);

            // Sort roots
            sortSiblings(map, roots);

            return { map, roots };
        });
    }, []);

    const removeMessages = useCallback((messageIds: string[]) => {
        setTree(({ map: prevMessageMap, roots: prevRoots }) => {
            const map = new Map(prevMessageMap);
            let roots = [...prevRoots];

            messageIds.forEach(messageId => {
                // Remove from map
                const nodeToRemove = map.get(messageId);
                if (!nodeToRemove) {
                    console.error(`Message ${messageId} not found in messageMap`);
                    return;
                }
                map.delete(messageId); // Remove the node from the messageMap

                // Remove from parent's children
                if (nodeToRemove.message.parent?.id) {
                    const parentNode = map.get(nodeToRemove.message.parent.id);
                    if (parentNode) {
                        parentNode.children = parentNode.children.filter(childId => childId !== messageId);
                        parentNode.children.push(...nodeToRemove.children);
                    }
                }

                // Remove from root
                roots = roots.filter(rootId => rootId !== messageId);
            });

            return { map, roots };
        });
    }, []);

    /**
     * Edits the content of a message in the tree. DOES NOT 
     * add a new version of the message. This method is useful when 
     * chatting with other participants, where branching is disabled.
     */
    const editMessage = useCallback((updatedMessage: (Partial<ChatMessageShape> & { id: string })) => {
        setTree(({ map: prevMessageMap, roots }) => {
            if (!prevMessageMap.has(updatedMessage.id)) {
                console.error(`Message ${updatedMessage.id} not found in messageMap. Cannot edit.`);
                return { map: prevMessageMap, roots }; // Return the previous state if the message is not found
            }

            // Create a new map for immutability
            const map = new Map(prevMessageMap);

            // Retrieve the node and update it with the new message data
            const node = map.get(updatedMessage.id);
            if (!node) {
                console.error(`Message ${updatedMessage.id} not found in messageMap. Cannot edit.`);
                return { map: prevMessageMap, roots }; // Return the previous state if the message is not found
            }

            const updatedNode = { ...node, message: { ...node.message, ...updatedMessage } };

            map.set(updatedMessage.id, updatedNode as MessageNode<ChatMessageShape>); // Update the map with the edited message node
            return { map, roots };
        });
    }, []);

    // When chatId changes, clear everything and fetch new data
    useEffect(function clearData() {
        clearMessages();
        setBranches({});
        if (chatId && chatId !== DUMMY_ID) {
            getTreeData({ chatId });
        }
    }, [chatId, clearMessages, getTreeData]);
    useEffect(() => {
        if (!searchTreeData || searchTreeData.messages.length === 0) return;
        addMessages(searchTreeData.messages.map(message => ({ ...message, status: "sent" })));
    }, [addMessages, searchTreeData]);

    // Return the necessary state and functions
    return {
        addMessages,
        branches,
        clearMessages,
        editMessage,
        isTreeLoading,
        messagesCount,
        removeMessages,
        setBranches,
        tree,
    };
}

/**
 * How long to wait to collect task context before timing out
 */
const GET_TASK_CONTEXT_TIMEOUT_MS = 500;

type UseMessageActionsProps = {
    activeTask: UseChatTaskReturn["activeTask"];
    addMessages: (newMessages: ChatMessage[]) => unknown;
    chat: ChatShape | null | undefined;
    contexts: UseChatTaskReturn["contexts"][string];
    editMessage: (updatedMessage: (Partial<ChatMessageShape> & { id: string; })) => unknown;
    isBotOnlyChat: boolean;
    language: string;
    setMessage: (updatedMessage: string) => unknown;
};

type UseMesssageActionsResult = {
    postMessage: (text: string) => Promise<unknown>;
    putMessage: (originalMessage: ChatMessageShape, text: string) => Promise<unknown>;
    regenerateResponse: (message: ChatMessageShape) => Promise<unknown>;
    replyToMessage: (messageBeingRepliedTo: ChatMessageShape, text: string) => Promise<unknown>;
    retryPostMessage: (failedMessage: ChatMessageShape) => Promise<unknown>;
};

/**
 * Performs various chat message actions by 
 * updating the server database and local tree structure
 */
export function useMessageActions({
    activeTask,
    addMessages,
    chat,
    contexts,
    editMessage,
    isBotOnlyChat,
    language,
    setMessage,
}: UseMessageActionsProps): UseMesssageActionsResult {
    const session = useContext(SessionContext);

    const [postMessageEndpoint] = useLazyFetch<ChatMessageCreateWithTaskInfoInput, ChatMessage>(endpointPostChatMessage);
    const [putMessageEndpoint] = useLazyFetch<ChatMessageUpdateWithTaskInfoInput, ChatMessage>(endpointPutChatMessage);

    /** Collects context data for the active task */
    const collectTaskContext = useCallback((taskInfo: LlmTaskInfo | null) => new Promise<TaskContextInfo>((resolve, reject) => {
        if (!chat?.id) return;
        // Only try to collect data from `useAutoFill`, which provides the current form's data as context
        if (!PubSub.get().hasSubscribers("requestTaskContext", (metadata) => metadata?.source === "useAutoFill")) {
            return;
        }

        let unsubscribe: (() => unknown) = noop;

        const timeoutId = setTimeout(() => {
            unsubscribe();
            reject(new Error("Task context collection timed out"));
        }, GET_TASK_CONTEXT_TIMEOUT_MS);

        const task = taskInfo?.task || null;
        unsubscribe = PubSub.get().subscribe("requestTaskContext", (data) => {
            if (data.__type !== "response" || chat.id !== data.chatId || data.task !== task) return;
            clearTimeout(timeoutId);
            unsubscribe();
            resolve(data.context);
        });

        PubSub.get().publish("requestTaskContext", { __type: "request", chatId: chat.id, task: task as LlmTask | null });
    }), [chat?.id]);

    const getTaskContexts = useCallback(async function getTaskContextCallback(): Promise<TaskContextInfo[]> {
        if (activeTask) {
            try {
                // Try to collect context data for the active task
                const context = await collectTaskContext(activeTask);
                // If found, it'll either replace an existing context or be added to the list
                const contextIndex = contexts.findIndex((c) => c.id === context.id);
                const updatedContexts = [...contexts];
                if (contextIndex > -1) {
                    updatedContexts[contextIndex] = context;
                } else {
                    updatedContexts.push(context);
                }
                return updatedContexts;
            } catch (error) {
                console.error("Failed to collect task context:", error);
                return { ...contexts };
            }
        } else {
            return [...contexts];
        }
    }, [activeTask, collectTaskContext, contexts]);

    const messagePosted = useCallback((newMessage: ChatMessage) => {
        addMessages([newMessage]);
        setMessage("");
    }, [addMessages, setMessage]);
    const messagePutted = useCallback((updatedMessage: ChatMessage) => {
        editMessage(updatedMessage);
    }, [editMessage]);

    /** Commit a new message */
    const postMessage = useCallback(async function postMessageCallback(text: string) {
        if (!chat || text.trim() === "") return;
        const taskContexts = await getTaskContexts();
        const message: ChatMessageCreateInput = {
            id: uuid(),
            chatConnect: chat.id,
            userConnect: getCurrentUser(session).id ?? "",
            versionIndex: 0,
            translationsCreate: [{
                id: uuid(),
                language,
                text,
            }],
        };
        fetchLazyWrapper<ChatMessageCreateWithTaskInfoInput, ChatMessage>({
            fetch: postMessageEndpoint,
            inputs: {
                chatId: chat.id,
                message,
                task: activeTask.task as LlmTask,
                taskContexts,
            },
            onSuccess: messagePosted,
        });
    }, [activeTask.task, chat, getTaskContexts, language, messagePosted, postMessageEndpoint, session]);

    /** Commit an existing message */
    const putMessage = useCallback(async function putMessageCallback(originalMessage: ChatMessageShape, text: string) {
        if (!chat) return;
        const taskContexts = await getTaskContexts();
        // Create new version if we're in a bot-only chat. This branches the conversation and allows the bot to respond.
        if (isBotOnlyChat) {
            const message: ChatMessageCreateInput = {
                id: uuid(),
                chatConnect: chat.id,
                parentConnect: originalMessage.id,
                userConnect: getCurrentUser(session).id ?? "",
                versionIndex: originalMessage.versionIndex + 1,
                translationsCreate: [{
                    id: uuid(),
                    language,
                    text,
                }],
            };
            fetchLazyWrapper<ChatMessageCreateWithTaskInfoInput, ChatMessage>({
                fetch: postMessageEndpoint,
                inputs: {
                    chatId: chat.id,
                    message,
                    task: activeTask.task as LlmTask,
                    taskContexts,
                },
                onSuccess: messagePosted,
            });
        }
        // Otherwise, edit the existing message
        else {
            const message: ChatMessageUpdateInput = {
                id: originalMessage.id,
                translationsDelete: originalMessage.translations.map((t) => t.id),
                translationsCreate: [{
                    id: uuid(),
                    language,
                    text,
                }],
            };
            fetchLazyWrapper<ChatMessageUpdateWithTaskInfoInput, ChatMessage>({
                fetch: putMessageEndpoint,
                inputs: {
                    chatId: chat.id,
                    message,
                    task: activeTask.task as LlmTask,
                    taskContexts,
                },
                onSuccess: messagePutted,
            });
        }
    }, [activeTask.task, chat, getTaskContexts, isBotOnlyChat, language, messagePosted, messagePutted, postMessageEndpoint, putMessageEndpoint, session]);

    const [regenerate] = useLazyFetch<RegenerateResponseInput, Success>(endpointPostRegenerateResponse);
    /** Regenerate a bot response */
    const regenerateResponse = useCallback(async function regenerateResponseCallback({ id }: ChatMessageShape) {
        const taskContexts = await getTaskContexts();
        fetchLazyWrapper<RegenerateResponseInput, Success>({
            fetch: regenerate,
            inputs: {
                messageId: id,
                task: activeTask.task as LlmTask,
                taskContexts,
            },
            successCondition: (data) => data && data.success === true,
            errorMessage: () => ({ messageKey: "ActionFailed" }),
        });
    }, [activeTask.task, getTaskContexts, regenerate]);

    /** Reply to a specific message */
    const replyToMessage = useCallback(async function replyToMessageCallback(messageBeingRepliedTo: ChatMessageShape, text: string) {
        if (!chat || text.trim() === "") return;
        const taskContexts = await getTaskContexts();
        const message: ChatMessageCreateInput = {
            id: uuid(),
            chatConnect: chat.id,
            parentConnect: messageBeingRepliedTo.id,
            userConnect: getCurrentUser(session).id ?? "",
            versionIndex: 0,
            translationsCreate: [{
                id: uuid(),
                language,
                text,
            }],
        };
        fetchLazyWrapper<ChatMessageCreateWithTaskInfoInput, ChatMessage>({
            fetch: postMessageEndpoint,
            inputs: {
                chatId: chat.id,
                message,
                task: activeTask.task as LlmTask,
                taskContexts,
            },
            onSuccess: messagePosted,
        });
    }, [activeTask.task, chat, getTaskContexts, language, messagePosted, postMessageEndpoint, session]);

    /** Retry posting a failed message */
    const retryPostMessage = useCallback(async function retryPostMessageCallback(failedMessage: ChatMessageShape) {
        if (!chat) return;
        const taskContexts = await getTaskContexts();

        const { parent, versionIndex, user } = failedMessage;

        const text = getTranslation(failedMessage, [language])?.text;

        if (!text) {
            console.error("Failed to retrieve message text for retry.");
            return;
        }

        // Prepare message input
        const message: ChatMessageCreateInput = {
            id: uuid(), // Generate new id
            chatConnect: chat.id,
            parentConnect: parent?.id ?? undefined,
            userConnect: user?.id ?? getCurrentUser(session).id ?? "",
            versionIndex: versionIndex ?? 0,
            translationsCreate: [{
                id: uuid(),
                language,
                text,
            }],
        };

        fetchLazyWrapper<ChatMessageCreateWithTaskInfoInput, ChatMessage>({
            fetch: postMessageEndpoint,
            inputs: {
                chatId: chat.id,
                message,
                task: activeTask.task as LlmTask,
                taskContexts,
            },
            onSuccess: messagePosted,
            onError: () => {
                // Handle error if needed
            },
        });

    }, [activeTask.task, chat, getTaskContexts, language, messagePosted, postMessageEndpoint, session]);

    return {
        postMessage,
        putMessage,
        regenerateResponse,
        replyToMessage,
        retryPostMessage,
    };
}

type UseMessageInputProps = {
    id: string;
    languages: readonly string[];
    message: string;
    postMessage: UseMesssageActionsResult["postMessage"];
    putMessage: UseMesssageActionsResult["putMessage"];
    replyToMessage: UseMesssageActionsResult["replyToMessage"];
    setMessage: (updatedMessage: string) => unknown;
}

/**
 * Handles the logic for writing, editing, and replying to chat messages.
 */
export function useMessageInput({
    id,
    languages,
    message,
    postMessage,
    putMessage,
    replyToMessage,
    setMessage,
}: UseMessageInputProps) {

    const [messageBeingRepliedTo, setMessageBeingRepliedTo] = useHistoryState<ChatMessageShape | null>(`${id}-reply`, null);
    const startReplyingToMessage = useCallback(function startReplyingToMessageCallback(message: ChatMessageShape) {
        setMessageBeingRepliedTo(message);
    }, [setMessageBeingRepliedTo]);
    const stopReplyingToMessage = useCallback(function stopReplyingToMessageCallback() {
        setMessageBeingRepliedTo(null);
    }, [setMessageBeingRepliedTo]);

    const [messageBeingEdited, setMessageBeingEdited] = useHistoryState<ChatMessageShape | null>(`${id}-edit`, null);
    const nonEditingText = useRef<string>("");
    const startEditingMessage = useCallback(function startEditingMessageCallback(messageToEdit: ChatMessageShape) {
        setMessageBeingEdited(messageToEdit);
        // Store the original message text to revert to if the user cancels editing
        nonEditingText.current = message;
        // Change the message text to the message being edited
        setMessage(getTranslation(messageToEdit, languages).text || "");
    }, [languages, message, setMessageBeingEdited, setMessage]);
    const stopEditingMessage = useCallback(function stopEditingMessageCallback() {
        setMessageBeingEdited(null);
        setMessage(nonEditingText.current);
        nonEditingText.current = "";
    }, [setMessage, setMessageBeingEdited]);

    /**
     * Handle the submit button based on if we're replying, editing, or neither
     */
    const submitMessage = useCallback(function handleSubmitCallback(updatedMessage: string) {
        const trimmed = updatedMessage.trim();
        if (trimmed.length === 0) return;

        if (messageBeingEdited) {
            putMessage(messageBeingEdited, trimmed);
            stopEditingMessage();
        } else if (messageBeingRepliedTo) {
            replyToMessage(messageBeingRepliedTo, trimmed);
            setMessage("");
            stopReplyingToMessage();
        } else {
            postMessage(trimmed);
            setMessage("");
        }
    }, [messageBeingEdited, messageBeingRepliedTo, postMessage, putMessage, replyToMessage, setMessage, stopEditingMessage, stopReplyingToMessage]);

    return {
        message,
        messageBeingEdited,
        messageBeingRepliedTo,
        startEditingMessage,
        startReplyingToMessage,
        stopEditingMessage,
        stopReplyingToMessage,
        setMessage,
        submitMessage,
    };
}
