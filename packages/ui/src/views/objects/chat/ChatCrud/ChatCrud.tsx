import { Chat, ChatCreateInput, ChatMessage, ChatMessageSearchTreeInput, ChatMessageSearchTreeResult, ChatParticipant, chatTranslationValidation, ChatUpdateInput, chatValidation, DUMMY_ID, endpointGetChat, endpointGetChatMessageTree, endpointPostChat, endpointPutChat, exists, LINKS, noopSubmit, orDefault, Session, uuid, VALYXA_ID } from "@local/shared";
import { Box, Button, Checkbox, IconButton, InputAdornment, Stack, Typography, useTheme } from "@mui/material";
import { errorToMessage, fetchLazyWrapper, hasErrorCode, ServerResponse, socket } from "api";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { ChatBubbleTree, TypingIndicator } from "components/ChatBubbleTree/ChatBubbleTree";
import { ChatSideMenu } from "components/dialogs/ChatSideMenu/ChatSideMenu";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { RichInputBase } from "components/inputs/RichInputBase/RichInputBase";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { TranslatedRichInput } from "components/inputs/TranslatedRichInput/TranslatedRichInput";
import { TranslatedTextInput } from "components/inputs/TranslatedTextInput/TranslatedTextInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { EditableTitle } from "components/text/EditableTitle/EditableTitle";
import { SessionContext } from "contexts/SessionContext";
import { Field, Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useDeleter } from "hooks/useDeleter";
import { useHistoryState } from "hooks/useHistoryState";
import { useLazyFetch } from "hooks/useLazyFetch";
import { findTargetMessage, useMessageTree } from "hooks/useMessageTree";
import { useObjectActions } from "hooks/useObjectActions";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { useWindowSize } from "hooks/useWindowSize";
import { TFunction } from "i18next";
import { CopyIcon, ListIcon, SendIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { parseSearchParams, useLocation } from "route";
import { FormContainer, pagePaddingBottom } from "styles";
import { AssistantTask } from "types";
import { getCurrentUser } from "utils/authentication/session";
import { BranchMap, getCookieMessageTree, getCookiePartialData, setCookieMatchingChat, setCookieMessageTree } from "utils/cookies";
import { getYou } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { getObjectUrl } from "utils/navigation/openObject";
import { uuidToBase36 } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { ChatShape, shapeChat } from "utils/shape/models/chat";
import { ChatInviteShape } from "utils/shape/models/chatInvite";
import { ChatMessageShape } from "utils/shape/models/chatMessage";
import { validateFormValues } from "utils/validateFormValues";
import { ChatCrudProps, ChatFormProps } from "../types";

/** Basic chatInfo for a new convo with Valyxa */
export const VALYXA_INFO = {
    ...getCookiePartialData({ __typename: "User", id: VALYXA_ID }),
    id: VALYXA_ID,
    isBot: true,
    name: "Valyxa" as const,
} as const;

export const chatInitialValues = (
    session: Session | undefined,
    task: AssistantTask | undefined,
    t: TFunction<"common", undefined, "common">,
    language: string,
    existing?: Partial<Chat> | null | undefined,
): ChatShape => {
    const messages: ChatMessageShape[] = existing?.messages ?? [];
    // If chatting with Valyxa, add start message so that the user 
    // sees something while the chat is loading
    if (exists(existing) && messages.length === 0 && existing.invites?.length === 1 && existing.invites?.some((invite: ChatInviteShape) => invite.user.id === VALYXA_ID)) {
        const startText = t(task ?? "start", { lng: language, ns: "tasks", defaultValue: "Hello👋, I'm Valyxa! How can I assist you?" });
        messages.push({
            __typename: "ChatMessage" as const,
            id: uuid(),
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            chat: {
                __typename: "Chat" as const,
                id: existing.id ?? DUMMY_ID,
            },
            isUnsent: true,
            reactionSummaries: [],
            translations: [{
                __typename: "ChatMessageTranslation" as const,
                id: DUMMY_ID,
                language,
                text: startText,
            }],
            user: {
                __typename: "User" as const,
                id: VALYXA_ID,
                isBot: true,
                name: "Valyxa",
            },
            you: {
                __typename: "ChatMessageYou" as const,
                canDelete: false,
                canUpdate: false,
                canReply: true,
                canReport: false,
                canReact: false,
                reaction: null,
            },
        });
    }
    return {
        __typename: "Chat" as const,
        id: DUMMY_ID,
        openToAnyoneWithInvite: false,
        organization: null,
        invites: [],
        labels: [],
        // Add yourself to the participants list
        participants: [{
            __typename: "ChatParticipant" as const,
            id: uuid(),
            user: {
                ...getCookiePartialData({ __typename: "User", id: getCurrentUser(session).id! }),
                id: getCurrentUser(session).id!,
                isBot: false,
                name: getCurrentUser(session).name!,
            },
        }] as ChatParticipant[],
        participantsDelete: [],
        ...existing,
        messages,
        translations: orDefault(existing?.translations, [{
            __typename: "ChatTranslation" as const,
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            name: t("NewChat"),
            description: "",
        }]),
    };
};

export const transformChatValues = (values: ChatShape, existing: ChatShape, isCreate: boolean) =>
    isCreate ? shapeChat.create(values) : shapeChat.update(existing, values);

/**
 * Finds messages that are yours or are unsent (i.e. bot's initial message), 
 * to make sure you don't attempt to modify other people's messages
 */
export const withoutOtherMessages = (chat: ChatShape, session?: Session) => ({
    ...chat,
    messages: chat.messages?.filter(m => m.user?.id === getCurrentUser(session).id || m.isUnsent) ?? [],
});

const ChatForm = ({
    context,
    disabled,
    dirty,
    display,
    existing,
    handleUpdate,
    isOpen,
    isReadLoading,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    task,
    values,
    ...props
}: ChatFormProps) => {
    const session = useContext(SessionContext);
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    const [message, setMessage] = useHistoryState<string>("chatMessage", context ?? "");
    const [usersTyping, setUsersTyping] = useState<ChatParticipant[]>([]);

    const {
        fetch,
        fetchCreate,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<Chat, ChatCreateInput, ChatUpdateInput>({
        isCreate: false, // We create chats automatically, so this should always be false
        isMutate: true,
        endpointCreate: endpointPostChat,
        endpointUpdate: endpointPutChat,
    });

    // Create chats automatically
    const chatCreateStatus = useRef<"notStarted" | "inProgress" | "complete">("notStarted");
    useEffect(() => {
        if (isOpen === false || values.id !== DUMMY_ID || chatCreateStatus.current !== "notStarted") return;
        chatCreateStatus.current = "inProgress";
        fetchLazyWrapper<ChatCreateInput, Chat>({
            fetch: fetchCreate,
            inputs: transformChatValues(withoutOtherMessages({ ...values, ...parseSearchParams() }, session), withoutOtherMessages(existing, session), true),
            onSuccess: (data) => {
                handleUpdate(data);
                if (display === "page") setLocation(getObjectUrl(data), { replace: true });
            },
            onCompleted: () => {
                chatCreateStatus.current = "complete";
                props.setSubmitting(false);
            },
        });
    }, [display, existing, fetchCreate, handleUpdate, isOpen, props, session, setLocation, values]);
    // Reset chatCreateStatus when the dialog is closed
    useEffect(() => {
        if (isOpen || chatCreateStatus.current === "inProgress") return;
        chatCreateStatus.current = "notStarted";
    }, [isOpen]);

    // When a chat is loaded, store chat ID by participants and task
    useEffect(() => {
        const userIdsWithoutYou = existing.participants?.filter(p => p.user?.id !== getCurrentUser(session).id).map(p => p.user?.id);
        if (existing.id === DUMMY_ID || userIdsWithoutYou.length === 0) return;
        setCookieMatchingChat(existing.id, userIdsWithoutYou, task);
    }, [existing.id, existing.participants, session, task]);

    const { addMessages, clearMessages, editMessage, messagesCount, removeMessages, tree } = useMessageTree<ChatMessageShape>([]);
    const [branches, setBranches] = useState<BranchMap>(getCookieMessageTree(existing.id)?.branches ?? {});

    // We query messages separate from the chat, since we must traverse the message tree
    const [getTreeData, { data: searchTreeData, loading: isTreeLoading }] = useLazyFetch<ChatMessageSearchTreeInput, ChatMessageSearchTreeResult>(endpointGetChatMessageTree);

    // When chatId changes, clear the message tree and branches, and fetch new data
    useEffect(() => {
        if (existing.id === DUMMY_ID) return;
        clearMessages();
        setBranches({});
        console.log("getting tree dataaaaaaaaaa", existing.id);
        getTreeData({ chatId: existing.id });
    }, [existing.id, clearMessages, getTreeData]);
    useEffect(() => {
        if (!searchTreeData || searchTreeData.messages.length === 0) return;
        addMessages(searchTreeData.messages);
    }, [addMessages, searchTreeData]);

    useEffect(() => {
        // Update the cookie with current branches
        setCookieMessageTree(existing.id, { branches, locationId: "someLocationId" }); // TODO locationId should be last chat message in view
    }, [branches, existing.id]);

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isTreeLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isTreeLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useCallback((updatedChat?: ChatShape) => {
        if (disabled) {
            PubSub.get().publish("snack", { messageKey: "Unauthorized", severity: "Error" });
            return;
        }
        fetchLazyWrapper<ChatUpdateInput, Chat>({
            fetch,
            inputs: transformChatValues(withoutOtherMessages(updatedChat ?? values, session), withoutOtherMessages(existing, session), false),
            onSuccess: (data) => {
                console.log("update success!", data);
                handleUpdate(data);
                setMessage("");
            },
            onCompleted: () => { props.setSubmitting(false); },
            onError: (data) => {
                PubSub.get().publish("snack", {
                    message: errorToMessage(data as ServerResponse, getUserLanguages(session)),
                    severity: "Error",
                    data,
                });
            },
            spinnerDelay: null,
        });
    }, [disabled, existing, fetch, handleUpdate, props, session, setMessage, values]);

    // Handle websocket connection/disconnection
    useEffect(() => {
        // Only connect to the websocket if the chat exists
        if (!existing?.id || existing.id === DUMMY_ID) return;
        socket.emit("joinRoom", existing.id, (response) => {
            if (response.error) {
                console.error("Failed to join chat room", response?.error);
            } else {
                console.info("Joined chat room");
            }
        });
        // Leave the chat room when the component is unmounted
        return () => {
            socket.emit("leaveRoom", existing.id, (response) => {
                if (response.error) {
                    console.error("Failed to leave chat room", response?.error);
                } else {
                    console.info("Left chat room");
                }
            });
        };
    }, [existing.id]);
    // Handle websocket events
    useEffect(() => {
        // When a message is received, add it to the chat
        socket.on("message", (message: ChatMessage) => {
            addMessages([message]);
        });
        // When a message is updated, update it in the chat
        socket.on("editMessage", (message: ChatMessage) => {
            editMessage(message);
        });
        // When a message is deleted, remove it from the chat
        socket.on("deleteMessage", (id: string) => {
            removeMessages([id]);
        });
        // Show the status of users typing
        socket.on("typing", ({ starting, stopping }: { starting?: string[], stopping?: string[] }) => {
            // Add every user that's typing
            const newTyping = [...usersTyping];
            for (const id of starting ?? []) {
                // Never add yourself
                if (id === getCurrentUser(session).id) continue;
                if (newTyping.some(p => p.user.id === id)) continue;
                const participant = existing.participants?.find(p => p.user.id === id);
                if (!participant) continue;
                newTyping.push(participant);
            }
            // Remove every user that stopped typing
            for (const id of stopping ?? []) {
                const index = newTyping.findIndex(p => p.user.id === id);
                if (index === -1) continue;
                newTyping.splice(index, 1);
            }
            setUsersTyping(newTyping);
        });
        // TODO add participants joining/leaving, making sure to update matching chat cache in cookies
        return () => {
            // Remove chat-specific event handlers
            socket.off("message");
            socket.off("typing");
        };
    }, [addMessages, editMessage, existing.participants, handleUpdate, message, removeMessages, session, usersTyping]);

    const handleReply = useCallback((message: ChatMessageShape) => {
        // Determine if we should edit an existing message or create a new one
        const existingMessageId = findTargetMessage(tree, branches, message, existing?.participants?.length ?? 0, session);
        // If editing an existing message, send pub/sub
        if (existingMessageId) {
            PubSub.get().publish("chatMessageEdit", existingMessageId);
            return;
        }
        // Otherwise, set the message input to "@[handle_of_user_youre_replying_to] "
        else {
            PubSub.get().publish("chatMessageEdit", false);
            setMessage(existingText => `@[${message.user?.name}] ${existingText}`);
        }
    }, [branches, existing?.participants?.length, session, setMessage, tree]);

    // Handle translations
    const {
        handleAddLanguage,
        handleDeleteLanguage,
        language,
        languages,
        setLanguage,
        translationErrors,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["description", "name"],
        validationSchema: chatTranslationValidation["update"]({ env: import.meta.env.PROD ? "production" : "development" }),
    });

    const addMessage = useCallback((text: string) => {
        const newMessage: ChatMessageShape = {
            __typename: "ChatMessage" as const,
            id: uuid(),
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            isUnsent: true,
            chat: {
                __typename: "Chat" as const,
                id: existing.id,
            },
            reactionSummaries: [],
            translations: [{
                __typename: "ChatMessageTranslation" as const,
                id: DUMMY_ID,
                language,
                text,
            }],
            user: {
                __typename: "User" as const,
                id: getCurrentUser(session).id ?? "",
                isBot: false,
                name: getCurrentUser(session).name ?? undefined,
            },
        };
        onSubmit({ ...values, messages: [...(values.messages ?? []), newMessage] });
    }, [existing.id, language, onSubmit, session, values]);

    const url = useMemo(() => `${window.location.origin}/chat/${uuidToBase36(values.id)}`, [values.id]);
    const copyLink = useCallback(() => {
        navigator.clipboard.writeText(url);
        PubSub.get().publish("snack", { messageKey: "CopiedToClipboard", severity: "Success" });
    }, [url]);

    const openSideMenu = useCallback(() => { PubSub.get().publish("sideMenu", { id: "chat-side-menu", idPrefix: task, isOpen: true }); }, [task]);
    const closeSideMenu = useCallback(() => { PubSub.get().publish("sideMenu", { id: "chat-side-menu", idPrefix: task, isOpen: false }); }, [task]);
    useEffect(() => {
        return () => {
            closeSideMenu();
        };
    }, [closeSideMenu]);

    const actionData = useObjectActions({
        object: existing,
        objectType: "Chat",
        setLocation,
        setObject: handleUpdate,
    });

    const {
        handleDelete,
        DeleteDialogComponent,
    } = useDeleter({
        object: existing,
        objectType: "Chat",
        onActionComplete: actionData.onActionComplete,
    });

    const [inputFocused, setInputFocused] = useState(false);
    const onFocus = useCallback(() => { setInputFocused(true); }, []);
    const onBlur = useCallback(() => { setInputFocused(false); }, []);

    const showBotWarning = useMemo(() => !disabled && usersTyping.length === 0 && messagesCount <= 2 && existing.participants?.some(i => i?.user?.isBot), [disabled, existing.participants, messagesCount, usersTyping]);

    return (
        <>
            {existing?.id && DeleteDialogComponent}
            <MaybeLargeDialog
                display={display}
                id="chat-dialog"
                isOpen={isOpen}
                onClose={onClose}
                sxs={{
                    paper: { minWidth: "100vw" },
                }}
            >
                <Box sx={{
                    display: "flex",
                    flexDirection: "column",
                    overflow: "hidden",
                    ...(display === "page" && {
                        maxHeight: "100vh",
                        height: "100vh",
                    }),
                }}>
                    <TopBar
                        display={display}
                        onClose={onClose}
                        startComponent={<IconButton
                            aria-label="Open chat menu"
                            onClick={openSideMenu}
                            sx={{
                                width: "48px",
                                height: "48px",
                                marginLeft: 1,
                                marginRight: 1,
                                cursor: "pointer",
                            }}
                        >
                            <ListIcon fill={palette.primary.contrastText} width="100%" height="100%" />
                        </IconButton>}
                        titleComponent={<EditableTitle
                            handleDelete={handleDelete}
                            isDeletable={!(values.id === DUMMY_ID || disabled)}
                            isEditable={!disabled}
                            language={language}
                            onClose={onSubmit}
                            onSubmit={onSubmit}
                            titleField="name"
                            subtitleField="description"
                            variant="header"
                            sxs={{ stack: { padding: 0 } }}
                            DialogContentForm={() => (
                                <>
                                    <BaseForm
                                        display="dialog"
                                        maxWidth={600}
                                        style={{
                                            paddingBottom: "16px",
                                            width: "600px",
                                        }}
                                    >
                                        <FormContainer>
                                            {/* TODO relationshiplist should be for connecting organization and inviting users. Can invite non-members even if organization specified, but the search should show members first */}
                                            <RelationshipList
                                                isEditing={true}
                                                objectType={"Chat"}
                                                sx={{ marginBottom: 4 }}
                                            />
                                            <TranslatedTextInput
                                                fullWidth
                                                label={t("Name")}
                                                language={language}
                                                name="name"
                                            />
                                            <TranslatedRichInput
                                                language={language}
                                                maxChars={2048}
                                                minRows={4}
                                                name="description"
                                                placeholder={t("Description")}
                                            />
                                            <LanguageInput
                                                currentLanguage={language}
                                                handleAdd={handleAddLanguage}
                                                handleDelete={handleDeleteLanguage}
                                                handleCurrent={setLanguage}
                                                languages={languages}
                                            />
                                            {/* Invite link */}
                                            <Stack direction="column" spacing={1}>
                                                <Stack direction="row" sx={{ alignItems: "center" }}>
                                                    <Typography variant="h6">{t(`OpenToAnyoneWithLink${values.openToAnyoneWithInvite ? "True" : "False"}`)}</Typography>
                                                    <HelpButton markdown={t("OpenToAnyoneWithLinkDescription")} />
                                                </Stack>
                                                <Stack direction="row" spacing={0}>
                                                    <Field
                                                        name="openToAnyoneWithInvite"
                                                        type="checkbox"
                                                        as={Checkbox}
                                                        sx={{
                                                            "&.MuiCheckbox-root": {
                                                                color: palette.secondary.main,
                                                            },
                                                        }}
                                                    />
                                                    {/* Show link with copy adornment*/}
                                                    <TextInput
                                                        disabled
                                                        fullWidth
                                                        id="invite-link"
                                                        label={t("InviteLink")}
                                                        variant="outlined"
                                                        value={url}
                                                        InputProps={{
                                                            endAdornment: (
                                                                <InputAdornment position="end">
                                                                    <IconButton
                                                                        aria-label={t("Copy")}
                                                                        onClick={copyLink}
                                                                        size="small"
                                                                    >
                                                                        <CopyIcon />
                                                                    </IconButton>
                                                                </InputAdornment>
                                                            ),
                                                        }}
                                                    />
                                                </Stack>
                                            </Stack>
                                        </FormContainer>
                                    </BaseForm>
                                </>
                            )}
                        />}
                    />
                    <Box sx={{
                        display: "flex",
                        flexDirection: "column",
                        flexGrow: 1,
                        margin: "auto",
                        overflowY: "auto",
                        width: "min(700px, 100%)",
                    }}>
                        <ChatBubbleTree
                            branches={branches}
                            editMessage={editMessage}
                            handleReply={handleReply}
                            handleRetry={() => { }}
                            removeMessages={removeMessages}
                            setBranches={setBranches}
                            tree={tree}
                        />
                    </Box>
                    {!showBotWarning && <TypingIndicator participants={usersTyping} />}
                    {/* Warning that you are talking to a bot */}
                    {showBotWarning &&
                        <Box display="flex" flexDirection="row" justifyContent="center" margin="auto" gap={0} p={1}>
                            <Typography variant="body2" sx={{ margin: "auto", maxWidth: "min(700px, 100%)" }}>{t("BotChatWarning")}</Typography>
                            <Button variant="text" sx={{ margin: "auto", textTransform: "none" }} onClick={() => {
                                PubSub.get().publish("alertDialog", {
                                    messageKey: "BotChatWarningDetails",
                                    buttons: [
                                        { labelKey: "Ok" },
                                    ],
                                });
                            }}>{t("LearnMore")}</Button>
                        </Box>
                    }
                    <RichInputBase
                        actionButtons={[{
                            Icon: SendIcon,
                            onClick: () => {
                                if (!existing) {
                                    PubSub.get().publish("snack", { message: "Chat not found", severity: "Error" });
                                    return;
                                }
                                if (message.trim() === "") return;
                                addMessage(message);
                            },
                        }]}
                        disabled={!existing}
                        disableAssistant={true}
                        fullWidth
                        getTaggableItems={async (searchString) => {
                            // Find all users in the chat, plus @Everyone
                            let users = [
                                //TODO handle @Everyone
                                ...(existing?.participants?.map(p => p.user) ?? []),
                            ];
                            // Filter out current user
                            users = users.filter(p => p.id !== getCurrentUser(session).id);
                            // Filter out users that don't match the search string
                            users = users.filter(p => p.name.toLowerCase().includes(searchString.toLowerCase()));
                            console.log("got taggable items", users, searchString);
                            return users;
                        }}
                        maxChars={1500}
                        maxRows={inputFocused ? 10 : 2}
                        minRows={1}
                        onBlur={onBlur}
                        onChange={setMessage}
                        onFocus={onFocus}
                        name="newMessage"
                        sxs={{
                            root: {
                                background: palette.primary.dark,
                                color: palette.primary.contrastText,
                                maxHeight: "min(75vh, 500px)",
                                width: "min(700px, 100%)",
                                margin: "auto",
                                marginBottom: { xs: display === "page" ? pagePaddingBottom : "0", md: "0" },
                            },
                            topBar: { borderRadius: 0, paddingLeft: isMobile ? "20px" : 0, paddingRight: isMobile ? "20px" : 0 },
                            bottomBar: { paddingLeft: isMobile ? "20px" : 0, paddingRight: isMobile ? "20px" : 0 },
                            textArea: {
                                border: "none",
                                background: palette.background.paper,
                            },
                        }}
                        value={message}
                    />
                </Box>
            </MaybeLargeDialog>
            <ChatSideMenu idPrefix={task} />
        </>
    );
};

export const ChatCrud = ({
    display,
    isCreate,
    isOpen,
    overrideObject,
    task,
    ...props
}: ChatCrudProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { isLoading: isReadLoading, object: existing, setObject: setExisting } = useObjectFromUrl<Chat, ChatShape>({
        ...endpointGetChat,
        onError: (errors) => {
            // If the chat doesn't exist, switch to create mode
            if (hasErrorCode({ errors }, "NotFound")) {
                if (display === "page") setLocation(`${LINKS.Chat}/add`, { replace: true, searchParams: parseSearchParams() });
                setExisting(chatInitialValues(session, task, t, getUserLanguages(session)[0]));
            }
        },
        displayError: display === "page" || isOpen === true, // Suppress errors from closed dialogs
        isCreate,
        objectType: "Chat",
        overrideObject: overrideObject as unknown as Chat,
        transform: (data) => chatInitialValues(session, task, t, getUserLanguages(session)[0], data),
    });
    const { canUpdate } = useMemo(() => getYou(existing), [existing]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateFormValues(values, existing, isCreate, transformChatValues, chatValidation)}
        >
            {(formik) =>
                <>
                    <ChatForm
                        disabled={!(isCreate || canUpdate)}
                        display={display}
                        existing={existing}
                        handleUpdate={setExisting}
                        isCreate={isCreate}
                        isReadLoading={isReadLoading}
                        isOpen={isOpen}
                        task={task}
                        {...props}
                        {...formik}
                    />
                </>
            }
        </Formik>
    );
};
