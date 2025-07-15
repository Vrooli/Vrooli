import type { Meta, StoryObj } from "@storybook/react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import { ChatInterface, type ChatInterfaceProps } from "./ChatInterface.js";
import { ViewDisplayType } from "../../types.js";
import { fixedHeightDecorator } from "../../__test/helpers/storybookDecorators.tsx";
import { signedInPremiumWithCreditsSession, signedInUserId } from "../../__test/storybookConsts.js";
import { ChatMessageInput } from "../inputs/ChatMessageInput/ChatMessageInput.js";
import { EmptyStateWelcome } from "./EmptyStateWelcome.js";
import { EmptyStateCompact } from "./EmptyStateCompact.js";
import { EmptyStateError } from "./EmptyStateError.js";
import { EmptyStateLoading } from "./EmptyStateLoading.js";
import { ChatBubbleTree } from "../ChatBubbleTree/ChatBubbleTree.js";
import { MessageTree } from "../../hooks/messages.js";
import { type BranchMap } from "../../utils/localStorage.js";
import { generatePK, MINUTES_10_MS, type ChatMessageShape, type ChatMessageStatus } from "@vrooli/shared";
import { useCallback, useMemo, useState } from "react";

const meta: Meta<typeof ChatInterface> = {
    title: "Components/Chat/ChatInterface",
    component: ChatInterface,
    parameters: {
        layout: "fullscreen",
    },
    tags: ["autodocs"],
    argTypes: {
        display: {
            control: { type: "select" },
            options: Object.values(ViewDisplayType),
        },
    },
};

export default meta;
type Story = StoryObj<typeof meta>;

// Helper function to handle suggestions in stories
const handleSuggestionClick = (suggestion: string) => {
    console.log("Suggestion clicked:", suggestion);
};

// Simplified ChatInterface component for stories that bypasses the complex chat system
function SimpleChatInterface({ 
    noMessagesContent,
    loadingContent,
    placeholder = "What would you like to do?",
    disabled = false,
    display = ViewDisplayType.Page,
    showSettingsButton = true,
    compact = false,
    isLoading = false,
    ...props 
}: ChatInterfaceProps) {
    return (
        <Box
            sx={{
                display: "flex",
                flexDirection: "column",
                height: compact ? "auto" : "100vh",
                minHeight: compact ? "300px" : "100vh",
                overflow: "hidden",
                backgroundColor: "background.default",
            }}
        >
            <Box
                sx={{
                    flex: 1,
                    overflowY: "auto",
                    overflowX: "hidden",
                    display: "flex",
                    flexDirection: "column",
                    backgroundColor: "background.default",
                    position: "relative",
                    paddingTop: 2,
                    paddingBottom: 4,
                }}
            >
                {isLoading ? loadingContent : noMessagesContent}
            </Box>
            
            <ChatMessageInput
                disabled={disabled || isLoading}
                display={display}
                isLoading={false}
                message=""
                messageBeingEdited={null}
                messageBeingRepliedTo={null}
                participantsTyping={[]}
                placeholder={placeholder}
                setMessage={() => {}}
                stopEditingMessage={() => {}}
                stopReplyingToMessage={() => {}}
                submitMessage={() => {}}
                taskInfo={{
                    activeTask: null,
                    inactiveTasks: [],
                    contexts: {},
                    unselectTask: () => {},
                    selectTask: () => {},
                }}
            />
        </Box>
    );
}

export const Default: Story = {
    render: (args) => <SimpleChatInterface {...args} noMessagesContent={<EmptyStateWelcome onSuggestionClick={handleSuggestionClick} bot={null} />} />,
    args: {
        display: ViewDisplayType.Page,
        showSettingsButton: true,
        placeholder: "What would you like to do?",
        disabled: false,
        compact: false,
    },
    decorators: [fixedHeightDecorator(600)],
    parameters: {
        session: signedInPremiumWithCreditsSession,
    },
};

export const NoMessages: Story = {
    render: (args) => <SimpleChatInterface {...args} noMessagesContent={<EmptyStateWelcome onSuggestionClick={handleSuggestionClick} bot={null} />} />,
    args: {
        display: ViewDisplayType.Page,
        showSettingsButton: true,
        placeholder: "What would you like to do?",
        disabled: false,
        compact: false,
    },
    decorators: [fixedHeightDecorator(600)],
    parameters: {
        session: signedInPremiumWithCreditsSession,
    },
};

