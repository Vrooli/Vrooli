import { EmailIcon, PhoneIcon } from "@shared/icons";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { IntegerInput } from "components/inputs/IntegerInput/IntegerInput";
import { PushList } from "components/lists/devices";
import { SettingsToggleListItem } from "components/lists/SettingsToggleListItem/SettingsToggleListItem";
import { Subheader } from "components/text/Subheader/Subheader";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useTranslation } from "react-i18next";
import { SettingsNotificationFormProps } from "../types";

export const SettingsNotificationForm = ({
    display,
    dirty,
    isLoading,
    onCancel,
    values,
    ...props
}: SettingsNotificationFormProps) => {
    const { t } = useTranslation();

    return (
        <BaseForm
            dirty={dirty}
            isLoading={isLoading}
            style={{
                width: { xs: '100%', md: 'min(100%, 700px)' },
                margin: 'auto',
                display: 'block',
            }}
        >
            {/* Overall notifications toggle */}
            <ListContainer>
                <SettingsToggleListItem
                    title={t('Notification', { count: 2 })}
                    description={t('PushNotificationToggleDescription')}
                    name="enabled"
                />
            </ListContainer>
            {/* Daily limit input */}
            <IntegerInput
                disabled={!values.enabled}
                label={t('DailyLimit')}
                min={0}
                name="dailyLimit"
            // tooltip={t('DailyLimitNotificationDescription')}
            />
            {/* Push notifications toggle */}
            <ListContainer>
                <SettingsToggleListItem
                    title={t('PushNotification', { count: 2 })}
                    description={t('PushNotificationToggleDescription')}
                    disabled={!values.enabled}
                    name="toPush"
                />
            </ListContainer>
            {/* Push Device list */}
            <Subheader
                Icon={PhoneIcon}
                title={t('Device', { count: 2 })} />
            <PushList
                handleUpdate={() => { }}
                list={[]}
            />
            {/* Email notifications toggle */}
            <ListContainer>
                <SettingsToggleListItem
                    title={t('EmailNotification', { count: 2 })}
                    description={t('EmailNotificationToggleDescription')}
                    disabled={!values.enabled}
                    name="toEmails"
                />
            </ListContainer>
            {/* Email list */}
            <Subheader
                Icon={EmailIcon}
                title={t('Email', { count: 2 })} />
            {/* <EmailList
                handleUpdate={updateEmails}
                list={profile?.emails ?? []}
                numVerifiedWallets={numVerifiedWallets}
            /> */}
            {/* Toggle individual categories */}
            {/* TODO */}
            <GridSubmitButtons
                display={display}
                errors={props.errors}
                isCreate={false}
                loading={props.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={props.setSubmitting}
                onSubmit={props.handleSubmit}
            />
        </BaseForm>
    )
}