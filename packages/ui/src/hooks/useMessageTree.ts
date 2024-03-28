import { ChatMessageSearchTreeInput, ChatMessageSearchTreeResult, ChatParticipant, DUMMY_ID, LlmTaskInfo, Session, endpointGetChatMessageTree } from "@local/shared";
import { SessionContext } from "contexts/SessionContext";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { getCurrentUser } from "utils/authentication/session";
import { BranchMap, getCookieMessageTree, getCookieTasksForMessage, setCookieMessageTree } from "utils/cookies";
import { getTranslation, getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { ChatMessageShape } from "utils/shape/models/chatMessage";
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
    versionIndex?: number;
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
export const findReplyMessage = (
    tree: MessageTree<MinimumChatMessage>,
    branches: BranchMap,
    replyingMessage: Pick<ChatMessageShape, "id" | "translations" | "user">,
    participants: Pick<ChatParticipant, "id" | "user">[],
    session: Session | undefined,
): string | null => {
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
};

/**
 * Attempts to find a suitable parent or sibling for an orphaned node.
 * 
 * This function first tries to attach the orphaned node to its grandparent, if possible. If the grandparent
 * does not exist or is not suitable, it then attempts to find the closest node by sequence number, and if that
 * fails, by timestamp. If no suitable node is found by any of these criteria, the orphaned node will remain a root.
 * 
 * NOTE: This may mess up versioned root messages. May need to add a db field later
 */
const findSuitableParentOrSibling = (
    messageMap: Map<string, MessageNode<MinimumChatMessage>>,
    orphanId: string,
): string | null => {
    const orphan = messageMap.get(orphanId);
    if (!orphan) return null;
    // First, try to attach to grandparent if possible
    const grandparentId = orphan.message.parent?.parent?.id;
    const grandparent = messageMap.get(grandparentId ?? "");
    if (grandparent) return grandparent.message.id;
    // If no grandparent, find the closest lesser sequence
    const closestBySequence = findClosestBySequence(messageMap, orphan.message.sequence);
    if (closestBySequence) return closestBySequence;
    // If no sequence match, find the closest lesser timestamp
    const closestByTimestamp = findClosestByTimestamp(messageMap, orphan.message.created_at);
    if (closestByTimestamp) return closestByTimestamp;
    // If all else fails, the node will remain a root
    return null;
};

/**
 * Finds the closest node by sequence number that is less than the sequence number of the given node.
 * 
 * This function iterates through all nodes in the message map and identifies the node with the closest lesser
 * sequence number compared to the provided sequence number. This is used to place orphaned nodes in a logical
 * position within the message tree based on their sequence.
 */
const findClosestBySequence = (
    messageMap: Map<string, MessageNode<MinimumChatMessage>>,
    sequence?: number,
): string | null => {
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
};

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
const findClosestByTimestamp = (
    messageMap: Map<string, MessageNode<MinimumChatMessage>>,
    timestamp?: string,
): string | null => {
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
};

/**
 * Processes and attempts to reattach orphaned nodes within the message tree.
 * 
 * This function iterates over all orphaned nodes and attempts to find a suitable parent or sibling for each,
 * based on a set of criteria including grandparent attachment, sequence number, and timestamp. If no suitable
 * location is found within the existing tree, the orphaned node is added as a new root node.
 */
const handleOrphanedNodes = (
    messageMap: Map<string, MessageNode<MinimumChatMessage>>,
    roots: string[],
    orphanIds: string[],
) => {
    if (orphanIds.length === 0) return;

    console.warn(`Found ${orphanIds.length} orphaned nodes. Attempting to reattach.`);

    const reattachedOrphans = new Set<string>();

    orphanIds.forEach(orphanId => {
        const suitableParentId = findSuitableParentOrSibling(messageMap, orphanId);
        const suitableParent = suitableParentId ? messageMap.get(suitableParentId) : null;
        if (suitableParent) {
            suitableParent.children.push(orphanId);
            reattachedOrphans.add(orphanId);
        } else if (!roots.includes(orphanId)) {
            roots.push(orphanId);
        }
    });

    // Remove reattached orphans from the list of orphanIds and rootIds if necessary
    reattachedOrphans.forEach(id => {
        const index = orphanIds.indexOf(id);
        if (index > -1) orphanIds.splice(index, 1);

        const rootIndex = roots.indexOf(id);
        if (rootIndex > -1) roots.splice(rootIndex, 1);
    });
};

/**
 * Sorts sibling nodes in the message tree based on their version index or, if not available, their sequence number.
 * 
 * This function is used to ensure that sibling nodes are displayed in the correct order within the message tree.
 * It first attempts to sort siblings based on their version index. If the version index is not available for comparison,
 * it falls back to sorting by sequence number.
 */
const sortSiblings = (
    map: Map<string, MessageNode<MinimumChatMessage>>,
    siblings: string[],
) => {
    siblings.sort((a, b) => {
        const aNode = map.get(a);
        const bNode = map.get(b);
        if (!aNode || !bNode) return 0; // If either node is not found, return 0 (no change)

        // Check if both messages have a versionIndex
        const hasVersionIndexA = aNode.message.versionIndex !== undefined;
        const hasVersionIndexB = bNode.message.versionIndex !== undefined;

        // Prefer comparing versionIndex if both messages have it
        if (hasVersionIndexA && hasVersionIndexB) {
            return (aNode.message.versionIndex ?? 0) - (bNode.message.versionIndex ?? 0);
        }
        // Fallback to less reliable sequence comparison
        else {
            return (aNode.message.sequence ?? 0) - (bNode.message.sequence ?? 0);
        }
    });
};

export const useMessageTree = (chatId: string) => {
    const session = useContext(SessionContext);

    // We query messages separate from the chat, since we must traverse the message tree
    const [getTreeData, { data: searchTreeData, loading: isTreeLoading }] = useLazyFetch<ChatMessageSearchTreeInput, ChatMessageSearchTreeResult>(endpointGetChatMessageTree);

    // The message tree structure
    const [tree, setTree] = useState<MessageTree<ChatMessageShape>>({ map: new Map(), roots: [] });
    const messagesCount = useMemo(() => tree.map.size, [tree.map.size]);

    // Tasks suggested or ran by messages
    const [messageTasks, setMessageTasks] = useState<Record<string, LlmTaskInfo[]>>({});
    const updateTasksForMessage = useCallback((messageId: string, tasks: LlmTaskInfo[]) => {
        setMessageTasks(prevTasks => {
            prevTasks[messageId] = tasks;
            return prevTasks;
        });
    }, []);

    /**
     * Clears all messages from the tree, resetting the state to its initial state.
     */
    const clearMessages = useCallback(() => {
        setTree({ map: new Map(), roots: [] });
    }, []);

    // Which branches of the tree are in view
    const [branches, setBranches] = useState<BranchMap>(getCookieMessageTree(chatId)?.branches ?? {});

    useEffect(() => {
        // Update the cookie with current branches
        setCookieMessageTree(chatId, { branches, locationId: "someLocationId" }); // TODO locationId should be last chat message in view
    }, [branches, chatId]);

    const addMessages = useCallback((newMessages: ChatMessageShape[]) => {
        if (!newMessages || newMessages.length === 0) return;
        setMessageTasks(prevTasks => {
            setTree(({ map: prevMessageMap, roots: prevRoots }) => {
                const map = new Map(prevMessageMap);
                const roots = [...prevRoots];
                const orphans: string[] = [];

                newMessages.forEach(message => {
                    if (map.has(message.id)) {
                        console.warn(`Message ${message.id} already exists in messageMap`);
                    } else {
                        map.set(message.id, { message, children: [] });
                    }
                    // Also find tasks associated with the message
                    const tasks = getCookieTasksForMessage(message.id);
                    if (tasks) {
                        prevTasks[message.id] = tasks.tasks;
                    }
                });

                newMessages.forEach(message => {
                    if (message.parent?.id) {
                        const parentNode = map.get(message.parent.id);
                        if (parentNode) {
                            // Since all edited versions have the same parentId and are not nested,
                            // they will be siblings in the tree structure.
                            parentNode.children.push(message.id);
                            // Sort children to ensure the correct order of message versions
                            sortSiblings(map, parentNode.children);
                        } else {
                            // This may be a root node, or it chould be an orphaned node 
                            orphans.push(message.id);
                        }
                    } else {
                        // If there's no parent ID, we'll assume it's a root node
                        roots.push(message.id);
                    }
                });

                // Attempt to reattach orphaned nodes.
                handleOrphanedNodes(map, roots, orphans);

                // Sort roots
                sortSiblings(map, roots);

                console.log("at the end of addMessages", prevTasks);
                return { map, roots };
            });
            return prevTasks;
        });
    }, []);

    const removeMessages = useCallback((messageIds: string[]) => {
        setMessageTasks(prevTasks => {
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

                    // Update tasks
                    if (prevTasks[messageId]) {
                        delete prevTasks[messageId];
                    }
                });

                return { map, roots };
            });
            return prevTasks;
        });
    }, []);

    /**
     * Edits the content of a message in the tree. DOES NOT 
     * add a new version of the message. This method is useful when 
     * chatting with other participants, where branching is disabled.
     */
    const editMessage = useCallback((updatedMessage: ChatMessageShape) => {
        setTree(({ map: prevMessageMap, roots }) => {
            if (!prevMessageMap.has(updatedMessage.id)) {
                console.error(`Message ${updatedMessage.id} not found in messageMap. Cannot edit.`);
                return { map: prevMessageMap, roots }; // Return the previous state if the message is not found
            }

            // Create a new map for immutability
            const map = new Map(prevMessageMap);

            // Retrieve the node and update it with the new message data
            const node = map.get(updatedMessage.id);
            const updatedNode = { ...node, message: updatedMessage };

            map.set(updatedMessage.id, updatedNode as MessageNode<ChatMessageShape>); // Update the map with the edited message node
            return { map, roots };
        });
    }, []);

    /**
     * Triggers a message reply, either by editing an existing message or creating a new one.
     */
    const replyToMessage = useCallback((message: ChatMessageShape) => {
        // // Determine if we should edit an existing message or create a new one
        // const existingMessageId = findTargetMessage(tree, branches, message, session);
        // // If editing an existing message, send pub/sub
        // if (existingMessageId) {
        //     PubSub.get().publish("chatMessageEdit", existingMessageId);
        //     return;
        // }
        // // Otherwise, set the message input to "@[handle_of_user_youre_replying_to] "
        // else {
        //     PubSub.get().publish("chatMessageEdit", false);
        //     setMessage(existingText => `@[${message.user?.name}] ${existingText}`);
        // }
        //TODO fix and probably put in useMessageActions
    }, [branches, session, tree]);

    // When chatId changes, clear everything and fetch new data
    useEffect(() => {
        if (chatId === DUMMY_ID) return;
        clearMessages();
        setBranches({});
        setMessageTasks({});
        getTreeData({ chatId });
    }, [chatId, clearMessages, getTreeData]);
    useEffect(() => {
        if (!searchTreeData || searchTreeData.messages.length === 0) return;
        addMessages(searchTreeData.messages.map(message => ({ ...message, status: "sent" })));
    }, [addMessages, searchTreeData]);

    // Listen to message changes and update the tree accordingly
    useEffect(() => {
        const unsubscribe = PubSub.get().subscribe("chatMessage", (data) => {
            if (data.chatId !== chatId) return;
            const { updatedMessage } = data.data;
            if (!updatedMessage) return;
            editMessage(updatedMessage);
        });
        return unsubscribe;
    }, [chatId, editMessage]);

    // Return the necessary state and functions
    return {
        addMessages,
        branches,
        clearMessages,
        editMessage,
        isTreeLoading,
        messagesCount,
        messageTasks,
        removeMessages,
        replyToMessage,
        setBranches,
        tree,
        updateTasksForMessage,
    };
};
