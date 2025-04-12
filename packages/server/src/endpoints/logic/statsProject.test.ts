import { ApiKeyPermission, StatPeriodType, StatsProjectSearchInput, uuid } from "@local/shared";
import { PeriodType, project as ProjectModelPrisma } from "@prisma/client"; // Correct import
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { statsProject_findMany } from "../generated/statsProject_findMany.js"; // Assuming this generated type exists
import { statsProject } from "./statsProject.js";

// Test data
const testProjectId1 = uuid();
const testProjectId2 = uuid();
const privateProjectId1 = uuid(); // Private Project owned by user1
const privateProjectId2 = uuid(); // Private Project owned by user2

// User IDs for ownership testing
const user1Id = uuid();
const user2Id = uuid();

// Sample Project data structure (adjust fields as necessary based on actual Project model)
// Projects might have team ownership or different privacy fields
const projectData1: Partial<ProjectModelPrisma> & { id: string } = {
    id: testProjectId1,
    isPrivate: false,
    ownedByUserId: null, // Placeholder: Assuming direct user ownership is possible and null means public
    // ownedByTeamId: null,
    // Add other required Project fields (e.g., name, versions)
};

const projectData2: Partial<ProjectModelPrisma> & { id: string } = {
    id: testProjectId2,
    isPrivate: false,
    ownedByUserId: null,
    // Add other required Project fields
};

const privateProjectData1: Partial<ProjectModelPrisma> & { id: string } = {
    id: privateProjectId1,
    isPrivate: true,
    ownedByUserId: user1Id, // Placeholder: Assuming owned by user1
    // Add other required Project fields
};

const privateProjectData2: Partial<ProjectModelPrisma> & { id: string } = {
    id: privateProjectId2,
    isPrivate: true,
    ownedByUserId: user2Id, // Placeholder: Assuming owned by user2
    // Add other required Project fields
};

const statsProjectData1 = {
    id: uuid(),
    projectId: testProjectId1,
    periodStart: new Date("2023-01-01"),
    periodEnd: new Date("2023-01-31"),
    periodType: PeriodType.Monthly,
    directories: 0,
    apis: 0,
    codes: 0,
    notes: 0,
    routines: 0,
    standards: 0,
    runCompletionTimeAverage: 0.0,
    projects: 0,
    runsStarted: 0,
    runsCompleted: 0,
    runContextSwitchesAverage: 0.0,
    teams: 0,
};

const statsProjectData2 = {
    id: uuid(),
    projectId: testProjectId2,
    periodStart: new Date("2023-02-01"),
    periodEnd: new Date("2023-02-28"),
    periodType: PeriodType.Monthly,
    directories: 0,
    apis: 0,
    codes: 0,
    notes: 0,
    routines: 0,
    standards: 0,
    runCompletionTimeAverage: 0.0,
    projects: 0,
    runsStarted: 0,
    runsCompleted: 0,
    runContextSwitchesAverage: 0.0,
    teams: 0,
};

const privateProjectStats1 = {
    id: uuid(),
    projectId: privateProjectId1,
    periodStart: new Date("2023-03-01"),
    periodEnd: new Date("2023-03-31"),
    periodType: PeriodType.Monthly,
    directories: 0,
    apis: 0,
    codes: 0,
    notes: 0,
    routines: 0,
    standards: 0,
    runCompletionTimeAverage: 0.0,
    projects: 0,
    runsStarted: 0,
    runsCompleted: 0,
    runContextSwitchesAverage: 0.0,
    teams: 0,
};

const privateProjectStats2 = {
    id: uuid(),
    projectId: privateProjectId2,
    periodStart: new Date("2023-03-01"),
    periodEnd: new Date("2023-03-31"),
    periodType: PeriodType.Monthly,
    directories: 0,
    apis: 0,
    codes: 0,
    notes: 0,
    routines: 0,
    standards: 0,
    runCompletionTimeAverage: 0.0,
    projects: 0,
    runsStarted: 0,
    runsCompleted: 0,
    runContextSwitchesAverage: 0.0,
    teams: 0,
};

