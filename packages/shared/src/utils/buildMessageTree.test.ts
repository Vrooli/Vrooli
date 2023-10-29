import { MessageNode, MessageTree, MinimumChatMessage } from "./buildMessageTree";

export class DebuggableMessageTree<T extends MinimumChatMessage> extends MessageTree<T> {
    constructor(private msgs: T[]) {
        super(msgs);
    }

    public assertTreeIntegrity(): void {
        // 1. Root Validity
        for (const root of this.roots) {
            if (root.message.parent?.id) {
                throw new Error(`Root message with ID ${root.message.id} has a parent. This is invalid.`);
            }
        }
        for (const [id, node] of this.messageMap.entries()) {
            if (!node.message.parent?.id && !this.roots.includes(node)) {
                throw new Error(`Message with ID ${id} doesn't have a parent but is not found in roots. This is invalid.`);
            }
        }

        // 3. Child-Parent Relationship
        for (const [id, node] of this.messageMap.entries()) {
            if (node.message.parent?.id) {
                const parentNode = this.messageMap.get(node.message.parent.id);
                if (!parentNode || !parentNode.children.includes(node)) {
                    throw new Error(`Message with ID ${id} is not a child of its parent in the tree.`);
                }
            }
        }

        // 4. Tree Integrity (Avoiding cycles and ensuring reachability from roots)
        const visited = new Set<string>();
        for (const root of this.roots) {
            this.traverseTree(root, visited);
        }
        for (const [id] of this.messageMap.entries()) {
            if (!visited.has(id)) {
                throw new Error(`Message with ID ${id} is not reachable from any root.`);
            }
        }

        // 5. Message Edit Order
        for (const [id, node] of this.messageMap.entries()) {
            for (let i = 1; i < node.children.length; i++) {
                const currChild = node.children[i];
                const lastChild = node.children[i - 1];
                if (!currChild?.message?.sequence || !lastChild?.message?.sequence) continue;
                if (currChild.message.sequence <= lastChild.message.sequence) {
                    throw new Error(`Children of message with ID ${id} are not sorted by sequence.`);
                }
            }
        }

        // 6. No Orphan Nodes
        for (const [id, node] of this.messageMap.entries()) {
            if (!node.message.parent?.id && !this.roots.includes(node)) {
                throw new Error(`Orphan node detected with ID ${id}.`);
            }
        }
    }

    // Helper recursive function for tree traversal
    private traverseTree(node: MessageNode<T>, visited: Set<string>): void {
        if (visited.has(node.message.id)) {
            throw new Error(`Cycle detected at message with ID ${node.message.id}.`);
        }
        visited.add(node.message.id);
        for (const child of node.children) {
            this.traverseTree(child, visited);
        }
    }
}

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

const runCommonTests = (caseData: MinimumChatMessage[], caseTitle: string) => {
    describe(`Common Tests for ${caseTitle}`, () => {
        it(`${caseTitle} - Tree is created successfully`, () => {
            const tree = new DebuggableMessageTree(caseData);
            expect(() => tree.assertTreeIntegrity()).not.toThrow();
        });

        it(`${caseTitle} - Result has the same number of messages as the input`, () => {
            const tree = new DebuggableMessageTree(caseData);
            // Count the total number of nodes in the tree
            let totalNodes = 0;
            for (const root of tree.getRoots()) {
                totalNodes += countNodesInTree(root);
            }
            expect(totalNodes).toBe(caseData.length);
        });

        it(`${caseTitle} - Roots and children are ordered by sequence`, () => {
            const tree = new DebuggableMessageTree<MinimumChatMessage>(caseData);
            const roots = tree.getRoots();

            // Check if roots are ordered by sequence
            for (let i = 0; i < roots.length - 1; i++) {
                const nextSequence = roots[i + 1]?.message?.sequence;
                expect(typeof nextSequence).toBe("number");
                expect(roots[i]?.message?.sequence).toBeLessThan(nextSequence ?? 0);
            }

            // Recursive function to check if children are ordered by sequence
            const checkChildOrder = (node: MessageNode<MinimumChatMessage>) => {
                for (let i = 0; i < node.children.length - 1; i++) {
                    const nextSequence = node.children[i + 1]?.message?.sequence;
                    expect(typeof nextSequence).toBe("number");
                    expect(node.children[i]?.message?.sequence).toBeLessThan(nextSequence ?? 0);
                }
                for (const child of node.children) {
                    checkChildOrder(child); // Recursion to check the order for the children
                }
            };

            for (const root of roots) {
                checkChildOrder(root);
            }
        });

        it(`${caseTitle} - Result is the same when shuffled`, () => {
            const shuffledCaseData = shuffle([...caseData]); // Create a shuffled copy of caseData
            const treeOriginal = new DebuggableMessageTree<MinimumChatMessage>(caseData);
            const treeShuffled = new DebuggableMessageTree<MinimumChatMessage>(shuffledCaseData);

            // Ensure trees are constructed without throwing errors
            expect(() => treeOriginal.assertTreeIntegrity()).not.toThrow();
            expect(() => treeShuffled.assertTreeIntegrity()).not.toThrow();

            // Ensure both trees have the same structure
            expect(treesHaveSameStructure(treeOriginal.getRoots() as MessageNode<MinimumChatMessage>[], treeShuffled.getRoots() as MessageNode<MinimumChatMessage>[])).toBe(true);
        });
    });
};

