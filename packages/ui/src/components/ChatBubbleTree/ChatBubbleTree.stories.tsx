import { ChatMessageShape, ChatMessageStatus, ChatSocketEventPayloads, generatePK, MINUTES_10_MS } from "@local/shared";
import { Box } from "@mui/material";
import { action } from "@storybook/addon-actions";
import { useCallback, useEffect, useState } from "react";
import { loggedOutSession, signedInPremiumWithCreditsSession, signedInUserId } from "../../__test/storybookConsts.js";
import { MessageTree } from "../../hooks/messages.js";
import { pagePaddingBottom } from "../../styles.js";
import { BranchMap } from "../../utils/localStorage.js";
import { ChatBubbleTree } from "./ChatBubbleTree.js";

const bot1Id = generatePK().toString();
const botMessage1Id = generatePK().toString();
const userMessage1Id = generatePK().toString();
const botMessage2Version1Id = generatePK().toString();
const botMessage2Version2Id = generatePK().toString();
const userMessage2Id = generatePK().toString();

const mockMessages: ChatMessageShape[] = [
    // Initial bot message
    {
        __typename: "ChatMessage" as const,
        id: botMessage1Id,
        // eslint-disable-next-line no-magic-numbers
        createdAt: new Date(Date.now() - 10 * MINUTES_10_MS).toISOString(),
        // eslint-disable-next-line no-magic-numbers
        updatedAt: new Date(Date.now() - 10 * MINUTES_10_MS).toISOString(),
        translations: [{
            __typename: "ChatMessageTranslation" as const,
            id: generatePK().toString(),
            language: "en",
            text: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Vivamus lacinia odio vitae vestibulum vestibulum.",
        }],
        reactionSummaries: [{
            __typename: "ReactionSummary" as const,
            count: 2,
            emoji: "👍",
        }],
        status: "sent" as ChatMessageStatus,
        user: {
            id: bot1Id,
            name: "AI Assistant",
            handle: "ai_assistant",
            isBot: true,
            __typename: "User",
        },
        versionIndex: 0,
        parent: null,
    },
    // Initial user message
    {
        __typename: "ChatMessage" as const,
        id: userMessage1Id,
        // eslint-disable-next-line no-magic-numbers
        createdAt: new Date(Date.now() - 9 * MINUTES_10_MS).toISOString(),
        // eslint-disable-next-line no-magic-numbers
        updatedAt: new Date(Date.now() - 9 * MINUTES_10_MS).toISOString(),
        translations: [{
            __typename: "ChatMessageTranslation" as const,
            id: generatePK().toString(),
            language: "en",
            text: `Lorem ipsum dolor sit amet, consectetur adipiscing elit. Quisque pellentesque nunc arcu, non scelerisque odio scelerisque ut. Cras volutpat in odio ac venenatis. Ut eleifend placerat ipsum, eu imperdiet lectus dapibus nec. Cras pharetra sollicitudin enim nec efficitur. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Ut mollis maximus varius. Donec eleifend turpis molestie arcu elementum, ut cursus nisi congue. Suspendisse sed nibh dapibus, tincidunt enim vitae, consectetur felis. Suspendisse vel nunc eros. Pellentesque sit amet sem dolor. Maecenas fermentum laoreet elit eget malesuada. Maecenas metus odio, dictum nec nunc lobortis, molestie aliquet justo.

Cras porttitor mauris eget dui dapibus maximus. Aliquam hendrerit risus et lacus finibus lobortis. Nulla molestie ligula elit, vitae molestie ipsum tempor vitae. Praesent rhoncus nibh neque, ac hendrerit dui varius eget. Nullam gravida quam mauris, vitae dapibus mi egestas id. Sed blandit interdum magna, sed tincidunt sem tempor id. Phasellus accumsan elit eu sapien vulputate finibus.

Aenean non sapien eget odio scelerisque feugiat. Praesent eget massa consectetur, euismod erat eget, maximus augue. Sed ut malesuada est. In mauris nisi, pretium consequat arcu ut, bibendum luctus ligula. Nulla pharetra iaculis blandit. Nullam laoreet nec ipsum eget scelerisque. Aenean vehicula pulvinar placerat. Duis iaculis condimentum tortor.

Duis facilisis diam est, sit amet volutpat ex fringilla eget. Morbi nisl erat, dapibus ut eros egestas, cursus faucibus urna. Curabitur nec ligula diam. Maecenas pretium vitae ligula ut dictum. Fusce cursus iaculis eleifend. In at enim vel est fermentum pharetra at vitae nisi. Aliquam erat volutpat.

In semper tortor ac mi condimentum rutrum. Mauris pulvinar vehicula urna sed sollicitudin. Morbi sit amet congue elit. Pellentesque eu erat eu eros vehicula lacinia. Proin vitae arcu ac nulla commodo faucibus. Aliquam a dolor tincidunt, efficitur augue vel, finibus risus. Mauris venenatis pellentesque tellus eget euismod. Donec consectetur luctus nulla id scelerisque. Fusce mollis felis sed euismod varius. Ut tempor nunc ac nibh facilisis mattis. Integer nec tellus augue. Suspendisse lobortis fermentum accumsan. Fusce imperdiet tristique arcu, id suscipit massa vestibulum sit amet. Morbi nec nisi eget leo tincidunt porttitor sit amet luctus massa.`,
        }],
        reactionSummaries: [],
        status: "sent" as ChatMessageStatus,
        user: {
            __typename: "User" as const,
            id: signedInUserId,
            name: "Test User",
            handle: "testuser",
            isBot: false,
        },
        versionIndex: 0,
        parent: {
            __typename: "ChatMessageParent" as const,
            id: botMessage1Id,
        },
    },
    // Second bot message - version 1
    {
        __typename: "ChatMessage" as const,
        id: botMessage2Version1Id,
        // eslint-disable-next-line no-magic-numbers
        createdAt: new Date(Date.now() - 8 * MINUTES_10_MS).toISOString(),
        // eslint-disable-next-line no-magic-numbers
        updatedAt: new Date(Date.now() - 8 * MINUTES_10_MS).toISOString(),
        translations: [{
            __typename: "ChatMessageTranslation" as const,
            id: generatePK().toString(),
            language: "en",
            text: `Suspendisse pulvinar feugiat lectus sit amet porta. Donec id sapien odio. Vivamus mauris ligula, bibendum sollicitudin urna ut, semper fermentum nulla. Nulla viverra facilisis porttitor. Integer cursus ullamcorper metus in lacinia. Vivamus nulla nunc, pellentesque et mattis id, consectetur ut dolor. Nam imperdiet nibh in venenatis blandit. Aliquam at elit vulputate, varius ante gravida, gravida enim. Morbi sollicitudin purus et condimentum luctus. Nullam pellentesque malesuada nisl, ut placerat nisl porta vitae. Morbi velit enim, tristique eget rutrum ut, vulputate vel neque. Quisque accumsan eu enim nec pretium. Vivamus porttitor mollis elit id ultricies.

Fusce lobortis viverra ullamcorper. Donec sed dolor semper magna facilisis maximus a eu ligula. Morbi eu ligula pretium, dictum ligula vitae, dapibus sapien. Praesent in sodales diam, nec dictum mi. Aenean lacus nibh, ultricies vel hendrerit et, dignissim non est. Nulla gravida vel quam vel pharetra. Sed bibendum, libero ullamcorper hendrerit gravida, odio lorem finibus velit, a mollis mauris eros a diam. Curabitur quam tellus, euismod vel fringilla nec, convallis eu mauris. Etiam ut metus at est cursus placerat eget et augue.

In et arcu augue. Ut ac convallis neque, eget ultricies elit. Cras eget ipsum sit amet justo molestie dapibus. Suspendisse cursus diam ac libero dapibus dignissim. Proin dapibus tempor nulla, ac cursus tortor congue nec. Vestibulum tempus tempus erat, non varius risus maximus tincidunt. Donec ac tincidunt mi.

Suspendisse eu sodales nisi, sed malesuada augue. Morbi eget porttitor purus, eget dignissim neque. Fusce eu viverra nulla. Integer a varius tortor, sed ultrices ante. Aenean ultrices lobortis tempus. Ut vestibulum sit amet arcu vitae molestie. Vestibulum aliquet libero sit amet diam congue, fringilla lacinia tellus auctor. Integer scelerisque tortor sed facilisis pharetra. Sed pharetra eros ex, posuere viverra tellus molestie non. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia curae; Nam consequat nibh id hendrerit pulvinar. In eleifend hendrerit massa, porta auctor enim aliquam vitae. Praesent eu leo lectus.`,
        }],
        reactionSummaries: [{
            __typename: "ReactionSummary" as const,
            count: 1,
            emoji: "🤔",
        }],
        status: "sent" as ChatMessageStatus,
        user: {
            __typename: "User" as const,
            id: bot1Id,
            name: "AI Assistant",
            handle: "ai_assistant",
            isBot: true,
        },
        versionIndex: 0,
        parent: {
            __typename: "ChatMessageParent" as const,
            id: userMessage1Id,
        },
    },
    // Second bot message - version 2 (i.e. has same parent as version 1, but version index is 1)
    {
        __typename: "ChatMessage" as const,
        id: botMessage2Version2Id,
        // eslint-disable-next-line no-magic-numbers
        createdAt: new Date(Date.now() - 7 * MINUTES_10_MS).toISOString(),
        // eslint-disable-next-line no-magic-numbers
        updatedAt: new Date(Date.now() - 7 * MINUTES_10_MS).toISOString(),
        translations: [{
            __typename: "ChatMessageTranslation" as const,
            id: generatePK().toString(),
            language: "en",
            text: `Etiam porttitor lectus dolor, eu cursus dolor sodales vel. Pellentesque semper metus id arcu posuere elementum. Cras nec eros non urna accumsan varius et nec orci. Morbi facilisis nibh sed ligula dictum, id tristique libero aliquam. Mauris consectetur libero eget ex laoreet gravida. Vivamus dignissim mattis diam, sollicitudin aliquet nisi tincidunt aliquet. Sed sapien neque, accumsan a luctus ac, ullamcorper at ex. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia curae; Suspendisse in magna in purus maximus gravida id non nunc. Praesent ut urna ipsum. Maecenas vulputate enim nec nisl tincidunt tristique. Pellentesque habitant morbi tristique senectus et netus et malesuada fames ac turpis egestas. Praesent dapibus, tellus vel rutrum cursus, massa purus condimentum sem, at sollicitudin metus dui sit amet sem. Integer egestas purus sit amet odio maximus congue vel vel est.

Class aptent taciti sociosqu ad litora torquent per conubia nostra, per inceptos himenaeos. Suspendisse potenti. In ut felis eros. Etiam egestas sed libero non ullamcorper. Donec aliquam, justo sit amet consectetur vulputate, odio metus hendrerit tortor, eu semper dui odio sit amet erat. Aliquam ullamcorper quis ex sit amet tincidunt. Nunc convallis lectus augue, vel vestibulum nisl tempor quis. Nam aliquam lorem lectus, vestibulum porttitor dolor viverra et. Nulla et urna sed orci sagittis ultrices sed eget lectus. Cras non erat et leo posuere tempor suscipit a nibh. Nam vitae ornare quam, non venenatis ex. Donec finibus lobortis elementum. In hac habitasse platea dictumst.

Aliquam est sapien, venenatis ac lacus in, commodo mollis felis. Integer cursus, ipsum sit amet consequat malesuada, sapien justo varius metus, et accumsan nunc est vitae sem. Nunc in finibus elit. Suspendisse potenti. Ut pharetra erat at velit tincidunt, eu consectetur quam tincidunt. Vivamus in odio ac purus viverra laoreet. Donec neque urna, hendrerit tempus iaculis condimentum, euismod ac orci. Donec nulla ipsum, suscipit non dictum a, semper a lectus. Ut luctus vehicula aliquam. Aenean commodo diam et euismod interdum. Proin ac arcu dolor. Vestibulum quis sodales elit, non hendrerit est. Vivamus eget quam in nibh sodales blandit et ac lacus.

Suspendisse sit amet cursus velit, efficitur pulvinar ex. Donec consectetur dolor in tortor tincidunt, sed suscipit erat dictum. Vestibulum risus tellus, dignissim ultrices pellentesque quis, varius vitae arcu. Sed elementum in mauris in dapibus. Fusce nec dui elementum, cursus ligula maximus, tristique eros. Aliquam venenatis sollicitudin arcu, ut aliquet tellus fringilla sit amet. Nulla ultricies quam ut lectus aliquet, porttitor tincidunt massa scelerisque. Pellentesque id massa eget ante elementum imperdiet a aliquam libero. Phasellus sed est iaculis, iaculis diam vitae, mollis libero. Duis eu maximus sem. Cras semper dolor augue, a posuere nisi volutpat ut. Morbi et felis dui. Nulla in commodo massa. Etiam ornare ornare ipsum.

Mauris ut arcu elit. Donec condimentum urna eros, a vehicula massa mattis non. Mauris venenatis purus nec nisl ullamcorper sagittis. Orci varius natoque penatibus et magnis dis parturient montes, nascetur ridiculus mus. Morbi feugiat massa felis, id accumsan diam rutrum et. Vestibulum hendrerit lectus augue. Curabitur sed eleifend velit.

Vivamus efficitur ullamcorper felis et dignissim. Etiam tempus, mi vitae tempus tincidunt, nunc leo tincidunt mi, eget dignissim ipsum lorem et magna. Fusce feugiat lectus sed justo egestas lobortis. Nam tempor lacus et sapien sodales, eu elementum enim pharetra. Vivamus interdum lorem a arcu sagittis, vel ultricies nisl congue. Nullam pulvinar sem id enim rutrum, id sagittis sem sagittis. Integer euismod scelerisque purus at convallis. Proin egestas, tortor molestie malesuada euismod, mi sem malesuada purus, a vulputate nunc justo et libero. Nullam fringilla dignissim lorem, quis euismod purus posuere aliquet. Fusce metus odio, efficitur nec lacus nec, aliquam dictum felis. Quisque sem diam, efficitur at egestas ut, auctor ac urna. Suspendisse dictum finibus felis, quis auctor velit convallis et. Sed erat felis, ornare nec ipsum eu, bibendum semper orci. Praesent sit amet ante ac est ornare congue vel a dolor.

Quisque elementum mi non ligula blandit egestas. Maecenas laoreet ante urna, pulvinar pellentesque ipsum interdum in. Sed elit lectus, aliquam ac mattis rhoncus, pulvinar in urna. Morbi tincidunt lorem erat, a congue nisl lacinia id. Nullam vitae odio quis sem efficitur finibus. Suspendisse ac dui est. Mauris aliquam finibus imperdiet. Fusce eleifend magna at venenatis consequat. Sed interdum, felis eu varius elementum, eros diam porta velit, at pretium nulla lectus gravida felis.`,
        }],
        reactionSummaries: [],
        status: "sent" as ChatMessageStatus,
        user: {
            __typename: "User" as const,
            id: bot1Id,
            name: "AI Assistant",
            handle: "ai_assistant",
            isBot: true,
        },
        versionIndex: 1,
        parent: {
            __typename: "ChatMessageParent" as const,
            id: userMessage1Id,
        },
    },
    // Second user message. After version 1 of the second bot message
    {
        __typename: "ChatMessage" as const,
        id: userMessage2Id,
        // eslint-disable-next-line no-magic-numbers
        createdAt: new Date(Date.now() - 6 * MINUTES_10_MS).toISOString(),
        // eslint-disable-next-line no-magic-numbers
        updatedAt: new Date(Date.now() - 6 * MINUTES_10_MS).toISOString(),
        translations: [{
            __typename: "ChatMessageTranslation" as const,
            id: generatePK().toString(),
            language: "en",
            text: "This is a reply to your message.",
        }],
        reactionSummaries: [],
        status: "sent" as ChatMessageStatus,
        user: {
            __typename: "User" as const,
            id: signedInUserId,
            name: "Test User",
            handle: "testuser",
            isBot: false,
        },
        versionIndex: 0,
        parent: {
            __typename: "ChatMessageParent" as const,
            id: botMessage2Version1Id,
        },
    },
];

