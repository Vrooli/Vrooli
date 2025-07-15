// AI_CHECK: TYPE_SAFETY=5 | LAST: 2025-07-01
import { type LINKS, type NotificationSettingsUpdateInput, type ProfileUpdateInput, type TranslationKeyCommon } from "@vrooli/shared";
import { type FormikProps } from "formik";
import { type IconInfo } from "../../icons/Icons.js";
import { type ViewDisplayType, type ViewProps } from "../../types.js";

export type SettingsPageType = "Data" | "Profile" | "Privacy" | "Authentication" | "Payment" | "Api" | "Display" | "Notification";
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

interface SettingsFormBaseProps {
    display: `${ViewDisplayType}`;
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
export interface SettingsPrivacyFormProps extends FormikProps<ProfileUpdateInput>, SettingsFormBaseProps {
    numVerifiedWallets: number;
}

export type SettingsProfileFormInput = ProfileUpdateInput & { 
    updatedAt?: string | null; // Used for cache busting on profile image
};
export interface SettingsProfileFormProps extends FormikProps<SettingsProfileFormInput>, SettingsFormBaseProps { }
