import { Stack } from "@mui/material"
import { SettingsNotificationsViewProps } from "../types";
import { useDisplayApolloError, usePromptBeforeUnload } from "utils";
import { useCustomMutation } from "api/hooks";
import { useFormik } from "formik";
import { NotificationSettings, NotificationSettingsCategory, NotificationSettingsUpdateInput } from "@shared/consts";
import { userValidation } from "@shared/validation";
import { BaseForm } from "forms";
import { ListContainer, SettingsList, SettingsToggleListItem, SettingsTopBar } from "components";
import { useTranslation } from "react-i18next";
import { notificationSettingsUpdate } from "api/generated/endpoints/notification_settingsUpdate";
import { notificationSettings } from "api/generated/endpoints/notification_settings";
import { useQuery } from "@apollo/client";
import { Wrap } from "types";
import { mutationWrapper } from "api";

export const SettingsNotificationsView = ({
    display = 'page',
    session,
}: SettingsNotificationsViewProps) => {
    const { t } = useTranslation();

    const { data, refetch, loading: isLoading, error } = useQuery<Wrap<NotificationSettings, 'notificationSettings'>>(notificationSettings, { errorPolicy: 'all' });
    useDisplayApolloError(error);

    // Handle update
    const [mutation, { loading: isUpdating }] = useCustomMutation<NotificationSettings, NotificationSettingsUpdateInput>(notificationSettingsUpdate);
    const formik = useFormik({
        initialValues: {
            includedEmails: [] as string[],
            includedSms: [] as string[],
            includedPush: [] as string[],
            toEmails: false,
            toSms: false,
            toPush: false,
            dailyLimit: 0,
            enabled: false,
            categories: [] as NotificationSettingsCategory[],
        },
        enableReinitialize: true,
        validationSchema: userValidation.update({}),
        onSubmit: (values) => {
            mutationWrapper<NotificationSettings, NotificationSettingsUpdateInput>({
                mutation,
                input: values,
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={() => { }}
                session={session}
                titleData={{
                    titleKey: 'Notification',
                    titleVariables: { count: 2 }
                }}
            />
            <Stack direction="row">
                <SettingsList />
                <BaseForm
                    isLoading={isLoading || isUpdating}
                    onSubmit={formik.handleSubmit}
                    style={{
                        width: { xs: '100%', md: 'min(100%, 700px)' },
                        margin: 'auto',
                        display: 'block',
                    }}
                >
                    {/* Overall notifications toggle */}
                    <ListContainer>
                        <SettingsToggleListItem
                            title={t('PushNotification', { count: 2 })}
                            description={t('PushNotificationSettingsDescription')}
                            checked={true}
                            onChange={() => { }}
                        />
                    </ListContainer>
                </BaseForm>
            </Stack>
        </>
    )
}