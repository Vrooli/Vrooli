import { PeriodType } from "@prisma/client";
import { ResourceType, generatePK, generatePublicId } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { logSiteStats } from "./site.js";

// Direct import to avoid problematic services
const { DbProvider } = await import("@vrooli/server");

describe("logSiteStats integration tests", () => {
    // Store test entity IDs for cleanup
    const testUserIds: bigint[] = [];
    const testTeamIds: bigint[] = [];
    const testResourceIds: bigint[] = [];
    const testResourceVersionIds: bigint[] = [];
    const testRunIds: bigint[] = [];
    const testSessionIds: string[] = [];
    const testEmailIds: bigint[] = [];
    const testWalletIds: bigint[] = [];
    const testStatsSiteIds: bigint[] = [];

    beforeEach(async () => {
        // Clear test ID arrays
        testUserIds.length = 0;
        testTeamIds.length = 0;
        testResourceIds.length = 0;
        testResourceVersionIds.length = 0;
        testRunIds.length = 0;
        testSessionIds.length = 0;
        testEmailIds.length = 0;
        testWalletIds.length = 0;
        testStatsSiteIds.length = 0;

        // Reset mocks
        vi.clearAllMocks();
    });

    afterEach(async () => {
        // Clean up test data
        const db = DbProvider.get();
        
        // Clean up in reverse dependency order
        if (testStatsSiteIds.length > 0) {
            await db.stats_site.deleteMany({ where: { id: { in: testStatsSiteIds } } });
        }
        if (testWalletIds.length > 0) {
            await db.wallet.deleteMany({ where: { id: { in: testWalletIds } } });
        }
        if (testEmailIds.length > 0) {
            await db.email.deleteMany({ where: { id: { in: testEmailIds } } });
        }
        if (testSessionIds.length > 0) {
            await db.session.deleteMany({ where: { id: { in: testSessionIds } } });
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
        if (testTeamIds.length > 0) {
            await db.team.deleteMany({ where: { id: { in: testTeamIds } } });
        }
        if (testUserIds.length > 0) {
            await db.user.deleteMany({ where: { id: { in: testUserIds } } });
        }
    });

    it("should create site stats with zero values when no data exists", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString(); // 24 hours ago
        const periodEnd = now.toISOString();

        await logSiteStats(PeriodType.Daily, periodStart, periodEnd);

        // Find the created stats entry
        const stats = await DbProvider.get().stats_site.findFirst({
            where: {
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(stats).not.toBeNull();
        testStatsSiteIds.push(stats!.id);

        expect(stats!.activeUsers).toBe(0);
        expect(stats!.teamsCreated).toBe(0);
        expect(stats!.verifiedEmailsCreated).toBe(0);
        expect(stats!.verifiedWalletsCreated).toBe(0);
        expect(stats!.runsStarted).toBe(0);
        expect(stats!.runsCompleted).toBe(0);
        expect(stats!.runCompletionTimeAverage).toBe(0);
        expect(stats!.runContextSwitchesAverage).toBe(0);
        expect(stats!.routineComplexityAverage).toBe(0);
    });

    it("should count active users correctly", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();
        const sessionTime = new Date(now.getTime() - 12 * 60 * 60 * 1000); // 12 hours ago

        // Create users (one bot, one regular)
        const regularUser = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Regular User",
                handle: "regularuser",
                isBot: false,
            },
        });
        testUserIds.push(regularUser.id);

        const botUser = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Bot User",
                handle: "botuser",
                isBot: true,
            },
        });
        testUserIds.push(botUser.id);

        // Create sessions within the period
        const session1 = await DbProvider.get().session.create({
            data: {
                id: "session1",
                user_id: regularUser.id,
                auth_id: generatePK(),
                last_refresh_at: sessionTime,
                expires_at: new Date(now.getTime() + 24 * 60 * 60 * 1000),
            },
        });
        testSessionIds.push(session1.id);

        const session2 = await DbProvider.get().session.create({
            data: {
                id: "session2",
                user_id: botUser.id,
                auth_id: generatePK(),
                last_refresh_at: sessionTime,
                expires_at: new Date(now.getTime() + 24 * 60 * 60 * 1000),
            },
        });
        testSessionIds.push(session2.id);

        await logSiteStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_site.findFirst({
            where: { periodStart, periodEnd, periodType: PeriodType.Daily },
        });

        expect(stats).not.toBeNull();
        testStatsSiteIds.push(stats!.id);

        // Should only count regular user, not bot
        expect(stats!.activeUsers).toBe(1);
    });

    it("should count teams created correctly", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();
        const teamCreatedTime = new Date(now.getTime() - 12 * 60 * 60 * 1000);

        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Team Owner",
                handle: "teamowner",
            },
        });
        testUserIds.push(owner.id);

        // Create teams within period
        const team1 = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                handle: "testteam1",
                createdAt: teamCreatedTime,
            },
        });
        testTeamIds.push(team1.id);

        const team2 = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                handle: "testteam2",
                createdAt: teamCreatedTime,
            },
        });
        testTeamIds.push(team2.id);

        await logSiteStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_site.findFirst({
            where: { periodStart, periodEnd, periodType: PeriodType.Daily },
        });

        expect(stats).not.toBeNull();
        testStatsSiteIds.push(stats!.id);
        expect(stats!.teamsCreated).toBe(2);
    });

    it("should count resources created by type correctly", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();
        const resourceCreatedTime = new Date(now.getTime() - 12 * 60 * 60 * 1000);

        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Resource Owner",
                handle: "resourceowner",
            },
        });
        testUserIds.push(owner.id);

        // Create different types of resources within period
        const routine = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                createdById: owner.id,
                resourceType: ResourceType.Routine,
                isDeleted: false,
                isInternal: false,
                createdAt: resourceCreatedTime,
            },
        });
        testResourceIds.push(routine.id);

        const api = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                createdById: owner.id,
                resourceType: ResourceType.Api,
                isDeleted: false,
                createdAt: resourceCreatedTime,
            },
        });
        testResourceIds.push(api.id);

        const project = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                createdById: owner.id,
                resourceType: ResourceType.Project,
                isDeleted: false,
                createdAt: resourceCreatedTime,
            },
        });
        testResourceIds.push(project.id);

        await logSiteStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_site.findFirst({
            where: { periodStart, periodEnd, periodType: PeriodType.Daily },
        });

        expect(stats).not.toBeNull();
        testStatsSiteIds.push(stats!.id);

        const resourcesCreated = stats!.resourcesCreatedByType as Record<string, number>;
        expect(resourcesCreated[ResourceType.Routine]).toBe(1);
        expect(resourcesCreated[ResourceType.Api]).toBe(1);
        expect(resourcesCreated[ResourceType.Project]).toBe(1);
    });

    it("should count completed resources and calculate completion times", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();
        const resourceCreatedTime = new Date(now.getTime() - 25 * 60 * 60 * 1000); // Created before period
        const resourceCompletedTime = new Date(now.getTime() - 12 * 60 * 60 * 1000); // Completed in period

        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Resource Owner",
                handle: "resourceowner2",
            },
        });
        testUserIds.push(owner.id);

        // Create a routine that's completed in the period
        const routine = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                createdById: owner.id,
                resourceType: ResourceType.Routine,
                isDeleted: false,
                isInternal: false,
                createdAt: resourceCreatedTime,
                completedAt: resourceCompletedTime,
            },
        });
        testResourceIds.push(routine.id);

        await logSiteStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_site.findFirst({
            where: { periodStart, periodEnd, periodType: PeriodType.Daily },
        });

        expect(stats).not.toBeNull();
        testStatsSiteIds.push(stats!.id);

        const resourcesCompleted = stats!.resourcesCompletedByType as Record<string, number>;
        expect(resourcesCompleted[ResourceType.Routine]).toBe(1);

        const completionTimes = stats!.resourceCompletionTimeAverageByType as Record<string, number>;
        expect(completionTimes[ResourceType.Routine]).toBeGreaterThan(0);
    });

    it("should count runs and calculate run metrics", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();
        const runStartedTime = new Date(now.getTime() - 20 * 60 * 60 * 1000);
        const runCompletedTime = new Date(now.getTime() - 12 * 60 * 60 * 1000);

        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Run Owner",
                handle: "runowner",
            },
        });
        testUserIds.push(owner.id);

        // Create a run that starts in the period
        const run1 = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                name: "Test Run 1",
                status: "Running",
                user: {
                    connect: { id: owner.id },
                },
                isPrivate: false,
                startedAt: runStartedTime,
            },
        });
        testRunIds.push(run1.id);

        // Create a run that completes in the period
        const run2 = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                name: "Test Run 2",
                status: "Completed",
                user: {
                    connect: { id: owner.id },
                },
                isPrivate: false,
                startedAt: runStartedTime,
                completedAt: runCompletedTime,
                contextSwitches: 5,
            },
        });
        testRunIds.push(run2.id);

        await logSiteStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_site.findFirst({
            where: { periodStart, periodEnd, periodType: PeriodType.Daily },
        });

        expect(stats).not.toBeNull();
        testStatsSiteIds.push(stats!.id);

        expect(stats!.runsStarted).toBe(2);
        expect(stats!.runsCompleted).toBe(1);
        expect(stats!.runCompletionTimeAverage).toBeGreaterThan(0);
        expect(stats!.runContextSwitchesAverage).toBe(5);
    });

    it("should count verified emails and wallets", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();
        const verifiedTime = new Date(now.getTime() - 12 * 60 * 60 * 1000);

        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Verified User",
                handle: "verifieduser",
            },
        });
        testUserIds.push(user.id);

        // Create verified email within period
        const email = await DbProvider.get().email.create({
            data: {
                id: generatePK(),
                userId: user.id,
                emailAddress: "verified@example.com",
                verified: true,
                createdAt: verifiedTime,
                verifiedAt: verifiedTime,
            },
        });
        testEmailIds.push(email.id);

        // Create verified wallet within period
        const wallet = await DbProvider.get().wallet.create({
            data: {
                id: generatePK(),
                userId: user.id,
                walletAddress: "0x1234567890abcdef",
                verified: true,
                createdAt: verifiedTime,
                verifiedAt: verifiedTime,
            },
        });
        testWalletIds.push(wallet.id);

        await logSiteStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_site.findFirst({
            where: { periodStart, periodEnd, periodType: PeriodType.Daily },
        });

        expect(stats).not.toBeNull();
        testStatsSiteIds.push(stats!.id);

        expect(stats!.verifiedEmailsCreated).toBe(1);
        expect(stats!.verifiedWalletsCreated).toBe(1);
    });

    it("should calculate routine complexity average correctly", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();
        const resourceCreatedTime = new Date(now.getTime() - 25 * 60 * 60 * 1000);
        const resourceCompletedTime = new Date(now.getTime() - 12 * 60 * 60 * 1000);

        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Routine Owner",
                handle: "routineowner",
            },
        });
        testUserIds.push(owner.id);

        // Create routine completed in period
        const routine = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                createdById: owner.id,
                resourceType: ResourceType.Routine,
                isDeleted: false,
                isInternal: false,
                createdAt: resourceCreatedTime,
                completedAt: resourceCompletedTime,
            },
        });
        testResourceIds.push(routine.id);

        // Create routine version with complexity
        const routineVersion = await DbProvider.get().resource_version.create({
            data: {
                id: generatePK(),
                rootId: routine.id,
                versionLabel: "1.0.0",
                complexity: 10,
                simplicity: 5,
                isLatest: true,
                isDeleted: false,
                isPrivate: false,
            },
        });
        testResourceVersionIds.push(routineVersion.id);

        await logSiteStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_site.findFirst({
            where: { periodStart, periodEnd, periodType: PeriodType.Daily },
        });

        expect(stats).not.toBeNull();
        testStatsSiteIds.push(stats!.id);
        expect(stats!.routineComplexityAverage).toBe(10);
    });

    it("should handle different period types", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000).toISOString(); // 7 days ago
        const periodEnd = now.toISOString();

        await logSiteStats(PeriodType.Weekly, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_site.findFirst({
            where: { periodStart, periodEnd, periodType: PeriodType.Weekly },
        });

        expect(stats).not.toBeNull();
        testStatsSiteIds.push(stats!.id);
        expect(stats!.periodType).toBe(PeriodType.Weekly);
    });

    it("should exclude deleted and internal resources correctly", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();
        const resourceCreatedTime = new Date(now.getTime() - 12 * 60 * 60 * 1000);

        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Resource Owner",
                handle: "resourceowner3",
            },
        });
        testUserIds.push(owner.id);

        // Create deleted routine (should be excluded)
        const deletedRoutine = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                createdById: owner.id,
                resourceType: ResourceType.Routine,
                isDeleted: true, // Deleted
                isInternal: false,
                createdAt: resourceCreatedTime,
            },
        });
        testResourceIds.push(deletedRoutine.id);

        // Create internal routine (should be excluded)
        const internalRoutine = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                createdById: owner.id,
                resourceType: ResourceType.Routine,
                isDeleted: false,
                isInternal: true, // Internal
                createdAt: resourceCreatedTime,
            },
        });
        testResourceIds.push(internalRoutine.id);

        // Create valid routine (should be included)
        const validRoutine = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                createdById: owner.id,
                resourceType: ResourceType.Routine,
                isDeleted: false,
                isInternal: false,
                createdAt: resourceCreatedTime,
            },
        });
        testResourceIds.push(validRoutine.id);

        await logSiteStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_site.findFirst({
            where: { periodStart, periodEnd, periodType: PeriodType.Daily },
        });

        expect(stats).not.toBeNull();
        testStatsSiteIds.push(stats!.id);

        const resourcesCreated = stats!.resourcesCreatedByType as Record<string, number>;
        expect(resourcesCreated[ResourceType.Routine]).toBe(1); // Only the valid one
    });

    it("should handle edge case with no active users", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();

        // Create user but no sessions in the period
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Inactive User",
                handle: "inactiveuser",
                isBot: false,
            },
        });
        testUserIds.push(user.id);

        await logSiteStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_site.findFirst({
            where: { periodStart, periodEnd, periodType: PeriodType.Daily },
        });

        expect(stats).not.toBeNull();
        testStatsSiteIds.push(stats!.id);
        expect(stats!.activeUsers).toBe(0);
    });
});
