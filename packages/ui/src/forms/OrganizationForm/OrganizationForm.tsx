import { DUMMY_ID, orDefault, Organization, organizationTranslationValidation, organizationValidation, Session } from "@local/shared";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ProfilePictureInput } from "components/inputs/ProfilePictureInput/ProfilePictureInput";
import { ResourceListHorizontalInput } from "components/inputs/ResourceListHorizontalInput/ResourceListHorizontalInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedRichInput } from "components/inputs/TranslatedRichInput/TranslatedRichInput";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { SessionContext } from "contexts/SessionContext";
import { BaseForm, BaseFormRef } from "forms/BaseForm/BaseForm";
import { OrganizationFormProps } from "forms/types";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { forwardRef, useContext } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer, FormSection } from "styles";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { validateAndGetYupErrors } from "utils/shape/general";
import { OrganizationShape, shapeOrganization } from "utils/shape/models/organization";

export const organizationInitialValues = (
    session: Session | undefined,
    existing?: Partial<Organization> | null | undefined,
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

export const transformOrganizationValues = (values: OrganizationShape, existing: OrganizationShape, isCreate: boolean) =>
    isCreate ? shapeOrganization.create(values) : shapeOrganization.update(existing, values);

export const validateOrganizationValues = async (values: OrganizationShape, existing: OrganizationShape, isCreate: boolean) => {
    const transformedValues = transformOrganizationValues(values, existing, isCreate);
    const validationSchema = organizationValidation[isCreate ? "create" : "update"]({});
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
                    />
                    <ProfilePictureInput
                        onBannerImageChange={(newPicture) => props.setFieldValue("bannerImage", newPicture)}
                        onProfileImageChange={(newPicture) => props.setFieldValue("profileImage", newPicture)}
                        name="profileImage"
                        profile={{ ...values }}
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
                            maxChars={2048}
                            minRows={4}
                            name="bio"
                            placeholder={t("Bio")}
                        />
                        <br />
                        <TagSelector name="tags" />
                    </FormSection>
                    <ResourceListHorizontalInput
                        isCreate={true}
                        parent={{ __typename: "Organization", id: values.id }}
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
