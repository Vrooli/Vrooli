/* eslint-disable testing-library/no-node-access */
import { act, renderHook } from "@testing-library/react";
import { MessageNode, MinimumChatMessage, useMessageTree } from "./useMessageTree";

/** First test case: Result should have messages in order from ID 1 to 10, each with node having a single child */
const case1 = [
    {
        id: "3",
        parent: { id: "2" },
        translations: [{
            id: "1001",
            language: "en",
            text: "Certainly! What seems to be the issue?",
        }],
        created_at: "2021-10-01T02:00:00Z",
        sequence: 2,
    },
    {
        id: "1",
        parent: null,
        translations: [{
            id: "1001",
            language: "en",
            text: "Hello, how can I help you?",
        }],
        created_at: "2021-10-01T00:00:00Z",
        sequence: 0,
    },
    {
        id: "2",
        parent: { id: "1" },
        translations: [{
            id: "1001",
            language: "en",
            text: "I need assistance with my account.",
        }],
        created_at: "2021-10-01T01:00:00Z",
        sequence: 1,
    },
    {
        id: "4",
        parent: { id: "3" },
        translations: [{
            id: "1001",
            language: "en",
            text: "I can't access my account.",
        }],
        created_at: "2021-10-01T03:00:00Z",
        sequence: 3,
    },
    {
        id: "5",
        parent: { id: "4" },
        translations: [{
            id: "1001",
            language: "en",
            text: "I see. Have you tried resetting your password?",
        }],
        created_at: "2021-10-01T04:00:00Z",
        sequence: 4,
    },
    {
        id: "6",
        parent: { id: "5" },
        translations: [{
            id: "1001",
            language: "en",
            text: "Yes, but I didn't receive the reset email.",
        }],
        created_at: "2021-10-01T05:00:00Z",
        sequence: 5,
    },
    {
        id: "7",
        parent: { id: "6" },
        translations: [{
            id: "1001",
            language: "en",
            text: "I can help with that. Can you provide your username?",
        }],
        created_at: "2021-10-01T06:00:00Z",
        sequence: 6,
    },
    {
        id: "8",
        parent: { id: "7" },
        translations: [{
            id: "1001",
            language: "en",
            text: "My username is johndoe123.",
        }],
        created_at: "2021-10-01T07:00:00Z",
        sequence: 7,
    },
    {
        id: "9",
        parent: { id: "8" },
        translations: [{
            id: "1001",
            language: "en",
            text: "Thank you, John. I have sent a password reset link to your email.",
        }],
        created_at: "2021-10-01T08:00:00Z",
        sequence: 8,
    },
    {
        id: "10",
        parent: { id: "9" },
        translations: [{
            id: "1001",
            language: "en",
            text: "Got it. Thanks!",
        }],
        created_at: "2021-10-01T09:00:00Z",
        sequence: 9,
    },
];

/** Second test case: 2 roots */
const case2 = [
    {
        id: "1",
        parent: null,
        translations: [{
            id: "2001",
            language: "en",
            text: "Welcome to our platform!",
        }],
        created_at: "2021-10-02T00:00:00Z",
        sequence: 0,
    },
    {
        id: "2",
        parent: null,
        translations: [{
            id: "2002",
            language: "en",
            text: "Hello, how can I assist you?",
        }],
        created_at: "2021-10-02T01:00:00Z",
        sequence: 1,
    },
    {
        id: "3",
        parent: { id: "2" },
        translations: [{
            id: "2003",
            language: "en",
            text: "I'm having trouble logging in.",
        }],
        created_at: "2021-10-02T02:00:00Z",
        sequence: 2,
    },
    {
        id: "4",
        parent: { id: "3" },
        translations: [{
            id: "2004",
            language: "en",
            text: "I see. Have you tried resetting your password?",
        }],
        created_at: "2021-10-02T03:00:00Z",
        sequence: 3,
    },
];

