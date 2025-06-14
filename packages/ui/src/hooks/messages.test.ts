import { renderHook } from "@testing-library/react";
import { DUMMY_ID, type ChatMessage } from "@vrooli/shared";
import { describe, it, expect, beforeAll, afterAll, beforeEach } from "vitest";
import { act } from "react";

import { MessageTree, useMessageTree, type MessageNode, type MinimumChatMessage } from "./messages.js";

const MessageTreeProperty = {
    map: "map",
    roots: "roots",
};

/** First test case: Result should have messages in order from ID 1 to 10, each with node having a single child */
const case1: MinimumChatMessage[] = [
    {
        id: "3",
        parent: { id: "2" },
        translations: [{
            id: "1001",
            language: "en",
            text: "Certainly! What seems to be the issue?",
        }],
        createdAt: "2021-10-01T02:00:00Z",
        sequence: 2,
        versionIndex: 0,
    },
    {
        id: "1",
        parent: null,
        translations: [{
            id: "1001",
            language: "en",
            text: "Hello, how can I help you?",
        }],
        createdAt: "2021-10-01T00:00:00Z",
        sequence: 0,
        versionIndex: 0,
    },
    {
        id: "2",
        parent: { id: "1" },
        translations: [{
            id: "1001",
            language: "en",
            text: "I need assistance with my account.",
        }],
        createdAt: "2021-10-01T01:00:00Z",
        sequence: 1,
        versionIndex: 0,
    },
    {
        id: "4",
        parent: { id: "3" },
        translations: [{
            id: "1001",
            language: "en",
            text: "I can't access my account.",
        }],
        createdAt: "2021-10-01T03:00:00Z",
        sequence: 3,
        versionIndex: 0,
    },
    {
        id: "5",
        parent: { id: "4" },
        translations: [{
            id: "1001",
            language: "en",
            text: "I see. Have you tried resetting your password?",
        }],
        createdAt: "2021-10-01T04:00:00Z",
        sequence: 4,
        versionIndex: 0,
    },
    {
        id: "6",
        parent: { id: "5" },
        translations: [{
            id: "1001",
            language: "en",
            text: "Yes, but I didn't receive the reset email.",
        }],
        createdAt: "2021-10-01T05:00:00Z",
        sequence: 5,
        versionIndex: 0,
    },
    {
        id: "7",
        parent: { id: "6" },
        translations: [{
            id: "1001",
            language: "en",
            text: "I can help with that. Can you provide your username?",
        }],
        createdAt: "2021-10-01T06:00:00Z",
        sequence: 6,
        versionIndex: 0,
    },
    {
        id: "8",
        parent: { id: "7" },
        translations: [{
            id: "1001",
            language: "en",
            text: "My username is johndoe123.",
        }],
        createdAt: "2021-10-01T07:00:00Z",
        sequence: 7,
        versionIndex: 0,
    },
    {
        id: "9",
        parent: { id: "8" },
        translations: [{
            id: "1001",
            language: "en",
            text: "Thank you, John. I have sent a password reset link to your email.",
        }],
        createdAt: "2021-10-01T08:00:00Z",
        sequence: 8,
        versionIndex: 0,
    },
    {
        id: "10",
        parent: { id: "9" },
        translations: [{
            id: "1001",
            language: "en",
            text: "Got it. Thanks!",
        }],
        createdAt: "2021-10-01T09:00:00Z",
        sequence: 9,
        versionIndex: 0,
    },
];

/** Second test case: 2 roots */
const case2: MinimumChatMessage[] = [
    {
        id: "1",
        parent: null,
        translations: [{
            id: "2001",
            language: "en",
            text: "Welcome to our platform!",
        }],
        createdAt: "2021-10-02T00:00:00Z",
        sequence: 0,
        versionIndex: 0,
    },
    {
        id: "2",
        parent: null,
        translations: [{
            id: "2002",
            language: "en",
            text: "Hello, how can I assist you?",
        }],
        createdAt: "2021-10-02T01:00:00Z",
        sequence: 1,
        versionIndex: 0,
    },
    {
        id: "3",
        parent: { id: "2" },
        translations: [{
            id: "2003",
            language: "en",
            text: "I'm having trouble logging in.",
        }],
        createdAt: "2021-10-02T02:00:00Z",
        sequence: 2,
        versionIndex: 0,
    },
    {
        id: "4",
        parent: { id: "3" },
        translations: [{
            id: "2004",
            language: "en",
            text: "I see. Have you tried resetting your password?",
        }],
        createdAt: "2021-10-02T03:00:00Z",
        sequence: 3,
        versionIndex: 0,
    },
];

