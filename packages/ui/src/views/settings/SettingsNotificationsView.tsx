// AI_CHECK: TYPE_SAFETY=fixed-mui-component-jsx-errors | LAST: 2025-06-30
import { Box } from "../../components/layout/Box.js";
import { Button } from "../../components/buttons/Button.js";
import { Divider } from "../../components/layout/Divider.js";
import { Stack } from "../../components/layout/Stack.js";
import { endpointsNotification, type Email, type NotificationSettings, type NotificationSettingsCategory, type NotificationSettingsUpdateInput, type PushDevice } from "@vrooli/shared";
import { Formik } from "formik";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { BottomActionsButtons } from "../../components/buttons/BottomActionsButtons.js";
import { ListContainer } from "../../components/containers/ListContainer.js";
import { IntegerInput } from "../../components/inputs/IntegerInput/IntegerInput.js";
import { SettingsList } from "../../components/lists/SettingsList/SettingsList.js";
import { SettingsToggleListItem } from "../../components/lists/SettingsToggleListItem/SettingsToggleListItem.js";
import { EmailList } from "../../components/lists/devices/EmailList.js";
import { PushList } from "../../components/lists/devices/PushList.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { SettingsContent } from "../../components/navigation/SettingsTopBar.js";
import { Title } from "../../components/text/Title.js";
import { BaseForm } from "../../forms/BaseForm/BaseForm.js";
import { useFetch, useLazyFetch } from "../../hooks/useFetch.js";
import { useProfileQuery } from "../../hooks/useProfileQuery.js";
import { ScrollBox } from "../../styles.js";
import { PubSub } from "../../utils/pubsub.js";
import { type SettingsNotificationFormProps, type SettingsNotificationsViewProps } from "./types.js";

const notificationsAllIconInfo = { name: "NotificationsAll", type: "Common" } as const;
const phoneIconInfo = { name: "Phone", type: "Common" } as const;
const emailIconInfo = { name: "Email", type: "Common" } as const;

function SettingsNotificationForm({
    display,
    dirty,
    isLoading,
    onCancel,
    values,
    ...props
}: SettingsNotificationFormProps) {
    const { t } = useTranslation();

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();

    const numVerifiedPhones = profile?.phones?.filter((phone) => phone.verified)?.length ?? 0;
    const numVerifiedWallets = profile?.wallets?.filter((wallet) => wallet.verified)?.length ?? 0;

    const updateEmails = useCallback((updatedList: Email[]) => {
        if (!profile) {
            PubSub.get().publish("snack", { message: t("CouldNotReadProfile", { ns: "error" }), severity: "Error" });
            return;
        }
        onProfileUpdate({ ...profile, emails: updatedList });
    }, [onProfileUpdate, profile, t]);

    const updatePushDevices = useCallback((updatedList: PushDevice[]) => {
        if (!profile) {
            PubSub.get().publish("snack", { message: t("CouldNotReadProfile", { ns: "error" }), severity: "Error" });
            return;
        }
        onProfileUpdate({ ...profile, pushDevices: updatedList });
    }, [onProfileUpdate, profile, t]);

    //TODO toggle for individual categories
    return (
        <BaseForm
            display={display}
            isLoading={isLoading}
        >
            <Stack
                direction="column"
                spacing="lg"
                className="tw-mx-auto tw-pl-2 tw-pr-2 tw-max-w-full tw-w-full"
                style={{ maxWidth: "min(100%, 700px)" }}
            >
                <Box>
                    <Title
                        iconInfo={notificationsAllIconInfo}
                        title={t("Notification", { count: 2 })}
                        variant="subheader"
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
                    <Box className="tw-flex tw-items-center tw-mb-4">
                        <IntegerInput
                            disabled={!values.enabled}
                            label={t("DailyLimit")}
                            min={0}
                            name="dailyLimit"
                            className="tw-justify-start tw-ml-1 tw-mr-1 tw-w-auto tw-flex-grow"
                        />
                        <Button
                            variant="outlined"
                            className="tw-ml-2"
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
                        iconInfo={phoneIconInfo}
                        title={t("Device", { count: 2 })}
                        variant="subheader"
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
                        iconInfo={emailIconInfo}
                        title={t("Email", { count: 2 })}
                        variant="subheader"
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
}

export function SettingsNotificationsView({
    display,
    onClose,
}: SettingsNotificationsViewProps) {
    const { t } = useTranslation();

    const { data, refetch, loading: isLoading } = useFetch<undefined, NotificationSettings>(endpointsNotification.getSettings);
    const [updateFetch, { loading: isUpdating }] = useLazyFetch<NotificationSettingsUpdateInput, NotificationSettings>(endpointsNotification.updateSettings);

    return (
        <ScrollBox>
            <Navbar title={t("Notification", { count: 2 })} />
            <SettingsContent>
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
            </SettingsContent>
        </ScrollBox>
    );
}
