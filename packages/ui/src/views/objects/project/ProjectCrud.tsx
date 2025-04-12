import { DUMMY_ID, DeleteOneInput, DeleteType, ListObject, LlmTask, ProjectShape, ProjectVersion, ProjectVersionCreateInput, ProjectVersionDirectoryShape, ProjectVersionShape, ProjectVersionUpdateInput, Session, Success, endpointsActions, endpointsProjectVersion, noopSubmit, orDefault, projectVersionTranslationValidation, projectVersionValidation, shapeProjectVersion } from "@local/shared";
import { Box, useTheme } from "@mui/material";
import { Formik, useField } from "formik";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper, useSubmitHelper } from "../../../api/fetchWrapper.js";
import { AutoFillButton } from "../../../components/buttons/AutoFillButton.js";
import { BottomActionsButtons } from "../../../components/buttons/BottomActionsButtons.js";
import { MaybeLargeDialog } from "../../../components/dialogs/LargeDialog/LargeDialog.js";
import { LanguageInput } from "../../../components/inputs/LanguageInput/LanguageInput.js";
import { TranslatedRichInput } from "../../../components/inputs/RichInput/RichInput.js";
import { TranslatedTextInput } from "../../../components/inputs/TextInput/TextInput.js";
import { VersionInput } from "../../../components/inputs/VersionInput/VersionInput.js";
import { DirectoryList } from "../../../components/lists/DirectoryList/DirectoryList.js";
import { RelationshipList } from "../../../components/lists/RelationshipList/RelationshipList.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { EditableTitle } from "../../../components/text/EditableTitle.js";
import { SessionContext } from "../../../contexts/session.js";
import { BaseForm } from "../../../forms/BaseForm/BaseForm.js";
import { useSaveToCache, useUpsertActions } from "../../../hooks/forms.js";
import { UseAutoFillProps, getAutoFillTranslationData, useAutoFill } from "../../../hooks/tasks.js";
import { useLazyFetch } from "../../../hooks/useLazyFetch.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useTranslatedFields } from "../../../hooks/useTranslatedFields.js";
import { useUpsertFetch } from "../../../hooks/useUpsertFetch.js";
import { useWindowSize } from "../../../hooks/useWindowSize.js";
import { FormContainer, FormSection } from "../../../styles.js";
import { getCurrentUser } from "../../../utils/authentication/session.js";
import { getDisplay } from "../../../utils/display/listTools.js";
import { combineErrorsWithTranslations, getUserLanguages } from "../../../utils/display/translationTools.js";
import { PubSub } from "../../../utils/pubsub.js";
import { validateFormValues } from "../../../utils/validateFormValues.js";
import { ProjectCrudProps, ProjectFormProps } from "./types.js";

function projectInitialValues(
    session: Session | undefined,
    existing?: Partial<ProjectVersion> | null | undefined,
): ProjectVersionShape {
    return {
        __typename: "ProjectVersion" as const,
        id: DUMMY_ID,
        isComplete: false,
        isPrivate: true,
        resourceList: {
            __typename: "ResourceList" as const,
            id: DUMMY_ID,
        },
        versionLabel: "1.0.0",
        ...existing,
        root: {
            __typename: "Project" as const,
            id: DUMMY_ID,
            isPrivate: false,
            owner: { __typename: "User", id: getCurrentUser(session)?.id ?? "" },
            parent: null,
            tags: [],
            ...existing?.root,
        },
        directories: orDefault<ProjectVersionDirectoryShape[]>(existing?.directories, [{
            __typename: "ProjectVersionDirectory" as const,
            id: DUMMY_ID,
            isRoot: true,
            childApiVersions: [],
            childCodeVersions: [],
            childNoteVersions: [],
            childProjectVersions: [],
            childRoutineVersions: [],
            childStandardVersions: [],
            childTeams: [],
        }]),
        translations: orDefault(existing?.translations, [{
            __typename: "ProjectVersionTranslation" as const,
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            name: "",
            description: "",
        }]),
    };
}

function transformProjectVersionValues(values: ProjectVersionShape, existing: ProjectVersionShape, isCreate: boolean) {
    return isCreate ? shapeProjectVersion.create(values) : shapeProjectVersion.update(existing, values);
}

const formSectionStyle = { overflowX: "hidden" } as const;

