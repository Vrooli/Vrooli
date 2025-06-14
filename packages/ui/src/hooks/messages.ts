import { DUMMY_ID, endpointsChatMessage, getTranslation, noop, type AITaskInfo, type ChatMessage, type ChatMessageCreateInput, type ChatMessageCreateWithTaskInfoInput, type ChatMessageSearchTreeInput, type ChatMessageSearchTreeResult, type ChatMessageShape, type ChatMessageUpdateInput, type ChatMessageUpdateWithTaskInfoInput, type ChatParticipant, type ChatShape, type LlmTask, type RegenerateResponseInput, type Session, type Success, type TaskContextInfo } from "@vrooli/shared";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { getValidatedPreferredModel } from "../api/ai.js";
import { fetchLazyWrapper } from "../api/fetchWrapper.js";
import { SessionContext } from "../contexts/session.js";
import { getCurrentUser } from "../utils/authentication/session.js";
import { getUserLanguages } from "../utils/display/translationTools.js";
import { getCookieMessageTree, setCookieMessageTree, type BranchMap } from "../utils/localStorage.js";
import { PubSub } from "../utils/pubsub.js";
import { type UseChatTaskReturn } from "./tasks.js";
import { useLazyFetch } from "./useFetch.js";
import { useHistoryState } from "./useHistoryState.js";

