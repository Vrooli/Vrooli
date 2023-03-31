/**
 * Removes expired premium status from users. This might already be handled by Stripe, but this is
 * a backup in case something goes wrong.
 * 
 * A cron job is triggered to run this every day at 4:20am (UTC).
 */
import cron from 'node-cron';
import { logger } from '../../events';
import { expirePremium } from './expirePremium';

// Cron syntax created using this website: https://crontab.guru/
// NOTE: Cron job starts at a weird time so it runs when there is less activity.
// This cron job is run every day at 4:20am (UTC)
const expirePremiumCron = '20 4 * * *';

/**
 * Initializes cron jobs for expiring premium status
 * This is called when the server starts up. 
 * See https://crontab.guru/ for more information on cron jobs.
 */
export const initExpirePremiumCronJob = () => {
    logger.info('Initializing expire premium cron job.', { trace: '0438' });
    try {
        // Start cron for expiring premium status
        cron.schedule(expirePremiumCron, () => {
            logger.info('Starting expire premium cron job.', { trace: '0439' });
            expirePremium().then(() => {
                logger.info(`✅ Expire premium cron job completed.`, { trace: '0440' });
            })
        });
        logger.info('✅ Expire premium cron job initialized');
    } catch (error) {
        logger.error('❌ Failed to initialize expire premium cron job.', { trace: '0441', error });
    }
};