/** Third test case: 1 root, 1 child, 3 grandchildren, 6 great-grandchildren (2 for each grandchild) */
const case3 = [
    {
        id: "1",
        parent: null,
        translations: [{
            id: "3001",
            language: "en",
            text: "Hello! How can I assist you today?",
        }],
        created_at: "2021-10-03T00:00:00Z",
        sequence: 0,
    },
    {
        id: "2",
        parent: { id: "1" },
        translations: [{
            id: "3002",
            language: "en",
            text: "I have multiple questions.",
        }],
        created_at: "2021-10-03T01:00:00Z",
        sequence: 1,
    },
    {
        id: "3",
        parent: { id: "2" },
        translations: [{
            id: "3003",
            language: "en",
            text: "What's the first question?",
        }],
        created_at: "2021-10-03T02:00:00Z",
        sequence: 2,
    },
    {
        id: "4",
        parent: { id: "2" },
        translations: [{
            id: "3004",
            language: "en",
            text: "What's the second question?",
        }],
        created_at: "2021-10-03T03:00:00Z",
        sequence: 3,
    },
    {
        id: "5",
        parent: { id: "2" },
        translations: [{
            id: "3005",
            language: "en",
            text: "What's the third question?",
        }],
        created_at: "2021-10-03T04:00:00Z",
        sequence: 4,
    },
    {
        id: "6",
        parent: { id: "3" },
        translations: [{
            id: "3006",
            language: "en",
            text: "How do I change my password?",
        }],
        created_at: "2021-10-03T05:00:00Z",
        sequence: 5,
    },
    {
        id: "7",
        parent: { id: "3" },
        translations: [{
            id: "3007",
            language: "en",
            text: "And how do I enable two-factor authentication?",
        }],
        created_at: "2021-10-03T06:00:00Z",
        sequence: 6,
    },
    {
        id: "8",
        parent: { id: "4" },
        translations: [{
            id: "3008",
            language: "en",
            text: "What are the supported payment methods?",
        }],
        created_at: "2021-10-03T07:00:00Z",
        sequence: 7,
    },
    {
        id: "9",
        parent: { id: "4" },
        translations: [{
            id: "3009",
            language: "en",
            text: "Do you accept cryptocurrency?",
        }],
        created_at: "2021-10-03T08:00:00Z",
        sequence: 8,
    },
    {
        id: "10",
        parent: { id: "5" },
        translations: [{
            id: "3010",
            language: "en",
            text: "What is your return policy?",
        }],
        created_at: "2021-10-03T09:00:00Z",
        sequence: 9,
    },
    {
        id: "11",
        parent: { id: "5" },
        translations: [{
            id: "3011",
            language: "en",
            text: "Do you offer international shipping?",
        }],
        created_at: "2021-10-03T10:00:00Z",
        sequence: 10,
    },
];

