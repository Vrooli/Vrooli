/**
 * Task type definitions for the queue system
 * 
 * This file defines all supported task types and their data shapes for type safety.
 * Using discriminated union types to ensure tasks are properly shaped based on their type.
 */

import { type RunConfig, type RunProgress, type RunTriggeredFrom, type SessionUser, type TaskContextInfo, type TaskStatus } from "@vrooli/shared";
import { type ExportConfig, type ImportConfig, type ImportData } from "../builders/importExport.js";
import { type BaseTaskData } from "./queueFactory.js";

// --------- Task Type Enum ---------

/**
 * Enum of all supported task types in the system.
 * 
 * Queue names are derived from these values. BullMQ, the underlying queue library,
 * restricts certain characters in queue names (e.g., colons are not allowed).
 * This enum uses hyphens as separators (e.g., "email-send") to comply with these restrictions
 * and ensure valid queue naming in Redis.
 */
export enum QueueTaskType {
    EMAIL_SEND = "email-send",
    EXPORT_USER_DATA = "export-user-data",
    IMPORT_USER_DATA = "import-user-data",
    LLM_COMPLETION = "llm-completion",
    PUSH_NOTIFICATION = "push-notification",
    RUN_START = "run-start",
    SANDBOX_EXECUTION = "sandbox-execution",
    SMS_MESSAGE = "sms-message",
    NOTIFICATION_CREATE = "notification-create",
    SWARM_EXECUTION = "swarm-execution",
}

// --------- Base Task Interface ---------

/**
 * Task interface with strong typing for the task type
 */
export interface Task extends BaseTaskData {
    type: QueueTaskType | string; // Allow string for backward compatibility
}

export interface EmailTask extends Task {
    type: QueueTaskType.EMAIL_SEND;
    to: string[];
    subject: string;
    text: string;
    html?: string;
}

export interface ExportUserDataTask extends Task {
    type: QueueTaskType.EXPORT_USER_DATA;
    config: ExportConfig;
}

export interface ImportUserDataTask extends Task {
    type: QueueTaskType.IMPORT_USER_DATA;
    data: ImportData;
    config: ImportConfig;
}

export interface LLMCompletionTask extends Task {
    type: QueueTaskType.LLM_COMPLETION;
    /**
     * The chat we're responding in
     */
    chatId: string;
    /**
     * The messageId of the message we're responding to. 
     * 
     * We pass this instead of the full message state to save space, 
     * since we can grab the state from the cache, and it already needs 
     * to be in the cache for the response generation to work.
     */
    messageId: string;
    /**
     * The model to use when generating the response.
     * 
     * If not provided, the model will be determined by the chat or 
     * bot's configuration.
     */
    model?: string;
    /**
     * Any context data for the task
     */
    taskContexts: TaskContextInfo[];
    /**
     * The user data of the user who triggered the bot response
     */
    userData: SessionUser;
    /**
     * Information about the bot that is responding.
     * This is optional and may not be present for all swarm tasks.
     */
    respondingBot?: { id?: string, publicId?: string, handle?: string };
}

/**
 * New three-tier swarm execution task
 */
export interface SwarmExecutionTask extends Task {
    type: QueueTaskType.SWARM_EXECUTION;
    swarmId: string;
    config: {
        name: string;
        description: string;
        goal: string;
        resources: {
            maxCredits: number;
            maxTokens: number;
            maxTime: number;
            tools: Array<{ name: string; description: string }>;
        };
        config: {
            model: string;
            temperature: number;
            autoApproveTools: boolean;
            parallelExecutionLimit: number;
        };
        organizationId?: string;
    };
    userData: SessionUser;
}

// All swarm-related task types
export type SwarmTask = LLMCompletionTask | SwarmExecutionTask;

export type PushSubscription = {
    endpoint: string;
    keys: {
        p256dh: string;
        auth: string;
    };
}

export type PushPayload = {
    body: string;
    icon?: string | null;
    link: string | null | undefined;
    title: string | null | undefined;
}

export interface PushNotificationTask extends Task {
    type: QueueTaskType.PUSH_NOTIFICATION;
    subscription: PushSubscription;
    payload: PushPayload;
}

// --------- Run Tasks ---------

/**
 * Base object for requesting a run from the server
 */
export interface RunTask extends Task, Pick<RunProgress, "runId"> {
    type: QueueTaskType.RUN_START,
    /**
     * The configuration for the run.
     * 
     * Should be provided if the run is new, but can be omitted if the run is being resumed.
     */
    config?: RunConfig;
    /** Inputs and outputs to use on current step, if not already in run object */
    formValues?: Record<string, unknown>;
    /**
     * Indicates if the run is new.
     */
    isNewRun: boolean;
    /** The run ID */
    resourceVersionId: string;
    /**
     * What triggered the run. This is a factor in determining
     * the queue priority of the task.
     */
    runFrom: RunTriggeredFrom;
    /**
     * ID of the user who started the run. Either a human if it matches the 
     * session user, or a bot if it doesn't.
     */
    startedById: string;
    /** The latest status of the command. */
    status: TaskStatus | `${TaskStatus}`;
    /** The user who's running the command (not the bot) */
    userData: SessionUser;
}

export interface SandboxTask extends Task {
    type: QueueTaskType.SANDBOX_EXECUTION;
    /** The ID of the code version in the database */
    codeVersionId: string;
    /**
     * The input to be passed to the user code.
     * 
     * Examples:
     * - { plainText: "Hello, world!" }
     * - { numbers: [1, 2, 3, 4, 5], operation: "sum" }
     * - [1, 2, 3, 4, 5]
     * - "Hello, world!"
     */
    input?: unknown;
    /**
     * True if the input is an array and should be passed as separate arguments to the user code (i.e. spread). 
     * Defaults to false.
     */
    shouldSpreadInput?: boolean;
    /** The status of the job process */
    status: TaskStatus | `${TaskStatus}`;
    /** The user who's running the command (not the bot) */
    userData: SessionUser;
}

// --------- SMS Tasks ---------

export interface SMSTask extends Task {
    type: QueueTaskType.SMS_MESSAGE;
    to: string[];
    body: string;
}

// --------- Notification Create Task ---------

/**
 * Task for creating an in-app notification record and emitting via WebSocket.
 */
export interface NotificationCreateTask extends Task {
    type: QueueTaskType.NOTIFICATION_CREATE;
    /** ID of the notification record (stringified BigInt) */
    id: string;
    /** User ID to notify */
    userId: string;
    /** Notification category */
    category: string;
    /** The title of the notification */
    title: string;
    /** The body/description of the notification */
    description?: string;
    /** Link to open when clicked */
    link?: string;
    /** Image/icon link */
    imgLink?: string;
    /** Whether to send a websocket event to the user */
    sendWebSocketEvent?: boolean;
}

// --------- All Task Types ---------

/**
 * Union of all possible task types in the system
 */
export type AnyTask =
    | EmailTask
    | ExportUserDataTask
    | ImportUserDataTask
    | SwarmTask
    | PushNotificationTask
    | RunTask
    | SandboxTask
    | SMSTask
    | NotificationCreateTask;

//TODO update these types to be the actual shapes
