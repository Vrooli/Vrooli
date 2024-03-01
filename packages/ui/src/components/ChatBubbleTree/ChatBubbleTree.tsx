import { Box, Typography } from "@mui/material";
import { ChatBubble } from "components/ChatBubble/ChatBubble";
import { SessionContext } from "contexts/SessionContext";
import { useDimensions } from "hooks/useDimensions";
import { MessageTree } from "hooks/useMessageTree";
import React, { Dispatch, SetStateAction, useContext, useEffect, useMemo, useState } from "react";
import { getCurrentUser } from "utils/authentication/session";
import { BranchMap } from "utils/cookies";
import { ListObject, getDisplay } from "utils/display/listTools";
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
        <Box
            display="flex"
            flexDirection="row"
            justifyContent="flex-start"
            alignItems="center"
            gap={0}
            p={1}
            sx={{
                width: "min(100%, 700px)",
                margin: "auto",
            }}
        >
            <Typography variant="body2" p={1}>{displayText} {dots}</Typography>
        </Box>
    );
};

export const ChatBubbleTree = ({
    branches,
    editMessage,
    handleReply,
    handleRetry,
    isBotOnlyChat,
    removeMessages,
    setBranches,
    tree,
}: {
    branches: BranchMap,
    editMessage: (updatedMessage: ChatMessageShape) => unknown;
    handleReply: (message: ChatMessageShape) => unknown;
    handleRetry: (message: ChatMessageShape) => unknown;
    isBotOnlyChat: boolean;
    removeMessages: (messageIds: string[]) => unknown;
    setBranches: Dispatch<SetStateAction<BranchMap>>;
    tree: MessageTree<ChatMessageShape>;
}) => {
    const session = useContext(SessionContext);
    const { dimensions, ref: dimRef } = useDimensions();

    const messageList = useMemo(() => {
        const renderMessage = (withSiblings: string[], activeIndex: number) => {
            // Find information for current message
            const siblingId = withSiblings[activeIndex];
            const sibling = siblingId ? tree.map.get(siblingId) : null;
            if (!sibling) return null;
            const isOwn = sibling.message.user?.id === getCurrentUser(session).id;

            // Find information for next message
            // Check the stored branch data first
            let childId = branches[siblingId];
            // Fallback to first child if no branch data is found
            if (!childId) childId = sibling.children[0];
            const activeChildIndex = Math.max(sibling.children.findIndex(id => id === childId), 0);

            if (!sibling) return null;
            return (
                <React.Fragment key={sibling.message.id} >
                    <ChatBubble
                        activeIndex={activeIndex}
                        chatWidth={dimensions.width}
                        isBotOnlyChat={isBotOnlyChat}
                        numSiblings={withSiblings.length}
                        onActiveIndexChange={(newIndex) => {
                            const siblingId = withSiblings[newIndex];
                            const parentId = sibling.message.parent?.id;
                            if (!siblingId) return;
                            if (!parentId) return; // TODO if root message, should reorder root
                            setBranches(prev => ({
                                ...prev,
                                [parentId]: siblingId,
                            }));
                        }}
                        onDeleted={(message) => { removeMessages([message.id]); }}
                        onReply={handleReply}
                        onRetry={handleRetry}
                        onUpdated={(updatedMessage) => { editMessage(updatedMessage); }}
                        message={sibling.message}
                        isOwn={isOwn}
                    />
                    {childId && renderMessage(sibling.children, activeChildIndex)}
                </React.Fragment>
            );
        };
        return renderMessage(tree.roots, 0);
    }, [branches, dimensions.width, editMessage, handleReply, handleRetry, isBotOnlyChat, removeMessages, session, setBranches, tree]);

    return (
        <Box ref={dimRef} sx={{ minHeight: "min(400px, 33vh)" }}>
            {messageList}
        </Box>
    );
};