/** Fourth test case: 1 root without any branches. But a message in the middle was deleted */
const case4 = [
    {
        "parent": {
            "id": "0e92e862-f80f-461d-b9db-ab1c61e53810",
            "parent": {
                "id": "228dff1c-db18-4da9-a4b6-fd33d751ff94",
            },

        },
        "id": "77483f3d-132b-48ce-a10d-0dd0f607905d",
        "sequence": 112,
        "versionIndex": 0,
    },
    {
        "parent": {
            "id": "228dff1c-db18-4da9-a4b6-fd33d751ff94",
            "parent": {
                "id": "a765bdec-e4b1-4927-b001-122b6d326b30",
            },

        },
        "id": "0e92e862-f80f-461d-b9db-ab1c61e53810",
        "sequence": 111,
        "versionIndex": 0,
    },
    {
        "parent": {
            "id": "a765bdec-e4b1-4927-b001-122b6d326b30",
            "parent": {
                "id": "1edf6c8f-3107-4ab7-894e-2805230950f3",
            },

        },
        "id": "228dff1c-db18-4da9-a4b6-fd33d751ff94",
        "sequence": 110,
        "versionIndex": 0,
    },
    {
        "parent": {
            "id": "1edf6c8f-3107-4ab7-894e-2805230950f3",
            "parent": {
                "id": "6aa3b3dd-93f5-49b3-86ba-1828df622e74",
            },

        },
        "id": "a765bdec-e4b1-4927-b001-122b6d326b30",
        "sequence": 109,
        "versionIndex": 0,
    },
    {
        "parent": {
            "id": "6aa3b3dd-93f5-49b3-86ba-1828df622e74",
            "parent": {
                "id": "cc7070c0-1948-41df-8d76-2c30f8810a50",
            },
        },
        "id": "1edf6c8f-3107-4ab7-894e-2805230950f3",
        "sequence": 108,
        "versionIndex": 0,
    },
    {
        "parent": {
            "id": "cc7070c0-1948-41df-8d76-2c30f8810a50",
            "parent": {
                "id": "bee03f1b-e94b-44ea-ad2f-a39518b66b24",
            },

        },
        "id": "6aa3b3dd-93f5-49b3-86ba-1828df622e74",
        "sequence": 107,
        "versionIndex": 0,
    },
    {
        "parent": {
            "id": "bee03f1b-e94b-44ea-ad2f-a39518b66b24",
            "parent": {
                "id": "ee5ce218-d878-41a1-a8e0-1f070bb482fc",
            },

        },
        "id": "cc7070c0-1948-41df-8d76-2c30f8810a50",
        "sequence": 106,
        "versionIndex": 0,
    },
    {
        "parent": {
            "id": "ee5ce218-d878-41a1-a8e0-1f070bb482fc",
            "parent": {
                "id": "5cd3c541-31d1-4ee1-a152-4e4e210a0ef0",
            },

        },
        "id": "bee03f1b-e94b-44ea-ad2f-a39518b66b24",
        "sequence": 105,
        "versionIndex": 0,
    },
    {
        "parent": {
            "id": "5cd3c541-31d1-4ee1-a152-4e4e210a0ef0",
            "parent": {
                "id": "710026fd-e5c9-493b-b0c7-82ceae668f83",
            },

        },
        "id": "ee5ce218-d878-41a1-a8e0-1f070bb482fc",
        "sequence": 104,
        "versionIndex": 0,
    },
    {
        "parent": {
            "id": "710026fd-e5c9-493b-b0c7-82ceae668f83",
            "parent": {
                "id": "53e60a3f-bd6c-40fe-abcc-b069782e3e3b",
            },

        },
        "id": "5cd3c541-31d1-4ee1-a152-4e4e210a0ef0",
        "sequence": 103,
        "versionIndex": 0,
    },
    {
        "parent": {
            "id": "53e60a3f-bd6c-40fe-abcc-b069782e3e3b",
            "parent": {
                "id": "f96c651d-2c33-4ace-99a8-73039d93ba71",
            },

        },
        "id": "710026fd-e5c9-493b-b0c7-82ceae668f83",
        "sequence": 98,
        "versionIndex": 0,
    },
    {
        "parent": {
            "id": "f96c651d-2c33-4ace-99a8-73039d93ba71",
            "parent": {
                "id": "af8392d4-9f7e-4dbc-a0a5-56f44360fdf3",
            },
        },
        "id": "53e60a3f-bd6c-40fe-abcc-b069782e3e3b",
        "sequence": 97,
        "versionIndex": 0,
    },
    {
        "parent": {
            "id": "af8392d4-9f7e-4dbc-a0a5-56f44360fdf3",
            "parent": {
                "id": "531f5edb-6179-4e13-a5e1-77fd30cda76d",
            },

        },
        "id": "f96c651d-2c33-4ace-99a8-73039d93ba71",
        "sequence": 96,
        "versionIndex": 0,
    },
    {
        "parent": {
            "id": "38963722-bf55-464e-9e32-2209cdc6ef70",
            "parent": {
                "id": "0ef23969-2021-4a34-99b0-314961d58ad5",
            },

        },
        "id": "531f5edb-6179-4e13-a5e1-77fd30cda76d",
        "sequence": 94,
        "versionIndex": 0,
    },
    {
        "parent": {
            "id": "0ef23969-2021-4a34-99b0-314961d58ad5",
            "parent": {
                "id": "a9330a80-cbeb-4526-aca7-c50081187756",
            },

        },
        "id": "38963722-bf55-464e-9e32-2209cdc6ef70",
        "sequence": 93,
        "versionIndex": 0,
    },
    {
        "parent": {
            "id": "a9330a80-cbeb-4526-aca7-c50081187756",
            "parent": null,

        },
        "id": "0ef23969-2021-4a34-99b0-314961d58ad5",
        "sequence": 92,
        "versionIndex": 0,
    },
    {
        "id": "a9330a80-cbeb-4526-aca7-c50081187756",
        "sequence": 91,
        "versionIndex": 0,
        "parent": null,
    },
] as MinimumChatMessage[];

/**
 * Checks the validity of root nodes in the message tree.
 * @param roots - Array of root nodes in the message tree.
 * @throws {Error} If any root node has a parent.
 */
