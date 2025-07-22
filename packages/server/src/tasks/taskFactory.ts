/**
 * Centralized test factory for creating valid task data with all required base properties.
 * This ensures type safety and consistency across all task-related tests.
 */

import { generatePK } from "@vrooli/shared";
import { type Job } from "bullmq";
import { type Task, type QueueTaskType, type BaseTaskData } from "./taskTypes.js";

/**
 * Creates valid task data with all required base properties.
 * 
 * @param type - The queue task type
 * @param specificData - Task-specific data (without base properties)
 * @param baseOverrides - Optional overrides for base properties
 * @returns Complete task data with all required properties
 * 
 * @example
 * const emailTask = createValidTaskData(
 *   QueueTaskType.EMAIL_SEND,
 *   { to: ["test@example.com"], subject: "Test", text: "Hello" }
 * );
 */
export function createValidTaskData<T extends Task>(
    type: QueueTaskType,
    specificData: Omit<T, keyof BaseTaskData>,
    baseOverrides?: Partial<BaseTaskData>,
): T {
    return {
        id: baseOverrides?.id || generatePK().toString(),
        type,
        userId: baseOverrides?.userId,
        status: baseOverrides?.status,
        ...specificData,
    } as T;
}

/**
 * Creates a mock BullMQ job with proper task data including required base properties.
 * 
 * @param taskType - The queue task type
 * @param taskData - Partial task data (will be merged with defaults)
 * @param jobOverrides - Optional overrides for job properties
 * @returns Mock BullMQ job with valid task data
 * 
 * @example
 * const job = createMockJob(
 *   QueueTaskType.EMAIL_SEND,
 *   { to: ["test@example.com"], subject: "Test", text: "Hello" }
 * );
 */
export function createMockJob<T extends Task>(
    taskType: QueueTaskType,
    taskData: Partial<T>,
    jobOverrides: Partial<Job<T>> = {},
): Job<T> {
    const defaultData = {
        id: taskData.id || generatePK().toString(),
        type: taskType,
        userId: taskData.userId,
        status: taskData.status,
        ...taskData,
    } as T;
    
    return {
        id: jobOverrides.id || `test-job-${Date.now()}`,
        data: defaultData,
        name: taskType,
        attemptsMade: jobOverrides.attemptsMade || 0,
        opts: jobOverrides.opts || {},
        timestamp: jobOverrides.timestamp || Date.now(),
        returnvalue: jobOverrides.returnvalue,
        stacktrace: jobOverrides.stacktrace || [],
        progress: jobOverrides.progress || 0,
        ...jobOverrides,
    } as Job<T>;
}

/**
 * Creates task data with a deduplication ID pattern.
 * Use this for tasks that should override existing jobs.
 * 
 * @param type - The queue task type
 * @param userId - User ID for the deduplication pattern
 * @param action - Action identifier for the deduplication pattern
 * @param specificData - Task-specific data
 * @returns Task data with deduplication ID
 * 
 * @example
 * const task = createTaskWithDeduplicationId(
 *   QueueTaskType.EMAIL_SEND,
 *   "user-123",
 *   "reset-password",
 *   { to: ["user@example.com"], subject: "Reset", text: "..." }
 * );
 */
export function createTaskWithDeduplicationId<T extends Task>(
    type: QueueTaskType,
    userId: string,
    action: string,
    specificData: Omit<T, keyof BaseTaskData>,
): T {
    return createValidTaskData(type, specificData, {
        id: `${userId}:${action}`,
        userId,
    });
}
