import { useTheme } from "@mui/material";
import { ProjectVersion, Session } from "@shared/consts";
import { DUMMY_ID } from "@shared/uuid";
import { projectVersionTranslationValidation, projectVersionValidation } from "@shared/validation";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { ProjectFormProps } from "forms/types";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session";
import { getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { validateAndGetYupErrors } from "utils/shape/general";
import { ProjectVersionShape, shapeProjectVersion } from "utils/shape/models/projectVersion";

export const projectInitialValues = (
    session: Session | undefined,
    existing?: ProjectVersion | null | undefined
): ProjectVersionShape => ({
    __typename: 'ProjectVersion' as const,
    id: DUMMY_ID,
    directoryListings: [],
    isComplete: false,
    isPrivate: true,
    resourceList: {
        __typename: 'ResourceList' as const,
        id: DUMMY_ID,
    },
    root: {
        __typename: 'Project' as const,
        id: DUMMY_ID,
        isPrivate: false,
        owner: { __typename: 'User', id: getCurrentUser(session)!.id! },
        parent: null,
        tags: [],
    },
    translations: [{
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        name: '',
        description: '',
    }],
    versionLabel: '1.0.0',
    ...existing,
});

export const transformProjectValues = (values: ProjectVersionShape, existing?: ProjectVersionShape) => {
    return existing === undefined
        ? shapeProjectVersion.create(values)
        : shapeProjectVersion.update(existing, values)
}

export const validateProjectValues = async (values: ProjectVersionShape, existing?: ProjectVersionShape) => {
    const transformedValues = transformProjectValues(values, existing);
    const validationSchema = existing === undefined
        ? projectVersionValidation.create({})
        : projectVersionValidation.update({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
}

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
        fields: ['description', 'name'],
        validationSchema: projectVersionTranslationValidation.update({}),
    });

    return (
        <>
            <BaseForm
                dirty={dirty}
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: 'block',
                    width: 'min(700px, 100vw - 16px)',
                    margin: 'auto',
                    paddingLeft: 'env(safe-area-inset-left)',
                    paddingRight: 'env(safe-area-inset-right)',
                    paddingBottom: 'calc(64px + env(safe-area-inset-bottom))',
                }}
            >
                <RelationshipList
                    isEditing={true}
                    objectType={'Project'}
                    zIndex={zIndex}
                    sx={{ marginBottom: 4 }}
                />
                <LanguageInput
                    currentLanguage={language}
                    handleAdd={handleAddLanguage}
                    handleDelete={handleDeleteLanguage}
                    handleCurrent={setLanguage}
                    languages={languages}
                    zIndex={zIndex + 1}
                />
                {/* TODO */}
                <VersionInput
                    fullWidth
                    versions={versions}
                />
                <GridSubmitButtons
                    display={display}
                    errors={{
                        ...props.errors,
                        ...translationErrors,
                    }}
                    isCreate={isCreate}
                    loading={props.isSubmitting}
                    onCancel={onCancel}
                    onSetSubmitting={props.setSubmitting}
                    onSubmit={props.handleSubmit}
                />
            </BaseForm>
        </>
    )
})