const checkRootValidity = <T extends MinimumChatMessage>(roots: MessageNode<T>[]) => {
    roots.forEach(root => {
        if (root.message.parent?.id) {
            throw new Error(`Root message with ID ${root.message.id} has a parent. This is invalid.`);
        }
    });
};

/**
 * Checks the validity of root nodes in the message tree.
 * @param roots - Array of root nodes in the message tree.
 * @throws {Error} If any root node has a parent.
 */
const checkChildParentRelationship = <T extends MinimumChatMessage>(messageMap: Map<string, MessageNode<T>>) => {
    messageMap.forEach((node, id) => {
        if (node.message.parent?.id) {
            const parentNode = messageMap.get(node.message.parent.id);
            if (!parentNode || !parentNode.children.includes(node)) {
                throw new Error(`Message with ID ${id} is not a child of its parent in the tree.`);
            }
        }
    });
};

/**
 * Ensures the integrity of the message tree by checking for cycles and verifying reachability from roots.
 * @param roots - Array of root nodes in the message tree.
 * @param messageMap - Map of message IDs to their corresponding nodes.
 * @throws {Error} If there are cycles or unreachable nodes.
 */
const checkTreeIntegrity = <T extends MinimumChatMessage>(roots: MessageNode<T>[], messageMap: Map<string, MessageNode<T>>): void => {
    const visited = new Set<string>();
    const traverseTree = (node: MessageNode<T>, visited: Set<string>): void => {
        if (visited.has(node.message.id)) {
            throw new Error(`Cycle detected at message with ID ${node.message.id}.`);
        }
        visited.add(node.message.id);
        node.children.forEach(child => traverseTree(child, visited));
    };

    roots.forEach(root => traverseTree(root, visited));
    messageMap.forEach((_, id) => {
        if (!visited.has(id)) {
            throw new Error(`Message with ID ${id} is not reachable from any root.`);
        }
    });
};

/**
 * Checks if the children of each node in the message map are sorted by their sequence number.
 * @param messageMap - Map of message IDs to their corresponding nodes.
 * @throws {Error} If children are not correctly sorted by sequence.
 */
const checkMessageEditOrder = <T extends MinimumChatMessage>(messageMap: Map<string, MessageNode<T>>): void => {
    messageMap.forEach((node, id) => {
        for (let i = 1; i < node.children.length; i++) {
            const currChildSequence = node.children[i]?.message?.sequence ?? -1; // Default to -1 if undefined
            const lastChildSequence = node.children[i - 1]?.message?.sequence ?? -1; // Default to -1 if undefined

            if (currChildSequence < lastChildSequence) {
                throw new Error(`Children of message with ID ${id} are not sorted by sequence.`);
            }
        }
    });
};

/**
 * Verifies that there are no orphan nodes in the message map.
 * @param messageMap - Map of message IDs to their corresponding nodes.
 * @param roots - Array of root nodes in the message tree.
 * @throws {Error} If there are orphan nodes in the message map.
 */
const checkNoOrphanNodes = <T extends MinimumChatMessage>(messageMap: Map<string, MessageNode<T>>, roots: MessageNode<T>[]): void => {
    messageMap.forEach((node, id) => {
        if (!node.message.parent?.id && !roots.includes(node)) {
            throw new Error(`Orphan node detected with ID ${id}.`);
        }
    });
};

type IntegrityCheck = "RootValidity" | "ChildParentRelationship" | "TreeIntegrity" | "MessageEditOrder" | "NoOrphanNodes";
/**
 * Performs a comprehensive integrity check on the message tree.
 * It includes checks for root validity, child-parent relationship,
 * tree integrity, message edit order, and the presence of orphan nodes.
 * 
 * @param roots - Array of root nodes in the message tree.
 * @param messageMap - Map of message IDs to their corresponding nodes.
 * @param skip - Array of checks to skip.
 * @throws {Error} If any integrity checks fail.
 */
const assertTreeIntegrity = <T extends MinimumChatMessage>(
    roots: MessageNode<T>[],
    messageMap: Map<string, MessageNode<T>>,
    skip?: IntegrityCheck[],
): void => {
    if (!skip) skip = [];
    if (!skip.includes("RootValidity")) checkRootValidity(roots);
    if (!skip.includes("ChildParentRelationship")) checkChildParentRelationship(messageMap);
    if (!skip.includes("TreeIntegrity")) checkTreeIntegrity(roots, messageMap);
    if (!skip.includes("MessageEditOrder")) checkMessageEditOrder(messageMap);
    if (!skip.includes("NoOrphanNodes")) checkNoOrphanNodes(messageMap, roots);
};

