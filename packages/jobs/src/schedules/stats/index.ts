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


/**
 * Calculates the unix timestamp of periodStart (earliest data to include) 
 * for the given time interval.
 * Hourly is the past hour
 * Daily is the past 24 hours
 * Weekly is the past 7 days
 * Monthly is the past 30 days
 * Yearly is the past 365 days
 * @param period Time interval to calculate the start of
 * @returns Unix timestamp of the start of the given time interval
 */
function getPeriodStart(period: PeriodType) {
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


// Cron syntax created using this website: https://crontab.guru/
// NOTE: Cron jobs start at weird times because: 
// 1. We stagger them to avoid overloading the database
// 2. We assume that users may be triggering routines on the hour/day/etc., 
// and ideally we want to run these jobs while there is less activity
/**
 * Maps PeriodTypes to their corresponding cron jobs
 */
export const periodCron = {
    /**
     * Hourly, at minute 3
     */
    [PeriodType.Hourly]: "3 * * * *",
    /**
     * Daily, at 5:07am (UTC)
     */
    [PeriodType.Daily]: "7 5 * * *",
    /**
     * Weekly, at 5:11am (UTC) on Sunday
     */
    [PeriodType.Weekly]: "11 5 * * 0",
    /**
     * Monthly, at 5:17am (UTC) on the 1st
     */
    [PeriodType.Monthly]: "17 5 1 * *",
    /**
     * Yearly, at 5:21am (UTC) on January 1st
     */
    [PeriodType.Yearly]: "21 5 1 1 *",
} as const;

/**
 * Handles logging of object and site-wide statistics.
 * 
 * Statistics are stored by PeriodType (i.e. hourly, daily, weekly, monthly, yearly), which 
 * each have their own cron job. These are used to calculate line graphs on the frontend, 
 * by combining all matching period rows within a periodStart and periodEnd.
 */

export const initStatsPeriod = (period: string) => {
    const periodStart = new Date(getPeriodStart(period as PeriodType)).toISOString();
    const periodEnd = new Date().toISOString();
    const params = [period as PeriodType, periodStart, periodEnd] as const;
    // Trigger each stat group
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
    ]);
};
