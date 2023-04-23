import cron from "node-cron";
import { logger } from "../../events";
const countsCron = "49 5 * * *";
export const initCountsCronJobs = () => {
    logger.info("Initializing counts cron jobs.", { trace: "0395" });
    try {
        cron.schedule(countsCron, () => {
            logger.info("Starting counts cron job.", { trace: "0396" });
        });
        logger.info("✅ Counts cron jobs initialized");
    }
    catch (error) {
        logger.error("❌ Failed to initialize counts cron jobs.", { trace: "0397", error });
    }
};
//# sourceMappingURL=base.js.map