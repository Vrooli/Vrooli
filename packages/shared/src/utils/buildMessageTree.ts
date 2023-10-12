import { ChatMessage } from "@local/shared";

/** Tree structure for displaying chat messages in the correct order */
export type MessageNode = {
    message: any;//ChatMessage;
    /** When there is more than one child, there are multiple versions (edits) of the message */
    children: MessageNode[];
};

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

/**
 * Constructs a hierarchical tree representation of message threads
 * from a flat array of ChatMessage objects.
 */
export class MessageTree {
    /** Map of message IDs to nodes */
    private messageMap: Map<string, MessageNode>;
    /** 
     * Array of nodes which are either the first message in the chat, or have 
     * not been linked to a parent message.
     */
    private roots: MessageNode[];

    constructor(private messages: any[]) {// ChatMessage[]) {
        // Initialize data structures
        this.messageMap = new Map<string, MessageNode>();
        this.roots = [];
        this.initMessageMap();
        // Build the tree
        this.buildTree();
    }

    private initMessageMap(): void {
        for (const message of this.messages) {
            this.messageMap.set(message.id, { message, children: [] });
        }
    }

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
    private buildTree(): void {
        for (const message of this.messages) {
            const node = this.messageMap.get(message.id);
            if (!node) {
                console.error(`Message ${message.id} not found in messageMap`);
                continue;
            }
            if (message.parentId) {
                const parentNode = this.messageMap.get(message.parentId);
                if (parentNode) {
                    // Since all edited versions have the same parentId and are not nested,
                    // they will be siblings in the tree structure.
                    parentNode.children.push(node);
                } else {
                    // If no parent node is found, consider it as a root node
                    this.roots.push(node);
                }
            } else {
                // If there's no parentId, it's a root node
                this.roots.push(node);
            }
        }
        // Sort the children of each node based on the sequence number to ensure
        // the correct order of message versions.
        this.sortChildren();
    }

    private sortChildren(): void {
        for (const root of this.roots) {
            this.sortNodeChildren(root);
        }
    }

    private sortNodeChildren(node: MessageNode): void {
        node.children.sort((a, b) => a.message.sequence - b.message.sequence);
        for (const child of node.children) {
            this.sortNodeChildren(child);  // Recursive call to sort children of children
        }
    }

    public getRoots(): MessageNode[] {
        return this.roots;
    }
}

// export type ChatMessage = { // For playground
//     __typename: 'ChatMessage';
//     created_at: string;
//     parent?: { id: string, created_at: string };
//     id: string;
//     translations: Array<{ id: string, language: string, text: string }>;
//   };


// First test case: Result should have messages in order from ID 1 to 10, each with node having a single child
const messages1 = [
    {
        id: "3",
        parent: { id: "2" },
        translations: [{
            id: "1001",
            language: "en",
            text: "Certainly! What seems to be the issue?",
        }],
        created_at: "2021-10-01T02:00:00Z",
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
    },
] as ChatMessage[];
const builder1 = new MessageTree(messages1);
const roots1 = builder1.getRoots();
