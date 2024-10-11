import { DUMMY_ID, endpointPutProfile, ProfileUpdateInput, profileValidation, shapeProfile, User, userTranslationValidation } from "@local/shared";
import { InputAdornment } from "@mui/material";
import { fetchLazyWrapper } from "api/fetchWrapper";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { ProfilePictureInput } from "components/inputs/ProfilePictureInput/ProfilePictureInput";
import { TranslatedRichInput } from "components/inputs/RichInput/RichInput";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsContent, SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { SessionContext } from "contexts";
import { Field, Formik } from "formik";
import { InnerForm, OuterForm } from "forms/BaseForm/BaseForm";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useProfileQuery } from "hooks/useProfileQuery";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { HandleIcon, UserIcon } from "icons";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { FormSection, ScrollBox } from "styles";
import { combineErrorsWithTranslations, getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { SettingsProfileFormProps, SettingsProfileViewProps } from "../types";

const nameInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <UserIcon />
        </InputAdornment>
    ),
} as const;
const handleInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <HandleIcon />
        </InputAdornment>
    ),
} as const;

function SettingsProfileForm({
    display,
    isLoading,
    onCancel,
    setFieldValue,
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

    const handleBannerImageChange = useCallback(function handleBannerImageChangeCallback(newPicture: File | null) {
        setFieldValue("bannerImage", newPicture);
    }, [setFieldValue]);
    const handleProfileImageChange = useCallback(function handleProfileImageChangeCallback(newPicture: File | null) {
        setFieldValue("profileImage", newPicture);
    }, [setFieldValue]);

    return (
        <OuterForm display={display}>
            <InnerForm
                display={display}
                isLoading={isLoading}
            >
                <ProfilePictureInput
                    onBannerImageChange={handleBannerImageChange}
                    onProfileImageChange={handleProfileImageChange}
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
                        InputProps={nameInputProps}
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
                        InputProps={handleInputProps}
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
            </InnerForm>
            <BottomActionsButtons
                display={display}
                errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                isCreate={false}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </OuterForm>
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

    const initialValues = useMemo(function initialValuesMemo() {
        return {
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
        } as const;
    }, [profile, session]);

    return (
        <ScrollBox>
            <SettingsTopBar
                display={display}
                onClose={onClose}
                title={t("Profile")}
            />
            <SettingsContent>
                <SettingsList />
                <Formik
                    enableReinitialize={true}
                    initialValues={initialValues}
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
            </SettingsContent>
        </ScrollBox>
    );
}
