import { LlmTask, LlmTaskStatus } from "../api/generated/graphqlTypes";

/**
 * A task that an AI can, is, or has performed, along with 
 * other information about the task for display and execution.
 */
export type LlmTaskInfo = {
    /** 
     * The action being performed. Can be thought of as a modifier to the command. 
     * For example, if the command is "note", the action could be "add" or "delete".
     * 
     * NOTE: This is in the user's language, not the server's language.
     */
    action: string | null;
    /**
     * The command string of the task, in the user's language.
     */
    command: string;
    /** Unique ID to track command */
    id: string;
    /** A user-friendly label to display to the user */
    label: string;
    /** The last time a status change occurred, as an ISO string */
    lastUpdated: string;
    /** Message associated with the command */
    messageId?: string;
    /**
     * Data passed in when executing the task.
     */
    properties: {
        [key: string]: string | boolean | number | null;
    } | null;
    /**
     * Label for result of task, if applicable. 
     * 
     * For example, if the task is to create a note, the resultLabel could be
     * the note's title.
     */
    resultLabel?: string
    /**
     * Link to open the result of the task, if applicable.
     * 
     * For example, if the task is to create a note, the resultLink could be
     * the link to view the note.
     */
    resultLink?: string;
    /**
     * The task being performed, as a language-independent type.
     */
    task: LlmTask | `${LlmTask}`;
    /**
     * The latest status of the command.
     */
    status: LlmTaskStatus | `${LlmTaskStatus}`;
};

export type ExistingTaskData = Record<string, string | number | boolean | null>;

/** State when looping through text to locate tasks */
export type CommandSection = "outside" | "code" | "command" | "action" | "propName" | "propValue";

/** State and other relevant information when looping through text to locate tasks */
export type CommandTransitionTrack = {
    section: CommandSection,
    buffer: string[],
    /** If true, command will be canceled later if we don't find a closing bracket */
    hasOpenBracket: boolean,
}

/** Information about a property provided with a command */
export type LlmTaskProperty = {
    name: string,
    type?: string,
    description?: string,
    example?: string,
    examples?: string[],
    is_required?: boolean
};

/** 
 * Command information, which can be used to validate commands or 
 * converted into a structured command to provide to the LLM as context
 */
export type LlmTaskUnstructuredConfig = {
    actions?: string[];
    properties?: (string | LlmTaskProperty)[],
    commands: Record<string, string>,
    label: string,
} & Record<string, any>;

/** Structured command information, which can be used to provide context to the LLM */
export type LlmTaskStructuredConfig = Record<string, any>;

/**
 * Information about all LLM tasks in a given language, including a function to 
 * convert unstructured task information into structured task information
 */
export type LlmTaskConfig = Record<LlmTask, (() => LlmTaskUnstructuredConfig)> & {
    /**
     * Prefix for suggested commands. These are commands that are not run right away.
     * Instead, they are suggested to the user, who can then choose to run them.
     */
    __suggested_prefix: string;
    /**
     * Builds context object to add to the LLM's context, so that it 
     * can start or execute commands
     */
    __construct_context: (data: LlmTaskUnstructuredConfig) => LlmTaskStructuredConfig;
    /**
     * Similar to __construct_context, but should force LLM to 
     * respond with a command - rather than making it optional
     */
    __construct_context_force: (data: LlmTaskUnstructuredConfig) => LlmTaskStructuredConfig;
    // Allow for additional properties, as long as they're prefixed with "__"
    [Key: `__${string}`]: any;
    [Key: `__${string}Properties`]: Record<string, Omit<LlmTaskProperty, "name">>;
}


/** Converts a command and optional action to a valid task name, or null if invalid */
export type CommandToTask = (command: string, action?: string | null) => (LlmTask | null);

export type PartialTaskInfo = Omit<LlmTaskInfo, "label" | "lastUpdated" | "status"> & {
    start: number;
    end: number;
};
export type MaybeLlmTaskInfo = Omit<PartialTaskInfo, "id" | "task"> & {
    task: LlmTask | `${LlmTask}` | null;
};
/** Task info that the server needs to run a task. Excludes any info only useful for the UI */
export type ServerLlmTaskInfo = Omit<LlmTaskInfo, "lastUpdated" | "status">;
