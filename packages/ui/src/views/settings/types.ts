import { CommonKey, LINKS, NotificationSettingsUpdateInput, ProfileEmailUpdateInput, ProfileUpdateInput } from "@local/shared";
import { FormikProps } from "formik";
import { SvgComponent } from "types";
import { ViewDisplayType, ViewProps } from "views/types";

export type SettingsPageType = "Data" | "Profile" | "Privacy" | "Authentication" | "Payment" | "Api" | "Display" | "Notification" | "FocusMode";
export type SettingsData = {
    title: SettingsPageType,
    titleVariables?: Record<string, string | number>,
    description: CommonKey,
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
export interface SettingsAuthenticationFormProps extends FormikProps<ProfileEmailUpdateInput>, SettingsFormBaseProps { }
export interface SettingsDisplayFormProps extends FormikProps<ProfileUpdateInput>, SettingsFormBaseProps { }
export interface SettingsNotificationFormProps extends FormikProps<NotificationSettingsUpdateInput>, SettingsFormBaseProps { }
export interface SettingsPrivacyFormProps extends FormikProps<ProfileUpdateInput>, SettingsFormBaseProps { }
export interface SettingsProfileFormProps extends FormikProps<any>, SettingsFormBaseProps {
    numVerifiedWallets: number;
}
