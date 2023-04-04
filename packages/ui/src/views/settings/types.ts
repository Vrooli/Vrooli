import { LINKS } from "@shared/consts";
import { SvgComponent } from "@shared/icons";
import { CommonKey } from "@shared/translations";
import { BaseViewProps } from "views/types";

export type SettingsPageType = 'Profile' | 'Privacy' | 'Authentication' | 'Display' | 'Notification' | 'Schedule';
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