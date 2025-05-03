/* eslint-disable @typescript-eslint/no-redeclare */
import { InputType, SERVER_VERSION, ValueOf } from "@local/shared";
import { ViewDisplayType } from "../types.js";

export const Forms = {
    ForgotPassword: "forgot-password",
    Login: "login",
    Profile: "profile",
    ResetPassword: "reset-password",
    SignUp: "signUp",
};
export type Forms = ValueOf<typeof Forms>;

/**
 * Prompts Build page to open a specific dialog
 */
export enum BuildAction {
    AddIncomingLink = "AddIncomingLink",
    AddOutgoingLink = "AddOutgoingLink",
    AddSubroutine = "AddSubroutine",
    EditSubroutine = "EditSubroutine",
    DeleteSubroutine = "DeleteSubroutine",
    OpenSubroutine = "OpenSubroutine",
    DeleteNode = "DeleteNode",
    UnlinkNode = "UnlinkNode",
    AddEndAfterNode = "AddEndAfterNode",
    AddListAfterNode = "AddListAfterNode",
    AddListBeforeNode = "AddListBeforeNode",
    MoveNode = "MoveNode",
}

/**
 * State of build page's routine run simulation
 */
export enum BuildRunState {
    Paused,
    Running,
    Stopped,
}

/**
 * Supported types of resources
 */
export enum ResourceType {
    Url = "Url",
    Wallet = "Wallet",
    Handle = "Handle",
}

export type InputTypeOption = {
    label: string,
    description: string,
    value: InputType
}
/**
 * Supported input types
 */
export const InputTypeOptions: InputTypeOption[] = [
    {
        label: "Text",
        description: "Box for typing in whatever you like",
        value: InputType.Text,
    },
    {
        label: "JSON",
        description: "Box for typing in data structured in a special format called JSON",
        value: InputType.JSON,
    },
    {
        label: "Integer",
        description: "Box for typing in whole numbers",
        value: InputType.IntegerInput,
    },
    {
        label: "Radio (Select One)",
        description: "Set of options where you can choose only one.",
        value: InputType.Radio,
    },
    {
        label: "Checkbox (Select any)",
        description: "Set of options where you can choose any number of options.",
        value: InputType.Checkbox,
    },
    {
        label: "Switch (On/Off)",
        description: "Flip switch for a true/false value.",
        value: InputType.Switch,
    },
    // {
    //     label: 'File Upload',
    //     value: InputType.Dropzone,
    // },
];

export enum RelationshipButtonType {
    Owner = "Owner",
    // Parent = "Parent",
    IsPrivate = "IsPrivate",
    IsComplete = "IsComplete",
    Members = "Members",
    Participants = "Participants",
}

/** Number of dummy items in a list for full page, typically */
export const DUMMY_LIST_LENGTH = 5;
/** Number of dummy items on mobile or shorter components */
export const DUMMY_LIST_LENGTH_SHORT = 3;
/** Determines dummy list length to use for component */
export function getDummyListLength(display: ViewDisplayType | `${ViewDisplayType}`) {
    return display === ViewDisplayType.Page ? DUMMY_LIST_LENGTH : DUMMY_LIST_LENGTH_SHORT;
}

/** Limit for chips displayed in multi selector components */
export const CHIP_LIST_LIMIT = 3;

/** Default minimum rows for rich inputs */
export const DEFAULT_MIN_ROWS = 4;

export const FONT_SIZE_MIN = 8;
export const FONT_SIZE_MAX = 24;

export const BUSINESS_DATA = {
    BUSINESS_NAME: "Vrooli",
    EMAIL: {
        Label: "info@vrooli.com",
        Link: "mailto:info@vrooli.com",
    },
    SUPPORT_EMAIL: {
        Label: "support@vrooli.com",
        Link: "mailto:support@vrooli.com",
    },
    SOCIALS: {
        Discord: "https://discord.gg/VyrDFzbmmF",
        GitHub: "https://github.com/MattHalloran/Vrooli",
        X: "https://x.com/VrooliOfficial",
    },
    APP_URL: "https://vrooli.com",
};

// Determine origin of API server
const windowObj = typeof global !== "undefined" && global.window
    ? global.window
    : typeof window !== "undefined"
        ? window
        : null;
