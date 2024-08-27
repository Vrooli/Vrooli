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
    DataConverter: "/code",
    DataStructure: "/ds",
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
    Prompt: "/prompt",
    Question: "/question",
    Quiz: "/quiz",
    Reminder: "/reminder",
    Report: "/report",
    ResetPassword: "/auth/password-reset",
    Routine: "/routine",
    Run: "/run",
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

export const DEFAULT_LANGUAGE = "en";

export enum CodeLanguage {
    Angular = "angular",
    Cpp = "cpp",
    Css = "css",
    Dockerfile = "dockerfile",
    Go = "go",
    Graphql = "graphql",
    Groovy = "groovy",
    Haskell = "haskell",
    Html = "html",
    Java = "java",
    Javascript = "javascript",
    Json = "json", // JSON which may or may not conform to a standard
    JsonStandard = "jsonStandard", // JSON which defines a standard for some data (could be JSON, a file, etc.)
    Nginx = "nginx",
    Nix = "nix",
    Php = "php",
    Powershell = "powershell",
    Protobuf = "protobuf",
    Puppet = "puppet",
    Python = "python",
    R = "r",
    Ruby = "ruby",
    Rust = "rust",
    Sass = "sass",
    Shell = "shell",
    Solidity = "solidity",
    Spreadsheet = "spreadsheet",
    Sql = "sql",
    Svelte = "svelte",
    Swift = "swift",
    Typescript = "typescript",
    Vb = "vb",
    Vbscript = "vbscript",
    Verilog = "verilog",
    Vhdl = "vhdl",
    Vue = "vue",
    Xml = "xml",
    Yacas = "yacas",
    Yaml = "yaml",
}
