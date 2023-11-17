import { useCallback, useEffect, useMemo, useState } from "react";
import { useStableObject } from "./useStableObject";

// TODO should query chat messages from the bottom up, using a query like this:
// const lastMessage = await prisma.chat_message.findFirst({
//     where: { chatId: yourChatId },
//     orderBy: { sequence: 'desc' },
//     include: {
//       parent: true,
//       // ... include any other related data you need
//     }
//   });
// TODO should query messages all at once if there are less than k in the chat, otherwise batch
// TODO preMap for chat messages should include info needed for creating chats. Since multiple chats
// can be created in one go, the pre shape function needs to query the latest chat message and create a map 
// of new chat ids to their parent ids. Also need way for messages with versionOfId set, needs to find parent
// of versionOfId's parent.
// TODO need to check validation logic to see if it's correct
// TODO when multiple bots are responding to a message, they need to set parent IDs correctly
// TODO UI should store last seen message, and load from there the next time the chat is opened. This way, 
// the user doesn't lose their place

export type MinimumChatMessage = {
    id: string;
    created_at?: string;
    parent?: {
        id: string;
        parent?: {
            id: string;
        } | null;
    } | null;
    versionIndex?: number;
    sequence?: number;
};

/** Tree structure for displaying chat messages in the correct order */
export type MessageNode<T extends MinimumChatMessage> = {
    message: T;
    /** When there is more than one child, there are multiple versions (edits) of the message */
    children: MessageNode<T>[];
};