/**
 * Shuffles array in place using Fisher-Yates (aka Durstenfeld) shuffle algorithm.
 * @param array The array to shuffle.
 */
const shuffle = <T>(array: T[]): T[] => {
    for (let i = array.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [array[i], array[j]] = [array[j], array[i]] as [T, T]; // Swap elements
    }
    return array;
};

/**
 * Compares two sets of roots to ensure they have the same structure.
 * @param roots1 The roots of the first tree to compare.
 * @param roots2 The roots of the second tree to compare.
 */
const treesHaveSameStructure = <T extends MinimumChatMessage>(roots1: MessageNode<T>[], roots2: MessageNode<T>[]): boolean => {
    if (roots1.length !== roots2.length) return false;
    for (let i = 0; i < roots1.length; i++) {
        if (!compareSubtrees(roots1[i] as MessageNode<T>, roots2[i] as MessageNode<T>)) return false;
    }
    return true;
};

/**
 * Recursively compares two subtrees to ensure they have the same structure.
 * @param tree1 The first subtree to compare.
 * @param tree2 The second subtree to compare.
 */
const compareSubtrees = <T extends MinimumChatMessage>(tree1: MessageNode<T>, tree2: MessageNode<T>): boolean => {
    if (!tree1 && !tree2) return true;
    if (!tree1 || !tree2) return false;
    if (tree1.message.id !== tree2.message.id) return false;
    if (tree1.children.length !== tree2.children.length) return false;
    for (let i = 0; i < tree1.children.length; i++) {
        if (!compareSubtrees(tree1.children[i] as MessageNode<T>, tree2.children[i] as MessageNode<T>)) {
            return false;
        }
    }
    return true;
};

/**
 * Verifies that a tree node has a single child at each level and 
 * the message IDs increase sequentially.
 * @param node The starting node to verify.
 * @param expectedId The expected message ID for the current node.
 */
const verifySingleNodeStructureAndSequentialIds = <T extends MinimumChatMessage>(node: MessageNode<T>, expectedId: string): boolean => {
    if (!node) return true;
    // Check if the current node's ID matches the expected ID
    if (node.message.id !== expectedId) return false;
    // Check if the node has at most one child
    if (node.children.length > 1) return false;
    // If there's a child, recursively check it with the next expected ID
    if (node.children.length === 1) {
        return verifySingleNodeStructureAndSequentialIds(node.children[0] as MessageNode<T>, (parseInt(expectedId) + 1).toString());
    }
    return true;
};

/**
 * Counts the number of nodes in the tree starting from the given node.
 * @param node The starting node for counting.
 */
const countNodesInTree = <T extends MinimumChatMessage>(node: MessageNode<T>): number => {
    if (!node) return 0;

    // Count the current node
    let count = 1;

    // Recursively count children
    for (const child of node.children) {
        count += countNodesInTree(child);
    }

    return count;
};

