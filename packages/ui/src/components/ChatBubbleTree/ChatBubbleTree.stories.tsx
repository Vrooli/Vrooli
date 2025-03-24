import { ChatMessageShape, ChatMessageStatus, ChatSocketEventPayloads, MINUTES_10_MS, uuid } from "@local/shared";
import { Box } from "@mui/material";
import { action } from "@storybook/addon-actions";
import { loggedOutSession, signedInPremiumWithCreditsSession, signedInUserId } from "../../__test/storybookConsts.js";
import { MessageTree } from "../../hooks/messages.js";
import { BranchMap } from "../../utils/localStorage.js";
import { ChatBubbleTree } from "./ChatBubbleTree.js";

const bot1Id = uuid();
const mockMessages: ChatMessageShape[] = [
    // Initial bot message
    {
        __typename: "ChatMessage" as const,
        id: "msg-1",
        // eslint-disable-next-line no-magic-numbers
        created_at: new Date(Date.now() - 10 * MINUTES_10_MS).toISOString(),
        // eslint-disable-next-line no-magic-numbers
        updated_at: new Date(Date.now() - 10 * MINUTES_10_MS).toISOString(),
        translations: [{
            __typename: "ChatMessageTranslation" as const,
            id: "trans-1",
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
        id: "msg-2",
        // eslint-disable-next-line no-magic-numbers
        created_at: new Date(Date.now() - 9 * MINUTES_10_MS).toISOString(),
        // eslint-disable-next-line no-magic-numbers
        updated_at: new Date(Date.now() - 9 * MINUTES_10_MS).toISOString(),
        translations: [{
            __typename: "ChatMessageTranslation" as const,
            id: "trans-2",
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
            id: "msg-1",
        },
    },
    // Second bot message - version 1
    {
        __typename: "ChatMessage" as const,
        id: "msg-3",
        // eslint-disable-next-line no-magic-numbers
        created_at: new Date(Date.now() - 8 * MINUTES_10_MS).toISOString(),
        // eslint-disable-next-line no-magic-numbers
        updated_at: new Date(Date.now() - 8 * MINUTES_10_MS).toISOString(),
        translations: [{
            __typename: "ChatMessageTranslation" as const,
            id: "trans-3",
            language: "en",
            text: "I'd be happy to help! What specific issue are you facing?",
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
            id: "msg-2",
        },
    },
    // Second bot message - version 2 (i.e. has same parent as version 1, but version index is 1)
    {
        __typename: "ChatMessage" as const,
        id: "msg-4",
        // eslint-disable-next-line no-magic-numbers
        created_at: new Date(Date.now() - 7 * MINUTES_10_MS).toISOString(),
        // eslint-disable-next-line no-magic-numbers
        updated_at: new Date(Date.now() - 7 * MINUTES_10_MS).toISOString(),
        translations: [{
            __typename: "ChatMessageTranslation" as const,
            id: "trans-4",
            language: "en",
            text: "I'd be happy to help! What specific issue are you facing?",
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
            id: "msg-2",
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

// Pre-create objects to avoid JSX scope creation
const defaultTree = createMessageTree(mockMessages);
const defaultBranches: BranchMap = {};

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
                <Story />
            </Outer>
        ),
    ],
};

/**
 * Default story: Basic chat conversation
 */
export function LoggedOut() {
    return (
        <ChatBubbleTree
            tree={defaultTree}
            branches={defaultBranches}
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
LoggedOut.parameters = {
    session: loggedOutSession,
};

export function SignedInPremiumWithCredits() {
    return (
        <ChatBubbleTree
            tree={defaultTree}
            branches={defaultBranches}
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
SignedInPremiumWithCredits.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Pre-create streaming message object
const streamingMessage: ChatSocketEventPayloads["responseStream"] = {
    __type: "stream",
    message: "I'm currently generating a response...",
    botId: "bot-1",
};

export function SingleMessage() {
    const singleMessage = mockMessages[0];
    const singleMessageTree = createMessageTree([singleMessage]);
    const singleMessageBranches: BranchMap = {};

    return (
        <ChatBubbleTree
            tree={singleMessageTree}
            branches={singleMessageBranches}
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
SingleMessage.parameters = {
    session: signedInPremiumWithCreditsSession,
};

/**
 * With Streaming Message: Shows a message being streamed
 */
export function StreamingMessage() {
    return (
        <ChatBubbleTree
            tree={defaultTree}
            branches={defaultBranches}
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
StreamingMessage.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Pre-create error message tree
const messagesWithError = [...mockMessages];
messagesWithError[2] = {
    ...messagesWithError[2],
    status: "failed" as ChatMessageStatus,
};
const errorTree = createMessageTree(messagesWithError);

/**
 * With Error Message: Shows a failed message state
 */
export function FailedMessage() {
    return (
        <ChatBubbleTree
            tree={errorTree}
            branches={defaultBranches}
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
FailedMessage.parameters = {
    session: signedInPremiumWithCreditsSession,
};

/**
 * Editing State: Shows the chat tree while editing a message
 */
export function EditingState() {
    return (
        <ChatBubbleTree
            tree={defaultTree}
            branches={defaultBranches}
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
EditingState.parameters = {
    session: signedInPremiumWithCreditsSession,
};

/**
 * With Custom Below Message List: Shows the chat tree with custom content below
 */
export function WithBelowMessageList() {
    return (
        <ChatBubbleTree
            tree={defaultTree}
            branches={defaultBranches}
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
WithBelowMessageList.parameters = {
    session: signedInPremiumWithCreditsSession,
};
