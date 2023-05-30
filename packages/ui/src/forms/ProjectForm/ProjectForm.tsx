import { DUMMY_ID, orDefault, ProjectVersion, projectVersionTranslationValidation, projectVersionValidation, Session } from "@local/shared";
import { Stack, useTheme } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { DirectoryListHorizontal } from "components/lists/directory";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { ProjectFormProps } from "forms/types";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { validateAndGetYupErrors } from "utils/shape/general";
import { ProjectVersionShape, shapeProjectVersion } from "utils/shape/models/projectVersion";

export const projectInitialValues = (
    session: Session | undefined,
    existing?: ProjectVersion | null | undefined,
): ProjectVersionShape => ({
    __typename: "ProjectVersion" as const,
    id: DUMMY_ID,
    isComplete: false,
    isPrivate: true,
    resourceList: {
        __typename: "ResourceList" as const,
        id: DUMMY_ID,
    },
    root: {
        __typename: "Project" as const,
        id: DUMMY_ID,
        isPrivate: false,
        owner: { __typename: "User", id: getCurrentUser(session)!.id! },
        parent: null,
        tags: [],
    },
    versionLabel: "1.0.0",
    ...existing,
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

export const transformProjectValues = (values: ProjectVersionShape, existing?: ProjectVersionShape) => {
    return existing === undefined
        ? shapeProjectVersion.create(values)
        : shapeProjectVersion.update(existing, values);
};

export const validateProjectValues = async (values: ProjectVersionShape, existing?: ProjectVersionShape) => {
    const transformedValues = transformProjectValues(values, existing);
    const validationSchema = projectVersionValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const ProjectForm = forwardRef<any, ProjectFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    values,
    versions,
    zIndex,
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
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: "block",
                    width: "min(700px, 100vw - 16px)",
                    margin: "auto",
                    paddingLeft: "env(safe-area-inset-left)",
                    paddingRight: "env(safe-area-inset-right)",
                    paddingBottom: "calc(64px + env(safe-area-inset-bottom))",
                }}
            >
                <Stack direction="column" spacing={4} sx={{
                    margin: 2,
                    marginBottom: 4,
                }}>
                    <RelationshipList
                        isEditing={true}
                        objectType={"Project"}
                        zIndex={zIndex}
                        sx={{ marginBottom: 4 }}
                    />
                    <Stack direction="column" spacing={2} sx={{
                        borderRadius: 2,
                        background: palette.mode === "dark" ? palette.background.paper : palette.background.default,
                        padding: 2,
                    }}>
                        <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleDeleteLanguage}
                            handleCurrent={setLanguage}
                            languages={languages}
                            zIndex={zIndex + 1}
                        />
                        <TranslatedTextField
                            fullWidth
                            label={t("Name")}
                            language={language}
                            name="name"
                        />
                        <TranslatedTextField
                            fullWidth
                            label={t("Description")}
                            language={language}
                            multiline
                            minRows={2}
                            maxRows={4}
                            name="description"
                        />
                    </Stack>
                    <DirectoryListHorizontal
                        canUpdate={true}
                        directory={directoryField.value}
                        handleUpdate={directoryHelpers.setValue}
                        loading={isLoading}
                        mutate={false}
                        zIndex={zIndex}
                    />
                    <VersionInput
                        fullWidth
                        versions={versions}
                    />
                </Stack>
                <GridSubmitButtons
                    display={display}
                    errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                    isCreate={isCreate}
                    loading={props.isSubmitting}
                    onCancel={onCancel}
                    onSetSubmitting={props.setSubmitting}
                    onSubmit={props.handleSubmit}
                />
            </BaseForm>
        </>
    );
});
