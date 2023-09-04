import { ValueOf } from ".";

export const LINKS = {
    About: "/about",
    Api: "/api",
    Awards: "/awards",
    BookmarkList: "/bookmarks",
    Calendar: "/calendar",
    Chat: "/chat",
    Comment: "/comment",
    Create: "/create",
    Example: "/routine/5f0f8f9b-f8f9-4f9b-8f9b-f8f9b8f9b8f9", // Links to example routine that is designed to showcase the UI. See ID of routine set in init seed file
    ForgotPassword: "/forgot-password",
    History: "/history",
    Home: "/", // Main dashboard for logged in users
    Inbox: "/inbox",
    MyStuff: "/my",
    Note: "/note",
    Organization: "/team",
    Premium: "/premium",
    Privacy: "/privacy", // Privacy policy
    Profile: "/profile",
    Project: "/project",
    Question: "/question",
    Quiz: "/quiz",
    Reminder: "/reminder",
    Report: "/report",
    Routine: "/routine",
    ResetPassword: "/password-reset",
    Search: "/search",
    Settings: "/settings",
    SettingsApi: "/settings/api",
    SettingsAuthentication: "/settings/auth",
    SettingsData: "/settings/data",
    SettingsDisplay: "/settings/display",
    SettingsFocusModes: "/settings/focus",
    SettingsNotifications: "/settings/notifications",
    SettingsPayments: "/settings/payments",
    SettingsPrivacy: "/settings/privacy",
    SettingsProfile: "/settings/profile",
    SmartContract: "/contract",
    Standard: "/standard",
    Start: "/start",
    Stats: "/stats", // Provides statistics for the website (no admin, so only place to see users, metrics, etc.)
    Tag: "/tag",
    Terms: "/terms", // Terms and conditions
    User: "/profile",
} as const;
export type LINKS = ValueOf<typeof LINKS>;

export const THEME = {
    Light: "light",
    Dark: "dark",
};
export type THEME = ValueOf<typeof THEME>;