export const DialogMode: Story = {
    render: (args) => (
        <SimpleChatInterface 
            {...args} 
            noMessagesContent={<EmptyStateCompact onSuggestionClick={handleSuggestionClick} />} 
        />
    ),
    args: {
        display: ViewDisplayType.Dialog,
        showSettingsButton: false,
        placeholder: "Type your message...",
        disabled: false,
        compact: true,
    },
    decorators: [
        (Story) => (
            <Box 
                height="500px" 
                width="400px" 
                bgcolor="background.paper" 
                border="1px solid"
                borderColor="divider"
                borderRadius={2}
                overflow="hidden"
            >
                <Story />
            </Box>
        ),
    ],
    parameters: {
        session: signedInPremiumWithCreditsSession,
    },
};

export const Disabled: Story = {
    render: (args) => (
        <SimpleChatInterface 
            {...args} 
            noMessagesContent={<EmptyStateError errorMessage="The chat service is currently disabled" />} 
        />
    ),
    args: {
        display: ViewDisplayType.Page,
        showSettingsButton: true,
        placeholder: "Chat is currently disabled",
        disabled: true,
        compact: false,
    },
    decorators: [fixedHeightDecorator(600)],
    parameters: {
        session: signedInPremiumWithCreditsSession,
    },
};

export const MobilePortrait: Story = {
    render: (args) => (
        <SimpleChatInterface 
            {...args} 
            noMessagesContent={<EmptyStateCompact onSuggestionClick={handleSuggestionClick} />} 
        />
    ),
    args: {
        display: ViewDisplayType.Page,
        showSettingsButton: true,
        placeholder: "What can I help you with?",
        disabled: false,
        compact: false,
    },
    decorators: [
        (Story) => (
            <Box 
                height="700px" 
                width="375px" 
                bgcolor="background.paper"
                border="1px solid"
                borderColor="divider"
                borderRadius="12px"
                overflow="hidden"
                mx="auto"
            >
                <Story />
            </Box>
        ),
    ],
    parameters: {
        session: signedInPremiumWithCreditsSession,
    },
};

export const BotChat: Story = {
    render: (args) => <SimpleChatInterface 
        {...args} 
        noMessagesContent={
            <EmptyStateWelcome 
                onSuggestionClick={handleSuggestionClick}
                bot={{
                    id: "test-bot-123",
                    name: "Assistant Pro",
                    isBot: true,
                    profileImage: null,
                    botSettings: {
                        persona: {
                            startingMessage: "Hello! I'm Assistant Pro, your specialized helper for technical documentation and code review. How can I assist you today?",
                        },
                    },
                }}
                isMultiParticipant={false}
            />
        } 
    />,
    args: {
        display: ViewDisplayType.Page,
        showSettingsButton: true,
        placeholder: "What would you like to do?",
        disabled: false,
        compact: false,
    },
    decorators: [fixedHeightDecorator(600)],
    parameters: {
        session: signedInPremiumWithCreditsSession,
    },
};

export const MultiParticipantChat: Story = {
    render: (args) => <SimpleChatInterface 
        {...args} 
        noMessagesContent={
            <EmptyStateWelcome 
                onSuggestionClick={handleSuggestionClick}
                bot={null}
                isMultiParticipant={true}
            />
        } 
    />,
    args: {
        display: ViewDisplayType.Page,
        showSettingsButton: true,
        placeholder: "What would you like to do?",
        disabled: false,
        compact: false,
    },
    decorators: [fixedHeightDecorator(600)],
    parameters: {
        session: signedInPremiumWithCreditsSession,
    },
};

export const Loading: Story = {
    render: (args) => <SimpleChatInterface {...args} isLoading={true} loadingContent={<EmptyStateLoading showBotInfo={false} />} />,
    args: {
        display: ViewDisplayType.Page,
        showSettingsButton: true,
        placeholder: "What would you like to do?",
        disabled: false,
        compact: false,
    },
    decorators: [fixedHeightDecorator(600)],
    parameters: {
        session: signedInPremiumWithCreditsSession,
    },
};

export const LoadingWithMessages: Story = {
    render: (args) => <SimpleChatInterface {...args} isLoading={true} loadingContent={<EmptyStateLoading showMessages={true} showBotInfo={false} />} />,
    args: {
        display: ViewDisplayType.Page,
        showSettingsButton: true,
        placeholder: "What would you like to do?",
        disabled: false,
        compact: false,
    },
    decorators: [fixedHeightDecorator(600)],
    parameters: {
        session: signedInPremiumWithCreditsSession,
    },
};

