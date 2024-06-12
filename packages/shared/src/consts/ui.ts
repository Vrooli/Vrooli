import { ValueOf } from ".";

export const LINKS = {
    About: "/about",
    Api: "/api",
    Awards: "/awards",
    BookmarkList: "/bookmarks",
    Calendar: "/calendar",
    Chat: "/chat",
    Code: "/code",
    Comment: "/comment",
    Create: "/create",
    Example: "/routine/5f0f8f9b-f8f9-4f9b-8f9b-f8f9b8f9b8f9", // Links to example routine that is designed to showcase the UI. See ID of routine set in init seed file
    ForgotPassword: "/auth/forgot-password",
    History: "/history",
    Home: "/", // Main dashboard for logged in users
    Inbox: "/inbox",
    Login: "/auth/login",
    MyStuff: "/my",
    Note: "/note",
    Pro: "/pro",
    Privacy: "/privacy", // Privacy policy
    Profile: "/profile",
    Project: "/project",
    Question: "/question",
    Quiz: "/quiz",
    Reminder: "/reminder",
    Report: "/report",
    Routine: "/routine",
    ResetPassword: "/auth/password-reset",
    Search: "/search",
    SearchVersion: "/search/version",
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
    Signup: "/auth/signup",
    SmartContract: "/contract",
    Standard: "/standard",
    Stats: "/stats", // Provides statistics for the website (no admin, so only place to see users, metrics, etc.)
    Tag: "/tag",
    Team: "/team",
    Terms: "/terms", // Terms and conditions
    User: "/profile",
} as const;
export type LINKS = ValueOf<typeof LINKS>;

export const THEME = {
    Light: "light",
    Dark: "dark",
};
export type THEME = ValueOf<typeof THEME>;
