import { ValueOf } from ".";

export const LINKS = {
    About: "/about",
    Api: "/api",
    Awards: "/awards",
    BookmarkList: "/bookmarks",
    Calendar: "/calendar",
    Comment: "/comment",
    Create: "/create",
    Example: "/routine/5f0f8f9b-f8f9-4f9b-8f9b-f8f9b8f9b8f9", // Links to example routine that is designed to showcase the UI. See ID of routine set in init seed file
    FAQ: "/#faq", // FAQ section of home page
    ForgotPassword: "/forgot-password",
    History: "/history",
    Home: "/", // Main dashboard for logged in users
    MyStuff: "/my",
    Note: "/note",
    Notifications: "/inbox",
    Organization: "/org",
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
    SettingsProfile: "/settings/profile",
    SettingsPrivacy: "/settings/privacy",
    SettingsAuthentication: "/settings/auth",
    SettingsDisplay: "/settings/display",
    SettingsNotifications: "/settings/notifications",
    SettingsFocusModes: "/settings/focus",
    SmartContract: "/contract",
    Standard: "/standard",
    Start: "/start",
    Stats: "/stats", // Provides statistics for the website (no admin, so only place to see users, metrics, etc.)
    Tag: "/tag",
    Terms: "/terms", // Terms and conditions
    Tutorial: "/tutorial",
    User: "/profile",
    Welcome: "/welcome", // Displays the first time you enter the application (either as guest or as logged in user)
};
export type LINKS = ValueOf<typeof LINKS>;

export const THEME = {
    Light: "light",
    Dark: "dark",
};
export type THEME = ValueOf<typeof THEME>;
