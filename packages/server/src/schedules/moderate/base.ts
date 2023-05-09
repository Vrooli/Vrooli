/**
 * Performs moderation checks for reports and pull requests
 * 
 * A cron job is triggered to run this every day at 4:43am (UTC).
 */
import { cronTimes } from "../cronTimes";
import { initializeCronJob } from "../initializeCronJob";
import { checkReportResponses } from "./reports";

/**
 * Initializes cron jobs for moderation
 */
export const initModerationCronJobs = () => {
    initializeCronJob(cronTimes.reports, checkReportResponses, "report moderation");
    // TODO
    // initializeCronJob(cronTimes.pullRequests, Moderate.checkPullRequestResponses, 'pull request moderation');
};
