import cron from "node-cron";
import { logger } from "../../events";
import { expirePremium } from "./expirePremium";
const expirePremiumCron = "20 4 * * *";
export const initExpirePremiumCronJob = () => {
    logger.info("Initializing expire premium cron job.", { trace: "0438" });
    try {
        cron.schedule(expirePremiumCron, () => {
            logger.info("Starting expire premium cron job.", { trace: "0439" });
            expirePremium().then(() => {
                logger.info("✅ Expire premium cron job completed.", { trace: "0440" });
            });
        });
        logger.info("✅ Expire premium cron job initialized");
    }
    catch (error) {
        logger.error("❌ Failed to initialize expire premium cron job.", { trace: "0441", error });
    }
};
//# sourceMappingURL=base.js.map