/** Third test case: 1 root, 1 child, 3 grandchildren, 6 great-grandchildren (2 for each grandchild) */
const case3: MinimumChatMessage[] = [
    {
        id: "1",
        parent: null,
        translations: [{
            id: "3001",
            language: "en",
            text: "Hello! How can I assist you today?",
        }],
        createdAt: "2021-10-03T00:00:00Z",
        sequence: 0,
        versionIndex: 0,
    },
    {
        id: "2",
        parent: { id: "1" },
        translations: [{
            id: "3002",
            language: "en",
            text: "I have multiple questions.",
        }],
        createdAt: "2021-10-03T01:00:00Z",
        sequence: 1,
        versionIndex: 0,
    },
    {
        id: "3",
        parent: { id: "2" },
        translations: [{
            id: "3003",
            language: "en",
            text: "What's the first question?",
        }],
        createdAt: "2021-10-03T02:00:00Z",
        sequence: 2,
        versionIndex: 0,
    },
    {
        id: "4",
        parent: { id: "2" },
        translations: [{
            id: "3004",
            language: "en",
            text: "What's the second question?",
        }],
        createdAt: "2021-10-03T03:00:00Z",
        sequence: 3,
        versionIndex: 0,
    },
    {
        id: "5",
        parent: { id: "2" },
        translations: [{
            id: "3005",
            language: "en",
            text: "What's the third question?",
        }],
        createdAt: "2021-10-03T04:00:00Z",
        sequence: 4,
        versionIndex: 0,
    },
    {
        id: "6",
        parent: { id: "3" },
        translations: [{
            id: "3006",
            language: "en",
            text: "How do I change my password?",
        }],
        createdAt: "2021-10-03T05:00:00Z",
        sequence: 5,
        versionIndex: 0,
    },
    {
        id: "7",
        parent: { id: "3" },
        translations: [{
            id: "3007",
            language: "en",
            text: "And how do I enable two-factor authentication?",
        }],
        createdAt: "2021-10-03T06:00:00Z",
        sequence: 6,
        versionIndex: 0,
    },
    {
        id: "8",
        parent: { id: "4" },
        translations: [{
            id: "3008",
            language: "en",
            text: "What are the supported payment methods?",
        }],
        createdAt: "2021-10-03T07:00:00Z",
        sequence: 7,
        versionIndex: 0,
    },
    {
        id: "9",
        parent: { id: "4" },
        translations: [{
            id: "3009",
            language: "en",
            text: "Do you accept cryptocurrency?",
        }],
        createdAt: "2021-10-03T08:00:00Z",
        sequence: 8,
        versionIndex: 0,
    },
    {
        id: "10",
        parent: { id: "5" },
        translations: [{
            id: "3010",
            language: "en",
            text: "What is your return policy?",
        }],
        createdAt: "2021-10-03T09:00:00Z",
        sequence: 9,
        versionIndex: 0,
    },
    {
        id: "11",
        parent: { id: "5" },
        translations: [{
            id: "3011",
            language: "en",
            text: "Do you offer international shipping?",
        }],
        createdAt: "2021-10-03T10:00:00Z",
        sequence: 10,
        versionIndex: 0,
    },
];

/** Fourth test case: 1 root without any branches. But a message in the middle was deleted */
const case4: MinimumChatMessage[] = [
    {
        parent: {
            id: "0e92e862-f80f-461d-b9db-ab1c61e53810",
            parent: {
                id: "228dff1c-db18-4da9-a4b6-fd33d751ff94",
            },

        },
        id: "77483f3d-132b-48ce-a10d-0dd0f607905d",
        sequence: 112,
        translations: [],
        versionIndex: 0,
    },
    {
        parent: {
            id: "228dff1c-db18-4da9-a4b6-fd33d751ff94",
            parent: {
                id: "a765bdec-e4b1-4927-b001-122b6d326b30",
            },

        },
        id: "0e92e862-f80f-461d-b9db-ab1c61e53810",
        sequence: 111,
        translations: [],
        versionIndex: 0,
    },
    {
        parent: {
            id: "a765bdec-e4b1-4927-b001-122b6d326b30",
            parent: {
                id: "1edf6c8f-3107-4ab7-894e-2805230950f3",
            },

        },
        id: "228dff1c-db18-4da9-a4b6-fd33d751ff94",
        sequence: 110,
        translations: [],
        versionIndex: 0,
    },
    {
        parent: {
            id: "1edf6c8f-3107-4ab7-894e-2805230950f3",
            parent: {
                id: "6aa3b3dd-93f5-49b3-86ba-1828df622e74",
            },

        },
        id: "a765bdec-e4b1-4927-b001-122b6d326b30",
        sequence: 109,
        translations: [],
        versionIndex: 0,
    },
    {
        parent: {
            id: "6aa3b3dd-93f5-49b3-86ba-1828df622e74",
            parent: {
                id: "cc7070c0-1948-41df-8d76-2c30f8810a50",
            },
        },
        id: "1edf6c8f-3107-4ab7-894e-2805230950f3",
        sequence: 108,
        translations: [],
        versionIndex: 0,
    },
    {
        parent: {
            id: "cc7070c0-1948-41df-8d76-2c30f8810a50",
            parent: {
                id: "bee03f1b-e94b-44ea-ad2f-a39518b66b24",
            },

        },
        id: "6aa3b3dd-93f5-49b3-86ba-1828df622e74",
        sequence: 107,
        translations: [],
        versionIndex: 0,
    },
    {
        parent: {
            id: "bee03f1b-e94b-44ea-ad2f-a39518b66b24",
            parent: {
                id: "ee5ce218-d878-41a1-a8e0-1f070bb482fc",
            },

        },
        id: "cc7070c0-1948-41df-8d76-2c30f8810a50",
        sequence: 106,
        translations: [],
        versionIndex: 0,
    },
    {
        parent: {
            id: "ee5ce218-d878-41a1-a8e0-1f070bb482fc",
            parent: {
                id: "5cd3c541-31d1-4ee1-a152-4e4e210a0ef0",
            },

        },
        id: "bee03f1b-e94b-44ea-ad2f-a39518b66b24",
        sequence: 105,
        translations: [],
        versionIndex: 0,
    },
    {
        parent: {
            id: "5cd3c541-31d1-4ee1-a152-4e4e210a0ef0",
            parent: {
                id: "710026fd-e5c9-493b-b0c7-82ceae668f83",
            },

        },
        id: "ee5ce218-d878-41a1-a8e0-1f070bb482fc",
        sequence: 104,
        translations: [],
        versionIndex: 0,
    },
    {
        parent: {
            id: "710026fd-e5c9-493b-b0c7-82ceae668f83",
            parent: {
                id: "53e60a3f-bd6c-40fe-abcc-b069782e3e3b",
            },

        },
        id: "5cd3c541-31d1-4ee1-a152-4e4e210a0ef0",
        sequence: 103,
        translations: [],
        versionIndex: 0,
    },
    {
        parent: {
            id: "53e60a3f-bd6c-40fe-abcc-b069782e3e3b",
            parent: {
                id: "f96c651d-2c33-4ace-99a8-73039d93ba71",
            },

        },
        id: "710026fd-e5c9-493b-b0c7-82ceae668f83",
        sequence: 98,
        translations: [],
        versionIndex: 0,
    },
    {
        parent: {
            id: "f96c651d-2c33-4ace-99a8-73039d93ba71",
            parent: {
                id: "af8392d4-9f7e-4dbc-a0a5-56f44360fdf3",
            },
        },
        id: "53e60a3f-bd6c-40fe-abcc-b069782e3e3b",
        sequence: 97,
        translations: [],
        versionIndex: 0,
    },
    {
        parent: {
            id: "af8392d4-9f7e-4dbc-a0a5-56f44360fdf3",
            parent: {
                id: "531f5edb-6179-4e13-a5e1-77fd30cda76d",
            },

        },
        id: "f96c651d-2c33-4ace-99a8-73039d93ba71",
        sequence: 96,
        translations: [],
        versionIndex: 0,
    },
    {
        parent: {
            id: "38963722-bf55-464e-9e32-2209cdc6ef70",
            parent: {
                id: "0ef23969-2021-4a34-99b0-314961d58ad5",
            },

        },
        id: "531f5edb-6179-4e13-a5e1-77fd30cda76d",
        sequence: 94,
        translations: [],
        versionIndex: 0,
    },
    {
        parent: {
            id: "0ef23969-2021-4a34-99b0-314961d58ad5",
            parent: {
                id: "a9330a80-cbeb-4526-aca7-c50081187756",
            },

        },
        id: "38963722-bf55-464e-9e32-2209cdc6ef70",
        sequence: 93,
        translations: [],
        versionIndex: 0,
    },
    {
        parent: {
            id: "a9330a80-cbeb-4526-aca7-c50081187756",
            parent: null,

        },
        id: "0ef23969-2021-4a34-99b0-314961d58ad5",
        sequence: 92,
        translations: [],
        versionIndex: 0,
    },
    {
        id: "a9330a80-cbeb-4526-aca7-c50081187756",
        sequence: 91,
        translations: [],
        versionIndex: 0,
        parent: null,
    },
];

