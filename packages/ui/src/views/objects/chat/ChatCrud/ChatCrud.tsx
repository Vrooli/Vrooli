import { Chat, ChatCreateInput, ChatInvite, ChatMessage, chatTranslationValidation, ChatUpdateInput, chatValidation, DUMMY_ID, endpointGetChat, endpointPostChat, endpointPutChat, exists, orDefault, Session, uuid, VALYXA_ID } from "@local/shared";
import { Box, IconButton, useTheme } from "@mui/material";
import { fetchLazyWrapper, socket } from "api";
import { ChatBubble } from "components/ChatBubble/ChatBubble";
import { ChatSideMenu } from "components/dialogs/ChatSideMenu/ChatSideMenu";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { RichInput } from "components/inputs/RichInput/RichInput";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Resizable, useDimensionContext } from "components/Resizable/Resizable";
import { EditableTitle } from "components/text/EditableTitle/EditableTitle";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { ChatFormProps } from "forms/types";
import { useDeleter } from "hooks/useDeleter";
import { useFormDialog } from "hooks/useFormDialog";
import { useObjectActions } from "hooks/useObjectActions";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { TFunction } from "i18next";
import { ListIcon, SendIcon } from "icons";
import { FormEvent, useCallback, useContext, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { AssistantTask } from "types";
import { getCurrentUser } from "utils/authentication/session";
import { getDisplay, getYou } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { getUserLanguages } from "utils/display/translationTools";
import { uuidToBase36 } from "utils/navigation/urlTools";
import { noopSubmit } from "utils/objects";
import { PubSub } from "utils/pubsub";
import { updateArray, validateAndGetYupErrors } from "utils/shape/general";
import { ChatShape, shapeChat } from "utils/shape/models/chat";
import { ChatCrudProps } from "../types";

export const chatInitialValues = (
    session: Session | undefined,
    task: AssistantTask | undefined,
    t: TFunction<"common", undefined, "common">,
    language: string,
    existing?: Partial<Chat> | null | undefined,
): ChatShape => {
    const messages = existing?.messages ?? [];
    // If chatting with Valyxa, add start message so that the user 
    // sees something while the chat is loading
    if (exists(existing) && messages.length === 0 && existing.invites?.length === 1 && existing.invites?.some((invite: ChatShape["invites"][0]) => invite.user.id === VALYXA_ID)) {
        const startText = t(task ?? "start", { lng: language, ns: "tasks", defaultValue: "Hello👋, I'm Valyxa! How can I assist you?" });
        messages.push({
            __typename: "ChatMessage" as const,
            id: uuid(),
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            chat: {
                __typename: "Chat" as const,
                id: existing.id,
            } as any,
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
        } as any);
    }
    console.log('initializing chat values', messages)
    return {
        __typename: "Chat" as const,
        id: uuid(),
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
            name: "",
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
    }] as ChatInvite[],
};

