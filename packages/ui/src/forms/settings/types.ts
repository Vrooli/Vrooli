import { NotificationSettingsUpdateInput, ProfileEmailUpdateInput, ProfileUpdateInput } from "@local/shared";
import { FormikProps } from "formik";
import { ViewDisplayType } from "views/types";

export interface SettingsAuthenticationFormProps extends FormikProps<ProfileEmailUpdateInput> {
    display: ViewDisplayType;
    isLoading: boolean;
    onCancel: () => void;
}

export interface SettingsDisplayFormProps extends FormikProps<ProfileUpdateInput> {
    display: ViewDisplayType;
    isLoading: boolean;
    onCancel: () => void;
}

export interface SettingsNotificationFormProps extends FormikProps<NotificationSettingsUpdateInput> {
    display: ViewDisplayType;
    isLoading: boolean;
    onCancel: () => void;
}

export interface SettingsPrivacyFormProps extends FormikProps<ProfileUpdateInput> {
    display: ViewDisplayType;
    isLoading: boolean;
    onCancel: () => void;
}

export interface SettingsProfileFormProps extends FormikProps<any> {
    display: ViewDisplayType;
    isLoading: boolean;
    numVerifiedWallets: number;
    onCancel: () => void;
}