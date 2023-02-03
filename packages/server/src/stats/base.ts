/**
 * Handles logging of object and site-wide statistics.
 * 
 * Statistics are stored by PeriodType (i.e. hourly, daily, weekly, monthly, yearly), which 
 * each have their own cron job. These are used to calculate bar graphs on the frontend, 
 * by combining all matching period rows within a periodStart and periodEnd.
 */
import cron from 'node-cron';
import { logger } from '../events';
import { PrismaType } from '../types';
import pkg, { PeriodType } from '@prisma/client';
import { logApiStats } from './api';
import { logOrganizationStats } from './organization';
import { logProjectStats } from './project';
import { logQuizStats } from './quiz';
import { logRoutineStats } from './routine';
import { logSiteStats } from './site';
import { logSmartContractStats } from './smartContract';
import { logStandardStats } from './standard';
import { logUserStats } from './user';
const { PrismaClient } = pkg;

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
    [PeriodType.Hourly]: '3 * * * *',
    /**
     * Daily, at 5:07am (UTC)
     */
    [PeriodType.Daily]: '7 5 * * *',
    /**
     * Weekly, at 5:11am (UTC) on Sunday
     */
    [PeriodType.Weekly]: '11 5 * * 0',
    /**
     * Monthly, at 5:17am (UTC) on the 1st
     */
    [PeriodType.Monthly]: '17 5 1 * *',
    /**
     * Yearly, at 5:21am (UTC) on January 1st
     */
    [PeriodType.Yearly]: '121 5 1 1 *',
} as const;

/**
 * Initializes cron jobs for logging statistics. 
 * This is called when the server starts up. 
 * See https://crontab.guru/ for more information on cron jobs.
 */