const runCommonTests = (caseData: MinimumChatMessage[], caseTitle: string, skip?: IntegrityCheck[]) => {
    describe(`Common Tests for ${caseTitle}`, () => {
        it(`${caseTitle} - Tree is created successfully`, () => {
            const { result } = renderHook(() => useMessageTree(caseData));

            // Assuming assertTreeIntegrity is a function that can be used outside the class
            expect(() => assertTreeIntegrity(result.current.tree.roots, result.current.tree.map, skip)).not.toThrow();
        });

        it(`${caseTitle} - Result has the same number of messages as the input`, () => {
            const { result } = renderHook(() => useMessageTree(caseData));

            let totalNodes = 0;
            for (const root of result.current.tree.roots) {
                totalNodes += countNodesInTree(root);
            }

            expect(totalNodes).toBe(caseData.length);
        });

        it(`${caseTitle} - Roots and children are ordered by sequence`, () => {
            const { result } = renderHook(() => useMessageTree(caseData));
            const roots = result.current.tree.roots;

            for (let i = 0; i < roots.length - 1; i++) {
                const nextSequence = roots[i + 1]?.message?.sequence;
                expect(typeof nextSequence).toBe("number");
                expect(roots[i]?.message?.sequence).toBeLessThan(nextSequence ?? 0);
            }

            const checkChildOrder = (node: MessageNode<MinimumChatMessage>) => {
                for (let i = 0; i < node.children.length - 1; i++) {
                    const nextSequence = node.children[i + 1]?.message?.sequence;
                    expect(typeof nextSequence).toBe("number");
                    expect(node.children[i]?.message?.sequence).toBeLessThan(nextSequence ?? 0);
                }
                node.children.forEach(checkChildOrder);
            };

            roots.forEach(checkChildOrder);
        });

        it(`${caseTitle} - Result is the same when shuffled`, () => {
            const shuffledCaseData = shuffle([...caseData]);
            const { result: originalResult } = renderHook(() => useMessageTree(caseData));
            const { result: shuffledResult } = renderHook(() => useMessageTree(shuffledCaseData));

            expect(() => assertTreeIntegrity(originalResult.current.tree.roots, originalResult.current.tree.map, skip)).not.toThrow();
            expect(() => assertTreeIntegrity(shuffledResult.current.tree.roots, shuffledResult.current.tree.map, skip)).not.toThrow();

            expect(treesHaveSameStructure(originalResult.current.tree.roots, shuffledResult.current.tree.roots)).toBe(true);
        });
    });
};

describe("useMessageTree Hook", () => {
    beforeAll(() => {
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        jest.spyOn(console, "error").mockImplementation(() => { });
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        jest.spyOn(console, "warn").mockImplementation(() => { });
    });
    afterAll(() => {
        jest.restoreAllMocks();
    });

    runCommonTests(case1, "Case 1");
    it("Case 1 - One message per level, and in expected order (sequential IDs in this case)", () => {
        const { result } = renderHook(() => useMessageTree(case1));
        const roots = result.current.tree.roots;

        expect(roots).toHaveLength(1);
        const root = roots[0];
        const isValidStructure = verifySingleNodeStructureAndSequentialIds(root, "1");
        expect(isValidStructure).toBe(true);
    });

    runCommonTests(case2, "Case 2");
    it("Case 2 - Has 2 roots, with first having no children, and second having one message per level", () => {
        const { result } = renderHook(() => useMessageTree(case2));
        const roots = result.current.tree.roots;

        expect(roots).toHaveLength(2);
        const firstRoot = roots[0];
        const secondRoot = roots[1];

        expect(firstRoot?.children).toHaveLength(0);
        const isValidStructureForSecondRoot = verifySingleNodeStructureAndSequentialIds(secondRoot, "2");
        expect(isValidStructureForSecondRoot).toBe(true);
    });

    runCommonTests(case3, "Case 3");
    it("Case 3 - Proper hierarchical structure with 1 root, 1 child, 3 grandchildren, and 6 great-grandchildren", () => {
        const { result } = renderHook(() => useMessageTree(case3));
        const roots = result.current.tree.roots;

        expect(roots).toHaveLength(1);
        const root = roots[0];

        expect(root?.children).toHaveLength(1);
        const child = root?.children[0];

        expect(child?.children).toHaveLength(3);
        for (const grandchild of (child?.children ?? [])) {
            expect(grandchild.children).toHaveLength(2);
        }
    });

    runCommonTests(case4, "Case 4", ["ChildParentRelationship"]); // Skip check for missing parent, since this case is built that way on purpose
    it("Case 4 - 1 root without any branches. But a message in the middle was deleted", () => {
        const { result } = renderHook(() => useMessageTree(case4));
        const roots = result.current.tree.roots;

        expect(roots).toHaveLength(1);
        const root = roots[0];

        expect(root?.children).toHaveLength(1);
    });
});

