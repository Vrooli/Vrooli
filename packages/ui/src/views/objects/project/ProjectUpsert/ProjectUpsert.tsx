import { DUMMY_ID, endpointGetProjectVersion, endpointPostProjectVersion, endpointPutProjectVersion, noopSubmit, orDefault, ProjectVersion, ProjectVersionCreateInput, projectVersionTranslationValidation, ProjectVersionUpdateInput, projectVersionValidation, Session } from "@local/shared";
import { useSubmitHelper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TranslatedRichInput } from "components/inputs/TranslatedRichInput/TranslatedRichInput";
import { TranslatedTextInput } from "components/inputs/TranslatedTextInput/TranslatedTextInput";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { DirectoryListHorizontal } from "components/lists/directory";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useSaveToCache } from "hooks/useSaveToCache";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useUpsertFetch } from "hooks/useUpsertFetch";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { getYou } from "utils/display/listTools";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { ProjectShape } from "utils/shape/models/project";
import { ProjectVersionShape, shapeProjectVersion } from "utils/shape/models/projectVersion";
import { ProjectVersionDirectoryShape } from "utils/shape/models/projectVersionDirectory";
import { validateFormValues } from "utils/validateFormValues";
import { ProjectFormProps, ProjectUpsertProps } from "../types";

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
        childNoteVersions: [],
        childOrganizations: [],
        childProjectVersions: [],
        childRoutineVersions: [],
        childSmartContractVersions: [],
        childStandardVersions: [],
    }]),
    translations: orDefault(existing?.translations, [{
        __typename: "ProjectVersionTranslation" as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        name: "",
        description: "",
    }]),
});

const transformProjectVersionValues = (values: ProjectVersionShape, existing: ProjectVersionShape, isCreate: boolean) =>
    isCreate ? shapeProjectVersion.create(values) : shapeProjectVersion.update(existing, values);

const ProjectForm = ({
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
    versions,
    ...props
}: ProjectFormProps) => {
    const session = useContext(SessionContext);
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
        validationSchema: projectVersionTranslationValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" }),
    });

    // For now, we'll only deal with one directory listing
    const [directoryField, , directoryHelpers] = useField("directories[0]");

    const { handleCancel, handleCompleted, isCacheOn } = useUpsertActions<ProjectVersion>({
        display,
        isCreate,
        objectId: values.id,
        objectType: "ProjectVersion",
        ...props,
    });
    const {
        fetch,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertFetch<ProjectVersion, ProjectVersionCreateInput, ProjectVersionUpdateInput>({
        isCreate,
        isMutate: true,
        endpointCreate: endpointPostProjectVersion,
        endpointUpdate: endpointPutProjectVersion,
    });
    useSaveToCache({ isCacheOn, isCreate, values, objectId: values.id, objectType: "ProjectVersion" });

    const isLoading = useMemo(() => isCreateLoading || isReadLoading || isUpdateLoading || props.isSubmitting, [isCreateLoading, isReadLoading, isUpdateLoading, props.isSubmitting]);

    const onSubmit = useSubmitHelper<ProjectVersionCreateInput | ProjectVersionUpdateInput, ProjectVersion>({
        disabled,
        existing,
        fetch,
        inputs: transformProjectVersionValues(values, existing, isCreate),
        isCreate,
        onSuccess: (data) => { handleCompleted(data); },
        onCompleted: () => { props.setSubmitting(false); },
    });

    return (
        <MaybeLargeDialog
            display={display}
            id="project-upsert-dialog"
            isOpen={isOpen}
            onClose={onClose}
        >
            <TopBar
                display={display}
                onClose={onClose}
                title={t(isCreate ? "CreateProject" : "UpdateProject")}
            />
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={700}
            >
                <FormContainer>
                    <RelationshipList
                        isEditing={true}
                        objectType={"Project"}
                        sx={{ marginBottom: 4 }}
                    />
                    <FormSection>
                        <TranslatedTextInput
                            autoFocus
                            fullWidth
                            label={t("Name")}
                            placeholder={t("NamePlaceholder")}
                            language={language}
                            name="name"
                        />
                        <TranslatedRichInput
                            language={language}
                            name="description"
                            maxChars={2048}
                            minRows={4}
                            maxRows={8}
                            placeholder={t("DescriptionPlaceholder")}
                        />
                        <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleDeleteLanguage}
                            handleCurrent={setLanguage}
                            languages={languages}
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
                hideButtons={disabled}
                isCreate={isCreate}
                loading={isLoading}
                onCancel={handleCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
            />
        </MaybeLargeDialog>
    );
};

export const ProjectUpsert = ({
    isCreate,
    isOpen,
    overrideObject,
    ...props
}: ProjectUpsertProps) => {
    const session = useContext(SessionContext);

    const { isLoading: isReadLoading, object: existing, setObject: setExisting } = useObjectFromUrl<ProjectVersion, ProjectVersionShape>({
        ...endpointGetProjectVersion,
        isCreate,
        objectType: "ProjectVersion",
        overrideObject,
        transform: (existing) => projectInitialValues(session, existing),
    });
    const { canUpdate } = useMemo(() => getYou(existing), [existing]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={existing}
            onSubmit={noopSubmit}
            validate={async (values) => await validateFormValues(values, existing, isCreate, transformProjectVersionValues, projectVersionValidation)}
        >
            {(formik) => <ProjectForm
                disabled={!(isCreate || canUpdate)}
                existing={existing}
                handleUpdate={setExisting}
                isCreate={isCreate}
                isReadLoading={isReadLoading}
                isOpen={isOpen}
                versions={(existing?.root as ProjectShape)?.versions?.map(v => v.versionLabel) ?? []}
                {...props}
                {...formik}
            />}
        </Formik>
    );
};
