/* eslint-disable @typescript-eslint/no-redeclare */
import { COOKIE, ValueOf } from '@local/shared';

export const Pubs = {
    ...COOKIE,
    Loading: "loading",
    LogOut: "logout",
    AlertDialog: "alertDialog",
    Session: "session",
    Snack: "snack",
    BurgerMenuOpen: "burgerMenuOpen",
    ArrowMenuOpen: "arrowMenuOpen",
    Theme: "theme",
    NodeDrag: "NodeDrag",
    NodeDrop: "NodeDrop",
    NodeSetPosition: "NodeSetPosition",
}
export type Pubs = ValueOf<typeof Pubs>;

export const Forms = {
    ForgotPassword: 'forgot-password',
    LogIn: 'logIn',
    Profile: 'profile',
    ResetPassword: 'reset-password',
    SignUp: 'signUp',
}
export type Forms = ValueOf<typeof Forms>;

export const DragTypes = {
    Node: 'node',
}
export type DragTypes = ValueOf<typeof DragTypes>;

/**
 * Only routines that are valid or incomplete can be run
 */
 export enum BuildStatus {
    Incomplete = 'Incomplete', // Routine would be valid, except there are unlinked nodes
    Invalid = 'Invalid', // Something is wrong with the routine (e.g. no end node)
    Valid = 'Valid', // The routine is valid, and all nodes are linked
}

/**
 * Prompts Build page to open a specific dialog
 */
export enum BuildDialogOption {
    AddRoutineItem,
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
    RoutineList = 'RoutineList',
    Decision = 'Decision',
    Subroutine = 'Subroutine',
}