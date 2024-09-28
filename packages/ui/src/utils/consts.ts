/* eslint-disable @typescript-eslint/no-redeclare */
import { InputType, ValueOf } from "@local/shared";

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
    FocusMode = "FocusMode",
    QuestionFor = "QuestionFor",
    Members = "Members",
    Participants = "Participants",
}

/** Number of dummy items in a list for full page, typically */
export const DUMMY_LIST_LENGTH = 5;
/** Number of dummy items on mobile or shorter components */
export const DUMMY_LIST_LENGTH_SHORT = 3;
/** Determines dummy list length to use for component */
export function getDummyListLength(display: "dialog" | "page" | "partial") {
    return display === "page" ? DUMMY_LIST_LENGTH : DUMMY_LIST_LENGTH_SHORT;
}

/** Limit for chips displayed in multi selector components */
export const CHIP_LIST_LIMIT = 3;

/** Default minimum rows for rich inputs */
export const DEFAULT_MIN_ROWS = 4;

export const FONT_SIZE_MIN = 8;
export const FONT_SIZE_MAX = 24;

export const LEFT_DRAWER_WIDTH = 280;
export const RIGHT_DRAWER_WIDTH = 280;

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
const isLocalhost: boolean = window.location.host.includes("localhost") || window.location.host.includes("192.168.") || window.location.host.includes("127.0.0.1");
const serverUrlProvided = Boolean(process.env.VITE_SERVER_URL && process.env.VITE_SERVER_URL.length > 0);
const portServer: string = process.env.VITE_PORT_SERVER ?? "5329";
export const apiUrlBase: string = isLocalhost ?
    `http://${window.location.hostname}:${portServer}/api` :
    serverUrlProvided ?
        `${process.env.VITE_SERVER_URL}` :
        `http://${process.env.VITE_SITE_IP}:${portServer}/api`;
export const restBase = "/v2/rest";
export const webSocketUrlBase: string = isLocalhost ?
    `http://${window.location.hostname}:${portServer}` :
    serverUrlProvided ?
        `${process.env.VITE_SERVER_URL}` :
        `http://${process.env.VITE_SITE_IP}:${portServer}`;
