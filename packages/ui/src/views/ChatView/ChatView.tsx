import { AddIcon, BotIcon, Chat, ChatMessage, DUMMY_ID, EditIcon, useLocation, UserIcon } from "@local/shared";
import { Avatar, Box, IconButton, Stack, Typography, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { MarkdownInput } from "components/inputs/MarkdownInput/MarkdownInput";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getDisplay } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { getTranslation, getUserLanguages } from "utils/display/translationTools";
import { getObjectUrl } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
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
    botSettings,
    chatId,
    context,
    display = "page",
    onClose,
    task,
    zIndex,
}: ChatViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const lng = useMemo(() => getUserLanguages(session)[0], [session]);

    const chat: Chat = {} as any; //TODO
    const [messages, setMessages] = useState<ChatMessage[]>([]); //TODO
    useEffect(() => {
        // If chatting with default AI assistant, add start message so that we don't need 
        // to query the server for it.
        if (chatId === "Valyxa") {
            const startText = t(task ?? "start", { lng, ns: "tasks", defaultValue: "Hello👋, I'm Valyxa! How can I assist you?" });
            setMessages([{
                __typename: "ChatMessage" as const,
                translations: [{
                    id: DUMMY_ID,
                    created_at: new Date().toISOString(),
                    updated_at: new Date().toISOString(),
                    language: lng,
                    text: startText,
                }],
                user: {
                    __typename: "User" as const,
                    id: "4b038f3b-f1f7-1f9b-8f4b-cff4b8f9b20f",
                    isBot: true,
                    name: "Valyxa",
                },
                you: {
                    canDelete: false,
                    canUpdate: false,
                    canReply: true,
                    canReport: false,
                    canReact: false,
                    reaction: null,
                },
            }] as any[]);
        }
    }, [chatId, lng, t, task]);

    const { title, subtitle } = useMemo(() => getDisplay(chat, getUserLanguages(session)), [chat, session]);

    const handleSendButtonClick = () => {
        // Implement this to send a new message.
    };
    console.log("context in chatview", messages);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={{
                editingMessage: "",
                newMessage: context ?? "",
            }}
            onSubmit={(values, helpers) => {
                const isEditing = values.editingMessage.trim().length > 0;
                if (isEditing) {
                    //TODO
                } else {
                    //TODO
                    // for now, just add the message to the list
                    const newMessage: ChatMessage = {
                        __typename: "ChatMessage" as const,
                        translations: [{
                            id: DUMMY_ID,
                            created_at: new Date().toISOString(),
                            updated_at: new Date().toISOString(),
                            language: lng,
                            text: values.newMessage,
                        }],
                        user: {
                            __typename: "User" as const,
                            id: session.user.id,
                            isBot: false,
                            name: session.user.name,
                        },
                        you: {
                            canDelete: true,
                            canUpdate: true,
                            canReply: true,
                            canReport: true,
                            canReact: true,
                            reaction: null,
                        },
                    };
                    setMessages([...messages, newMessage]);
                    helpers.setFieldValue("newMessage", "");
                }
            }}
        // validate={async (values) => await validateNoteValues(values, existing)}
        >
            {(formik) => <>
                <TopBar
                    display={display}
                    onClose={() => {
                        if (formik.values.editingMessage.trim().length > 0) {
                            PubSub.get().publishAlertDialog({
                                messageKey: "UnsavedChangesBeforeCancel",
                                buttons: [
                                    { labelKey: "Yes", onClick: () => { onClose(); } },
                                    { labelKey: "No" },
                                ],
                            });
                        } else {
                            onClose();
                        }
                    }}
                    titleData={{ title: firstString(title, botSettings ? "AI Chat" : "Chat") }}
                />
                <Stack direction="column" spacing={4}>
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
                                        onClick={() => {
                                            const url = getObjectUrl(message.user);
                                            if (formik.values.editingMessage.trim().length > 0) {
                                                PubSub.get().publishAlertDialog({
                                                    messageKey: "UnsavedChangesBeforeContinue",
                                                    buttons: [
                                                        { labelKey: "Yes", onClick: () => { setLocation(url); } },
                                                        { labelKey: "No" },
                                                    ],
                                                });
                                            } else {
                                                setLocation(url);
                                            }
                                        }}
                                        sx={{
                                            bgcolor: message.user.isBot ? "grey" : undefined,
                                            cursor: "pointer",
                                        }}
                                    >
                                        {message.user.isBot ? <BotIcon width="75%" height="75%" /> : <UserIcon width="75%" height="75%" />}
                                    </Avatar>
                                )}
                                <Box sx={{ ml: !message.you.canUpdate ? 2 : 0, mr: message.you.canUpdate ? 2 : 0 }}>
                                    <ChatBubble isOwnMessage={message.you.canUpdate}>
                                        <Typography>{getTranslation(message, getUserLanguages(session), true)?.text ?? ""}</Typography>
                                        {message.you.canUpdate && (
                                            <IconButton onClick={() => {
                                                const message = messages.find((m) => m.id === message.id);
                                                if (!message) return;
                                                formik.setFieldValue("editingMessage", getTranslation(message, getUserLanguages(session), true)?.text ?? "");
                                            }} sx={{ position: "absolute", top: 0, right: 0 }}>
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
                        <MarkdownInput
                            disableAssistant={true}
                            fullWidth
                            minRows={1}
                            maxRows={15}
                            name="newMessage"
                            sxs={{
                                bar: { borderRadius: 0 },
                                textArea: { paddingRight: 4, borderRadius: 0 },
                            }}
                            zIndex={zIndex}
                        />
                        <ColorIconButton
                            aria-label='fetch-handles'
                            background={palette.secondary.main}
                            onClick={() => { formik.handleSubmit(); }}
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
            </>}
        </Formik>
    );
};
