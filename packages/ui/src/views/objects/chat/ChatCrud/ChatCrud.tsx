import { Chat, ChatCreateInput, ChatMessage, ChatMessageSearchTreeInput, ChatMessageSearchTreeResult, ChatParticipant, chatTranslationValidation, ChatUpdateInput, chatValidation, DUMMY_ID, endpointGetChat, endpointGetChatMessageTree, endpointPostChat, endpointPutChat, exists, noopSubmit, orDefault, Session, uuid, VALYXA_ID } from "@local/shared";
import { Box, Checkbox, IconButton, InputAdornment, Stack, Typography, useTheme } from "@mui/material";
import { errorToMessage, fetchLazyWrapper, ServerResponse, socket } from "api";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { ChatBubbleTree } from "components/ChatBubbleTree/ChatBubbleTree";
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
import { useObjectActions } from "hooks/useObjectActions";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { TFunction } from "i18next";
import { CopyIcon, ListIcon, SendIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { FormContainer, FormSection, pagePaddingBottom } from "styles";
import { AssistantTask } from "types";
import { getCurrentUser } from "utils/authentication/session";
import { removeCookieFormData } from "utils/cookies";
import { getYou } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { getUserLanguages } from "utils/display/translationTools";
import { uuidToBase36 } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { addToArray, updateArray } from "utils/shape/general";
import { ChatShape, shapeChat } from "utils/shape/models/chat";
import { ChatInviteShape } from "utils/shape/models/chatInvite";
import { ChatMessageShape } from "utils/shape/models/chatMessage";
import { validateFormValues } from "utils/validateFormValues";
import { ChatCrudProps, ChatFormProps } from "../types";

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
                id: "4b038f3b-f1f7-1f9b-8f4b-cff4b8f9b20f",
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
        participants: [],
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

/** Basic chatInfo for a new convo with Valyxa */
export const assistantChatInfo: ChatCrudProps["overrideObject"] = {
    __typename: "Chat" as const,
    invites: [{
        __typename: "ChatInvite" as const,
        id: uuid(),
        user: {
            __typename: "User" as const,
            id: VALYXA_ID,
            isBot: true,
            name: "Valyxa",
        },
    }] as unknown as ChatInviteShape[],
};

const ChatForm = ({
    context,
    disabled,
    dirty,
    existing,
    handleUpdate,
    isCreate,
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
    const display = toDisplay(isOpen);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const [message, setMessage] = useHistoryState<string>("chatMessage", context ?? "");
    const [usersTyping, setUsersTyping] = useState<ChatParticipant[]>([]);

    // We query messages separate from the chat, since we must traverse the message tree
    const [getPageData, { data: searchTreeData, loading: isSearchTreeLoading }] = useLazyFetch<ChatMessageSearchTreeInput, ChatMessageSearchTreeResult>(endpointGetChatMessageTree);
    const [allMessages, setAllMessages] = useState<ChatMessageShape[]>(existing.messages ?? []);
    useEffect(() => {
        if (isCreate) return;
        getPageData({ chatId: existing.id });
    }, [existing.id, isCreate, getPageData]);
    useEffect(() => {
        if (!searchTreeData || searchTreeData.messages.length === 0) return;
        // Add to all messages, making sure to only add messages that aren't already there
        setAllMessages(curr => {
            const hash = {};
            // Create hash from curr data
            for (const item of curr) {
                hash[item.id] = item;
            }
            // Add unique items from parsedData
            for (const item of searchTreeData.messages) {
                if (!hash[item.id]) {
                    hash[item.id] = item;
                }
            }
            console.log("setting all messages hash:", hash);
            return Object.values(hash);
        });
    }, [searchTreeData]);

    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<Chat, ChatCreateInput, ChatUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointPostChat,
        endpointUpdate: endpointPutChat,
    });

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || isSearchTreeLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, isSearchTreeLoading, props.isSubmitting]);

    const onSubmit = useCallback((updatedChat?: ChatShape) => {
        if (disabled) {
            PubSub.get().publishSnack({ messageKey: "Unauthorized", severity: "Error" });
            return;
        }
        console.log("onsubmittttt values", updatedChat ?? values);
        console.log("onsubmittttt transformed", JSON.stringify(transformChatValues(updatedChat ?? values, existing, isCreate)));
        /**
         * Finds messages that are yours or are unsent (i.e. bot's initial message), 
         * to make sure you don't attempt to modify other people's messages
         */
        const withoutOtherMessages = (chat: ChatShape) => ({
            ...chat,
            messages: chat.messages.filter(m => m.user?.id === getCurrentUser(session).id || m.isUnsent),
        });
        fetchLazyWrapper<ChatCreateInput | ChatUpdateInput, Chat>({
            fetch,
            inputs: transformChatValues(withoutOtherMessages(updatedChat ?? values), withoutOtherMessages(existing), isCreate),
            onSuccess: (data) => {
                console.log("update success!", data);
                handleUpdate(data);
                setMessage("");
                removeCookieFormData(`Chat-${isCreate ? DUMMY_ID : data.id}`);
            },
            onCompleted: () => { props.setSubmitting(false); },
            onError: (data) => {
                PubSub.get().publishSnack({
                    message: errorToMessage(data as ServerResponse, getUserLanguages(session)),
                    severity: "Error",
                    data,
                });
            },
        });
    }, [disabled, existing, fetch, handleUpdate, isCreate, props, session, setMessage, values]);

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
            // Make sure it's inserted in the correct order, using the created_at field.
            handleUpdate(c => ({
                ...c,
                messages: addToArray(
                    c.messages,
                    message as ChatMessageShape,
                ).sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime()),
            }));
        });
        // When a message is updated, update it in the chat
        socket.on("editMessage", (message: ChatMessage) => {
            handleUpdate(c => ({
                ...c,
                messages: updateArray(
                    c.messages,
                    c.messages.findIndex(m => m.id === message.id),
                    message as ChatMessageShape,
                ),
            }));
        });
        // When a message is deleted, remove it from the chat
        socket.on("deleteMessage", (id: string) => {
            handleUpdate(c => ({
                ...c,
                messages: c.messages.filter(m => m.id !== id),
            }));
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
        return () => {
            // Remove chat-specific event handlers
            socket.off("message");
            socket.off("typing");
        };
    }, [existing.participants, handleUpdate, message, session, usersTyping]);

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
        validationSchema: chatTranslationValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" }),
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
            } as any,
            translations: [{
                __typename: "ChatMessageTranslation" as const,
                id: DUMMY_ID,
                language,
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
        console.log("updating messages", values.messages, newMessage);
        onSubmit({ ...values, messages: [...values.messages, newMessage] });
    }, [existing.id, language, onSubmit, session, values]);

    const url = useMemo(() => `${window.location.origin}/chat/${uuidToBase36(values.id)}`, [values.id]);
    const copyLink = useCallback(() => {
        navigator.clipboard.writeText(url);
        PubSub.get().publishSnack({ messageKey: "CopiedToClipboard", severity: "Success" });
    }, [url]);

    const openSideMenu = useCallback(() => { PubSub.get().publishSideMenu({ id: "chat-side-menu", idPrefix: task, isOpen: true }); }, [task]);
    const closeSideMenu = useCallback(() => { PubSub.get().publishSideMenu({ id: "chat-side-menu", idPrefix: task, isOpen: false }); }, [task]);
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
                            isDeletable={!(isCreate || disabled)}
                            isEditable={!disabled}
                            language={language}
                            onClose={onSubmit}
                            onSubmit={onSubmit}
                            titleField="name"
                            subtitleField="description"
                            variant="subheader"
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
                                            <FormSection sx={{
                                                overflowX: "hidden",
                                            }}>
                                                <LanguageInput
                                                    currentLanguage={language}
                                                    handleAdd={handleAddLanguage}
                                                    handleDelete={handleDeleteLanguage}
                                                    handleCurrent={setLanguage}
                                                    languages={languages}
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
                                            </FormSection>
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
                            allMessages={allMessages}
                            chatId={existing.id}
                            usersTyping={usersTyping}
                        />
                        {/* <Box sx={{ minHeight: "min(400px, 33vh)" }}>
                            {allMessages.map((message: ChatMessageShape, index) => {
                                const isOwn = message.user?.id === getCurrentUser(session).id;
                                return <ChatBubble
                                    key={index}
                                    chatWidth={dimensions.width}
                                    message={message}
                                    index={index}
                                    isOwn={isOwn}
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
                                />;
                            })}
                            <TypingIndicator participants={typing} />
                        </Box> */}
                    </Box>
                    {/* If it's a new chat and the participants contain a bot, then add a warning that you are talking to a bot */}
                    {isCreate &&
                        (existing.participants?.some(p => p.user.isBot) || existing.invites?.some(i => i.user.id === VALYXA_ID)) &&
                        <Typography variant="body2" p={1} sx={{ margin: "auto", maxWidth: "min(700px, 100%)" }}>{t("BotChatWarning")}</Typography>
                    }
                    <RichInputBase
                        actionButtons={[{
                            Icon: SendIcon,
                            onClick: () => {
                                if (!existing) {
                                    PubSub.get().publishSnack({ message: "Chat not found", severity: "Error" });
                                    return;
                                }
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
                        minRows={1}
                        onChange={setMessage}
                        name="newMessage"
                        sxs={{
                            root: {
                                background: palette.primary.dark,
                                color: palette.primary.contrastText,
                                maxHeight: "min(50vh, 500px)",
                                width: "min(700px, 100%)",
                                margin: "auto",
                                marginBottom: { xs: display === "page" ? pagePaddingBottom : "0", md: "0" },
                            },
                            bar: { borderRadius: 0 },
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
    isCreate,
    isOpen,
    overrideObject,
    task,
    ...props
}: ChatCrudProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    const { isLoading: isReadLoading, object: existing, setObject: setExisting } = useObjectFromUrl<Chat, ChatShape>({
        ...endpointGetChat,
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
