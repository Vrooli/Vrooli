import { Typography, useTheme } from "@mui/material";
import { ListContainer } from "components/containers/ListContainer/ListContainer";
import { SettingsToggleListItem } from "components/lists/SettingsToggleListItem/SettingsToggleListItem";
import { useField } from "formik";
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
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [isPrivateField] = useField<boolean>('isPrivate');

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
            {isPrivateField.value && (
                <Typography variant="body2" sx={{
                    marginTop: 4,
                    marginBottom: 1,
                    color: palette.warning.main,
                }}>
                    All of your content is private. Turn off private mode to change specific settings.
                </Typography>
            )}
            <ListContainer>
                <SettingsToggleListItem
                    disabled={isPrivateField.value}
                    title={t('PrivateApis')}
                    name="isPrivateApis"
                />
                <SettingsToggleListItem
                    disabled={isPrivateField.value}
                    title={t('PrivateBookmarks')}
                    name="isPrivateBookmarks"
                />
                <SettingsToggleListItem
                    disabled={isPrivateField.value}
                    title={t('PrivateProjects')}
                    name="isPrivateProjects"
                />
                <SettingsToggleListItem
                    disabled={isPrivateField.value}
                    title={t('PrivateRoutines')}
                    name="isPrivateRoutines"
                />
                <SettingsToggleListItem
                    disabled={isPrivateField.value}
                    title={t('PrivateSmartContracts')}
                    name="isPrivateSmartContracts"
                />
                <SettingsToggleListItem
                    disabled={isPrivateField.value}
                    title={t('PrivateStandards')}
                    name="isPrivateStandards"
                />
            </ListContainer>
        </BaseForm>
    )
}