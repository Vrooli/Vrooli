/**
 * Creates a sitemap files and a sitemap index file for public user-generated content. This includes 
 * APIs, notes, organizations, projects, questions, routines, smart contracts, standards, and users.
 * 
 * A cron job is triggered to run this every day at 4:43am (UTC).
 */
import cron from 'node-cron';
import { logger } from '../../events';
import { genSitemap } from './genSitemap';

// Cron syntax created using this website: https://crontab.guru/
// NOTE: Cron job starts at a weird time so it runs when there is less activity.
// This cron job is run every day at 4:43am (UTC)
const sitemapCron = '43 4 * * *';

/**
 * Initializes cron jobs for sitemap creation 
 * This is called when the server starts up. 
 * See https://crontab.guru/ for more information on cron jobs.
 */
export const initSitemapCronJob = () => {
    logger.info('Initializing sitemap cron job.', { trace: '0398' });
    try {
        genSitemap().then(() => {
            logger.info(`✅ Sitemap cron job completed.`, { trace: '0399' });
        })
        // Start cron for caching events
        cron.schedule(sitemapCron, () => {
            logger.info('Starting stiemap cron job.', { trace: '0400' });
            // TODO: Cache events
        });
        logger.info('✅ Sitemap cron job initialized');
    } catch (error) {
        logger.error('❌ Failed to initialize sitemap cron job.', { trace: '0401', error });
    }
};