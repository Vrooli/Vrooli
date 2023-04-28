import { CommonKey, LINKS, SvgComponent } from "@local/shared";
import { BaseViewProps } from "views/types";

export type SettingsPageType = "Profile" | "Privacy" | "Authentication" | "Display" | "Notification" | "FocusMode";
export type SettingsData = {
    title: SettingsPageType,
    description: CommonKey,
    link: keyof typeof LINKS,
    Icon: SvgComponent,
};

export type SettingsViewProps = BaseViewProps

export type SettingsAuthenticationViewProps = BaseViewProps
export type SettingsDisplayViewProps = BaseViewProps
export type SettingsNotificationsViewProps = BaseViewProps
export type SettingsPrivacyViewProps = BaseViewProps
export type SettingsProfileViewProps = BaseViewProps
export type SettingsFocusModesViewProps = BaseViewProps