export const LoadingCompact: Story = {
    render: (args) => <SimpleChatInterface {...args} isLoading={true} loadingContent={<EmptyStateLoading showBotInfo={false} />} />,
    args: {
        display: ViewDisplayType.Dialog,
        showSettingsButton: false,
        placeholder: "Type your message...",
        disabled: false,
        compact: true,
    },
    decorators: [
        (Story) => (
            <Box 
                height="500px" 
                width="400px" 
                bgcolor="background.paper" 
                border="1px solid"
                borderColor="divider"
                borderRadius={2}
                overflow="hidden"
            >
                <Story />
            </Box>
        ),
    ],
    parameters: {
        session: signedInPremiumWithCreditsSession,
    },
};

export const LoadingBotChat: Story = {
    render: (args) => <SimpleChatInterface {...args} isLoading={true} loadingContent={<EmptyStateLoading showBotInfo={true} />} />,
    args: {
        display: ViewDisplayType.Page,
        showSettingsButton: true,
        placeholder: "What would you like to do?",
        disabled: false,
        compact: false,
    },
    decorators: [fixedHeightDecorator(600)],
    parameters: {
        session: signedInPremiumWithCreditsSession,
    },
};

// Generate bot ID for messages
const bot1Id = generatePK().toString();

// Create sample messages for demonstration
const botMessageId = generatePK().toString();
const userMessageId = generatePK().toString();
const botResponseId = generatePK().toString();

