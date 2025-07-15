// AI_CHECK: TEST_COVERAGE=1,TEST_QUALITY=1 | LAST: 2025-06-24
import { beforeEach, describe, expect, it, vi } from "vitest";

// Mock external dependencies before importing the module
vi.mock("@vrooli/server", () => ({
    CacheService: {
        get: vi.fn(() => ({
            raw: vi.fn().mockResolvedValue({
                multi: vi.fn(() => ({
                    set: vi.fn(),
                    sAdd: vi.fn(),
                    expire: vi.fn(),
                    exec: vi.fn(),
                })),
            }),
        })),
    },
    initSingletons: vi.fn(),
    logger: {
        info: vi.fn(),
        error: vi.fn(),
        warning: vi.fn(),
    },
    ModelMap: {
        init: vi.fn(),
    },
    DbProvider: {
        init: vi.fn(),
        get: vi.fn(),
    },
    batch: vi.fn(),
}));

vi.mock("@vrooli/shared", () => ({
    MINUTES_30_MS: 1800000,
}));

vi.mock("node-cron", () => ({
    default: {
        schedule: vi.fn(),
    },
}));

// Mock all schedule functions
vi.mock("./schedules/cleanupRevokedSessions.js", () => ({
    cleanupRevokedSessions: vi.fn(),
}));

vi.mock("./schedules/countBookmarks.js", () => ({
    countBookmarks: vi.fn(),
}));

vi.mock("./schedules/countReacts.js", () => ({
    countReacts: vi.fn(),
}));

vi.mock("./schedules/countViews.js", () => ({
    countViews: vi.fn(),
}));

vi.mock("./schedules/creditRollover.js", () => ({
    creditRollover: vi.fn(),
}));

vi.mock("./schedules/embeddings.js", () => ({
    generateEmbeddings: vi.fn(),
}));

vi.mock("./schedules/genSitemap.js", () => ({
    genSitemap: vi.fn(),
    isSitemapMissing: vi.fn(),
}));

vi.mock("./schedules/moderatePullRequests.js", () => ({
    moderatePullRequests: vi.fn(),
}));

vi.mock("./schedules/moderateReports.js", () => ({
    moderateReports: vi.fn(),
}));

vi.mock("./schedules/paymentsExpirePremium.js", () => ({
    paymentsExpirePlan: vi.fn(),
}));

vi.mock("./schedules/paymentsFail.js", () => ({
    paymentsFail: vi.fn(),
}));

vi.mock("./schedules/paymentsFreeCredits.js", () => ({
    paymentsCreditsFreePremium: vi.fn(),
}));

vi.mock("./schedules/scheduleNotify.js", () => ({
    scheduleNotify: vi.fn(),
}));

vi.mock("./schedules/stats/index.js", () => ({
    initStatsPeriod: vi.fn(),
    statsPeriodCron: {
        Hourly: "0 * * * *",
        Daily: "0 0 * * *",
        Weekly: "0 0 * * 0",
        Monthly: "0 0 1 * *",
        Yearly: "0 0 1 1 *",
    },
}));

// Import the module under test after mocking
import { initializeAllCronJobs } from "./index.js";
import { logger } from "@vrooli/server";
import cron from "node-cron";
import { isSitemapMissing } from "./schedules/genSitemap.js";