describe("DebuggableMessageTree", () => {
    runCommonTests(case1, "Case 1");
    it("Case 1 - One message per level, and in expected order (sequential IDs in this case)", () => {
        const tree = new DebuggableMessageTree<MinimumChatMessage>(case1);
        const roots = tree.getRoots();
        // There should only be one root
        expect(roots.length).toBe(1);
        const root = roots[0];
        // Assuming the first message ID is "1" and they increase sequentially.
        const isValidStructure = verifySingleNodeStructureAndSequentialIds(root as MessageNode<MinimumChatMessage>, "1");
        expect(isValidStructure).toBe(true);
    });

    runCommonTests(case2, "Case 2");
    it("Case 2 - Has 2 roots, with first having no children, and second having one message per level", () => {
        const tree = new DebuggableMessageTree<MinimumChatMessage>(case2);
        const roots = tree.getRoots();
        // There should be two roots
        expect(roots.length).toBe(2);
        const firstRoot = roots[0];
        const secondRoot = roots[1];
        // First root should have no children
        expect(firstRoot?.children.length).toBe(0);
        // The second root should have one message per level. We can verify this by reusing the verifySingleNodeStructureAndSequentialIds function.
        // Assuming the first message ID of the second root is "2" and they increase sequentially.
        const isValidStructureForSecondRoot = verifySingleNodeStructureAndSequentialIds(secondRoot as MessageNode<MinimumChatMessage>, "2");
        expect(isValidStructureForSecondRoot).toBe(true);
    });

    runCommonTests(case3, "Case 3");
    it("Case 3 - Proper hierarchical structure with 1 root, 1 child, 3 grandchildren, and 6 great-grandchildren", () => {
        const tree = new DebuggableMessageTree<MinimumChatMessage>(case3);
        const roots = tree.getRoots();
        // There should only be one root
        expect(roots.length).toBe(1);
        const root = roots[0];
        // Root should have 1 child
        expect(root?.children?.length).toBe(1);
        const child = root?.children[0];
        // Child should have 3 grandchildren
        expect(child?.children?.length).toBe(3);
        // Each grandchild should have 2 children
        for (const grandchild of (child?.children ?? [])) {
            expect(grandchild.children.length).toBe(2);
        }
    });
});

describe("MessageTree Operations", () => {
    const tree = new DebuggableMessageTree<MinimumChatMessage>([...case3]); // Create a copy of case3 to not mutate the original

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

    it("Add Message - Adds a new message to case3's 2nd grandchild's first child", () => {
        tree.addMessage(newMessage);

        // First message
        const root = tree.getRoots()[0] as MessageNode<MinimumChatMessage>;
        // First child
        const child = root.children[0] as MessageNode<MinimumChatMessage>;
        // Second grandchild
        const grandchild = child.children[1] as MessageNode<MinimumChatMessage>;
        // Second grandchild's first child
        const greatGrandchild = grandchild.children[0] as MessageNode<MinimumChatMessage>;
        // (Hopefully) the new message
        const greatGreatGrandchild = greatGrandchild.children[0] as MessageNode<MinimumChatMessage>;
        // Check if the new message was added correctly (i.e. is a child, and is the only child)
        expect(greatGreatGrandchild.message.id).toBe(newMessage.id);
        expect(greatGrandchild.children.length).toBe(1);
        expect(greatGreatGrandchild.children.length).toBe(0);
    });

    it("Edit Message - Edits the newly added message", () => {
        const updatedMessage = {
            ...newMessage,
            translations: [{
                id: "10020",
                language: "en",
                text: "This is an edited message.",
            }],
        };

        tree.editMessage(updatedMessage);

        // First message
        const root = tree.getRoots()[0] as MessageNode<MinimumChatMessage>;
        // First child
        const child = root.children[0] as MessageNode<MinimumChatMessage>;
        // Second grandchild
        const grandchild = child.children[1] as MessageNode<MinimumChatMessage>;
        // Second grandchild's first child
        const greatGrandchild = grandchild.children[0] as MessageNode<MinimumChatMessage>;
        // (Hopefully) the edited message
        const greatGreatGrandchild = greatGrandchild.children[0] as MessageNode<MinimumChatMessage>;
        // Perform same checks for the new message, but also make sure that the text was updated
        expect(greatGreatGrandchild.message.id).toBe(newMessage.id);
        expect(greatGrandchild.children.length).toBe(1);
        expect(greatGreatGrandchild.children.length).toBe(0);
        expect((greatGreatGrandchild.message as unknown as { translations: { text: string }[] }).translations[0]?.text).toBe("This is an edited message.");
    });

    it("Remove Message - Removes the edited message and ensures tree structure matches original case3", () => {
        tree.removeMessage("20");

        const treeOriginal = new DebuggableMessageTree<MinimumChatMessage>(case3);

        // Ensure both trees have the same structure
        expect(treesHaveSameStructure(treeOriginal.getRoots() as MessageNode<MinimumChatMessage>[], tree.getRoots() as MessageNode<MinimumChatMessage>[])).toBe(true);
    });
});

// Some additional tests that could be added in the future:
// - Circular references
// - Orphan messages (i.e. references to parents that don't exist)
// - Duplicate IDs
