import { ChatMessage } from "@local/shared";

/** Tree structure for displaying chat messages in the correct order */
export type MessageNode = {
    message: ChatMessage;
    /** When there is more than one child, there are multiple versions of the message */
    children: MessageNode[];
};

// TODO parent should be connected automatically
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

/**
 * Constructs a hierarchical tree representation of message threads
 * from a flat array of ChatMessage objects, based on their forkId and isFork fields.
 */
export class MessageTreeBuilder {
    /** Map of message IDs to nodes */
    private messageMap: Map<string, MessageNode>;
    /** 
     * Array of nodes which are either the first message in the chat, or have 
     * not been linked to a parent message.
     */
    private roots: MessageNode[];

    constructor(private messages: ChatMessage[]) {
        // Initialize data structures
        this.messageMap = new Map<string, MessageNode>();
        this.roots = [];
        this.initMessageMap();
        // Build the tree
        this.buildTree();
        // Go back through the tree and fix any errors, typically caused 
        // by race conditions where multiple messages set the same forkId (parent).
        this.processTreeRoots();
    }

    private initMessageMap(): void {
        for (const message of this.messages) {
            this.messageMap.set(message.id, { message, children: [] });
        }
    }

    /**
     * 2. Loops through each message and links nodes based on the forkId and isFork fields:
     *    - If forkId is null, the message is a root message.
     *    - If forkId points to an existing message and isFork is false, the forkId message is the 
     *      parent and the current message is added as a child.
     *    - If forkId points to an existing message and isFork is true, the current message is a 
     *      sibling of the forkId message. The function finds the parent of the forkId message and 
     *      adds the current message as a sibling, inserting it in the correct position based on 
     *      the created_at date.
     *    - If forkId points to a missing message, the current message is treated as a root message.
     */
    private buildTree(): void {
        for (const message of this.messages) {
            const node = this.messageMap.get(message.id);
            if (!node) {
                console.error(`Message ${message.id} not found in messageMap`);
                continue;
            }
            if (message.fork?.id) {
                const forkNode = this.messageMap.get(message.fork.id);
                if (forkNode) {
                    forkNode.children.push(node);
                } else {
                    this.roots.push(node);
                }
            } else {
                this.roots.push(node);
            }
        }
    }

    private processTreeRoots(): void {
        for (const root of this.roots) {
            this.processTree(root);
        }
    }

    /**
     * Processes a subtree to correct structure, sort children, and handle race conditions.
     * 
     * @param node - The root node of the subtree to process.
     */
    private processTree(node: MessageNode): void {
        // Sort children by created_at date.
        node.children.sort((a, b) => new Date(a.message.created_at).getTime() - new Date(b.message.created_at).getTime());

        const siblings: MessageNode[] = [];
        const responses: MessageNode[] = [];

        // Separate children into siblings (isFork = true) and responses (isFork = false).
        for (const child of node.children) {
            if (child.message.isFork) {
                siblings.push(child);
            } else {
                responses.push(child);
            }
        }

        // Handle race condition by re-arranging nodes.
        // If there are multiple siblings, find their correct parent and move them.
        for (const sibling of siblings) {
            const parent = this.findCorrectParent(sibling, node);
            if (parent !== node) {
                parent.children.push(sibling);
            }
        }

        // Re-assign the children array with the correct structure.
        node.children = [...responses, ...siblings];

        // Recursively process the children.
        for (const child of node.children) {
            this.processTree(child);
        }
    }

    /**
     * Finds the correct parent for a given node, handling race conditions.
     * 
     * @param node - The node whose correct parent is to be found.
     * @param currentNode - The current node being processed.
     * @returns The correct parent node.
     */
    private findCorrectParent(node: MessageNode, currentNode: MessageNode): MessageNode {
        let correctParent: MessageNode = currentNode;  // Initialize with the current node
        let parentNode = currentNode;  // Start the traversal at the current node

        while (parentNode) {
            // Check if the parentNode or any of its siblings are before the given node
            // and after the given node's current parent.
            if (new Date(parentNode.message.created_at).getTime() < new Date(node.message.created_at).getTime() &&
                new Date(parentNode.message.created_at).getTime() > new Date(node.message.fork?.created_at).getTime()) {
                correctParent = parentNode;
                break;  // If a valid parent is found, break the loop
            }

            // Check siblings of the parentNode
            const siblings = this.getSiblings(parentNode);
            for (const sibling of siblings) {
                if (new Date(sibling.message.created_at).getTime() < new Date(node.message.created_at).getTime() &&
                    new Date(sibling.message.created_at).getTime() > new Date(node.message.fork?.created_at).getTime()) {
                    correctParent = sibling;
                    break;  // If a valid parent is found among siblings, break the loop
                }
            }

            // Move up the tree to the parent of the parentNode
            const newParentNode = this.findParentOf(parentNode);
            if (newParentNode) {
                parentNode = newParentNode;
            } else {
                break;  // If no parent is found, break the loop to prevent an infinite loop
            }
        }
        return correctParent;
    }

