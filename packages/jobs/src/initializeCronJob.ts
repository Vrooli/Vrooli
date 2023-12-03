import { logger } from "@local/server";
import cron from "node-cron";

/**
 * Initializes a cron job with error handling and logging.
 *
 * @param cronSchedule - A cron schedule string.
 * @param job - A function that returns a Promise for the cron job.
 * @param description - An optional description for logging purposes.
 */
export const initializeCronJob = (
    cronSchedule: string,
    job: () => Promise<unknown>,
    description = "",
): void => {
    try {
        // Schedule the cron job
        cron.schedule(cronSchedule, () => {
            // Log the start of the cron job execution
            logger.info(`Starting${description ? " " + description : ""} cron job.`, { trace: "0396" });

            // Execute the job and handle its completion
            job().then(() => {
                // Log the successful completion of the cron job
                logger.info(`✅${description ? " " + description : ""} cron job completed.`, { trace: "0397" });
            });
        });
    } catch (error) {
        // Log the error if the cron job initialization failed
        logger.error(`❌ Failed to initialize${description ? " " + description : ""} cron job.`, { error, trace: "0398" });
    }
};