// Helper functions for repairing invalid message trees
const findSuitableParentOrSibling = (newMessageMap: Map<string, MessageNode<MinimumChatMessage>>, orphan: MessageNode<MinimumChatMessage>) => {
    // First, try to attach to grandparent if possible
    const grandparentId = orphan.message.parent?.parent?.id;
    if (grandparentId && newMessageMap.has(grandparentId)) {
        return newMessageMap.get(grandparentId);
    }
    // If no grandparent, find the closest lesser sequence
    const closestBySequence = findClosestBySequence(newMessageMap, orphan.message.sequence);
    if (closestBySequence) return closestBySequence;
    // If no sequence match, find the closest lesser timestamp
    const closestByTimestamp = findClosestByTimestamp(newMessageMap, orphan.message.created_at);
    if (closestByTimestamp) return closestByTimestamp;
    // If all else fails, the node will remain a root
    return null;
};
const findClosestBySequence = (newMessageMap: Map<string, MessageNode<MinimumChatMessage>>, sequence?: number) => {
    let closestNode: MessageNode<MinimumChatMessage> | null = null;
    let closestSequence = -Infinity;
    if (!sequence) return null;
    newMessageMap.forEach(node => {
        if (node.message.sequence && node.message.sequence < sequence && node.message.sequence > closestSequence) {
            closestSequence = node.message.sequence;
            closestNode = node;
        }
    });
    return closestNode;
};
const findClosestByTimestamp = (newMessageMap: Map<string, MessageNode<MinimumChatMessage>>, timestamp?: string) => {
    let closestNode: MessageNode<MinimumChatMessage> | null = null;
    let closestTimestamp = -Infinity;
    if (!timestamp) return null;
    newMessageMap.forEach(node => {
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
const handleOrphanedNodes = (newMessageMap: Map<string, MessageNode<MinimumChatMessage>>, newRoots: MessageNode<MinimumChatMessage>[], orphanedNodes: MessageNode<MinimumChatMessage>[]) => {
    if (orphanedNodes.length === 0) return;
    console.warn(`Found ${orphanedNodes.length} orphaned nodes. This is likely due to the server not reparing the message tree correctly when a message is deleted.`);
    orphanedNodes.forEach(orphan => {
        const suitableParent = findSuitableParentOrSibling(newMessageMap, orphan);
        if (suitableParent) {
            suitableParent.children.push(orphan);
        } else {
            newRoots.push(orphan);
        }
    });
};

const sortSiblings = (siblings: MessageNode<MinimumChatMessage>[]) => {
    siblings.sort((a, b) => {
        // Check if both messages have a versionIndex
        const hasVersionIndexA = a.message.versionIndex !== undefined;
        const hasVersionIndexB = b.message.versionIndex !== undefined;

        // Prefer comparing versionIndex if both messages have it
        if (hasVersionIndexA && hasVersionIndexB) {
            return (a.message.versionIndex ?? 0) - (b.message.versionIndex ?? 0);
        }
        // Fallback to less reliable sequence comparison
        else {
            return (a.message.sequence ?? 0) - (b.message.sequence ?? 0);
        }
    });
};


export const useMessageTree = <T extends MinimumChatMessage>(initialMessages: T[]) => {
    // const [messageMap, setMessageMap] = useState<Map<string, MessageNode<T>>>(new Map());
    // const [roots, setRoots] = useState<MessageNode<T>[]>([]);
    const [tree, setTree] = useState({ map: new Map<string, MessageNode<T>>(), roots: [] as MessageNode<T>[] });

    const messagesCount = useMemo(() => tree.map.size, [tree.map.size]);

    // Memoize the initial messages to prevent unnecessary useEffect triggers
    const stableInitialMessages = useStableObject(initialMessages);

    /**
     * Constructs a hierarchical tree representation of the message threads.
     * 
     * The method iterates through the array of messages. For each message:
     * 1. It retrieves the corresponding node from the messageMap.
     * 2. If the message has a parentId, it finds the parent node and adds the current message node as a child of the parent node.
     *    All edited versions of a message, having the same parentId, will become siblings in the tree structure.
     * 3. If the message doesn't have a parentId, it's treated as a root node and added to the roots array.
     * 
     * This way, a hierarchical tree structure is built which represents the message threads, including handling multiple versions (edits) of messages.
     */
    useEffect(() => {
        // Initialize data structures
        const newMap = new Map<string, MessageNode<T>>();
        const newRoots: MessageNode<T>[] = [];
        const orphanedNodes: MessageNode<T>[] = [];

        // Add message nodes to the map first
        for (const message of stableInitialMessages) {
            newMap.set(message.id, { message, children: [] });
        }
        // Build the tree
        for (const message of stableInitialMessages) {
            const node = newMap.get(message.id);
            if (!node) {
                console.error(`Message ${message.id} not found in messageMap`);
                continue;
            }
            if (message.parent?.id) {
                const parentNode = newMap.get(message.parent.id);
                if (parentNode) {
                    // Since all edited versions have the same parentId and are not nested,
                    // they will be siblings in the tree structure.
                    parentNode.children.push(node);
                    // Sort children to ensure the correct order of message versions
                    sortSiblings(parentNode.children);
                } else {
                    // This may be a root node, or it chould be an orphaned node 
                    orphanedNodes.push(node);
                }
            } else {
                // If there's no parent ID, we'll assume it's a root node
                newRoots.push(node);
            }
        }

        // Attempt to reattach orphaned nodes. 
        handleOrphanedNodes(newMap, newRoots, orphanedNodes);

        // Sort roots
        sortSiblings(newRoots);

        // Set the state
        setTree({ map: newMap, roots: newRoots });
    }, [stableInitialMessages]);

    const addMessages = useCallback((newMessages: T[]) => {
        setTree(({ map: prevMessageMap, roots: prevRoots }) => {
            const newMap = new Map(prevMessageMap);
            const newRoots: MessageNode<T>[] = [];
            const orphanedNodes: MessageNode<T>[] = [];

            // Add message nodes to the map first, if they don't already exist
            newMessages.forEach(message => {
                if (newMap.has(message.id)) {
                    console.warn(`Message ${message.id} already exists in messageMap`);
                } else {
                    newMap.set(message.id, { message, children: [] });
                }
            });

            newMessages.forEach(message => {
                const node = newMap.get(message.id);
                if (!node) {
                    console.error(`Message ${message.id} not found in messageMap`);
                    return;
                }

                if (message.parent?.id) {
                    const parentNode = newMap.get(message.parent.id);
                    if (parentNode) {
                        // Since all edited versions have the same parentId and are not nested,
                        // they will be siblings in the tree structure.
                        parentNode.children.push(node);
                        // Sort children to ensure the correct order of message versions
                        sortSiblings(parentNode.children);
                    } else {
                        // This may be a root node, or it chould be an orphaned node 
                        orphanedNodes.push(node);
                    }
                } else {
                    // If there's no parent ID, we'll assume it's a root node
                    newRoots.push(node);
                }
            });

            // Attempt to reattach orphaned nodes.
            handleOrphanedNodes(newMap, newRoots, orphanedNodes);

            return { map: newMap, roots: [...prevRoots, ...newRoots] };
        });
    }, []);

    const removeMessages = useCallback((messageIds: string[]) => {
        setTree(({ map: prevMessageMap, roots: prevRoots }) => {
            const newMap = new Map(prevMessageMap);
            const newRoots = [...prevRoots]; // Copy roots to avoid direct mutation
            let updatedRoots = false;

            messageIds.forEach(messageId => {
                const nodeToRemove = newMap.get(messageId);
                if (!nodeToRemove) {
                    console.error(`Message ${messageId} not found in messageMap`);
                    return;
                }

                newMap.delete(messageId); // Remove the node from the messageMap

                if (nodeToRemove.message.parent?.id) {
                    const parentNode = newMap.get(nodeToRemove.message.parent.id);
                    if (parentNode) {
                        parentNode.children = parentNode.children.filter(child => child.message.id !== messageId);
                        parentNode.children.push(...nodeToRemove.children);
                    }
                } else {
                    updatedRoots = true;
                    // Efficiently filter out the root node
                    for (let i = 0; i < newRoots.length; i++) {
                        if (newRoots[i].message.id === messageId) {
                            newRoots.splice(i, 1);
                            break;
                        }
                    }
                }
            });

            return { map: newMap, roots: updatedRoots ? newRoots : prevRoots };
        });
    }, []);

    /**
     * Edits the content of a message in the tree. DOES NOT 
     * add a new version of the message. This method is useful when 
     * chatting with other participants, where branching is disabled.
     */
    const editMessage = useCallback((updatedMessage: T) => {
        setTree(({ map: prevMessageMap, roots }) => {
            if (!prevMessageMap.has(updatedMessage.id)) {
                console.error(`Message ${updatedMessage.id} not found in messageMap. Cannot edit.`);
                return { map: prevMessageMap, roots }; // Return the previous state if the message is not found
            }

            // Create a new map for immutability
            const newMap = new Map(prevMessageMap);

            // Retrieve the node and update it with the new message data
            const node = newMap.get(updatedMessage.id);
            const updatedNode = { ...node, message: updatedMessage };

            newMap.set(updatedMessage.id, updatedNode as MessageNode<T>); // Update the map with the edited message node
            return { map: newMap, roots };
        });
    }, []);

    /**
     * Clears all messages from the tree, resetting the state to its initial state.
     */
    const clearMessages = useCallback(() => {
        setTree({ map: new Map(), roots: [] });
    }, []);

    // Return the necessary state and functions
    return {
        tree,
        messagesCount,
        addMessages,
        removeMessages,
        editMessage,
        clearMessages,
    };
};
