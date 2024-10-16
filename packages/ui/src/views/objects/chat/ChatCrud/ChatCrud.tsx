import { Chat, ChatCreateInput, ChatInviteShape, ChatInviteStatus, ChatMessageShape, ChatParticipantShape, ChatShape, chatTranslationValidation, ChatUpdateInput, chatValidation, DUMMY_ID, endpointGetChat, endpointPostChat, endpointPutChat, exists, getObjectUrl, LINKS, noopSubmit, orDefault, parseSearchParams, Session, shapeChat, uuid, uuidToBase36, VALYXA_ID } from "@local/shared";
import { Box, Button, Checkbox, IconButton, InputAdornment, Stack, styled, Typography } from "@mui/material";
import { errorToMessage, hasErrorCode } from "api/errorParser";
import { fetchLazyWrapper } from "api/fetchWrapper";
import { ServerResponse } from "api/types";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { ChatBubbleTree } from "components/ChatBubbleTree/ChatBubbleTree";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { ChatMessageInput } from "components/inputs/ChatMessageInput/ChatMessageInput";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TranslatedRichInput } from "components/inputs/RichInput/RichInput";
import { TextInput, TranslatedTextInput } from "components/inputs/TextInput/TextInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { EditableTitle } from "components/text/EditableTitle/EditableTitle";
import { SessionContext } from "contexts";
import { Field, Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useMessageActions, useMessageInput, useMessageTree } from "hooks/messages";
import { useDeleter, useObjectActions } from "hooks/objectActions";
import { useChatTasks } from "hooks/tasks";
import { useHistoryState } from "hooks/useHistoryState";
import { useManagedObject } from "hooks/useManagedObject";
import { useSocketChat } from "hooks/useSocketChat";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { TFunction } from "i18next";
import { AddIcon, CopyIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { FormContainer, FormSection, ScrollBox } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { getUserLanguages } from "utils/display/translationTools";
import { getCookiePartialData, setCookieMatchingChat } from "utils/localStorage";
import { PubSub } from "utils/pubsub";
import { validateFormValues } from "utils/validateFormValues";
import { ChatCrudProps, ChatFormProps } from "../types";

/** Basic chatInfo for a new convo with Valyxa */
export const VALYXA_INFO = {
    ...getCookiePartialData({ __typename: "User", id: VALYXA_ID }),
    id: VALYXA_ID,
    isBot: true,
    name: "Valyxa" as const,
} as const;

export const CHAT_DEFAULTS = {
    __typename: "Chat" as const,
    id: DUMMY_ID,
    openToAnyoneWithInvite: false,
    invites: [{
        __typename: "ChatInvite" as const,
        id: DUMMY_ID,
        status: ChatInviteStatus.Pending,
        user: VALYXA_INFO,
    }],
} as unknown as Chat;

const MESSAGE_LIST_ID = "chatMessage";

export function chatInitialValues(
    session: Session | undefined,
    t: TFunction<"common", undefined, "common">,
    language: string,
    existing?: Partial<Chat> | null | undefined,
): ChatShape {
    const messages: ChatMessageShape[] = (existing?.messages ?? []).map(m => ({ ...m, status: "sent" }));
    // If chatting with Valyxa, add start message so that the user 
    // sees something while the chat is loading
    if (exists(existing) && messages.length === 0 && existing.invites?.length === 1 && existing.invites?.some((invite: ChatInviteShape) => invite.user.id === VALYXA_ID)) {
        const startText = t("start", { lng: language, ns: "tasks", defaultValue: "Hello👋 How can I assist you?" });
        messages.push({
            __typename: "ChatMessage" as const,
            id: uuid(),
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            chat: {
                __typename: "Chat" as const,
                id: existing.id ?? DUMMY_ID,
            },
            status: "unsent",
            versionIndex: 0,
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
    const currentUser = getCurrentUser(session);
    return {
        __typename: "Chat" as const,
        id: DUMMY_ID,
        openToAnyoneWithInvite: false,
        team: null,
        invites: [],
        labels: [],
        // Add yourself to the participants list
        participants: (currentUser.id ? [{
            __typename: "ChatParticipant" as const,
            id: uuid(),
            user: {
                ...getCookiePartialData({ __typename: "User", id: currentUser.id }),
                __typename: "User" as const,
                id: currentUser.id,
                isBot: false,
                name: currentUser.id,
            },
        }] : []) as ChatParticipantShape[],
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
}

export function transformChatValues(values: ChatShape, existing: ChatShape, isCreate: boolean) {
    return isCreate ? shapeChat.create(values) : shapeChat.update(existing, values);
}

/**
 * Finds messages that are yours or are unsent (i.e. bot's initial message), 
 * to make sure you don't attempt to modify other people's messages
 */
export function withModifiableMessages(chat: ChatShape, session?: Session) {
    return {
        ...chat,
        messages: chat.messages?.filter(m => m.user?.id === getCurrentUser(session).id || m.status === "unsent") ?? [],
    };
}

/**
 * Finds messages that are only yours
 */
export function withYourMessages(chat: ChatShape, session?: Session) {
    return {
        ...chat,
        messages: chat.messages?.filter(m => m.user?.id === getCurrentUser(session).id) ?? [],
    };
}

const InviteCheckboxField = styled(Field)(({ theme }) => ({
    "&.MuiCheckbox-root": {
        color: theme.palette.secondary.main,
    },
}));

const ChatTreeContainer = styled(Box)(() => ({
    position: "relative",
    display: "flex",
    flexDirection: "column",
    flexGrow: 1,
    margin: "auto",
    overflowY: "auto",
    width: "min(700px, 100%)",
}));

const dialogStyle = {
    paper: { minWidth: "100vw" },
} as const;
const editableTitleStyle = { stack: { padding: 0 } } as const;
const addChatButtonStyle = { margin: 1, borderRadius: 8, padding: "4px 8px" } as const;
const relationshipListStyle = { marginBottom: 4 } as const;
const titleDialogFormStyle = {
    padding: "16px",
    width: "600px",
} as const;

function ChatForm({
    disabled,
    display,
    existing,
    handleUpdate,
    isOpen,
    isReadLoading,
    onClose,
    values,
    ...props
}: ChatFormProps) {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

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
    useEffect(function createNewChatEffecct() {
        if (isOpen !== true || values.id !== DUMMY_ID || chatCreateStatus.current !== "notStarted") return;
        chatCreateStatus.current = "inProgress";
        // Search params are often used to set chat name, but might not include some fields we need to make the yup validation happy
        const withSearchParams = { ...values, ...parseSearchParams() };
        withSearchParams.translations = withSearchParams.translations?.map(t => ({ ...t, id: t.id ?? DUMMY_ID })) ?? [];
        fetchLazyWrapper<ChatCreateInput, Chat>({
            fetch: fetchCreate,
            inputs: transformChatValues(withModifiableMessages(withSearchParams, session), withYourMessages(existing, session), true),
            onSuccess: (data) => {
                handleUpdate({ ...data, messages: [] });
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

    useEffect(() => {
        setParticipants(existing.participants ?? []);
    }, [existing.participants]);
    // When a chat is loaded, store chat ID by participants and task
    useEffect(() => {
        const userIds = existing.participants?.map(p => p.user?.id) ?? [];
        if (existing.id === DUMMY_ID || userIds.length === 0) return;
        setCookieMatchingChat(existing.id, userIds);
    }, [existing.id, existing.participants, session]);

    const messageTree = useMessageTree(existing.id);

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || messageTree.isTreeLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, messageTree.isTreeLoading, isUpdateLoading, props.isSubmitting]);

    // Handle translations
    const {
        handleAddLanguage,
        handleDeleteLanguage,
        language,
        languages,
        setLanguage,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        validationSchema: chatTranslationValidation["update"]({ env: process.env.NODE_ENV }),
    });

    const [participants, setParticipants] = useState<Omit<ChatParticipantShape, "chat">[]>([]);
    const [usersTyping, setUsersTyping] = useState<Omit<ChatParticipantShape, "chat">[]>([]);

    const [message, setMessage] = useHistoryState<string>(`${MESSAGE_LIST_ID}-message`, "");
    const taskInfo = useChatTasks({ chatId: existing.id });
    const isBotOnlyChat = participants?.every(p => p.user?.isBot || p.user?.id === getCurrentUser(session).id) ?? false;
    const messageActions = useMessageActions({
        activeTask: taskInfo.activeTask,
        addMessages: messageTree.addMessages,
        chat: values,
        contexts: taskInfo.contexts[taskInfo.activeTask.taskId] || [],
        editMessage: messageTree.editMessage,
        isBotOnlyChat,
        language,
        setMessage,
    });
    const messageInput = useMessageInput({
        id: MESSAGE_LIST_ID,
        languages,
        message,
        postMessage: messageActions.postMessage,
        putMessage: messageActions.putMessage,
        replyToMessage: messageActions.replyToMessage,
        setMessage,
    });
    const { messageStream } = useSocketChat({
        addMessages: messageTree.addMessages,
        chat: existing,
        editMessage: messageTree.editMessage,
        participants,
        removeMessages: messageTree.removeMessages,
        setParticipants,
        setUsersTyping,
        usersTyping,
    });

    const newChat = useCallback(function newChatCallback() {
        if (isLoading || existing.id === DUMMY_ID) return;
        // Create a new chat with the same participants
        fetchLazyWrapper<ChatCreateInput, Chat>({
            fetch: fetchCreate,
            inputs: {
                id: uuid(),
                invitesCreate: existing.participants?.map(p => ({
                    id: uuid(),
                    chatConnect: existing.id,
                    userConnect: p.user.id,
                })),
                translationsCreate: existing.translations?.map(t => ({
                    ...t,
                    id: uuid(),
                })) ?? [],
            },
            onSuccess: (data) => {
                handleUpdate({ ...data, messages: [] });
                if (display === "page") setLocation(getObjectUrl(data), { replace: true });
                setUsersTyping([]);
                //TODO simple update would be ideal, but there is a bug where it switches back to the original chat and messes up message retrieval
                window.location.reload();
            },
            onCompleted: () => { props.setSubmitting(false); },
        });
    }, [display, existing.id, existing.participants, existing.translations, fetchCreate, handleUpdate, isLoading, props, setLocation]);

    const onSubmit = useCallback(function onSubmitCallback(updatedChat?: ChatShape) {
        return new Promise<Chat>((resolve, reject) => {
            if (disabled) {
                PubSub.get().publish("snack", { messageKey: "Unauthorized", severity: "Error" });
                reject();
                return;
            }
            // Clear typed message
            let oldMessage: string | undefined;
            setMessage((m) => {
                oldMessage = m;
                return "";
            });
            fetchLazyWrapper<ChatUpdateInput, Chat>({
                fetch,
                inputs: transformChatValues(withModifiableMessages(updatedChat ?? values, session), withYourMessages(existing, session), false),
                onSuccess: (data) => {
                    handleUpdate({ ...data, messages: [] });
                    resolve(data);
                },
                onCompleted: () => { props.setSubmitting(false); },
                onError: (data) => {
                    // Put typed message back if there was a problem
                    setMessage(oldMessage ?? "");
                    PubSub.get().publish("snack", {
                        message: errorToMessage(data as ServerResponse, getUserLanguages(session)),
                        severity: "Error",
                        data,
                    });
                    reject(data);
                },
                spinnerDelay: null,
            });
        });
    }, [disabled, existing, fetch, handleUpdate, props, session, setMessage, values]);

    const url = useMemo(() => `${window.location.origin}/chat/${uuidToBase36(values.id)}`, [values.id]);
    const copyLink = useCallback(() => {
        navigator.clipboard.writeText(url);
        PubSub.get().publish("snack", { messageKey: "CopiedToClipboard", severity: "Success" });
    }, [url]);

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

    const outerBoxStyle = useMemo(function outerBoxStyleMemo() {
        return {
            display: "flex",
            flexDirection: "column",
            overflow: "hidden",
            ...(display === "page" && {
                maxHeight: "calc(100vh - env(safe-area-inset-bottom))",
                height: "calc(100vh - env(safe-area-inset-bottom))",
            }),
        } as const;
    }, [display]);

    const copyInviteLinkInputProps = useMemo(function copyInviteLinkInputPropsMemo() {
        return {
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
        };
    }, [copyLink, t]);

    const titleDialogContentForm = useCallback(function titleDialogContentFormCallback() {
        return (
            <ScrollBox>
                <BaseForm
                    display="dialog"
                    isNested={display === "dialog"}
                    maxWidth={600}
                    style={titleDialogFormStyle}
                >
                    <FormContainer>
                        {/* TODO relationshiplist should be for connecting team and inviting users. Can invite non-members even if team specified, but the search should show members first */}
                        <RelationshipList
                            isEditing={true}
                            objectType={"Chat"}
                            sx={relationshipListStyle}
                        />
                        <FormSection>
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
                        </FormSection>
                        <FormSection variant="transparent">
                            <Stack direction="row" alignItems="center">
                                <Typography variant="h6">{t(`OpenToAnyoneWithLink${values.openToAnyoneWithInvite ? "True" : "False"}`)}</Typography>
                                <HelpButton markdown={t("OpenToAnyoneWithLinkDescription")} />
                            </Stack>
                            <Stack direction="row" spacing={0}>
                                <InviteCheckboxField
                                    name="openToAnyoneWithInvite"
                                    type="checkbox"
                                    as={Checkbox}
                                />
                                <TextInput
                                    disabled
                                    fullWidth
                                    id="invite-link"
                                    label={t("InviteLink")}
                                    variant="outlined"
                                    value={url}
                                    InputProps={copyInviteLinkInputProps}
                                />
                            </Stack>
                        </FormSection>
                    </FormContainer>
                </BaseForm>
            </ScrollBox>
        );
    }, [display, t, language, handleAddLanguage, handleDeleteLanguage, setLanguage, languages, values.openToAnyoneWithInvite, url, copyInviteLinkInputProps]);

    return (
        <>
            {existing?.id && DeleteDialogComponent}
            <MaybeLargeDialog
                display={display}
                id="chat-dialog"
                isOpen={isOpen}
                onClose={onClose}
                sxs={dialogStyle}
            >
                <Box sx={outerBoxStyle}>
                    <TopBar
                        display={display}
                        onClose={onClose}
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
                            sxs={editableTitleStyle}
                            DialogContentForm={titleDialogContentForm}
                        />}
                        below={existing.id !== DUMMY_ID && isBotOnlyChat && !isLoading && <Box
                            display="flex"
                            flexDirection="row"
                            justifyContent="space-around"
                            alignItems="center"
                            maxWidth="min(100vw, 1000px)"
                            margin="auto"
                        >
                            <Button
                                color="primary"
                                onClick={newChat}
                                variant="contained"
                                sx={addChatButtonStyle}
                                startIcon={<AddIcon />}
                            >
                                {t("NewChat")}
                            </Button>
                        </Box>}
                    />
                    <ChatTreeContainer>
                        <ChatBubbleTree
                            branches={messageTree.branches}
                            handleEdit={messageInput.startEditingMessage}
                            handleRegenerateResponse={messageActions.regenerateResponse}
                            handleReply={messageInput.startReplyingToMessage}
                            handleRetry={messageActions.retryPostMessage}
                            isBotOnlyChat={isBotOnlyChat}
                            isEditingMessage={Boolean(messageInput.messageBeingEdited)}
                            isReplyingToMessage={Boolean(messageInput.messageBeingRepliedTo)}
                            messageStream={messageStream}
                            removeMessages={messageTree.removeMessages}
                            setBranches={messageTree.setBranches}
                            tree={messageTree.tree}
                        />
                    </ChatTreeContainer>
                    <ChatMessageInput
                        disabled={!existing}
                        display={display}
                        isLoading={isLoading}
                        message={message}
                        messageBeingEdited={messageInput.messageBeingEdited}
                        messageBeingRepliedTo={messageInput.messageBeingRepliedTo}
                        participantsAll={participants}
                        participantsTyping={usersTyping}
                        messagesCount={messageTree.messagesCount}
                        placeholder={t("MessagePlaceholder")}
                        setMessage={setMessage}
                        stopEditingMessage={messageInput.stopEditingMessage}
                        stopReplyingToMessage={messageInput.stopReplyingToMessage}
                        submitMessage={messageInput.submitMessage}
                        taskInfo={taskInfo}
                    />
                </Box>
            </MaybeLargeDialog>
        </>
    );
}

export function ChatCrud({
    display,
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: ChatCrudProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<Chat, ChatShape>({
        ...endpointGetChat,
        onError: function onLoadError(errors) {
            // If the chat doesn't exist, switch to create mode
            if (hasErrorCode({ errors }, "NotFound")) {
                if (display === "page") setLocation(`${LINKS.Chat}/add`, { replace: true, searchParams: parseSearchParams() });
                setExisting(chatInitialValues(session, t, getUserLanguages(session)[0]));
            }
        },
        disabled: display === "dialog" && isOpen !== true,
        displayError: display === "page" || isOpen === true,
        isCreate,
        objectType: "Chat",
        overrideObject: overrideObject as unknown as Chat,
        transform: (data) => chatInitialValues(session, t, getUserLanguages(session)[0], data),
    });

    async function validateValues(values: ChatShape) {
        return await validateFormValues(values, existing, isCreate, transformChatValues, chatValidation);
    }

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) =>
                <>
                    <ChatForm
                        disabled={!(isCreate || permissions.canUpdate)}
                        display={display}
                        existing={existing}
                        handleUpdate={setExisting}
                        isCreate={isCreate}
                        isReadLoading={isReadLoading}
                        isOpen={isOpen}
                        {...props}
                        {...formik}
                    />
                </>
            }
        </Formik>
    );
}
