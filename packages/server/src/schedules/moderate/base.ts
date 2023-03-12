/**
 * Performs moderation checks for reports and pull requests
 * 
 * A cron job is triggered to run this every day at 4:43am (UTC).
 */
import cron from 'node-cron';
import { logger } from '../../events';
import { checkReportResponses } from './reports';

// Cron syntax created using this website: https://crontab.guru/
// NOTE: Cron job starts at a weird time so it runs when there is less activity.
// Cron jobs run every day at 4:56am and 4:58am (UTC)
const reportCron = '56 4 * * *';
const pullRequestCron = '58 4 * * *';

/**
 * Initializes cron jobs for moderation
 * This is called when the server starts up. 
 * See https://crontab.guru/ for more information on cron jobs.
 */
export const initModerationCronJobs = () => {
    logger.info('Initializing moderation cron jobs.', { trace: '0403' });
    try {
        // Start crons
        cron.schedule(reportCron, () => {
            logger.info('Starting report moderation cron job.', { trace: '0404' });
            checkReportResponses().then(() => {
                logger.info(`✅ Report moderation completed.`, { trace: '0405' });
            });
        });
        // TODO
        // cron.schedule(pullRequestCron, () => {
        //     logger.info('Starting pull request moderation cron job.', { trace: '0406' });
        //     Moderate.checkPullRequestResponses().then(() => {
        //         logger.info(`✅ Pull request moderation completed.`, { trace: '0407' });
        //     });
        // });
        logger.info('✅ Moderation cron jobs initialized');
    } catch (error) {
        logger.error('❌ Failed to initialize sitemap cron job.', { trace: '0408', error });
    }
};