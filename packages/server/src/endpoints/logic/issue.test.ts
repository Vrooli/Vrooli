import { type FindByIdInput, type IssueCloseInput, type IssueCreateInput, IssueFor, type IssueSearchInput, IssueStatus } from "@vrooli/shared";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { assertFindManyResultIds } from "../../__test/helpers.js";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { issue_closeOne } from "../generated/issue_closeOne.js";
import { issue_createOne } from "../generated/issue_createOne.js";
import { issue_findMany } from "../generated/issue_findMany.js";
import { issue_findOne } from "../generated/issue_findOne.js";
import { issue } from "./issue.js";
// Import database fixtures for seeding
import { seedIssues } from "../../__test/fixtures/db/issueFixtures.js";
import { UserDbFactory, seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
// Import validation fixtures for API input testing
import { issueTestDataFactory } from "@vrooli/shared";
import { cleanupGroups } from "../../__test/helpers/testCleanupHelpers.js";
import { validateCleanup } from "../../__test/helpers/testValidation.js";

/**
 * Test suite for the Issue endpoint (findOne, findMany, createOne, updateOne, closeOne)
 */
describe("EndpointsIssue", () => {
    let testUsers: any[];
    let team: any;
    let issues: any[];

    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);

        // Seed test users
        const seedResult = await seedTestUsers(DbProvider.get(), 2);
        testUsers = seedResult.records;
    });

    afterEach(async () => {
        // Validate cleanup to detect any missed records
        const orphans = await validateCleanup(DbProvider.get(), {
            tables: ["user", "user_auth", "email", "session"],
            logOrphans: true,
        });
        if (orphans.length > 0) {
            console.warn("Test cleanup incomplete:", orphans);
        }
    });

    beforeEach(async () => {
        // Clean up using dependency-ordered cleanup helpers
        await cleanupGroups.minimal(DbProvider.get());

        // Create a team for issue ownership
        team = await DbProvider.get().team.create({
            data: {
                id: UserDbFactory.createMinimal().id,
                publicId: UserDbFactory.createMinimal().publicId,
                permissions: "{}",
                owner: { connect: { id: testUsers[0].id } },
                translations: {
                    create: {
                        id: UserDbFactory.createMinimal().id,
                        language: "en",
                        name: "Test Team",
                    },
                },
            },
        });

        // Seed issues using database fixtures
        const issuesUser1 = await seedIssues(DbProvider.get(), {
            createdById: testUsers[0].id,
            count: 2,
            forObject: { id: team.id, type: "Team" },
            withTranslations: true,
        });

        // Add one more issue created by second user
        const issuesUser2 = await seedIssues(DbProvider.get(), {
            createdById: testUsers[1].id,
            count: 1,
            forObject: { id: team.id, type: "Team" },
            withTranslations: true,
        });

        issues = [...issuesUser1.records, ...issuesUser2.records];
    });

    afterAll(async () => {
        // Clean up
        await CacheService.get().flushAll();
        await DbProvider.deleteAll();

        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("returns own issue for authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });
                const input: FindByIdInput = { id: issues[0].id };
                const result = await issue.findOne({ input }, { req, res }, issue_findOne);
                expect(result).not.toBeNull();
                expect(result.id).toEqual(issues[0].id);
            });

            it("returns another user's issue", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });
                const input: FindByIdInput = { id: issues[2].id }; // Created by user 2
                const result = await issue.findOne({ input }, { req, res }, issue_findOne);
                expect(result).not.toBeNull();
                expect(result.id).toEqual(issues[2].id);
            });

            it("returns issue when not authenticated", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: FindByIdInput = { id: issues[0].id };
                const result = await issue.findOne({ input }, { req, res }, issue_findOne);
                expect(result).not.toBeNull();
                expect(result.id).toEqual(issues[0].id);
            });

            it("returns issue with API key public read", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, {
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });
                const input: FindByIdInput = { id: issues[1].id };
                const result = await issue.findOne({ input }, { req, res }, issue_findOne);
                expect(result).not.toBeNull();
                expect(result.id).toEqual(issues[1].id);
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns issues without filters for authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });
                const input: IssueSearchInput = { take: 10 };
                const expectedIds = issues.map(i => i.id);
                const result = await issue.findMany({ input }, { req, res }, issue_findMany);
                expect(result).not.toBeNull();
                expect(result.edges).toBeInstanceOf(Array);
                assertFindManyResultIds(expect, result, expectedIds);
            });

            it("returns issues by object type and id", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });
                const input: IssueSearchInput = {
                    forObjectType: IssueFor.Team,
                    forIds: [team.id],
                    take: 10,
                };
                const result = await issue.findMany({ input }, { req, res }, issue_findMany);
                expect(result).not.toBeNull();
                expect(result.edges).toBeInstanceOf(Array);
                expect(result.edges.length).toBe(3); // All issues are for the team
            });

            it("returns issues when not authenticated", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: IssueSearchInput = { take: 10 };
                const result = await issue.findMany({ input }, { req, res }, issue_findMany);
                expect(result).not.toBeNull();
                expect(result.edges).toBeInstanceOf(Array);
            });

            it("returns issues with API key public read", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, {
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });
                const input: IssueSearchInput = { take: 10 };
                const result = await issue.findMany({ input }, { req, res }, issue_findMany);
                expect(result).not.toBeNull();
                expect(result.edges).toBeInstanceOf(Array);
            });
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates an issue for authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                // Use validation fixtures for API input
                const input: IssueCreateInput = issueTestDataFactory.createMinimal({
                    forConnect: team.id,
                    translationsCreate: [{
                        language: "en",
                        name: "New Issue",
                        description: "Description of the new issue",
                    }],
                });

                const result = await issue.createOne({ input }, { req, res }, issue_createOne);
                expect(result).not.toBeNull();
                expect(result.id).toBeDefined();
                expect(result.translations).toHaveLength(1);
                expect(result.translations[0].name).toBe("New Issue");
                expect(result.status).toBe(IssueStatus.Open);
            });

            it("API key with write permissions can create issue", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, {
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                // Use complete fixture for comprehensive test
                const input: IssueCreateInput = issueTestDataFactory.createComplete({
                    forConnect: team.id,
                });

                const result = await issue.createOne({ input }, { req, res }, issue_createOne);
                expect(result).not.toBeNull();
                expect(result.id).toBeDefined();
            });
        });

        describe("invalid", () => {
            it("throws error for not logged in user", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: IssueCreateInput = issueTestDataFactory.createMinimal({
                    forConnect: team.id,
                });

                await expect(async () => {
                    await issue.createOne({ input }, { req, res }, issue_createOne);
                }).rejects.toThrow();
            });
        });
    });

    describe("closeOne", () => {
        describe("valid", () => {
            it("closes own issue", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                const input: IssueCloseInput = {
                    id: issues[0].id,
                    closedReason: "Issue has been resolved",
                };

                const result = await issue.closeOne({ input }, { req, res }, issue_closeOne);
                expect(result).not.toBeNull();
                expect(result.status).toBe(IssueStatus.ClosedResolved);
                expect(result.closedReason).toBe("Issue has been resolved");
                expect(result.closedAt).toBeDefined();
            });

            it("team owner can close team issue", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id, // Team owner
                });

                const input: IssueCloseInput = {
                    id: issues[2].id, // Issue created by user 2
                    closedReason: "Closed by team owner",
                    status: IssueStatus.ClosedDuplicate,
                };

                const result = await issue.closeOne({ input }, { req, res }, issue_closeOne);
                expect(result).not.toBeNull();
                expect(result.status).toBe(IssueStatus.ClosedDuplicate);
                expect(result.closedReason).toBe("Closed by team owner");
            });
        });

        describe("invalid", () => {
            it("throws error when not issue creator or team owner", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[1].id,
                });

                const input: IssueCloseInput = {
                    id: issues[0].id, // Created by user 0
                    closedReason: "Should not be able to close",
                };

                await expect(async () => {
                    await issue.closeOne({ input }, { req, res }, issue_closeOne);
                }).rejects.toThrow();
            });

            it("throws error for not logged in user", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: IssueCloseInput = {
                    id: issues[0].id,
                    closedReason: "Should not work",
                };

                await expect(async () => {
                    await issue.closeOne({ input }, { req, res }, issue_closeOne);
                }).rejects.toThrow();
            });
        });
    });
});
