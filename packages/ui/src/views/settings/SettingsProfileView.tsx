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
import { SettingsContent, SettingsTopBar } from "../../components/navigation/SettingsTopBar.js";
import { SessionContext } from "../../contexts.js";
import { InnerForm, OuterForm } from "../../forms/BaseForm/BaseForm.js";
import { useLazyFetch } from "../../hooks/useLazyFetch.js";
import { useProfileQuery } from "../../hooks/useProfileQuery.js";
import { useTranslatedFields } from "../../hooks/useTranslatedFields.js";
import { IconCommon } from "../../icons/Icons.js";
import { FormSection, ScrollBox } from "../../styles.js";
import { combineErrorsWithTranslations, getUserLanguages } from "../../utils/display/translationTools.js";
import { PubSub } from "../../utils/pubsub.js";
import { SettingsProfileFormProps, SettingsProfileViewProps } from "./types.js";

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
                <FormSection marginTop={2}>
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
    const [fetch, { loading: isUpdating }] = useLazyFetch<ProfileUpdateInput, User>(endpointsUser.profileUpdate);

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

    const handleSubmit = useCallback(function handleSubmitCallback(values: ProfileUpdateInput, helpers: FormikHelpers<ProfileUpdateInput>) {
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
    }, [profile, fetch, onProfileUpdate]);

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
                    onSubmit={handleSubmit}
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
