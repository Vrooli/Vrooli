import { logger } from "@local/server";
import cron from "node-cron";
import { generateEmbeddings } from "./schedules";
import { countBookmarks } from "./schedules/countBookmarks";
import { countReacts } from "./schedules/countReacts";
import { genSitemap, isSitemapMissing } from "./schedules/genSitemap";
import { moderatePullRequests } from "./schedules/moderatePullRequests";
import { moderateReports } from "./schedules/moderateReports";
import { paymentsExpirePremium } from "./schedules/paymentsExpirePremium";
import { paymentsFail } from "./schedules/paymentsFail";
import { paymentsFreeCredits } from "./schedules/paymentsFreeCredits";
import { scheduleNotify } from "./schedules/scheduleNotify";
import { initStatsPeriod } from "./schedules/stats";

type CronJobDefinition = {
    schedule: string;
    jobFunction: (schedule: string) => (unknown | Promise<unknown>);
    description: string;
    runRightAway?: () => (boolean | Promise<boolean>);
};

const cronJobs: Record<string, CronJobDefinition> = {
    countsBookmarks: {
        schedule: "49 5 1 * *", // Every month at 5:49am (UTC)
        jobFunction: countBookmarks,
        description: "verify bookmark counts",
    },
    countsVotes: {
        schedule: "51 5 1 * *", // Every month at 5:51am (UTC)
        jobFunction: countReacts,
        description: "verify react counts",
    },
    embeddings: {
        schedule: "6 * * * *", // Every hour at the 6th minute
        jobFunction: generateEmbeddings,
        description: "generate embeddings",
    },
    events: {
        schedule: "9 5 * * *", // Every day at 5:09am (UTC)
        jobFunction: scheduleNotify,
        description: "schedule calendar events",
    },
    paymentsExpirePremium: {
        schedule: "20 4 * * *", // Every day at 4:20am (UTC)
        jobFunction: paymentsExpirePremium,
        description: "expire premium",
    },
    paymentsFail: {
        schedule: "22 4 * * *", // Every day at 4:22am (UTC)
        jobFunction: paymentsFail,
        description: "fail payments",
    },
    paymentsFreeCredits: {
        schedule: "0 0 1 * *", // Every month on the 1st at midnight (UTC)
        jobFunction: paymentsFreeCredits,
        description: "free credits",
    },
    pullRequests: {
        schedule: "58 4 * * *", // Every day at 4:58am (UTC)
        jobFunction: moderatePullRequests,
        description: "pull request moderation",
    },
    reports: {
        schedule: "56 4 * * *", // Every day at 4:56am (UTC)
        jobFunction: moderateReports,
        description: "report moderation",
    },
    sitemaps: {
        schedule: "43 4 * * *", // Every day at 4:43am (UTC)
        jobFunction: genSitemap,
        description: "create sitemaps",
        runRightAway: isSitemapMissing,
    },
    statsHourly: {
        schedule: "3 * * * *", // Every hour at the 3rd minute
        jobFunction: (schedule) => { initStatsPeriod(schedule); },
        description: "generate hourly stats",
    },
    statsDaily: {
        schedule: "7 5 * * *", // Every day at 5:07am (UTC)
        jobFunction: (schedule) => { initStatsPeriod(schedule); },
        description: "generate hourly stats",
    },
    statsWeekly: {
        schedule: "11 5 * * 0", // Every Sunday at 5:11am (UTC)
        jobFunction: (schedule) => { initStatsPeriod(schedule); },
        description: "generate hourly stats",
    },
    statsMonthly: {
        schedule: "17 5 1 * *", // Every month on the 1st at 5:17am (UTC)
        jobFunction: (schedule) => { initStatsPeriod(schedule); },
        description: "generate hourly stats",
    },
    statsYearly: {
        schedule: "21 5 1 1 *", // Every year on January 1st at 5:21am (UTC)
        jobFunction: (schedule) => { initStatsPeriod(schedule); },
        description: "generate hourly stats",
    },
};

class ConcurrencyLimiter {
    private maxConcurrentJobs: number;
    private currentJobCount = 0;

    constructor(maxConcurrentJobs: number) {
        this.maxConcurrentJobs = maxConcurrentJobs;
    }

    canRunJob(): boolean {
        return this.currentJobCount < this.maxConcurrentJobs;
    }

    jobStarted() {
        this.currentJobCount++;
    }

    jobFinished() {
        this.currentJobCount--;
    }
}

// Initialize the concurrency limiter
const limiter = new ConcurrencyLimiter(5); // Adjust the number as needed
const jobStatus = new Map<string, boolean>();

export const initializeCronJob = (
    schedule: string,
    job: (schedule: string) => (unknown | Promise<unknown>),
    description: string,
): void => {
    try {
        cron.schedule(schedule, async () => {
            if (jobStatus.get(description) === true) {
                logger.warn(`Skipped ${description} cron job as it's still running.`);
                return;
            }

            if (!limiter.canRunJob()) {
                logger.warn(`Skipped ${description} cron job due to concurrency limit.`);
                return;
            }

            jobStatus.set(description, true);
            limiter.jobStarted();
            logger.info(`Starting ${description} cron job.`, { trace: "0396" });

            try {
                await job(schedule);
                logger.info(`✅ ${description} cron job completed.`, { trace: "0397" });
            } catch (error) {
                logger.error(`❌ Error in ${description} cron job.`, { error, trace: "0398" });
            } finally {
                jobStatus.set(description, false);
                limiter.jobFinished();
            }
        });
    } catch (error) {
        logger.error(`❌ Failed to initialize ${description} cron job.`, { error, trace: "0399" });
    }
};

export const initializeAllCronJobs = () => {
    Object.values(cronJobs).forEach(async cronJob => {
        if (cronJob.runRightAway) {
            const runNow = await cronJob.runRightAway();
            if (runNow) {
                await cronJob.jobFunction(cronJob.schedule);
            }
        }
        initializeCronJob(cronJob.schedule, cronJob.jobFunction, cronJob.description);
    });
};

initializeAllCronJobs();