export const initStatsCronJobs = () => {
    logger.info('Initializing stats cron jobs.', { trace: '0209' });
    // Start cron for each period
    for (const [period, schedule] of Object.entries(periodCron)) {
        cron.schedule(schedule, () => {
            logger.info(`Starting ${period} stats cron job.`, { trace: '0216' });
            // Trigger each stat group
            Promise.all([
                logSiteStats(period as PeriodType),
                logApiStats(period as PeriodType),
                logOrganizationStats(period as PeriodType),
                logProjectStats(period as PeriodType),
                logQuizStats(period as PeriodType),
                logRoutineStats(period as PeriodType),
                logSmartContractStats(period as PeriodType),
                logStandardStats(period as PeriodType),
                logUserStats(period as PeriodType),
            ]).then(() => {
                logger.info(`✅ ${period} stats cron job completed.`, { trace: '0217' });
            });
        });
    }
    logger.info('✅ Stats cron jobs initialized');
};

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
function getPeriodStart(period: PeriodType): number {
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



// /**
//  * Calculate active users
//  * @param timeInterval Time interval to calculate for
//  * @param prisma PrismaType instance
//  * @returns Number of users who used the app in the given time interval
//  */
// async function calculateActiveUsers(timeInterval: StatTimeInterval, prisma: PrismaType): Promise<number> {
//     // Count users in database who have used the site in the last time interval
//     const activeUsers = await prisma.user.count({
//         where: {
//             lastSessionVerified: { gte: new Date(getTimeIntervalStart(timeInterval)) }
//         },
//     });
//     return activeUsers;
// }

// /**
//  * Calculate verified wallets
//  * @param timeInterval Time interval to calculate active users for
//  * @param prisma PrismaType instance
//  * @returns Number of wallets verified within the given time interval
//  */
// async function calculateVerifiedWallets(timeInterval: StatTimeInterval, prisma: PrismaType): Promise<number> {
//     // Count wallets in database that have been verified in the last time interval
//     const verifiedWallets = await prisma.wallet.count({
//         where: {
//             AND: [
//                 { verified: true },
//                 { lastVerifiedTime: { gte: new Date(getTimeIntervalStart(timeInterval)) } },
//             ]
//         }
//     });
//     return verifiedWallets;
// }

// /**
//  * Calculate verified emails
//  * @param timeInterval Time interval to calculate active users for
//  * @param prisma PrismaType instance
//  * @returns Number of emails verified within the given time interval
//  */
// async function calculateVerifiedEmails(timeInterval: StatTimeInterval, prisma: PrismaType): Promise<number> {
//     // Count emails in database that have been verified in the last time interval
//     const verifiedEmails = await prisma.email.count({
//         where: {
//             AND: [
//                 { verified: true },
//                 { lastVerifiedTime: { gte: new Date(getTimeIntervalStart(timeInterval)) } },
//             ]
//         }
//     });
//     return verifiedEmails;
// }

// /**
//  * Calculate organizations
//  * @param timeInterval Time interval for created_at
//  * @param prisma PrismaType instance
//  * @returns Number of organizations created in the given time interval
//  */
// async function calculateOrganizations(timeInterval: StatTimeInterval, prisma: PrismaType): Promise<number> {
//     // Count organizations in database created in the last time interval
//     const organizations = await prisma.organization.count({
//         where: {
//             created_at: { gte: new Date(getTimeIntervalStart(timeInterval)) }
//         },
//     });
//     return organizations;
// }

// /**
//  * Calculate projects
//  * @param timeInterval Time interval for created_at
//  * @param prisma PrismaType instance
//  * @returns Number of projects created in the given time interval
//  */
// async function calculateProjects(timeInterval: StatTimeInterval, prisma: PrismaType): Promise<number> {
//     // Count projects in database created in the last time interval
//     const projects = await prisma.project.count({
//         where: {
//             created_at: { gte: new Date(getTimeIntervalStart(timeInterval)) }
//         },
//     });
//     return projects;
// }

// /**
//  * Calculate routines
//  * @param timeInterval Time interval for created_at
//  * @param prisma PrismaType instance
//  * @returns Number of routines created in the given time interval
//  */
// async function calculateRoutines(timeInterval: StatTimeInterval, prisma: PrismaType): Promise<number> {
//     // Count routines in database created in the last time interval
//     const routines = await prisma.routine.count({
//         where: {
//             AND: [
//                 { created_at: { gte: new Date(getTimeIntervalStart(timeInterval)) } },
//                 { isInternal: false },
//             ]
//         },
//     });
//     return routines;
// }

// /**
//  * Calculate standards
//  * @param timeInterval Time interval for created_at
//  * @param prisma PrismaType instance
//  * @returns Number of standards created in the given time interval
//  */
// async function calculateStandards(timeInterval: StatTimeInterval, prisma: PrismaType): Promise<number> {
//     // Count standards in database created in the last time interval
//     const standards = await prisma.standard.count({
//         where: {
//             created_at: { gte: new Date(getTimeIntervalStart(timeInterval)) }
//         },
//     });
//     return standards;
// }

// /**
//  * Calculate all statistics for a given time interval
//  * @param timeInterval Time interval to calculate statistics for
//  * @return An object containing all statistics for the given time interval, 
//  * where each key is a StatType, and the value is the calculated statistic as a number
//  */
// async function calculateStats(timeInterval: StatTimeInterval): Promise<{ [key in StatType]: number } & { timestamp: number } | undefined> {
//     logger.info('Starting to calculate site statistics', { trace: '0211' });
//     const prisma = new PrismaClient();
//     let results: any = undefined;
//     try {
//         results = {
//             timestamp: Date.now(),
//             [StatType.ActiveUsers]: await calculateActiveUsers(timeInterval, prisma),
//             [StatType.VerifiedWallets]: await calculateVerifiedWallets(timeInterval, prisma),
//             [StatType.VerifiedEmails]: await calculateVerifiedEmails(timeInterval, prisma),
//             [StatType.Organizations]: await calculateOrganizations(timeInterval, prisma),
//             [StatType.Projects]: await calculateProjects(timeInterval, prisma),
//             [StatType.Routines]: await calculateRoutines(timeInterval, prisma),
//             // [StatType.RoutinesCompleted]: await calculateRoutinesCompleted(timeInterval, prisma),
//             // [StatType.RoutinesCompletedTimeTotal]: await calculateRoutinesCompletedTimeTotal(timeInterval, prisma),
//             // [StatType.RoutinesCompletedTimeAverage]: await calculateRoutinesCompletedTimeAverage(timeInterval, prisma),
//             [StatType.Standards]: await calculateStandards(timeInterval, prisma),
//         }
//         prisma.$disconnect();
//     } catch (error) {
//         logger.error('Caught error calculating stats', { trace: '0000', error });
//     } finally {
//         return results;
//     }
// }

// /**
//  * Calculates statistics for the specified time interval, and appends them to the correct files. 
//  * Each statistic type is stored in its own file (and folder).
//  * If the time interval has been completed, then new files are created.
//  * @param timeInterval The time interval the statistic is being calculated for.
//  */
// async function logStats(timeInterval: StatTimeInterval) {
//     try {
//         // // Query the database for the statistics relevant to this time interval
//         // const stats = await calculateStats(timeInterval);
//         // // Create and save new MongoDB object for this time interval
//         // switch (timeInterval) {
//         //     case StatTimeInterval.Daily:
//         //         const dailyStats = new StatDay(stats);
//         //         await dailyStats.save();
//         //         break;
//         //     case StatTimeInterval.Weekly:
//         //         const weeklyStats = new StatWeek(stats);
//         //         await weeklyStats.save();
//         //         break;
//         //     case StatTimeInterval.Monthly:
//         //         const monthlyStats = new StatMonth(stats);
//         //         await monthlyStats.save();
//         //         break;
//         //     case StatTimeInterval.Yearly:
//         //         const yearlyStats = new StatYear(stats);
//         //         await yearlyStats.save();
//         //         break;
//         //     case StatTimeInterval.AllTime:
//         //         const allTimeStats = new StatAllTime(stats);
//         //         await allTimeStats.save();
//         //         break;
//         // }
//     } catch (error) {
//         logger.error('Caught error logging stats', { trace: '0192', error });
//     }
// }