    /**
     * Gets the siblings of a given node.
     */
    private getSiblings(node: MessageNode): MessageNode[] {
        const parent = this.findParentOf(node);
        return parent ? parent.children.filter(child => child !== node) : [];
    }

    /**
     * Finds the parent of a given node.
     */
    private findParentOf(node: MessageNode): MessageNode | null {
        if (node.message.fork?.id) {
            const parentNode = this.messageMap.get(node.message.fork.id);
            return parentNode || null;
        }
        return null;
    }

    public getRoots(): MessageNode[] {
        return this.roots;
    }
}

// export type ChatMessage = { // For playground
//     __typename: 'ChatMessage';
//     created_at: string;
//     fork?: { id: string, created_at: string };
//     id: string;
//     isFork: boolean;
//     translations: Array<{ id: string, language: string, text: string }>;
//   };


// First test case: Result should have messages in order from ID 1 to 10, each with node having a single child
const messages1 = [
    {
        id: "3",
        isFork: false,
        fork: { id: "2" },
        translations: [{
            id: "1001",
            language: "en",
            text: "Certainly! What seems to be the issue?",
        }],
        created_at: "2021-10-01T02:00:00Z",
    },
    {
        id: "1",
        isFork: false,
        fork: null,
        translations: [{
            id: "1001",
            language: "en",
            text: "Hello, how can I help you?",
        }],
        created_at: "2021-10-01T00:00:00Z",
    },
    {
        id: "2",
        isFork: false,
        fork: { id: "1" },
        translations: [{
            id: "1001",
            language: "en",
            text: "I need assistance with my account.",
        }],
        created_at: "2021-10-01T01:00:00Z",
    },
    {
        id: "4",
        isFork: false,
        fork: { id: "2" },
        translations: [{
            id: "1001",
            language: "en",
            text: "I can't access my account.",
        }],
        created_at: "2021-10-01T03:00:00Z",
    },
    {
        id: "5",
        isFork: false,
        fork: { id: "4" },
        translations: [{
            id: "1001",
            language: "en",
            text: "I see. Have you tried resetting your password?",
        }],
        created_at: "2021-10-01T04:00:00Z",
    },
    {
        id: "6",
        // text: "Yes, but I didn't receive the reset email.",
        isFork: false,
        fork: { id: "5" },
        translations: [{
            id: "1001",
            language: "en",
            text: "Yes, but I didn't receive the reset email.",
        }],
        created_at: "2021-10-01T05:00:00Z",
    },
    {
        id: "7",
        isFork: false,
        fork: { id: "6" },
        translations: [{
            id: "1001",
            language: "en",
            text: "I can help with that. Can you provide your username?",
        }],
        created_at: "2021-10-01T06:00:00Z",
    },
    {
        id: "8",
        isFork: false,
        fork: { id: "7" },
        translations: [{
            id: "1001",
            language: "en",
            text: "My username is johndoe123.",
        }],
        created_at: "2021-10-01T07:00:00Z",
    },
    {
        id: "9",
        isFork: false,
        fork: { id: "8" },
        translations: [{
            id: "1001",
            language: "en",
            text: "Thank you, John. I have sent a password reset link to your email.",
        }],
        created_at: "2021-10-01T08:00:00Z",
    },
    {
        id: "10",
        isFork: false,
        fork: { id: "9" },
        translations: [{
            id: "1001",
            language: "en",
            text: "Got it. Thanks!",
        }],
        created_at: "2021-10-01T09:00:00Z",
    },
] as ChatMessage[];
const builder1 = new MessageTreeBuilder(messages1);
const roots1 = builder1.getRoots();
