import { ChatMessageSearchTreeInput, ChatMessageSearchTreeResult, DUMMY_ID, endpointGetChatMessageTree } from "@local/shared";
import { Box, Typography } from "@mui/material";
import { ChatBubble } from "components/ChatBubble/ChatBubble";
import { SessionContext } from "contexts/SessionContext";
import { useDimensions } from "hooks/useDimensions";
import { useLazyFetch } from "hooks/useLazyFetch";
import { MessageNode, useMessageTree } from "hooks/useMessageTree";
import React, { useContext, useEffect, useState } from "react";
import { getCurrentUser } from "utils/authentication/session";
import { BranchMap, getCookieMessageTree, setCookieMessageTree } from "utils/cookies";
import { getDisplay, ListObject } from "utils/display/listTools";
import { ChatMessageShape } from "utils/shape/models/chatMessage";

const getTypingIndicatorText = (participants: ListObject[], maxChars: number) => {
    if (participants.length === 0) return "";
    if (participants.length === 1) return `${getDisplay(participants[0]).title} is typing`;
    if (participants.length === 2) return `${getDisplay(participants[0]).title}, ${getDisplay(participants[1]).title} are typing`;
    let text = `${getDisplay(participants[0]).title}, ${getDisplay(participants[1]).title}`;
    let remainingCount = participants.length - 2;
    while (remainingCount > 0 && (text.length + getDisplay(participants[remainingCount]).title.length + 5) <= maxChars) {
        text += `, ${participants[remainingCount]}`;
        remainingCount--;
    }
    if (remainingCount === 0) return `${text} are typing`;
    return `${text}, +${remainingCount} are typing`;
};

export const TypingIndicator = ({
    maxChars = 30,
    participants,
}: {
    maxChars?: number,
    participants: ListObject[]
}) => {
    const [dots, setDots] = useState("");

    useEffect(() => {
        const interval = setInterval(() => {
            if (dots.length < 3) setDots(dots + ".");
            else setDots("");
        }, 500);

        return () => clearInterval(interval);
    }, [dots]);

    const displayText = getTypingIndicatorText(participants, maxChars);

    if (!displayText) return null;

    return (
        <Box display="flex" flexDirection="row" justifyContent="flex-start" alignItems="center" gap={0} p={1}>
            <Typography variant="body2" p={1}>{displayText} {dots}</Typography>
        </Box>
    );
};

export const ChatBubbleTree = ({
    chatId,
    handleMessagesCountChange,
    handleReply,
    handleRetry,
}: {
    chatId: string;
    handleMessagesCountChange: (count: number) => unknown;
    handleReply: (message: ChatMessageShape) => unknown;
    handleRetry: (message: ChatMessageShape) => unknown;
}) => {
    const session = useContext(SessionContext);
    const { dimensions, ref: dimRef } = useDimensions();

    const { tree, messagesCount, addMessages, removeMessages, editMessage, clearMessages } = useMessageTree<ChatMessageShape>([]);
    const [branches, setBranches] = useState<BranchMap>(getCookieMessageTree(chatId)?.branches ?? {});
    useEffect(() => { handleMessagesCountChange(messagesCount); }, [handleMessagesCountChange, messagesCount]);

    // We query messages separate from the chat, since we must traverse the message tree
    const [getTreeData, { data: searchTreeData }] = useLazyFetch<ChatMessageSearchTreeInput, ChatMessageSearchTreeResult>(endpointGetChatMessageTree);

    // When chatId changes, clear the message tree and branches, and fetch new data
    useEffect(() => {
        if (chatId === DUMMY_ID) return;
        clearMessages();
        setBranches({});
        getTreeData({ chatId });
    }, [chatId, clearMessages, getTreeData]);
    useEffect(() => {
        if (!searchTreeData || searchTreeData.messages.length === 0) return;
        addMessages(searchTreeData.messages);
    }, [addMessages, searchTreeData]);

    useEffect(() => {
        // Update the cookie with current branches
        setCookieMessageTree(chatId, { branches, locationId: "someLocationId" }); // Adjust locationId as necessary
    }, [branches, chatId]);

    const renderMessage = (withSiblings: MessageNode<ChatMessageShape>[], activeIndex: number) => {
        const activeParent = activeIndex >= 0 && activeIndex < withSiblings.length ? withSiblings[activeIndex] : null;
        const activeChildId = activeParent ? branches[activeParent.message.id] : null;
        const activeChildIndex = (activeParent && activeChildId) ?
            activeParent.children.findIndex(child => child.message.id === activeChildId) :
            0;
        const isOwn = activeParent?.message.user?.id === getCurrentUser(session).id;

        if (!activeParent) return null;
        return (
            <React.Fragment key={activeParent.message.id} >
                <ChatBubble
                    activeIndex={activeIndex}
                    chatWidth={dimensions.width}
                    messagesCount={withSiblings.length}
                    onActiveIndexChange={(newIndex) => {
                        const childId = newIndex >= 0 && newIndex < activeParent.children.length ?
                            activeParent.children[newIndex].message.id :
                            null;
                        if (!childId) return;
                        setBranches(prevBranches => ({
                            ...prevBranches,
                            [activeParent.message.id]: childId,
                        }));
                    }}
                    onDeleted={(message) => { removeMessages([message.id]); }}
                    onReply={handleReply}
                    onRetry={handleRetry}
                    onUpdated={(updatedMessage) => { editMessage(updatedMessage); }}
                    message={activeParent.message}
                    isOwn={isOwn}
                />
                {renderMessage(activeParent.children, activeChildIndex)}
            </React.Fragment>
        );
    };

    return (
        <Box ref={dimRef} sx={{ minHeight: "min(400px, 33vh)" }}>
            {renderMessage(tree.roots, 0)}
        </Box>
    );
};
