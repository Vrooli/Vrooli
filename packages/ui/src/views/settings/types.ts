import { LINKS, NotificationSettingsUpdateInput, ProfileUpdateInput, TranslationKeyCommon } from "@local/shared";
import { FormikProps } from "formik";
import { SvgComponent, ViewDisplayType, ViewProps } from "types";

export type SettingsPageType = "Data" | "Profile" | "Privacy" | "Authentication" | "Payment" | "Api" | "Display" | "Notification" | "FocusMode";
export type SettingsData = {
    title: SettingsPageType,
    titleVariables?: Record<string, string | number>,
    description: TranslationKeyCommon,
    link: LINKS,
    Icon: SvgComponent,
};

export type SettingsViewProps = ViewProps

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
export interface SettingsAuthenticationFormProps extends FormikProps<{
    currentPassword: string;
    newPassword: string;
    newPasswordConfirmation: string;
}>, SettingsFormBaseProps { }
export interface SettingsDataFormProps extends FormikProps<any>, SettingsFormBaseProps { } //TODO
export interface SettingsDisplayFormProps extends FormikProps<ProfileUpdateInput>, SettingsFormBaseProps { }
export interface SettingsNotificationFormProps extends FormikProps<NotificationSettingsUpdateInput>, SettingsFormBaseProps { }
export interface SettingsPrivacyFormProps extends FormikProps<ProfileUpdateInput>, SettingsFormBaseProps { }
export interface SettingsProfileFormProps extends FormikProps<any>, SettingsFormBaseProps {
    numVerifiedWallets: number;
}
