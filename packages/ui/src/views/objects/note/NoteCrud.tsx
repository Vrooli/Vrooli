import { DUMMY_ID, DeleteOneInput, DeleteType, ListObject, LlmTask, NoteVersion, NoteVersionCreateInput, NoteVersionShape, NoteVersionUpdateInput, OwnerShape, Session, Success, endpointsActions, endpointsNoteVersion, noopSubmit, noteVersionTranslationValidation, noteVersionValidation, orDefault, shapeNoteVersion } from "@local/shared";
import { Box, IconButton, Tooltip, styled, useTheme } from "@mui/material";
import { Formik } from "formik";
import { useCallback, useContext, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper, useSubmitHelper } from "../../../api/fetchWrapper.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { EllipsisActionButton } from "../../../components/buttons/EllipsisActionButton/EllipsisActionButton.js";
import { MaybeLargeDialog } from "../../../components/dialogs/LargeDialog/LargeDialog.js";
import { SelectLanguageMenu } from "../../../components/dialogs/SelectLanguageMenu/SelectLanguageMenu.js";
import { TranslatedRichInput } from "../../../components/inputs/RichInput/RichInput.js";
import { TranslatedTextInput } from "../../../components/inputs/TextInput/TextInput.js";
import { ObjectActionsRow } from "../../../components/lists/ObjectActionsRow/ObjectActionsRow.js";
import { RelationshipList } from "../../../components/lists/RelationshipList/RelationshipList.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { EditableTitle } from "../../../components/text/EditableTitle.js";
import { ActiveChatContext } from "../../../contexts/activeChat.js";
import { SessionContext } from "../../../contexts/session.js";
import { BaseForm, InnerForm } from "../../../forms/BaseForm/BaseForm.js";
import { useSaveToCache, useUpsertActions } from "../../../hooks/forms.js";
import { useObjectActions } from "../../../hooks/objectActions.js";
import { useLazyFetch } from "../../../hooks/useLazyFetch.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useTranslatedFields } from "../../../hooks/useTranslatedFields.js";
import { useUpsertFetch } from "../../../hooks/useUpsertFetch.js";
import { useWindowSize } from "../../../hooks/useWindowSize.js";
import { IconCommon } from "../../../icons/Icons.js";
import { useLocation } from "../../../route/router.js";
import { FormContainer, FormSection, ScrollBox } from "../../../styles.js";
import { ObjectAction } from "../../../utils/actions/objectActions.js";
import { getCurrentUser } from "../../../utils/authentication/session.js";
import { taskToTaskInfo } from "../../../utils/display/chatTools.js";
import { getDisplay } from "../../../utils/display/listTools.js";
import { combineErrorsWithTranslations, getUserLanguages } from "../../../utils/display/translationTools.js";
import { PubSub } from "../../../utils/pubsub.js";
import { validateFormValues } from "../../../utils/validateFormValues.js";
import { NoteCrudProps } from "../../../views/objects/note/types.js";
import { NoteFormProps } from "./types.js";

const sideActionButtonsExclude = [ObjectAction.Delete, ObjectAction.Edit] as const;

function noteInitialValues(
    session: Session | undefined,
    existing?: Partial<NoteVersion> | null | undefined,
): NoteVersionShape {
    return {
        __typename: "NoteVersion" as const,
        id: DUMMY_ID,
        directoryListings: [],
        isPrivate: true,
        versionLabel: existing?.versionLabel ?? "1.0.0",
        ...existing,
        root: {
            __typename: "Note" as const,
            id: DUMMY_ID,
            isPrivate: true,
            owner: { __typename: "User", id: getCurrentUser(session)?.id ?? "" } as OwnerShape,
            parent: null,
            tags: [],
            ...existing?.root,
        },
        translations: orDefault(existing?.translations, [{
            __typename: "NoteVersionTranslation" as const,
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            description: "",
            name: "New Note",
            pages: [{
                __typename: "NotePage" as const,
                id: DUMMY_ID,
                pageIndex: 0,
                text: "",
            }],
        }]),
    };
}

function transformNoteVersionValues(values: NoteVersionShape, existing: NoteVersionShape, isCreate: boolean) {
    return isCreate ? shapeNoteVersion.create(values) : shapeNoteVersion.update(existing, values);
}

