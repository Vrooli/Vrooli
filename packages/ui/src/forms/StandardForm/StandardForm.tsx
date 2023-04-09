import { Stack } from "@mui/material";
import { Session, StandardVersion } from "@shared/consts";
import { orDefault } from "@shared/utils";
import { DUMMY_ID } from "@shared/uuid";
import { standardVersionTranslationValidation, standardVersionValidation } from "@shared/validation";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ResourceListHorizontalInput } from "components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { StandardInput } from "components/inputs/standards/StandardInput/StandardInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { StandardFormProps } from "forms/types";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session";
import { InputTypeOptions } from "utils/consts";
import { getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { validateAndGetYupErrors } from "utils/shape/general";
import { shapeStandardVersion, StandardVersionShape } from "utils/shape/models/standardVersion";

export const standardInitialValues = (
    session: Session | undefined,
    existing?: StandardVersion | null | undefined
): StandardVersionShape => ({
    __typename: 'StandardVersion' as const,
    id: DUMMY_ID,
    directoryListings: [],
    isComplete: false,
    isPrivate: false,
    isFile: false,
    standardType: InputTypeOptions[0].value,
    props: JSON.stringify({}),
    default: JSON.stringify({}),
    yup: JSON.stringify({}),
    resourceList: {
        __typename: 'ResourceList' as const,
        id: DUMMY_ID,
    },
    root: {
        __typename: 'Standard' as const,
        id: DUMMY_ID,
        isInternal: false,
        isPrivate: false,
        owner: { __typename: 'User', id: getCurrentUser(session)!.id! },
        parent: null,
        permissions: JSON.stringify({}),
        tags: [],
    },
    versionLabel: '1.0.0',
    ...existing,
    translations: orDefault(existing?.translations, [{
        __typename: 'StandardVersionTranslation' as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        description: '',
        jsonVariable: null, //TODO
        name: '',
    }]),
});

export const transformStandardValues = (values: StandardVersionShape, existing?: StandardVersionShape) => {
    return existing === undefined
        ? shapeStandardVersion.create(values)
        : shapeStandardVersion.update(existing, values)
}

export const validateStandardValues = async (values: StandardVersionShape, existing?: StandardVersionShape) => {
    const transformedValues = transformStandardValues(values, existing);
    const validationSchema = existing === undefined
        ? standardVersionValidation.create({})
        : standardVersionValidation.update({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
}

export const StandardForm = forwardRef<any, StandardFormProps>(({
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
        fields: ['description'],
        validationSchema: standardVersionTranslationValidation.update({}),
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
                <Stack direction="column" spacing={4} sx={{
                    margin: 2,
                    marginBottom: 4,
                }}>
                    <RelationshipList
                        isEditing={true}
                        objectType={'Standard'}
                        zIndex={zIndex}
                    />
                    <Stack direction="column" spacing={2}>
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
                            label={t('Name')}
                            language={language}
                            name="name"
                        />
                        <TranslatedTextField
                            fullWidth
                            label={t('Description')}
                            language={language}
                            multiline
                            minRows={4}
                            maxRows={8}
                            name="description"
                        />
                    </Stack>
                    <StandardInput
                        fieldName="preview"
                        zIndex={zIndex}
                    />
                    <ResourceListHorizontalInput
                        isCreate={true}
                        zIndex={zIndex}
                    />
                    <TagSelector name="root.tags" />
                    <VersionInput
                        fullWidth
                        versions={versions}
                    />
                </Stack>
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