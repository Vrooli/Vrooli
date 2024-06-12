import { Email, endpointGetNotificationSettings, endpointPutNotificationSettings, NotificationSettings, NotificationSettingsCategory, NotificationSettingsUpdateInput, PushDevice } from "@local/shared";
import { Box, Button, Divider, Stack } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { IntegerInput } from "components/inputs/IntegerInput/IntegerInput";
import { EmailList } from "components/lists/devices/EmailList/EmailList";
import { PushList } from "components/lists/devices/PushList/PushList";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsToggleListItem } from "components/lists/SettingsToggleListItem/SettingsToggleListItem";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { Title } from "components/text/Title/Title";
import { Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useFetch } from "hooks/useFetch";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useProfileQuery } from "hooks/useProfileQuery";
import { EmailIcon, NotificationsAllIcon, PhoneIcon } from "icons";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { pagePaddingBottom } from "styles";
import { PubSub } from "utils/pubsub";
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

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();

    const numVerifiedPhones = profile?.phones?.filter((phone) => phone.verified)?.length ?? 0;
    const numVerifiedWallets = profile?.wallets?.filter((wallet) => wallet.verified)?.length ?? 0;

    const updateEmails = useCallback((updatedList: Email[]) => {
        if (!profile) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadProfile", severity: "Error" });
            return;
        }
        onProfileUpdate({ ...profile, emails: updatedList });
    }, [onProfileUpdate, profile]);

    const updatePushDevices = useCallback((updatedList: PushDevice[]) => {
        if (!profile) {
            PubSub.get().publish("snack", { messageKey: "CouldNotReadProfile", severity: "Error" });
            return;
        }
        onProfileUpdate({ ...profile, pushDevices: updatedList });
    }, [onProfileUpdate, profile]);

    //TODO toggle for individual categories
    return (
        <BaseForm
            display={display}
            isLoading={isLoading}
        >
            <Stack
                direction="column"
                spacing={4}
                m="auto"
                pl={2}
                pr={2}
                sx={{
                    maxWidth: "min(100%, 700px)",
                    width: "100%",
                }}
            >
                <Box>
                    <Title
                        Icon={NotificationsAllIcon}
                        title={t("Notification", { count: 2 })}
                        variant="subheader"
                        sxs={{ stack: { marginBottom: 2 } }}
                    />
                    <ListContainer sx={{ marginBottom: 6 }}>
                        <SettingsToggleListItem
                            title={t("Notification", { count: 2 })}
                            description={t("PushNotificationToggleDescription")}
                            name="enabled"
                        />
                    </ListContainer>
                    <Title
                        title={"Daily Notification Limit"}
                        variant="subsection"
                        help={"Limit the number of notifications you receive each day. This limit applies to all notifications, across all categories."}
                    />
                    {/* <IntegerInput
                        disabled={!values.enabled}
                        label={t("DailyLimit")}
                        min={0}
                        name="dailyLimit"
                        sx={{
                            justifyContent: "flex-start",
                            marginLeft: 1,
                            marginRight: 1,
                        }}
                    /> */}
                    <Box sx={{ display: "flex", alignItems: "center", marginBottom: 4 }}>
                        <IntegerInput
                            disabled={!values.enabled}
                            label={t("DailyLimit")}
                            min={0}
                            name="dailyLimit"
                            sx={{
                                justifyContent: "flex-start",
                                marginLeft: 1,
                                marginRight: 1,
                                width: "auto", // ensure the input doesn't stretch
                                flexGrow: 1,
                            }}
                        />
                        <Button
                            variant="outlined"
                            sx={{ ml: 2 }}
                            onClick={() => { /* Logic to disable the daily limit */ }}
                            disabled={!values.enabled}
                        >
                            Disable Limit
                        </Button>
                    </Box>
                </Box>
                <Divider />
                <Box>
                    <Title
                        Icon={PhoneIcon}
                        title={t("Device", { count: 2 })}
                        variant="subheader"
                        sxs={{ stack: { marginBottom: 2 } }}
                    />
                    <PushList
                        handleUpdate={updatePushDevices}
                        list={profile?.pushDevices ?? []}
                    />
                    <ListContainer>
                        <SettingsToggleListItem
                            title={t("PushNotification", { count: 2 })}
                            description={t("PushNotificationToggleDescription")}
                            disabled={!values.enabled}
                            name="toPush"
                        />
                    </ListContainer>
                </Box>
                <Divider />
                <Box>
                    <Title
                        help={t("EmailListHelp")}
                        Icon={EmailIcon}
                        title={t("Email", { count: 2 })}
                        variant="subheader"
                        sxs={{ stack: { marginBottom: 2 } }}
                    />
                    <EmailList
                        handleUpdate={updateEmails}
                        list={profile?.emails ?? []}
                        numOtherVerified={numVerifiedPhones + numVerifiedWallets}
                    />
                    <ListContainer sx={{ marginTop: 4 }}>
                        <SettingsToggleListItem
                            title={t("EmailNotification", { count: 2 })}
                            description={t("EmailNotificationToggleDescription")}
                            disabled={!values.enabled}
                            name="toEmails"
                        />
                    </ListContainer>
                </Box>
                <BottomActionsButtons
                    display={display}
                    errors={props.errors}
                    isCreate={false}
                    loading={props.isSubmitting}
                    onCancel={onCancel}
                    onSetSubmitting={props.setSubmitting}
                    onSubmit={props.handleSubmit}
                />
            </Stack>
        </BaseForm>
    );
};

export const SettingsNotificationsView = ({
    display,
    onClose,
}: SettingsNotificationsViewProps) => {
    const { t } = useTranslation();

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
            </Stack>
        </>
    );
};
