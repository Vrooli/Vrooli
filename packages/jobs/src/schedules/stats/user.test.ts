// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-24
import { PeriodType } from "@prisma/client";
import { ResourceType, generatePK, generatePublicId } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// Direct imports to avoid problematic services
const { DbProvider } = await import("@vrooli/server");
const { logUserStats } = await import("./user.js");

describe("logUserStats integration tests", () => {
    // Store test entity IDs for cleanup
    const testUserIds: bigint[] = [];
    const testTeamIds: bigint[] = [];
    const testResourceIds: bigint[] = [];
    const testRunIds: bigint[] = [];
    const testStatsUserIds: bigint[] = [];

    beforeEach(async () => {
        // Clear test ID arrays
        testUserIds.length = 0;
        testTeamIds.length = 0;
        testResourceIds.length = 0;
        testRunIds.length = 0;
        testStatsUserIds.length = 0;

        // Reset mocks
        vi.clearAllMocks();
    });

    afterEach(async () => {
        // Clean up test data
        const db = DbProvider.get();
        
        // Clean up in reverse dependency order
        if (testStatsUserIds.length > 0) {
            await db.stats_user.deleteMany({ where: { id: { in: testStatsUserIds } } });
        }
        if (testRunIds.length > 0) {
            await db.run.deleteMany({ where: { id: { in: testRunIds } } });
        }
        if (testResourceIds.length > 0) {
            await db.resource.deleteMany({ where: { id: { in: testResourceIds } } });
        }
        if (testTeamIds.length > 0) {
            await db.team.deleteMany({ where: { id: { in: testTeamIds } } });
        }
        if (testUserIds.length > 0) {
            await db.user.deleteMany({ where: { id: { in: testUserIds } } });
        }
    });

    it("should create user stats with zero values when no activity exists", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();

        // Create user with recent update to be included
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Inactive User",
                handle: `inactiveuser_${Date.now()}`,
                // Use current time to ensure it's definitely within window
            },
        });
        testUserIds.push(user.id);

        await logUserStats(PeriodType.Daily, periodStart, periodEnd);

        // Find the created stats entry
        const stats = await DbProvider.get().stats_user.findFirst({
            where: {
                userId: user.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(stats).not.toBeNull();
        testStatsUserIds.push(stats!.id);

        expect(stats!.teamsCreated).toBe(0);
        
        // For JSON fields, check they're either null/empty or have zero totals
        const resourcesCreatedTotal = stats!.resourcesCreatedByType 
            ? Object.values(stats!.resourcesCreatedByType as Record<string, number>).reduce((a, b) => a + b, 0) 
            : 0;
        expect(resourcesCreatedTotal).toBe(0);
        
        const resourcesCompletedTotal = stats!.resourcesCompletedByType 
            ? Object.values(stats!.resourcesCompletedByType as Record<string, number>).reduce((a, b) => a + b, 0) 
            : 0;
        expect(resourcesCompletedTotal).toBe(0);
        
        expect(stats!.runsStarted).toBe(0);
        expect(stats!.runsCompleted).toBe(0);
        expect(stats!.runCompletionTimeAverage).toBe(0);
        expect(stats!.runContextSwitchesAverage).toBe(0);
    });

    it("should count teams created by user correctly", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();
        const teamCreatedTime = new Date(now.getTime() - 12 * 60 * 60 * 1000);

        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Team Creator",
                handle: `teamcreator_${Date.now()}`,
                // Use current time to ensure it's definitely within window
            },
        });
        testUserIds.push(user.id);

        // Create teams within period
        const team1 = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: { connect: { id: user.id } },
                handle: `userteam1_${Date.now()}`,
                createdAt: teamCreatedTime,
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: "Test Team 1",
                    }],
                },
            },
        });
        testTeamIds.push(team1.id);

        const team2 = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: { connect: { id: user.id } },
                handle: `userteam2_${Date.now()}`,
                createdAt: teamCreatedTime,
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        name: "Test Team 2",
                    }],
                },
            },
        });
        testTeamIds.push(team2.id);

        await logUserStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_user.findFirst({
            where: {
                userId: user.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(stats).not.toBeNull();
        testStatsUserIds.push(stats!.id);
        expect(stats!.teamsCreated).toBe(2);
    });

    it("should count resources created and completed correctly", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();
        const resourceCreatedTime = new Date(now.getTime() - 20 * 60 * 60 * 1000);
        const resourceCompletedTime = new Date(now.getTime() - 12 * 60 * 60 * 1000);

        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Resource Creator",
                handle: `resourcecreator_${Date.now()}`,
            },
        });
        testUserIds.push(user.id);

        // Create resource created in period
        const createdResource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: { connect: { id: user.id } },
                resourceType: ResourceType.Routine,
                isDeleted: false,
                hasCompleteVersion: false,
                createdAt: resourceCreatedTime,
            },
        });
        testResourceIds.push(createdResource.id);

        // Create resource completed in period
        const completedResource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: { connect: { id: user.id } },
                resourceType: ResourceType.Project,
                isDeleted: false,
                hasCompleteVersion: true,
                createdAt: resourceCreatedTime,
                completedAt: resourceCompletedTime,
            },
        });
        testResourceIds.push(completedResource.id);

        await logUserStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_user.findFirst({
            where: {
                userId: user.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(stats).not.toBeNull();
        testStatsUserIds.push(stats!.id);

        // Check resourcesCreatedByType - should have 2 total (1 Routine + 1 Project)
        const resourcesCreatedByType = stats!.resourcesCreatedByType as Record<string, number> || {};
        const totalCreated = Object.values(resourcesCreatedByType).reduce((sum, count) => sum + count, 0);
        expect(totalCreated).toBe(2);
        
        // Check resourcesCompletedByType - should have 1 total
        const resourcesCompletedByType = stats!.resourcesCompletedByType as Record<string, number> || {};
        const totalCompleted = Object.values(resourcesCompletedByType).reduce((sum, count) => sum + count, 0);
        expect(totalCompleted).toBe(1);
        
        // Check completion time average exists
        const completionTimeByType = stats!.resourceCompletionTimeAverageByType as Record<string, number> || {};
        const hasCompletionTime = Object.values(completionTimeByType).some(time => time > 0);
        expect(hasCompletionTime).toBe(true);
    });

    it("should count user runs correctly", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();
        const runStartedTime = new Date(now.getTime() - 20 * 60 * 60 * 1000);
        const runCompletedTime = new Date(now.getTime() - 12 * 60 * 60 * 1000);

        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Runner User",
                handle: `runneruser_${Date.now()}`,
            },
        });
        testUserIds.push(user.id);

        // Create run that starts in period
        const runStarted = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                name: "Started Run",
                status: "InProgress",
                user: { connect: { id: user.id } },
                isPrivate: false,
                startedAt: runStartedTime,
            },
        });
        testRunIds.push(runStarted.id);

        // Create run that completes in period
        const runCompleted = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                name: "Completed Run",
                status: "Completed",
                user: { connect: { id: user.id } },
                isPrivate: false,
                startedAt: runStartedTime,
                completedAt: runCompletedTime,
                timeElapsed: 8 * 60 * 60, // 8 hours in seconds
                contextSwitches: 4,
            },
        });
        testRunIds.push(runCompleted.id);

        await logUserStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_user.findFirst({
            where: {
                userId: user.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(stats).not.toBeNull();
        testStatsUserIds.push(stats!.id);

        expect(stats!.runsStarted).toBe(2); // Both runs started in period
        expect(stats!.runsCompleted).toBe(1); // Only one completed
        expect(stats!.runCompletionTimeAverage).toBe(8 * 60 * 60); // 8 hours
        expect(stats!.runContextSwitchesAverage).toBe(4);
    });

    it("should calculate run averages correctly with multiple completed runs", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();
        const runStartedTime = new Date(now.getTime() - 20 * 60 * 60 * 1000);
        const runCompletedTime = new Date(now.getTime() - 12 * 60 * 60 * 1000);

        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Multi Runner",
                handle: `multirunner_${Date.now()}`,
            },
        });
        testUserIds.push(user.id);

        // Create multiple completed runs
        const run1 = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                name: "Run 1",
                status: "Completed",
                user: { connect: { id: user.id } },
                isPrivate: false,
                startedAt: runStartedTime,
                completedAt: runCompletedTime,
                timeElapsed: 3600, // 1 hour
                contextSwitches: 2,
            },
        });
        testRunIds.push(run1.id);

        const run2 = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                name: "Run 2",
                status: "Completed",
                user: { connect: { id: user.id } },
                isPrivate: false,
                startedAt: runStartedTime,
                completedAt: runCompletedTime,
                timeElapsed: 7200, // 2 hours
                contextSwitches: 6,
            },
        });
        testRunIds.push(run2.id);

        await logUserStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_user.findFirst({
            where: {
                userId: user.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(stats).not.toBeNull();
        testStatsUserIds.push(stats!.id);

        expect(stats!.runsCompleted).toBe(2);
        expect(stats!.runCompletionTimeAverage).toBe(5400); // (3600 + 7200) / 2 = 5400 seconds
        expect(stats!.runContextSwitchesAverage).toBe(4); // (2 + 6) / 2 = 4
    });

    it("should calculate resource completion time average correctly", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();
        const createdTime = new Date(now.getTime() - 25 * 60 * 60 * 1000); // Created before period
        const completedTime1 = new Date(now.getTime() - 20 * 60 * 60 * 1000); // Completed in period
        const completedTime2 = new Date(now.getTime() - 15 * 60 * 60 * 1000); // Completed in period

        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Completion Timer",
                handle: `completiontimer_${Date.now()}`,
            },
        });
        testUserIds.push(user.id);

        // Create two completed resources with different completion times
        const resource1 = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: { connect: { id: user.id } },
                resourceType: ResourceType.Routine,
                isDeleted: false,
                hasCompleteVersion: true,
                createdAt: createdTime,
                completedAt: completedTime1, // 5 hours to complete
            },
        });
        testResourceIds.push(resource1.id);

        const resource2 = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: { connect: { id: user.id } },
                resourceType: ResourceType.Project,
                isDeleted: false,
                hasCompleteVersion: true,
                createdAt: createdTime,
                completedAt: completedTime2, // 10 hours to complete
            },
        });
        testResourceIds.push(resource2.id);

        await logUserStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_user.findFirst({
            where: {
                userId: user.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(stats).not.toBeNull();
        testStatsUserIds.push(stats!.id);

        // Check total completed resources
        const resourcesCompletedByType = stats!.resourcesCompletedByType as Record<string, number> || {};
        const totalCompleted = Object.values(resourcesCompletedByType).reduce((sum, count) => sum + count, 0);
        expect(totalCompleted).toBe(2);
        
        // Should be average of the two completion times
        const expectedTime1 = completedTime1.getTime() - createdTime.getTime();
        const expectedTime2 = completedTime2.getTime() - createdTime.getTime();
        const expectedAverage = (expectedTime1 + expectedTime2) / 2;
        
        // Check completion time average exists in the JSON
        const completionTimeByType = stats!.resourceCompletionTimeAverageByType as Record<string, number> || {};
        const avgTimes = Object.values(completionTimeByType);
        expect(avgTimes.length).toBeGreaterThan(0);
        
        // The average should be close to our expected average
        const actualAverage = avgTimes.reduce((sum, time) => sum + time, 0) / avgTimes.length;
        expect(Math.abs(actualAverage - expectedAverage)).toBeLessThan(1000); // Within 1 second
    });

    it("should exclude deleted resources", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();
        const resourceCreatedTime = new Date(now.getTime() - 12 * 60 * 60 * 1000);

        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Deleter User",
                handle: `deleteruser_${Date.now()}`,
            },
        });
        testUserIds.push(user.id);

        // Create deleted resource (should be excluded)
        const deletedResource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: { connect: { id: user.id } },
                resourceType: ResourceType.Routine,
                isDeleted: true, // Deleted
                hasCompleteVersion: false,
                createdAt: resourceCreatedTime,
            },
        });
        testResourceIds.push(deletedResource.id);

        // Create valid resource (should be included)
        const validResource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdBy: { connect: { id: user.id } },
                resourceType: ResourceType.Project,
                isDeleted: false,
                hasCompleteVersion: false,
                createdAt: resourceCreatedTime,
            },
        });
        testResourceIds.push(validResource.id);

        await logUserStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_user.findFirst({
            where: {
                userId: user.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(stats).not.toBeNull();
        testStatsUserIds.push(stats!.id);
        // Check only the valid resource was counted
        const resourcesCreatedByType = stats!.resourcesCreatedByType as Record<string, number> || {};
        const totalCreated = Object.values(resourcesCreatedByType).reduce((sum, count) => sum + count, 0);
        expect(totalCreated).toBe(1); // Only the valid one
    });

    it("should only include users updated within 90 days", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();

        // Create user updated more than 90 days ago (should be excluded)
        const oldUser = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Old User",
                handle: `olduser_${Date.now()}`,
                updatedAt: new Date(now.getTime() - 95 * 24 * 60 * 60 * 1000), // 95 days ago
            },
        });
        testUserIds.push(oldUser.id);

        // Create user updated within 90 days (should be included)
        const recentUser = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Recent User",
                handle: `recentuser_${Date.now()}`,
                // Use current time to ensure it's definitely within window
            },
        });
        testUserIds.push(recentUser.id);

        await logUserStats(PeriodType.Daily, periodStart, periodEnd);

        // Should only create stats for recent user
        const oldUserStats = await DbProvider.get().stats_user.findFirst({
            where: {
                userId: oldUser.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });
        expect(oldUserStats).toBeNull();

        const recentUserStats = await DbProvider.get().stats_user.findFirst({
            where: {
                userId: recentUser.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });
        expect(recentUserStats).not.toBeNull();
        testStatsUserIds.push(recentUserStats!.id);
    });

    it("should handle different period types", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000).toISOString(); // 7 days ago
        const periodEnd = now.toISOString();

        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Weekly User",
                handle: `weeklyuser_${Date.now()}`,
            },
        });
        testUserIds.push(user.id);

        await logUserStats(PeriodType.Weekly, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_user.findFirst({
            where: {
                userId: user.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Weekly,
            },
        });

        expect(stats).not.toBeNull();
        testStatsUserIds.push(stats!.id);
        expect(stats!.periodType).toBe(PeriodType.Weekly);
    });

    it("should handle runs without time elapsed gracefully", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();
        const runStartedTime = new Date(now.getTime() - 20 * 60 * 60 * 1000);
        const runCompletedTime = new Date(now.getTime() - 12 * 60 * 60 * 1000);

        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "No Time User",
                handle: `notimeuser_${Date.now()}`,
            },
        });
        testUserIds.push(user.id);

        // Create run without timeElapsed
        const run = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                name: "No Time Run",
                status: "Completed",
                user: { connect: { id: user.id } },
                isPrivate: false,
                startedAt: runStartedTime,
                completedAt: runCompletedTime,
                timeElapsed: null, // No time elapsed
                contextSwitches: 3,
            },
        });
        testRunIds.push(run.id);

        await logUserStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_user.findFirst({
            where: {
                userId: user.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(stats).not.toBeNull();
        testStatsUserIds.push(stats!.id);

        expect(stats!.runsCompleted).toBe(1);
        expect(stats!.runCompletionTimeAverage).toBe(0); // No time elapsed contributes 0
        expect(stats!.runContextSwitchesAverage).toBe(3);
    });

    it("should handle multiple users in batch processing", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();

        // Create multiple users
        const users = [];
        for (let i = 0; i < 3; i++) {
            const user = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: `Batch User ${i}`,
                    handle: `batchuser${i}_${Date.now()}`,
                },
            });
            testUserIds.push(user.id);
            users.push(user);
        }

        await logUserStats(PeriodType.Daily, periodStart, periodEnd);

        // Should create stats for all users
        const allStats = await DbProvider.get().stats_user.findMany({
            where: {
                userId: { in: users.map(u => u.id) },
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(allStats).toHaveLength(3);
        allStats.forEach(stat => testStatsUserIds.push(stat.id));
    });

    it("should handle missing data gracefully", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();

        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Graceful User",
                handle: `gracefuluser_${Date.now()}`,
            },
        });
        testUserIds.push(user.id);

        // Don't create any resources, runs, or teams for this user

        await logUserStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_user.findFirst({
            where: {
                userId: user.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(stats).not.toBeNull();
        testStatsUserIds.push(stats!.id);

        // Should have default values when no data exists
        const completionTimeByType = stats!.resourceCompletionTimeAverageByType as Record<string, number> || {};
        const avgTimes = Object.values(completionTimeByType);
        expect(avgTimes.length).toBe(0); // No completion times
        
        const resourcesCompletedByType = stats!.resourcesCompletedByType as Record<string, number> || {};
        const totalCompleted = Object.values(resourcesCompletedByType).reduce((sum, count) => sum + count, 0);
        expect(totalCompleted).toBe(0);
        
        const resourcesCreatedByType = stats!.resourcesCreatedByType as Record<string, number> || {};
        const totalCreated = Object.values(resourcesCreatedByType).reduce((sum, count) => sum + count, 0);
        expect(totalCreated).toBe(0);
        expect(stats!.runCompletionTimeAverage).toBe(0);
        expect(stats!.runContextSwitchesAverage).toBe(0);
        expect(stats!.runsCompleted).toBe(0);
        expect(stats!.runsStarted).toBe(0);
        expect(stats!.teamsCreated).toBe(0);
    });
});
