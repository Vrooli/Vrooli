import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { SettingsToggleListItem } from "components/lists/SettingsToggleListItem/SettingsToggleListItem";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useTranslation } from "react-i18next";
import { SettingsPrivacyFormProps } from "../types";

export const SettingsPrivacyForm = ({
    display,
    dirty,
    isLoading,
    onCancel,
    values,
    ...props
}: SettingsPrivacyFormProps) => {
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
                    title={t('PrivateAccount')}
                    description={t('PushNotificationToggleDescription')}
                    name="isPrivate"
                />
            </ListContainer>
            {/* By object type */}
            <ListContainer>
                <SettingsToggleListItem
                    title={t('isPrivateApis')}
                    name="isPrivateApis"
                />
                <SettingsToggleListItem
                    title={t('isPrivateBookmarks')}
                    name="isPrivateBookmarks"
                />
                <SettingsToggleListItem
                    title={t('isPrivateProjects')}
                    name="isPrivateProjects"
                />
                <SettingsToggleListItem
                    title={t('isPrivateRoutines')}
                    name="isPrivateRoutines"
                />
                <SettingsToggleListItem
                    title={t('isPrivateSmartContracts')}
                    name="isPrivateSmartContracts"
                />
                <SettingsToggleListItem
                    title={t('isPrivateStandards')}
                    name="isPrivateStandards"
                />
            </ListContainer>
        </BaseForm>
    )
}