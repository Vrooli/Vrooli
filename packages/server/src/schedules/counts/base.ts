/**
 * Handles recalculating the counts for votes, bookmarks, reputation score, and any 
 * other counts that are stored in the database. Ideally these counts are always 
 * accurate, but there could be a bug in the code. If any counts are changed, 
 * an error is logged.
 * 
 * This is run infrequently, and is not a critical job.
 */
import cron from "node-cron";
import { logger } from "../../events";

// Cron syntax created using this website: https://crontab.guru/
// NOTE: Cron job starts at a weird time so it runs when there is less activity.
// This cron job is run every month at 5:49am (UTC)
const countsCron = "49 5 * * *";

/**
 * Initializes cron jobs for count checking 
 * This is called when the server starts up. 
 * See https://crontab.guru/ for more information on cron jobs.
 */
export const initCountsCronJobs = () => {
    logger.info("Initializing counts cron jobs.", { trace: "0395" });
    try {
        // Start cron
        cron.schedule(countsCron, () => {
            logger.info("Starting counts cron job.", { trace: "0396" });
            // TODO
        });
        logger.info("✅ Counts cron jobs initialized");
    } catch (error) {
        logger.error("❌ Failed to initialize counts cron jobs.", { trace: "0397", error });
    }
};
