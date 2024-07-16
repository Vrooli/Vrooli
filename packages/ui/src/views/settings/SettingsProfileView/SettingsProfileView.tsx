import { DUMMY_ID, endpointPutProfile, ProfileUpdateInput, profileValidation, User, userTranslationValidation } from "@local/shared";
import { Box, InputAdornment, Stack } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ProfilePictureInput } from "components/inputs/ProfilePictureInput/ProfilePictureInput";
import { TranslatedRichInput } from "components/inputs/RichInput/RichInput";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { SessionContext } from "contexts/SessionContext";
import { Field, Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useProfileQuery } from "hooks/useProfileQuery";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { HandleIcon, UserIcon } from "icons";
import { useContext } from "react";
import { useTranslation } from "react-i18next";
import { FormSection, pagePaddingBottom } from "styles";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { shapeProfile } from "utils/shape/models/profile";
import { SettingsProfileFormProps, SettingsProfileViewProps } from "../types";

function SettingsProfileForm({
    display,
    dirty,
    isLoading,
    numVerifiedWallets,
    onCancel,
    values,
    ...props
}: SettingsProfileFormProps) {
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
        validationSchema: userTranslationValidation.update({ env: process.env.NODE_ENV }),
    });

    return (
        <>
            <BaseForm
                display={display}
                isLoading={isLoading}
                maxWidth={600}
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
                    <Field
                        fullWidth
                        autoComplete="name"
                        name="name"
                        label={t("Name")}
                        placeholder={t("NamePlaceholder")}
                        as={TextInput}
                        InputProps={{
                            startAdornment: (
                                <InputAdornment position="start">
                                    <UserIcon />
                                </InputAdornment>
                            ),
                        }}
                        error={props.touched.name && Boolean(props.errors.name)}
                        helperText={props.touched.name && props.errors.name}
                    />
                    <Field
                        fullWidth
                        autoComplete="handle"
                        name="handle"
                        label={t("Handle")}
                        placeholder={t("HandlePlaceholder")}
                        as={TextInput}
                        InputProps={{
                            startAdornment: (
                                <InputAdornment position="start">
                                    <HandleIcon />
                                </InputAdornment>
                            ),
                        }}
                        error={props.touched.handle && Boolean(props.errors.handle)}
                        helperText={props.touched.handle && props.errors.handle}
                    />
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
}

export function SettingsProfileView({
    display,
    onClose,
}: SettingsProfileViewProps) {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();
    const [fetch, { loading: isUpdating }] = useLazyFetch<ProfileUpdateInput, User>(endpointPutProfile);

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={onClose}
                title={t("Profile")}
            />
            <Stack direction="row" sx={{ paddingBottom: pagePaddingBottom }}>
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
                                PubSub.get().publish("snack", { messageKey: "CouldNotReadProfile", severity: "Error" });
                                return;
                            }
                            fetchLazyWrapper<ProfileUpdateInput, User>({
                                fetch,
                                inputs: shapeProfile.update(profile, {
                                    id: profile.id,
                                    ...values,
                                    __typename: "User",
                                }),
                                successMessage: () => ({ messageKey: "SettingsUpdated" }),
                                onSuccess: (updated) => { onProfileUpdate(updated); },
                                onCompleted: () => { helpers.setSubmitting(false); },
                            });
                        }}
                        validationSchema={profileValidation.update({ env: process.env.NODE_ENV })}
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
}
