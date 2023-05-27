import { AddIcon, Chat, ChatMessage, EditIcon } from "@local/shared";
import { Avatar, Box, IconButton, Stack, Typography, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { MarkdownInputBase } from "components/inputs/MarkdownInputBase/MarkdownInputBase";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getDisplay } from "utils/display/listTools";
import { getTranslation, getUserLanguages } from "utils/display/translationTools";
import { SessionContext } from "utils/SessionContext";
import { ChatViewProps } from "views/types";

interface ChatBubbleProps {
    isOwnMessage: boolean;
    children: React.ReactNode;
}

export const ChatBubble = ({ isOwnMessage, children }: ChatBubbleProps) => (
    <Box
        sx={{
            display: "inline-flex",
            flexDirection: "column",
            alignItems: isOwnMessage ? "flex-end" : "flex-start",
            p: 1,
            pl: isOwnMessage ? 2 : 1,
            pr: isOwnMessage ? 1 : 2,
            backgroundColor: isOwnMessage ? "#dcf8c6" : "#fff",
            borderRadius: "8px",
            boxShadow: 1,
            maxWidth: "70%",
            position: "relative",
        }}
    >
        {children}
    </Box>
);

export const ChatView = ({
    display = "page",
    zIndex,
}: ChatViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const lng = useMemo(() => getUserLanguages(session)[0], [session]);

    const chat: Chat = {} as any; //TODO
    const messages: ChatMessage[] = []; //TODO

    const { title, subtitle } = useMemo(() => getDisplay(chat, getUserLanguages(session)), [chat, lng, session]);

    const [editingText, setEditingText] = useState<string>("");
    const [newMessageText, setNewMessageText] = useState<string>("");

    const handleAvatarClick = (userId: string) => {
        // Implement this to navigate to user's profile.
    };

    const handleEditMessageClick = (messageId: string) => {
        const message = messages.find((m) => m.id === messageId);
        if (!message) return;
        setEditingText(getTranslation(message, getUserLanguages(session), true)?.text ?? "");
    };

    const handleSendButtonClick = () => {
        // Implement this to send a new message.
    };

    return (
        <>
            <TopBar
                display={display}
                onClose={() => { }}
                titleData={{ title }}
            />
            <Stack direction="column" spacing={2}>
                <Box sx={{ overflowY: "auto", maxHeight: "calc(100vh - 64px)" }}>
                    {messages.map((message: ChatMessage, index) => (
                        <Box
                            key={index}
                            sx={{ display: "flex", justifyContent: message.you.canUpdate ? "flex-end" : "flex-start", p: 2 }}
                        >
                            {!message.you.canUpdate && (
                                <Avatar
                                    // src={message.user.avatar} TODO
                                    alt={message.user.name ?? message.user.handle}
                                    onClick={() => handleAvatarClick(message.user.id)}
                                    sx={{ bgcolor: message.user.isBot ? "grey" : undefined }}
                                />
                            )}
                            <Box sx={{ ml: !message.you.canUpdate ? 2 : 0, mr: message.you.canUpdate ? 2 : 0 }}>
                                <ChatBubble isOwnMessage={message.you.canUpdate}>
                                    <Typography>{getTranslation(message, getUserLanguages(session), true)?.text ?? ""}</Typography>
                                    {message.you.canUpdate && (
                                        <IconButton onClick={() => handleEditMessageClick(message.id)} sx={{ position: "absolute", top: 0, right: 0 }}>
                                            <EditIcon />
                                        </IconButton>
                                    )}
                                    {/* <ReactionButtons reactions={message.reactions} messageId={message.id} /> */}
                                </ChatBubble>
                            </Box>
                        </Box>
                    ))}
                </Box>
                <Box>
                    <MarkdownInputBase
                        disableAssistant={true}
                        fullWidth
                        minRows={1}
                        maxRows={15}
                        name="newMessage"
                        value={newMessageText}
                        onChange={(value) => setNewMessageText(value)}
                        sxs={{
                            root: { flexGrow: 1 },
                            textArea: { paddingRight: 8 },
                        }}
                        zIndex={zIndex}
                    />
                    <ColorIconButton
                        aria-label='fetch-handles'
                        background={palette.secondary.main}
                        onClick={handleSendButtonClick}
                        sx={{
                            borderRadius: "100%",
                            position: "absolute",
                            right: 0,
                            bottom: 0,
                            margin: 1,
                        }}
                    >
                        <AddIcon />
                    </ColorIconButton>
                </Box>
            </Stack>
        </>
    );
};