const host = windowObj?.location?.host ?? "";
const hostname = windowObj?.location?.hostname ?? "";
const isLocalhost: boolean = host.includes("localhost") || host.includes("192.168.") || host.includes("127.0.0.1");
const serverUrlProvided = Boolean(process.env.VITE_API_URL && process.env.VITE_API_URL.length > 0);
const portServer: string = process.env.VITE_PORT_API ?? "5329";
export const apiUrlBase: string = isLocalhost ?
    `http://${hostname}:${portServer}/api` :
    serverUrlProvided ?
        `${process.env.VITE_API_URL}` :
        `http://${process.env.VITE_SITE_IP}:${portServer}/api`;
export const restBase = `/${SERVER_VERSION}`;
export const webSocketUrlBase: string = isLocalhost ?
    `http://${hostname}:${portServer}` :
    serverUrlProvided ?
        `${process.env.VITE_API_URL}` :
        `http://${process.env.VITE_SITE_IP}:${portServer}`;

/**
 * Distance before a click is considered a drag
 */
export const DRAG_THRESHOLD = 10;

/** Maximum width of the chat input */
export const MAX_CHAT_INPUT_WIDTH = 800;

export const ELEMENT_IDS = {
    AdaptiveLayout: "adaptive-layout",
    AdvancedSearchDialog: "advanced-search-dialog",
    AdvancedSearchDialogTitle: "advanced-search-dialog-title",
    BottomNav: "bottom-nav",
    CommandPalette: "command-palette",
    DashboardEventList: "dashboard-event-list",
    DashboardReminderList: "dashboard-reminder-list",
    DashboardResourceList: "dashboard-resource-list",
    EventCards: "event-cards",
    FindInPage: "find-in-page",
    FormRunView: "form-run-view",
    FullPageSpinner: "full-page-spinner",
    LandingViewSlideContainerNeon: "neon-container",
    LandingViewSlideContainerSky: "sky-container",
    LandingViewSlideWorkflow: "revolutionize-workflow",
    LandingViewSlideChats: "chats",
    LandingViewSlideRoutines: "routines",
    LandingViewSlideTeams: "teams",
    LandingViewSlidePricing: "pricing",
    LandingViewSlideGetStarted: "get-started",
    LeftDrawer: "left-drawer",
    MyStuffTabs: "my-stuff-tabs",
    PageContainer: "page-container",
    ProViewDonateBox: "donate",
    ProViewFAQBox: "faq",
    ProViewFeatures: "features",
    RelationshipList: "relationship-list",
    ReminderCards: "reminder-cards",
    ResourceCards: "resource-cards",
    RightDrawer: "right-drawer",
    RoutineGenerateSettings: "routine-generate-settings",
    RoutineMultiStepCrudDialog: "routine-multi-step-crud-dialog",
    RoutineMultiStepCrudGraph: "routine-multi-step-crud-graph",
    RoutineTypeForm: "routine-type-form",
    RoutineSingleStepUpsertDialog: "routine-single-step-upsert-dialog",
    RoutineWizardDialog: "routine-wizard-dialog",
    SearchTabs: "search-tabs",
    SelectBookmarkListDialog: "select-bookmark-list-dialog",
    SiteNavigatorMenuMessageTree: "site-navigator-menu-message-tree",
    SiteNavigatorMenuIcon: "site-navigator-menu-icon",
    TasksRow: "tasks-row",
    TeamUpsertDialog: "team-upsert-dialog",
    Tutorial: "tutorial",
    UserMenu: "user-menu",
    UserMenuAccountList: "user-menu-account-list",
    UserMenuDisplaySettings: "user-menu-display-settings",
    UserMenuProfileIcon: "user-menu-profile-icon",
    UserMenuQuickLinks: "user-menu-quick-links",
    WalletInstallDialogTitle: "wallet-install-dialog-title",
    WalletSelectDialogTitle: "wallet-select-dialog-title",
} as const;

export const ELEMENT_CLASSES = {
    ScrollBox: "scroll-box",
    SearchBar: "search-bar",
} as const;

export const Z_INDEX = {
    Popup: 1000,
    TutorialDialog: 800,
    CommandPalette: 700,
    CookieSettingsDialog: 200,
    Dialog: 100,
    Drawer: 60,
    TopBar: 50,
    BottomNav: 50,
    ActionButton: 20,
    PageElement: 10,
    Page: 0,
};
