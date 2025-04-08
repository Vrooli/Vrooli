import { HOURS_1_S, Success, TaskStatus, TaskStatusInfo } from "@local/shared";
import Bull, { Queue } from "bull";
import path from "path";
import { fileURLToPath } from "url";
import { logger } from "../events/logger.js";

const dirname = path.dirname(fileURLToPath(import.meta.url));
const importExtension = ".js";

export const LOGGER_PATH = path.join(dirname, "../events/logger" + importExtension);
export const REDIS_CONN_PATH = path.join(dirname, "../redisConn" + importExtension);
export const SERVER_PATH = path.join(dirname, "../server" + importExtension);
export function getProcessPath(folder: string) {
    return path.join(dirname, `./${folder}/process${importExtension}`);
}

export const DEFAULT_JOB_OPTIONS = {
    removeOnComplete: {
        age: HOURS_1_S,
        count: 10_000,
    },
    removeOnFail: {
        age: HOURS_1_S,
        count: 10_000,
    },
} as const;

/**
 * Helper function to add a job to a Bull queue.
 * @param queue The Bull queue instance.
 * @param data The data to be processed by the job.
 * @param options Options for the job, such as jobId and timeout.
 * @returns Indicates if the job was successfully queued.
 */
export async function addJobToQueue<T>(
    queue: Bull.Queue<T>,
    data: T,
    options: Bull.JobOptions,
): Promise<Success> {
    try {
        const job = await queue.add(data, options);
        if (job) {
            // NOTE: If this is returning true but the job isn't being run, 
            // make sure that you're giving the jobs unique jobIds.
            return { __typename: "Success" as const, success: true };
        } else {
            logger.error("Failed to queue the task.", { trace: "0550", data });
            return { __typename: "Success" as const, success: false };
        }
    } catch (error) {
        logger.error("Error adding task to queue", { trace: "0551", error, data });
        return { __typename: "Success" as const, success: false };
    }
}

type TaskQueue = Queue<{
    __process: string;
    status: TaskStatus | `${TaskStatus}`;
    userData: { id: string };
} | { __process: "Test" }>

/**
 * Get the statuses of multiple tasks from a specified queue.
 * @param queue The queue to fetch jobs from.
 * @param taskIds Array of task IDs for which to fetch the statuses.
 * @returns Promise that resolves to an array of objects with task ID and status.
 */
export async function getTaskStatuses(queue: TaskQueue, taskIds: string[]): Promise<TaskStatusInfo[]> {
    const taskStatusInfos = await Promise.all(taskIds.map(async (taskId) => {
        try {
            const job = await queue.getJob(taskId);
            if (job && job.data.__process !== "Test") {
                return { __typename: "TaskStatusInfo" as const, id: taskId, status: (job.data as { status: TaskStatus }).status };
            } else {
                return { __typename: "TaskStatusInfo" as const, id: taskId, status: null };  // Task not found, return null status
            }
        } catch (error) {
            console.error(`Failed to retrieve job ${taskId}:`, error);
            return { __typename: "TaskStatusInfo" as const, id: taskId, status: null };  // Return null if there is an error fetching the job
        }
    }));

    return taskStatusInfos;
}

/**
 * Update a task's status
 * @param queue The queue containing the job to update
 * @param jobId The job ID (also the task ID) of the task
 * @param status The new status of the task
 * @param userId The user ID of the user who triggered the task. 
 * Only they are allowed to change the status of the task.
 */
export async function changeTaskStatus(
    queue: TaskQueue,
    jobId: string,
    status: TaskStatus | `${TaskStatus}`,
    userId: string,
): Promise<Success> {
    try {
        const job = await queue.getJob(jobId);
        if (!job) {
            // If job isn't found but we're changing to a start or end state, consider it a success
            if (["Completed", "Failed", "Suggested"].includes(status)) {
                logger.info(`Task with jobId ${jobId} not found, but considered a success.`);
                return { __typename: "Success" as const, success: true };
            }
            // Otherwise, consider it an error 
            else {
                logger.error(`Task with jobId ${jobId} not found.`);
                return { __typename: "Success" as const, success: false };
            }
        }

        // Check if the user has permission to change the status
        if ((job.data as { userData: { id: string } }).userData && (job.data as { userData: { id: string } }).userData.id !== userId) {
            logger.error(`User ${userId} is not authorized to change the status of job ${jobId}.`);
            return { __typename: "Success" as const, success: false };
        }

        // Update the job status
        await job.update({ ...job.data, status });
        return { __typename: "Success" as const, success: true };
    } catch (error) {
        logger.error(`Failed to change status for task with jobId ${jobId}.`, { error });
        return { __typename: "Success" as const, success: false };
    }
}
