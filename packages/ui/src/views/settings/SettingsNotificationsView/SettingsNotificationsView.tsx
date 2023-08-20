import { endpointGetNotificationSettings, endpointPutNotificationSettings, NotificationSettings, NotificationSettingsCategory, NotificationSettingsUpdateInput } from "@local/shared";
import { Box, Stack } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { Formik } from "formik";
import { SettingsNotificationForm } from "forms/settings/SettingsNotificationsForm/SettingsNotificationsForm";
import { useDisplayServerError } from "hooks/useDisplayServerError";
import { useFetch } from "hooks/useFetch";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useTranslation } from "react-i18next";
import { toDisplay } from "utils/display/pageTools";
import { SettingsNotificationsViewProps } from "../types";

export const SettingsNotificationsView = ({
    isOpen,
    onClose,
    zIndex,
}: SettingsNotificationsViewProps) => {
    const { t } = useTranslation();
    const display = toDisplay(isOpen);

    const { data, refetch, loading: isLoading, errors } = useFetch<undefined, NotificationSettings>({
        ...endpointGetNotificationSettings,
    });
    useDisplayServerError(errors);
    const [updateFetch, { loading: isUpdating }] = useLazyFetch<NotificationSettingsUpdateInput, NotificationSettings>(endpointPutNotificationSettings);

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={onClose}
                title={t("Notification", { count: 2 })}
                zIndex={zIndex}
            />
            <Stack direction="row">
                <SettingsList />
                <Box m="auto" mt={2}>
                    <Formik
                        enableReinitialize={true}
                        initialValues={{
                            includedEmails: [] as string[],
                            includedSms: [] as string[],
                            includedPush: [] as string[],
                            toEmails: false,
                            toSms: false,
                            toPush: false,
                            dailyLimit: 0,
                            enabled: true,
                            categories: [] as NotificationSettingsCategory[],
                        } as NotificationSettingsUpdateInput}
                        onSubmit={(values, helpers) =>
                            fetchLazyWrapper<NotificationSettingsUpdateInput, NotificationSettings>({
                                fetch: updateFetch,
                                inputs: values,
                                onCompleted: () => { helpers.setSubmitting(false); },
                            })
                        }
                    >
                        {(formik) => <SettingsNotificationForm
                            display={display}
                            isLoading={isLoading || isUpdating}
                            onCancel={formik.resetForm}
                            zIndex={zIndex}
                            {...formik}
                        />}
                    </Formik>
                </Box>
            </Stack>
        </>
    );
};
