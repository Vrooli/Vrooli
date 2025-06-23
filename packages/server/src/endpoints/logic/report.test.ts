import { type FindByIdInput, type ReportCreateInput, ReportFor, type ReportSearchInput, ReportStatus, type ReportUpdateInput, generatePK, generatePublicId } from "@vrooli/shared";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { assertFindManyResultIds } from "../../__test/helpers.js";
import { loggedInUserNoPremiumData, mockAuthenticatedSession, mockLoggedOutSession, seedMockAdminUser } from "../../__test/session.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { report_createOne } from "../generated/report_createOne.js";
import { report_findMany } from "../generated/report_findMany.js";
import { report_findOne } from "../generated/report_findOne.js";
import { report_updateOne } from "../generated/report_updateOne.js";
import { report } from "./report.js";
// Import database fixtures for seeding
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
// Use local report factory
import { ReportDbFactory } from "../../__test/fixtures/db/reportFixtures.js";

describe("EndpointsReport", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
    });

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    afterEach(async () => {
        // Clean up test data after each test
        await DbProvider.get().report.deleteMany({});
        await DbProvider.get().user.deleteMany({});
    });

    // Helper function to create test data
    const createTestData = async () => {
        // Note: CacheService.get().flushAll() removed - not needed with transactions
        
        // Seed admin user and test users using database fixtures
        const adminUser = await seedMockAdminUser();
        const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

        // Create reports directly since seedReports doesn't support status
        const report1 = await DbProvider.get().report.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: testUsers[0].id,
                userId: testUsers[0].id,
                reason: "Test reason 1",
                details: "Test details 1",
                language: "en",
                status: ReportStatus.Open,
            },
        });
        const seededReport1 = report1;

        const report2 = await DbProvider.get().report.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                createdById: testUsers[1].id,
                userId: testUsers[1].id,
                reason: "Test reason 2",
                details: "Test details 2",
                language: "en",
                status: ReportStatus.Open,
            },
        });
        const seededReport2 = report2;
        
        return { adminUser, testUsers, seededReport1, seededReport2 };
    };

    describe("findOne", () => {
        it("returns a report by id for authenticated user", async () => {
            const { testUsers, seededReport1 } = await createTestData();
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id,
            });
            const input: FindByIdInput = { id: seededReport1.id };
            const result = await report.findOne({ input }, { req, res }, report_findOne);
            expect(result).not.toBeNull();
            expect(result.id).toEqual(seededReport1.id);
            expect(result.reason).toEqual(seededReport1.reason);
        });

        it("returns a report by id when not authenticated", async () => {
            const { seededReport2 } = await createTestData();
            const { req, res } = await mockLoggedOutSession();
            const input: FindByIdInput = { id: seededReport2.id };
            const result = await report.findOne({ input }, { req, res }, report_findOne);
            expect(result).not.toBeNull();
            expect(result.id).toEqual(seededReport2.id);
        });

        it("throws error for non-existent report", async () => {
            await createTestData();
            const { req, res } = await mockLoggedOutSession();
            const input: FindByIdInput = { id: "non-existent-id" };

            await expect(async () => {
                await report.findOne({ input }, { req, res }, report_findOne);
            }).rejects.toThrow();
        });
    });

    describe("findMany", () => {
        it("returns all reports for authenticated user", async () => {
            const { testUsers, seededReport1, seededReport2 } = await createTestData();
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id,
            });
            const input: ReportSearchInput = { take: 10 };
            const expectedIds = [
                seededReport1.id,
                seededReport2.id,
            ];
            const result = await report.findMany({ input }, { req, res }, report_findMany);
            expect(result).not.toBeNull();
            expect(result.edges).toBeInstanceOf(Array);
            assertFindManyResultIds(expect, result, expectedIds);
        });

        it("filters reports by userId", async () => {
            const { testUsers, seededReport1 } = await createTestData();
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id,
            });
            const input: ReportSearchInput = { take: 10, userId: testUsers[0].id };
            const expectedIds = [
                seededReport1.id,
            ];
            const result = await report.findMany({ input }, { req, res }, report_findMany);
            assertFindManyResultIds(expect, result, expectedIds);
        });

        it("supports updatedTimeFrame filter", async () => {
            const { testUsers, seededReport1 } = await createTestData();
            // Make report1 old
            const old = new Date("2020-01-01");
            await DbProvider.get().report.update({ where: { id: seededReport1.id }, data: { updatedAt: old } });
            const input: ReportSearchInput = {
                take: 10,
                updatedTimeFrame: { after: new Date("2019-12-31"), before: new Date("2020-01-02") },
            };
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: testUsers[0].id });
            const result = await report.findMany({ input }, { req, res }, report_findMany);
            expect(result.edges).toHaveLength(1);
            expect(result.edges![0]!.node!.id).toBe(seededReport1.id);
        });

        it("allows anonymous access to all reports", async () => {
            await createTestData();
            const { req, res } = await mockLoggedOutSession();
            const input: ReportSearchInput = { take: 10 };
            const result = await report.findMany({ input }, { req, res }, report_findMany);
            expect(result.edges).toHaveLength(2);
        });
    });

    describe("createOne", () => {
        it("creates a new report for authenticated user", async () => {
            const { testUsers } = await createTestData();
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id,
            });

            // Create report input directly
            const input: ReportCreateInput = {
                id: generatePK().toString(),
                createdForType: ReportFor.User,
                createdForConnect: testUsers[1].id.toString(),
                language: "en",
                reason: "Test reason",
                details: "Test details",
            } as ReportCreateInput;

            const result = await report.createOne({ input }, { req, res }, report_createOne);
            expect(result.id).toBeDefined();
            expect(result.status).toBe(ReportStatus.Open);
            expect(result.reason).toBe("Test reason");
            expect(result.you?.canRespond).toBe(true);
        });

        it("does not allow duplicate open report on same object", async () => {
            const { testUsers } = await createTestData();
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id,
            });

            // Create report input directly
            const baseInput: ReportCreateInput = {
                id: generatePK().toString(),
                createdForType: ReportFor.User,
                createdForConnect: testUsers[1].id.toString(),
                language: "en",
                reason: "Reason A",
            };

            await report.createOne({ input: baseInput }, { req, res }, report_createOne);

            const dupInput: ReportCreateInput = {
                id: generatePK().toString(),
                createdForType: ReportFor.User,
                createdForConnect: testUsers[1].id.toString(),
                language: "en",
                reason: "Reason A",
            };

            await expect(async () => {
                await report.createOne({ input: dupInput }, { req, res }, report_createOne);
            }).rejects.toThrow(); // Duplicate report should fail
        });

        it("throws error when not authenticated", async () => {
            const { testUsers } = await createTestData();
            const { req, res } = await mockLoggedOutSession();

            // Create report input directly
            const input: ReportCreateInput = {
                id: generatePK().toString(),
                createdForType: ReportFor.User,
                createdForConnect: testUsers[0].id.toString(),
                language: "en",
                reason: "No auth",
            };

            await expect(async () => {
                await report.createOne({ input }, { req, res }, report_createOne);
            }).rejects.toThrow(); // Unauthenticated should fail
        });
    });

    describe("updateOne", () => {
        it("denies update for non-admin user", async () => {
            const { testUsers, seededReport1 } = await createTestData();
            // Use testUsers[1] to test non-creator access
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[1].id,
            });
            const input: ReportUpdateInput = { id: seededReport1.id, reason: "Updated" };

            await expect(async () => {
                await report.updateOne({ input }, { req, res }, report_updateOne);
            }).rejects.toThrow(); // Non-admin should fail
        });

        it("allows creator to update their own report", async () => {
            const { testUsers, seededReport1 } = await createTestData();
            // The first user created seededReport1
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id,
            });

            const input: ReportUpdateInput = { id: seededReport1.id, reason: "Updated by creator" };
            const result = await report.updateOne({ input }, { req, res }, report_updateOne);
            expect(result.reason).toBe("Updated by creator");
        });

        it("throws error when updating non-existent report", async () => {
            const { adminUser } = await createTestData();
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: adminUser.id,
            });

            const input: ReportUpdateInput = { id: "non-existent-id", reason: "No such" };

            await expect(async () => {
                await report.updateOne({ input }, { req, res }, report_updateOne);
            }).rejects.toThrow();
        });

        it("throws when not authenticated", async () => {
            const { seededReport1 } = await createTestData();
            const { req, res } = await mockLoggedOutSession();
            const input: ReportUpdateInput = { id: seededReport1.id, reason: "Should fail" };

            await expect(async () => {
                await report.updateOne({ input }, { req, res }, report_updateOne);
            }).rejects.toThrow();
        });
    });
}); 