/**
 * Checks the validity of root nodes in the message tree.
 * @param map - Map of message IDs to their corresponding nodes.
 * @param roots - Array of root nodes in the message tree.
 * @throws {Error} If any root node has a parent.
 */
function checkRootValidity<T extends MinimumChatMessage>(
    map: Map<string, MessageNode<T>>,
    roots: string[],
) {
    roots.forEach(rootId => {
        const parent = map.get(rootId)?.message.parent;
        if (parent) {
            throw new Error(`Root message with ID ${rootId} has a parent. This is invalid.`);
        }
    });
}

/**
 * Checks that every message that defines a parent is in that parent's children array.
 * @param map - Map of message IDs to their corresponding nodes.
 * @throws {Error} If any message is not a child of its parent in the tree.
 */
function checkChildParentRelationship<T extends MinimumChatMessage>(
    map: Map<string, MessageNode<T>>,
) {
    map.forEach((node, id) => {
        if (node.message.parent?.id) {
            const parentNode = map.get(node.message.parent.id);
            if (!parentNode || !parentNode.children.includes(id)) {
                throw new Error(`Message with ID ${id} is not a child of its parent in the tree.`);
            }
        }
    });
}

/**
 * Ensures the integrity of the message tree by checking for cycles and verifying reachability from roots.
 * @param map - Map of message IDs to their corresponding nodes.
 * @param roots - Array of root nodes in the message tree.
 * @throws {Error} If there are cycles or unreachable nodes.
 */
function checkTreeIntegrity<T extends MinimumChatMessage>(
    map: Map<string, MessageNode<T>>,
    roots: string[],
): void {
    const visited = new Set<string>();
    function traverseTree(messageId: string, visited: Set<string>): void {
        if (visited.has(messageId)) {
            throw new Error(`Cycle detected at message with ID ${messageId}.`);
        }
        visited.add(messageId);
        const node = map.get(messageId);
        if (!node) {
            throw new Error(`Message with ID ${messageId} is not in the map.`);
        }
        node.children.forEach(child => traverseTree(child, visited));
    }

    roots.forEach(rootId => traverseTree(rootId, visited));
    map.forEach((_, id) => {
        if (!visited.has(id)) {
            throw new Error(`Message with ID ${id} is not reachable from any root.`);
        }
    });
}

/**
 * Checks if the children of each node in the message map are sorted by their sequence number.
 * @param map - Map of message IDs to their corresponding nodes.
 * @throws {Error} If children are not correctly sorted by sequence.
 */