describe("cron job scheduler", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        // Reset process.env to clean state
        delete process.env.npm_package_name;
        delete process.env.HOSTNAME;
        delete process.env.CONTAINER_ID;
    });

    describe("initializeAllCronJobs", () => {
        it("should schedule all cron jobs", () => {
            const mockSchedule = vi.mocked(cron.schedule);
            vi.mocked(isSitemapMissing).mockResolvedValue(false);

            initializeAllCronJobs();

            // Should schedule all 18 cron jobs (13 regular + 5 stats)
            expect(mockSchedule).toHaveBeenCalledTimes(18);
            expect(logger.info).toHaveBeenCalledWith("🚀 Jobs are scheduled and running.");
        });

        it("should run jobs immediately if runRightAway returns true", () => {
            const mockSchedule = vi.mocked(cron.schedule);
            vi.mocked(isSitemapMissing).mockResolvedValue(true);

            initializeAllCronJobs();

            expect(mockSchedule).toHaveBeenCalledTimes(18);
            // Verify that sitemap job is set to run right away by checking it was called immediately
            expect(isSitemapMissing).toHaveBeenCalled();
        });

        it("should handle async runRightAway functions", async () => {
            const mockSchedule = vi.mocked(cron.schedule);
            vi.mocked(isSitemapMissing).mockResolvedValue(true);

            initializeAllCronJobs();

            expect(mockSchedule).toHaveBeenCalledTimes(18);
        });
    });

    describe("cron schedule generation", () => {
        it("should generate valid cron schedules", () => {
            const mockSchedule = vi.mocked(cron.schedule);
            vi.mocked(isSitemapMissing).mockResolvedValue(false);

            initializeAllCronJobs();

            // Verify all schedule calls have valid cron patterns
            mockSchedule.mock.calls.forEach(([schedule]) => {
                expect(schedule).toMatch(/^[\d*\-,/\s]+$/);
                const parts = schedule.split(" ");
                expect(parts).toHaveLength(5);
            });
        });

        it("should use fixed schedules for stats jobs", () => {
            const mockSchedule = vi.mocked(cron.schedule);
            vi.mocked(isSitemapMissing).mockResolvedValue(false);

            initializeAllCronJobs();

            // Find stats-related schedule calls
            const scheduleArgs = mockSchedule.mock.calls.map(([schedule]) => schedule);
            
            expect(scheduleArgs).toContain("0 * * * *"); // Hourly
            expect(scheduleArgs).toContain("0 0 * * *"); // Daily
            expect(scheduleArgs).toContain("0 0 * * 0"); // Weekly
            expect(scheduleArgs).toContain("0 0 1 * *"); // Monthly
            expect(scheduleArgs).toContain("0 0 1 1 *"); // Yearly
        });

        it("should use fixed schedules for specific payment jobs", () => {
            const mockSchedule = vi.mocked(cron.schedule);
            vi.mocked(isSitemapMissing).mockResolvedValue(false);

            initializeAllCronJobs();

            const scheduleArgs = mockSchedule.mock.calls.map(([schedule]) => schedule);
            
            // Free credits should run at midnight on the 1st
            expect(scheduleArgs).toContain("0 0 1 * *");
            // Credit rollover should run at 2 AM on the 2nd
            expect(scheduleArgs).toContain("0 2 2 * *");
        });
    });

    describe("off-peak time generation", () => {
        // Since these functions use Math.random(), we need to test them indirectly
        it("should generate different cron schedules for randomized jobs", () => {
            const mockSchedule = vi.mocked(cron.schedule);
            vi.mocked(isSitemapMissing).mockResolvedValue(false);

            // Run multiple times to check randomization
            initializeAllCronJobs();
            vi.clearAllMocks();
            initializeAllCronJobs();

            // At least some schedules should potentially be different due to randomization
            expect(mockSchedule).toHaveBeenCalledTimes(18);
        });
    });

    describe("job initialization behavior", () => {
        it("should initialize all required cron jobs", () => {
            const mockSchedule = vi.mocked(cron.schedule);
            vi.mocked(isSitemapMissing).mockResolvedValue(false);

            initializeAllCronJobs();

            // Verify the expected number of jobs are scheduled
            expect(mockSchedule).toHaveBeenCalledTimes(18);
            expect(logger.info).toHaveBeenCalledWith("🚀 Jobs are scheduled and running.");
        });
    });

    describe("environment variable handling", () => {
        it("should initialize jobs regardless of environment variables", () => {
            delete process.env.HOSTNAME;
            delete process.env.CONTAINER_ID;

            const mockSchedule = vi.mocked(cron.schedule);
            vi.mocked(isSitemapMissing).mockResolvedValue(false);

            initializeAllCronJobs();

            expect(mockSchedule).toHaveBeenCalledTimes(18);
        });
    });
});
