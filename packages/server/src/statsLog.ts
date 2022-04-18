/**
 * Handles logging of statistics for the website as a whole.
 * Statistics for individual users/organizations/routines/etc. are handled elsewhere.
 * Statistic types handled here are: 
 * - Active users
 * - Verified wallets
 * - Verified emails
 * - Number of organizations
 * - Number of projects
 * - Number of routines
 * - Number of standards
 * - Number of routines completed
 * - Total time spent executing routines (which were completed)
 * - Average time spent executing routines (per completed routine)
 * 
 * Each statistic type has a daily/weekly/monthly/yearly variant. Each variant is stored in a separate file, and is updated about 100 times in its respective time interval.
 * The all-time variant is stored in a separate file, and is updated once per day.
 * 
 * Each statistic also calculates its velocity (i.e. how much the stat changed since the last update) and 
 * its acceleration (i.e. how much the velocity changed since the last update).
 */
import cron from 'node-cron';
import fs from 'fs';
import { PrismaType } from './types';
import pkg from '@prisma/client';
const { PrismaClient } = pkg;

// Location of persistent storage directory
const STAT_DIR = `${process.env.PROJECT_DIR}/data/stats`;

// Daily triggered every 15 minutes
const dailyCron = '*/15 * * * *';
// Weekly triggered every hour
const weeklyCron = '0 * * * *';
// Monthly triggered 4 times per day
const monthlyCron = '0 0 4 * *';
// Yearly (and all-time) triggered once per day
const yearlyCron = '0 0 * * *';

/**
 * Types of stats that can be calculated
 */
export enum StatType {
    ActiveUsers = 'activeUsers',
    VerifiedWallets = 'verifiedWallets',
    VerifiedEmails = 'verifiedEmails',
    Organizations = 'organizations',
    Projects = 'projects',
    Routines = 'routines',
    // RoutinesCompleted = 'routinesCompleted',
    // RoutinesCompletedTimeTotal = 'routinesCompletedTimeTotal',
    // RoutinesCompletedTimeAverage = 'routinesCompletedTimeAverage',
    Standards = 'standards',
}

/**
 * Stat intervals
 */
export enum StatTimeInterval {
    Daily = 'daily',
    Weekly = 'weekly',
    Monthly = 'monthly',
    Yearly = 'yearly',
    AllTime = 'allTime',
}

/**
 * Type of meta-stat
 */
export enum StatMotion {
    /**
     * The actual stat
     */
    Position = 'position',
    /**
     * A measurement of how much the stat changed since the last update
     */
    Velocity = 'velocity',
    /**
     * A measurement of how much the stat change has changed since the last update
     */
    Acceleration = 'acceleration',
}

/**
 * Calculates the unix timestamp of the start of the given time interval
 * Daily is the start of today
 * Weekly is the start of the current week
 * Monthly is the start of the current month
 * Yearly is the start of the current year
 * All-time is 0
 * @param timeInterval Time interval to calculate the start of
 * @returns Unix timestamp of the start of the given time interval
 */
function getTimeIntervalStart(timeInterval: StatTimeInterval): number {
    switch (timeInterval) {
        // Daily is easy, since it's the current day
        case StatTimeInterval.Daily:
            const today = new Date();
            return today.setHours(0, 0, 0, 0);
        // Weekly is the first day of the current week
        case StatTimeInterval.Weekly:
            const firstDayOfWeek = new Date();
            return firstDayOfWeek.setDate(firstDayOfWeek.getDate() - firstDayOfWeek.getDay());
        // Monthly is the first day of the current month    
        case StatTimeInterval.Monthly:
            const firstDayOfMonth = new Date();
            return firstDayOfMonth.setDate(0);
        // Yearly is the first day of the current year
        case StatTimeInterval.Yearly:
        case StatTimeInterval.AllTime:
            const firstDayOfYear = new Date();
            firstDayOfYear.setMonth(0);
            return firstDayOfYear.setDate(0);
    }
}

/**
 * Calculates the time string for an interval's file name. 
 * Daily is the current day 
 * Weekly is the first day of the current week
 * Monthly is the first day of the current month
 * Yearly is the first day of the current year
 * All-time is the first day of the current year
 * @param timeInterval Time interval to calculate the time string for
 * @returns String representing the current time interval file (e.g. 20220420)
 */
 function getTimeString(timeInterval: StatTimeInterval): string {
    // Get the timestamp of the start of the current time interval
    // Except for all-time, where we want the start of the current year
    const start = getTimeIntervalStart(timeInterval === StatTimeInterval.AllTime ? StatTimeInterval.Yearly : timeInterval);
    return (new Date(start).toISOString().split('T')[0]).replace(/-/g, '')
}

