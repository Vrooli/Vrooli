import { DUMMY_ID, DeleteOneInput, DeleteType, ListObject, NoteVersion, NoteVersionCreateInput, NoteVersionUpdateInput, Session, Success, endpointGetNoteVersion, endpointPostDeleteOne, endpointPostNoteVersion, endpointPutNoteVersion, noopSubmit, noteVersionTranslationValidation, noteVersionValidation, orDefault } from "@local/shared";
import { Box, IconButton, Tooltip, useTheme } from "@mui/material";
import { fetchLazyWrapper, useSubmitHelper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { EllipsisActionButton } from "components/buttons/EllipsisActionButton/EllipsisActionButton";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TranslatedRichInput } from "components/inputs/RichInput/RichInput";
import { TranslatedTextInput } from "components/inputs/TextInput/TextInput";
import { ObjectActionsRow } from "components/lists/ObjectActionsRow/ObjectActionsRow";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { EditableTitle } from "components/text/EditableTitle/EditableTitle";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useObjectActions } from "hooks/useObjectActions";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { useWindowSize } from "hooks/useWindowSize";
import { MagicIcon } from "icons";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { FormContainer, FormSection } from "styles";
import { ObjectAction } from "utils/actions/objectActions";
import { getCurrentUser } from "utils/authentication/session";
import { getDisplay } from "utils/display/listTools";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { NoteVersionShape, shapeNoteVersion } from "utils/shape/models/noteVersion";
import { OwnerShape } from "utils/shape/models/types";
import { validateFormValues } from "utils/validateFormValues";
import { NoteCrudProps } from "views/objects/note/types";
import { NoteFormProps } from "../types";

const noteInitialValues = (
    session: Session | undefined,
    existing?: Partial<NoteVersion> | null | undefined,
): NoteVersionShape => ({
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
});

const transformNoteVersionValues = (values: NoteVersionShape, existing: NoteVersionShape, isCreate: boolean) =>
    isCreate ? shapeNoteVersion.create(values) : shapeNoteVersion.update(existing, values);

const NoteForm = ({
    disabled,
    dirty,
    display,
    existing,
    handleUpdate,
    isCreate,
    isOpen,
    isReadLoading,
    onCancel,
    onClose,
    onCompleted,
    onDeleted,
    values,
    ...props
}: NoteFormProps) => {
    const session = useContext(SessionContext);
    const { credits } = useMemo(() => getCurrentUser(session), [session]);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const { breakpoints, palette } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    const { handleCancel, handleCompleted, handleDeleted } = useUpsertActions<NoteVersion>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "NoteVersion",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<NoteVersion, NoteVersionCreateInput, NoteVersionUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointPostNoteVersion,
        endpointUpdate: endpointPutNoteVersion,
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
        handleAddLanguage,
        handleDeleteLanguage,
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
    const [deleteMutation, { loading: isDeleteLoading }] = useLazyFetch<DeleteOneInput, Success>(endpointPostDeleteOne);
    const handleDelete = useCallback(() => {
        const performDelete = () => {
            fetchLazyWrapper<DeleteOneInput, Success>({
                fetch: deleteMutation,
                inputs: { id: values.id, objectType: DeleteType.Note },
                successCondition: (data) => data.success,
                successMessage: () => ({ messageKey: "ObjectDeleted", messageVariables: { name: getDisplay(values as ListObject).title ?? t("Note", { count: 1 }) } }),
                onSuccess: () => { handleDeleted(values as NoteVersion); },
                errorMessage: () => ({ messageKey: "FailedToDelete" }),
            });
        };
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

    const sideActionButtons = useMemo(() => {
        const buttons: JSX.Element[] = [];
        if (!isCreate) {
            buttons.push(
                <ObjectActionsRow
                    actionData={actionData}
                    exclude={[ObjectAction.Delete, ObjectAction.Edit]}
                    object={values as ListObject}
                />,
            );
        }
        if (disabled && languages.length > 1) {
            buttons.push(
                <Box sx={{
                    display: "flex",
                    justifyContent: "center",
                    alignItems: "center",
                }}>
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
                        <MagicIcon fill={palette.secondary.contrastText} width="36px" height="36px" />
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

    return (
        <>
            <MaybeLargeDialog
                display={display}
                id="note-crud-dialog"
                isOpen={isOpen}
                onClose={onClose}
                sxs={{ paper: { height: "100%" } }}
            >
                <Box sx={{
                    display: "flex",
                    flexDirection: "column",
                    height: "100vh",
                    overflow: "hidden",
                }}>
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
                            sxs={{
                                stack: {
                                    padding: 0,
                                    ...(display === "page" && !isMobile ? {
                                        margin: "auto",
                                        maxWidth: "800px",
                                        paddingTop: 1,
                                        paddingBottom: 1,
                                    } : {}),
                                },
                            }}
                            DialogContentForm={() => (
                                <>
                                    <BaseForm
                                        display="dialog"
                                        style={{
                                            width: "min(700px, 100vw)",
                                            paddingBottom: "16px",
                                        }}
                                    >
                                        <FormContainer>
                                            <RelationshipList
                                                isEditing={!disabled}
                                                objectType={"Note"}
                                                sx={{ marginBottom: 4 }}
                                            />
                                            <FormSection sx={{ overflowX: "hidden" }}>
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
                                                    placeholder={t("DescriptionPlaceholder")}
                                                />
                                            </FormSection>
                                        </FormContainer>
                                    </BaseForm>
                                </>
                            )}
                        />}
                    />
                    <BaseForm
                        display={display}
                        isLoading={isLoading}
                        style={{ display: "contents" }}
                    >
                        <TranslatedRichInput
                            language={language}
                            autoFocus
                            name="pages[0].text"
                            placeholder={t("PleaseBeNice")}
                            disabled={disabled}
                            sxs={{
                                root: {
                                    display: "flex",
                                    position: "relative",
                                    width: "100%",
                                    maxWidth: "800px",
                                    borderRadius: { xs: 0, md: 1 },
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
                            }}
                        />
                    </BaseForm>
                    <Box sx={{
                        position: "absolute",
                        left: 0,
                        right: 0,
                        bottom: isMobile ? "calc(64px - 8px)" : 0,
                    }}>
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
                    </Box>
                </Box>
            </MaybeLargeDialog>
        </>
    );
};

export const NoteCrud = ({
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: NoteCrudProps) => {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useObjectFromUrl<NoteVersion, NoteVersionShape>({
        ...endpointGetNoteVersion,
        isCreate,
        objectType: "NoteVersion",
        overrideObject,
        transform: (data) => noteInitialValues(session, data),
    });

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateFormValues(values, existing, isCreate, transformNoteVersionValues, noteVersionValidation)}
        >
            {(formik) =>
                <>
                    <NoteForm
                        disabled={!(isCreate || permissions.canUpdate)}
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
};
