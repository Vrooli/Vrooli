import { Chat, ChatCreateInput, ChatMessage, chatTranslationValidation, ChatUpdateInput, chatValidation, DUMMY_ID, endpointGetChat, endpointPostChat, endpointPutChat, exists, orDefault, Session, uuid, VALYXA_ID } from "@local/shared";
import { Box, Checkbox, IconButton, InputAdornment, Stack, TextField, Typography, useTheme } from "@mui/material";
import { fetchLazyWrapper, socket } from "api";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { ChatBubble } from "components/ChatBubble/ChatBubble";
import { ChatSideMenu } from "components/dialogs/ChatSideMenu/ChatSideMenu";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { RichInputBase } from "components/inputs/RichInputBase/RichInputBase";
import { TranslatedRichInput } from "components/inputs/TranslatedRichInput/TranslatedRichInput";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Resizable, useDimensionContext } from "components/Resizable/Resizable";
import { EditableTitle } from "components/text/EditableTitle/EditableTitle";
import { SessionContext } from "contexts/SessionContext";
import { Field, Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { ChatFormProps } from "forms/types";
import { useDeleter } from "hooks/useDeleter";
import { useFormDialog } from "hooks/useFormDialog";
import { useObjectActions } from "hooks/useObjectActions";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { TFunction } from "i18next";
import { CopyIcon, ListIcon, SendIcon } from "icons";
import { Dispatch, SetStateAction, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { FormContainer, FormSection, pagePaddingBottom } from "styles";
import { AssistantTask } from "types";
import { getCurrentUser } from "utils/authentication/session";
import { getYou } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { getUserLanguages } from "utils/display/translationTools";
import { uuidToBase36 } from "utils/navigation/urlTools";
import { noopSubmit } from "utils/objects";
import { PubSub } from "utils/pubsub";
import { updateArray, validateAndGetYupErrors } from "utils/shape/general";
import { ChatShape, shapeChat } from "utils/shape/models/chat";
import { ChatInviteShape } from "utils/shape/models/chatInvite";
import { ChatMessageShape } from "utils/shape/models/chatMessage";
import { ViewDisplayType } from "views/types";
import { ChatCrudProps } from "../types";

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
            isFork: true,
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
    console.log("initializing chat values", messages);
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

export const validateChatValues = async (values: ChatShape, existing: ChatShape, isCreate: boolean) => {
    const transformedValues = transformChatValues(values, existing, isCreate);
    const validationSchema = chatValidation[isCreate ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

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

const NewMessageContainer = ({
    chat,
    display,
    message,
    handleSubmit,
    setMessage,
}: {
    chat: ChatShape,
    display: ViewDisplayType,
    message: string,
    setMessage: Dispatch<SetStateAction<string>>,
    handleSubmit: (text: string) => unknown,
}) => {
    const session = useContext(SessionContext);
    const dimensions = useDimensionContext();
    console.log("newmessagecontainer dimensions", dimensions);

    return (
        <RichInputBase
            actionButtons={[{
                Icon: SendIcon,
                onClick: () => {
                    if (!chat) {
                        PubSub.get().publishSnack({ message: "Chat not found", severity: "Error" });
                        return;
                    }
                    handleSubmit(message);
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
                console.log("got taggable items", users, searchString);
                return users;
            }}
            maxChars={1500}
            minRows={4}
            maxRows={15}
            onChange={setMessage}
            name="newMessage"
            sxs={{
                root: {
                    height: dimensions.height,
                    width: "100%",
                    // When BottomNav is shown, need to make room for it
                    paddingBottom: { xs: display === "page" ? pagePaddingBottom : 0, md: 0 },
                },
                bar: { borderRadius: 0 },
                textArea: { paddingRight: 4, border: "none", height: "100%" },
            }}
            value={message}
        />
    );
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

    const [message, setMessage] = useState<string>(context ?? "");

    const {
        fetch,
        handleCancel,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<Chat, ChatCreateInput, ChatUpdateInput>({
        display,
        endpointCreate: endpointPostChat,
        endpointUpdate: endpointPutChat,
        isCreate,
        onCancel,
        onCompleted,
        onDeleted,
    });
    const { formRef, handleClose } = useFormDialog({ handleCancel });

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useCallback((updatedChat?: ChatShape) => {
        if (disabled) {
            PubSub.get().publishSnack({ messageKey: "Unauthorized", severity: "Error" });
            return;
        }
        console.log("onsubmittttt values", updatedChat ?? values);
        console.log("onsubmittttt transformed", transformChatValues(updatedChat ?? values, existing, isCreate));
        // Filters out messages that aren't yours, except for ones marked as "isUnsent". This 
        // flag is used both to show messages you sent that haven't been fully sent yet, but also 
        // initial messages when chatting with a bot (which also haven't been sent yet, as the 
        // chat is not created until you send the first message)
        const withoutOtherMessages = (chat: ChatShape) => ({
            ...chat,
            messages: chat.messages.filter(m => m.user?.id === getCurrentUser(session).id || m.isUnsent),
        });
        fetchLazyWrapper<ChatCreateInput | ChatUpdateInput, Chat>({
            fetch,
            inputs: transformChatValues(withoutOtherMessages(updatedChat ?? values), withoutOtherMessages(existing), isCreate),
            onSuccess: (data) => {
                handleUpdate(data);
                setMessage("");
            }, // Most forms will use handleComplete, but we may still be chatting. So don't close the form.
            onCompleted: () => { props.setSubmitting(false); },
        });
    }, [disabled, existing, fetch, handleUpdate, isCreate, props, session, values]);

    // Handle websocket for chat messages (e.g. new message, new reactions, etc.)
    useEffect(() => {
        // Only connect to the websocket if the chat exists
        if (!existing?.id || existing.id === DUMMY_ID) return;
        socket.emit("joinRoom", existing.id, (response) => {
            if (response.error) {
                // handle error
                console.error(response.error);
            } else {
                console.log("Joined chat room");
            }
        });

        // Define chat-specific event handlers
        socket.on("message", (message: ChatMessage) => {
            console.log("GOT MESSAGE", message);
            // Add message to chat if it's not already there. 
            // Make sure it is inserted in the correct order, using the created_at field.
            // Find index to insert message at
            handleUpdate(c => ({
                ...c,
                messages: updateArray(
                    c.messages,
                    c.messages.findIndex(m => m.created_at > message.created_at),
                    message as ChatMessageShape,
                ),
            }));
        });

        // Leave the chat room when the component is unmounted
        return () => {
            socket.emit("leaveRoom", existing.id, (response) => {
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
    }, [existing.id, handleUpdate]);

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
        validationSchema: chatTranslationValidation[isCreate ? "create" : "update"]({}),
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
        onSubmit({
            ...values,
            messages: [...values.messages, newMessage],
        });
    }, [existing.id, language, onSubmit, session, values]);

    const url = useMemo(() => `${window.location.origin}/chat/${uuidToBase36(values.id)}`, [values.id]);
    const copyLink = useCallback(() => {
        navigator.clipboard.writeText(url);
        PubSub.get().publishSnack({ messageKey: "CopiedToClipboard", severity: "Success" });
    }, [url]);

    const openSideMenu = useCallback(() => { PubSub.get().publishSideMenu({ id: "chat-side-menu", isOpen: true }); }, []);
    const closeSideMenu = useCallback(() => { PubSub.get().publishSideMenu({ id: "chat-side-menu", isOpen: false }); }, []);
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
                onClose={handleClose}
            >
                <TopBar
                    display={display}
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
                        onSubmit={onSubmit}
                        titleField="name"
                        subtitleField="description"
                        variant="subheader"
                        sxs={{ stack: { padding: 0 } }}
                        DialogContentForm={() => (
                            <>
                                <BaseForm
                                    dirty={dirty}
                                    display="dialog"
                                    maxWidth={600}
                                    style={{
                                        paddingBottom: "16px",
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
                                            <TranslatedTextField
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
                                                {/* Enable/disable */}
                                                {/* <Checkbox
                                        id="open-to-anyone"
                                        size="small"
                                        name='openToAnyoneWithInvite'
                                        color='secondary'
            
                                    /> */}
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
                                                <TextField
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
                    display: "table",
                    margin: "auto",
                    overflowY: "auto",
                    maxHeight: "calc(100vh - 64px)",
                    minHeight: "calc(100vh - 64px)",
                    minWidth: "min(500px, 100vw)",
                }}>
                    {existing.messages.map((message: ChatMessageShape, index) => {
                        const isOwn = message.user?.id === getCurrentUser(session).id;
                        return <ChatBubble
                            key={index}
                            message={message}
                            index={index}
                            isOwn={isOwn}
                            onUpdated={(updatedMessage) => {
                                handleUpdate(c => ({
                                    ...c,
                                    messages: updateArray(
                                        c.messages,
                                        c.messages.findIndex(m => m.id === updatedMessage.id),
                                        updatedMessage,
                                    ),
                                }));
                            }}
                        />;
                    })}
                </Box>
                <Resizable
                    id="chat-message-input"
                    min={150}
                    max={"50vh"}
                    position="top"
                    sx={{
                        position: "sticky",
                        bottom: 0,
                        height: "min(50vh, 250px)",
                        background: palette.primary.dark,
                        color: palette.primary.contrastText,
                    }}>
                    <NewMessageContainer
                        chat={existing}
                        display={display}
                        message={message}
                        handleSubmit={addMessage}
                        setMessage={setMessage}
                    />
                </Resizable>
            </MaybeLargeDialog>
            <ChatSideMenu />
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
        objectType: "Chat",
        overrideObject: overrideObject as unknown as Chat,
        transform: (data) => chatInitialValues(session, task, t, getUserLanguages(session)[0], data),
    });
    const { canUpdate } = useMemo(() => getYou(existing), [existing]);
    useEffect(() => {
        console.log("EXISTIGN CHAT", existing);
    }, [existing]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateChatValues(values, existing, isCreate)}
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


//TODO
// <Formik
//     enableReinitialize={true}
//     initialValues={{
//         editingMessage: "",
//         newMessage: context ?? "",
//     }}
//     onSubmit={(values, helpers) => {
//         if (!chat) return;
//         const isEditing = values.editingMessage.trim().length > 0;
//         if (isEditing) {
//             //TODO
//         } else {
//             //TODO
//             const text = values.newMessage.trim();
//             if (text.length === 0) return;
//             // for now, just add the message to the list
//             const newMessage: ChatMessage & { isUnsent?: boolean } = {
//                 __typename: "ChatMessage" as const,
//                 id: uuid(),
//                 created_at: new Date().toISOString(),
//                 updated_at: new Date().toISOString(),
//                 isUnsent: true,
//                 chat: {
//                     __typename: "Chat" as const,
//                     id: chat.id,
//                 } as any,
//                 translations: [{
//                     __typename: "ChatMessageTranslation" as const,
//                     id: DUMMY_ID,
//                     language: lng,
//                     text,
//                 }],
//                 user: {
//                     __typename: "User" as const,
//                     id: getCurrentUser(session).id,
//                     isBot: false,
//                     name: getCurrentUser(session).name,
//                 },
//                 you: {
//                     canDelete: true,
//                     canUpdate: true,
//                     canReply: true,
//                     canReport: true,
//                     canReact: true,
//                     reaction: null,
//                 },
//             } as any;
//             console.log("creating message 0", newMessage);
//             setMessages([...messages, newMessage]);
//             helpers.setFieldValue("newMessage", "");
//         }
//     }}
// // validate={async (values) => await validateChatValues(values, existing)}
// ></Formik>
