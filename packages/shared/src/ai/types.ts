/* c8 ignore start */
import { type LlmTask, type TaskStatus } from "../api/types.js";

/**
 * Context is any JSON-serializable data that can be used to provide context to the LLM.
 */
export type TaskContextInfo = {
    /**
     * A unique ID to identify the context. This is used to update or remove the context, 
     * and only needs to be unique for the UI. If you want to use this as the ID for multiple 
     * users (e.g. for bull tasks), you should prefix the ID with the user ID.
     */
    id: string;
    /**
     * Human-readable identifier for the context (e.g., "Code Review", "Search")
     * @optional For backward compatibility
     */
    name?: string;
    /**
     * Detailed explanation of what the context contains
     * @optional For backward compatibility
     */
    description?: string;
    /**
     * Machine-readable category for programmatic processing (e.g., "code", "search", "analysis", "math")
     * @optional For backward compatibility
     */
    type?: string;
    /**
     * The JSON-serializable data to provide to the LLM.
     */
    data: unknown;
    /**
     * Label to display to the user
     */
    label: string;
    /**
     * Template to use for stringifying the data for the LLM.
     */
    template?: string | null;
    /**
     * Template variables to use for stringifying the data for the LLM. 
     * Defaults to:
     * - task: <TASK>
     * - data: <DATA>
     */
    templateVariables?: {
        data?: string | null;
        task?: string | null;
    } | null;
}

/**
 * A task that an AI can, is, or has performed, along with 
 * other information about the task for display and execution.
 */
export type AITaskInfo = {
    /** A user-friendly label to display to the user */
    label: string;
    /** The last time a status change occurred, as an ISO string */
    lastUpdated: string;
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
    /** Unique ID to track task */
    taskId: string;
    /**
     * The latest status of the command.
     */
    status: TaskStatus | `${TaskStatus}`;
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
export type AITaskProperty = {
    name: string,
    type?: string,
    description?: string,
    example?: any,
    examples?: string[],
    is_required?: boolean
    default?: string | number | boolean | null;
    /** Used to describe nested data if type is an array or object */
    properties?: Record<string, Omit<AITaskProperty, "name">>;
};

/** 
 * Command information, which can be used to validate commands or 
 * converted into a structured command to provide to the LLM as context
 */
export type AITaskUnstructuredConfig = {
    actions?: string[];
    properties?: (string | AITaskProperty)[],
    commands: Record<string, string>,
    label: string,
} & Record<string, any>;

/** Structured command information, which can be used to provide context to the LLM */
export type LlmTaskStructuredConfig = Record<string, any>;

type JSONValue = string | number | boolean | null | JSONObject | JSONArray;
type JSONObject = { [key: string]: JSONValue };
type JSONArray = JSONValue[];

export type TaskNameMap = {
    command: Record<string, string>,
    action: Record<string, string>,
};

/**
 * Information about all LLM tasks in a given language, compiled into a single JSON object. 
 * This should not include any functions, and should be safe to fetch from the server.
 */
export type AITaskConfigJSON = Record<LlmTask, AITaskUnstructuredConfig> & {
    __suggested_prefix: string;
}

/**
 * Converts commands and actions into strings, which can be combined to create valid LlmTask values. 
 */
export type CommandToTask = (command: string, action?: string | null) => (LlmTask | null);

export type PartialTaskInfo = Omit<AITaskInfo, "label" | "lastUpdated" | "status" | "taskId"> & {
    start: number;
    end: number;
};
export type MaybeLlmTaskInfo = Omit<PartialTaskInfo, "taskId" | "task"> & {
    task: LlmTask | `${LlmTask}` | null;
};
/** Task info that the server needs to run a task. Excludes any info only useful for the UI */
export type ServerLlmTaskInfo = Omit<AITaskInfo, "lastUpdated" | "status">;