const OuterFormBox = styled(Box)(() => ({
    display: "flex",
    flexDirection: "column",
    height: "100vh",
    overflow: "hidden",
}));

const dialogStyle = { paper: { height: "100%" } } as const;
const relationshipListStyle = { marginBottom: 4 } as const;
const titleDialogFormStyle = {
    padding: "16px",
    width: "600px",
} as const;
const formStyle = { display: "contents" } as const;

function NoteForm({
    disabled,
    display,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    onClose,
    values,
    ...props
}: NoteFormProps) {
    const session = useContext(SessionContext);
    const { credits } = useMemo(() => getCurrentUser(session), [session]);
    const { chat } = useContext(ActiveChatContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    const { handleCancel, handleCompleted, handleDeleted } = useUpsertActions<NoteVersion>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "NoteVersion",
        rootObjectId: values.root?.id,
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<NoteVersion, NoteVersionCreateInput, NoteVersionUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointsNoteVersion.createOne,
        endpointUpdate: endpointsNoteVersion.updateOne,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "NoteVersion" });

    const onSubmit = useSubmitHelper<NoteVersionCreateInput | NoteVersionUpdateInput, NoteVersion>({
        disabled,
        existing,
        fetch,
        inputs: transformNoteVersionValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    const {
        language,
        languages,
        setLanguage,
        translationErrors,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        validationSchema: noteVersionTranslationValidation.read({ env: process.env.NODE_ENV }),
    });

    const actionData = useObjectActions({
        object: existing as ListObject,
        objectType: "NoteVersion",
        setLocation,
        setObject: handleUpdate,
    });

    // Handle delete
    const [deleteMutation, { loading: isDeleteLoading }] = useLazyFetch<DeleteOneInput, Success>(endpointsActions.deleteOne);
    const handleDelete = useCallback(() => {
        function performDelete() {
            fetchLazyWrapper<DeleteOneInput, Success>({
                fetch: deleteMutation,
                inputs: { id: values.id, objectType: DeleteType.Note },
                successCondition: (data) => data.success,
                successMessage: () => ({ messageKey: "ObjectDeleted", messageVariables: { name: getDisplay(values as ListObject).title ?? t("Note", { count: 1 }) } }),
                onSuccess: () => { handleDeleted(values as NoteVersion); },
                errorMessage: () => ({ messageKey: "FailedToDelete" }),
            });
        }
        PubSub.get().publish("alertDialog", {
            messageKey: "DeleteConfirm",
            buttons: [{
                labelKey: "Delete",
                onClick: performDelete,
            }, {
                labelKey: "Cancel",
            }],
        });
    }, [deleteMutation, values, t, handleDeleted]);

    const openAssistantDialog = useCallback(() => {
        // Hacky, but it works
        const assistantButton = document.getElementById("input-container-pages[0].text-toolbar-assistant");
        assistantButton?.click();
    }, []);

    // Suggest the autofill task to the main chat
    useEffect(function suggestTaskEffect() {
        const chatId = chat?.id;
        if (!chatId) return;
        PubSub.get().publish("chatTask", {
            chatId,
            tasks: {
                add: {
                    inactive: {
                        behavior: "onlyIfNoTaskType",
                        value: [taskToTaskInfo(isCreate ? LlmTask.NoteAdd : LlmTask.NoteUpdate)],
                    },
                },
            },
        });
    }, [chat?.id, isCreate]);

    const sideActionButtons = useMemo(() => {
        const buttons: JSX.Element[] = [];
        if (!isCreate) {
            buttons.push(
                <ObjectActionsRow
                    actionData={actionData}
                    exclude={sideActionButtonsExclude}
                    object={values as ListObject}
                />,
            );
        }
        if (disabled && languages.length > 1) {
            buttons.push(
                <Box display="flex" justifyContent="center" alignItems="center">
                    <SelectLanguageMenu
                        currentLanguage={language}
                        handleCurrent={setLanguage}
                        languages={languages}
                    />
                </Box>,
            );
        }
        if ((isCreate || !disabled) && credits && BigInt(credits) > 0) {
            buttons.push(
                <Tooltip title={t("AIAssistant")} placement="top">
                    <IconButton
                        aria-label={t("AIAssistant")}
                        onClick={openAssistantDialog}
                        sx={{ background: palette.secondary.main }}
                    >
                        <IconCommon name="Magic" fill="secondary.contrastText" size={36} />
                    </IconButton>
                </Tooltip>,
            );
        }
        if (buttons.length === 0) {
            return null;
        }
        if (buttons.length === 1) {
            return buttons[0];
        }
        return (
            <EllipsisActionButton>
                <>
                    {buttons}
                </>
            </EllipsisActionButton>
        );
    }, [actionData, credits, disabled, isCreate, language, languages, openAssistantDialog, palette.secondary.contrastText, palette.secondary.main, setLanguage, t, values]);

    const isLoading = useMemo(() => isCreateLoading || isDeleteLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isDeleteLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

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
                        <RelationshipList
                            isEditing={!disabled}
                            objectType={"Note"}
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
                                placeholder={t("DescriptionPlaceholder")}
                            />
                        </FormSection>
                    </FormContainer>
                </BaseForm>
            </ScrollBox>
        );
    }, [display, disabled, t, language]);

    const topBarStyle = useMemo(function topBarStyleMemo() {
        return {
            stack: {
                padding: 0,
                ...(display === "page" && !isMobile ? {
                    margin: "auto",
                    maxWidth: "800px",
                    paddingTop: 1,
                    paddingBottom: 1,
                } : {}),
            },
        } as const;
    }, [display, isMobile]);

    const pageInputStyle = useMemo(function pageInputStyleMemo() {
        return {
            root: {
                display: "flex",
                position: "relative",
                width: "100%",
                maxWidth: "800px",
                borderRadius: 0,
                overflow: "hidden",
                margin: "auto",
                flex: 1,
            },
            topBar: {
                borderRadius: 0,
            },
            inputRoot: {
                borderRadius: 0,
                height: "100%",
                overflow: "scroll",
                background: palette.background.paper,
                border: "none",
                flex: 1,
            },
            textArea: {
                paddingBottom: "128px",
            },
        } as const;
    }, [palette.background.paper]);

    return (
        <>
            <MaybeLargeDialog
                display={display}
                id="note-crud-dialog"
                isOpen={isOpen}
                onClose={onClose}
                sxs={dialogStyle}
            >
                <OuterFormBox>
                    <TopBar
                        display={display}
                        keepVisible
                        onClose={onClose}
                        titleComponent={<EditableTitle
                            handleDelete={handleDelete}
                            isDeletable={!(isCreate || disabled)}
                            isEditable={!disabled}
                            language={language}
                            titleField="name"
                            subtitleField="description"
                            variant="subheader"
                            sxs={topBarStyle}
                            DialogContentForm={titleDialogContentForm}
                        />}
                    />
                    <InnerForm
                        display={display}
                        isLoading={isLoading}
                        style={formStyle}
                    >
                        <TranslatedRichInput
                            language={language}
                            name="pages[0].text"
                            placeholder={t("PleaseBeNice")}
                            disabled={disabled}
                            sxs={pageInputStyle}
                        />
                    </InnerForm>
                    <BottomActionsButtons
                        display={display}
                        errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                        hideButtons={disabled}
                        isCreate={isCreate}
                        loading={isLoading}
                        onCancel={handleCancel}
                        onSetSubmitting={props.setSubmitting}
                        onSubmit={onSubmit}
                        sideActionButtons={sideActionButtons}
                    />
                </OuterFormBox>
            </MaybeLargeDialog>
        </>
    );
}

export function NoteCrud({
    display,
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: NoteCrudProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<NoteVersion, NoteVersionShape>({
        ...endpointsNoteVersion.findOne,
        disabled: display === "dialog" && isOpen !== true,
        isCreate,
        objectType: "NoteVersion",
        overrideObject,
        transform: (data) => noteInitialValues(session, data),
    });

    async function validateValues(values: NoteVersionShape) {
        return await validateFormValues(values, existing, isCreate, transformNoteVersionValues, noteVersionValidation);
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
                    <NoteForm
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
