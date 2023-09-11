import { DUMMY_ID, endpointGetProjectVersion, endpointPostProjectVersion, endpointPutProjectVersion, orDefault, ProjectVersion, ProjectVersionCreateInput, projectVersionTranslationValidation, ProjectVersionUpdateInput, projectVersionValidation, Session } from "@local/shared";
import { useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TranslatedRichInput } from "components/inputs/TranslatedRichInput/TranslatedRichInput";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { DirectoryListHorizontal } from "components/lists/directory";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik, useField } from "formik";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { ProjectFormProps } from "forms/types";
import { useFormDialog } from "hooks/useFormDialog";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { toDisplay } from "utils/display/pageTools";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { validateAndGetYupErrors } from "utils/shape/general";
import { ProjectShape } from "utils/shape/models/project";
import { ProjectVersionShape, shapeProjectVersion } from "utils/shape/models/projectVersion";
import { ProjectUpsertProps } from "../types";

const projectInitialValues = (
    session: Session | undefined,
    existing?: Partial<ProjectVersion> | null | undefined,
): ProjectVersionShape => ({
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
        owner: { __typename: "User", id: getCurrentUser(session)!.id! },
        parent: null,
        tags: [],
        ...existing?.root,
    },
    directories: orDefault(existing?.directories, [{
        __typename: "ProjectVersionDirectory" as const,
        id: DUMMY_ID,
        isRoot: true,
        childApiVersions: [],
        childNoteVersions: [],
        childOrganizations: [],
        childProjectVersions: [],
        childRoutineVersions: [],
        childSmartContractVersions: [],
        childStandardVersions: [],
    } as any]),
    translations: orDefault(existing?.translations, [{
        __typename: "ProjectVersionTranslation" as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        name: "",
        description: "",
    }]),
});

const transformProjectValues = (values: ProjectVersionShape, existing: ProjectVersionShape, isCreate: boolean) =>
    isCreate ? shapeProjectVersion.create(values) : shapeProjectVersion.update(existing, values);

const validateProjectValues = async (values: ProjectVersionShape, existing: ProjectVersionShape, isCreate: boolean) => {
    const transformedValues = transformProjectValues(values, existing, isCreate);
    const validationSchema = projectVersionValidation[isCreate ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

const ProjectForm = forwardRef<BaseFormRef | undefined, ProjectFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    values,
    versions,
    ...props
}, ref) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();

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
        validationSchema: projectVersionTranslationValidation[isCreate ? "create" : "update"]({}),
    });

    // For now, we'll only deal with one directory listing
    const [directoryField, , directoryHelpers] = useField("directories[0]");

    return (
        <>
            <BaseForm
                dirty={dirty}
                display={display}
                isLoading={isLoading}
                maxWidth={700}
                ref={ref}
            >
                <FormContainer>
                    <RelationshipList
                        isEditing={true}
                        objectType={"Project"}
                        sx={{ marginBottom: 4 }}
                    />
                    <FormSection>
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
                            name="description"
                            maxChars={2048}
                            minRows={4}
                            maxRows={8}
                            placeholder={t("Description")}
                        />
                    </FormSection>
                    <DirectoryListHorizontal
                        canUpdate={true}
                        directory={directoryField.value}
                        handleUpdate={directoryHelpers.setValue}
                        loading={isLoading}
                        mutate={false}
                    />
                    <VersionInput
                        fullWidth
                        versions={versions}
                    />
                </FormContainer>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                isCreate={isCreate}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </>
    );
});

export const ProjectUpsert = ({
    isCreate,
    isOpen,
    onCancel,
    onCompleted,
    overrideObject,
}: ProjectUpsertProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<ProjectVersion, ProjectVersionShape>({
        ...endpointGetProjectVersion,
        objectType: "ProjectVersion",
        overrideObject,
        transform: (existing) => projectInitialValues(session, existing),
    });

    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<ProjectVersion, ProjectVersionCreateInput, ProjectVersionUpdateInput>({
        display,
        endpointCreate: endpointPostProjectVersion,
        endpointUpdate: endpointPutProjectVersion,
        isCreate,
        onCancel,
        onCompleted,
    });
    const { formRef, handleClose } = useFormDialog({ handleCancel });

    return (
        <MaybeLargeDialog
            display={display}
            id="project-upsert-dialog"
            isOpen={isOpen}
            onClose={handleClose}
        >
            <TopBar
                display={display}
                onClose={handleClose}
                title={t(isCreate ? "CreateProject" : "UpdateProject")}
            />
            <Formik
                enableReinitialize={true}
                initialValues={existing}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<ProjectVersionCreateInput | ProjectVersionUpdateInput, ProjectVersion>({
                        fetch,
                        inputs: transformProjectValues(values, existing, isCreate),
                        onSuccess: (data) => { handleCompleted(data); },
                        onCompleted: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateProjectValues(values, existing, isCreate)}
            >
                {(formik) => <ProjectForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    onClose={handleClose}
                    ref={formRef}
                    versions={(existing?.root as ProjectShape)?.versions?.map(v => v.versionLabel) ?? []}
                    {...formik}
                />}
            </Formik>
        </MaybeLargeDialog>
    );
};
