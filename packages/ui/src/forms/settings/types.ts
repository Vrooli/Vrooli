import { NotificationSettingsUpdateInput, ProfileEmailUpdateInput } from "@shared/consts";
import { FormikProps } from "formik";
import { ViewDisplayType } from "views/types";

export interface SettingsAuthenticationFormProps extends FormikProps<ProfileEmailUpdateInput> {
    display: ViewDisplayType;
    isLoading: boolean;
    onCancel: () => void;
}

export interface SettingsDisplayFormProps extends FormikProps<NotificationSettingsUpdateInput> {
    display: ViewDisplayType;
    isLoading: boolean;
    onCancel: () => void;
}

export interface SettingsNotificationFormProps extends FormikProps<NotificationSettingsUpdateInput> {
    display: ViewDisplayType;
    isLoading: boolean;
    onCancel: () => void;
}

export interface SettingsPrivacyFormProps extends FormikProps<NotificationSettingsUpdateInput> {
    display: ViewDisplayType;
    isLoading: boolean;
    onCancel: () => void;
}

export interface SettingsProfileFormProps extends FormikProps<NotificationSettingsUpdateInput> {
    display: ViewDisplayType;
    isLoading: boolean;
    onCancel: () => void;
}

export interface SettingsSchedulesFormProps extends FormikProps<NotificationSettingsUpdateInput> {
    display: ViewDisplayType;
    isLoading: boolean;
    onCancel: () => void;
}