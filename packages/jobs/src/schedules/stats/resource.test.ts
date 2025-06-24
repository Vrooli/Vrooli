// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-24
import { PeriodType, RunStatus } from "@prisma/client";
import { ResourceType, generatePK, generatePublicId } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { logResourceStats } from "./resource.js";

// Direct import to avoid problematic services
const { DbProvider } = await import("@vrooli/server");

describe("logResourceStats integration tests", () => {
    // Store test entity IDs for cleanup
    const testUserIds: bigint[] = [];
    const testResourceIds: bigint[] = [];
    const testResourceVersionIds: bigint[] = [];
    const testRunIds: bigint[] = [];
    const testStatsResourceIds: bigint[] = [];

    beforeEach(async () => {
        // Clear test ID arrays
        testUserIds.length = 0;
        testResourceIds.length = 0;
        testResourceVersionIds.length = 0;
        testRunIds.length = 0;
        testStatsResourceIds.length = 0;

        // Reset mocks
        vi.clearAllMocks();
    });

    afterEach(async () => {
        // Clean up test data
        const db = DbProvider.get();
        
        // Clean up in reverse dependency order
        if (testStatsResourceIds.length > 0) {
            await db.stats_resource.deleteMany({ where: { id: { in: testStatsResourceIds } } });
        }
        if (testRunIds.length > 0) {
            await db.run.deleteMany({ where: { id: { in: testRunIds } } });
        }
        if (testResourceVersionIds.length > 0) {
            await db.resource_version.deleteMany({ where: { id: { in: testResourceVersionIds } } });
        }
        if (testResourceIds.length > 0) {
            await db.resource.deleteMany({ where: { id: { in: testResourceIds } } });
        }
        if (testUserIds.length > 0) {
            await db.user.deleteMany({ where: { id: { in: testUserIds } } });
        }
    });

    it("should create resource stats with zero values when no runs exist", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();

        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Resource Owner",
                handle: "resourceowner",
            },
        });
        testUserIds.push(user.id);

        // Create resource
        const resource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: user.id,
                resourceType: ResourceType.Routine,
                isDeleted: false,
            },
        });
        testResourceIds.push(resource.id);

        // Create latest resource version
        const resourceVersion = await DbProvider.get().resource_version.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                rootId: resource.id,
                versionIndex: 1,
                versionLabel: "1.0.0",
                complexity: 5,
                isLatest: true,
                isDeleted: false,
                isPrivate: false,
            },
        });
        testResourceVersionIds.push(resourceVersion.id);

        await logResourceStats(PeriodType.Daily, periodStart, periodEnd);

        // Find the created stats entry
        const stats = await DbProvider.get().stats_resource.findFirst({
            where: {
                resourceId: resource.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(stats).not.toBeNull();
        testStatsResourceIds.push(stats!.id);

        expect(stats!.runsStarted).toBe(0);
        expect(stats!.runsCompleted).toBe(0);
        expect(stats!.runCompletionTimeAverage).toBe(0);
        expect(stats!.runContextSwitchesAverage).toBe(0);
    });

    it("should count runs started and completed correctly", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();
        const runStartedTime = new Date(now.getTime() - 20 * 60 * 60 * 1000); // 20 hours ago
        const runCompletedTime = new Date(now.getTime() - 12 * 60 * 60 * 1000); // 12 hours ago

        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Resource Owner",
                handle: "resourceowner",
            },
        });
        testUserIds.push(user.id);

        const resource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: user.id,
                resourceType: ResourceType.Routine,
                isDeleted: false,
            },
        });
        testResourceIds.push(resource.id);

        const resourceVersion = await DbProvider.get().resource_version.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                rootId: resource.id,
                versionIndex: 1,
                versionLabel: "1.0.0",
                complexity: 5,
                isLatest: true,
                isDeleted: false,
                isPrivate: false,
            },
        });
        testResourceVersionIds.push(resourceVersion.id);

        // Create run that starts in period
        const runStarted = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                name: "Started Run",
                status: RunStatus.InProgress,
                user: {
                    connect: { id: user.id },
                },
                resourceVersion: {

                    connect: { id: resourceVersion.id },

                },
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
                status: RunStatus.Completed,
                user: {
                    connect: { id: user.id },
                },
                resourceVersion: {

                    connect: { id: resourceVersion.id },

                },
                isPrivate: false,
                startedAt: runStartedTime,
                completedAt: runCompletedTime,
                timeElapsed: 8 * 60 * 60, // 8 hours in seconds
                contextSwitches: 3,
            },
        });
        testRunIds.push(runCompleted.id);

        await logResourceStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_resource.findFirst({
            where: {
                resourceId: resource.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(stats).not.toBeNull();
        testStatsResourceIds.push(stats!.id);

        expect(stats!.runsStarted).toBe(2); // Both runs started in period
        expect(stats!.runsCompleted).toBe(1); // Only one completed
        expect(stats!.runCompletionTimeAverage).toBe(8 * 60 * 60); // 8 hours
        expect(stats!.runContextSwitchesAverage).toBe(3);
    });

    it("should calculate averages correctly with multiple completed runs", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();
        const runStartedTime = new Date(now.getTime() - 20 * 60 * 60 * 1000);
        const runCompletedTime = new Date(now.getTime() - 12 * 60 * 60 * 1000);

        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Resource Owner",
                handle: "resourceowner2",
            },
        });
        testUserIds.push(user.id);

        const resource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: user.id,
                resourceType: ResourceType.Routine,
                isDeleted: false,
            },
        });
        testResourceIds.push(resource.id);

        const resourceVersion = await DbProvider.get().resource_version.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                rootId: resource.id,
                versionIndex: 1,
                versionLabel: "1.0.0",
                complexity: 5,
                isLatest: true,
                isDeleted: false,
                isPrivate: false,
            },
        });
        testResourceVersionIds.push(resourceVersion.id);

        // Create multiple completed runs
        const run1 = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                name: "Run 1",
                status: RunStatus.Completed,
                user: {
                    connect: { id: user.id },
                },
                resourceVersion: {

                    connect: { id: resourceVersion.id },

                },
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
                status: RunStatus.Completed,
                user: {
                    connect: { id: user.id },
                },
                resourceVersion: {

                    connect: { id: resourceVersion.id },

                },
                isPrivate: false,
                startedAt: runStartedTime,
                completedAt: runCompletedTime,
                timeElapsed: 7200, // 2 hours
                contextSwitches: 4,
            },
        });
        testRunIds.push(run2.id);

        await logResourceStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_resource.findFirst({
            where: {
                resourceId: resource.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(stats).not.toBeNull();
        testStatsResourceIds.push(stats!.id);

        expect(stats!.runsCompleted).toBe(2);
        expect(stats!.runCompletionTimeAverage).toBe(5400); // (3600 + 7200) / 2 = 5400 seconds
        expect(stats!.runContextSwitchesAverage).toBe(3); // (2 + 4) / 2 = 3
    });

    it("should only include latest resource versions", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();
        const runStartedTime = new Date(now.getTime() - 12 * 60 * 60 * 1000);

        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Resource Owner",
                handle: "resourceowner3",
            },
        });
        testUserIds.push(user.id);

        const resource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: user.id,
                resourceType: ResourceType.Routine,
                isDeleted: false,
            },
        });
        testResourceIds.push(resource.id);

        // Create old version (not latest)
        const oldVersion = await DbProvider.get().resource_version.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                rootId: resource.id,
                versionIndex: 1,
                versionLabel: "1.0.0",
                complexity: 5,
                isLatest: false, // Not latest
                isDeleted: false,
                isPrivate: false,
            },
        });
        testResourceVersionIds.push(oldVersion.id);

        // Create latest version
        const latestVersion = await DbProvider.get().resource_version.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                rootId: resource.id,
                versionIndex: 2,
                versionLabel: "2.0.0",
                complexity: 7,
                isLatest: true, // Latest
                isDeleted: false,
                isPrivate: false,
            },
        });
        testResourceVersionIds.push(latestVersion.id);

        // Create run for old version (should not be included)
        const oldRun = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                name: "Old Run",
                status: RunStatus.InProgress,
                user: {
                    connect: { id: user.id },
                },
                resourceVersion: {

                    connect: { id: oldVersion.id },

                },
                isPrivate: false,
                startedAt: runStartedTime,
            },
        });
        testRunIds.push(oldRun.id);

        // Create run for latest version (should be included)
        const latestRun = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                name: "Latest Run",
                status: RunStatus.InProgress,
                user: {
                    connect: { id: user.id },
                },
                resourceVersion: {

                    connect: { id: latestVersion.id },

                },
                isPrivate: false,
                startedAt: runStartedTime,
            },
        });
        testRunIds.push(latestRun.id);

        await logResourceStats(PeriodType.Daily, periodStart, periodEnd);

        // Should only have stats for the latest version
        const allStats = await DbProvider.get().stats_resource.findMany({
            where: {
                resourceId: resource.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(allStats).toHaveLength(1);
        testStatsResourceIds.push(allStats[0].id);

        expect(allStats[0].runsStarted).toBe(1); // Only the latest version run
    });

    it("should exclude deleted resource versions", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();

        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Resource Owner",
                handle: "resourceowner4",
            },
        });
        testUserIds.push(user.id);

        const resource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: user.id,
                resourceType: ResourceType.Routine,
                isDeleted: false,
            },
        });
        testResourceIds.push(resource.id);

        // Create deleted resource version
        const deletedVersion = await DbProvider.get().resource_version.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                rootId: resource.id,
                versionIndex: 1,
                versionLabel: "1.0.0",
                complexity: 5,
                isLatest: true,
                isDeleted: true, // Deleted
                isPrivate: false,
            },
        });
        testResourceVersionIds.push(deletedVersion.id);

        await logResourceStats(PeriodType.Daily, periodStart, periodEnd);

        // Should not create stats for deleted version
        const stats = await DbProvider.get().stats_resource.findFirst({
            where: {
                resourceId: resource.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(stats).toBeNull();
    });

    it("should exclude resource versions with deleted roots", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();

        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Resource Owner",
                handle: "resourceowner5",
            },
        });
        testUserIds.push(user.id);

        // Create deleted resource
        const deletedResource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: user.id,
                resourceType: ResourceType.Routine,
                isDeleted: true, // Deleted root
            },
        });
        testResourceIds.push(deletedResource.id);

        const resourceVersion = await DbProvider.get().resource_version.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                rootId: deletedResource.id,
                versionLabel: "1.0.0",
                complexity: 5,
                isLatest: true,
                isDeleted: false,
                isPrivate: false,
            },
        });
        testResourceVersionIds.push(resourceVersion.id);

        await logResourceStats(PeriodType.Daily, periodStart, periodEnd);

        // Should not create stats for version with deleted root
        const stats = await DbProvider.get().stats_resource.findFirst({
            where: {
                resourceId: deletedResource.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(stats).toBeNull();
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
                name: "Resource Owner",
                handle: "resourceowner6",
            },
        });
        testUserIds.push(user.id);

        const resource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: user.id,
                resourceType: ResourceType.Routine,
                isDeleted: false,
            },
        });
        testResourceIds.push(resource.id);

        const resourceVersion = await DbProvider.get().resource_version.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                rootId: resource.id,
                versionIndex: 1,
                versionLabel: "1.0.0",
                complexity: 5,
                isLatest: true,
                isDeleted: false,
                isPrivate: false,
            },
        });
        testResourceVersionIds.push(resourceVersion.id);

        // Create run without timeElapsed
        const run = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                name: "No Time Run",
                status: RunStatus.Completed,
                user: {
                    connect: { id: user.id },
                },
                resourceVersion: {

                    connect: { id: resourceVersion.id },

                },
                isPrivate: false,
                startedAt: runStartedTime,
                completedAt: runCompletedTime,
                timeElapsed: null, // No time elapsed
                contextSwitches: 2,
            },
        });
        testRunIds.push(run.id);

        await logResourceStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_resource.findFirst({
            where: {
                resourceId: resource.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(stats).not.toBeNull();
        testStatsResourceIds.push(stats!.id);

        expect(stats!.runsCompleted).toBe(1);
        expect(stats!.runCompletionTimeAverage).toBe(0); // No time elapsed contributes 0
        expect(stats!.runContextSwitchesAverage).toBe(2);
    });

    it("should handle different period types", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000).toISOString(); // 7 days ago
        const periodEnd = now.toISOString();

        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Resource Owner",
                handle: "resourceowner7",
            },
        });
        testUserIds.push(user.id);

        const resource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: user.id,
                resourceType: ResourceType.Routine,
                isDeleted: false,
            },
        });
        testResourceIds.push(resource.id);

        const resourceVersion = await DbProvider.get().resource_version.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                rootId: resource.id,
                versionIndex: 1,
                versionLabel: "1.0.0",
                complexity: 5,
                isLatest: true,
                isDeleted: false,
                isPrivate: false,
            },
        });
        testResourceVersionIds.push(resourceVersion.id);

        await logResourceStats(PeriodType.Weekly, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_resource.findFirst({
            where: {
                resourceId: resource.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Weekly,
            },
        });

        expect(stats).not.toBeNull();
        testStatsResourceIds.push(stats!.id);
        expect(stats!.periodType).toBe(PeriodType.Weekly);
    });

    it("should handle multiple resources in batch", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();

        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Resource Owner",
                handle: "resourceowner8",
            },
        });
        testUserIds.push(user.id);

        // Create multiple resources and versions
        const resources = [];
        const resourceVersions = [];
        
        for (let i = 0; i < 3; i++) {
            const resource = await DbProvider.get().resource.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    createdById: user.id,
                    resourceType: ResourceType.Routine,
                    isDeleted: false,
                },
            });
            testResourceIds.push(resource.id);
            resources.push(resource);

            const resourceVersion = await DbProvider.get().resource_version.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    rootId: resource.id,
                    versionIndex: 1,
                    versionLabel: "1.0.0",
                    complexity: 5,
                    isLatest: true,
                    isDeleted: false,
                    isPrivate: false,
                },
            });
            testResourceVersionIds.push(resourceVersion.id);
            resourceVersions.push(resourceVersion);
        }

        await logResourceStats(PeriodType.Daily, periodStart, periodEnd);

        // Should create stats for all resources
        const allStats = await DbProvider.get().stats_resource.findMany({
            where: {
                resourceId: { in: resources.map(r => r.id) },
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(allStats).toHaveLength(3);
        allStats.forEach(stat => testStatsResourceIds.push(stat.id));
    });
});
