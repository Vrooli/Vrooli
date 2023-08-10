import { CommonKey, LINKS } from "@local/shared";
import { SvgComponent } from "types";
import { ViewProps } from "views/types";

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