const NewMessageContainer = ({
    chat,
    handleSubmit,
}: {
    chat: ChatShape,
    handleSubmit: (e?: FormEvent<HTMLFormElement> | undefined) => unknown,
}) => {
    const session = useContext(SessionContext);
    const dimensions = useDimensionContext();
    console.log("newmessagecontainer dimensions", dimensions);

    return (
        <RichInput
            actionButtons={[{
                Icon: SendIcon,
                onClick: () => {
                    if (!chat) {
                        PubSub.get().publishSnack({ message: "Chat not found", severity: "Error" });
                        return;
                    }
                    handleSubmit();
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
            name="newMessage"
            sxs={{
                root: { height: dimensions.height, width: "100%" },
                bar: { borderRadius: 0 },
                textArea: { paddingRight: 4, border: "none", height: "100%" },
            }}
        />
    );
};

const ChatForm = ({
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

    const {
        fetch,
        handleCancel,
        handleCompleted,
        handleDeleted,
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

    const onSubmit = useCallback(() => {
        if (disabled) {
            PubSub.get().publishSnack({ messageKey: "Unauthorized", severity: "Error" });
            return;
        }
        fetchLazyWrapper<ChatCreateInput | ChatUpdateInput, Chat>({
            fetch,
            inputs: transformChatValues(values, existing, isCreate),
            onSuccess: (data) => { handleCompleted(data); },
            onCompleted: () => { props.setSubmitting(false); },
        });
    }, [disabled, existing, fetch, handleCompleted, isCreate, props, values]);

    // TODO create when either first message is sent, or when title, description, etc. change
    // // Unlike other forms, we'll create the object right away
    // useEffect(() => {
    //     if (!isCreate || !isOpen) return;
    //     // onSubmit();
    // }, [isCreate, isOpen, onSubmit]);

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
            // Add message to chat if it's not already there. 
            // Make sure it is inserted in the correct order, using the created_at field.
            // Find index to insert message at
            handleUpdate(c => ({
                ...c,
                messages: updateArray(
                    c.messages,
                    c.messages.findIndex(m => m.created_at > message.created_at),
                    message,
                ),
            } as Chat));
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
    } = useDeleter({
        objectId: existing?.id,
        objectName: getDisplay(existing).title,
        objectType: "Chat",
        onActionComplete: actionData.onActionComplete,
    });

    return (
        // {object?.id && DeleteDialogComponent}
        <>
            <MaybeLargeDialog
                display={display}
                id="chat-dialog"
                isOpen={isOpen}
                onClose={handleClose}
            >
                <TopBar
                    display={display}
                    // onClose={() => {
                    //     if (formik.values.editingMessage.trim().length > 0) {
                    //         PubSub.get().publishAlertDialog({
                    //             messageKey: "UnsavedChangesBeforeCancel",
                    //             buttons: [
                    //                 { labelKey: "Yes", onClick: () => { tryOnClose(onClose, setLocation); } },
                    //                 { labelKey: "No" },
                    //             ],
                    //         });
                    //     } else {
                    //         tryOnClose(onClose, setLocation);
                    //     }
                    // }}
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
                        titleField="name"
                        subtitleField="description"
                        validationEnabled={false}
                        variant="subheader"
                        sxs={{ stack: { padding: 0 } }}
                    />}
                />
                <Box sx={{
                    overflowY: "auto",
                    maxHeight: "calc(100vh - 64px)",
                    minHeight: "calc(100vh - 64px)",
                }}>
                    {existing.messages.map((message: ChatMessage, index) => {
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
                    <NewMessageContainer chat={existing} handleSubmit={props.handleSubmit} />
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
        overrideObject,
        transform: (data) => chatInitialValues(session, task, t, getUserLanguages(session)[0], data),
    });
    const { canUpdate } = useMemo(() => getYou(existing), [existing]);

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


//TODO
// return (
//     <>
//         <BaseForm
//             dirty={dirty}
//             display={display}
//             isLoading={isLoading}
//             maxWidth={700}
//             ref={ref}
//         >
//             <FormContainer>
//                 {/* TODO relationshiplist should be for connecting organization and inviting users. Can invite non-members even if organization specified, but the search should show members first */}
//                 <RelationshipList
//                     isEditing={true}
//                     objectType={"Chat"}
//                     sx={{ marginBottom: 4 }}
//                 />
//                 <FormSection sx={{
//                     overflowX: "hidden",
//                 }}>
//                     <LanguageInput
//                         currentLanguage={language}
//                         handleAdd={handleAddLanguage}
//                         handleDelete={handleDeleteLanguage}
//                         handleCurrent={setLanguage}
//                         languages={languages}
//                     />
//                     <Field
//                         fullWidth
//                         name="name"
//                         label={t("Name")}
//                         as={TextField}
//                     />
//                     <TranslatedRichInput
//                         language={language}
//                         maxChars={2048}
//                         minRows={4}
//                         name="description"
//                         placeholder={t("Description")}
//                     />
//                 </FormSection>
//                 {/* Invite link */}
//                 <Stack direction="column" spacing={1}>
//                     <Stack direction="row" sx={{ alignItems: "center" }}>
//                         <Typography variant="h6">{t(`OpenToAnyoneWithLink${values.openToAnyoneWithInvite ? "True" : "False"}`)}</Typography>
//                         <HelpButton markdown={t("OpenToAnyoneWithLinkDescription")} />
//                     </Stack>
//                     <Stack direction="row" spacing={0}>
//                         {/* Enable/disable */}
//                         {/* <Checkbox
//                             id="open-to-anyone"
//                             size="small"
//                             name='openToAnyoneWithInvite'
//                             color='secondary'

//                         /> */}
//                         <Field
//                             name="openToAnyoneWithInvite"
//                             type="checkbox"
//                             as={Checkbox}
//                             sx={{
//                                 "&.MuiCheckbox-root": {
//                                     color: palette.secondary.main,
//                                 },
//                             }}
//                         />
//                         {/* Show link with copy adornment*/}
//                         <TextField
//                             disabled
//                             fullWidth
//                             id="invite-link"
//                             label={t("InviteLink")}
//                             variant="outlined"
//                             value={url}
//                             InputProps={{
//                                 endAdornment: (
//                                     <InputAdornment position="end">
//                                         <IconButton
//                                             aria-label={t("Copy")}
//                                             onClick={copyLink}
//                                             size="small"
//                                         >
//                                             <CopyIcon />
//                                         </IconButton>
//                                     </InputAdornment>
//                                 ),
//                             }}
//                         />
//                     </Stack>
//                 </Stack>
//             </FormContainer>
//         </BaseForm>
//         <BottomActionsButtons
//             display={display}
//             errors={combineErrorsWithTranslations(props.errors, translationErrors)}
//             isCreate={isCreate}
//             loading={props.isSubmitting}
//             onCancel={onCancel}
//             onSetSubmitting={props.setSubmitting}
//             onSubmit={props.handleSubmit}
//         />
//     </>
// );