/**
 * Calculate active users
 * @param timeInterval Time interval to calculate for
 * @param prisma PrismaType instance
 * @returns Number of users who used the app in the given time interval
 */
async function calculateActiveUsers(timeInterval: StatTimeInterval, prisma: PrismaType): Promise<number> {
    console.log('activeusers start', timeInterval)
    console.log(getTimeIntervalStart(timeInterval), new Date(getTimeIntervalStart(timeInterval)) )
    // Count users in database who have used the site in the last time interval
    const activeUsers = await prisma.user.count({
        where: {
            lastSessionVerified: { gte: new Date(getTimeIntervalStart(timeInterval)) }
        },
    });
    return activeUsers;
}

/**
 * Calculate verified wallets
 * @param timeInterval Time interval to calculate active users for
 * @param prisma PrismaType instance
 * @returns Number of wallets verified within the given time interval
 */
async function calculateVerifiedWallets(timeInterval: StatTimeInterval, prisma: PrismaType): Promise<number> {
    // Count wallets in database that have been verified in the last time interval
    const verifiedWallets = await prisma.wallet.count({
        where: {
            AND: [
                { verified: true },
                { lastVerifiedTime: { gte: new Date(getTimeIntervalStart(timeInterval)) } },
            ]
        }
    });
    return verifiedWallets;
}

/**
 * Calculate verified emails
 * @param timeInterval Time interval to calculate active users for
 * @param prisma PrismaType instance
 * @returns Number of emails verified within the given time interval
 */
async function calculateVerifiedEmails(timeInterval: StatTimeInterval, prisma: PrismaType): Promise<number> {
    // Count emails in database that have been verified in the last time interval
    const verifiedEmails = await prisma.email.count({
        where: {
            AND: [
                { verified: true },
                { lastVerifiedTime: { gte: new Date(getTimeIntervalStart(timeInterval)) } },
            ]
        }
    });
    return verifiedEmails;
}

/**
 * Calculate organizations
 * @param timeInterval Time interval for created_at
 * @param prisma PrismaType instance
 * @returns Number of organizations created in the given time interval
 */
async function calculateOrganizations(timeInterval: StatTimeInterval, prisma: PrismaType): Promise<number> {
    // Count organizations in database created in the last time interval
    const organizations = await prisma.organization.count({
        where: {
            created_at: { gte: new Date(getTimeIntervalStart(timeInterval)) }
        },
    });
    return organizations;
}

/**
 * Calculate projects
 * @param timeInterval Time interval for created_at
 * @param prisma PrismaType instance
 * @returns Number of projects created in the given time interval
 */
async function calculateProjects(timeInterval: StatTimeInterval, prisma: PrismaType): Promise<number> {
    // Count projects in database created in the last time interval
    const projects = await prisma.project.count({
        where: {
            created_at: { gte: new Date(getTimeIntervalStart(timeInterval)) }
        },
    });
    return projects;
}

/**
 * Calculate routines
 * @param timeInterval Time interval for created_at
 * @param prisma PrismaType instance
 * @returns Number of routines created in the given time interval
 */
async function calculateRoutines(timeInterval: StatTimeInterval, prisma: PrismaType): Promise<number> {
    // Count routines in database created in the last time interval
    const routines = await prisma.routine.count({
        where: {
            created_at: { gte: new Date(getTimeIntervalStart(timeInterval)) }
        },
    });
    return routines;
}

/**
 * Calculate standards
 * @param timeInterval Time interval for created_at
 * @param prisma PrismaType instance
 * @returns Number of standards created in the given time interval
 */
async function calculateStandards(timeInterval: StatTimeInterval, prisma: PrismaType): Promise<number> {
    // Count standards in database created in the last time interval
    const standards = await prisma.standard.count({
        where: {
            created_at: { gte: new Date(getTimeIntervalStart(timeInterval)) }
        },
    });
    return standards;
}

/**
 * Calculate all statistics for a given time interval
 * @param timeInterval Time interval to calculate statistics for
 * @return An object containing all statistics for the given time interval, 
 * where each key is a StatType, and the value is the calculated statistic as a number
 */
