import { NotificationSettingsUpdateInput, ProfileEmailUpdateInput, ProfileUpdateInput } from "@local/shared";
import { FormikProps } from "formik";
import { ViewDisplayType } from "views/types";

interface SettingsFormBaseProps {
    display: ViewDisplayType;
    isLoading: boolean;
    onCancel: () => void;
    zIndex: number;
}

export interface SettingsAuthenticationFormProps extends FormikProps<ProfileEmailUpdateInput>, SettingsFormBaseProps { }

export interface SettingsDisplayFormProps extends FormikProps<ProfileUpdateInput>, SettingsFormBaseProps { }

export interface SettingsNotificationFormProps extends FormikProps<NotificationSettingsUpdateInput>, SettingsFormBaseProps { }

export interface SettingsPrivacyFormProps extends FormikProps<ProfileUpdateInput>, SettingsFormBaseProps { }

export interface SettingsProfileFormProps extends FormikProps<any>, SettingsFormBaseProps {
    numVerifiedWallets: number;
}
