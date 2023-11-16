import { MessageNode, MessageTree } from "@local/shared";
import { Box, Typography } from "@mui/material";
import { ChatBubble } from "components/ChatBubble/ChatBubble";
import { SessionContext } from "contexts/SessionContext";
import { useDimensions } from "hooks/useDimensions";
import React, { Dispatch, SetStateAction, useContext, useEffect, useState } from "react";
import { getCurrentUser } from "utils/authentication/session";
import { ChatMessageBranch, getCookieMessageTree, setCookieMessageTree } from "utils/cookies";
import { getDisplay, ListObject } from "utils/display/listTools";
import { updateArray } from "utils/shape/general";
import { ChatShape } from "utils/shape/models/chat";
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
    allMessages,
    chatId,
    handleUpdate,
}: {
    allMessages: ChatMessageShape[];
    chatId: string;
    handleUpdate: Dispatch<SetStateAction<ChatShape>>;
}) => {
    const session = useContext(SessionContext);
    const { dimensions, ref: dimRef } = useDimensions();
    console.log("in chat bubble tree: allMessages", allMessages.map(m => m.translations));

    // Extract branches from the cookie
    const initialMessageTreeData = getCookieMessageTree(chatId);
    const [branches, setBranches] = useState<ChatMessageBranch[]>(initialMessageTreeData?.branches ?? []);

    const messageTree = new MessageTree(allMessages);
    console.log("in chat bubble tree: messageTree", messageTree);

    useEffect(() => {
        // Update the cookie with current branches
        setCookieMessageTree(chatId, { branches, locationId: "someLocationId" }); // Adjust locationId as necessary
    }, [branches, chatId]);

    const changeBranch = (messageId: string, childId: string) => {
        setBranches(prevBranches => ({
            ...prevBranches,
            [messageId]: childId,
        }));
    };
    const renderMessage = (node: MessageNode<ChatMessageShape>) => {
        // Find the active child, or default to first child
        const activeChildId = branches[node.message.id];
        const activeChild = activeChildId ?
            node.children.find(child => child.message.id === activeChildId) :
            node.children.length > 0 ?
                node.children[0] :
                undefined;
        const isOwn = node.message.user?.id === getCurrentUser(session).id;

        return (
            <React.Fragment key={node.message.id} >
                <ChatBubble
                    chatWidth={dimensions.width}
                    onDeleted={(deletedMessage) => {
                        handleUpdate(c => ({
                            ...c,
                            // TODO should be deleting messages in tree instead
                            messages: c.messages.filter(m => m.id !== deletedMessage.id),
                        }));
                    }}
                    onUpdated={(updatedMessage) => {
                        handleUpdate(c => ({
                            ...c,
                            // TODO should be updating messages in tree instead
                            messages: updateArray(
                                c.messages,
                                c.messages.findIndex(m => m.id === updatedMessage.id),
                                updatedMessage,
                            ),
                        }));
                    }}
                    message={node.message}
                    index={activeChildId}  // TODO This was auto-generated - might need to be adjusted if it isn't intended to be the ID
                    isOwn={isOwn}
                />
                {activeChild && renderMessage(activeChild)}
            </React.Fragment>
        );
    };

    return (
        <Box ref={dimRef} sx={{ minHeight: "min(400px, 33vh)" }}>
            {
                messageTree.getRoots().map(root => {
                    return renderMessage(root);
                })
            }
        </Box>
    );
};
