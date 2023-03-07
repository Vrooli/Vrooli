import { BaseViewProps } from "views/types";

export type SettingsPageType = 'Profile' | 'Privacy' | 'Authentication' | 'Display' | 'Notification' | 'Schedule';

export interface SettingsViewProps extends BaseViewProps {}

export interface SettingsAuthenticationViewProps extends BaseViewProps {}
export interface SettingsDisplayViewProps extends BaseViewProps {}
export interface SettingsNotificationsViewProps extends BaseViewProps {}
export interface SettingsPrivacyViewProps extends BaseViewProps {}
export interface SettingsProfileViewProps extends BaseViewProps {}
export interface SettingsSchedulesViewProps extends BaseViewProps {}