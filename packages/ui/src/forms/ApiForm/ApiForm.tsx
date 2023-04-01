import { Stack, TextField } from "@mui/material";
import { ApiVersion, Session } from "@shared/consts";
import { DUMMY_ID } from "@shared/uuid";
import { apiVersionTranslationValidation, apiVersionValidation } from "@shared/validation";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ResourceListHorizontalInput } from "components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { Field } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { ApiFormProps } from "forms/types";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session";
import { getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { validateAndGetYupErrors } from "utils/shape/general";
import { ApiVersionShape, shapeApiVersion } from "utils/shape/models/apiVersion";

export const apiInitialValues = (
    session: Session | undefined,
    existing?: ApiVersion | null | undefined
): ApiVersionShape => ({
    __typename: 'ApiVersion' as const,
    id: DUMMY_ID,
    callLink: '',
    directoryListings: [],
    isComplete: false,
    isPrivate: false,
    resourceList: {
        __typename: 'ResourceList' as const,
        id: DUMMY_ID,
    },
    root: {
        __typename: 'Api' as const,
        id: DUMMY_ID,
        isPrivate: false,
        owner: { __typename: 'User', id: getCurrentUser(session)!.id! },
        tags: [],
    },
    translations: [{
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        details: '',
        name: '',
        summary: '',
    }],
    versionLabel: '1.0.0',
    ...existing,
});

export const transformApiValues = (o: ApiVersionShape, u?: ApiVersionShape) => {
    return u === undefined
        ? shapeApiVersion.create(o)
        : shapeApiVersion.update(o, u)
}

export const validateApiValues = async (values: ApiVersionShape, isCreate: boolean) => {
    const transformedValues = transformApiValues(values);
    const validationSchema = isCreate
        ? apiVersionValidation.create({})
        : apiVersionValidation.update({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
}

export const ApiForm = forwardRef<any, ApiFormProps>(({
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

    // useEffect(() => {
    //     const params = parseSearchParams();
    //     if (typeof params.tag === 'string') setTags([{ tag: params.tag }]);
    //     else if (Array.isArray(params.tags)) setTags(params.tags.map((t: any) => ({ tag: t })));
    // }, []);

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
        fields: ['details', 'name', 'summary'],
        validationSchema: apiVersionTranslationValidation.update({}),
    });

    return (
        <>
            <BaseForm
                dirty={dirty}
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: 'block',
                    maxWidth: '700px',
                    margin: 'auto',
                }}
            >
                <Stack direction="column" spacing={4} sx={{
                    margin: 2,
                    marginBottom: 4,
                }}>
                    <RelationshipList
                        isEditing={true}
                        objectType={'Api'}
                        zIndex={zIndex}
                    />
                    <Field
                        fullWidth
                        name="callLink"
                        label={"Endpoint URL"}
                        helperText={"The full URL to the endpoint."}
                        as={TextField}
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
                            label={"Summary"}
                            language={language}
                            multiline
                            minRows={2}
                            maxRows={2}
                            name="summary"
                        />
                        <TranslatedTextField
                            fullWidth
                            label={"Details"}
                            language={language}
                            multiline
                            minRows={4}
                            maxRows={8}
                            name="details"
                        />
                    </Stack>
                    <ResourceListHorizontalInput
                        isCreate={true}
                        zIndex={zIndex}
                    />
                    <TagSelector />
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