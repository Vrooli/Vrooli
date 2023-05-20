/**
 * Handles recalculating the counts for votes, bookmarks, reputation score, and any 
 * other counts that are stored in the database. Ideally these counts are always 
 * accurate, but there could be a bug in the code. If any counts are changed, 
 * an error is logged.
 */
import { cronTimes } from "../cronTimes";
import { initializeCronJob } from "../initializeCronJob";

/**
 * Initializes cron jobs for count checking
 */
export const initCountsCronJobs = () => {
    initializeCronJob(cronTimes.counts, async () => {
        //TODO
    }, "count checking");
};
