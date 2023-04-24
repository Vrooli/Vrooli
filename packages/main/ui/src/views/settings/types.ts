import { LINKS } from "@local/consts";
import { SvgComponent } from "@local/icons";
import { CommonKey } from "@local/translations";
import { BaseViewProps } from "../types";

export type SettingsPageType = "Profile" | "Privacy" | "Authentication" | "Display" | "Notification" | "FocusMode";
export type SettingsData = {
    title: SettingsPageType,
    description: CommonKey,
    link: keyof typeof LINKS,
    Icon: SvgComponent,
};

export interface SettingsViewProps extends BaseViewProps { }

export interface SettingsAuthenticationViewProps extends BaseViewProps { }
export interface SettingsDisplayViewProps extends BaseViewProps { }
export interface SettingsNotificationsViewProps extends BaseViewProps { }
export interface SettingsPrivacyViewProps extends BaseViewProps { }
export interface SettingsProfileViewProps extends BaseViewProps { }
export interface SettingsFocusModesViewProps extends BaseViewProps { }