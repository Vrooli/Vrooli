import { endpointsUser, ProfileUpdateInput, profileValidation, User } from "@local/shared";
import { Box, styled, Typography } from "@mui/material";
import { Formik, FormikHelpers, FormikProps, useField } from "formik";
import { useCallback, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { ListContainer } from "../../components/containers/ListContainer.js";
import { SettingsList } from "../../components/lists/SettingsList/SettingsList.js";
import { SettingsToggleListItem } from "../../components/lists/SettingsToggleListItem/SettingsToggleListItem.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { SettingsContent } from "../../components/navigation/SettingsTopBar.js";
import { BaseForm } from "../../forms/BaseForm/BaseForm.js";
import { useAutoSave } from "../../hooks/useAutoSave.js";
import { useLazyFetch } from "../../hooks/useFetch.js";
import { useProfileQuery } from "../../hooks/useProfileQuery.js";
import { ScrollBox } from "../../styles.js";
import { PubSub } from "../../utils/pubsub.js";
import { SettingsPrivacyFormProps, SettingsPrivacyViewProps } from "./types.js";

const PrivateWarningText = styled(Typography)(({ theme }) => ({
    // eslint-disable-next-line no-magic-numbers
    marginBottom: theme.spacing(4),
    color: theme.palette.warning.main,
    fontStyle: "italic",
}));

const mainListContainerStyle = { marginBottom: 2 } as const;
const categoriesListContainerStyle = { marginBottom: 4 } as const;
const outerFormStyle = { width: "100%" } as const;

function SettingsPrivacyForm({
    display,
    isLoading,
    onCancel,
    ...props
}: SettingsPrivacyFormProps) {
    const { t } = useTranslation();

    const [isPrivateField] = useField<boolean>("isPrivate");

    return (
        <Box sx={outerFormStyle}>
            <BaseForm
                display={display}
                isLoading={isLoading}
            >
                {/* Overall notifications toggle */}
                <ListContainer sx={mainListContainerStyle}>
                    <SettingsToggleListItem
                        title={t("PrivateAccount")}
                        description={t("PushNotificationToggleDescription")}
                        name="isPrivate"
                    />
                </ListContainer>
                {/* By object type */}
                {isPrivateField.value && (
                    <PrivateWarningText variant="body2">
                        All of your content is private. Turn off private mode to change specific settings.
                    </PrivateWarningText>
                )}
                <ListContainer sx={categoriesListContainerStyle}>
                    <SettingsToggleListItem
                        disabled={isPrivateField.value}
                        title={t("PrivateBookmarks")}
                        name="isPrivateBookmarks"
                    />
                    <SettingsToggleListItem
                        disabled={isPrivateField.value}
                        title={t("PrivateResources")}
                        name="isPrivateResources"
                    />
                </ListContainer>
            </BaseForm>
        </Box>
    );
}


export function SettingsPrivacyView({
    display,
}: SettingsPrivacyViewProps) {
    const { t } = useTranslation();
    const formikRef = useRef<FormikProps<ProfileUpdateInput>>(null);

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();
    const [fetch, { loading: isUpdating }] = useLazyFetch<ProfileUpdateInput, User>(endpointsUser.profileUpdate);

    const initialValues = useMemo<ProfileUpdateInput>(function initialValuesMemo() {
        if (!profile) {
            return {
                id: "",
                isPrivate: false,
                isPrivateBookmarks: false,
                isPrivateResources: false,
            };
        }

        return {
            id: profile.id,
            isPrivate: profile.isPrivate ?? false,
            isPrivateBookmarks: profile.isPrivateBookmarks ?? false,
            isPrivateResources: profile.isPrivateResources ?? false,
        };
    }, [profile]);

    const handleSubmit = useCallback(function handleSubmitCallback(values: ProfileUpdateInput, helpers: FormikHelpers<ProfileUpdateInput>) {
        if (!profile) {
            PubSub.get().publish("snack", { message: t("CouldNotReadProfile", { ns: "error" }), severity: "Error" });
            return;
        }
        const inputs = { ...values, id: profile.id };
        fetchLazyWrapper<ProfileUpdateInput, User>({
            fetch,
            inputs,
            onSuccess: (data) => { onProfileUpdate(data); },
            onCompleted: () => { helpers.setSubmitting(false); },
        });
    }, [fetch, onProfileUpdate, profile, t]);

    const handleSave = useCallback(() => {
        if (formikRef.current) {
            formikRef.current.submitForm();
        }
    }, []);

    useAutoSave<ProfileUpdateInput>({
        formikRef,
        handleSave,
    });

    return (
        <ScrollBox>
            <Navbar title={t("Privacy")} />
            <SettingsContent>
                <SettingsList />
                <Formik
                    enableReinitialize={true}
                    initialValues={initialValues}
                    onSubmit={handleSubmit}
                    validationSchema={profileValidation.update({ env: process.env.NODE_ENV })}
                    innerRef={formikRef}
                >
                    {(formik) => <SettingsPrivacyForm
                        display={display}
                        isLoading={isProfileLoading || isUpdating}
                        onCancel={formik.resetForm}
                        {...formik}
                    />}
                </Formik>
            </SettingsContent>
        </ScrollBox>
    );
}
