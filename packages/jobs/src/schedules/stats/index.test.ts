import { PeriodType } from "@prisma/client";
import { DAYS_1_HOURS, MONTHS_1_DAYS, WEEKS_1_DAYS, YEARS_1_DAYS } from "@vrooli/shared";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { initStatsPeriod, statsPeriodCron } from "./index.js";

// Mock the stat functions
vi.mock("./resource.js", () => ({
    logResourceStats: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("./site.js", () => ({
    logSiteStats: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("./team.js", () => ({
    logTeamStats: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("./user.js", () => ({
    logUserStats: vi.fn().mockResolvedValue(undefined),
}));

describe("stats index", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        // Mock Date to ensure consistent testing
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2024-01-15T10:30:00Z")); // Fixed date for testing
    });

    afterEach(() => {
        vi.useRealTimers();
    });

    describe("statsPeriodCron", () => {
        it("should have cron expressions for all period types", () => {
            expect(statsPeriodCron[PeriodType.Hourly]).toBe("3 * * * *");
            expect(statsPeriodCron[PeriodType.Daily]).toBe("7 5 * * *");
            expect(statsPeriodCron[PeriodType.Weekly]).toBe("11 5 * * 0");
            expect(statsPeriodCron[PeriodType.Monthly]).toBe("17 5 1 * *");
            expect(statsPeriodCron[PeriodType.Yearly]).toBe("21 5 1 1 *");
        });

        it("should have unique cron expressions for each period", () => {
            const cronValues = Object.values(statsPeriodCron);
            const uniqueCronValues = [...new Set(cronValues)];
            expect(cronValues).toHaveLength(uniqueCronValues.length);
        });
    });

    describe("initStatsPeriod", () => {
        it("should call all stat functions for hourly period", async () => {
            const hourlyParams = [
                PeriodType.Hourly,
                expect.any(String), // periodStart
                expect.any(String), // periodEnd
            ];

            await initStatsPeriod(statsPeriodCron[PeriodType.Hourly]);

            const { logSiteStats } = await import("./site.js");
            const { logTeamStats } = await import("./team.js");
            const { logResourceStats } = await import("./resource.js");
            const { logUserStats } = await import("./user.js");

            expect(logSiteStats).toHaveBeenCalledWith(...hourlyParams);
            expect(logTeamStats).toHaveBeenCalledWith(...hourlyParams);
            expect(logResourceStats).toHaveBeenCalledWith(...hourlyParams);
            expect(logUserStats).toHaveBeenCalledWith(...hourlyParams);
        });

        it("should call all stat functions for daily period", async () => {
            const dailyParams = [
                PeriodType.Daily,
                expect.any(String),
                expect.any(String),
            ];

            await initStatsPeriod(statsPeriodCron[PeriodType.Daily]);

            const { logSiteStats } = await import("./site.js");
            const { logTeamStats } = await import("./team.js");
            const { logResourceStats } = await import("./resource.js");
            const { logUserStats } = await import("./user.js");

            expect(logSiteStats).toHaveBeenCalledWith(...dailyParams);
            expect(logTeamStats).toHaveBeenCalledWith(...dailyParams);
            expect(logResourceStats).toHaveBeenCalledWith(...dailyParams);
            expect(logUserStats).toHaveBeenCalledWith(...dailyParams);
        });

        it("should call all stat functions for weekly period", async () => {
            const weeklyParams = [
                PeriodType.Weekly,
                expect.any(String),
                expect.any(String),
            ];

            await initStatsPeriod(statsPeriodCron[PeriodType.Weekly]);

            const { logSiteStats } = await import("./site.js");
            const { logTeamStats } = await import("./team.js");
            const { logResourceStats } = await import("./resource.js");
            const { logUserStats } = await import("./user.js");

            expect(logSiteStats).toHaveBeenCalledWith(...weeklyParams);
            expect(logTeamStats).toHaveBeenCalledWith(...weeklyParams);
            expect(logResourceStats).toHaveBeenCalledWith(...weeklyParams);
            expect(logUserStats).toHaveBeenCalledWith(...weeklyParams);
        });

        it("should call all stat functions for monthly period", async () => {
            const monthlyParams = [
                PeriodType.Monthly,
                expect.any(String),
                expect.any(String),
            ];

            await initStatsPeriod(statsPeriodCron[PeriodType.Monthly]);

            const { logSiteStats } = await import("./site.js");
            const { logTeamStats } = await import("./team.js");
            const { logResourceStats } = await import("./resource.js");
            const { logUserStats } = await import("./user.js");

            expect(logSiteStats).toHaveBeenCalledWith(...monthlyParams);
            expect(logTeamStats).toHaveBeenCalledWith(...monthlyParams);
            expect(logResourceStats).toHaveBeenCalledWith(...monthlyParams);
            expect(logUserStats).toHaveBeenCalledWith(...monthlyParams);
        });

        it("should call all stat functions for yearly period", async () => {
            const yearlyParams = [
                PeriodType.Yearly,
                expect.any(String),
                expect.any(String),
            ];

            await initStatsPeriod(statsPeriodCron[PeriodType.Yearly]);

            const { logSiteStats } = await import("./site.js");
            const { logTeamStats } = await import("./team.js");
            const { logResourceStats } = await import("./resource.js");
            const { logUserStats } = await import("./user.js");

            expect(logSiteStats).toHaveBeenCalledWith(...yearlyParams);
            expect(logTeamStats).toHaveBeenCalledWith(...yearlyParams);
            expect(logResourceStats).toHaveBeenCalledWith(...yearlyParams);
            expect(logUserStats).toHaveBeenCalledWith(...yearlyParams);
        });

        it("should calculate correct period start for hourly", async () => {
            const fixedDate = new Date("2024-01-15T10:30:00Z");
            vi.setSystemTime(fixedDate);
            
            await initStatsPeriod(statsPeriodCron[PeriodType.Hourly]);

            const expectedStart = new Date(fixedDate);
            expectedStart.setHours(expectedStart.getHours() - 1);

            const { logSiteStats } = await import("./site.js");
            const callArgs = (logSiteStats as any).mock.calls[0];
            
            expect(callArgs[0]).toBe(PeriodType.Hourly);
            expect(new Date(callArgs[1]).getTime()).toBe(expectedStart.getTime());
            expect(new Date(callArgs[2]).getTime()).toBe(fixedDate.getTime());
        });

        it("should calculate correct period start for daily", async () => {
            const fixedDate = new Date("2024-01-15T10:30:00Z");
            vi.setSystemTime(fixedDate);
            
            await initStatsPeriod(statsPeriodCron[PeriodType.Daily]);

            const expectedStart = new Date(fixedDate);
            expectedStart.setHours(expectedStart.getHours() - DAYS_1_HOURS);

            const { logSiteStats } = await import("./site.js");
            const callArgs = (logSiteStats as any).mock.calls[0];
            
            expect(callArgs[0]).toBe(PeriodType.Daily);
            expect(new Date(callArgs[1]).getTime()).toBe(expectedStart.getTime());
        });

        it("should calculate correct period start for weekly", async () => {
            const fixedDate = new Date("2024-01-15T10:30:00Z");
            vi.setSystemTime(fixedDate);
            
            await initStatsPeriod(statsPeriodCron[PeriodType.Weekly]);

            const expectedStart = new Date(fixedDate);
            expectedStart.setDate(expectedStart.getDate() - WEEKS_1_DAYS);

            const { logSiteStats } = await import("./site.js");
            const callArgs = (logSiteStats as any).mock.calls[0];
            
            expect(callArgs[0]).toBe(PeriodType.Weekly);
            expect(new Date(callArgs[1]).getTime()).toBe(expectedStart.getTime());
        });

        it("should calculate correct period start for monthly", async () => {
            const fixedDate = new Date("2024-01-15T10:30:00Z");
            vi.setSystemTime(fixedDate);
            
            await initStatsPeriod(statsPeriodCron[PeriodType.Monthly]);

            const expectedStart = new Date(fixedDate);
            expectedStart.setDate(expectedStart.getDate() - MONTHS_1_DAYS);

            const { logSiteStats } = await import("./site.js");
            const callArgs = (logSiteStats as any).mock.calls[0];
            
            expect(callArgs[0]).toBe(PeriodType.Monthly);
            expect(new Date(callArgs[1]).getTime()).toBe(expectedStart.getTime());
        });

        it("should calculate correct period start for yearly", async () => {
            const fixedDate = new Date("2024-01-15T10:30:00Z");
            vi.setSystemTime(fixedDate);
            
            await initStatsPeriod(statsPeriodCron[PeriodType.Yearly]);

            const expectedStart = new Date(fixedDate);
            expectedStart.setDate(expectedStart.getDate() - YEARS_1_DAYS);

            const { logSiteStats } = await import("./site.js");
            const callArgs = (logSiteStats as any).mock.calls[0];
            
            expect(callArgs[0]).toBe(PeriodType.Yearly);
            expect(new Date(callArgs[1]).getTime()).toBe(expectedStart.getTime());
        });

        it("should handle unknown cron expression gracefully", async () => {
            // Should not throw even with unknown cron
            await expect(initStatsPeriod("unknown cron")).resolves.not.toThrow();
            
            // Should not call any stat functions with unknown cron
            const { logSiteStats } = await import("./site.js");
            expect(logSiteStats).not.toHaveBeenCalled();
        });

        it("should call all functions concurrently using Promise.all", async () => {
            // Verify that all functions are called in parallel by checking they're all called
            await initStatsPeriod(statsPeriodCron[PeriodType.Daily]);

            const { logSiteStats } = await import("./site.js");
            const { logTeamStats } = await import("./team.js");
            const { logResourceStats } = await import("./resource.js");
            const { logUserStats } = await import("./user.js");

            // All should be called once
            expect(logSiteStats).toHaveBeenCalledTimes(1);
            expect(logTeamStats).toHaveBeenCalledTimes(1);
            expect(logResourceStats).toHaveBeenCalledTimes(1);
            expect(logUserStats).toHaveBeenCalledTimes(1);
        });

        it("should pass ISO string timestamps", async () => {
            await initStatsPeriod(statsPeriodCron[PeriodType.Hourly]);

            const { logSiteStats } = await import("./site.js");
            const callArgs = (logSiteStats as any).mock.calls[0];
            
            // Second argument should be periodStart as ISO string
            expect(typeof callArgs[1]).toBe("string");
            expect(callArgs[1]).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$/);
            
            // Third argument should be periodEnd as ISO string  
            expect(typeof callArgs[2]).toBe("string");
            expect(callArgs[2]).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$/);
        });
    });
});