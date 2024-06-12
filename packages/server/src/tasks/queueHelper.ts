import { Success } from "@local/shared";
import Bull from "bull";
import { logger } from "../events/logger";

/**
 * Helper function to add a job to a Bull queue.
 * @param queue The Bull queue instance.
 * @param data The data to be processed by the job.
 * @param options Options for the job, such as jobId and timeout.
 * @returns Indicates if the job was successfully queued.
 */
export const addJobToQueue = async <T>(
    queue: Bull.Queue<T>,
    data: T,
    options: Bull.JobOptions,
): Promise<Success> => {
    try {
        const job = await queue.add(data, options);
        if (job) {
            return { __typename: "Success" as const, success: true };
        } else {
            logger.error("Failed to queue the task.", { trace: "0550", data });
            return { __typename: "Success" as const, success: false };
        }
    } catch (error) {
        logger.error("Error adding task to queue", { trace: "0551", error, data });
        return { __typename: "Success" as const, success: false };
    }
};
