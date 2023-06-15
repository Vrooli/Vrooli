import { AddIcon, Chat, ChatCreateInput, ChatMessage, ChatMessageCreateInput, DUMMY_ID, endpointGetChat, endpointPostChat, endpointPostChatMessage, FindByIdInput, useLocation, uuid, uuidValidate, VALYXA_ID } from "@local/shared";
import { Box, Stack, useTheme } from "@mui/material";
import { fetchLazyWrapper, socket } from "api";
import { ChatBubble } from "components/ChatBubble/ChatBubble";
import { MarkdownInput } from "components/inputs/MarkdownInput/MarkdownInput";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session";
import { getDisplay } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { getUserLanguages } from "utils/display/translationTools";
import { useDisplayServerError } from "utils/hooks/useDisplayServerError";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { updateArray } from "utils/shape/general";
import { shapeChat } from "utils/shape/models/chat";
import { ChatViewProps } from "views/types";

export const ChatView = ({
    botSettings,
    chatInfo,
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

    const [getData, { loading: isFindLoading, errors: findErrors }] = useLazyFetch<FindByIdInput, Chat>(endpointGetChat);
    const [createChat, { loading: isCreateLoading, errors: createErrors }] = useLazyFetch<ChatCreateInput, Chat>(endpointPostChat);
    const [chat, setChat] = useState<Chat>();
    useDisplayServerError(findErrors ?? createErrors);
    useEffect(() => {
        if (chat) return;
        // Check if the chat already exists
        const alreadyExists = chatInfo?.id && uuidValidate(chatInfo.id);
        // If so, find chat by id
        if (alreadyExists) {
            console.log("getting chat", chatInfo);
            fetchLazyWrapper<FindByIdInput, Chat>({
                fetch: getData,
                inputs: { id: chatInfo.id! },
                onSuccess: (data) => { setChat(data); },
            });
        }
        // Otherwise, start a new chat
        else {
            fetchLazyWrapper<ChatCreateInput, Chat>({
                fetch: createChat,
                inputs: shapeChat.create({
                    ...chatInfo,
                    id: uuid(),
                    openToAnyoneWithInvite: chatInfo?.openToAnyoneWithInvite ?? false,
                }),
                onSuccess: (data) => { setChat(data); },
            });
        }
    }, [chat, chatInfo, createChat, getData]);

    // Handle websocket for chat messages (e.g. new message, new reactions, etc.)
    useEffect(() => {
        // Only connect to the websocket if the chat exists
        if (!chat?.id) return;
        socket.emit("joinRoom", chat.id, (response) => {
            if (response.error) {
                // handle error
                console.error(response.error);
            } else {
                console.log("Joined chat room");
            }
        });

        // Define chat-specific event handlers
        socket.on("message", (message) => {
            // handle incoming chat message
            console.log(message);
        });

        // Leave the chat room when the component is unmounted
        return () => {
            socket.emit("leaveRoom", chat.id, (response) => {
                if (response.error) {
                    // handle error
                    console.error(response.error);
                } else {
                    console.log("Left chat room");
                }
            });

            // Remove chat-specific event handlers
            socket.off("message");
        };
    }, [chat?.id]);

    const { title, subtitle } = useMemo(() => getDisplay(chat, getUserLanguages(session)), [chat, session]);

    const [messages, setMessages] = useState<(ChatMessage & { isUnsent?: boolean })[]>([]);
    useEffect(() => {
        if (chat) {
            // If chatting with default AI assistant, keep start message in chat.
            const chattingWithValyxa = chatInfo.invites?.some((invite: Chat["invites"][0]) => invite.user.id === VALYXA_ID);
            setMessages(m => chattingWithValyxa && m.length === 1 ? [m[0], ...chat.messages] : chat.messages);
        }
    }, [chat]);
    useEffect(() => {
        // If chatting with default AI assistant, add start message so that we don't need 
        // to query the server for it.
        if (chatInfo.invites?.some((invite: Chat["invites"][0]) => invite.user.id === VALYXA_ID)) {
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
    }, [chatInfo, lng, t, task]);

    const [createMessage, { loading: isCreateMessageLoading }] = useLazyFetch<ChatMessageCreateInput, ChatMessage>(endpointPostChatMessage);
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
                    const text = values.newMessage.trim();
                    if (text.length === 0) return;
                    // for now, just add the message to the list
                    const newMessage: ChatMessage & { isUnsent?: boolean } = {
                        __typename: "ChatMessage" as const,
                        created_at: new Date().toISOString(),
                        updated_at: new Date().toISOString(),
                        isUnsent: true,
                        translations: [{
                            __typename: "ChatMessageTranslation" as const,
                            id: DUMMY_ID,
                            language: lng,
                            text,
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
                            return <ChatBubble
                                key={index}
                                message={message}
                                index={index}
                                isOwn={isOwn}
                                onUpdated={(updatedMessage) => {
                                    setMessages(updateArray(messages,
                                        messages.findIndex(m => m.id === updatedMessage.id),
                                        updatedMessage,
                                    ));
                                }}
                                zIndex={zIndex}
                            />;
                        })}
                    </Box>
                    <Box sx={{
                        background: palette.primary.dark,
                        color: palette.primary.contrastText,
                    }}>
                        <MarkdownInput
                            actionButtons={[{
                                Icon: AddIcon,
                                onClick: () => {
                                    if (!chat) {
                                        PubSub.get().publishSnack({ message: "Chat not found", severity: "Error" });
                                        return;
                                    }
                                    formik.handleSubmit();
                                },
                            }]}
                            disabled={!chat}
                            disableAssistant={true}
                            fullWidth
                            maxChars={1500}
                            minRows={4}
                            maxRows={15}
                            name="newMessage"
                            sxs={{
                                bar: { borderRadius: 0 },
                                textArea: { paddingRight: 4, border: "none" },
                            }}
                            zIndex={zIndex}
                        />
                    </Box>
                </Stack>
            </>}
        </Formik >
    );
};
