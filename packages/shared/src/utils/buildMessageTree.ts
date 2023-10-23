import { ChatMessage } from "@local/shared";

/** Tree structure for displaying chat messages in the correct order */
export type MessageNode = {
    message: ChatMessage;
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
    protected messageMap: Map<string, MessageNode>;
    /** 
     * Array of nodes which are either the first message in the chat, or have 
     * not been linked to a parent message.
     */
    protected roots: MessageNode[];

    constructor(private messages: ChatMessage[]) {
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
            if (message.parent?.id) {
                const parentNode = this.messageMap.get(message.parent.id);
                if (parentNode) {
                    // Since all edited versions have the same parentId and are not nested,
                    // they will be siblings in the tree structure.
                    parentNode.children.push(node);
                } else {
                    // If no parent node is found, consider it as a root node
                    this.roots.push(node);
                }
            } else {
                // If there's no parent ID, we'll assume it's a root node
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

    public addMessage(message: ChatMessage): void {
        // Create a new node for the message
        const newNode: MessageNode = {
            message,
            children: [],
        };

        // Update the messageMap with the new node
        this.messageMap.set(message.id, newNode);

        if (message.parent?.id) {
            const parentNode = this.messageMap.get(message.parent.id);
            if (parentNode) {
                // Add the new message node as a child of the parent node
                parentNode.children.push(newNode);
                // Sort the children of the parent node to maintain order
                this.sortNodeChildren(parentNode);
            } else {
                // If no parent node is found, consider it as a root node
                this.roots.push(newNode);
            }
        } else {
            // If there's no parent ID, it's a root node
            this.roots.push(newNode);
        }
    }

    public removeMessage(messageId: string): void {
        const nodeToRemove = this.messageMap.get(messageId);
        if (!nodeToRemove) {
            console.error(`Message ${messageId} not found in messageMap`);
            return;
        }

        // Remove the node from the messageMap
        this.messageMap.delete(messageId);

        if (nodeToRemove.message.parent?.id) {
            const parentNode = this.messageMap.get(nodeToRemove.message.parent.id);
            if (parentNode) {
                // Remove the message node from the parent's children
                parentNode.children = parentNode.children.filter(child => child.message.id !== messageId);

                // If the nodeToRemove had children, link them to its parent
                parentNode.children.push(...nodeToRemove.children);
            }
        } else {
            // If the node to remove is a root node, remove it from the roots array
            this.roots = this.roots.filter(root => root.message.id !== messageId);
        }
    }

    /**
     * Edits the content of a message in the tree. DOES NOT 
     * add a new version of the message. This method is useful when 
     * chatting with other participants, where branching is disabled.
     */
    public editMessage(messageId: string, updatedMessage: ChatMessage): void {
        const node = this.messageMap.get(messageId);
        if (node) {
            // Update the message with the new data
            node.message = updatedMessage;
        } else {
            console.error(`Message ${messageId} not found in messageMap. Cannot edit.`);
        }
    }
}
