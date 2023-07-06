import { DUMMY_ID, orDefault, Organization, organizationTranslationValidation, organizationValidation, Session } from "@local/shared";
import { useTheme } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ResourceListHorizontalInput } from "components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedMarkdownInput } from "components/inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { OrganizationFormProps } from "forms/types";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { validateAndGetYupErrors } from "utils/shape/general";
import { OrganizationShape, shapeOrganization } from "utils/shape/models/organization";

export const organizationInitialValues = (
    session: Session | undefined,
    existing?: Organization | null | undefined,
): OrganizationShape => ({
    __typename: "Organization" as const,
    id: DUMMY_ID,
    isOpenToNewMembers: false,
    isPrivate: false,
    tags: [],
    ...existing,
    translations: orDefault(existing?.translations, [{
        __typename: "OrganizationTranslation" as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        name: "",
        bio: "",
    }]),
});

export const transformOrganizationValues = (values: OrganizationShape, existing?: OrganizationShape) => {
    return existing === undefined
        ? shapeOrganization.create(values)
        : shapeOrganization.update(existing, values);
};

export const validateOrganizationValues = async (values: OrganizationShape, existing?: OrganizationShape) => {
    const transformedValues = transformOrganizationValues(values, existing);
    const validationSchema = organizationValidation[existing === undefined ? "create" : "update"]({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
};

export const OrganizationForm = forwardRef<BaseFormRef | undefined, OrganizationFormProps>(({
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
        fields: ["bio", "name"],
        validationSchema: organizationTranslationValidation[isCreate ? "create" : "update"]({}),
    });

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
                        objectType={"Organization"}
                        zIndex={zIndex}
                    />
                    <ResourceListHorizontalInput
                        isCreate={true}
                        zIndex={zIndex}
                    />
                    <FormSection>
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
                        <TranslatedMarkdownInput
                            language={language}
                            maxChars={2048}
                            minRows={4}
                            name="bio"
                            placeholder={t("Bio")}
                            zIndex={zIndex}
                        />
                        <br />
                        <TagSelector
                            name="tags"
                            zIndex={zIndex}
                        />
                    </FormSection>
                </FormContainer>
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