function checkMessageEditOrder<T extends MinimumChatMessage>(map: Map<string, MessageNode<T>>): void {
    map.forEach((node, id) => {
        for (let i = 1; i < node.children.length; i++) {
            const currChildSequence = map.get(node.children[i])?.message?.sequence ?? -1; // Default to -1 if undefined
            const lastChildSequence = map.get(node.children[i - 1])?.message?.sequence ?? -1; // Default to -1 if undefined

            if (currChildSequence < lastChildSequence) {
                throw new Error(`Children of message with ID ${id} are not sorted by sequence.`);
            }
        }
    });
}

/**
 * Verifies that all orphan nodes in the message map are listed 
 * in the roots array.
 * @param map - Map of message IDs to their corresponding nodes.
 * @param roots - Array of root nodes in the message tree.
 * @throws {Error} If there are orphan nodes in the message map.
 */
function checkNoOrphanNodes<T extends MinimumChatMessage>(
    map: Map<string, MessageNode<T>>,
    roots: string[],
): void {
    map.forEach((node, id) => {
        if (!node.message.parent?.id && !roots.includes(id)) {
            throw new Error(`Orphan node detected with ID ${id}.`);
        }
    });
}

type IntegrityCheck = "RootValidity" | "ChildParentRelationship" | "TreeIntegrity" | "MessageEditOrder" | "NoOrphanNodes";

class TreeTestUtils {
    /**
    * Performs a comprehensive integrity check on the message tree.
    * It includes checks for root validity, child-parent relationship,
    * tree integrity, message edit order, and the presence of orphan nodes.
    * 
    * @param map - Map of message IDs to their corresponding nodes.
    * @param roots - Array of root nodes in the message tree.
    * @param skip - Array of checks to skip.
    * @throws {Error} If any integrity checks fail.
    */
    static assertTreeIntegrity<T extends MinimumChatMessage>(
        map: Map<string, MessageNode<T>>,
        roots: string[],
        skip?: IntegrityCheck[],
    ): void {
        if (!skip) skip = [];
        if (!skip.includes("RootValidity")) checkRootValidity(map, roots);
        if (!skip.includes("ChildParentRelationship")) checkChildParentRelationship(map);
        if (!skip.includes("TreeIntegrity")) checkTreeIntegrity(map, roots);
        if (!skip.includes("MessageEditOrder")) checkMessageEditOrder(map);
        if (!skip.includes("NoOrphanNodes")) checkNoOrphanNodes(map, roots);
    }

    /**
    * Shuffles array in place using Fisher-Yates (aka Durstenfeld) shuffle algorithm.
    * @param array The array to shuffle.
    */
    static shuffle<T>(array: T[]): T[] {
        for (let i = array.length - 1; i > 0; i--) {
            const j = Math.floor(Math.random() * (i + 1));
            [array[i], array[j]] = [array[j], array[i]] as [T, T]; // Swap elements
        }
        return array;
    }

    /**
     * Compares two sets of roots to ensure they have the same structure.
     * @param map The map of message IDs to their corresponding nodes.
     * @param roots1 The roots of the first tree to compare.
     * @param roots2 The roots of the second tree to compare.
     */
    static treesHaveSameStructure<T extends MinimumChatMessage>(
        map: Map<string, MessageNode<T>>,
        roots1: string[],
        roots2: string[],
    ): boolean {
        if (roots1.length !== roots2.length) return false;
        for (let i = 0; i < roots1.length; i++) {
            if (!this.compareSubtrees(map, roots1[i], roots2[i])) return false;
        }
        return true;
    }

    /**
     * Recursively compares two subtrees to ensure they have the same structure.
     * @param map The map of message IDs to their corresponding nodes.
     * @param tree1Id The first subtree to compare.
     * @param tree2Id The second subtree to compare.
     */
    static compareSubtrees<T extends MinimumChatMessage>(
        map: Map<string, MessageNode<T>>,
        tree1Id: string,
        tree2Id: string,
    ): boolean {
        const tree1 = map.get(tree1Id);
        const tree2 = map.get(tree2Id);
        if (!tree1 && !tree2) return true;
        if (!tree1 || !tree2) return false;
        if (tree1.message.id !== tree2.message.id) return false;
        if (tree1.children.length !== tree2.children.length) return false;
        for (let i = 0; i < tree1.children.length; i++) {
            if (!this.compareSubtrees(map, tree1.children[i], tree2.children[i])) {
                return false;
            }
        }
        return true;
    }

    /**
     * Verifies that a tree node has a single child at each level and 
     * the message IDs increase sequentially. 
     * 
     * NOTE: We've set up the tests so that the message IDs should always be 
     * sequential if the tree is built correctly. In real-world scenarios, 
     * the IDs would be uuids, and we wouldn't be able to rely on this.
     * 
     * @param map The map of message IDs to their corresponding nodes.
     * @param rootId The ID of the root node to start the verification.
     * @param minId The expected ID of the first node in the tree, and thus the 
     * smallest ID in the tree.
     */
    static verifySingleNodeStructureAndSequentialIds<T extends MinimumChatMessage>(
        map: Map<string, MessageNode<T>>,
        rootId: string,
        minId: string,
    ): boolean {
        const node = map.get(rootId);
        if (!node) return false;
        // Check if the current node's ID is at least as large as the minimum expected ID
        const nodeId = parseInt(node.message.id);
        if (nodeId < parseInt(minId)) return false;
        // Check if the node has at most one child
        if (node.children.length > 1) return false;
        // If there's a child, recursively check it with the next expected ID
        if (node.children.length === 1) {
            return this.verifySingleNodeStructureAndSequentialIds(map, node.children[0], (parseInt(minId) + 1).toString());
        }
        return true;
    }

