import { DUMMY_ID, endpointPutProfile, ProfileUpdateInput, User, userTranslationValidation, userValidation } from "@local/shared";
import { Box, Stack, TextField } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ProfilePictureInput } from "components/inputs/ProfilePictureInput/ProfilePictureInput";
import { TranslatedRichInput } from "components/inputs/TranslatedRichInput/TranslatedRichInput";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { SessionContext } from "contexts/SessionContext";
import { Field, Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useProfileQuery } from "hooks/useProfileQuery";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { useContext } from "react";
import { useTranslation } from "react-i18next";
import { FormSection } from "styles";
import { toDisplay } from "utils/display/pageTools";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { shapeProfile } from "utils/shape/models/profile";
import { createPrims } from "utils/shape/models/tools";
import { SettingsProfileFormProps, SettingsProfileViewProps } from "../types";

const SettingsProfileForm = ({
    display,
    dirty,
    isLoading,
    numVerifiedWallets,
    onCancel,
    values,
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
                />
                <FormSection sx={{ marginTop: 2 }}>
                    <LanguageInput
                        currentLanguage={language}
                        handleAdd={handleAddLanguage}
                        handleDelete={handleDeleteLanguage}
                        handleCurrent={setLanguage}
                        languages={languages}
                    />
                    <Field fullWidth name="name" label={t("Name")} as={TextField} />
                    <Field fullWidth name="handle" label={t("Handle")} as={TextField} />
                    <TranslatedRichInput
                        language={language}
                        maxChars={2048}
                        minRows={4}
                        name="bio"
                        placeholder={t("Bio")}
                    />
                </FormSection>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                isCreate={false}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </>
    );
};

export const SettingsProfileView = ({
    isOpen,
    onClose,
}: SettingsProfileViewProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();
    const [fetch, { loading: isUpdating }] = useLazyFetch<ProfileUpdateInput, User>(endpointPutProfile);

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={onClose}
                title={t("Profile")}
            />
            <Stack direction="row">
                <SettingsList />
                <Box m="auto" mt={2}>
                    <Formik
                        enableReinitialize={true}
                        initialValues={{
                            bannerImage: profile?.bannerImage ?? null,
                            handle: profile?.handle ?? null,
                            name: profile?.name ?? "",
                            profileImage: profile?.profileImage ?? null,
                            translations: profile?.translations?.length ? profile.translations : [{
                                id: DUMMY_ID,
                                language: getUserLanguages(session)[0],
                                bio: "",
                            }],
                            updated_at: profile?.updated_at ?? null, // Used for cache busting on profile image
                        }}
                        onSubmit={(values, helpers) => {
                            if (!profile) {
                                PubSub.get().publishSnack({ messageKey: "CouldNotReadProfile", severity: "Error" });
                                return;
                            }
                            console.log("submitting profile update: values", values);
                            console.log("submitting profile update: profile", profile);
                            console.log("submitting profile update: shapeProfile.update(profile, values)", shapeProfile.update(profile, {
                                id: profile.id,
                                ...values,
                            }));
                            console.log("test1", createPrims(values, "profileImage"));
                            fetchLazyWrapper<ProfileUpdateInput, User>({
                                fetch,
                                inputs: shapeProfile.update(profile, {
                                    id: profile.id,
                                    ...values,
                                }),
                                successMessage: () => ({ messageKey: "SettingsUpdated" }),
                                onSuccess: (updated) => { onProfileUpdate(updated); },
                                onCompleted: () => { helpers.setSubmitting(false); },
                            });
                        }}
                        validationSchema={userValidation.update({})}
                    >
                        {(formik) => <SettingsProfileForm
                            display={display}
                            isLoading={isProfileLoading || isUpdating}
                            numVerifiedWallets={profile?.wallets?.filter(w => w.verified)?.length ?? 0}
                            onCancel={formik.resetForm}
                            {...formik}
                        />}
                    </Formik>
                </Box>
            </Stack>
        </>
    );
};
