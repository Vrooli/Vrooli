import { StatPeriodType, StatsTeamSearchInput, uuid } from "@local/shared";
import { PeriodType, team as TeamModelPrisma } from "@prisma/client";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { statsTeam_findMany } from "../generated/statsTeam_findMany.js"; // Assuming this generated type exists
import { statsTeam } from "./statsTeam.js";

// Test data
const testTeamId1 = uuid(); // Public team
const testTeamId2 = uuid(); // Private team for user1/team1
const testTeamId3 = uuid(); // Private team for user2/team2

// User IDs
const user1Id = uuid(); // Member of team 1 & 2
const user2Id = uuid(); // Member of team 1 & 3
const user3Id = uuid(); // Not a member of any tested team

// Sample Team data structure
const teamData1: Partial<TeamModelPrisma> & { id: string } = {
    id: testTeamId1,
    isPrivate: false,
    // Add other required Team fields (e.g., name, handle)
};

const teamData2: Partial<TeamModelPrisma> & { id: string } = {
    id: testTeamId2,
    isPrivate: true, // Team for user1
    // Add other required Team fields
};

const teamData3: Partial<TeamModelPrisma> & { id: string } = {
    id: testTeamId3,
    isPrivate: true, // Team for user2
    // Add other required Team fields
};

// Adjust fields based on actual StatsTeam model
const statsTeamData1 = {
    id: uuid(),
    teamId: testTeamId1,
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
};

const statsTeamData2 = {
    id: uuid(),
    teamId: testTeamId2,
    periodStart: new Date("2023-02-01"),
    periodEnd: new Date("2023-02-28"),
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
};

const statsTeamData3 = {
    id: uuid(),
    teamId: testTeamId3,
    periodStart: new Date("2023-03-01"),
    periodEnd: new Date("2023-03-31"),
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
};

