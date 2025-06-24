// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-24
import { PeriodType, RunStatus } from "@prisma/client";
import { generatePK, generatePublicId } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { logTeamStats } from "./team.js";

// Direct import to avoid problematic services
const { DbProvider } = await import("@vrooli/server");

describe("logTeamStats integration tests", () => {
    // Store test entity IDs for cleanup
    const testUserIds: bigint[] = [];
    const testTeamIds: bigint[] = [];
    const testResourceIds: bigint[] = [];
    const testRunIds: bigint[] = [];
    const testMembershipIds: bigint[] = [];
    const testStatsTeamIds: bigint[] = [];

    beforeEach(async () => {
        // Clear test ID arrays
        testUserIds.length = 0;
        testTeamIds.length = 0;
        testResourceIds.length = 0;
        testRunIds.length = 0;
        testMembershipIds.length = 0;
        testStatsTeamIds.length = 0;

        // Reset mocks
        vi.clearAllMocks();
    });

    afterEach(async () => {
        // Clean up test data
        const db = DbProvider.get();
        
        // Clean up in reverse dependency order
        if (testStatsTeamIds.length > 0) {
            await db.stats_team.deleteMany({ where: { id: { in: testStatsTeamIds } } });
        }
        if (testMembershipIds.length > 0) {
            await db.member.deleteMany({ where: { id: { in: testMembershipIds } } });
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

    it("should create team stats with zero run values when no runs exist", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();

        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Team Owner",
                handle: "teamowner",
            },
        });
        testUserIds.push(owner.id);

        const member = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Team Member",
                handle: "teammember",
            },
        });
        testUserIds.push(member.id);

        // Create team
        const team = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                handle: "testteam",
            },
        });
        testTeamIds.push(team.id);

        // Add team member
        const membership = await DbProvider.get().member.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                team: { connect: { id: team.id } },
                user: { connect: { id: member.id } },
            },
        });
        testMembershipIds.push(membership.id);

        // Create team resource
        const resource = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                ownedByTeamId: team.id,
                resourceType: "Routine",
                isDeleted: false,
            },
        });
        testResourceIds.push(resource.id);

        await logTeamStats(PeriodType.Daily, periodStart, periodEnd);

        // Find the created stats entry
        const stats = await DbProvider.get().stats_team.findFirst({
            where: {
                teamId: team.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(stats).not.toBeNull();
        testStatsTeamIds.push(stats!.id);

        expect(stats!.members).toBe(1); // One member
        expect(stats!.resources).toBe(1); // One resource
        expect(stats!.runsStarted).toBe(0);
        expect(stats!.runsCompleted).toBe(0);
        expect(stats!.runCompletionTimeAverage).toBe(0);
        expect(stats!.runContextSwitchesAverage).toBe(0);
    });

    it("should count team runs correctly", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();
        const runStartedTime = new Date(now.getTime() - 20 * 60 * 60 * 1000); // 20 hours ago
        const runCompletedTime = new Date(now.getTime() - 12 * 60 * 60 * 1000); // 12 hours ago

        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Team Owner",
                handle: "teamowner2",
            },
        });
        testUserIds.push(owner.id);

        const team = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                handle: "testteam2",
            },
        });
        testTeamIds.push(team.id);

        // Create run that starts in period
        const runStarted = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                name: "Started Run",
                status: RunStatus.InProgress,
                user: { connect: { id: owner.id } },
                team: { connect: { id: team.id } },
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
                user: { connect: { id: owner.id } },
                team: { connect: { id: team.id } },
                isPrivate: false,
                startedAt: runStartedTime,
                completedAt: runCompletedTime,
                timeElapsed: 8 * 60 * 60, // 8 hours in seconds
                contextSwitches: 5,
            },
        });
        testRunIds.push(runCompleted.id);

        await logTeamStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_team.findFirst({
            where: {
                teamId: team.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(stats).not.toBeNull();
        testStatsTeamIds.push(stats!.id);

        expect(stats!.runsStarted).toBe(2); // Both runs started in period
        expect(stats!.runsCompleted).toBe(1); // Only one completed
        expect(stats!.runCompletionTimeAverage).toBe(8 * 60 * 60); // 8 hours
        expect(stats!.runContextSwitchesAverage).toBe(5);
    });

    it("should calculate run averages correctly with multiple completed runs", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();
        const runStartedTime = new Date(now.getTime() - 20 * 60 * 60 * 1000);
        const runCompletedTime = new Date(now.getTime() - 12 * 60 * 60 * 1000);

        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Team Owner",
                handle: "teamowner3",
            },
        });
        testUserIds.push(owner.id);

        const team = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                handle: "testteam3",
            },
        });
        testTeamIds.push(team.id);

        // Create multiple completed runs
        const run1 = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                name: "Run 1",
                status: RunStatus.Completed,
                user: { connect: { id: owner.id } },
                team: { connect: { id: team.id } },
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
                user: { connect: { id: owner.id } },
                team: { connect: { id: team.id } },
                isPrivate: false,
                startedAt: runStartedTime,
                completedAt: runCompletedTime,
                timeElapsed: 7200, // 2 hours
                contextSwitches: 4,
            },
        });
        testRunIds.push(run2.id);

        await logTeamStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_team.findFirst({
            where: {
                teamId: team.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(stats).not.toBeNull();
        testStatsTeamIds.push(stats!.id);

        expect(stats!.runsCompleted).toBe(2);
        expect(stats!.runCompletionTimeAverage).toBe(5400); // (3600 + 7200) / 2 = 5400 seconds
        expect(stats!.runContextSwitchesAverage).toBe(3); // (2 + 4) / 2 = 3
    });

    it("should count team members and resources correctly", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();

        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Team Owner",
                handle: "teamowner4",
            },
        });
        testUserIds.push(owner.id);

        const team = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                handle: "testteam4",
            },
        });
        testTeamIds.push(team.id);

        // Add multiple members
        const members = [];
        for (let i = 0; i < 3; i++) {
            const member = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: `Member ${i}`,
                    handle: `member${i}`,
                },
            });
            testUserIds.push(member.id);
            members.push(member);

            const membership = await DbProvider.get().member.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    team: { connect: { id: team.id } },
                    user: { connect: { id: member.id } },
                },
            });
            testMembershipIds.push(membership.id);
        }

        // Add multiple resources
        for (let i = 0; i < 2; i++) {
            const resource = await DbProvider.get().resource.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    createdById: owner.id,
                    ownedByTeamId: team.id,
                    resourceType: "Routine",
                    isDeleted: false,
                },
            });
            testResourceIds.push(resource.id);
        }

        await logTeamStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_team.findFirst({
            where: {
                teamId: team.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(stats).not.toBeNull();
        testStatsTeamIds.push(stats!.id);

        expect(stats!.members).toBe(3); // Three members
        expect(stats!.resources).toBe(2); // Two resources
    });

    it("should handle runs without time elapsed gracefully", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();
        const runStartedTime = new Date(now.getTime() - 20 * 60 * 60 * 1000);
        const runCompletedTime = new Date(now.getTime() - 12 * 60 * 60 * 1000);

        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Team Owner",
                handle: "teamowner5",
            },
        });
        testUserIds.push(owner.id);

        const team = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                handle: "testteam5",
            },
        });
        testTeamIds.push(team.id);

        // Create run without timeElapsed
        const run = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                name: "No Time Run",
                status: RunStatus.Completed,
                user: { connect: { id: owner.id } },
                team: { connect: { id: team.id } },
                isPrivate: false,
                startedAt: runStartedTime,
                completedAt: runCompletedTime,
                timeElapsed: null, // No time elapsed
                contextSwitches: 3,
            },
        });
        testRunIds.push(run.id);

        await logTeamStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_team.findFirst({
            where: {
                teamId: team.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(stats).not.toBeNull();
        testStatsTeamIds.push(stats!.id);

        expect(stats!.runsCompleted).toBe(1);
        expect(stats!.runCompletionTimeAverage).toBe(0); // No time elapsed contributes 0
        expect(stats!.runContextSwitchesAverage).toBe(3);
    });

    it("should handle different period types", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000).toISOString(); // 7 days ago
        const periodEnd = now.toISOString();

        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Team Owner",
                handle: "teamowner6",
            },
        });
        testUserIds.push(owner.id);

        const team = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                handle: "testteam6",
            },
        });
        testTeamIds.push(team.id);

        await logTeamStats(PeriodType.Weekly, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_team.findFirst({
            where: {
                teamId: team.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Weekly,
            },
        });

        expect(stats).not.toBeNull();
        testStatsTeamIds.push(stats!.id);
        expect(stats!.periodType).toBe(PeriodType.Weekly);
    });

    it("should only count runs associated with the team", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();
        const runStartedTime = new Date(now.getTime() - 12 * 60 * 60 * 1000);

        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Team Owner",
                handle: "teamowner7",
            },
        });
        testUserIds.push(owner.id);

        const team1 = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                handle: "testteam7a",
            },
        });
        testTeamIds.push(team1.id);

        const team2 = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                handle: "testteam7b",
            },
        });
        testTeamIds.push(team2.id);

        // Create run for team1
        const team1Run = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                name: "Team 1 Run",
                status: RunStatus.InProgress,
                user: { connect: { id: owner.id } },
                team: { connect: { id: team1.id } },
                isPrivate: false,
                startedAt: runStartedTime,
            },
        });
        testRunIds.push(team1Run.id);

        // Create run for team2
        const team2Run = await DbProvider.get().run.create({
            data: {
                id: generatePK(),
                name: "Team 2 Run",
                status: RunStatus.InProgress,
                user: { connect: { id: owner.id } },
                team: { connect: { id: team2.id } },
                isPrivate: false,
                startedAt: runStartedTime,
            },
        });
        testRunIds.push(team2Run.id);

        await logTeamStats(PeriodType.Daily, periodStart, periodEnd);

        // Check stats for team1
        const team1Stats = await DbProvider.get().stats_team.findFirst({
            where: {
                teamId: team1.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(team1Stats).not.toBeNull();
        testStatsTeamIds.push(team1Stats!.id);
        expect(team1Stats!.runsStarted).toBe(1); // Only team1's run

        // Check stats for team2
        const team2Stats = await DbProvider.get().stats_team.findFirst({
            where: {
                teamId: team2.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(team2Stats).not.toBeNull();
        testStatsTeamIds.push(team2Stats!.id);
        expect(team2Stats!.runsStarted).toBe(1); // Only team2's run
    });

    it("should handle teams with no members or resources", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();

        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Team Owner",
                handle: "teamowner8",
            },
        });
        testUserIds.push(owner.id);

        // Create empty team (no members or resources)
        const emptyTeam = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: owner.id,
                handle: "emptyteam",
            },
        });
        testTeamIds.push(emptyTeam.id);

        await logTeamStats(PeriodType.Daily, periodStart, periodEnd);

        const stats = await DbProvider.get().stats_team.findFirst({
            where: {
                teamId: emptyTeam.id,
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(stats).not.toBeNull();
        testStatsTeamIds.push(stats!.id);

        expect(stats!.members).toBe(0);
        expect(stats!.resources).toBe(0);
        expect(stats!.runsStarted).toBe(0);
        expect(stats!.runsCompleted).toBe(0);
    });

    it("should handle multiple teams in batch processing", async () => {
        const now = new Date();
        const periodStart = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
        const periodEnd = now.toISOString();

        const owner = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Team Owner",
                handle: "teamowner9",
            },
        });
        testUserIds.push(owner.id);

        // Create multiple teams
        const teams = [];
        for (let i = 0; i < 3; i++) {
            const team = await DbProvider.get().team.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    createdById: owner.id,
                    handle: `batchteam${i}`,
                },
            });
            testTeamIds.push(team.id);
            teams.push(team);
        }

        await logTeamStats(PeriodType.Daily, periodStart, periodEnd);

        // Should create stats for all teams
        const allStats = await DbProvider.get().stats_team.findMany({
            where: {
                teamId: { in: teams.map(t => t.id) },
                periodStart,
                periodEnd,
                periodType: PeriodType.Daily,
            },
        });

        expect(allStats).toHaveLength(3);
        allStats.forEach(stat => testStatsTeamIds.push(stat.id));
    });
});