// Create message tree
function createMessageTree(messages: ChatMessageShape[]): MessageTree<ChatMessageShape> {
    const tree = new MessageTree<ChatMessageShape>();

    // Add all messages to the tree
    messages.forEach(msg => {
        tree.addMessageToTree(msg);
    });

    return tree;
}

function useStoryMessages(initialMessages: ChatMessageShape[]) {
    const [tree, setTree] = useState(() => createMessageTree(initialMessages));
    const [branches, setBranches] = useState<BranchMap>({});

    const addMessages = useCallback((newMessages: ChatMessageShape[]) => {
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

    const handleReactionAdd = useCallback((message: ChatMessageShape, emoji: string) => {
        if (message.status !== "sent") return;

        // Update the message with the new reaction
        const existingReaction = message.reactionSummaries.find((r) => r.emoji === emoji);
        const updatedReactionSummaries = existingReaction
            ? message.reactionSummaries.map((r) => r.emoji === emoji ? { ...r, count: r.count + 1 } : r)
            : [...message.reactionSummaries, { __typename: "ReactionSummary" as const, emoji, count: 1 }];

        editMessage({
            ...message,
            reactionSummaries: updatedReactionSummaries,
        });

        action("handleReactionAdd")({ message, emoji });
    }, [editMessage]);

    const handleEdit = useCallback((message: ChatMessageShape) => {
        action("handleEdit")(message);
    }, []);

    const handleRegenerateResponse = useCallback((message: ChatMessageShape) => {
        // Create a new message with the same text but a new ID and higher version index
        const newMessage: ChatMessageShape = {
            ...message,
            id: generatePK().toString(),
            versionIndex: (message.versionIndex ?? 0) + 1,
            reactionSummaries: [],
            status: "sent",
        };
        addMessages([newMessage]);
        action("handleRegenerateResponse")(message);
    }, [addMessages]);

    const handleReply = useCallback((message: ChatMessageShape) => {
        // Create a new message as a reply
        const newMessage: ChatMessageShape = {
            __typename: "ChatMessage",
            id: generatePK().toString(),
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            translations: [{
                __typename: "ChatMessageTranslation",
                id: generatePK().toString(),
                language: "en",
                text: "This is a reply to your message.",
            }],
            reactionSummaries: [],
            status: "sent",
            user: {
                __typename: "User",
                id: bot1Id,
                name: "AI Assistant",
                handle: "ai_assistant",
                isBot: true,
            },
            versionIndex: 0,
            parent: {
                __typename: "ChatMessageParent",
                id: message.id,
            },
        };
        addMessages([newMessage]);
        action("handleReply")(message);
    }, [addMessages]);

    const handleRetry = useCallback((message: ChatMessageShape) => {
        // Create a new message with the same content but sent status
        const newMessage: ChatMessageShape = {
            ...message,
            id: generatePK().toString(),
            status: "sent",
        };
        addMessages([newMessage]);
        removeMessages([message.id]);
        action("handleRetry")(message);
    }, [addMessages, removeMessages]);

    return {
        tree,
        branches,
        setBranches,
        handleEdit,
        handleReactionAdd,
        handleRegenerateResponse,
        handleReply,
        handleRetry,
        removeMessages,
    };
}

const outerStyle = {
    height: `calc(100vh - ${pagePaddingBottom})`,
    maxWidth: "800px",
    padding: "20px",
    border: "1px solid #ccc",
} as const;

function Outer({ children }: { children: React.ReactNode }) {
    return (
        <Box sx={outerStyle}>
            {children}
        </Box>
    );
}

/**
 * Storybook configuration for ChatBubbleTree
 */
export default {
    title: "Components/Chat/ChatBubbleTree",
    component: ChatBubbleTree,
    decorators: [
        (Story) => (
            <Outer>
                <Story />
            </Outer>
        ),
    ],
};

/**
 * Default story: Basic chat conversation
 */
export function LoggedOut() {
    const {
        tree,
        branches,
        setBranches,
        handleEdit,
        handleReactionAdd,
        handleRegenerateResponse,
        handleReply,
        handleRetry,
        removeMessages,
    } = useStoryMessages(mockMessages);

    return (
        <ChatBubbleTree
            tree={tree}
            branches={branches}
            setBranches={setBranches}
            handleEdit={handleEdit}
            handleRegenerateResponse={handleRegenerateResponse}
            handleReply={handleReply}
            handleRetry={handleRetry}
            removeMessages={removeMessages}
            isBotOnlyChat={true}
            isEditingMessage={false}
            isReplyingToMessage={false}
            messageStream={null}
        />
    );
}
LoggedOut.parameters = {
    session: loggedOutSession,
};

export function SignedInPremiumWithCredits() {
    const {
        tree,
        branches,
        setBranches,
        handleEdit,
        handleReactionAdd,
        handleRegenerateResponse,
        handleReply,
        handleRetry,
        removeMessages,
    } = useStoryMessages(mockMessages);

    return (
        <ChatBubbleTree
            tree={tree}
            branches={branches}
            setBranches={setBranches}
            handleEdit={handleEdit}
            handleRegenerateResponse={handleRegenerateResponse}
            handleReply={handleReply}
            handleRetry={handleRetry}
            removeMessages={removeMessages}
            isBotOnlyChat={true}
            isEditingMessage={false}
            isReplyingToMessage={false}
            messageStream={null}
        />
    );
}
SignedInPremiumWithCredits.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Mock socket service for streaming story
const STREAM_INTERVAL_MS = 150;
function useMockSocketStream() {
    const [messageStream, setMessageStream] = useState<ChatSocketEventPayloads["responseStream"] | null>(null);

    useEffect(() => {
        let currentText = "";
        const fullText = `I've analyzed your code and I can help you implement this feature. Let me break down the solution into multiple parts.

First, let's look at the core data structure we'll need. Here's how we can implement it:

\`\`\`typescript
interface TreeNode<T> {
    value: T;
    children: TreeNode<T>[];
    metadata: {
        id: string;
        parentId: string | null;
        depth: number;
        index: number;
    };
}

class Tree<T> {
    private root: TreeNode<T> | null = null;
    private nodeMap: Map<string, TreeNode<T>> = new Map();

    constructor(data?: T) {
        if (data) {
            this.root = this.createNode(data, null);
        }
    }

    private createNode(value: T, parentId: string | null): TreeNode<T> {
        return {
            value,
            children: [],
            metadata: {
                id: crypto.randomUUID(),
                parentId,
                depth: 0,
                index: 0
            }
        };
    }

    public insert(value: T, parentId: string | null): string {
        const newNode = this.createNode(value, parentId);
        
        if (!parentId) {
            if (!this.root) {
                this.root = newNode;
            } else {
                throw new Error("Root already exists");
            }
        } else {
            const parent = this.nodeMap.get(parentId);
            if (!parent) {
                throw new Error("Parent node not found");
            }
            parent.children.push(newNode);
            newNode.metadata.depth = parent.metadata.depth + 1;
            newNode.metadata.index = parent.children.length - 1;
        }

        this.nodeMap.set(newNode.metadata.id, newNode);
        return newNode.metadata.id;
    }
}
\`\`\`

This implementation provides several key benefits:
1. Efficient node lookup through the Map data structure
2. Proper typing support with TypeScript generics
3. Metadata tracking for each node including depth and position
4. Immutable node references

Let's also consider how we can optimize the tree traversal operations. We should implement both depth-first and breadth-first search algorithms to support different use cases:

\`\`\`typescript
public traverse(strategy: 'dfs' | 'bfs' = 'dfs'): T[] {
    if (!this.root) return [];
    
    if (strategy === 'dfs') {
        return this.depthFirstTraversal(this.root);
    } else {
        return this.breadthFirstTraversal(this.root);
    }
}
\`\`\`

Finally, make sure to add comprehensive error handling and validation to ensure data integrity. The tree structure should maintain its invariants even after multiple operations.`;
        // Split by spaces, then combine any words that are smaller than 3 characters
        const words: string[] = [];
        const splitText = fullText.split(" ");
        for (const word of splitText) {
            if (word.length < 3) {
                words[words.length - 1] += " " + word;
            } else {
                words.push(word);
            }
        }
        let currentWordIndex = 0;

        // Emit a new word every 200ms
        const interval = setInterval(() => {
            if (currentWordIndex < words.length) {
                currentText += (currentWordIndex > 0 ? " " : "") + words[currentWordIndex];
                setMessageStream({
                    __type: "stream",
                    message: currentText,
                    botId: bot1Id,
                });
                currentWordIndex++;
            } else {
                // End the stream
                setMessageStream({
                    __type: "end",
                    message: currentText,
                    botId: bot1Id,
                });
                clearInterval(interval);
            }
        }, STREAM_INTERVAL_MS);

        return () => clearInterval(interval);
    }, []);

    return messageStream;
}

export function SingleMessage() {
    const {
        tree,
        branches,
        setBranches,
        handleEdit,
        handleReactionAdd,
        handleRegenerateResponse,
        handleReply,
        handleRetry,
        removeMessages,
    } = useStoryMessages([mockMessages[0]]);

    return (
        <ChatBubbleTree
            tree={tree}
            branches={branches}
            setBranches={setBranches}
            handleEdit={handleEdit}
            handleRegenerateResponse={handleRegenerateResponse}
            handleReply={handleReply}
            handleRetry={handleRetry}
            removeMessages={removeMessages}
            isBotOnlyChat={true}
            isEditingMessage={false}
            isReplyingToMessage={false}
            messageStream={null}
        />
    );
}
SingleMessage.parameters = {
    session: signedInPremiumWithCreditsSession,
};

/**
 * With Streaming Message: Shows a message being streamed with realistic word-by-word updates
 */
export function StreamingMessage() {
    const {
        tree,
        branches,
        setBranches,
        handleEdit,
        handleReactionAdd,
        handleRegenerateResponse,
        handleReply,
        handleRetry,
        removeMessages,
    } = useStoryMessages(mockMessages);

    // Use our mock socket stream
    const messageStream = useMockSocketStream();

    return (
        <ChatBubbleTree
            tree={tree}
            branches={branches}
            setBranches={setBranches}
            handleEdit={handleEdit}
            handleRegenerateResponse={handleRegenerateResponse}
            handleReply={handleReply}
            handleRetry={handleRetry}
            removeMessages={removeMessages}
            isBotOnlyChat={true}
            isEditingMessage={false}
            isReplyingToMessage={false}
            messageStream={messageStream}
        />
    );
}
StreamingMessage.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Pre-create error message tree
const messagesWithError = [...mockMessages];
messagesWithError[2] = {
    ...messagesWithError[2],
    status: "failed" as ChatMessageStatus,
};

/**
 * With Error Message: Shows a failed message state
 */
export function FailedMessage() {
    const {
        tree,
        branches,
        setBranches,
        handleEdit,
        handleReactionAdd,
        handleRegenerateResponse,
        handleReply,
        handleRetry,
        removeMessages,
    } = useStoryMessages(messagesWithError);

    return (
        <ChatBubbleTree
            tree={tree}
            branches={branches}
            setBranches={setBranches}
            handleEdit={handleEdit}
            handleRegenerateResponse={handleRegenerateResponse}
            handleReply={handleReply}
            handleRetry={handleRetry}
            removeMessages={removeMessages}
            isBotOnlyChat={true}
            isEditingMessage={false}
            isReplyingToMessage={false}
            messageStream={null}
        />
    );
}
FailedMessage.parameters = {
    session: signedInPremiumWithCreditsSession,
};

/**
 * Editing State: Shows the chat tree while editing a message
 */
export function EditingState() {
    const {
        tree,
        branches,
        setBranches,
        handleEdit,
        handleReactionAdd,
        handleRegenerateResponse,
        handleReply,
        handleRetry,
        removeMessages,
    } = useStoryMessages(mockMessages);

    return (
        <ChatBubbleTree
            tree={tree}
            branches={branches}
            setBranches={setBranches}
            handleEdit={handleEdit}
            handleRegenerateResponse={handleRegenerateResponse}
            handleReply={handleReply}
            handleRetry={handleRetry}
            removeMessages={removeMessages}
            isBotOnlyChat={true}
            isEditingMessage={true}
            isReplyingToMessage={false}
            messageStream={null}
        />
    );
}
EditingState.parameters = {
    session: signedInPremiumWithCreditsSession,
};
