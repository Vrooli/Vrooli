import { DAYS_1_HOURS, MONTHS_1_DAYS, WEEKS_1_DAYS, YEARS_1_DAYS } from "@local/shared";
import { PeriodType } from "@prisma/client";
import { logApiStats } from "./api.js";
import { logCodeStats } from "./code.js";
import { logProjectStats } from "./project.js";
import { logQuizStats } from "./quiz.js";
import { logRoutineStats } from "./routine.js";
import { logSiteStats } from "./site.js";
import { logStandardStats } from "./standard.js";
import { logTeamStats } from "./team.js";
import { logUserStats } from "./user.js";

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
            return now.setHours(now.getHours() - DAYS_1_HOURS);
        case PeriodType.Weekly:
            return now.setDate(now.getDate() - WEEKS_1_DAYS);
        case PeriodType.Monthly:
            return now.setDate(now.getDate() - MONTHS_1_DAYS);
        case PeriodType.Yearly:
            return now.setDate(now.getDate() - YEARS_1_DAYS);
    }
}

/**
 * Cron times for generating statistics of each period type. 
 * We don't use random off-peak times here because we want to ensure that the stats are generated at the same time every period, 
 * even across server restarts.
 */
export const statsPeriodCron = {
    [PeriodType.Hourly]: "3 * * * *", // Hourly, at minute 3
    [PeriodType.Daily]: "7 5 * * *", // Daily, at 5:07am (UTC)
    [PeriodType.Weekly]: "11 5 * * 0", // Weekly, at 5:11am (UTC) on Sunday
    [PeriodType.Monthly]: "17 5 1 * *", // Monthly, at 5:17am (UTC) on the 1st
    [PeriodType.Yearly]: "21 5 1 1 *", // Yearly, at 5:21am (UTC) on January 1st
} as const;

/**
 * Handles logging of object and site-wide statistics.
 * 
 * Statistics are stored by PeriodType (i.e. hourly, daily, weekly, monthly, yearly), which 
 * each have their own cron job. These are used to calculate line graphs on the frontend, 
 * by combining all matching period rows within a periodStart and periodEnd.
 */

export function initStatsPeriod(cron: string) {
    const period = Object.keys(statsPeriodCron).find(key => statsPeriodCron[key as PeriodType] === cron) as PeriodType;
    const periodStart = new Date(getPeriodStart(period)).toISOString();
    const periodEnd = new Date().toISOString();
    const params = [period, periodStart, periodEnd] as const;
    // Trigger each stat group
    Promise.all([
        logSiteStats(...params),
        logApiStats(...params),
        logTeamStats(...params),
        logProjectStats(...params),
        logQuizStats(...params),
        logRoutineStats(...params),
        logCodeStats(...params),
        logStandardStats(...params),
        logUserStats(...params),
    ]);
}
