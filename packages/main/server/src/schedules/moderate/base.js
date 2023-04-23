import cron from "node-cron";
import { logger } from "../../events";
import { checkReportResponses } from "./reports";
const reportCron = "56 4 * * *";
const pullRequestCron = "58 4 * * *";
export const initModerationCronJobs = () => {
    logger.info("Initializing moderation cron jobs.", { trace: "0403" });
    try {
        cron.schedule(reportCron, () => {
            logger.info("Starting report moderation cron job.", { trace: "0404" });
            checkReportResponses().then(() => {
                logger.info("✅ Report moderation completed.", { trace: "0405" });
            });
        });
        logger.info("✅ Moderation cron jobs initialized");
    }
    catch (error) {
        logger.error("❌ Failed to initialize sitemap cron job.", { trace: "0408", error });
    }
};
//# sourceMappingURL=base.js.map