const mockMessages: ChatMessageShape[] = [
    // Initial bot message
    {
        __typename: "ChatMessage" as const,
        id: botMessageId,
        createdAt: new Date(Date.now() - 10 * MINUTES_10_MS).toISOString(),
        updatedAt: new Date(Date.now() - 10 * MINUTES_10_MS).toISOString(),
        translations: [{
            __typename: "ChatMessageTranslation" as const,
            id: generatePK().toString(),
            language: "en",
            text: "Hello! I'm Valyxa, your AI assistant. How can I help you today?",
        }],
        reactionSummaries: [],
        status: "sent" as ChatMessageStatus,
        user: {
            id: bot1Id,
            name: "Valyxa",
            handle: "valyxa",
            isBot: true,
            __typename: "User",
        },
        versionIndex: 0,
        parent: null,
    },
    // User message
    {
        __typename: "ChatMessage" as const,
        id: userMessageId,
        createdAt: new Date(Date.now() - 8 * MINUTES_10_MS).toISOString(),
        updatedAt: new Date(Date.now() - 8 * MINUTES_10_MS).toISOString(),
        translations: [{
            __typename: "ChatMessageTranslation" as const,
            id: generatePK().toString(),
            language: "en",
            text: "Can you help me understand how to use the chat interface?",
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
            id: botMessageId,
        },
    },
    // Bot response
    {
        __typename: "ChatMessage" as const,
        id: botResponseId,
        createdAt: new Date(Date.now() - 6 * MINUTES_10_MS).toISOString(),
        updatedAt: new Date(Date.now() - 6 * MINUTES_10_MS).toISOString(),
        translations: [{
            __typename: "ChatMessageTranslation" as const,
            id: generatePK().toString(),
            language: "en",
            text: `Of course! Here's a quick guide to using the chat interface:

**Key Features:**
- 💬 Type your message in the input box at the bottom
- 🔄 Edit or regenerate messages by hovering over them
- 👍 React to messages with emojis
- 🌿 Navigate between different versions of AI responses
- ⚙️ Configure chat settings using the settings button

**Tips:**
1. You can use **Markdown** formatting in your messages
2. Code blocks are supported with syntax highlighting
3. Multi-line messages work great - just press Shift+Enter

Would you like me to demonstrate any specific feature?`,
        }],
        reactionSummaries: [{
            __typename: "ReactionSummary" as const,
            count: 1,
            emoji: "👍",
        }],
        status: "sent" as ChatMessageStatus,
        user: {
            id: bot1Id,
            name: "Valyxa",
            handle: "valyxa",
            isBot: true,
            __typename: "User",
        },
        versionIndex: 0,
        parent: {
            __typename: "ChatMessageParent" as const,
            id: userMessageId,
        },
    },
];

// Create message tree (same pattern as ChatBubbleTree stories)
function createMessageTree(messages: ChatMessageShape[]): MessageTree<ChatMessageShape> {
    const tree = new MessageTree<ChatMessageShape>();

    // Add all messages to the tree
    messages.forEach(msg => {
        tree.addMessageToTree(msg);
    });

    return tree;
}

// Helper hook to manage message tree
function useStoryMessages(initialMessages: ChatMessageShape[]) {
    const [tree, setTree] = useState(() => createMessageTree(initialMessages));
    const [branches, setBranches] = useState<BranchMap>({});

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

    return {
        tree,
        branches,
        setBranches,
        removeMessages,
    };
}


// Simple test using exact ChatBubbleTree pattern
export const WithMessages: Story = {
    render: (args) => {
        // Copy the exact working pattern from ChatBubbleTree stories
        const tree = useMemo(() => {
            const messageTree = new MessageTree<ChatMessageShape>();
            mockMessages.forEach(msg => {
                messageTree.addMessageToTree(msg);
            });
            return messageTree;
        }, []);
        const [branches, setBranches] = useState<BranchMap>({});
        
        const removeMessages = useCallback((messageIds: string[]) => {
            console.log("Remove messages:", messageIds);
        }, []);
        
        console.log("Direct tree test:", {
            messageCount: mockMessages.length,
            treeCount: tree.getMessagesCount(),
            roots: tree.getRoots().length,
        });
        
        return (
            <Box
                sx={{
                    display: "flex",
                    flexDirection: "column",
                    height: "100vh",
                    overflow: "hidden",
                    backgroundColor: "background.default",
                }}
            >
                <Box
                    sx={{
                        flex: 1,
                        overflowY: "auto",
                        overflowX: "hidden",
                        display: "flex",
                        flexDirection: "column",
                        backgroundColor: "background.default",
                        position: "relative",
                        paddingTop: 2,
                        paddingBottom: 4,
                    }}
                >
                    <ChatBubbleTree
                        branches={branches}
                        handleEdit={() => console.log("Edit message")}
                        handleRegenerateResponse={() => console.log("Regenerate response")}
                        handleReply={() => console.log("Reply to message")}
                        handleRetry={() => console.log("Retry message")}
                        isBotOnlyChat={true}
                        isEditingMessage={false}
                        isReplyingToMessage={false}
                        messageStream={null}
                        removeMessages={removeMessages}
                        setBranches={setBranches}
                        tree={tree}
                    />
                    
                    {/* Debug overlay */}
                    <Box sx={{
                        position: "absolute",
                        top: 10,
                        right: 10,
                        backgroundColor: "rgba(0,0,0,0.8)",
                        color: "white",
                        padding: 1,
                        borderRadius: 1,
                        fontSize: "12px",
                    }}>
                        Tree: {tree.getMessagesCount()} messages, {tree.getRoots().length} roots
                    </Box>
                </Box>
                
                <ChatMessageInput
                    disabled={false}
                    display={ViewDisplayType.Page}
                    isLoading={false}
                    message=""
                    messageBeingEdited={null}
                    messageBeingRepliedTo={null}
                    participantsTyping={[]}
                    placeholder="Type a message..."
                    setMessage={() => {}}
                    stopEditingMessage={() => {}}
                    stopReplyingToMessage={() => {}}
                    submitMessage={() => {}}
                    taskInfo={{
                        activeTask: null,
                        inactiveTasks: [],
                        contexts: {},
                        unselectTask: () => {},
                        selectTask: () => {},
                    }}
                />
            </Box>
        );
    },
    args: {
        display: ViewDisplayType.Page,
        showSettingsButton: true,
        placeholder: "Type a message...",
        disabled: false,
        compact: false,
    },
    decorators: [fixedHeightDecorator(600)],
    parameters: {
        session: signedInPremiumWithCreditsSession,
    },
};

// Simplified stories that work
export const WithMessagesCompact: Story = {
    render: (args) => (
        <Box 
            height="500px" 
            width="400px" 
            bgcolor="background.paper" 
            border="1px solid"
            borderColor="divider"
            borderRadius={2}
            overflow="hidden"
        >
            <SimpleChatInterface 
                {...args} 
                noMessagesContent={<EmptyStateCompact onSuggestionClick={handleSuggestionClick} />} 
            />
        </Box>
    ),
    args: {
        display: ViewDisplayType.Dialog,
        showSettingsButton: false,
        placeholder: "Type your message...",
        disabled: false,
        compact: true,
    },
    parameters: {
        session: signedInPremiumWithCreditsSession,
    },
};
