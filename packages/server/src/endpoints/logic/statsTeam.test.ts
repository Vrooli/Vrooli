import { StatPeriodType, type StatsTeamSearchInput, generatePK, generatePublicId } from "@vrooli/shared";
import { PeriodType } from "@prisma/client";
import { describe, it, expect, beforeAll, afterAll, vi } from "vitest";
import { UserDbFactory } from "../../__test/fixtures/db/userFixtures.js";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { statsTeam_findMany } from "../generated/statsTeam_findMany.js";
import { statsTeam } from "./statsTeam.js";

describe("EndpointsStatsTeam", () => {
    let loggerErrorStub: any;
    let loggerInfoStub: any;

    beforeAll(async () => {
        loggerErrorStub = vi.spyOn(logger, "error").mockImplementation(() => undefined);
        loggerInfoStub = vi.spyOn(logger, "info").mockImplementation(() => undefined);
    });

    beforeEach(async () => {
        // Clean up tables used in tests
        const prisma = DbProvider.get();
        await prisma.member.deleteMany();
        await prisma.stats_team.deleteMany();
        await prisma.team.deleteMany();
        await prisma.user.deleteMany();
    });

    afterAll(async () => {
        loggerErrorStub.mockRestore();
        loggerInfoStub.mockRestore();
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns stats for public teams and teams the user is a member of when logged in", async () => {
                // Create users
                const user1 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1",
                        handle: "test-user-1",
                    }),
                });

                const user2 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 2",
                        handle: "test-user-2",
                    }),
                });

                // Create teams
                const publicTeam = await DbProvider.get().team.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        isPrivate: false,
                        translations: {
                            create: {
                                id: generatePK(),
                                language: "en",
                                name: "Public Team 1",
                                bio: "A public team",
                            },
                        },
                    },
                });

                const privateTeam1 = await DbProvider.get().team.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        isPrivate: true,
                        translations: {
                            create: {
                                id: generatePK(),
                                language: "en",
                                name: "Private Team 2",
                                bio: "User 1's private team",
                            },
                        },
                    },
                });

                const privateTeam2 = await DbProvider.get().team.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        isPrivate: true,
                        translations: {
                            create: {
                                id: generatePK(),
                                language: "en",
                                name: "Private Team 3",
                                bio: "User 2's private team",
                            },
                        },
                    },
                });

                // Create memberships
                await DbProvider.get().member.createMany({
                    data: [
                        // User 1 in public team and private team 1
                        { id: generatePK(), publicId: generatePublicId(), teamId: publicTeam.id, userId: user1.id, isAdmin: false },
                        { id: generatePK(), publicId: generatePublicId(), teamId: privateTeam1.id, userId: user1.id, isAdmin: true },
                        // User 2 in public team and private team 2
                        { id: generatePK(), publicId: generatePublicId(), teamId: publicTeam.id, userId: user2.id, isAdmin: false },
                        { id: generatePK(), publicId: generatePublicId(), teamId: privateTeam2.id, userId: user2.id, isAdmin: true },
                    ],
                });

                // Create stats
                const publicTeamStats = await DbProvider.get().stats_team.create({
                    data: {
                        id: generatePK(),
                        teamId: publicTeam.id,
                        periodStart: new Date("2023-01-01"),
                        periodEnd: new Date("2023-01-31"),
                        periodType: PeriodType.Monthly,
                        apis: 0,
                        codes: 0,
                        members: 2,
                        notes: 0,
                        projects: 0,
                        routines: 0,
                        standards: 0,
                        runRoutinesStarted: 0,
                        runRoutinesCompleted: 0,
                        runRoutineCompletionTimeAverage: 0.0,
                        runRoutineContextSwitchesAverage: 0.0,
                    },
                });

                const privateTeam1Stats = await DbProvider.get().stats_team.create({
                    data: {
                        id: generatePK(),
                        teamId: privateTeam1.id,
                        periodStart: new Date("2023-02-01"),
                        periodEnd: new Date("2023-02-28"),
                        periodType: PeriodType.Monthly,
                        apis: 0,
                        codes: 0,
                        members: 1,
                        notes: 0,
                        projects: 0,
                        routines: 0,
                        standards: 0,
                        runRoutinesStarted: 0,
                        runRoutinesCompleted: 0,
                        runRoutineCompletionTimeAverage: 0.0,
                        runRoutineContextSwitchesAverage: 0.0,
                    },
                });

                const privateTeam2Stats = await DbProvider.get().stats_team.create({
                    data: {
                        id: generatePK(),
                        teamId: privateTeam2.id,
                        periodStart: new Date("2023-03-01"),
                        periodEnd: new Date("2023-03-31"),
                        periodType: PeriodType.Monthly,
                        apis: 0,
                        codes: 0,
                        members: 1,
                        notes: 0,
                        projects: 0,
                        routines: 0,
                        standards: 0,
                        runRoutinesStarted: 0,
                        runRoutinesCompleted: 0,
                        runRoutineCompletionTimeAverage: 0.0,
                        runRoutineContextSwitchesAverage: 0.0,
                    },
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: user1.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsTeamSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsTeam.findMany({ input }, { req, res }, statsTeam_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                // User 1 should see stats for public team and their private team 1
                expect(resultIds).toContain(publicTeamStats.id.toString());
                expect(resultIds).toContain(privateTeam1Stats.id.toString());
                // User 1 should NOT see stats for private team 2
                expect(resultIds).not.toContain(privateTeam2Stats.id.toString());
            });

            it("returns correct stats for a different logged in user (user 2)", async () => {
                // Create users
                const user1 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1",
                        handle: "test-user-1",
                    }),
                });

                const user2 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 2",
                        handle: "test-user-2",
                    }),
                });

                // Create teams
                const publicTeam = await DbProvider.get().team.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        isPrivate: false,
                        translations: {
                            create: {
                                id: generatePK(),
                                language: "en",
                                name: "Public Team 1",
                                bio: "A public team",
                            },
                        },
                    },
                });

                const privateTeam1 = await DbProvider.get().team.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        isPrivate: true,
                        translations: {
                            create: {
                                id: generatePK(),
                                language: "en",
                                name: "Private Team 2",
                                bio: "User 1's private team",
                            },
                        },
                    },
                });

                const privateTeam2 = await DbProvider.get().team.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        isPrivate: true,
                        translations: {
                            create: {
                                id: generatePK(),
                                language: "en",
                                name: "Private Team 3",
                                bio: "User 2's private team",
                            },
                        },
                    },
                });

                // Create memberships
                await DbProvider.get().member.createMany({
                    data: [
                        // User 1 in public team and private team 1
                        { id: generatePK(), publicId: generatePublicId(), teamId: publicTeam.id, userId: user1.id, isAdmin: false },
                        { id: generatePK(), publicId: generatePublicId(), teamId: privateTeam1.id, userId: user1.id, isAdmin: true },
                        // User 2 in public team and private team 2
                        { id: generatePK(), publicId: generatePublicId(), teamId: publicTeam.id, userId: user2.id, isAdmin: false },
                        { id: generatePK(), publicId: generatePublicId(), teamId: privateTeam2.id, userId: user2.id, isAdmin: true },
                    ],
                });

                // Create stats
                const publicTeamStats = await DbProvider.get().stats_team.create({
                    data: {
                        id: generatePK(),
                        teamId: publicTeam.id,
                        periodStart: new Date("2023-01-01"),
                        periodEnd: new Date("2023-01-31"),
                        periodType: PeriodType.Monthly,
                        apis: 0,
                        codes: 0,
                        members: 2,
                        notes: 0,
                        projects: 0,
                        routines: 0,
                        standards: 0,
                        runRoutinesStarted: 0,
                        runRoutinesCompleted: 0,
                        runRoutineCompletionTimeAverage: 0.0,
                        runRoutineContextSwitchesAverage: 0.0,
                    },
                });

                const privateTeam1Stats = await DbProvider.get().stats_team.create({
                    data: {
                        id: generatePK(),
                        teamId: privateTeam1.id,
                        periodStart: new Date("2023-02-01"),
                        periodEnd: new Date("2023-02-28"),
                        periodType: PeriodType.Monthly,
                        apis: 0,
                        codes: 0,
                        members: 1,
                        notes: 0,
                        projects: 0,
                        routines: 0,
                        standards: 0,
                        runRoutinesStarted: 0,
                        runRoutinesCompleted: 0,
                        runRoutineCompletionTimeAverage: 0.0,
                        runRoutineContextSwitchesAverage: 0.0,
                    },
                });

                const privateTeam2Stats = await DbProvider.get().stats_team.create({
                    data: {
                        id: generatePK(),
                        teamId: privateTeam2.id,
                        periodStart: new Date("2023-03-01"),
                        periodEnd: new Date("2023-03-31"),
                        periodType: PeriodType.Monthly,
                        apis: 0,
                        codes: 0,
                        members: 1,
                        notes: 0,
                        projects: 0,
                        routines: 0,
                        standards: 0,
                        runRoutinesStarted: 0,
                        runRoutinesCompleted: 0,
                        runRoutineCompletionTimeAverage: 0.0,
                        runRoutineContextSwitchesAverage: 0.0,
                    },
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: user2.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsTeamSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsTeam.findMany({ input }, { req, res }, statsTeam_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                // User 2 should see stats for public team and their private team 2
                expect(resultIds).toContain(publicTeamStats.id.toString());
                expect(resultIds).toContain(privateTeam2Stats.id.toString());
                // User 2 should NOT see stats for private team 1
                expect(resultIds).not.toContain(privateTeam1Stats.id.toString());
            });

            it("returns only public stats for a user not in any private teams", async () => {
                const user3 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 3",
                        handle: "test-user-3",
                    }),
                });

                // Create teams
                const publicTeam = await DbProvider.get().team.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        isPrivate: false,
                        translations: {
                            create: {
                                id: generatePK(),
                                language: "en",
                                name: "Public Team 1",
                                bio: "A public team",
                            },
                        },
                    },
                });

                const privateTeam1 = await DbProvider.get().team.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        isPrivate: true,
                        translations: {
                            create: {
                                id: generatePK(),
                                language: "en",
                                name: "Private Team 2",
                                bio: "A private team",
                            },
                        },
                    },
                });

                // Create stats
                const publicTeamStats = await DbProvider.get().stats_team.create({
                    data: {
                        id: generatePK(),
                        teamId: publicTeam.id,
                        periodStart: new Date("2023-01-01"),
                        periodEnd: new Date("2023-01-31"),
                        periodType: PeriodType.Monthly,
                        apis: 0,
                        codes: 0,
                        members: 0,
                        notes: 0,
                        projects: 0,
                        routines: 0,
                        standards: 0,
                        runRoutinesStarted: 0,
                        runRoutinesCompleted: 0,
                        runRoutineCompletionTimeAverage: 0.0,
                        runRoutineContextSwitchesAverage: 0.0,
                    },
                });

                const privateTeamStats = await DbProvider.get().stats_team.create({
                    data: {
                        id: generatePK(),
                        teamId: privateTeam1.id,
                        periodStart: new Date("2023-01-01"),
                        periodEnd: new Date("2023-01-31"),
                        periodType: PeriodType.Monthly,
                        apis: 0,
                        codes: 0,
                        members: 0,
                        notes: 0,
                        projects: 0,
                        routines: 0,
                        standards: 0,
                        runRoutinesStarted: 0,
                        runRoutinesCompleted: 0,
                        runRoutineCompletionTimeAverage: 0.0,
                        runRoutineContextSwitchesAverage: 0.0,
                    },
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: user3.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsTeamSearchInput = { periodType: StatPeriodType.Monthly };
                const result = await statsTeam.findMany({ input }, { req, res }, statsTeam_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                // User 3 should only see stats for public team
                expect(resultIds).toContain(publicTeamStats.id.toString());
                expect(resultIds).not.toContain(privateTeamStats.id.toString());
            });

            it("filters by periodType", async () => {
                const user1 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1",
                        handle: "test-user-1",
                    }),
                });

                const team = await DbProvider.get().team.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        isPrivate: false,
                        translations: {
                            create: {
                                id: generatePK(),
                                language: "en",
                                name: "Test Team",
                                bio: "A test team",
                            },
                        },
                    },
                });

                await DbProvider.get().member.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        teamId: team.id,
                        userId: user1.id,
                        isAdmin: true,
                    },
                });

                const monthlyStats = await DbProvider.get().stats_team.create({
                    data: {
                        id: generatePK(),
                        teamId: team.id,
                        periodStart: new Date("2023-01-01"),
                        periodEnd: new Date("2023-01-31"),
                        periodType: PeriodType.Monthly,
                        apis: 0,
                        codes: 0,
                        members: 1,
                        notes: 0,
                        projects: 0,
                        routines: 0,
                        standards: 0,
                        runRoutinesStarted: 0,
                        runRoutinesCompleted: 0,
                        runRoutineCompletionTimeAverage: 0.0,
                        runRoutineContextSwitchesAverage: 0.0,
                    },
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: user1.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsTeamSearchInput = { periodType: StatPeriodType.Monthly };
                const result = await statsTeam.findMany({ input }, { req, res }, statsTeam_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                const resultIds = result.edges!.map(edge => edge?.node?.id);
                expect(resultIds).toContain(monthlyStats.id.toString());
            });

            it("filters by time range", async () => {
                const user1 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1",
                        handle: "test-user-1",
                    }),
                });

                const team = await DbProvider.get().team.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        isPrivate: false,
                        translations: {
                            create: {
                                id: generatePK(),
                                language: "en",
                                name: "Test Team",
                                bio: "A test team",
                            },
                        },
                    },
                });

                await DbProvider.get().member.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        teamId: team.id,
                        userId: user1.id,
                        isAdmin: true,
                    },
                });

                const janStats = await DbProvider.get().stats_team.create({
                    data: {
                        id: generatePK(),
                        teamId: team.id,
                        periodStart: new Date("2023-01-01"),
                        periodEnd: new Date("2023-01-31"),
                        periodType: PeriodType.Monthly,
                        apis: 0,
                        codes: 0,
                        members: 1,
                        notes: 0,
                        projects: 0,
                        routines: 0,
                        standards: 0,
                        runRoutinesStarted: 0,
                        runRoutinesCompleted: 0,
                        runRoutineCompletionTimeAverage: 0.0,
                        runRoutineContextSwitchesAverage: 0.0,
                    },
                });

                const febStats = await DbProvider.get().stats_team.create({
                    data: {
                        id: generatePK(),
                        teamId: team.id,
                        periodStart: new Date("2023-02-01"),
                        periodEnd: new Date("2023-02-28"),
                        periodType: PeriodType.Monthly,
                        apis: 0,
                        codes: 0,
                        members: 1,
                        notes: 0,
                        projects: 0,
                        routines: 0,
                        standards: 0,
                        runRoutinesStarted: 0,
                        runRoutinesCompleted: 0,
                        runRoutineCompletionTimeAverage: 0.0,
                        runRoutineContextSwitchesAverage: 0.0,
                    },
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: user1.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsTeamSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: {
                        after: new Date("2023-01-01"),
                        before: new Date("2023-01-31"),
                    },
                };
                const result = await statsTeam.findMany({ input }, { req, res }, statsTeam_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                const resultIds = result.edges!.map(edge => edge?.node?.id);
                expect(resultIds).toContain(janStats.id.toString());
                expect(resultIds).not.toContain(febStats.id.toString());
            });

            it("API key - public permissions returns only public team stats", async () => {
                const user1 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1",
                        handle: "test-user-1",
                    }),
                });

                const publicTeam = await DbProvider.get().team.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        isPrivate: false,
                        translations: {
                            create: {
                                id: generatePK(),
                                language: "en",
                                name: "Public Team",
                                bio: "A public team",
                            },
                        },
                    },
                });

                const privateTeam = await DbProvider.get().team.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        isPrivate: true,
                        translations: {
                            create: {
                                id: generatePK(),
                                language: "en",
                                name: "Private Team",
                                bio: "A private team",
                            },
                        },
                    },
                });

                const publicTeamStats = await DbProvider.get().stats_team.create({
                    data: {
                        id: generatePK(),
                        teamId: publicTeam.id,
                        periodStart: new Date("2023-01-01"),
                        periodEnd: new Date("2023-01-31"),
                        periodType: PeriodType.Monthly,
                        apis: 0,
                        codes: 0,
                        members: 0,
                        notes: 0,
                        projects: 0,
                        routines: 0,
                        standards: 0,
                        runRoutinesStarted: 0,
                        runRoutinesCompleted: 0,
                        runRoutineCompletionTimeAverage: 0.0,
                        runRoutineContextSwitchesAverage: 0.0,
                    },
                });

                const privateTeamStats = await DbProvider.get().stats_team.create({
                    data: {
                        id: generatePK(),
                        teamId: privateTeam.id,
                        periodStart: new Date("2023-01-01"),
                        periodEnd: new Date("2023-01-31"),
                        periodType: PeriodType.Monthly,
                        apis: 0,
                        codes: 0,
                        members: 0,
                        notes: 0,
                        projects: 0,
                        routines: 0,
                        standards: 0,
                        runRoutinesStarted: 0,
                        runRoutinesCompleted: 0,
                        runRoutineCompletionTimeAverage: 0.0,
                        runRoutineContextSwitchesAverage: 0.0,
                    },
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: user1.id.toString() };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: StatsTeamSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsTeam.findMany({ input }, { req, res }, statsTeam_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                // Only public team stats should be returned
                expect(resultIds).toContain(publicTeamStats.id.toString());
                expect(resultIds).not.toContain(privateTeamStats.id.toString());
            });

            it("not logged in returns only public team stats", async () => {
                const publicTeam = await DbProvider.get().team.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        isPrivate: false,
                        translations: {
                            create: {
                                id: generatePK(),
                                language: "en",
                                name: "Public Team",
                                bio: "A public team",
                            },
                        },
                    },
                });

                const privateTeam = await DbProvider.get().team.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        isPrivate: true,
                        translations: {
                            create: {
                                id: generatePK(),
                                language: "en",
                                name: "Private Team",
                                bio: "A private team",
                            },
                        },
                    },
                });

                const publicTeamStats = await DbProvider.get().stats_team.create({
                    data: {
                        id: generatePK(),
                        teamId: publicTeam.id,
                        periodStart: new Date("2023-01-01"),
                        periodEnd: new Date("2023-01-31"),
                        periodType: PeriodType.Monthly,
                        apis: 0,
                        codes: 0,
                        members: 0,
                        notes: 0,
                        projects: 0,
                        routines: 0,
                        standards: 0,
                        runRoutinesStarted: 0,
                        runRoutinesCompleted: 0,
                        runRoutineCompletionTimeAverage: 0.0,
                        runRoutineContextSwitchesAverage: 0.0,
                    },
                });

                const privateTeamStats = await DbProvider.get().stats_team.create({
                    data: {
                        id: generatePK(),
                        teamId: privateTeam.id,
                        periodStart: new Date("2023-01-01"),
                        periodEnd: new Date("2023-01-31"),
                        periodType: PeriodType.Monthly,
                        apis: 0,
                        codes: 0,
                        members: 0,
                        notes: 0,
                        projects: 0,
                        routines: 0,
                        standards: 0,
                        runRoutinesStarted: 0,
                        runRoutinesCompleted: 0,
                        runRoutineCompletionTimeAverage: 0.0,
                        runRoutineContextSwitchesAverage: 0.0,
                    },
                });

                const { req, res } = await mockLoggedOutSession();

                const input: StatsTeamSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsTeam.findMany({ input }, { req, res }, statsTeam_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                expect(resultIds).toContain(publicTeamStats.id.toString());
                expect(resultIds).not.toContain(privateTeamStats.id.toString());
            });
        });

        describe("invalid", () => {
            it("invalid time range format should throw error", async () => {
                const user1 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1",
                        handle: "test-user-1",
                    }),
                });
                
                const testUser = { ...loggedInUserNoPremiumData(), id: user1.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsTeamSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: { after: new Date("invalid"), before: new Date("invalid") },
                };

                try {
                    await statsTeam.findMany({ input }, { req, res }, statsTeam_findMany);
                    expect.fail("Expected an error");
                } catch (error) {
                    expect(error).toBeInstanceOf(Error);
                }
            });

            it("invalid periodType should throw error", async () => {
                const user1 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1",
                        handle: "test-user-1",
                    }),
                });
                
                const testUser = { ...loggedInUserNoPremiumData(), id: user1.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = { periodType: "InvalidPeriod" as any };

                try {
                    await statsTeam.findMany({ input: input as StatsTeamSearchInput }, { req, res }, statsTeam_findMany);
                    expect.fail("Expected an error");
                } catch (error) {
                    expect(error).toBeInstanceOf(Error);
                }
            });

            it("cannot see stats of private team you are not a member of when searching by name", async () => {
                const user1 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1",
                        handle: "test-user-1",
                    }),
                });

                const privateTeam = await DbProvider.get().team.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        isPrivate: true,
                        translations: {
                            create: {
                                id: generatePK(),
                                language: "en",
                                name: "Private Team 3",
                                bio: "A team user is not in",
                            },
                        },
                    },
                });

                await DbProvider.get().stats_team.create({
                    data: {
                        id: generatePK(),
                        teamId: privateTeam.id,
                        periodStart: new Date("2023-01-01"),
                        periodEnd: new Date("2023-01-31"),
                        periodType: PeriodType.Monthly,
                        apis: 0,
                        codes: 0,
                        members: 0,
                        notes: 0,
                        projects: 0,
                        routines: 0,
                        standards: 0,
                        runRoutinesStarted: 0,
                        runRoutinesCompleted: 0,
                        runRoutineCompletionTimeAverage: 0.0,
                        runRoutineContextSwitchesAverage: 0.0,
                    },
                });
                
                const testUser = { ...loggedInUserNoPremiumData(), id: user1.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                // Search for team by name
                const input: StatsTeamSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    searchString: "Private Team 3",
                };
                const result = await statsTeam.findMany({ input }, { req, res }, statsTeam_findMany);

                expect(result).not.toBeNull();
                expect(result.edges!.length).toBe(0);
            });
        });
    });
});