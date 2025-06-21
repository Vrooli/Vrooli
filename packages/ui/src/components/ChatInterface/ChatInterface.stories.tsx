import type { Meta, StoryObj } from "@storybook/react";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import { ChatInterface, type ChatInterfaceProps } from "./ChatInterface.js";
import { ViewDisplayType } from "../../types.js";
import { fixedHeightDecorator } from "../../__test/helpers/storybookDecorators.tsx";
import { signedInPremiumWithCreditsSession } from "../../__test/storybookConsts.js";
import { ChatMessageInput } from "../inputs/ChatMessageInput/ChatMessageInput.js";
import { EmptyStateWelcome } from "./EmptyStateWelcome.js";
import { EmptyStateCompact } from "./EmptyStateCompact.js";
import { EmptyStateError } from "./EmptyStateError.js";
import { EmptyStateLoading } from "./EmptyStateLoading.js";

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
                height: compact ? "auto" : "100%",
                minHeight: compact ? "300px" : "400px",
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
                            startingMessage: "Hello! I'm Assistant Pro, your specialized helper for technical documentation and code review. How can I assist you today?"
                        }
                    }
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