describe("EndpointsStatsTeam", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async function beforeEach() {
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Create test users
        await DbProvider.get().user.create({
            data: { id: user1Id, name: "Test User 1", handle: "test-user-1", status: "Unlocked", isBot: false, isBotDepictingPerson: false, isPrivate: false, auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] } },
        });
        await DbProvider.get().user.create({
            data: { id: user2Id, name: "Test User 2", handle: "test-user-2", status: "Unlocked", isBot: false, isBotDepictingPerson: false, isPrivate: false, auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] } },
        });
        await DbProvider.get().user.create({
            data: { id: user3Id, name: "Test User 3", handle: "test-user-3", status: "Unlocked", isBot: false, isBotDepictingPerson: false, isPrivate: false, auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] } },
        });

        // Create test teams (ensure all required fields are present)
        // Placeholder: Add actual required fields for team creation (like permissions)
        await DbProvider.get().team.createMany({
            data: [
                { ...teamData1, permissions: JSON.stringify({}) /* name: "Public Team 1", handle: "public-team-1" */ }, // Removed name/handle, added permissions
                { ...teamData2, permissions: JSON.stringify({}) /* name: "Private Team 2", handle: "private-team-2" */ }, // Removed name/handle, added permissions
                { ...teamData3, permissions: JSON.stringify({}) /* name: "Private Team 3", handle: "private-team-3" */ },  // Removed name/handle, added permissions
            ],
        });

        // Create memberships (ensure all required fields are present)
        await DbProvider.get().member.createMany({
            data: [
                // User 1 in Public Team 1 and Private Team 2
                { id: uuid(), teamId: testTeamId1, userId: user1Id, permissions: JSON.stringify({}) /*, role: Prisma.MemberRole.Member */ }, // Added permissions
                { id: uuid(), teamId: testTeamId2, userId: user1Id, permissions: JSON.stringify({}) /*, role: Prisma.MemberRole.Owner */ }, // Added permissions
                // User 2 in Public Team 1 and Private Team 3
                { id: uuid(), teamId: testTeamId1, userId: user2Id, permissions: JSON.stringify({}) /*, role: Prisma.MemberRole.Member */ }, // Added permissions
                { id: uuid(), teamId: testTeamId3, userId: user2Id, permissions: JSON.stringify({}) /*, role: Prisma.MemberRole.Owner */ }, // Added permissions
            ],
        });

        // Create fresh test stats data
        await DbProvider.get().stats_team.createMany({
            data: [
                statsTeamData1, // Public team
                statsTeamData2, // Private team 2 (user1)
                statsTeamData3,  // Private team 3 (user2)
            ],
        });
    });

    after(async function after() {
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns stats for public teams and teams the user is a member of when logged in", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id }; // User 1 is in team 1 and 2
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsTeamSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsTeam.findMany({ input }, { req, res }, statsTeam_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                // User 1 should see stats for public team 1 and their private team 2
                expect(resultIds).to.include(statsTeamData1.id);
                expect(resultIds).to.include(statsTeamData2.id);
                // User 1 should NOT see stats for private team 3
                expect(resultIds).to.not.include(statsTeamData3.id);
            });

            it("returns correct stats for a different logged in user (user 2)", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user2Id }; // User 2 is in team 1 and 3
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsTeamSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsTeam.findMany({ input }, { req, res }, statsTeam_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                // User 2 should see stats for public team 1 and their private team 3
                expect(resultIds).to.include(statsTeamData1.id);
                expect(resultIds).to.include(statsTeamData3.id);
                // User 2 should NOT see stats for private team 2
                expect(resultIds).to.not.include(statsTeamData2.id);
            });

            it("returns only public stats for a user not in any private teams (user 3)", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user3Id }; // User 3 is not in team 2 or 3
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsTeamSearchInput = { periodType: StatPeriodType.Monthly };
                const result = await statsTeam.findMany({ input }, { req, res }, statsTeam_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                // User 3 should only see stats for public team 1
                expect(resultIds).to.include(statsTeamData1.id);
                expect(resultIds).to.not.include(statsTeamData2.id);
                expect(resultIds).to.not.include(statsTeamData3.id);
            });

            it("filters by periodType", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsTeamSearchInput = { periodType: StatPeriodType.Monthly };
                const result = await statsTeam.findMany({ input }, { req, res }, statsTeam_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);
                expect(resultIds).to.include(statsTeamData1.id); // Jan
                expect(resultIds).to.include(statsTeamData2.id); // Feb
                // Mar stats (team 3) shouldn't be visible to user 1 anyway
            });

            it("filters by time range", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsTeamSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: {
                        after: new Date("2023-01-01"),
                        before: new Date("2023-01-31"),
                    },
                };
                const result = await statsTeam.findMany({ input }, { req, res }, statsTeam_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);
                expect(resultIds).to.include(statsTeamData1.id); // Jan stats for public team 1
                expect(resultIds).to.not.include(statsTeamData2.id); // Feb stats for private team 2
                expect(resultIds).to.not.include(statsTeamData3.id); // Mar stats for private team 3
            });

            it("API key - public permissions returns only public team stats", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id }; // User context might still be needed by readManyHelper
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: StatsTeamSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                const result = await statsTeam.findMany({ input }, { req, res }, statsTeam_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                // Only public team stats should be returned
                expect(resultIds).to.include(statsTeamData1.id);
                expect(resultIds).to.not.include(statsTeamData2.id);
                expect(resultIds).to.not.include(statsTeamData3.id);
            });

            it("not logged in returns only public team stats", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: StatsTeamSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly,
                };
                // Assuming readManyHelper allows public access for teams
                const result = await statsTeam.findMany({ input }, { req, res }, statsTeam_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                expect(resultIds).to.include(statsTeamData1.id);
                expect(resultIds).to.not.include(statsTeamData2.id);
                expect(resultIds).to.not.include(statsTeamData3.id);
            });
        });

        describe("invalid", () => {
            it("invalid time range format should throw error", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsTeamSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: { after: new Date("invalid"), before: new Date("invalid") },
                };

                try {
                    await statsTeam.findMany({ input }, { req, res }, statsTeam_findMany);
                    expect.fail("Expected an error");
                } catch (error) {
                    expect(error).to.be.instanceOf(Error);
                }
            });

            it("invalid periodType should throw error", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = { periodType: "InvalidPeriod" as any };

                try {
                    await statsTeam.findMany({ input: input as StatsTeamSearchInput }, { req, res }, statsTeam_findMany);
                    expect.fail("Expected an error");
                } catch (error) {
                    expect(error).to.be.instanceOf(Error);
                }
            });

            it("cannot see stats of private team you are not a member of when searching by name", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id }; // User 1 is NOT in team 3
                const { req, res } = await mockAuthenticatedSession(testUser);

                // Search for team 3 by name
                const input: StatsTeamSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    searchString: "Private Team 3", // Name of team user1 is not in
                };
                const result = await statsTeam.findMany({ input }, { req, res }, statsTeam_findMany);

                expect(result).to.not.be.null;
                expect(result.edges!.length).to.equal(0);
                expect(result.edges!.every(edge => edge?.node?.id !== statsTeamData3.id)).to.be.true;
            });
        });
    });
}); 
