/* eslint-disable @typescript-eslint/no-redeclare */
import { COOKIE, ValueOf } from '@local/shared';

export const Pubs = {
    ...COOKIE,
    Loading: "loading",
    LogOut: "logout",
    AlertDialog: "alertDialog",
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
 * Only orchestrations that are valid or incomplete can be run
 */
 export enum OrchestrationStatus {
    Incomplete = 'Incomplete', // Orchestration would be valid, except there are unlinked nodes
    Invalid = 'Invalid', // Something is wrong with the orchestration (e.g. no end node)
    Valid = 'Valid', // The orchestration is valid, and all nodes are linked
}

/**
 * Prompts Orchestration page to open a specific dialog
 */
export enum OrchestrationDialogOption {
    AddRoutineItem,
    ViewRoutineItem,
}