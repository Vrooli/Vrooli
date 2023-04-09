/**
 * Handles sending push notifications for upcoming scheduled events in the database
 */
import cron from 'node-cron';
import { logger } from '../../events';
import { scheduleNotify } from './scheduleNotify';

// Cron syntax created using this website: https://crontab.guru/
// NOTE: Cron job starts at a weird time so it runs when there is less activity.
// This cron job is run every day at 5:09am (UTC)
const eventsCron = '9 5 * * *';

/**
 * Initializes cron jobs for schedule event notifications.
 * This is called when the server starts up. 
 * See https://crontab.guru/ for more information on cron jobs.
 */
export const initEventsCronJobs = () => {
    logger.info('Initializing events cron jobs.', { trace: '0391' });
    try {
        // Start cron
        cron.schedule(eventsCron, () => {
            logger.info('Starting schedule events cron job.', { trace: '0392' });
            scheduleNotify().then(() => {
                logger.info('✅ Schedule events cron job completed.', { trace: '0393' });
            });
        });
        logger.info('✅ Events cron jobs initialized');
    } catch (error) {
        logger.error('❌ Failed to initialize events cron jobs.', { trace: '0394', error });
    }
};