    /**
     * Counts the number of nodes in the tree starting from the given node.
     * @param map The map of message IDs to their corresponding nodes.
     * @param messageId The ID of the node to start counting from.
     */
    static countNodesInTree<T extends MinimumChatMessage>(
        map: Map<string, MessageNode<T>>,
        messageId: string,
    ): number {
        const node = map.get(messageId);
        if (!node) return 0;

        // Count the current node
        let count = 1;

        // Recursively count children
        for (const childId of node.children) {
            count += this.countNodesInTree(map, childId);
        }

        return count;
    }
}

function runCommonTests(caseData: MinimumChatMessage[], chatId: string, caseTitle: string, skip?: IntegrityCheck[]) {
    describe(`Common Tests for ${caseTitle}`, () => {
        it(`${caseTitle} - Tree is created successfully`, () => {
            const { result } = renderHook(() => useMessageTree(chatId));

            act(() => {
                result.current.addMessages(caseData as ChatMessage[]);
            });

            const map = result.current.tree[MessageTreeProperty.map];
            const roots = result.current.tree[MessageTreeProperty.roots];
            expect(() => TreeTestUtils.assertTreeIntegrity(map, roots, skip)).not.toThrow();
        });

        it(`${caseTitle} - Result has the same number of messages as the input`, () => {
            const { result } = renderHook(() => useMessageTree(chatId));

            act(() => {
                result.current.addMessages(caseData as ChatMessage[]);
            });

            let totalNodes = 0;
            const map = result.current.tree[MessageTreeProperty.map];
            const roots = result.current.tree[MessageTreeProperty.roots];
            for (const rootId of roots) {
                totalNodes += TreeTestUtils.countNodesInTree(map, rootId);
            }

            expect(totalNodes).toBe(caseData.length);
        });

        it(`${caseTitle} - Roots and children are ordered by sequence`, () => {
            const { result } = renderHook(() => useMessageTree(chatId));

            act(() => {
                result.current.addMessages(caseData as ChatMessage[]);
            });

            const map = result.current.tree[MessageTreeProperty.map];
            const roots = result.current.tree[MessageTreeProperty.roots];

            for (let i = 0; i < roots.length - 1; i++) {
                const nextSequence = map.get(roots[i + 1])?.message?.sequence;
                expect(typeof nextSequence).toBe("number");
                expect(map.get(roots[i])?.message?.sequence).toBeLessThan(nextSequence ?? 0);
            }

            function checkChildOrder(messageId: string) {
                const node = map.get(messageId);
                expect(node).toBeDefined();
                if (!node) return;
                for (let i = 0; i < node.children.length - 1; i++) {
                    const nextSequence = map.get(node.children[i + 1])?.message?.sequence;
                    expect(typeof nextSequence).toBe("number");
                    expect(map.get(node.children[i])?.message?.sequence).toBeLessThan(nextSequence ?? 0);
                }
                node.children.forEach(checkChildOrder);
            }

            roots.forEach(checkChildOrder);
        });

        it(`${caseTitle} - Result is the same when shuffled`, () => {
            const repeatCount = 10; // Run the test multiple times to maximize our chances of catching any issues
            for (let i = 0; i < repeatCount; i++) {
                const shuffledCaseData = TreeTestUtils.shuffle([...caseData]);

                const { result: originalResult } = renderHook(() => useMessageTree(chatId));
                act(() => {
                    originalResult.current.addMessages(caseData as ChatMessage[]);
                });

                const { result: shuffledResult } = renderHook(() => useMessageTree(chatId));
                act(() => {
                    shuffledResult.current.addMessages(shuffledCaseData as ChatMessage[]);
                });

                const originalMap = originalResult.current.tree[MessageTreeProperty.map];
                const originalRoots = originalResult.current.tree[MessageTreeProperty.roots];
                expect(() => TreeTestUtils.assertTreeIntegrity(originalMap, originalRoots, skip)).not.toThrow();

                const shuffledMap = shuffledResult.current.tree[MessageTreeProperty.map];
                const shuffledRoots = shuffledResult.current.tree[MessageTreeProperty.roots];
                expect(() => TreeTestUtils.assertTreeIntegrity(shuffledMap, shuffledRoots, skip)).not.toThrow();

                // Check that the trees have the same structure
                expect(TreeTestUtils.treesHaveSameStructure(originalMap, originalRoots, shuffledRoots)).toBe(true);
            }
        });
    });
}

