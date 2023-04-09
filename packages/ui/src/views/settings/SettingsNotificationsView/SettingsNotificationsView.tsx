import { useQuery } from "@apollo/client";
import { Stack } from "@mui/material";
import { NotificationSettings, NotificationSettingsCategory, NotificationSettingsUpdateInput } from "@shared/consts";
import { mutationWrapper } from "api";
import { notificationSettings } from "api/generated/endpoints/notification_settings";
import { notificationSettingsUpdate } from "api/generated/endpoints/notification_settingsUpdate";
import { useCustomMutation } from "api/hooks";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { Formik } from "formik";
import { SettingsNotificationForm } from "forms/settings/SettingsNotificationsForm/SettingsNotificationsForm";
import { Wrap } from "types";
import { useDisplayApolloError } from "utils/hooks/useDisplayApolloError";
import { SettingsNotificationsViewProps } from "../types";

export const SettingsNotificationsView = ({
    display = 'page',
}: SettingsNotificationsViewProps) => {

    const { data, refetch, loading: isLoading, error } = useQuery<Wrap<NotificationSettings, 'notificationSettings'>>(notificationSettings, { errorPolicy: 'all' });
    useDisplayApolloError(error);
    const [mutation, { loading: isUpdating }] = useCustomMutation<NotificationSettings, NotificationSettingsUpdateInput>(notificationSettingsUpdate);

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    titleKey: 'Notification',
                    titleVariables: { count: 2 }
                }}
            />
            <Stack direction="row">
                <SettingsList />
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
                        mutationWrapper<NotificationSettings, NotificationSettingsUpdateInput>({
                            mutation,
                            input: values,
                            onError: () => { helpers.setSubmitting(false) }
                        })
                    }
                >
                    {(formik) => <SettingsNotificationForm
                        display={display}
                        isLoading={isLoading || isUpdating}
                        onCancel={formik.resetForm}
                        {...formik}
                    />}
                </Formik>
            </Stack>
        </>
    )
}