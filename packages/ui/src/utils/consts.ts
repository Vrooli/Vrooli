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
 * A general status state for an object
 */
export enum Status {
    /**
     * Routine would be valid, except there are unlinked nodes
     */
    Incomplete = "Incomplete",
    /**
     * Something is wrong with the routine (e.g. no end node)
     */
    Invalid = "Invalid",
    /**
     * The routine is valid, and all nodes are linked
     */
    Valid = "Valid",
}

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
 * Non-input form elements
 */
export enum FormStructureType {
    Divider = "Divider",
    Header = "Header",
}

/**
 * State of build page's routine run simulation
 */
export enum BuildRunState {
    Paused,
    Running,
    Stopped,
}

export enum RoutineStepType {
    RoutineList = "RoutineList",
    Decision = "Decision",
    Subroutine = "Subroutine",
}

export enum ProjectStepType {
    Directory = "Directory",
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
