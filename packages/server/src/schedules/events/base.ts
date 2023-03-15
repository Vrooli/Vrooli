/**
 * Handles caching upcoming scheduled events in the database. Events are already stored 
 * in the database using a window, timeZone, and the Recurrence table, but this 
 * approach is not performant for viewing upcoming events on the home page.
 * 
 * Caches are calculated for the next 60 days, and stored as an array of the start and
 * end times of each recurrence.
 * 
 * The cache is called immediately when a new event is created, when an updated event changes 
 * a relevant field (i.e. timeZone, window, or recurrence). A cron job is also used to
 * update the cache every week, for any events that have their window start/end times
 * within the next 60 days.
 * 
 * A separate cron job runs every day to check the upcoming events cache, and send reminder 
 * notifications to any users who have subscribed to an event occurring within the next 26 hours. 
 * For each subscribed user, we check the logic for how many reminders to send (i.e. 1 day before, 
 * 1 hour before, etc.), and set up a Redis job for each reminder. This cron job is also used to
 * trigger automatic routines
 */
import cron from 'node-cron';
import { logger } from '../../events';

// Cron syntax created using this website: https://crontab.guru/
// NOTE: Cron job starts at a weird time so it runs when there is less activity.
// This cron job is run every day at 5:09am (UTC)
const cacheEventsCron = '9 5 * * *';
// This cron job is run every day at 5:23am (UTC)
const checkEventsCron = '23 5 * * *';

/**
 * Initializes cron jobs for event caching. 
 * This is called when the server starts up. 
 * See https://crontab.guru/ for more information on cron jobs.
 */
export const initEventsCronJobs = () => {
    logger.info('Initializing events cron jobs.', { trace: '0391' });
    try {
        // Start cron for caching events
        cron.schedule(cacheEventsCron, () => {
            logger.info('Starting cache events cron job.', { trace: '0392' });
            // TODO: Cache events
        });
        // Run check events cron job immediately, in case 
        // the last check was missed
        // TODO: Check events
        // Then start cron for checking events
        cron.schedule(checkEventsCron, () => {
            logger.info('Starting check events cron job.', { trace: '0393' });
            // TODO: Check events
        });

        logger.info('✅ Events cron jobs initialized');
    } catch (error) {
        logger.error('❌ Failed to initialize events cron jobs.', { trace: '0394', error });
    }
};