describe("MessageTree Operations", () => {
    beforeAll(() => {
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        jest.spyOn(console, "error").mockImplementation(() => { });
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        jest.spyOn(console, "warn").mockImplementation(() => { });
    });
    afterAll(() => {
        jest.restoreAllMocks();
    });

    const initialMessages = [...case3];
    const newMessage = {
        id: "20",
        parent: { id: "8" },  // case3's 2nd grandchild's first child ID
        translations: [{
            id: "10020",
            language: "en",
            text: "This is a new added message.",
        }],
        created_at: "2021-10-02T01:00:00Z",
        sequence: 10,
    };

    it("Add Message - Adds a new message to the tree", () => {
        const { result } = renderHook(() => useMessageTree(initialMessages));

        act(() => {
            result.current.addMessages([newMessage]);
        });

        // Find and verify the newly added message
        const newMessageNode = result.current.tree.map.get(newMessage.id);
        expect(newMessageNode).toBeDefined();
        expect(newMessageNode!.message.id).toBe(newMessage.id);
    });

    it("Add Message - Adding all messages later is the same as initializing with those messages", () => {
        const { result: withInitialResult } = renderHook(() => useMessageTree(initialMessages));
        const { result: withoutInitialResult } = renderHook(() => useMessageTree<MinimumChatMessage>([]));

        // Add all messages to the empty tree
        act(() => {
            withoutInitialResult.current.addMessages(initialMessages);
        });

        // Verify that the resulting trees are the same
        expect(treesHaveSameStructure<MinimumChatMessage>(withInitialResult.current.tree.roots, withoutInitialResult.current.tree.roots)).toBe(true);
    });

    it("Add Message - Won't add the same message twice", () => {
        const { result } = renderHook(() => useMessageTree([...initialMessages, newMessage]));

        const initialNodeCount = result.current.tree.map.size;

        act(() => {
            result.current.addMessages([newMessage]);
        });

        expect(result.current.tree.map.size).toBe(initialNodeCount);
    });

    it("Add Message - Won't add the same messages (plural) twice", () => {
        const { result } = renderHook(() => useMessageTree<MinimumChatMessage>([]));

        act(() => {
            result.current.addMessages(case4);
            result.current.addMessages(case4);
            result.current.addMessages(case4);
        });

        expect(result.current.tree.map.size).toBe(case4.length);
    });

    it("Edit Message - Edits the newly added message", () => {
        const { result } = renderHook(() => useMessageTree([...initialMessages, newMessage]));

        const updatedMessage = {
            ...newMessage,
            translations: [{
                id: "10020",
                language: "en",
                text: "This is an edited message.",
            }],
        };

        act(() => {
            result.current.editMessage(updatedMessage);
        });

        const updatedNode = result.current.tree.map.get(updatedMessage.id);
        expect(updatedNode).toBeDefined();
        expect(updatedNode!.message).toEqual(updatedMessage);
    });

    it("Edit Message - Doesn't edit if the message is missing", () => {
        const { result } = renderHook(() => useMessageTree(initialMessages));

        const updatedMessage = {
            ...newMessage,
            translations: [{
                id: "10020",
                language: "en",
                text: "This is an edited message.",
            }],
        };

        act(() => {
            result.current.editMessage(updatedMessage);
        });

        const updatedNode = result.current.tree.map.get(updatedMessage.id);
        // Expect the node to be undefined since it wasn't added
        expect(updatedNode).toBeUndefined();
    });

    it("Remove Message - Removes the edited message", () => {
        const { result } = renderHook(() => useMessageTree(initialMessages));

        act(() => {
            result.current.removeMessages([newMessage.id]);
        });

        const removedNode = result.current.tree.map.get(newMessage.id);
        expect(removedNode).toBeUndefined();
    });

    it("Clear Messages - Resets the message tree", () => {
        const { result } = renderHook(() => useMessageTree(initialMessages));

        // Add a message to ensure the tree is not empty
        act(() => {
            result.current.addMessages([newMessage]);
        });

        // Verify that the tree is not empty
        expect(result.current.tree.map.size).toBeGreaterThan(0);
        expect(result.current.tree.roots.length).toBeGreaterThan(0);

        // Clear the message tree
        act(() => {
            result.current.clearMessages();
        });

        // Verify that the message tree is reset
        expect(result.current.tree.map.size).toBe(0);
        expect(result.current.tree.roots).toHaveLength(0);
    });

    it("Clear Messages - Clearing then adding messages is the same as initializing with those messages", () => {
        const { result: result1 } = renderHook(() => useMessageTree(initialMessages));
        const { result: result2 } = renderHook(() => useMessageTree(initialMessages));

        // Add all messages to the empty tree
        act(() => {
            result2.current.clearMessages();
            result2.current.addMessages(initialMessages);
        });

        // Verify that the resulting trees are the same
        expect(treesHaveSameStructure(result1.current.tree.roots, result2.current.tree.roots)).toBe(true);
    });
});

// Some additional tests that could be added in the future:
// - Circular references
// - Duplicate IDs