describe("useMessageTree", () => {
    const chatId = DUMMY_ID;
    let consoleErrorStub: sinon.SinonStub;
    let consoleWarnStub: sinon.SinonStub;

    beforeAll(() => {
        consoleErrorStub = sinon.stub(console, "error");
        consoleWarnStub = sinon.stub(console, "warn");
    });

    beforeEach(() => {
        consoleErrorStub.resetHistory();
        consoleWarnStub.resetHistory();
    });

    afterAll(() => {
        consoleErrorStub.restore();
        consoleWarnStub.restore();
    });

    describe("Test cases have proper structure", () => {
        runCommonTests(case1, chatId, "Case 1");
        it("Case 1 - One message per level, and in expected order (sequential IDs in this case)", () => {
            const { result } = renderHook(() => useMessageTree(chatId));

            act(() => {
                result.current.addMessages(case1 as ChatMessage[]);
            });

            const map = result.current.tree[MessageTreeProperty.map];
            const roots = result.current.tree[MessageTreeProperty.roots];

            expect(roots).toHaveLength(1);
            const rootId = roots[0];
            const isValidStructure = TreeTestUtils.verifySingleNodeStructureAndSequentialIds(map, rootId, "1");
            expect(isValidStructure).toBe(true);
        });

        runCommonTests(case2, chatId, "Case 2");
        it("Case 2 - Has 2 roots, with first having no children, and second having one message per level", () => {
            const { result } = renderHook(() => useMessageTree(chatId));

            act(() => {
                result.current.addMessages(case2 as ChatMessage[]);
            });

            const map = result.current.tree[MessageTreeProperty.map];
            const roots = result.current.tree[MessageTreeProperty.roots];

            expect(roots).toHaveLength(2);
            const firstRootId = roots[0];
            const firstRoot = map.get(firstRootId);
            const secondRootId = roots[1];

            expect(firstRoot?.children).toHaveLength(0);
            const isValidStructureForSecondRoot = TreeTestUtils.verifySingleNodeStructureAndSequentialIds(map, secondRootId, "2");
            expect(isValidStructureForSecondRoot).toBe(true);
        });

        runCommonTests(case3, chatId, "Case 3");
        it("Case 3 - Proper hierarchical structure with 1 root, 1 child, 3 grandchildren, and 6 great-grandchildren", () => {
            const { result } = renderHook(() => useMessageTree(chatId));

            act(() => {
                result.current.addMessages(case3 as ChatMessage[]);
            });

            const map = result.current.tree[MessageTreeProperty.map];
            const roots = result.current.tree[MessageTreeProperty.roots];

            expect(roots).toHaveLength(1);
            const rootId = roots[0];
            const root = map.get(rootId);

            expect(root?.children).toHaveLength(1);
            const childId = root?.children[0];
            const child = map.get(childId ?? "");

            expect(child?.children).toHaveLength(3);
            for (const grandchildId of (child?.children ?? [])) {
                const grandchild = map.get(grandchildId);
                expect(grandchild?.children).toHaveLength(2);
            }
        });

        runCommonTests(case4, chatId, "Case 4", ["ChildParentRelationship"]); // Skip check for missing parent, since this case is built that way on purpose
        it("Case 4 - 1 root without any branches, but a message in the middle was deleted", () => {
            const { result } = renderHook(() => useMessageTree(chatId));

            act(() => {
                result.current.addMessages(case4 as ChatMessage[]);
            });

            const map = result.current.tree[MessageTreeProperty.map];
            const roots = result.current.tree[MessageTreeProperty.roots];

            expect(roots).toHaveLength(1);
            const rootId = roots[0];
            const root = map.get(rootId);

            expect(root?.children).toHaveLength(1);
        });
    });

    describe("Tree operations", () => {
        const chatId = DUMMY_ID;

        const initialMessages = [...case3];
        const newMessage = {
            id: "20",
            parent: { id: "8" },  // case3's 2nd grandchild's first child ID
            translations: [{
                id: "10020",
                language: "en",
                text: "This is a new added message.",
            }],
            createdAt: "2021-10-02T01:00:00Z",
            sequence: 10,
        };

        describe("Add Message", () => {
            it("Adds a new message to the tree", () => {
                const { result } = renderHook(() => useMessageTree(chatId));

                act(() => {
                    result.current.addMessages(initialMessages as ChatMessage[]);
                });

                act(() => {
                    result.current.addMessages([newMessage] as ChatMessage[]);
                });

                // Find and verify the newly added message
                const newMessageNode = result.current.tree[MessageTreeProperty.map].get(newMessage.id);
                expect(newMessageNode).toBeDefined();
                expect(newMessageNode!.message.id).toBe(newMessage.id);
            });

            it("Won't add the same message twice", () => {
                const { result } = renderHook(() => useMessageTree(chatId));
                let initialNodeCount = 0;

                act(() => {
                    result.current.addMessages([...initialMessages, newMessage] as ChatMessage[]);
                });

                initialNodeCount = result.current.tree[MessageTreeProperty.map].size;

                act(() => {
                    result.current.addMessages([newMessage] as ChatMessage[]);
                });

                expect(result.current.tree[MessageTreeProperty.map].size).toBe(initialNodeCount);
            });

            it("Won't add the same messages (plural) twice", () => {
                const { result } = renderHook(() => useMessageTree(chatId));

                act(() => {
                    result.current.addMessages(case4 as ChatMessage[]);
                });

                act(() => {
                    result.current.addMessages(case4 as ChatMessage[]);
                });

                act(() => {
                    result.current.addMessages(case4 as ChatMessage[]);
                });

                expect(result.current.tree[MessageTreeProperty.map].size).toBe(case4.length);
            });
        });

        describe("Edit Message", () => {
            it("Edits the newly added message", () => {
                const { result } = renderHook(() => useMessageTree(chatId));

                act(() => {
                    result.current.addMessages([...initialMessages, newMessage] as ChatMessage[]);
                });

                const updatedMessage = {
                    ...newMessage,
                    translations: [{
                        id: "10020",
                        language: "en",
                        text: "This is an edited message.",
                    }],
                };

                act(() => {
                    result.current.editMessage(updatedMessage as ChatMessage);
                });

                const updatedNode = result.current.tree[MessageTreeProperty.map].get(updatedMessage.id);
                expect(updatedNode).toBeDefined();
                expect(updatedNode!.message).toEqual(updatedMessage);
            });

            it("Doesn't edit if the message is missing", () => {
                const { result } = renderHook(() => useMessageTree(chatId));

                act(() => {
                    result.current.addMessages(initialMessages as ChatMessage[]);
                });

                const updatedMessage = {
                    ...newMessage,
                    translations: [{
                        id: "10020",
                        language: "en",
                        text: "This is an edited message.",
                    }],
                };

                act(() => {
                    result.current.editMessage(updatedMessage as ChatMessage);
                });

                const updatedNode = result.current.tree[MessageTreeProperty.map].get(updatedMessage.id);
                // Expect the node to be undefined since it wasn't added
                expect(updatedNode).toBeUndefined();
            });
        });

        describe("Remove Message", () => {
            it("Removes the edited message", () => {
                const { result } = renderHook(() => useMessageTree(chatId));

                act(() => {
                    result.current.addMessages([...initialMessages, newMessage] as ChatMessage[]);
                });

                act(() => {
                    result.current.removeMessages([newMessage.id]);
                });

                const removedNode = result.current.tree[MessageTreeProperty.map].get(newMessage.id);
                expect(removedNode).toBeUndefined();
            });
        });

        describe("Clear Messages", () => {
            it("Resets the message tree", () => {
                const { result } = renderHook(() => useMessageTree(chatId));

                act(() => {
                    result.current.addMessages(initialMessages as ChatMessage[]);
                });

                // Verify that the tree is not empty
                expect(result.current.tree[MessageTreeProperty.map].size).toBeGreaterThan(0);
                expect(result.current.tree[MessageTreeProperty.roots].length).toBeGreaterThan(0);

                // Clear the message tree
                act(() => {
                    result.current.clearMessages();
                });

                // Verify that the message tree is reset
                expect(result.current.tree[MessageTreeProperty.map].size).toBe(0);
                expect(result.current.tree[MessageTreeProperty.roots]).toHaveLength(0);
            });

            it("Clearing then adding back messages is the same as just adding messages", () => {
                const { result: result1 } = renderHook(() => useMessageTree(chatId));
                const { result: result2 } = renderHook(() => useMessageTree(chatId));

                // Add all messages to the empty tree
                act(() => {
                    result1.current.addMessages(initialMessages as ChatMessage[]);
                });
                act(() => {
                    result2.current.clearMessages();
                });
                act(() => {
                    result2.current.addMessages(initialMessages as ChatMessage[]);
                });

                // Verify that the resulting trees are the same
                const result1Map = result1.current.tree[MessageTreeProperty.map];
                const result1Roots = result1.current.tree[MessageTreeProperty.roots];
                const result2Roots = result2.current.tree[MessageTreeProperty.roots];
                expect(TreeTestUtils.treesHaveSameStructure(result1Map, result1Roots, result2Roots)).toBe(true);
            });
        });

        it("should maintain tree state across multiple operations", () => {
            const { result } = renderHook(() => useMessageTree(chatId));

            // Add initial messages
            act(() => {
                result.current.addMessages([{
                    id: "1",
                    parent: null,
                    translations: [{
                        id: "t1",
                        language: "en",
                        text: "First message",
                    }],
                    sequence: 1,
                    versionIndex: 0,
                }] as ChatMessage[]);
            });

            // Add a child message
            act(() => {
                result.current.addMessages([{
                    id: "2",
                    parent: { id: "1" },
                    translations: [{
                        id: "t2",
                        language: "en",
                        text: "Reply",
                    }],
                    sequence: 2,
                    versionIndex: 0,
                }] as ChatMessage[]);
            });

            // Edit the first message
            act(() => {
                result.current.editMessage({
                    id: "1",
                    translations: [{
                        id: "t3",
                        language: "en",
                        text: "Edited first message",
                    }],
                });
            });

            // Remove the second message
            act(() => {
                result.current.removeMessages(["2"]);
            });

            // Verify final state
            const tree = result.current.tree;
            expect(tree.getMessagesCount()).toBe(1);
            expect(tree.getRoots()).toHaveLength(1);
            expect(tree.getRoots()[0]).toBe("1");

            const firstMessage = tree.getMap().get("1");
            expect(firstMessage?.message.translations[0].text).toBe("Edited first message");
            expect(firstMessage?.children).toHaveLength(0);
        });

        it("should handle branch updates correctly", () => {
            const { result } = renderHook(() => useMessageTree(chatId));

            // Add messages with multiple versions
            act(() => {
                result.current.addMessages([
                    {
                        id: "1",
                        parent: null,
                        translations: [{ id: "t1", language: "en", text: "Original" }],
                        sequence: 1,
                        versionIndex: 0,
                    },
                    {
                        id: "2",
                        parent: { id: "1" },
                        translations: [{ id: "t2", language: "en", text: "Version 1" }],
                        sequence: 2,
                        versionIndex: 1,
                    },
                    {
                        id: "3",
                        parent: { id: "1" },
                        translations: [{ id: "t3", language: "en", text: "Version 2" }],
                        sequence: 3,
                        versionIndex: 2,
                    },
                ] as ChatMessage[]);
            });

            // Update branches
            act(() => {
                result.current.setBranches({ "1": "3" }); // Select version 2
            });

            // Verify branch selection
            expect(result.current.branches["1"]).toBe("3");
        });
    });
});

