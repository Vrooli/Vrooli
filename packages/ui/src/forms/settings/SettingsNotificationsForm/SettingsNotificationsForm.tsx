import { EmailIcon, PhoneIcon } from "@local/shared";
import { Stack } from "@mui/material";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { IntegerInput } from "components/inputs/IntegerInput/IntegerInput";
import { PushList } from "components/lists/devices";
import { SettingsToggleListItem } from "components/lists/SettingsToggleListItem/SettingsToggleListItem";
import { Title } from "components/text/Title/Title";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useTranslation } from "react-i18next";
import { SettingsNotificationFormProps } from "../types";

export const SettingsNotificationForm = ({
    display,
    dirty,
    isLoading,
    onCancel,
    values,
    zIndex,
    ...props
}: SettingsNotificationFormProps) => {
    const { t } = useTranslation();

    return (
        <BaseForm
            dirty={dirty}
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
                    zIndex={zIndex}
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
                    zIndex={zIndex}
                />
                {/* <EmailList
                handleUpdate={updateEmails}
                list={profile?.emails ?? []}
                numVerifiedWallets={numVerifiedWallets}
            /> */}
                {/* Toggle individual categories */}
                {/* TODO */}
            </Stack>
            <GridSubmitButtons
                display={display}
                errors={props.errors}
                isCreate={false}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
                zIndex={zIndex}
            />
        </BaseForm>
    );
};
