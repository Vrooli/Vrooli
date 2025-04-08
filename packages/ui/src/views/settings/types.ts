import { LINKS, NotificationSettingsUpdateInput, ProfileUpdateInput, TranslationKeyCommon } from "@local/shared";
import { FormikProps } from "formik";
import { IconInfo } from "../../icons/Icons.js";
import { ViewDisplayType, ViewProps } from "../../types.js";

export type SettingsPageType = "Data" | "Profile" | "Privacy" | "Authentication" | "Payment" | "Api" | "Display" | "Notification" | "FocusMode";
export type SettingsData = {
    description: TranslationKeyCommon,
    iconInfo: IconInfo,
    link: LINKS,
    title: SettingsPageType,
    titleVariables?: Record<string, string | number>,
};

export type SettingsApiViewProps = ViewProps
export type SettingsAuthenticationViewProps = ViewProps
export type SettingsDataViewProps = ViewProps
export type SettingsDisplayViewProps = ViewProps
export type SettingsNotificationsViewProps = ViewProps
export type SettingsPaymentViewProps = ViewProps
export type SettingsPrivacyViewProps = ViewProps
export type SettingsProfileViewProps = ViewProps
export type SettingsFocusModesViewProps = ViewProps

interface SettingsFormBaseProps {
    display: ViewDisplayType;
    isLoading: boolean;
    onCancel: () => unknown;
}
export type SettingsAuthenticationFormValues = {
    currentPassword: string;
    newPassword: string;
    newPasswordConfirmation: string;
};
export interface SettingsAuthenticationFormProps extends FormikProps<SettingsAuthenticationFormValues>, SettingsFormBaseProps { }
export interface SettingsDataFormProps extends FormikProps<any>, SettingsFormBaseProps { } //TODO
export interface SettingsDisplayFormProps extends FormikProps<ProfileUpdateInput>, SettingsFormBaseProps { }
export interface SettingsNotificationFormProps extends FormikProps<NotificationSettingsUpdateInput>, SettingsFormBaseProps { }
export interface SettingsPrivacyFormProps extends FormikProps<ProfileUpdateInput>, SettingsFormBaseProps { }
export interface SettingsProfileFormProps extends FormikProps<any>, SettingsFormBaseProps {
    numVerifiedWallets: number;
}
