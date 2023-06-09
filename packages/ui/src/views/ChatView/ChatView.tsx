import { AddIcon, BotIcon, Chat, ChatMessage, DUMMY_ID, EditIcon, useLocation, UserIcon } from "@local/shared";
import { Avatar, Box, IconButton, Stack, useTheme } from "@mui/material";
import { MarkdownInput } from "components/inputs/MarkdownInput/MarkdownInput";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { Formik } from "formik";
import { useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session";
import { getDisplay } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { getTranslation, getUserLanguages } from "utils/display/translationTools";
import { getObjectUrl } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { ChatViewProps } from "views/types";

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
                        created_at: new Date().toISOString(),
                        updated_at: new Date().toISOString(),
                        translations: [{
                            __typename: "ChatMessageTranslation" as const,
                            id: DUMMY_ID,
                            language: lng,
                            text: values.newMessage,
                        }],
                        user: {
                            __typename: "User" as const,
                            id: getCurrentUser(session).id,
                            isBot: false,
                            name: getCurrentUser(session).name,
                        },
                        you: {
                            canDelete: true,
                            canUpdate: true,
                            canReply: true,
                            canReport: true,
                            canReact: true,
                            reaction: null,
                        },
                    } as any;
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
                    title={firstString(title, botSettings ? "AI Chat" : "Chat")}
                />
                <Stack direction="column" spacing={4}>
                    <Box sx={{ overflowY: "auto", maxHeight: "calc(100vh - 64px)" }}>
                        {messages.map((message: ChatMessage, index) => {
                            const isOwn = message.you.canUpdate;
                            return (
                                // Box with message, avatar, and reactions
                                <Box
                                    key={index}
                                    sx={{ display: "flex", justifyContent: isOwn ? "flex-end" : "flex-start", p: 2 }}
                                >
                                    {/* Avatar only display if it's not your message */}
                                    {!isOwn && (
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
                                                boxShadow: 2,
                                                cursor: "pointer",
                                            }}
                                        >
                                            {message.user.isBot ? <BotIcon width="75%" height="75%" /> : <UserIcon width="75%" height="75%" />}
                                        </Avatar>
                                    )}
                                    {/* Message bubble with reaction */}
                                    <Box sx={{ ml: !isOwn ? 2 : 0, mr: isOwn ? 2 : 0 }}>
                                        <Box
                                            sx={{
                                                display: "block",
                                                p: 1,
                                                pl: 2,
                                                pr: 2,
                                                ml: isOwn ? "auto" : 0,
                                                mr: isOwn ? 0 : "auto",
                                                backgroundColor: isOwn ? "#88d17e" : "#fff",
                                                borderRadius: "8px",
                                                boxShadow: 2,
                                                maxWidth: "80%",
                                                position: "relative",
                                                wordBreak: "break-word",
                                            }}
                                        >
                                            <MarkdownDisplay
                                                content={getTranslation(message, getUserLanguages(session), true)?.text}
                                                sx={{
                                                    // Make room for the edit button
                                                    ...(isOwn ? { pr: 6 } : {}),
                                                }}
                                            />
                                            {isOwn && (
                                                <IconButton onClick={() => {
                                                    const message = messages.find((m) => m.id === message.id);
                                                    if (!message) return;
                                                    formik.setFieldValue("editingMessage", getTranslation(message as ChatMessage, getUserLanguages(session), true)?.text ?? "");
                                                }} sx={{ position: "absolute", top: 0, right: 0 }}>
                                                    <EditIcon />
                                                </IconButton>
                                            )}
                                        </Box>
                                    </Box>
                                </Box>
                            );
                        })}
                    </Box>
                    <Box>
                        <MarkdownInput
                            actionButtons={[{
                                Icon: AddIcon,
                                onClick: () => { formik.handleSubmit(); },
                            }]}
                            disableAssistant={true}
                            fullWidth
                            maxChars={1500}
                            minRows={1}
                            maxRows={15}
                            name="newMessage"
                            sxs={{
                                bar: { borderRadius: 0 },
                                textArea: { paddingRight: 4, borderRadius: 0 },
                            }}
                            zIndex={zIndex}
                        />
                    </Box>
                </Stack>
            </>}
        </Formik>
    );
};
