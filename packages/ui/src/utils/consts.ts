/* eslint-disable @typescript-eslint/no-redeclare */
import { ValueOf } from '@shared/consts';

export const Forms = {
    ForgotPassword: 'forgot-password',
    LogIn: 'logIn',
    Profile: 'profile',
    ResetPassword: 'reset-password',
    SignUp: 'signUp',
}
export type Forms = ValueOf<typeof Forms>;

/**
 * A general status state for an object
 */
 export enum Status {
    /**
     * Routine would be valid, except there are unlinked nodes
     */
    Incomplete = 'Incomplete',
    /**
     * Something is wrong with the routine (e.g. no end node)
     */
    Invalid = 'Invalid',
    /**
     * The routine is valid, and all nodes are linked
     */
    Valid = 'Valid',
}

/**
 * Prompts Build page to open a specific dialog
 */
export enum BuildAction {
    AddIncomingLink = 'AddIncomingLink',
    AddOutgoingLink = 'AddOutgoingLink',
    AddSubroutine = 'AddSubroutine',
    EditSubroutine = 'EditSubroutine',
    DeleteSubroutine = 'DeleteSubroutine',
    OpenSubroutine = 'OpenSubroutine',
    DeleteNode = 'DeleteNode',
    UnlinkNode = 'UnlinkNode',
    AddEndAfterNode = 'AddEndAfterNode',
    AddListAfterNode = 'AddListAfterNode',
    AddListBeforeNode = 'AddListBeforeNode',
    MoveNode = 'MoveNode',
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

/**
 * Supported types of resources
 */
export enum ResourceType {
    Url = 'Url',
    Wallet = 'Wallet',
    Handle = 'Handle',
}