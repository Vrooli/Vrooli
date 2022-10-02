/* eslint-disable @typescript-eslint/no-redeclare */
import { InputType, ValueOf } from '@shared/consts';

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

export type InputTypeOption = { label: string, value: InputType }
/**
 * Supported input types
 */
export const InputTypeOptions: InputTypeOption[] = [
    {
        label: 'Text',
        value: InputType.TextField,
    },
    {
        label: 'JSON',
        value: InputType.JSON,
    },
    {
        label: 'Integer',
        value: InputType.QuantityBox
    },
    {
        label: 'Radio (Select One)',
        value: InputType.Radio,
    },
    {
        label: 'Checkbox (Select any)',
        value: InputType.Checkbox,
    },
    {
        label: 'Switch (On/Off)',
        value: InputType.Switch,
    },
    // {
    //     label: 'File Upload',
    //     value: InputType.Dropzone,
    // },
    {
        label: 'Markdown',
        value: InputType.Markdown
    },
]