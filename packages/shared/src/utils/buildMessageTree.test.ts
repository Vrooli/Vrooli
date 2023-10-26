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

// First test case: Result should have messages in order from ID 1 to 10, each with node having a single child
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
        parent: { id: "2" },
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

describe("DebuggableMessageTree", () => {
    it("should assert tree integrity without errors for case 1", () => {
        const tree = new DebuggableMessageTree(case1);
        expect(() => tree.assertTreeIntegrity()).not.toThrow();
    });
});
