import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import FormControl from "@mui/material/FormControl";
import FormLabel from "@mui/material/FormLabel";
import RadioGroup from "@mui/material/RadioGroup";
import FormControlLabel from "@mui/material/FormControlLabel";
import Radio from "@mui/material/Radio";
import Select from "@mui/material/Select";
import MenuItem from "@mui/material/MenuItem";
import { Switch } from "../inputs/Switch/Switch.js";
import { Slider } from "../inputs/Slider.js";
// Simple action replacement
const action = (name: string) => (...args: any[]) => console.log(`Action: ${name}`, args);
import { MINUTES_10_MS, generatePK, type ChatMessageShape, type ChatMessageStatus, type ChatSocketEventPayloads, type ReactionSummary, type ChatMessageRunConfig } from "@vrooli/shared";
import { useCallback, useEffect, useMemo, useState } from "react";
import { loggedOutSession, signedInPremiumWithCreditsSession, signedInUserId } from "../../__test/storybookConsts.js";
import { MessageTree } from "../../hooks/messages.js";
import { pagePaddingBottom } from "../../styles.js";
import { type BranchMap } from "../../utils/localStorage.js";
import { ChatBubbleTree, ChatBubble } from "./ChatBubbleTree.js";
import { borderedContainerDecorator } from "../../__test/helpers/storybookDecorators.tsx";

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


/**
 * Storybook configuration for ChatBubbleTree
 */
