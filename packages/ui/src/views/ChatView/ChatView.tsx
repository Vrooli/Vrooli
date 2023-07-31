import { Chat, ChatCreateInput, ChatMessage, DUMMY_ID, endpointGetChat, endpointPostChat, FindByIdInput, LINKS, orDefault, uuid, uuidValidate, VALYXA_ID } from "@local/shared";
import { Box, Stack, useTheme } from "@mui/material";
import { fetchLazyWrapper, socket } from "api";
import { ChatBubble } from "components/ChatBubble/ChatBubble";
import { MarkdownInput } from "components/inputs/MarkdownInput/MarkdownInput";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { AddIcon } from "icons";
import { useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { getCurrentUser } from "utils/authentication/session";
import { getDisplay } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { getUserLanguages } from "utils/display/translationTools";
import { useDisplayServerError } from "utils/hooks/useDisplayServerError";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { base36ToUuid, tryOnClose } from "utils/navigation/urlTools";
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
        // Check if the chat already exists, or if the URL has an id
        let chatId = chatInfo?.id;
        if (!chatId && window.location.pathname.startsWith(LINKS.Chat)) {
            chatId = base36ToUuid(window.location.pathname.split("/")[2]);
        }
        const alreadyExists = chatId && uuidValidate(chatId);
        // If so, find chat by id
        if (alreadyExists) {
            console.log("getting chat", chatInfo);
            fetchLazyWrapper<FindByIdInput, Chat>({
                fetch: getData,
                inputs: { id: chatId as string },
                onSuccess: (data) => {
                    console.log("GOT CHAT!!!", data);
                    setChat(data);
                },
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
                    translations: orDefault(chatInfo?.translations, [{
                        __typename: "ChatTranslation" as const,
                        id: uuid(),
                        language: lng,
                        name: chatInfo?.participants?.length === 1 ? firstString(chatInfo.participants[0].user?.name) : "New Chat",
                    }]),
                }),
                onSuccess: (data) => { setChat(data); },
            });
        }
    }, [chat, chatInfo, createChat, getData, lng]);

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
        socket.on("message", (message: ChatMessage) => {
            // Add message to chat if it's not already there. 
            // Make sure it is inserted in the correct order, using the created_at field.
            // Find index to insert message at
            setChat(c => ({
                ...c,
                messages: updateArray(
                    c!.messages ?? [],
                    c!.messages.findIndex(m => m.created_at > message.created_at),
                    message,
                ),
            } as Chat));
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
            setMessages(chat.messages.sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime()));
        }
    }, [chat]);
    useEffect(() => {
        // If chatting with default AI assistant, add start message so that the user 
        // sees something while the chat is loading
        if (chat && chat.messages.length === 0 && chat.invites?.length === 1 && chat.invites?.some((invite: Chat["invites"][0]) => invite.user.id === VALYXA_ID)) {
            const startText = t(task ?? "start", { lng, ns: "tasks", defaultValue: "Hello👋, I'm Valyxa! How can I assist you?" });
            setMessages([{
                __typename: "ChatMessage" as const,
                id: uuid(),
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                chat: {
                    __typename: "Chat" as const,
                    id: chat.id,
                } as any,
                translations: [{
                    __typename: "ChatMessageTranslation" as const,
                    id: DUMMY_ID,
                    language: lng,
                    text: startText,
                }],
                user: {
                    __typename: "User" as const,
                    id: "4b038f3b-f1f7-1f9b-8f4b-cff4b8f9b20f",
                    isBot: true,
                    name: "Valyxa",
                } as any,
                you: {
                    __typename: "ChatMessageYou" as const,
                    canDelete: false,
                    canUpdate: false,
                    canReply: true,
                    canReport: false,
                    canReact: false,
                    reaction: null,
                },
            }] as any);
        }
    }, [chat, lng, t, task]);

    console.log("context in chatview", messages, chat?.messages);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={{
                editingMessage: "",
                newMessage: context ?? "",
            }}
            onSubmit={(values, helpers) => {
                if (!chat) return;
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
                        id: uuid(),
                        created_at: new Date().toISOString(),
                        updated_at: new Date().toISOString(),
                        isUnsent: true,
                        chat: {
                            __typename: "Chat" as const,
                            id: chat.id,
                        } as any,
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
                    console.log("creating message 0", newMessage);
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
                                    { labelKey: "Yes", onClick: () => { tryOnClose(onClose, setLocation); } },
                                    { labelKey: "No" },
                                ],
                            });
                        } else {
                            tryOnClose(onClose, setLocation);
                        }
                    }}
                    // TODO change title so that when pressed, you can switch chats or add a new chat
                    title={firstString(title, botSettings ? "AI Chat" : "Chat")}
                    zIndex={zIndex}
                />
                {/* TODO add ChatSideMenu component */}
                <Stack direction="column" spacing={4}>
                    <Box sx={{ overflowY: "auto", maxHeight: "calc(100vh - 64px)" }}>
                        {messages.map((message: ChatMessage, index) => {
                            const isOwn = message.you.canUpdate || message.user?.id === getCurrentUser(session).id;
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
                            getTaggableItems={async (searchString) => {
                                // Find all users in the chat, plus @Everyone
                                let users = [
                                    //TODO handle @Everyone
                                    ...(chat?.participants?.map(p => p.user) ?? []),
                                ];
                                // Filter out current user
                                users = users.filter(p => p.id !== getCurrentUser(session).id);
                                // Filter out users that don't match the search string
                                users = users.filter(p => p.name.toLowerCase().includes(searchString.toLowerCase()));
                                return users;
                            }}
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
