import cron from "node-cron";
import { logger } from "../../events";
import { scheduleNotify } from "./scheduleNotify";
const eventsCron = "9 5 * * *";
export const initEventsCronJobs = () => {
    logger.info("Initializing events cron jobs.", { trace: "0391" });
    try {
        cron.schedule(eventsCron, () => {
            logger.info("Starting schedule events cron job.", { trace: "0392" });
            scheduleNotify().then(() => {
                logger.info("✅ Schedule events cron job completed.", { trace: "0393" });
            });
        });
        logger.info("✅ Events cron jobs initialized");
    }
    catch (error) {
        logger.error("❌ Failed to initialize events cron jobs.", { trace: "0394", error });
    }
};
//# sourceMappingURL=base.js.map