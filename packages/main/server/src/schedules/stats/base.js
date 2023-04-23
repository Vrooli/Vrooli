import cron from "node-cron";
import { logger } from "../../events";
import { PeriodType } from "@prisma/client";
import { logApiStats } from "./api";
import { logOrganizationStats } from "./organization";
import { logProjectStats } from "./project";
import { logQuizStats } from "./quiz";
import { logRoutineStats } from "./routine";
import { logSiteStats } from "./site";
import { logSmartContractStats } from "./smartContract";
import { logStandardStats } from "./standard";
import { logUserStats } from "./user";
function getPeriodStart(period) {
    const now = new Date();
    switch (period) {
        case PeriodType.Hourly:
            return now.setHours(now.getHours() - 1);
        case PeriodType.Daily:
            return now.setHours(now.getHours() - 24);
        case PeriodType.Weekly:
            return now.setDate(now.getDate() - 7);
        case PeriodType.Monthly:
            return now.setDate(now.getDate() - 30);
        case PeriodType.Yearly:
            return now.setDate(now.getDate() - 365);
    }
}
export const periodCron = {
    [PeriodType.Hourly]: "3 * * * *",
    [PeriodType.Daily]: "7 5 * * *",
    [PeriodType.Weekly]: "11 5 * * 0",
    [PeriodType.Monthly]: "17 5 1 * *",
    [PeriodType.Yearly]: "21 5 1 1 *",
};
export const initStatsCronJobs = () => {
    logger.info("Initializing stats cron jobs.", { trace: "0209" });
    try {
        for (const [period, schedule] of Object.entries(periodCron)) {
            cron.schedule(schedule, () => {
                logger.info(`Starting ${period} stats cron job.`, { trace: "0216" });
                const periodStart = new Date(getPeriodStart(period)).toISOString();
                const periodEnd = new Date().toISOString();
                const params = [period, periodStart, periodEnd];
                Promise.all([
                    logSiteStats(...params),
                    logApiStats(...params),
                    logOrganizationStats(...params),
                    logProjectStats(...params),
                    logQuizStats(...params),
                    logRoutineStats(...params),
                    logSmartContractStats(...params),
                    logStandardStats(...params),
                    logUserStats(...params),
                ]).then(() => {
                    logger.info(`✅ ${period} stats cron job completed.`, { trace: "0217" });
                });
            });
        }
        logger.info("✅ Stats cron jobs initialized");
    }
    catch (error) {
        logger.error("❌ Failed to initialize stats cron jobs.", { trace: "0382", error });
    }
};
//# sourceMappingURL=base.js.map