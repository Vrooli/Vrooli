import { ChatMessageShape, ChatMessageStatus, ChatSocketEventPayloads } from "@local/shared";
import { Box } from "@mui/material";
import { action } from "@storybook/addon-actions";
import { SessionContext } from "../../contexts.js";
import { MessageTree } from "../../hooks/messages.js";
import { BranchMap } from "../../utils/localStorage.js";
import { ChatBubbleTree } from "./ChatBubbleTree.js";

// Mock session context
const mockSession = {
    user: {
        id: "user-1",
        name: "Test User",
        handle: "testuser",
    },
};

// Mock message data
const mockMessages: ChatMessageShape[] = [
    {
        id: "msg-1",
        translations: [{ language: "en", text: "Hello! How can I help you today?" }],
        reactionSummaries: [{ emoji: "👍", count: 2 }],
        status: "sent" as ChatMessageStatus,
        user: {
            id: "bot-1",
            name: "AI Assistant",
            handle: "ai_assistant",
            isBot: true,
        },
        versionIndex: 0,
    },
    {
        id: "msg-2",
        translations: [{ language: "en", text: "I need help with my code." }],
        reactionSummaries: [],
        status: "sent" as ChatMessageStatus,
        user: {
            id: "user-1",
            name: "Test User",
            handle: "testuser",
            isBot: false,
        },
        versionIndex: 0,
    },
    {
        id: "msg-3",
        translations: [{ language: "en", text: "I'd be happy to help! What specific issue are you facing?" }],
        reactionSummaries: [{ emoji: "🤔", count: 1 }],
        status: "sent" as ChatMessageStatus,
        user: {
            id: "bot-1",
            name: "AI Assistant",
            handle: "ai_assistant",
            isBot: true,
        },
        versionIndex: 0,
    },
];

// Create message tree
const createMessageTree = (messages: ChatMessageShape[]): MessageTree<ChatMessageShape> => {
    const map = new Map();
    messages.forEach(msg => map.set(msg.id, {
        message: msg,
        children: [],
    }));
    return {
        map,
        roots: [messages[0].id],
    };
};

const outerStyle = {
    height: "500px",
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
                <SessionContext.Provider value={mockSession}>
                    <Story />
                </SessionContext.Provider>
            </Outer>
        ),
    ],
};

/**
 * Default story: Basic chat conversation
 */
export function Default() {
    const tree = createMessageTree(mockMessages);
    const branches: BranchMap = {};

    return (
        <ChatBubbleTree
            tree={tree}
            branches={branches}
            setBranches={action("setBranches")}
            handleEdit={action("handleEdit")}
            handleRegenerateResponse={action("handleRegenerateResponse")}
            handleReply={action("handleReply")}
            handleRetry={action("handleRetry")}
            removeMessages={action("removeMessages")}
            isBotOnlyChat={true}
            isEditingMessage={false}
            isReplyingToMessage={false}
            messageStream={null}
        />
    );
}

/**
 * With Streaming Message: Shows a message being streamed
 */
export function WithStreamingMessage() {
    const tree = createMessageTree(mockMessages);
    const branches: BranchMap = {};
    const streamingMessage: ChatSocketEventPayloads["responseStream"] = {
        __type: "stream",
        message: "I'm currently generating a response...",
        botId: "bot-1",
    };

    return (
        <ChatBubbleTree
            tree={tree}
            branches={branches}
            setBranches={action("setBranches")}
            handleEdit={action("handleEdit")}
            handleRegenerateResponse={action("handleRegenerateResponse")}
            handleReply={action("handleReply")}
            handleRetry={action("handleRetry")}
            removeMessages={action("removeMessages")}
            isBotOnlyChat={true}
            isEditingMessage={false}
            isReplyingToMessage={false}
            messageStream={streamingMessage}
        />
    );
}

/**
 * With Error Message: Shows a failed message state
 */
export function WithErrorMessage() {
    const messagesWithError = [...mockMessages];
    messagesWithError[2] = {
        ...messagesWithError[2],
        status: "failed" as ChatMessageStatus,
    };

    const tree = createMessageTree(messagesWithError);
    const branches: BranchMap = {};

    return (
        <ChatBubbleTree
            tree={tree}
            branches={branches}
            setBranches={action("setBranches")}
            handleEdit={action("handleEdit")}
            handleRegenerateResponse={action("handleRegenerateResponse")}
            handleReply={action("handleReply")}
            handleRetry={action("handleRetry")}
            removeMessages={action("removeMessages")}
            isBotOnlyChat={true}
            isEditingMessage={false}
            isReplyingToMessage={false}
            messageStream={null}
        />
    );
}

/**
 * Editing State: Shows the chat tree while editing a message
 */
export function EditingState() {
    const tree = createMessageTree(mockMessages);
    const branches: BranchMap = {};

    return (
        <ChatBubbleTree
            tree={tree}
            branches={branches}
            setBranches={action("setBranches")}
            handleEdit={action("handleEdit")}
            handleRegenerateResponse={action("handleRegenerateResponse")}
            handleReply={action("handleReply")}
            handleRetry={action("handleRetry")}
            removeMessages={action("removeMessages")}
            isBotOnlyChat={true}
            isEditingMessage={true}
            isReplyingToMessage={false}
            messageStream={null}
        />
    );
}

/**
 * With Custom Below Message List: Shows the chat tree with custom content below
 */
export function WithBelowMessageList() {
    const tree = createMessageTree(mockMessages);
    const branches: BranchMap = {};

    return (
        <ChatBubbleTree
            tree={tree}
            branches={branches}
            setBranches={action("setBranches")}
            handleEdit={action("handleEdit")}
            handleRegenerateResponse={action("handleRegenerateResponse")}
            handleReply={action("handleReply")}
            handleRetry={action("handleRetry")}
            removeMessages={action("removeMessages")}
            isBotOnlyChat={true}
            isEditingMessage={false}
            isReplyingToMessage={false}
            messageStream={null}
            belowMessageList={
                <Box sx={{ textAlign: "center", color: "text.secondary" }}>
                    Custom content below messages
                </Box>
            }
        />
    );
} 