function ProjectForm({
    disabled,
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
    versions,
    ...props
}: ProjectFormProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const { breakpoints } = useTheme();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

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
        validationSchema: projectVersionTranslationValidation.create({ env: process.env.NODE_ENV }),
    });

    // For now, we'll only deal with one directory listing
    const [directoryField, , directoryHelpers] = useField("directories[0]");

    const { handleCancel, handleCompleted, handleDeleted } = useUpsertActions<ProjectVersion>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "ProjectVersion",
        rootObjectId: values.root?.id,
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<ProjectVersion, ProjectVersionCreateInput, ProjectVersionUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointsProjectVersion.createOne,
        endpointUpdate: endpointsProjectVersion.updateOne,
    });
    useSaveToCache({ isCreate, values, objectId: values.id, objectType: "ProjectVersion" });

    // Handle delete
    const [deleteMutation, { loading: isDeleteLoading }] = useLazyFetch<DeleteOneInput, Success>(endpointsActions.deleteOne);
    const handleDelete = useCallback(function handleDeleteCallback() {
        function performDelete() {
            fetchLazyWrapper<DeleteOneInput, Success>({
                fetch: deleteMutation,
                inputs: { id: values.id, objectType: DeleteType.Project },
                successCondition: (data) => data.success,
                successMessage: () => ({ messageKey: "ObjectDeleted", messageVariables: { name: getDisplay(values as ListObject).title ?? t("Project", { count: 1 }) } }),
                onSuccess: () => { handleDeleted(values as ProjectVersion); },
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

    const getAutoFillInput = useCallback(function getAutoFillInput() {
        return {
            ...getAutoFillTranslationData(values, language),
            //TODO
        };
    }, [language, values]);

    const shapeAutoFillResult = useCallback(function shapeAutoFillResultCallback(data: Parameters<UseAutoFillProps["shapeAutoFillResult"]>[0]) {
        const originalValues = { ...values };
        const updatedValues = {} as any; //TODO
        console.log("in shapeAutoFillResult", language, data, originalValues, updatedValues);
        return { originalValues, updatedValues };
    }, [language, values]);

    const { autoFill, isAutoFillLoading } = useAutoFill({
        getAutoFillInput,
        shapeAutoFillResult,
        handleUpdate,
        task: isCreate ? LlmTask.ProjectAdd : LlmTask.ProjectUpdate,
    });

    const isLoading = useMemo(() => isAutoFillLoading || isCreateLoading || isDeleteLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isAutoFillLoading, isCreateLoading, isDeleteLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<ProjectVersionCreateInput | ProjectVersionUpdateInput, ProjectVersion>({
        disabled,
        existing,
        fetch,
        inputs: transformProjectVersionValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    const topBarStyle = useMemo(function topBarStyleMemo() {
        return {
            stack: {
                padding: 0,
                ...(display === "page" && !isMobile ? {
                    margin: "auto",
                    maxWidth: "700px",
                    paddingTop: 1,
                    paddingBottom: 1,
                } : {}),
            },
        } as const;
    }, [display, isMobile]);

    const titleDialogContentForm = useCallback(function titleDialogContentFormCallback() {
        return (
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
                        objectType={"Project"}
                        sx={{ marginBottom: 4 }}
                    />
                    <FormSection sx={formSectionStyle}>
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
                    <VersionInput
                        fullWidth
                        versions={versions}
                    />
                </FormContainer>
            </BaseForm>
        );
    }, [disabled, handleAddLanguage, handleDeleteLanguage, language, languages, setLanguage, t, versions]);

    return (
        <MaybeLargeDialog
            display={display}
            id="project-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <Box display="flex" flexDirection="column" minHeight="100vh">
                <TopBar
                    display={display}
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
                <BaseForm
                    display={display}
                    isLoading={isLoading}
                    style={{
                        width: "min(700px, 100vw)",
                        flex: 1,
                        margin: "unset",
                        marginLeft: "auto",
                        marginRight: "auto",
                    }}
                >
                    <DirectoryList
                        canUpdate={!disabled}
                        directory={directoryField.value}
                        handleUpdate={directoryHelpers.setValue}
                        loading={isLoading}
                        mutate={false}
                    />
                </BaseForm>
                <BottomActionsButtons
                    display={display}
                    errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                    hideButtons={disabled}
                    isCreate={isCreate}
                    loading={isLoading}
                    onCancel={handleCancel}
                    onSetSubmitting={props.setSubmitting}
                    onSubmit={onSubmit}
                    sideActionButtons={<AutoFillButton
                        handleAutoFill={autoFill}
                        isAutoFillLoading={isAutoFillLoading}
                    />}
                />
            </Box>
        </MaybeLargeDialog>
    );
}

export function ProjectCrud({
    display,
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: ProjectCrudProps) {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, permissions, setObject: setExisting } = useManagedObject<ProjectVersion, ProjectVersionShape>({
        ...endpointsProjectVersion.findOne,
        disabled: display === "dialog" && isOpen !== true,
        isCreate,
        objectType: "ProjectVersion",
        overrideObject,
        transform: (existing) => projectInitialValues(session, existing),
    });

    async function validateValues(values: ProjectVersionShape) {
        return await validateFormValues(values, existing, isCreate, transformProjectVersionValues, projectVersionValidation);
    }

    const versions = useMemo(function versionsMemo() {
        return (existing?.root as ProjectShape)?.versions?.map(v => v.versionLabel) ?? [];
    }, [existing]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={validateValues}
        >
            {(formik) => <ProjectForm
                disabled={!(isCreate || permissions.canUpdate)}
                display={display}
                existing={existing}
                handleUpdate={setExisting}
                isCreate={isCreate}
                isReadLoading={isReadLoading}
                isOpen={isOpen}
                versions={versions}
                {...props}
                {...formik}
            />}
        </Formik>
    );
}
