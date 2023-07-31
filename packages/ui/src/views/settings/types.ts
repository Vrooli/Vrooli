import { CommonKey, LINKS } from "@local/shared";
import { SvgComponent } from "types";
import { BaseViewProps } from "views/types";

export type SettingsPageType = "Data" | "Profile" | "Privacy" | "Authentication" | "Payment" | "Api" | "Display" | "Notification" | "FocusMode";
export type SettingsData = {
    title: SettingsPageType,
    titleVariables?: Record<string, string | number>,
    description: CommonKey,
    link: LINKS,
    Icon: SvgComponent,
};

export type SettingsViewProps = BaseViewProps

export type SettingsApiViewProps = BaseViewProps
export type SettingsAuthenticationViewProps = BaseViewProps
export type SettingsDataViewProps = BaseViewProps
export type SettingsDisplayViewProps = BaseViewProps
export type SettingsNotificationsViewProps = BaseViewProps
export type SettingsPaymentViewProps = BaseViewProps
export type SettingsPrivacyViewProps = BaseViewProps
export type SettingsProfileViewProps = BaseViewProps
export type SettingsFocusModesViewProps = BaseViewProps
