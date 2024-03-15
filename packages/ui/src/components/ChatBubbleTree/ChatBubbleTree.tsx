import { LlmTaskInfo } from "@local/shared";
import { Box, IconButton, Typography, useTheme } from "@mui/material";
import { ChatBubble } from "components/ChatBubble/ChatBubble";
import { Dimensions } from "components/graphs/types";
import { SessionContext } from "contexts/SessionContext";
import { MessageTree } from "hooks/useMessageTree";
import { ArrowDownIcon } from "icons";
import { Dispatch, RefObject, SetStateAction, useContext, useEffect, useMemo, useState } from "react";
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

export const ScrollToBottomButton = ({
    containerRef,
}: {
    containerRef: RefObject<HTMLElement>,
}) => {
    const { palette } = useTheme();
    const [isVisible, setIsVisible] = useState(false);

    useEffect(() => {
        const checkScrollPosition = () => {
            const scroll = containerRef.current;
            if (!scroll) return;

            // Threshold to determine "close to the bottom"
            // Adjust this value based on your needs
            const threshold = 100;
            const isCloseToBottom = scroll.scrollHeight - scroll.scrollTop - scroll.clientHeight < threshold;

            setIsVisible(!isCloseToBottom);
        };

        const scroll = containerRef.current;
        scroll?.addEventListener("scroll", checkScrollPosition);
        setTimeout(checkScrollPosition, 2000);
        return () => {
            scroll?.removeEventListener("scroll", checkScrollPosition);
        };
    }, [containerRef]); // Re-run effect if containerRef changes

    const scrollToBottom = () => {
        const scroll = containerRef.current;
        if (!scroll) return;
        const lastMessage = scroll.lastElementChild;
        if (lastMessage) {
            lastMessage.scrollIntoView({ behavior: "smooth" });
        } else {
            scroll.scrollTop = scroll.scrollHeight;
        }
    };

    return (
        <IconButton
            onClick={scrollToBottom}
            size="small"
            sx={{
                background: palette.background.paper,
                position: "absolute",
                bottom: 8,
                left: "50%",
                transform: "translateX(-50%)",
                opacity: isVisible ? 0.8 : 0,
                transition: "opacity 0.2s ease-in-out !important",
            }}
        >
            <ArrowDownIcon fill={palette.background.textSecondary} />
        </IconButton>
    );
};

type MessageRenderData = {
    activeIndex: number;
    key: string;
    numSiblings: number;
    onActiveIndexChange: (newIndex: number) => unknown;
    onDeleted: (message: ChatMessageShape) => unknown;
    message: ChatMessageShape;
    isOwn: boolean;
}

export const ChatBubbleTree = ({
    branches,
    dimensions,
    dimRef,
    editMessage,
    handleReply,
    handleRetry,
    isBotOnlyChat,
    messageTasks,
    removeMessages,
    setBranches,
    tree,
}: {
    branches: BranchMap,
    dimensions: Dimensions,
    dimRef: RefObject<HTMLElement>,
    editMessage: (updatedMessage: ChatMessageShape) => unknown,
    handleReply: (message: ChatMessageShape) => unknown,
    handleRetry: (message: ChatMessageShape) => unknown,
    isBotOnlyChat: boolean,
    messageTasks: Record<string, LlmTaskInfo[]>,
    removeMessages: (messageIds: string[]) => unknown,
    setBranches: Dispatch<SetStateAction<BranchMap>>,
    tree: MessageTree<ChatMessageShape>,
}) => {
    const session = useContext(SessionContext);

    const messageData = useMemo<MessageRenderData[]>(() => {
        const renderMessage = (withSiblings: string[], activeIndex: number): MessageRenderData[] => {
            // Find information for current message
            const siblingId = withSiblings[activeIndex];
            const sibling = siblingId ? tree.map.get(siblingId) : null;
            if (!sibling) return [];
            const isOwn = sibling.message.user?.id === getCurrentUser(session).id;

            // Find information for next message
            // Check the stored branch data first
            let childId = branches[siblingId];
            // Fallback to first child if no branch data is found
            if (!childId) childId = sibling.children[0];
            const activeChildIndex = Math.max(sibling.children.findIndex(id => id === childId), 0);

            if (!sibling) return [];
            return [
                {
                    activeIndex,
                    key: sibling.message.id,
                    numSiblings: withSiblings.length,
                    onActiveIndexChange: (newIndex) => {
                        const siblingId = withSiblings[newIndex];
                        const parentId = sibling.message.parent?.id;
                        if (!siblingId) return;
                        if (!parentId) return; // TODO if root message, should reorder root
                        setBranches(prev => ({
                            ...prev,
                            [parentId]: siblingId,
                        }));
                    },
                    onDeleted: (message) => { removeMessages([message.id]); },
                    message: sibling.message,
                    isOwn,
                },
                ...childId ? renderMessage(sibling.children, activeChildIndex) : [],
            ];
        };
        return renderMessage(tree.roots, 0);
    }, [branches, removeMessages, session, setBranches, tree.map, tree.roots]);

    return (
        <Box ref={dimRef} sx={{
            position: "relative",
            display: "flex",
            flexDirection: "column",
            overflow: "auto",
            height: "auto",
        }}>
            {messageData.map((data) => (
                <ChatBubble
                    key={data.key}
                    activeIndex={data.activeIndex}
                    chatWidth={dimensions.width}
                    isBotOnlyChat={isBotOnlyChat}
                    numSiblings={data.numSiblings}
                    onActiveIndexChange={data.onActiveIndexChange}
                    onDeleted={data.onDeleted}
                    onReply={handleReply}
                    onRetry={handleRetry}
                    onUpdated={(updatedMessage) => { editMessage(updatedMessage); }}
                    message={data.message}
                    isOwn={data.isOwn}
                    tasks={messageTasks[data.message.id] || []}
                />
            ))}
        </Box>
    );
};
