import { useQuery } from "@apollo/client";
import { NotificationSettings, notificationSettings, NotificationSettingsCategory, notificationSettingsUpdate, NotificationSettingsUpdateInput } from "@local/shared";
import { Stack } from "@mui/material";
import { mutationWrapper, useCustomMutation } from "api";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { Formik } from "formik";
import { SettingsNotificationForm } from "forms/settings/SettingsNotificationsForm/SettingsNotificationsForm";
import { Wrap } from "types";
import { useDisplayServerError } from "utils/hooks/useDisplayServerError";
import { SettingsNotificationsViewProps } from "../types";

export const SettingsNotificationsView = ({
    display = "page",
    onClose,
    zIndex,
}: SettingsNotificationsViewProps) => {

    const { data, refetch, loading: isLoading, error } = useQuery<Wrap<NotificationSettings, "notificationSettings">>(notificationSettings, { errorPolicy: "all" });
    useDisplayServerError(error);
    const [mutation, { loading: isUpdating }] = useCustomMutation<NotificationSettings, NotificationSettingsUpdateInput>(notificationSettingsUpdate);

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={onClose}
                titleData={{
                    titleKey: "Notification",
                    titleVariables: { count: 2 },
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
                            onError: () => { helpers.setSubmitting(false); },
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
            </Stack>
        </>
    );
};
