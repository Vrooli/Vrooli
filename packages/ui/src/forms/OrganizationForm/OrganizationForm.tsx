import { Checkbox, FormControlLabel, Stack, Tooltip } from "@mui/material";
import { Organization, Session } from "@shared/consts";
import { orDefault } from "@shared/utils";
import { DUMMY_ID } from "@shared/uuid";
import { organizationTranslationValidation, organizationValidation } from "@shared/validation";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ResourceListHorizontalInput } from "components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { OrganizationFormProps } from "forms/types";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { validateAndGetYupErrors } from "utils/shape/general";
import { OrganizationShape, shapeOrganization } from "utils/shape/models/organization";

export const organizationInitialValues = (
    session: Session | undefined,
    existing?: Organization | null | undefined
): OrganizationShape => ({
    __typename: 'Organization' as const,
    id: DUMMY_ID,
    isOpenToNewMembers: false,
    isPrivate: false,
    tags: [],
    ...existing,
    translations: orDefault(existing?.translations, [{
        __typename: 'OrganizationTranslation' as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        name: '',
        bio: '',
    }]),
});

export const transformOrganizationValues = (values: OrganizationShape, existing?: OrganizationShape) => {
    return existing === undefined
        ? shapeOrganization.create(values)
        : shapeOrganization.update(existing, values)
}

export const validateOrganizationValues = async (values: OrganizationShape, existing?: OrganizationShape) => {
    const transformedValues = transformOrganizationValues(values, existing);
    const validationSchema = existing === undefined
        ? organizationValidation.create({})
        : organizationValidation.update({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
}

export const OrganizationForm = forwardRef<any, OrganizationFormProps>(({
    display,
    dirty,
    isCreate,
    isLoading,
    isOpen,
    onCancel,
    values,
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
        fields: ['bio', 'name'],
        validationSchema: organizationTranslationValidation[isCreate ? 'create' : 'update']({}),
    });

    const [fieldIsOpen] = useField('isOpenToNewMembers');

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
                        objectType={'Organization'}
                        zIndex={zIndex}
                    />
                    <ResourceListHorizontalInput
                        isCreate={true}
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
                            label={"Bio"}
                            language={language}
                            multiline
                            minRows={2}
                            maxRows={4}
                            name="bio"
                        />
                    </Stack>
                    <TagSelector name="tags" />
                    <Tooltip placement={'top'} title='Indicates if this organization should be displayed when users are looking for an organization to join'>
                        <FormControlLabel
                            label='Open to new members?'
                            control={
                                <Checkbox
                                    id='organization-is-open-to-new-members'
                                    size="medium"
                                    name='isOpenToNewMembers'
                                    color='secondary'
                                    checked={fieldIsOpen.value}
                                    onChange={fieldIsOpen.onChange}
                                />
                            }
                        />
                    </Tooltip>
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
    )
})