export default {
    title: "Components/Chat/ChatBubbleTree",
    component: ChatBubbleTree,
    decorators: [borderedContainerDecorator(800)],
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
                    chunk: words[currentWordIndex],
                    botId: bot1Id,
                });
                currentWordIndex++;
            } else {
                // End the stream
                setMessageStream({
                    __type: "end",
                    finalMessage: currentText,
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

// Create Story type for the showcase story
type Story = {
    render: () => JSX.Element;
};

/**
 * ChatBubble Showcase: Interactive playground for testing individual ChatBubble components
 */
export const ChatBubbleShowcase: Story = {
    render: () => {
        // Message configuration state
        const [messageFrom, setMessageFrom] = useState<"you" | "user" | "bot">("bot");
        const [messageStatus, setMessageStatus] = useState<ChatMessageStatus>("sent");
        const [reactionCount, setReactionCount] = useState<number>(0);
        const [reportCount, setReportCount] = useState<number>(0);
        const [contentType, setContentType] = useState<"none" | "short" | "long" | "code">("short");
        const [hasVersions, setHasVersions] = useState(false);
        const [activeVersion, setActiveVersion] = useState(0);
        
        // Generate user/bot data based on messageFrom
        const userId = messageFrom === "you" ? signedInUserId : generatePK().toString();
        const userName = messageFrom === "you" ? "You" : messageFrom === "bot" ? "AI Assistant" : "Other User";
        const userHandle = messageFrom === "you" ? "you" : messageFrom === "bot" ? "ai_assistant" : "otheruser";
        const isBot = messageFrom === "bot";
        const isOwn = messageFrom === "you";
        
        // Generate message content based on contentType
        const getMessageContent = () => {
            switch (contentType) {
                case "none":
                    return "";
                case "short":
                    return "This is a short message to test the chat bubble appearance.";
                case "long":
                    return `This is a much longer message that demonstrates how the chat bubble handles extensive content. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.

This message has multiple paragraphs to show how the bubble expands and handles text wrapping. It should display nicely with proper spacing and formatting.`;
                case "code":
                    return `Here's an example of how to implement a binary search algorithm:

\`\`\`typescript
function binarySearch<T>(arr: T[], target: T, compareFn?: (a: T, b: T) => number): number {
    let left = 0;
    let right = arr.length - 1;
    
    while (left <= right) {
        const mid = Math.floor((left + right) / 2);
        const comparison = compareFn ? compareFn(arr[mid], target) : 0;
        
        if (comparison === 0) {
            return mid;
        } else if (comparison < 0) {
            left = mid + 1;
        } else {
            right = mid - 1;
        }
    }
    
    return -1; // Element not found
}

// Example usage:
const numbers = [1, 3, 5, 7, 9, 11, 13, 15];
const index = binarySearch(numbers, 7);
console.log(\`Found at index: \${index}\`); // Output: Found at index: 3
\`\`\`

This implementation uses a generic type parameter and an optional comparison function for flexibility.`;
                default:
                    return "Default message content";
            }
        };
        
        // Generate reactions based on reactionCount
        const generateReactions = (): ReactionSummary[] => {
            if (reactionCount === 0) return [];
            
            const emojis = ["👍", "❤️", "😂", "🎉", "🤔", "👏", "🔥", "💯", "😍", "🚀", "💪", "🙏", "😊", "🤩", "👀", "💡", "⭐", "✨", "🎯", "💎", "🌟", "🔮", "🎨", "🎵", "🌈", "⚡", "🎪", "🌸", "🎭", "🎸"];
            const reactions: ReactionSummary[] = [];
            
            // Distribute reaction count across different emojis
            let remaining = reactionCount;
            // Use more emojis as reaction count increases to better showcase the toggle feature
            const numEmojis = Math.min(
                reactionCount > 80 ? 12 : 
                reactionCount > 60 ? 10 : 
                reactionCount > 40 ? 8 : 
                reactionCount > 20 ? 6 : 
                reactionCount > 10 ? 4 : 
                reactionCount > 5 ? 3 : 2, 
                emojis.length
            );
            
            for (let i = 0; i < numEmojis && remaining > 0; i++) {
                const count = i === numEmojis - 1 
                    ? remaining 
                    : Math.ceil(remaining / (numEmojis - i) * (0.5 + Math.random() * 0.5));
                
                reactions.push({
                    __typename: "ReactionSummary" as const,
                    emoji: emojis[i],
                    count: Math.min(count, remaining),
                });
                
                remaining -= count;
            }
            
            return reactions;
        };
        
        // Create the message object
        const message: ChatMessageShape = {
            __typename: "ChatMessage" as const,
            id: generatePK().toString(),
            createdAt: new Date(Date.now() - 5 * MINUTES_10_MS).toISOString(),
            updatedAt: new Date(Date.now() - 5 * MINUTES_10_MS).toISOString(),
            translations: [{
                __typename: "ChatMessageTranslation" as const,
                id: generatePK().toString(),
                language: "en",
                text: getMessageContent(),
            }],
            reactionSummaries: generateReactions(),
            status: messageStatus,
            user: {
                __typename: "User" as const,
                id: userId,
                name: userName,
                handle: userHandle,
                isBot: isBot,
            },
            versionIndex: activeVersion,
            parent: null,
            // Add report info if reportCount > 0
            ...(reportCount > 0 && {
                reportsCount: reportCount,
                you: {
                    canDelete: true,
                    canReact: true,
                    canReply: true,
                    canUpdate: isOwn,
                    isBookmarked: false,
                    reaction: null,
                },
            }),
        };
        
        const handleActiveIndexChange = (index: number) => {
            setActiveVersion(index);
            action("handleActiveIndexChange")(index);
        };
        
        return (
            <Box sx={{ 
                p: 2, 
                height: "100vh", 
                overflow: "auto",
                bgcolor: "background.default" 
            }}>
                <Box sx={{ 
                    display: "flex", 
                    gap: 2, 
                    flexDirection: { xs: "column", lg: "row" },
                    maxWidth: 1400, 
                    mx: "auto" 
                }}>
                    {/* Controls Section */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        height: "fit-content",
                        minWidth: { lg: 320 }
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>ChatBubble Controls</Typography>
                        
                        <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
                            {/* Message From Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Message From</FormLabel>
                                <RadioGroup
                                    value={messageFrom}
                                    onChange={(e) => setMessageFrom(e.target.value as "you" | "user" | "bot")}
                                    sx={{ gap: 0.5 }}
                                >
                                    <FormControlLabel value="you" control={<Radio size="small" />} label="You (current user)" sx={{ m: 0 }} />
                                    <FormControlLabel value="user" control={<Radio size="small" />} label="Another User" sx={{ m: 0 }} />
                                    <FormControlLabel value="bot" control={<Radio size="small" />} label="Bot" sx={{ m: 0 }} />
                                </RadioGroup>
                            </FormControl>
                            
                            {/* Message Status Control */}
                            <FormControl size="small" fullWidth>
                                <FormLabel sx={{ fontSize: "0.875rem", mb: 1 }}>Message Status</FormLabel>
                                <Select
                                    value={messageStatus}
                                    onChange={(e) => setMessageStatus(e.target.value as ChatMessageStatus)}
                                >
                                    <MenuItem value="unsent">Unsent</MenuItem>
                                    <MenuItem value="editing">Editing</MenuItem>
                                    <MenuItem value="sending">Sending</MenuItem>
                                    <MenuItem value="sent">Sent</MenuItem>
                                    <MenuItem value="failed">Failed</MenuItem>
                                </Select>
                            </FormControl>
                            
                            {/* Reaction Count Control */}
                            <FormControl component="fieldset" size="small">
                                <Slider
                                    value={reactionCount}
                                    onChange={(value) => setReactionCount(value)}
                                    min={0}
                                    max={100}
                                    label={`Reactions: ${reactionCount}`}
                                    showValue={false}
                                    marks={[
                                        { value: 0, label: "None" },
                                        { value: 5, label: "Few" },
                                        { value: 15, label: "Many" },
                                        { value: 50, label: "Lots" },
                                        { value: 100, label: "Max" },
                                    ]}
                                />
                            </FormControl>
                            
                            {/* Report Count Control */}
                            <FormControl component="fieldset" size="small">
                                <Slider
                                    value={reportCount}
                                    onChange={(value) => setReportCount(value)}
                                    min={0}
                                    max={10}
                                    label={`Reports: ${reportCount}`}
                                    showValue={false}
                                    marks={Array.from({ length: 11 }, (_, i) => ({ value: i }))}
                                />
                            </FormControl>
                            
                            {/* Content Type Control */}
                            <FormControl component="fieldset" size="small">
                                <FormLabel component="legend" sx={{ fontSize: "0.875rem", mb: 1 }}>Content Type</FormLabel>
                                <RadioGroup
                                    value={contentType}
                                    onChange={(e) => setContentType(e.target.value as "none" | "short" | "long" | "code")}
                                    sx={{ gap: 0.5 }}
                                >
                                    <FormControlLabel value="none" control={<Radio size="small" />} label="No Content" sx={{ m: 0 }} />
                                    <FormControlLabel value="short" control={<Radio size="small" />} label="Short Text" sx={{ m: 0 }} />
                                    <FormControlLabel value="long" control={<Radio size="small" />} label="Long Text" sx={{ m: 0 }} />
                                    <FormControlLabel value="code" control={<Radio size="small" />} label="Code Block" sx={{ m: 0 }} />
                                </RadioGroup>
                            </FormControl>
                            
                            {/* Has Versions Control */}
                            <FormControl component="fieldset" size="small">
                                <Switch
                                    checked={hasVersions}
                                    onChange={(checked) => setHasVersions(checked)}
                                    size="sm"
                                    label="Has Multiple Versions"
                                    labelPosition="right"
                                />
                            </FormControl>
                        </Box>
                    </Box>
                    
                    {/* Preview Section */}
                    <Box sx={{ 
                        p: 3, 
                        bgcolor: "background.paper", 
                        borderRadius: 2, 
                        boxShadow: 1,
                        flex: 1,
                        overflow: "hidden"
                    }}>
                        <Typography variant="h5" sx={{ mb: 3 }}>ChatBubble Preview</Typography>
                        
                        <Box sx={{ 
                            bgcolor: "background.default", 
                            borderRadius: 2, 
                            p: 2,
                            minHeight: 400,
                            display: "flex",
                            alignItems: "flex-start",
                            overflow: "auto"
                        }}>
                            <Box sx={{ width: "100%", maxWidth: 800 }}>
                                <ChatBubble
                                    activeIndex={activeVersion}
                                    chatWidth={800}
                                    message={message}
                                    numSiblings={hasVersions ? 3 : 1}
                                    isBotOnlyChat={false}
                                    isOwn={isOwn}
                                    onActiveIndexChange={handleActiveIndexChange}
                                    onDeleted={action("onDeleted")}
                                    onEdit={action("onEdit")}
                                    onReply={action("onReply")}
                                    onRegenerateResponse={action("onRegenerateResponse")}
                                    onRetry={action("onRetry")}
                                />
                            </Box>
                        </Box>
                        
                        {/* State Information */}
                        <Box sx={{ mt: 3 }}>
                            <Typography variant="subtitle2" color="text.secondary">
                                Current State Information:
                            </Typography>
                            <Typography variant="body2" color="text.secondary" component="pre" sx={{ mt: 1 }}>
                                {JSON.stringify({
                                    isOwn,
                                    isBot,
                                    status: messageStatus,
                                    reactionCount,
                                    reportCount,
                                    hasVersions,
                                    activeVersion: hasVersions ? activeVersion : 0,
                                }, null, 2)}
                            </Typography>
                        </Box>
                    </Box>
                </Box>
            </Box>
        );
    },
};
ChatBubbleShowcase.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Generate long content for performance testing
function generateLongText(paragraphs: number = 3): string {
    const sentences = [
        "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
        "Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.",
        "Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.",
        "Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.",
        "Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium.",
        "Totam rem aperiam, eaque ipsa quae ab illo inventore veritatis et quasi architecto beatae vitae dicta sunt explicabo.",
        "Nemo enim ipsam voluptatem quia voluptas sit aspernatur aut odit aut fugit, sed quia consequuntur magni dolores.",
        "Neque porro quisquam est, qui dolorem ipsum quia dolor sit amet, consectetur, adipisci velit.",
        "Ut enim ad minima veniam, quis nostrum exercitationem ullam corporis suscipit laboriosam.",
        "Quis autem vel eum iure reprehenderit qui in ea voluptate velit esse quam nihil molestiae consequatur."
    ];
    
    const paragraphTexts: string[] = [];
    for (let p = 0; p < paragraphs; p++) {
        const sentenceCount = 4 + Math.floor(Math.random() * 4); // 4-7 sentences per paragraph
        const paragraphSentences: string[] = [];
        for (let s = 0; s < sentenceCount; s++) {
            paragraphSentences.push(sentences[Math.floor(Math.random() * sentences.length)]);
        }
        paragraphTexts.push(paragraphSentences.join(" "));
    }
    
    return paragraphTexts.join("\n\n");
}

// Generate code block content for performance testing
function generateCodeContent(): string {
    const codeExamples = [
        // JavaScript/TypeScript function
        `Here's a function to handle user authentication:

\`\`\`typescript
async function authenticateUser(email: string, password: string): Promise<AuthResult> {
    try {
        // Validate input
        if (!email || !password) {
            throw new Error('Email and password are required');
        }
        
        // Hash the password
        const hashedPassword = await bcrypt.hash(password, 10);
        
        // Query the database
        const user = await db.users.findOne({ 
            email: email.toLowerCase(),
            password: hashedPassword 
        });
        
        if (!user) {
            return { success: false, error: 'Invalid credentials' };
        }
        
        // Generate JWT token
        const token = jwt.sign(
            { userId: user.id, email: user.email },
            process.env.JWT_SECRET,
            { expiresIn: '24h' }
        );
        
        return { success: true, token, user };
    } catch (error) {
        console.error('Authentication error:', error);
        return { success: false, error: 'Authentication failed' };
    }
}
\`\`\``,
        
        // React component with hooks
        `I've created a custom hook for managing form state:

\`\`\`tsx
import { useState, useCallback, useEffect } from 'react';

interface FormState<T> {
    values: T;
    errors: Partial<Record<keyof T, string>>;
    touched: Partial<Record<keyof T, boolean>>;
    isSubmitting: boolean;
    isValid: boolean;
}

export function useForm<T extends Record<string, any>>(
    initialValues: T,
    validate?: (values: T) => Partial<Record<keyof T, string>>
) {
    const [state, setState] = useState<FormState<T>>({
        values: initialValues,
        errors: {},
        touched: {},
        isSubmitting: false,
        isValid: true,
    });

    const handleChange = useCallback((name: keyof T, value: any) => {
        setState(prev => ({
            ...prev,
            values: { ...prev.values, [name]: value },
            touched: { ...prev.touched, [name]: true },
        }));
    }, []);

    const handleSubmit = useCallback(async (onSubmit: (values: T) => Promise<void>) => {
        setState(prev => ({ ...prev, isSubmitting: true }));
        
        try {
            if (validate) {
                const errors = validate(state.values);
                if (Object.keys(errors).length > 0) {
                    setState(prev => ({ 
                        ...prev, 
                        errors, 
                        isSubmitting: false 
                    }));
                    return;
                }
            }
            
            await onSubmit(state.values);
            setState(prev => ({ ...prev, isSubmitting: false }));
        } catch (error) {
            console.error('Form submission error:', error);
            setState(prev => ({ ...prev, isSubmitting: false }));
        }
    }, [state.values, validate]);

    return {
        ...state,
        handleChange,
        handleSubmit,
        reset: () => setState({
            values: initialValues,
            errors: {},
            touched: {},
            isSubmitting: false,
            isValid: true,
        }),
    };
}
\`\`\``,
        
        // Python data processing
        `Here's a Python script for data analysis:

\`\`\`python
import pandas as pd
import numpy as np
from typing import List, Dict, Tuple
import matplotlib.pyplot as plt

class DataAnalyzer:
    def __init__(self, data_path: str):
        self.df = pd.read_csv(data_path)
        self.results = {}
    
    def preprocess_data(self) -> pd.DataFrame:
        """Clean and prepare data for analysis"""
        # Remove duplicates
        self.df = self.df.drop_duplicates()
        
        # Handle missing values
        numeric_columns = self.df.select_dtypes(include=[np.number]).columns
        self.df[numeric_columns] = self.df[numeric_columns].fillna(self.df[numeric_columns].mean())
        
        # Convert date columns
        date_columns = [col for col in self.df.columns if 'date' in col.lower()]
        for col in date_columns:
            self.df[col] = pd.to_datetime(self.df[col])
        
        return self.df
    
    def calculate_statistics(self) -> Dict[str, float]:
        """Calculate basic statistics"""
        stats = {
            'mean': self.df.select_dtypes(include=[np.number]).mean().to_dict(),
            'median': self.df.select_dtypes(include=[np.number]).median().to_dict(),
            'std': self.df.select_dtypes(include=[np.number]).std().to_dict(),
            'correlation': self.df.corr().to_dict()
        }
        
        self.results['statistics'] = stats
        return stats
    
    def generate_report(self, output_path: str = 'report.html'):
        """Generate HTML report with visualizations"""
        html_content = f"""
        <html>
        <head><title>Data Analysis Report</title></head>
        <body>
            <h1>Data Analysis Results</h1>
            <h2>Dataset Overview</h2>
            <p>Shape: {self.df.shape}</p>
            <p>Columns: {', '.join(self.df.columns)}</p>
            
            <h2>Statistics</h2>
            <pre>{self.results.get('statistics', 'No statistics calculated')}</pre>
        </body>
        </html>
        """
        
        with open(output_path, 'w') as f:
            f.write(html_content)
        
        return output_path

# Usage example
analyzer = DataAnalyzer('sales_data.csv')
analyzer.preprocess_data()
stats = analyzer.calculate_statistics()
report_path = analyzer.generate_report()
print(f"Report generated at: {report_path}")
\`\`\``,
        
        // SQL query
        `Here's an optimized SQL query for the report:

\`\`\`sql
WITH monthly_sales AS (
    SELECT 
        DATE_TRUNC('month', order_date) as month,
        product_id,
        SUM(quantity * unit_price) as revenue,
        COUNT(DISTINCT customer_id) as unique_customers,
        AVG(quantity * unit_price) as avg_order_value
    FROM orders o
    JOIN order_items oi ON o.order_id = oi.order_id
    WHERE order_date >= CURRENT_DATE - INTERVAL '12 months'
    GROUP BY DATE_TRUNC('month', order_date), product_id
),
product_rankings AS (
    SELECT 
        month,
        product_id,
        revenue,
        unique_customers,
        avg_order_value,
        ROW_NUMBER() OVER (PARTITION BY month ORDER BY revenue DESC) as revenue_rank,
        LAG(revenue) OVER (PARTITION BY product_id ORDER BY month) as prev_month_revenue
    FROM monthly_sales
)
SELECT 
    pr.month,
    p.product_name,
    pr.revenue,
    pr.unique_customers,
    pr.avg_order_value,
    pr.revenue_rank,
    CASE 
        WHEN pr.prev_month_revenue IS NULL THEN 'N/A'
        ELSE ROUND(((pr.revenue - pr.prev_month_revenue) / pr.prev_month_revenue * 100), 2)::text || '%'
    END as growth_rate
FROM product_rankings pr
JOIN products p ON pr.product_id = p.product_id
WHERE pr.revenue_rank <= 10
ORDER BY pr.month DESC, pr.revenue_rank;
\`\`\``,
        
        // Algorithm implementation
        `Here's an efficient algorithm for finding the shortest path:

\`\`\`javascript
class Graph {
    constructor() {
        this.adjacencyList = new Map();
    }
    
    addVertex(vertex) {
        if (!this.adjacencyList.has(vertex)) {
            this.adjacencyList.set(vertex, []);
        }
    }
    
    addEdge(source, destination, weight) {
        this.addVertex(source);
        this.addVertex(destination);
        
        this.adjacencyList.get(source).push({ node: destination, weight });
        this.adjacencyList.get(destination).push({ node: source, weight });
    }
    
    dijkstra(start, end) {
        const distances = {};
        const previous = {};
        const priorityQueue = new PriorityQueue();
        
        // Initialize distances
        for (let vertex of this.adjacencyList.keys()) {
            distances[vertex] = vertex === start ? 0 : Infinity;
            priorityQueue.enqueue(vertex, distances[vertex]);
            previous[vertex] = null;
        }
        
        while (!priorityQueue.isEmpty()) {
            const current = priorityQueue.dequeue().element;
            
            if (current === end) {
                // Reconstruct path
                const path = [];
                let temp = end;
                while (temp !== null) {
                    path.unshift(temp);
                    temp = previous[temp];
                }
                return { distance: distances[end], path };
            }
            
            if (distances[current] === Infinity) break;
            
            // Check neighbors
            for (let neighbor of this.adjacencyList.get(current)) {
                const altDistance = distances[current] + neighbor.weight;
                
                if (altDistance < distances[neighbor.node]) {
                    distances[neighbor.node] = altDistance;
                    previous[neighbor.node] = current;
                    priorityQueue.updatePriority(neighbor.node, altDistance);
                }
            }
        }
        
        return { distance: Infinity, path: [] };
    }
}

// Priority queue implementation for Dijkstra's algorithm
class PriorityQueue {
    constructor() {
        this.elements = [];
    }
    
    enqueue(element, priority) {
        this.elements.push({ element, priority });
        this.bubbleUp(this.elements.length - 1);
    }
    
    dequeue() {
        const min = this.elements[0];
        const end = this.elements.pop();
        
        if (this.elements.length > 0) {
            this.elements[0] = end;
            this.sinkDown(0);
        }
        
        return min;
    }
    
    isEmpty() {
        return this.elements.length === 0;
    }
    
    bubbleUp(index) {
        const element = this.elements[index];
        
        while (index > 0) {
            const parentIndex = Math.floor((index - 1) / 2);
            const parent = this.elements[parentIndex];
            
            if (element.priority >= parent.priority) break;
            
            this.elements[parentIndex] = element;
            this.elements[index] = parent;
            index = parentIndex;
        }
    }
    
    sinkDown(index) {
        const length = this.elements.length;
        const element = this.elements[index];
        
        while (true) {
            const leftChildIndex = 2 * index + 1;
            const rightChildIndex = 2 * index + 2;
            let swap = null;
            
            if (leftChildIndex < length) {
                const leftChild = this.elements[leftChildIndex];
                if (leftChild.priority < element.priority) {
                    swap = leftChildIndex;
                }
            }
            
            if (rightChildIndex < length) {
                const rightChild = this.elements[rightChildIndex];
                if (rightChild.priority < element.priority && 
                    rightChild.priority < this.elements[leftChildIndex].priority) {
                    swap = rightChildIndex;
                }
            }
            
            if (swap === null) break;
            
            this.elements[index] = this.elements[swap];
            this.elements[swap] = element;
            index = swap;
        }
    }
}
\`\`\``
    ];
    
    return codeExamples[Math.floor(Math.random() * codeExamples.length)];
}

// Generate realistic reactions for performance testing
function generateManyReactions(count: number): ReactionSummary[] {
    const emojis = ["👍", "❤️", "😂", "🎉", "🤔", "👏", "🔥", "💯", "😍", "🚀", "💪", "🙏", "😊", "🤩", "👀", "💡", "⭐", "✨"];
    const reactions: ReactionSummary[] = [];
    
    for (let i = 0; i < Math.min(count, emojis.length); i++) {
        reactions.push({
            __typename: "ReactionSummary",
            emoji: emojis[i],
            count: 1 + Math.floor(Math.random() * 10),
        });
    }
    
    return reactions;
}

// Create hundreds of messages for performance testing
function createPerformanceTestMessages(messageCount: number = 1000): ChatMessageShape[] {
    const messages: ChatMessageShape[] = [];
    const userIds = [bot1Id, signedInUserId];
    const userNames = ["AI Assistant", "Test User"];
    const userHandles = ["ai_assistant", "testuser"];
    const isBot = [true, false];
    
    let currentTime = Date.now() - (messageCount * 30000); // Start 30 seconds apart
    
    for (let i = 0; i < messageCount; i++) {
        const userIndex = i % 2; // Alternate between bot and user
        const messageId = generatePK().toString();
        const parentId = i > 0 ? messages[i - 1].id : null;
        
        // Mix code blocks with regular text - 30% chance for code content
        const includeCode = Math.random() < 0.3;
        const messageText = includeCode 
            ? generateCodeContent() 
            : generateLongText(2 + Math.floor(Math.random() * 4)); // 2-5 paragraphs
        
        messages.push({
            __typename: "ChatMessage",
            id: messageId,
            createdAt: new Date(currentTime + (i * 30000)).toISOString(),
            updatedAt: new Date(currentTime + (i * 30000)).toISOString(),
            translations: [{
                __typename: "ChatMessageTranslation",
                id: generatePK().toString(),
                language: "en",
                text: messageText,
            }],
            reactionSummaries: Math.random() > 0.3 ? generateManyReactions(Math.floor(Math.random() * 8)) : [], // 70% chance of reactions
            status: "sent" as ChatMessageStatus,
            user: {
                __typename: "User",
                id: userIds[userIndex],
                name: userNames[userIndex],
                handle: userHandles[userIndex],
                isBot: isBot[userIndex],
            },
            versionIndex: 0,
            parent: parentId ? {
                __typename: "ChatMessageParent",
                id: parentId,
            } : null,
        });
    }
    
    return messages;
}

/**
 * Performance Test: Hundreds of messages with long content to test rendering performance
 * Use this story to benchmark performance optimizations
 */
export function PerformanceTest() {
    const performanceMessages = useMemo(() => createPerformanceTestMessages(1000), []); // 1000 messages for stress testing
    
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
    } = useStoryMessages(performanceMessages);

    // Add performance monitoring
    const renderStart = performance.now();
    
    useEffect(() => {
        const renderEnd = performance.now();
        console.log(`📊 ChatBubbleTree Performance Test - Render time: ${(renderEnd - renderStart).toFixed(2)}ms`);
        console.log(`📊 Message count: ${performanceMessages.length}`);
        console.log(`📊 Tree size: ${tree.getMap().size} nodes`);
    });

    return (
        <>
            <Box sx={{ 
                position: "absolute", 
                top: 10, 
                right: 10, 
                background: "rgba(0,0,0,0.7)", 
                color: "white", 
                padding: 1, 
                borderRadius: 1,
                fontSize: "0.8rem",
                zIndex: 1000
            }}>
                📊 Performance Test: {performanceMessages.length} messages
            </Box>
            <ChatBubbleTree
                tree={tree}
                branches={branches}
                setBranches={setBranches}
                handleEdit={handleEdit}
                handleRegenerateResponse={handleRegenerateResponse}
                handleReply={handleReply}
                handleRetry={handleRetry}
                removeMessages={removeMessages}
                isBotOnlyChat={false}
                isEditingMessage={false}
                isReplyingToMessage={false}
                messageStream={null}
            />
        </>
    );
}
PerformanceTest.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Mock run configurations for testing - includes ALL possible RunStatus states
function createMockRuns(): ChatMessageRunConfig[] {
    return [
        // Completed run (no button)
        {
            runId: generatePK().toString(),
            resourceVersionId: generatePK().toString(),
            resourceVersionName: "Data Processing Routine",
            taskId: generatePK().toString(),
            runStatus: "Completed",
            createdAt: new Date(Date.now() - 5 * MINUTES_10_MS).toISOString(),
            completedAt: new Date(Date.now() - 2 * MINUTES_10_MS).toISOString(),
        },
        // InProgress run (pause button)
        {
            runId: generatePK().toString(),
            resourceVersionId: generatePK().toString(), 
            resourceVersionName: "File Converter",
            taskId: generatePK().toString(),
            runStatus: "InProgress",
            createdAt: new Date(Date.now() - 3 * MINUTES_10_MS).toISOString(),
        },
        // Failed run (retry button)
        {
            runId: generatePK().toString(),
            resourceVersionId: generatePK().toString(),
            resourceVersionName: "API Integration",
            taskId: generatePK().toString(), 
            runStatus: "Failed",
            createdAt: new Date(Date.now() - 4 * MINUTES_10_MS).toISOString(),
            completedAt: new Date(Date.now() - 1 * MINUTES_10_MS).toISOString(),
        },
        // Scheduled run (play button)
        {
            runId: generatePK().toString(),
            resourceVersionId: generatePK().toString(),
            resourceVersionName: "Email Automation",
            taskId: generatePK().toString(),
            runStatus: "Scheduled",
            createdAt: new Date(Date.now() - 2 * MINUTES_10_MS).toISOString(),
        },
        // Paused run (play button)
        {
            runId: generatePK().toString(),
            resourceVersionId: generatePK().toString(),
            resourceVersionName: "Batch Image Processing",
            taskId: generatePK().toString(),
            runStatus: "Paused",
            createdAt: new Date(Date.now() - 6 * MINUTES_10_MS).toISOString(),
        },
        // Cancelled run (no button)
        {
            runId: generatePK().toString(),
            resourceVersionId: generatePK().toString(),
            resourceVersionName: "Large File Upload",
            taskId: generatePK().toString(),
            runStatus: "Cancelled",
            createdAt: new Date(Date.now() - 7 * MINUTES_10_MS).toISOString(),
            completedAt: new Date(Date.now() - 5 * MINUTES_10_MS).toISOString(),
        },
    ];
}

// Create messages with runs
const messagesWithRuns: ChatMessageShape[] = [
    // Initial bot message 
    {
        __typename: "ChatMessage" as const,
        id: generatePK().toString(),
        createdAt: new Date(Date.now() - 10 * MINUTES_10_MS).toISOString(),
        updatedAt: new Date(Date.now() - 10 * MINUTES_10_MS).toISOString(),
        translations: [{
            __typename: "ChatMessageTranslation" as const,
            id: generatePK().toString(),
            language: "en",
            text: "I'll help you process those files and integrate with the API. Let me start the necessary routines.",
        }],
        reactionSummaries: [],
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
        config: {
            runs: createMockRuns(),
        },
    },
    // User follow-up message
    {
        __typename: "ChatMessage" as const,
        id: generatePK().toString(),
        createdAt: new Date(Date.now() - 6 * MINUTES_10_MS).toISOString(),
        updatedAt: new Date(Date.now() - 6 * MINUTES_10_MS).toISOString(),
        translations: [{
            __typename: "ChatMessageTranslation" as const,
            id: generatePK().toString(),
            language: "en",
            text: "Great! I can see the progress. The data processing completed successfully, but it looks like the API integration failed. Can you retry that one?",
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
            id: generatePK().toString(),
        },
    },
    // Bot response with another run
    {
        __typename: "ChatMessage" as const,
        id: generatePK().toString(),
        createdAt: new Date(Date.now() - 2 * MINUTES_10_MS).toISOString(),
        updatedAt: new Date(Date.now() - 2 * MINUTES_10_MS).toISOString(),
        translations: [{
            __typename: "ChatMessageTranslation" as const,
            id: generatePK().toString(),
            language: "en",
            text: "I see the API integration failed due to authentication issues. Let me retry with updated credentials.",
        }],
        reactionSummaries: [{
            __typename: "ReactionSummary" as const,
            count: 1,
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
        parent: {
            __typename: "ChatMessageParent" as const,
            id: generatePK().toString(),
        },
        config: {
            runs: [
                // Alternative way to show InProgress (both InProgress and "Running" should show pause button)
                {
                    runId: generatePK().toString(),
                    resourceVersionId: generatePK().toString(),
                    resourceVersionName: "API Integration (Retry)",
                    taskId: generatePK().toString(),
                    runStatus: "InProgress",
                    createdAt: new Date(Date.now() - 1 * MINUTES_10_MS).toISOString(),
                },
                // Test with "Running" status for compatibility
                {
                    runId: generatePK().toString(),
                    resourceVersionId: generatePK().toString(),
                    resourceVersionName: "Real-time Data Sync",
                    taskId: generatePK().toString(),
                    runStatus: "Running",
                    createdAt: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
                },
            ],
        },
    },
];

/**
 * With Routine Executors: Shows messages with routine executions displayed underneath
 */
export function WithRoutineExecutors() {
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
    } = useStoryMessages(messagesWithRuns);

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
WithRoutineExecutors.parameters = {
    session: signedInPremiumWithCreditsSession,
};
