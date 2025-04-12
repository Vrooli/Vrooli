import { DUMMY_ID, endpointsUser, ProfileUpdateInput, profileValidation, shapeProfile, User, userTranslationValidation } from "@local/shared";
import { InputAdornment } from "@mui/material";
import { Field, Formik, FormikHelpers } from "formik";
import { useCallback, useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { BottomActionsButtons } from "../../components/buttons/BottomActionsButtons.js";
import { LanguageInput } from "../../components/inputs/LanguageInput/LanguageInput.js";
import { ProfilePictureInput } from "../../components/inputs/ProfilePictureInput/ProfilePictureInput.js";
import { TranslatedRichInput } from "../../components/inputs/RichInput/RichInput.js";
import { TextInput } from "../../components/inputs/TextInput/TextInput.js";
import { SettingsList } from "../../components/lists/SettingsList/SettingsList.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { SettingsContent } from "../../components/navigation/SettingsTopBar.js";
import { PageContainer } from "../../components/Page/Page.js";
import { SessionContext } from "../../contexts/session.js";
import { InnerForm, OuterForm } from "../../forms/BaseForm/BaseForm.js";
import { useLazyFetch } from "../../hooks/useLazyFetch.js";
import { useProfileQuery } from "../../hooks/useProfileQuery.js";
import { useTranslatedFields } from "../../hooks/useTranslatedFields.js";
import { IconCommon } from "../../icons/Icons.js";
import { FormSection, ScrollBox } from "../../styles.js";
import { combineErrorsWithTranslations, getUserLanguages } from "../../utils/display/translationTools.js";
import { PubSub } from "../../utils/pubsub.js";
import { SettingsProfileFormProps, SettingsProfileViewProps } from "./types.js";

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
        updated_at: values.updated_at,
    }), [values.id, values.name, values.handle, values.profileImage, values.bannerImage, values.updated_at]);

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
                    <TranslatedRichInput
                        language={language}
                        maxChars={2048}
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
        updated_at: profile?.updated_at ?? null, // Used for cache busting on profile image
    } as const), [profile, session]);

    // Form submission handler
    const handleSubmit = useCallback((values: ProfileUpdateInput, helpers: FormikHelpers<ProfileUpdateInput>) => {
        if (!profile) {
            PubSub.get().publish("snack", { message: t("CouldNotReadProfile", { ns: "error" }), severity: "Error" });
            return;
        }

        const inputs = shapeProfile.update(profile, {
            ...values,
            id: profile.id,
            isPrivate: values.isPrivate ?? undefined,
            isPrivateApis: values.isPrivateApis ?? undefined,
            isPrivateApisCreated: values.isPrivateApisCreated ?? undefined,
            isPrivateBookmarks: values.isPrivateBookmarks ?? undefined,
            isPrivateMemberships: values.isPrivateMemberships ?? undefined,
            isPrivateProjects: values.isPrivateProjects ?? undefined,
            isPrivateProjectsCreated: values.isPrivateProjectsCreated ?? undefined,
            isPrivatePullRequests: values.isPrivatePullRequests ?? undefined,
            isPrivateQuestionsAnswered: values.isPrivateQuestionsAnswered ?? undefined,
            isPrivateQuestionsAsked: values.isPrivateQuestionsAsked ?? undefined,
            isPrivateQuizzesCreated: values.isPrivateQuizzesCreated ?? undefined,
            isPrivateRoles: values.isPrivateRoles ?? undefined,
            isPrivateRoutines: values.isPrivateRoutines ?? undefined,
            isPrivateRoutinesCreated: values.isPrivateRoutinesCreated ?? undefined,
            isPrivateStandards: values.isPrivateStandards ?? undefined,
            isPrivateStandardsCreated: values.isPrivateStandardsCreated ?? undefined,
            isPrivateTeamsCreated: values.isPrivateTeamsCreated ?? undefined,
            isPrivateVotes: values.isPrivateVotes ?? undefined,
            name: values.name ?? undefined,
            theme: values.theme ?? undefined,
            __typename: "User",
        });

        if (!inputs) {
            console.warn("[SettingsProfileView.handleSubmit] No updates detected");
            return;
        }

        inputs.id = profile.id;
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
                    <Formik
                        enableReinitialize={true}
                        initialValues={initialValues}
                        onSubmit={handleSubmit}
                        validationSchema={profileValidation.update({ env: process.env.NODE_ENV })}
                    >
                        {(formik) => (
                            <SettingsProfileForm
                                display={display}
                                isLoading={isProfileLoading || isUpdating}
                                numVerifiedWallets={profile?.wallets?.filter(w => w.verified)?.length ?? 0}
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
