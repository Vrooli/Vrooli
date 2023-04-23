import cron from "node-cron";
import { logger } from "../../events";
import { genSitemap, genSitemapIfNotExists } from "./genSitemap";
const sitemapCron = "43 4 * * *";
export const initSitemapCronJob = () => {
    logger.info("Initializing sitemap cron job.", { trace: "0398" });
    try {
        genSitemapIfNotExists().then(() => {
            cron.schedule(sitemapCron, () => {
                logger.info("Starting stiemap cron job.", { trace: "0400" });
                genSitemap().then(() => {
                    logger.info("✅ Sitemap cron job completed.", { trace: "0399" });
                });
            });
        });
        logger.info("✅ Sitemap cron job initialized");
    }
    catch (error) {
        logger.error("❌ Failed to initialize sitemap cron job.", { trace: "0401", error });
    }
};
//# sourceMappingURL=base.js.map