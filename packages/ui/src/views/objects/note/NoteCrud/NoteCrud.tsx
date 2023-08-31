import { DeleteOneInput, DeleteType, DUMMY_ID, endpointGetNoteVersion, endpointPostDeleteOne, endpointPostNoteVersion, endpointPutNoteVersion, NoteVersion, NoteVersionCreateInput, noteVersionTranslationValidation, NoteVersionUpdateInput, noteVersionValidation, orDefault, Session, Success } from "@local/shared";
import { Box, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { EllipsisActionButton } from "components/buttons/EllipsisActionButton/EllipsisActionButton";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TranslatedRichInput } from "components/inputs/TranslatedRichInput/TranslatedRichInput";
import { ObjectActionsRow } from "components/lists/ObjectActionsRow/ObjectActionsRow";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { EditableTitle } from "components/text/EditableTitle/EditableTitle";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { NoteFormProps } from "forms/types";
import { useFormDialog } from "hooks/useFormDialog";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useObjectActions } from "hooks/useObjectActions";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { ObjectAction } from "utils/actions/objectActions";
import { getCurrentUser } from "utils/authentication/session";
import { getDisplay, getYou, ListObject } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { firstString } from "utils/display/stringTools";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { noopSubmit } from "utils/objects";
import { PubSub } from "utils/pubsub";
import { validateAndGetYupErrors } from "utils/shape/general";
import { NoteVersionShape, shapeNoteVersion } from "utils/shape/models/noteVersion";
import { OwnerShape } from "utils/shape/models/types";
import { NoteCrudProps } from "views/objects/note/types";

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

const transformNoteValues = (values: NoteVersionShape, existing: NoteVersionShape, isCreate: boolean) =>
    isCreate ? shapeNoteVersion.create(values) : shapeNoteVersion.update(existing, values);

const validateNoteValues = async (values: NoteVersionShape, existing: NoteVersionShape, isCreate: boolean) => {
    const transformedValues = transformNoteValues(values, existing, isCreate);
    console.log("validating note value", values, transformedValues);
    const validationSchema = noteVersionValidation[isCreate ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

const NoteForm = ({
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
    values,
    ...props
}: NoteFormProps) => {
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const {
        fetch,
        handleCancel,
        handleCompleted,
        handleDeleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<NoteVersion, NoteVersionCreateInput, NoteVersionUpdateInput>({
        display,
        endpointCreate: endpointPostNoteVersion,
        endpointUpdate: endpointPutNoteVersion,
        isCreate,
        onCancel,
        onCompleted,
        onDeleted,
    });
    const { formRef, handleClose } = useFormDialog({ handleCancel });

    const onSubmit = useCallback(() => {
        if (disabled) {
            PubSub.get().publishSnack({ messageKey: "Unauthorized", severity: "Error" });
            return;
        }
        console.log("in onSubmit - values: ", values);
        console.log("in onSubmit - transformNoteValues: ", transformNoteValues(values, existing, isCreate));
        fetchLazyWrapper<NoteVersionCreateInput | NoteVersionUpdateInput, NoteVersion>({
            fetch,
            inputs: transformNoteValues(values, existing, isCreate),
            onSuccess: (data) => { handleCompleted(data); },
            onCompleted: () => { props.setSubmitting(false); },
        });
    }, [disabled, existing, fetch, handleCompleted, isCreate, props, values]);

    const {
        handleAddLanguage,
        handleDeleteLanguage,
        language,
        languages,
        setLanguage,
        translationErrors,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["description", "name", "pages[0].text"],
        validationSchema: noteVersionTranslationValidation[isCreate ? "create" : "update"]({}),
    });
    console.log("noteform", props.errors, translationErrors);

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
        PubSub.get().publishAlertDialog({
            messageKey: "DeleteConfirm",
            buttons: [{
                labelKey: "Delete",
                onClick: performDelete,
            }, {
                labelKey: "Cancel",
            }],
        });
    }, [deleteMutation, values, t, handleDeleted]);

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || isDeleteLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isDeleteLoading, isUpdateLoading, props.isSubmitting]);

    return (
        <MaybeLargeDialog
            display={display}
            id="note-crud-dialog"
            isOpen={isOpen}
            onClose={handleClose}
            sxs={{ paper: { height: "100%" } }}
        >
            <TopBar
                display={display}
                onClose={handleClose}
                title={firstString(getDisplay(values as ListObject).title, t(isCreate ? "CreateNote" : disabled ? "Note" : "UpdateNote", { count: 1 }))}
                help={getDisplay(values as ListObject).subtitle}
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
            <BaseForm
                dirty={dirty}
                display={display}
                isLoading={isLoading}
                ref={formRef}
                style={{
                    width: "min(800px, 100vw)",
                    height: "100%",
                    paddingBottom: 0,
                }}
            >
                <TranslatedRichInput
                    language={language}
                    name="pages[0].text"
                    placeholder={t("PleaseBeNice")}
                    disabled={disabled}
                    minRows={10}
                    sxs={{
                        bar: {
                            borderRadius: 0,
                            position: "sticky",
                            top: 0,
                        },
                        root: {
                            height: "100%",
                            position: "relative",
                            maxWidth: "800px",
                            ...(display === "page" ? {
                                marginBottom: 4,
                                borderRadius: { xs: 0, md: 1 },
                                overflow: "overlay",
                            } : {}),
                        },
                        textArea: {
                            borderRadius: 0,
                            resize: "none",
                            height: "100%",
                            overflow: "hidden", // Container handles scrolling
                            background: palette.background.paper,
                            border: "none",
                            ...(display === "page" ? {
                                minHeight: "100vh",
                            } : {}),
                        },
                    }}
                />
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                hideButtons={disabled}
                isCreate={isCreate}
                loading={props.isSubmitting}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
                sideActionButtons={{
                    children: (
                        <EllipsisActionButton>
                            <>
                                <Box sx={{
                                    display: "flex",
                                    justifyContent: "center",
                                    alignItems: "center",
                                }}>
                                    {!disabled ? <LanguageInput
                                        currentLanguage={language}
                                        handleAdd={handleAddLanguage}
                                        handleDelete={handleDeleteLanguage}
                                        handleCurrent={setLanguage}
                                        languages={languages}
                                    /> : languages.length > 1 ? <SelectLanguageMenu
                                        currentLanguage={language}
                                        handleCurrent={setLanguage}
                                        languages={languages}
                                    /> : undefined}
                                    {!disabled && <RelationshipList
                                        isEditing={true}
                                        objectType={"Note"}
                                    />}
                                </Box>
                                {!isCreate && (
                                    <ObjectActionsRow
                                        actionData={actionData}
                                        exclude={[ObjectAction.Delete, ObjectAction.Edit]}
                                        object={values as ListObject}
                                    />
                                )}
                            </>
                        </EllipsisActionButton>
                    ),
                }}
            />
        </MaybeLargeDialog>
    );
};

export const NoteCrud = ({
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: NoteCrudProps) => {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, setObject: setExisting } = useObjectFromUrl<NoteVersion, NoteVersionShape>({
        ...endpointGetNoteVersion,
        objectType: "NoteVersion",
        overrideObject,
        transform: (data) => noteInitialValues(session, data),
    });
    const { canUpdate } = useMemo(() => getYou(existing as ListObject), [existing]);
    console.log("EXISTING", existing);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateNoteValues(values, existing, isCreate)}
        >
            {(formik) =>
                <>
                    <NoteForm
                        disabled={!(isCreate || canUpdate)}
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
