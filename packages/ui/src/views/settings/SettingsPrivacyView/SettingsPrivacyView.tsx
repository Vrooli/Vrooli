import { endpointPutProfile, ProfileUpdateInput, User, userValidation } from "@local/shared";
import { Box, Stack, Typography, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsToggleListItem } from "components/lists/SettingsToggleListItem/SettingsToggleListItem";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useProfileQuery } from "hooks/useProfileQuery";
import { useTranslation } from "react-i18next";
import { pagePaddingBottom } from "styles";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { SettingsPrivacyFormProps, SettingsPrivacyViewProps } from "../types";

const SettingsPrivacyForm = ({
    display,
    dirty,
    isLoading,
    onCancel,
    values,
    ...props
}: SettingsPrivacyFormProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [isPrivateField] = useField<boolean>("isPrivate");

    return (
        <>
            <BaseForm
                dirty={dirty}
                display={display}
                isLoading={isLoading}
            >
                {/* Overall notifications toggle */}
                <ListContainer sx={{ marginBottom: 2 }}>
                    <SettingsToggleListItem
                        title={t("PrivateAccount")}
                        description={t("PushNotificationToggleDescription")}
                        name="isPrivate"
                    />
                </ListContainer>
                {/* By object type */}
                {isPrivateField.value && (
                    <Typography variant="body2" sx={{
                        marginTop: 4,
                        marginBottom: 1,
                        color: palette.warning.main,
                    }}>
                        All of your content is private. Turn off private mode to change specific settings.
                    </Typography>
                )}
                <ListContainer sx={{ marginBottom: 4 }}>
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
                        title={t("PrivateSmartContracts")}
                        name="isPrivateSmartContracts"
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
                errors={props.errors as any}
                isCreate={false}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </>
    );
};


export const SettingsPrivacyView = ({
    isOpen,
    onClose,
}: SettingsPrivacyViewProps) => {
    const { t } = useTranslation();
    const display = toDisplay(isOpen);

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();
    const [fetch, { loading: isUpdating }] = useLazyFetch<ProfileUpdateInput, User>(endpointPutProfile);

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={onClose}
                title={t("Privacy")}
            />
            <Stack direction="row" sx={{ paddingBottom: pagePaddingBottom }}>
                <SettingsList />
                <Box m="auto" mt={2}>
                    <Formik
                        enableReinitialize={true}
                        initialValues={{
                            isPrivate: profile?.isPrivate ?? false,
                            isPrivateApis: profile?.isPrivateApis ?? false,
                            isPrivateBookmarks: profile?.isPrivateBookmarks ?? false,
                            isPrivateProjects: profile?.isPrivateProjects ?? false,
                            isPrivateRoutines: profile?.isPrivateRoutines ?? false,
                            isPrivateSmartContracts: profile?.isPrivateSmartContracts ?? false,
                            isPrivateStandards: profile?.isPrivateStandards ?? false,
                        } as ProfileUpdateInput}
                        onSubmit={(values, helpers) => {
                            if (!profile) {
                                PubSub.get().publishSnack({ messageKey: "CouldNotReadProfile", severity: "Error" });
                                return;
                            }
                            fetchLazyWrapper<ProfileUpdateInput, User>({
                                fetch,
                                inputs: values,
                                successMessage: () => ({ messageKey: "SettingsUpdated" }),
                                onSuccess: (data) => { onProfileUpdate(data); },
                                onCompleted: () => { helpers.setSubmitting(false); },
                            });
                        }}
                        validationSchema={userValidation.update({})}
                    >
                        {(formik) => <SettingsPrivacyForm
                            display={display}
                            isLoading={isProfileLoading || isUpdating}
                            onCancel={formik.resetForm}
                            {...formik}
                        />}
                    </Formik>
                </Box>
            </Stack>
        </>
    );
};