describe("MessageTree Class Operations", () => {
    let tree: MessageTree<MinimumChatMessage>;

    beforeEach(() => {
        tree = new MessageTree<MinimumChatMessage>();
    });

    describe("addMessagesBatch", () => {
        it("should handle empty input", () => {
            tree.addMessagesBatch([]);
            expect(tree.getMessagesCount()).toBe(0);
        });

        it("should handle null input", () => {
            // @ts-expect-error Testing null input
            tree.addMessagesBatch(null);
            expect(tree.getMessagesCount()).toBe(0);
        });

        it("should add messages in correct sequence order regardless of input order", () => {
            const messages = [
                {
                    id: "2",
                    sequence: 2,
                    translations: [],
                    versionIndex: 0,
                },
                {
                    id: "1",
                    sequence: 1,
                    translations: [],
                    versionIndex: 0,
                },
                {
                    id: "3",
                    sequence: 3,
                    translations: [],
                    versionIndex: 0,
                },
            ];

            tree.addMessagesBatch(messages);
            const roots = tree.getRoots();
            expect(roots).toHaveLength(3);
            expect(roots[0]).toBe("1");
            expect(roots[1]).toBe("2");
            expect(roots[2]).toBe("3");
        });

        it("should handle duplicate messages by keeping the latest version", () => {
            const originalMessage = {
                id: "1",
                translations: [{
                    id: "t1",
                    language: "en",
                    text: "Original text",
                }],
                sequence: 1,
                versionIndex: 0,
            };

            const updatedMessage = {
                ...originalMessage,
                translations: [{
                    id: "t2",
                    language: "en",
                    text: "Updated text",
                }],
            };

            tree.addMessagesBatch([originalMessage]);
            tree.addMessagesBatch([updatedMessage]);

            expect(tree.getMessagesCount()).toBe(1);
            const node = tree.getMap().get("1");
            expect(node?.message.translations[0].text).toBe("Updated text");
        });

        it("should handle complex parent-child relationships", () => {
            const messages = [
                {
                    id: "1",
                    parent: null,
                    sequence: 1,
                    translations: [],
                    versionIndex: 0,
                },
                {
                    id: "2",
                    parent: { id: "1" },
                    sequence: 2,
                    translations: [],
                    versionIndex: 0,
                },
                {
                    id: "3",
                    parent: { id: "2" },
                    sequence: 3,
                    translations: [],
                    versionIndex: 0,
                },
            ];

            tree.addMessagesBatch(messages);

            const root = tree.getMap().get("1");
            expect(root?.children).toContain("2");

            const child = tree.getMap().get("2");
            expect(child?.children).toContain("3");
        });

        it("should handle orphaned messages gracefully", () => {
            const messages = [
                {
                    id: "2",
                    parent: { id: "1" }, // Parent doesn't exist
                    sequence: 2,
                    translations: [],
                    versionIndex: 0,
                },
                {
                    id: "3",
                    parent: { id: "2" },
                    sequence: 3,
                    translations: [],
                    versionIndex: 0,
                },
            ];

            tree.addMessagesBatch(messages);

            // Messages should be added as roots since parent doesn't exist
            const roots = tree.getRoots();
            expect(roots).toContain("2");

            // Child relationship should be maintained
            const parent = tree.getMap().get("2");
            expect(parent?.children).toContain("3");
        });
    });

    describe("editMessage", () => {
        it("should return false when editing non-existent message", () => {
            const result = tree.editMessage({
                id: "nonexistent",
                translations: [],
            });
            expect(result).toBe(false);
        });

        it("should successfully edit existing message", () => {
            const originalMessage = {
                id: "1",
                translations: [{
                    id: "t1",
                    language: "en",
                    text: "Original text",
                }],
                sequence: 1,
                versionIndex: 0,
            };

            tree.addMessagesBatch([originalMessage]);

            const result = tree.editMessage({
                id: "1",
                translations: [{
                    id: "t2",
                    language: "en",
                    text: "Updated text",
                }],
            });

            expect(result).toBe(true);
            const node = tree.getMap().get("1");
            expect(node?.message.translations[0].text).toBe("Updated text");
        });

        it("should preserve child relationships when editing", () => {
            const messages = [
                {
                    id: "1",
                    parent: null,
                    sequence: 1,
                    translations: [],
                    versionIndex: 0,
                },
                {
                    id: "2",
                    parent: { id: "1" },
                    sequence: 2,
                    translations: [],
                    versionIndex: 0,
                },
            ];

            tree.addMessagesBatch(messages);

            const result = tree.editMessage({
                id: "1",
                translations: [{
                    id: "t1",
                    language: "en",
                    text: "Updated text",
                }],
            });

            expect(result).toBe(true);
            const node = tree.getMap().get("1");
            expect(node?.children).toContain("2");
        });

        it("should only update specified fields", () => {
            const originalMessage = {
                id: "1",
                translations: [{
                    id: "t1",
                    language: "en",
                    text: "Original text",
                }],
                sequence: 1,
                versionIndex: 0,
                user: { id: "user1" },
            };

            tree.addMessagesBatch([originalMessage]);

            const result = tree.editMessage({
                id: "1",
                translations: [{
                    id: "t2",
                    language: "en",
                    text: "Updated text",
                }],
            });

            expect(result).toBe(true);
            const node = tree.getMap().get("1");
            expect(node?.message.translations[0].text).toBe("Updated text");
            expect(node?.message.sequence).toBe(1);
            expect(node?.message.user?.id).toBe("user1");
        });
    });
});

// Some additional tests that could be added in the future:
// - Circular references
// - Duplicate IDs
