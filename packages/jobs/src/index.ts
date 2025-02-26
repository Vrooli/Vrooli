/* eslint-disable no-magic-numbers */
import { initSingletons, logger } from "@local/server";
import { MINUTES_30_MS } from "@local/shared";
import cron from "node-cron";
import { cleanupRevokedSessions } from "./schedules/cleanupRevokedSessions";
import { countBookmarks } from "./schedules/countBookmarks";
import { countReacts } from "./schedules/countReacts";
import { countViews } from "./schedules/countViews";
import { generateEmbeddings } from "./schedules/embeddings";
import { genSitemap, isSitemapMissing } from "./schedules/genSitemap";
import { moderatePullRequests } from "./schedules/moderatePullRequests";
import { moderateReports } from "./schedules/moderateReports";
import { paymentsExpirePremium } from "./schedules/paymentsExpirePremium";
import { paymentsFail } from "./schedules/paymentsFail";
import { paymentsCreditsFreePremium } from "./schedules/paymentsFreeCredits";
import { scheduleNotify } from "./schedules/scheduleNotify";
import { initStatsPeriod, statsPeriodCron } from "./schedules/stats";

/** How many jobs can run at once */
const MAX_JOB_CONCURRENCY = 5;
/** How many jobs can be in the queue before we start skipping them */
const MAX_QUEUE_LENGTH = 20;
/** How long a job can run before we consider it failed */
const JOB_TIMEOUT_MS = MINUTES_30_MS;

type JobFunction = (schedule: string) => unknown | Promise<unknown>;
type CronSchedule = string;

type CronJobDefinition = {
    description: string;
    jobFunction: JobFunction;
    runRightAway?: () => (boolean | Promise<boolean>);
    schedule: CronSchedule;
};

/**
 * @returns A random off-peak minute (i.e. not on the hour or multiples of 15)
 */
function offPeakMinute() {
    const peakMinutes = new Set([0, 15, 30, 45]);
    let minute: number;
    do {
        minute = Math.floor(Math.random() * 60);
    } while (peakMinutes.has(minute));
    return minute;
}

/**
 * @returns A random off-peak hour (i.e. between 0-5 or 22-23)
 */
function offPeakHour() {
    // 0.75 probability of returning 0-5
    // 0.25 probability of returning 22-23
    if (Math.random() < 0.75) {
        return Math.floor(Math.random() * 6); // 0-5
    } else {
        return Math.floor(Math.random() * 2) + 22; // 22-23
    }
}

type CronPart = number | string | null;

/**
 * Generates a cron job string with support for off-peak scheduling
 * @param minute Minute (0-59), "*", or null for random off-peak time
 * @param hour Hour (0-23), "*", or null for random off-peak time
 * @param dayOfMonth Day of month (1-31), "*", or null for random time
 * @param month Month (1-12), "*", or null for random time
 * @param dayOfWeek Day of week (0-6, 0=Sunday), "*", or null for random time
 * @returns {string} Valid cron job string
 */
function generateCronJob(
    minute: CronPart,
    hour: CronPart,
    dayOfMonth: CronPart,
    month: CronPart,
    dayOfWeek: CronPart,
): string {
    // Helper function for random values when null is provided
    function getRandomValue(min: number, max: number) {
        return Math.floor(Math.random() * (max - min + 1)) + min;
    }

    const cron = [
        minute ?? offPeakMinute(),
        hour ?? offPeakHour(),
        dayOfMonth ?? getRandomValue(1, 31),
        month ?? getRandomValue(1, 12),
        dayOfWeek ?? getRandomValue(0, 6),
    ];

    return cron.join(" ");
}

