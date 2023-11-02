import { endpointGetNotificationSettings, endpointPutNotificationSettings, NotificationSettings, NotificationSettingsCategory, NotificationSettingsUpdateInput } from "@local/shared";
import { Box, Stack } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { IntegerInput } from "components/inputs/IntegerInput/IntegerInput";
import { PushList } from "components/lists/devices";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsToggleListItem } from "components/lists/SettingsToggleListItem/SettingsToggleListItem";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { Title } from "components/text/Title/Title";
import { Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useFetch } from "hooks/useFetch";
import { useLazyFetch } from "hooks/useLazyFetch";
import { EmailIcon, PhoneIcon } from "icons";
import { useTranslation } from "react-i18next";
import { pagePaddingBottom } from "styles";
import { toDisplay } from "utils/display/pageTools";
import { SettingsNotificationFormProps, SettingsNotificationsViewProps } from "../types";

const SettingsNotificationForm = ({
    display,
    dirty,
    isLoading,
    onCancel,
    values,
    ...props
}: SettingsNotificationFormProps) => {
    const { t } = useTranslation();

    return (
        <>
            <BaseForm
                display={display}
                isLoading={isLoading}
            >
                <Stack direction="column" spacing={4}>
                    {/* Overall notifications toggle */}
                    <ListContainer>
                        <SettingsToggleListItem
                            title={t("Notification", { count: 2 })}
                            description={t("PushNotificationToggleDescription")}
                            name="enabled"
                        />
                    </ListContainer>
                    {/* Daily limit input */}
                    <IntegerInput
                        disabled={!values.enabled}
                        label={t("DailyLimit")}
                        min={0}
                        name="dailyLimit"
                    // tooltip={t('DailyLimitNotificationDescription')}
                    />
                    {/* Push notifications toggle */}
                    <ListContainer>
                        <SettingsToggleListItem
                            title={t("PushNotification", { count: 2 })}
                            description={t("PushNotificationToggleDescription")}
                            disabled={!values.enabled}
                            name="toPush"
                        />
                    </ListContainer>
                    {/* Push Device list */}
                    <Title
                        Icon={PhoneIcon}
                        title={t("Device", { count: 2 })}
                        variant="subheader"
                    />
                    <PushList
                        handleUpdate={() => { }}
                        list={[]}
                    />
                    {/* Email notifications toggle */}
                    <ListContainer>
                        <SettingsToggleListItem
                            title={t("EmailNotification", { count: 2 })}
                            description={t("EmailNotificationToggleDescription")}
                            disabled={!values.enabled}
                            name="toEmails"
                        />
                    </ListContainer>
                    {/* Email list */}
                    <Title
                        Icon={EmailIcon}
                        title={t("Email", { count: 2 })}
                        variant="subheader"
                    />
                    {/* <EmailList
                handleUpdate={updateEmails}
                list={profile?.emails ?? []}
                numVerifiedWallets={numVerifiedWallets}
            /> */}
                    {/* Toggle individual categories */}
                    {/* TODO */}
                </Stack>
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
        </>
    );
};

export const SettingsNotificationsView = ({
    isOpen,
    onClose,
}: SettingsNotificationsViewProps) => {
    const { t } = useTranslation();
    const display = toDisplay(isOpen);

    const { data, refetch, loading: isLoading } = useFetch<undefined, NotificationSettings>(endpointGetNotificationSettings);
    const [updateFetch, { loading: isUpdating }] = useLazyFetch<NotificationSettingsUpdateInput, NotificationSettings>(endpointPutNotificationSettings);

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={onClose}
                title={t("Notification", { count: 2 })}
            />
            <Stack direction="row" sx={{ paddingBottom: pagePaddingBottom }}>
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
                            {...formik}
                        />}
                    </Formik>
                </Box>
            </Stack>
        </>
    );
};