describe("EndpointsStatsProject", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async function beforeEach() {
        this.timeout(15_000);

        // Clean previous test data
        await DbProvider.get().stats_project.deleteMany({
            where: { projectId: { in: [testProjectId1, testProjectId2, privateProjectId1, privateProjectId2] } }
        });
        await DbProvider.get().project.deleteMany({
            where: { id: { in: [testProjectId1, testProjectId2, privateProjectId1, privateProjectId2] } }
        });
        await DbProvider.get().user.deleteMany({
            where: { id: { in: [user1Id, user2Id] } }
        });

        // Create test users individually
        await DbProvider.get().user.create({
            data: { id: user1Id, name: "Test User 1", handle: "test-user-1", status: "Unlocked", isBot: false, isBotDepictingPerson: false, isPrivate: false, auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] } }
        });
        await DbProvider.get().user.create({
            data: { id: user2Id, name: "Test User 2", handle: "test-user-2", status: "Unlocked", isBot: false, isBotDepictingPerson: false, isPrivate: false, auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] } }
        });

        // Create test projects (ensure all required fields are present)
        // Placeholder: Assuming projects need a name, permissions and potentially versions
        await DbProvider.get().project.createMany({
            data: [
                { ...projectData1, permissions: JSON.stringify({}), /* versions: { create: [...] } */ },
                { ...projectData2, permissions: JSON.stringify({}), /* versions: { create: [...] } */ },
                { ...privateProjectData1, permissions: JSON.stringify({}), /* versions: { create: [...] } */ },
                { ...privateProjectData2, permissions: JSON.stringify({}), /* versions: { create: [...] } */ }
            ].map(p => ({ // Adjust ownership fields based on actual model
                ...p,
                ownedByUserId: p.ownedByUserId ?? undefined,
                // ownedByTeamId: p.ownedByTeamId ?? undefined,
            }))
        });

        // Create fresh test stats data
        await DbProvider.get().stats_project.createMany({
            data: [
                statsProjectData1,
                statsProjectData2,
                privateProjectStats1,
                privateProjectStats2
            ]
        });
    });

    after(async function after() {
        this.timeout(15_000);

        // Clean up test data
        await DbProvider.get().stats_project.deleteMany({
            where: { projectId: { in: [testProjectId1, testProjectId2, privateProjectId1, privateProjectId2] } }
        });
        await DbProvider.get().project.deleteMany({
            where: { id: { in: [testProjectId1, testProjectId2, privateProjectId1, privateProjectId2] } }
        });
        await DbProvider.get().user.deleteMany({
            where: { id: { in: [user1Id, user2Id] } }
        });

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns stats for public and owned projects when logged in", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id }; // User 1 owns privateProject1
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsProjectSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly
                };
                const result = await statsProject.findMany({ input }, { req, res }, statsProject_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                // User 1 should see public projects and their own private project
                expect(resultIds).to.include(statsProjectData1.id);
                expect(resultIds).to.include(statsProjectData2.id);
                expect(resultIds).to.include(privateProjectStats1.id);
                // User 1 should NOT see user 2's private project stats
                expect(resultIds).to.not.include(privateProjectStats2.id);
            });

            it("filters by periodType", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsProjectSearchInput = { periodType: StatPeriodType.Monthly };
                const result = await statsProject.findMany({ input }, { req, res }, statsProject_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);
                expect(resultIds).to.include(statsProjectData1.id);
                expect(resultIds).to.include(statsProjectData2.id);
                expect(resultIds).to.include(privateProjectStats1.id);
            });

            it("filters by time range", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsProjectSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: {
                        after: new Date("2023-01-01"),
                        before: new Date("2023-01-31")
                    }
                };
                const result = await statsProject.findMany({ input }, { req, res }, statsProject_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);
                expect(resultIds).to.include(statsProjectData1.id); // Should include Jan stats
                expect(resultIds).to.not.include(statsProjectData2.id); // Should exclude Feb stats
                expect(resultIds).to.not.include(privateProjectStats1.id); // Should exclude Mar stats
            });

            it("API key - public permissions returns only public projects", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = { [ApiKeyPermission.ReadPublic]: true } as Record<ApiKeyPermission, boolean>;
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: StatsProjectSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly
                };
                const result = await statsProject.findMany({ input }, { req, res }, statsProject_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                expect(resultIds).to.include(statsProjectData1.id);
                expect(resultIds).to.include(statsProjectData2.id);
                expect(resultIds).to.not.include(privateProjectStats1.id);
                expect(resultIds).to.not.include(privateProjectStats2.id);
            });

            it("not logged in returns only public projects", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: StatsProjectSearchInput = {
                    take: 10,
                    periodType: StatPeriodType.Monthly
                };
                // Assuming readManyHelper allows public access for projects when not logged in
                const result = await statsProject.findMany({ input }, { req, res }, statsProject_findMany);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                const resultIds = result.edges!.map(edge => edge?.node?.id);

                expect(resultIds).to.include(statsProjectData1.id);
                expect(resultIds).to.include(statsProjectData2.id);
                expect(resultIds).to.not.include(privateProjectStats1.id);
                expect(resultIds).to.not.include(privateProjectStats2.id);
            });
        });

        describe("invalid", () => {
            it("invalid time range format should throw error", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: StatsProjectSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    periodTimeFrame: { after: new Date("invalid"), before: new Date("invalid") }
                };

                try {
                    await statsProject.findMany({ input }, { req, res }, statsProject_findMany);
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
                    await statsProject.findMany({ input: input as StatsProjectSearchInput }, { req, res }, statsProject_findMany);
                    expect.fail("Expected an error");
                } catch (error) {
                    expect(error).to.be.instanceOf(Error);
                }
            });

            it("cannot see stats of private project you don't own when searching by name", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id }; // User 1
                const { req, res } = await mockAuthenticatedSession(testUser);

                // Search for User 2's private project
                const input: StatsProjectSearchInput = {
                    periodType: StatPeriodType.Monthly,
                    searchString: "Private Project 2"
                };
                const result = await statsProject.findMany({ input }, { req, res }, statsProject_findMany);

                expect(result).to.not.be.null;
                expect(result.edges!.length).to.equal(0);
                expect(result.edges!.every(edge => edge?.node?.id !== privateProjectStats2.id)).to.be.true;
            });
        });
    });
}); 