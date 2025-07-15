import { InputAdornment } from "@mui/material";
import { DUMMY_ID, endpointsUser, profileValidation, shapeProfile, userTranslationValidation, type ProfileUpdateInput, type User } from "@vrooli/shared";
import { Field, Formik, type FormikHelpers } from "formik";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { PageContainer } from "../../components/Page/Page.js";
import { BottomActionsButtons } from "../../components/buttons/BottomActionsButtons.js";
import { TranslatedAdvancedInput } from "../../components/inputs/AdvancedInput/AdvancedInput.js";
import { LanguageInput } from "../../components/inputs/LanguageInput/LanguageInput.js";
import { ProfilePictureInput } from "../../components/inputs/ProfilePictureInput/ProfilePictureInput.js";
import { TextInput } from "../../components/inputs/TextInput/TextInput.js";
import { SettingsList } from "../../components/lists/SettingsList/SettingsList.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { SettingsContent } from "../../components/navigation/SettingsTopBar.js";
import { SessionContext } from "../../contexts/session.js";
import { InnerForm, OuterForm } from "../../forms/BaseForm/BaseForm.js";
import { useLazyFetch } from "../../hooks/useFetch.js";
import { useProfileQuery } from "../../hooks/useProfileQuery.js";
import { useTranslatedFields } from "../../hooks/useTranslatedFields.js";
import { IconCommon } from "../../icons/Icons.js";
import { FormSection, ScrollBox } from "../../styles.js";
import { ViewDisplayType } from "../../types.js";
import { combineErrorsWithTranslations, getUserLanguages } from "../../utils/display/translationTools.js";
import { PubSub } from "../../utils/pubsub.js";
import { type SettingsProfileFormInput, type SettingsProfileFormProps, type SettingsProfileViewProps } from "./types.js";

// Input adornments for form fields
const nameInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon name="User" />
        </InputAdornment>
    ),
} as const;

const handleInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon name="Handle" />
        </InputAdornment>
    ),
} as const;

/**
 * Form component for user profile settings
 */
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

    // Image change handlers
    const handleBannerImageChange = useCallback((newPicture: File | null) => {
        setFieldValue("bannerImage", newPicture);
    }, [setFieldValue]);

    const handleProfileImageChange = useCallback((newPicture: File | null) => {
        setFieldValue("profileImage", newPicture);
    }, [setFieldValue]);

    // Create memoized profile object for ProfilePictureInput
    const profileData = useMemo(() => ({
        __typename: "User" as const,
        id: values.id,
        name: values.name,
        handle: values.handle,
        profileImage: values.profileImage,
        bannerImage: values.bannerImage,
    }), [values.id, values.name, values.handle, values.profileImage, values.bannerImage]);

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
                    profile={profileData}
                />
                <FormSection marginTop={2}>
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
                    <TranslatedAdvancedInput
                        language={language}
                        features={{ maxChars: 2048 }}
                        name="bio"
                        placeholder={t("Bio")}
                    />
                    <LanguageInput
                        currentLanguage={language}
                        handleAdd={handleAddLanguage}
                        handleDelete={handleDeleteLanguage}
                        handleCurrent={setLanguage}
                        languages={languages}
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

/**
 * Settings view for user profile
 */
export function SettingsProfileView({
    display,
}: SettingsProfileViewProps) {
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();
    const [fetch, { loading: isUpdating }] = useLazyFetch<ProfileUpdateInput, User>(endpointsUser.profileUpdate);

    // Initial form values
    const initialValues = useMemo(() => ({
        bannerImage: profile?.bannerImage ?? null,
        handle: profile?.handle ?? null,
        id: profile?.id ?? "",
        name: profile?.name ?? "",
        profileImage: profile?.profileImage ?? null,
        translations: profile?.translations?.length ? profile.translations : [{
            id: DUMMY_ID,
            language: getUserLanguages(session)[0],
            bio: "",
        }],
        updatedAt: profile?.updatedAt ?? null, // Used for cache busting on profile image
    }), [profile, session]);

    // Form submission handler
    const handleSubmit = useCallback((values: SettingsProfileFormInput, helpers: FormikHelpers<SettingsProfileFormInput>) => {
        if (!profile) {
            PubSub.get().publish("snack", { message: t("CouldNotReadProfile", { ns: "error" }), severity: "Error" });
            return;
        }

        const inputs = shapeProfile.update(profile, {
            ...values,
            id: profile.id,
            __typename: "User",
        });

        if (!inputs) {
            console.warn("[SettingsProfileView.handleSubmit] No updates detected");
            return;
        }

        fetchLazyWrapper<ProfileUpdateInput, User>({
            fetch,
            inputs,
            successMessage: () => ({ message: t("SettingsUpdated") }),
            onSuccess: (updated) => { onProfileUpdate(updated); },
            onCompleted: () => { helpers.setSubmitting(false); },
        });
    }, [profile, fetch, onProfileUpdate, t]);

    return (
        <PageContainer size="fullSize">
            <ScrollBox>
                <Navbar title={t("Profile")} />
                <SettingsContent>
                    <SettingsList />
                    <Formik<SettingsProfileFormInput>
                        enableReinitialize={true}
                        initialValues={initialValues}
                        onSubmit={handleSubmit}
                        validationSchema={profileValidation.update({ env: process.env.NODE_ENV })}
                    >
                        {(formik) => (
                            <SettingsProfileForm
                                display={display}
                                isLoading={isProfileLoading || isUpdating}
                                onCancel={formik.resetForm}
                                {...formik}
                            />
                        )}
                    </Formik>
                </SettingsContent>
            </ScrollBox>
        </PageContainer>
    );
}
