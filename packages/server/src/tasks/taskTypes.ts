/**
 * Task type definitions for the queue system
 * 
 * This file defines all supported task types and their data shapes for type safety.
 * Using discriminated union types to ensure tasks are properly shaped based on their type.
 */

import { TaskContextInfo, type SessionUser } from "@local/shared";
import { type ExportConfig, type ImportConfig, type ImportData } from "../builders/importExport.js";
import { type BaseTaskData } from "./queueFactory.js";

// --------- Task Type Enum ---------

/**
 * Enum of all supported task types in the system
 */
export enum QueueTaskType {
    EMAIL_SEND = "email:send",
    EXPORT_USER_DATA = "export:user-data",
    IMPORT_USER_DATA = "import:user-data",
    LLM_COMPLETION = "llm:completion",
    PUSH_NOTIFICATION = "push:notification",
    RUN_WORKFLOW = "run:workflow",
    RUN_NODE = "run:node",
    SANDBOX_EXECUTION = "sandbox:execution",
    SMS_MESSAGE = "sms:message",
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
}

// All LLM task types
export type LLMTask = LLMCompletionTask;

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

export interface RunTaskBase extends Task {
    type: QueueTaskType.RUN_WORKFLOW | QueueTaskType.RUN_NODE;
    runId: string;
}

export interface RunWorkflowTask extends RunTaskBase {
    type: QueueTaskType.RUN_WORKFLOW;
    workflowId: string;
    inputs: Record<string, any>;
}

export interface RunNodeTask extends RunTaskBase {
    type: QueueTaskType.RUN_NODE;
    nodeId: string;
    inputs: Record<string, any>;
}

// All run task types
export type RunTask = RunWorkflowTask | RunNodeTask;

// --------- Sandbox Tasks ---------

export interface SandboxExecutionTask extends Task {
    type: QueueTaskType.SANDBOX_EXECUTION;
    code: string;
    context: Record<string, any>;
    timeout?: number;
}

// --------- SMS Tasks ---------

export interface SMSTask extends Task {
    type: QueueTaskType.SMS_MESSAGE;
    to: string[];
    body: string;
}

// --------- All Task Types ---------

/**
 * Union of all possible task types in the system
 */
export type AnyTask =
    | EmailTask
    | ExportUserDataTask
    | ImportUserDataTask
    | LLMTask
    | PushNotificationTask
    | RunTask
    | SandboxExecutionTask
    | SMSTask;

//TODO update these types to be the actual shapes
