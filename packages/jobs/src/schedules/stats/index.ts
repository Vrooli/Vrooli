import { PeriodType } from "@prisma/client";
import { logApiStats } from "./api";
import { logCodeStats } from "./code";
import { logProjectStats } from "./project";
import { logQuizStats } from "./quiz";
import { logRoutineStats } from "./routine";
import { logSiteStats } from "./site";
import { logStandardStats } from "./standard";
import { logTeamStats } from "./team";
import { logUserStats } from "./user";

const HOURS_IN_DAY = 24;
const DAYS_IN_WEEK = 7;
const DAYS_IN_MONTH = 30;
const DAYS_IN_YEAR = 365;

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
            return now.setHours(now.getHours() - HOURS_IN_DAY);
        case PeriodType.Weekly:
            return now.setDate(now.getDate() - DAYS_IN_WEEK);
        case PeriodType.Monthly:
            return now.setDate(now.getDate() - DAYS_IN_MONTH);
        case PeriodType.Yearly:
            return now.setDate(now.getDate() - DAYS_IN_YEAR);
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
export const statsPeriodCron = {
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