async function calculateStats(timeInterval: StatTimeInterval): Promise<{ [key in StatType]: number }> {
    console.log('calculatestats start', timeInterval);
    const prisma = new PrismaClient();
    console.log('got prisma')
    return {
        [StatType.ActiveUsers]: await calculateActiveUsers(timeInterval, prisma),
        [StatType.VerifiedWallets]: await calculateVerifiedWallets(timeInterval, prisma),
        [StatType.VerifiedEmails]: await calculateVerifiedEmails(timeInterval, prisma),
        [StatType.Organizations]: await calculateOrganizations(timeInterval, prisma),
        [StatType.Projects]: await calculateProjects(timeInterval, prisma),
        [StatType.Routines]: await calculateRoutines(timeInterval, prisma),
        // [StatType.RoutinesCompleted]: await calculateRoutinesCompleted(timeInterval, prisma),
        // [StatType.RoutinesCompletedTimeTotal]: await calculateRoutinesCompletedTimeTotal(timeInterval, prisma),
        // [StatType.RoutinesCompletedTimeAverage]: await calculateRoutinesCompletedTimeAverage(timeInterval, prisma),
        [StatType.Standards]: await calculateStandards(timeInterval, prisma),
    }
}

/**
 * Calculates statistics for the specified time interval, and appends them to the correct files. 
 * Each statistic type is stored in its own file (and folder).
 * If the time interval has been completed, then new files are created.
 * @param timeInterval The time interval the statistic is being calculated for.
 */
async function logStats(timeInterval: StatTimeInterval) {
    console.log('logstats start', timeInterval);
    // Query the database for the statistics relevant to this time interval
    const stats = await calculateStats(timeInterval);
    console.log('logstats stats', JSON.stringify(stats));
    // Determine time string for file name (e.g. daily -> '-daily-20200101', weekly -> '-weekly-20200101')
    let timeString = `-${timeInterval}-${getTimeString(timeInterval)}`;
    console.log('logstats timeString', timeString);
    // If STAT_DIR does not exist, create it
    if (!fs.existsSync(STAT_DIR)) {
        console.log('logstats mkdir triggered stat_dir', STAT_DIR);
        fs.mkdirSync(STAT_DIR);
    }
    // Loop through each calculated stat, and store it in the correct location
    Object.entries(stats).forEach(([statType, stat]) => {
        console.log('in loop', statType, stat);
        // Determine the file name for the statistic
        const fileName = `${statType}${timeString}.json`;
        // Determine the file path for the statistic
        const filePath = `${STAT_DIR}/${statType}/${fileName}`;
        // Determine the folder path for the statistic
        const folderPath = `${STAT_DIR}/${statType}`;
        console.log('logstats filepath', filePath);
        // If the folder doesn't exist, create it
        if (!fs.existsSync(folderPath)) {
            console.log('logstats mkdir triggered folderPath', folderPath);
            fs.mkdirSync(folderPath);
        }
        // If the file doesn't exist, create it
        if (!fs.existsSync(filePath)) {
            console.log('writing to filePath', filePath);
            fs.writeFileSync(filePath, JSON.stringify(stat));
        }
        // Otherwise, append the new stat to the file
        else {
            console.log('appending to filePath', filePath);
            fs.appendFileSync(filePath, JSON.stringify(stat));
        }
    });
}

/**
 * Initializes cron jobs for logging statistics. 
 * This is called when the server starts up. 
 * Each statistics type has a cron for each time interval, which is triggered 
 * about 100 times in that time interval. The exception is all-time statistics,
 * which are triggered once per day. 
 * See https://crontab.guru/ for more information on cron jobs.
 */
export const initStatsCronJobs = () => {
    // Daily
    cron.schedule(dailyCron, () => {
        console.log('daily start')
        logStats(StatTimeInterval.Daily);
    });
    // Weekly
    cron.schedule(weeklyCron, () => {
        console.log('weekly start')
        logStats(StatTimeInterval.Weekly);
    });
    // Monthly
    cron.schedule(monthlyCron, () => {
        console.log('monthly start')
        logStats(StatTimeInterval.Monthly);
    });
    // Yearly/All-time (they use the same time interval)
    cron.schedule(yearlyCron, () => {
        console.log('yearly start')
        logStats(StatTimeInterval.Yearly);
        logStats(StatTimeInterval.AllTime);
    });
    console.info('✅ Stats cron jobs initialized');
};