import { userTranslationValidation } from "@local/shared";
import { TextField } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ProfilePictureInput } from "components/inputs/ProfilePictureInput/ProfilePictureInput";
import { TranslatedMarkdownInput } from "components/inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { Field } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useContext } from "react";
import { useTranslation } from "react-i18next";
import { FormSection } from "styles";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { SettingsProfileFormProps } from "../types";

export const SettingsProfileForm = ({
    display,
    dirty,
    isLoading,
    numVerifiedWallets,
    onCancel,
    values,
    zIndex,
    ...props
}: SettingsProfileFormProps) => {
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
        fields: ["bio"],
        validationSchema: userTranslationValidation.update({}),
    });

    return (
        <>
            <BaseForm
                dirty={dirty}
                display={display}
                isLoading={isLoading}
                maxWidth={500}
            >
                <ProfilePictureInput
                    onBannerImageChange={(newPicture) => props.setFieldValue("bannerImage", newPicture)}
                    onProfileImageChange={(newPicture) => props.setFieldValue("profileImage", newPicture)}
                    name="profileImage"
                    profile={{ __typename: "User", ...values }}
                    zIndex={zIndex}
                />
                <FormSection sx={{ marginTop: 2 }}>
                    <LanguageInput
                        currentLanguage={language}
                        handleAdd={handleAddLanguage}
                        handleDelete={handleDeleteLanguage}
                        handleCurrent={setLanguage}
                        languages={languages}
                        zIndex={zIndex}
                    />
                    <Field fullWidth name="handle" label={t("Handle")} as={TextField} />
                    <Field fullWidth name="name" label={t("Name")} as={TextField} />
                    <TranslatedMarkdownInput
                        language={language}
                        maxChars={2048}
                        minRows={4}
                        name="bio"
                        placeholder={t("Bio")}
                        zIndex={zIndex}
                    />
                </FormSection>
            </BaseForm>
            <GridSubmitButtons
                display={display}
                errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                isCreate={false}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
                zIndex={zIndex}
            />
        </>
    );
};
