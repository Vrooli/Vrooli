/**
 * Creates sitemap files and a sitemap index file for public user-generated content. This includes 
 * APIs, notes, organizations, projects, questions, routines, smart contracts, standards, and users.
 * 
 * A cron job is triggered to run this every day at 4:43am (UTC).
 */
import { logger } from "../../events";
import { cronTimes } from "../cronTimes";
import { initializeCronJob } from "../initializeCronJob";
import { genSitemap, genSitemapIfNotExists } from "./genSitemap";

/**
 * Initializes cron jobs for sitemap creation
 */
export const initSitemapCronJob = () => {
    logger.info("Initializing sitemap cron job.", { trace: "0398" });
    try {
        // Generate sitemaps right away if they don't already exist
        genSitemapIfNotExists().then(() => {
            initializeCronJob(cronTimes.sitemaps, genSitemap, "sitemap");
        });
        logger.info("✅ Sitemap cron job initialized");
    } catch (error) {
        logger.error("❌ Failed to initialize sitemap cron job.", { trace: "0401", error });
    }
};
