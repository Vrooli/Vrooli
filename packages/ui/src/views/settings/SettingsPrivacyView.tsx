import { endpointsUser, ProfileUpdateInput, profileValidation, User } from "@local/shared";
import { Box, styled, Typography } from "@mui/material";
import { Formik, FormikHelpers, useField } from "formik";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { BottomActionsButtons } from "../../components/buttons/BottomActionsButtons.js";
import { ListContainer } from "../../components/containers/ListContainer/ListContainer.js";
import { SettingsList } from "../../components/lists/SettingsList/SettingsList.js";
import { SettingsToggleListItem } from "../../components/lists/SettingsToggleListItem/SettingsToggleListItem.js";
import { SettingsContent, SettingsTopBar } from "../../components/navigation/SettingsTopBar.js";
import { BaseForm } from "../../forms/BaseForm/BaseForm.js";
import { useLazyFetch } from "../../hooks/useLazyFetch.js";
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
                        title={t("PrivateApis")}
                        name="isPrivateApis"
                    />
                    <SettingsToggleListItem
                        disabled={isPrivateField.value}
                        title={t("PrivateBookmarks")}
                        name="isPrivateBookmarks"
                    />
                    <SettingsToggleListItem
                        disabled={isPrivateField.value}
                        title={t("PrivateCodes")}
                        name="isPrivateCodes"
                    />
                    <SettingsToggleListItem
                        disabled={isPrivateField.value}
                        title={t("PrivateProjects")}
                        name="isPrivateProjects"
                    />
                    <SettingsToggleListItem
                        disabled={isPrivateField.value}
                        title={t("PrivateRoutines")}
                        name="isPrivateRoutines"
                    />
                    <SettingsToggleListItem
                        disabled={isPrivateField.value}
                        title={t("PrivateStandards")}
                        name="isPrivateStandards"
                    />
                </ListContainer>
            </BaseForm>
            <BottomActionsButtons
                display={display}
                errors={props.errors}
                isCreate={false}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </Box>
    );
}


export function SettingsPrivacyView({
    display,
    onClose,
}: SettingsPrivacyViewProps) {
    const { t } = useTranslation();

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();
    const [fetch, { loading: isUpdating }] = useLazyFetch<ProfileUpdateInput, User>(endpointsUser.profileUpdate);

    const initialValues = useMemo<ProfileUpdateInput>(function initialValuesMemo() {
        return {
            isPrivate: profile?.isPrivate ?? false,
            isPrivateApis: profile?.isPrivateApis ?? false,
            isPrivateBookmarks: profile?.isPrivateBookmarks ?? false,
            isPrivateCodes: profile?.isPrivateCodes ?? false,
            isPrivateProjects: profile?.isPrivateProjects ?? false,
            isPrivateRoutines: profile?.isPrivateRoutines ?? false,
            isPrivateStandards: profile?.isPrivateStandards ?? false,
        };
    }, [profile?.isPrivate, profile?.isPrivateApis, profile?.isPrivateBookmarks, profile?.isPrivateCodes, profile?.isPrivateProjects, profile?.isPrivateRoutines, profile?.isPrivateStandards]);

    const handleSubmit = useCallback(function handleSubmitCallback(values: ProfileUpdateInput, helpers: FormikHelpers<ProfileUpdateInput>) {
        if (!profile) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadProfile", severity: "Error" });
            return;
        }
        fetchLazyWrapper<ProfileUpdateInput, User>({
            fetch,
            inputs: values,
            successMessage: () => ({ messageKey: "SettingsUpdated" }),
            onSuccess: (data) => { onProfileUpdate(data); },
            onCompleted: () => { helpers.setSubmitting(false); },
        });
    }, [fetch, onProfileUpdate, profile]);

    return (
        <ScrollBox>
            <SettingsTopBar
                display={display}
                onClose={onClose}
                title={t("Privacy")}
            />
            <SettingsContent>
                <SettingsList />
                <Formik
                    enableReinitialize={true}
                    initialValues={initialValues}
                    onSubmit={handleSubmit}
                    validationSchema={profileValidation.update({ env: process.env.NODE_ENV })}
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