export type MinimumChatMessage = {
    id: string;
    createdAt?: string;
    parent?: {
        id: string;
        parent?: {
            id: string;
        } | null;
        parentId?: string | null;
    } | null;
    parentId?: string | null;
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

export class MessageTree<T extends MinimumChatMessage> {
    private map: Map<string, MessageNode<T>>;
    private roots: string[];

    constructor(
        map?: Map<string, MessageNode<T>>,
        roots?: string[],
    ) {
        this.map = map ?? new Map();
        this.roots = roots ?? [];
    }

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
    findReplyMessage(
        branches: BranchMap,
        replyingMessage: Pick<ChatMessageShape, "id" | "translations" | "user">,
        participants: Pick<ChatParticipant, "id" | "user">[],
        session: Session | undefined,
    ): string | null {
        const userData = getCurrentUser(session);
        const userLanguages = getUserLanguages(session);
        const otherParticipants = participants.filter(p => p.id !== userData.id);

        let currentNode = this.map.get(replyingMessage.id);
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
            const nextNode = this.map.get(selectedChild);
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
    findSuitableParentOrSibling(orphanId: string): string | null {
        const orphan = this.map.get(orphanId);
        if (!orphan) return null;

        // First, try to attach to the direct parent if possible
        const parentId = orphan.message.parent?.id || orphan.message.parentId;
        const parentNode = parentId ? this.map.get(parentId) : null;
        if (parentNode) return parentId || null;

        // Then, try to attach to the grandparent if possible
        const grandparentId = orphan.message.parent?.parent?.id || orphan.message.parent?.parentId;
        const grandparent = grandparentId ? this.map.get(grandparentId) : null;
        if (grandparent) return grandparentId || null;

        // If no parent or grandparent, find the closest lesser sequence
        const closestBySequence = this.findClosestBySequence(orphan.message.sequence);
        if (closestBySequence) return closestBySequence;

        // If no sequence match, find the closest lesser timestamp
        const closestByTimestamp = this.findClosestByTimestamp(orphan.message.createdAt);
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
    findClosestBySequence(sequence?: number): string | null {
        if (!sequence) return null;
        let closestId: string | null = null;
        let closestSequence = -Infinity;

        this.map.forEach((node, id) => {
            const nodeSequence = node.message.sequence;
            if (nodeSequence !== undefined && nodeSequence < sequence && nodeSequence > closestSequence) {
                closestSequence = nodeSequence;
                closestId = id;
            }
        });

        return closestId;
    }

    /**
     * Finds the closest node by timestamp that is earlier than the timestamp of the given node.
     * 
     * This function searches through the message map to find the node with the closest timestamp that is still
     * earlier than the given node's timestamp. This helps in placing orphaned nodes in a logical position within
     * the message tree when sequence numbers are not available or applicable.
     */
    findClosestByTimestamp(timestamp?: string): string | null {
        if (!timestamp) return null;
        let closestId: string | null = null;
        let closestTimestamp = -Infinity;

        this.map.forEach((node, id) => {
            const nodeCreatedAt = node.message.createdAt;
            if (!nodeCreatedAt) return;

            const nodeTimestamp = new Date(nodeCreatedAt).getTime();
            const targetTimestamp = new Date(timestamp).getTime();

            if (nodeTimestamp < targetTimestamp && nodeTimestamp > closestTimestamp) {
                closestTimestamp = nodeTimestamp;
                closestId = id;
            }
        });

        return closestId;
    }

    /**
     * Processes and attempts to reattach orphaned nodes within the message tree.
     * 
     * This function iterates over all orphaned nodes and attempts to find a suitable parent or sibling for each,
     * based on a set of criteria including grandparent attachment, sequence number, and timestamp. If no suitable
     * location is found within the existing tree, the orphaned node is added as a new root node.
     */
    handleOrphanedNodes(orphanIds: string[]) {
        if (orphanIds.length === 0) return;

        // Found orphaned nodes - attempting to reattach

        // Sort orphans by sequence to ensure consistent processing order
        const sortedOrphans = [...orphanIds].sort((a, b) => {
            const nodeA = this.map.get(a);
            const nodeB = this.map.get(b);
            return (nodeA?.message.sequence ?? 0) - (nodeB?.message.sequence ?? 0);
        });

        // Process each orphan
        for (const orphanId of sortedOrphans) {
            const orphanNode = this.map.get(orphanId);
            if (!orphanNode) continue;

            // Try to find a suitable parent
            const suitableParentId = this.findSuitableParentOrSibling(orphanId);
            if (!suitableParentId) {
                // If no suitable parent found, make it a root
                if (!this.roots.includes(orphanId)) {
                    this.roots = [...this.roots, orphanId];
                    this.sortSiblings(this.roots);
                }
                continue;
            }

            // Get the suitable parent node
            const suitableParent = this.map.get(suitableParentId);
            if (!suitableParent) continue;

            // Remove from roots if it was previously a root
            if (this.roots.includes(orphanId)) {
                this.roots = this.roots.filter(id => id !== orphanId);
            }

            // Add to new parent's children if not already there
            if (!suitableParent.children.includes(orphanId)) {
                const newChildren = [...suitableParent.children, orphanId];
                this.sortSiblings(newChildren);
                this.map.set(suitableParentId, { ...suitableParent, children: newChildren });
            }
        }

        // Final sort of roots
        this.sortSiblings(this.roots);
    }

    /**
     * Sorts sibling nodes in the message tree based on their version index or, if not available, their sequence number.
     * 
     * This function is used to ensure that sibling nodes are displayed in the correct order within the message tree.
     * It first attempts to sort siblings based on their version index. If the version index is not available for comparison,
     * it falls back to sorting by sequence number.
     */
    sortSiblings(siblings: string[]) {
        siblings.sort((a, b) => {
            const aNode = this.map.get(a);
            const bNode = this.map.get(b);
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

    removeMessageFromTree(messageId: string) {
        const nodeToRemove = this.map.get(messageId);
        if (!nodeToRemove) {
            // Message not found in messageMap
            return;
        }

        // If this node has children, they need to be reattached
        const orphanedChildren = [...nodeToRemove.children];

        // Remove from parent's children
        const parentId = nodeToRemove.message.parent?.id || nodeToRemove.message.parentId;
        if (parentId) {
            const parentNode = this.map.get(parentId);
            if (parentNode) {
                // Create new children array without the removed message but with its children
                const newChildren = parentNode.children
                    .filter(childId => childId !== messageId)
                    .concat(orphanedChildren);
                this.sortSiblings(newChildren);
                this.map.set(parentId, { ...parentNode, children: newChildren });
            }
        } else {
            // Remove from roots and add children as new roots
            this.roots = this.roots
                .filter(rootId => rootId !== messageId)
                .concat(orphanedChildren);
            this.sortSiblings(this.roots);
        }

        // Remove the node from the map
        this.map.delete(messageId);
    }

    addMessageToTree(message: T) {
        const messageId = message.id;
        const parentId = message.parent?.id || message.parentId;

        // Add or update the node in the map
        if (!this.map.has(messageId)) {
            this.map.set(messageId, { message, children: [] });
        } else {
            // Update existing message data while preserving children
            const existingNode = this.map.get(messageId);
            if (existingNode) {
                this.map.set(messageId, { ...existingNode, message });
            }
        }

        // If the message has a parent
        if (parentId) {
            const parentNode = this.map.get(parentId);
            if (parentNode) {
                // Only add to parent's children if not already there
                if (!parentNode.children.includes(messageId)) {
                    const newChildren = [...parentNode.children, messageId];
                    this.sortSiblings(newChildren);
                    this.map.set(parentId, { ...parentNode, children: newChildren });
                }
                return null; // Message was properly attached to parent
            }
            return messageId; // Parent not found, mark as orphan
        } else {
            // Message is a root if it has no parent
            if (!this.roots.includes(messageId)) {
                this.roots = [...this.roots, messageId];
                this.sortSiblings(this.roots);
            }
            return null; // Message was properly added as root
        }
    }

    getMessagesCount() {
        return this.map.size;
    }

    getMap() {
        return this.map;
    }

    getRoots() {
        return this.roots;
    }

    /**
     * Adds multiple messages to the tree at once, handling orphans and maintaining tree integrity.
     * @param messages Array of messages to add to the tree
     */
    addMessagesBatch(messages: T[]) {
        if (!messages || messages.length === 0) return;

        // Sort messages by sequence to ensure consistent processing order
        const sortedMessages = [...messages].sort((a, b) =>
            (a.sequence ?? 0) - (b.sequence ?? 0),
        );

        // First pass: add all messages to the map
        const orphans: string[] = [];
        for (const message of sortedMessages) {
            // Remove existing message if it exists
            if (this.map.has(message.id)) {
                this.removeMessageFromTree(message.id);
            }

            // Add message and track if it's an orphan
            const orphanId = this.addMessageToTree(message);
            if (orphanId) {
                orphans.push(orphanId);
            }
        }

        // Second pass: handle orphans
        if (orphans.length > 0) {
            // Try to handle orphans multiple times as some might depend on others
            let remainingOrphans = [...orphans];
            const maxAttempts = 3;
            let attempts = 0;

            while (remainingOrphans.length > 0 && attempts < maxAttempts) {
                const orphansToProcess = [...remainingOrphans];
                remainingOrphans = [];

                for (const orphanId of orphansToProcess) {
                    // Try to add the orphan again
                    const message = this.map.get(orphanId)?.message;
                    if (!message) continue;

                    const stillOrphan = this.addMessageToTree(message);
                    if (stillOrphan) {
                        remainingOrphans.push(orphanId);
                    }
                }
                attempts++;
            }

            // Handle any remaining orphans using findSuitableParentOrSibling
            if (remainingOrphans.length > 0) {
                this.handleOrphanedNodes(remainingOrphans);
            }
        }
    }

    /**
     * Edits the content of a message in the tree. DOES NOT 
     * add a new version of the message. This method is useful when 
     * chatting with other participants, where branching is disabled.
     * @param updatedMessage The updated message data
     * @returns true if the edit was successful, false otherwise
     */
    editMessage(updatedMessage: Partial<T> & { id: string }): boolean {
        if (!this.map.has(updatedMessage.id)) {
            // Message not found in messageMap - cannot edit
            return false;
        }

        const node = this.map.get(updatedMessage.id);
        if (!node) {
            // Message not found in messageMap - cannot edit
            return false;
        }

        const updatedNode = {
            ...node,
            message: { ...node.message, ...updatedMessage },
        };

        this.map.set(updatedMessage.id, updatedNode as MessageNode<T>);
        return true;
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
    const [getTreeData, { data: searchTreeData, loading: isTreeLoading }] = useLazyFetch<ChatMessageSearchTreeInput, ChatMessageSearchTreeResult>(endpointsChatMessage.findTree);

    // The message tree structure
    const [tree, setTree] = useState<MessageTree<ChatMessageShape>>(new MessageTree<ChatMessageShape>());

    /**
     * Clears all messages from the tree, resetting the state to its initial state.
     */
    const clearMessages = useCallback(() => {
        setTree(new MessageTree<ChatMessageShape>());
    }, []);

    // Which branches of the tree are in view
    // Initialize from cookie if chatId is present
    const initialBranches = useMemo(() => {
        return chatId ? getCookieMessageTree(chatId)?.branches ?? {} : {};
    }, [chatId]);
    const [branches, setBranches] = useState<BranchMap>(initialBranches);

    // Function to update branches state and cookie, including locationId
    const updateBranchesAndLocation = useCallback((parentId: string, newActiveChildId: string) => {
        setBranches(prevBranches => {
            const newBranches = {
                ...prevBranches,
                [parentId]: newActiveChildId,
            };
            // Update the cookie with current branches and the new locationId
            if (chatId) {
                setCookieMessageTree(chatId, { branches: newBranches, locationId: newActiveChildId });
            }
            return newBranches;
        });
    }, [chatId]);

    const addMessages = useCallback((newMessages: ChatMessageShape[]) => {
        if (!newMessages || newMessages.length === 0) return;
        setTree((prevTree) => {
            const newTree = new MessageTree<ChatMessageShape>(
                new Map(prevTree.getMap()),
                [...prevTree.getRoots()],
            );
            newTree.addMessagesBatch(newMessages);
            return newTree;
        });
    }, []);

    const removeMessages = useCallback((messageIds: string[]) => {
        setTree((prevTree) => {
            const newTree = new MessageTree<ChatMessageShape>(
                new Map(prevTree.getMap()),
                [...prevTree.getRoots()],
            );

            messageIds.forEach(messageId => {
                newTree.removeMessageFromTree(messageId);
            });

            return newTree;
        });
    }, []);

    const editMessage = useCallback((updatedMessage: (Partial<ChatMessageShape> & { id: string })) => {
        setTree((prevTree) => {
            const newTree = new MessageTree<ChatMessageShape>(
                new Map(prevTree.getMap()),
                [...prevTree.getRoots()],
            );

            if (!newTree.editMessage(updatedMessage)) {
                return prevTree;
            }

            return newTree;
        });
    }, []);

    // When chatId changes, clear everything, read cookie, and fetch new data
    useEffect(() => {
        clearMessages();
        let startId: string | undefined = undefined;
        let initialBranchesFromCookie: BranchMap = {};

        if (chatId && chatId !== DUMMY_ID) {
            const cookieData = getCookieMessageTree(chatId);
            startId = cookieData?.locationId;
            initialBranchesFromCookie = cookieData?.branches ?? {};
            // Getting tree data for chat
            getTreeData({ chatId, startId });
        }
        setBranches(initialBranchesFromCookie);

    }, [chatId, clearMessages, getTreeData]);

    // Add fetched messages to the tree
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
        removeMessages,
        // setBranches, // Expose updateBranchesAndLocation instead
        updateBranchesAndLocation,
        tree,
    };
}

/**
 * How long to wait to collect task context before timing out
 */
const GET_TASK_CONTEXT_TIMEOUT_MS = 500;

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
    const model = getValidatedPreferredModel();

    const [postMessageEndpoint] = useLazyFetch<ChatMessageCreateWithTaskInfoInput, ChatMessage>(endpointsChatMessage.createOne);
    const [putMessageEndpoint] = useLazyFetch<ChatMessageUpdateWithTaskInfoInput, ChatMessage>(endpointsChatMessage.updateOne);

    /** Collects context data for the active task */
    const collectTaskContext = useCallback((taskInfo: AITaskInfo | null) => new Promise<TaskContextInfo | null>((resolve, reject) => {
        if (!chat?.id) {
            reject(new Error("Chat ID not found"));
            return;
        }
        // Only try to collect data from `useAutoFill`, which provides the current form's data as context
        if (!PubSub.get().hasSubscribers("requestTaskContext", (metadata) => metadata?.source === "useAutoFill")) {
            resolve(null);
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
                if (!context) {
                    return [...contexts];
                }
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
                return [...contexts];
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
            id: DUMMY_ID,
            chatConnect: chat.id,
            language,
            userConnect: getCurrentUser(session).id ?? "",
            text,
            versionIndex: 0,
        };
        fetchLazyWrapper<ChatMessageCreateWithTaskInfoInput, ChatMessage>({
            fetch: postMessageEndpoint,
            inputs: {
                message,
                model,
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
                id: DUMMY_ID,
                chatConnect: chat.id,
                language,
                parentConnect: originalMessage.id,
                text,
                userConnect: getCurrentUser(session).id ?? "",
                versionIndex: originalMessage.versionIndex + 1,
            };
            fetchLazyWrapper<ChatMessageCreateWithTaskInfoInput, ChatMessage>({
                fetch: postMessageEndpoint,
                inputs: {
                    message,
                    model,
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
                text,
            };
            fetchLazyWrapper<ChatMessageUpdateWithTaskInfoInput, ChatMessage>({
                fetch: putMessageEndpoint,
                inputs: {
                    message,
                    model,
                    task: activeTask.task as LlmTask,
                    taskContexts,
                },
                onSuccess: messagePutted,
            });
        }
    }, [activeTask.task, chat, getTaskContexts, isBotOnlyChat, language, messagePosted, messagePutted, postMessageEndpoint, putMessageEndpoint, session]);

    const [regenerate] = useLazyFetch<RegenerateResponseInput, Success>(endpointsChatMessage.regenerateResponse);
    /** Regenerate a bot response */
    const regenerateResponse = useCallback(async function regenerateResponseCallback({ id }: ChatMessageShape) {
        const taskContexts = await getTaskContexts();
        fetchLazyWrapper<RegenerateResponseInput, Success>({
            fetch: regenerate,
            inputs: {
                messageId: id,
                model,
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
            id: DUMMY_ID,
            chatConnect: chat.id,
            language,
            parentConnect: messageBeingRepliedTo.id,
            text,
            userConnect: getCurrentUser(session).id ?? "",
            versionIndex: 0,
        };
        fetchLazyWrapper<ChatMessageCreateWithTaskInfoInput, ChatMessage>({
            fetch: postMessageEndpoint,
            inputs: {
                message,
                model,
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

        const { parent, parentId, versionIndex, user } = failedMessage;

        const text = getTranslation(failedMessage, [language])?.text;

        if (!text) {
            // Failed to retrieve message text for retry
            return;
        }

        // Prepare message input
        const message: ChatMessageCreateInput = {
            id: DUMMY_ID,
            chatConnect: chat.id,
            language,
            parentConnect: parent?.id ?? parentId ?? undefined,
            text,
            userConnect: user?.id ?? getCurrentUser(session).id ?? "",
            versionIndex: versionIndex ?? 0,
        };

        fetchLazyWrapper<ChatMessageCreateWithTaskInfoInput, ChatMessage>({
            fetch: postMessageEndpoint,
            inputs: {
                message,
                model,
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
    setMessage?: (updatedMessage: string) => unknown;
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
        setMessage?.(getTranslation(messageToEdit, languages).text || "");
    }, [languages, message, setMessageBeingEdited, setMessage]);
    const stopEditingMessage = useCallback(function stopEditingMessageCallback() {
        setMessageBeingEdited(null);
        setMessage?.(nonEditingText.current);
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
            setMessage?.("");
            stopReplyingToMessage();
        } else {
            postMessage(trimmed);
            setMessage?.("");
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