const cronJobs: CronJobDefinition[] = [
    {
        description: "verify bookmark counts",
        jobFunction: countBookmarks,
        schedule: generateCronJob(null, null, 1, "*", "*"), // Off-peak time on the 1st of every month
    },
    {
        description: "verify react counts",
        jobFunction: countReacts,
        schedule: generateCronJob(null, null, 1, "*", "*"), // Off-peak time on the 1st of every month
    },
    {
        description: "verify view counts",
        jobFunction: countViews,
        schedule: generateCronJob(null, null, 1, "*", "*"), // Off-peak time on the 1st of every month
    },
    {
        description: "generate embeddings",
        jobFunction: generateEmbeddings,
        schedule: generateCronJob(null, "*", "*", "*", "*"), // Off-peak time every hour
    },
    {
        description: "schedule calendar events",
        jobFunction: scheduleNotify,
        schedule: generateCronJob(null, null, "*", "*", "*"), // Off-peak time every day
    },
    {
        description: "expire premium",
        jobFunction: paymentsExpirePremium,
        schedule: generateCronJob(null, null, "*", "*", "*"), // Off-peak time every day
    },
    {
        description: "fail payments",
        jobFunction: paymentsFail,
        schedule: generateCronJob(null, null, "*", "*", "*"), // Off-peak time every day
    },
    {
        description: "free credits for premium users",
        jobFunction: paymentsCreditsFreePremium,
        schedule: generateCronJob(0, 0, 1, "*", "*"), // Midnight on the 1st of every month
    },
    {
        description: "pull request moderation",
        jobFunction: moderatePullRequests,
        schedule: generateCronJob(null, null, "*", "*", "*"), // Off-peak time every day
    },
    {
        description: "report moderation",
        jobFunction: moderateReports,
        schedule: generateCronJob(null, null, "*", "*", "*"), // Off-peak time every day
    },
    {
        description: "revoke sessions",
        jobFunction: cleanupRevokedSessions,
        schedule: generateCronJob(null, null, "*", "*", "*"), // Off-peak time every day
    },
    {
        description: "create sitemaps",
        jobFunction: genSitemap,
        runRightAway: isSitemapMissing,
        schedule: generateCronJob(null, null, "*", "*", "*"), // Off-peak time every day
    },
    {
        description: "generate hourly stats",
        jobFunction: initStatsPeriod,
        schedule: statsPeriodCron.Hourly,
    },
    {
        description: "generate hourly stats",
        jobFunction: initStatsPeriod,
        schedule: statsPeriodCron.Daily,
    },
    {
        description: "generate hourly stats",
        jobFunction: initStatsPeriod,
        schedule: statsPeriodCron.Weekly,
    },
    {
        description: "generate hourly stats",
        jobFunction: initStatsPeriod,
        schedule: statsPeriodCron.Monthly,
    },
    {
        description: "generate hourly stats",
        jobFunction: initStatsPeriod,
        schedule: statsPeriodCron.Yearly,
    },
];

class CronJobQueue {
    private queue: Array<JobFunction> = [];
    private runningJobs = 0;

    constructor(private maxConcurrentJobs: number, private maxQueueLength: number) { }

    addJob(job: JobFunction, schedule: CronSchedule, description: string) {
        if (this.queue.length >= this.maxQueueLength) {
            logger.error(`Queue is full. Cannot add job: ${description}`, { trace: "0396" });
            return;
        }

        this.queue.push(async () => {
            this.runningJobs++;
            try {
                logger.info(`Starting job: ${description}`, { trace: "0397" });
                await this.runWithTimeout(job, schedule, description);
                logger.info(`Job completed: ${description}`);
            } catch (error) {
                logger.error(`Error in job: ${description}`, { error, trace: "0398" });
            } finally {
                this.runningJobs--;
                this.processQueue(schedule);
            }
        });

        if (this.runningJobs < this.maxConcurrentJobs) {
            this.processQueue(schedule);
        } else if (this.queue.length > this.maxConcurrentJobs) {
            logger.warning(`Queue length exceeded expectations: ${this.queue.length}`, { trace: "0399" });
        }
    }

    private processQueue(schedule: CronSchedule) {
        if (this.runningJobs >= this.maxConcurrentJobs || this.queue.length === 0) {
            return;
        }
        const nextJob = this.queue.shift();
        if (nextJob) {
            nextJob(schedule);
        }
    }

    private async runWithTimeout(job: JobFunction, schedule: CronSchedule, description: string) {
        const timeoutPromise = new Promise((_, reject) =>
            setTimeout(() => reject(new Error(`Job timeout: ${description}`)), JOB_TIMEOUT_MS),
        );
        await Promise.race([job(schedule), timeoutPromise]);
    }
}

export function initializeAllCronJobs() {
    const jobQueue = new CronJobQueue(MAX_JOB_CONCURRENCY, MAX_QUEUE_LENGTH);

    cronJobs.forEach(cronJob => {
        const shouldRunNow = typeof cronJob.runRightAway === "function" ? cronJob.runRightAway() : false;
        if (shouldRunNow) {
            jobQueue.addJob(cronJob.jobFunction, cronJob.schedule, cronJob.description);
        }

        cron.schedule(cronJob.schedule, () => {
            jobQueue.addJob(cronJob.jobFunction, cronJob.schedule, cronJob.description);
        });
    });

    logger.info("🚀 Jobs are scheduled and running.");
}

if (process.env.npm_package_name === "@local/jobs") {
    await initSingletons();
    initializeAllCronJobs();
}
