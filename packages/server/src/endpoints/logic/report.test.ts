import { type FindByIdInput, type ReportCreateInput, ReportFor, type ReportSearchInput, ReportStatus, type ReportUpdateInput } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
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
import { seedReports } from "../../__test/fixtures/db/reportFixtures.js";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
// Import validation fixtures for API input testing
import { reportTestDataFactory } from "@vrooli/shared/validation/models/index.js";

describe("EndpointsReport", () => {
    let testUsers: any[];
    let adminUser: any;
    let seededReport1: any;
    let seededReport2: any;

    beforeAll(() => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Reset Redis and database tables
        await CacheService.get().flushAll();
        await DbProvider.deleteAll();

        // Seed admin user and test users using database fixtures
        adminUser = await seedMockAdminUser();
        testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

        // Seed reports using database fixtures
        const reports = await seedReports(DbProvider.get(), {
            createdById: testUsers[0].id,
            count: 1,
            forObject: { id: testUsers[0].id, type: "User" },
            status: ReportStatus.Open,
            withDetails: true,
        });
        seededReport1 = reports[0];

        const reports2 = await seedReports(DbProvider.get(), {
            createdById: testUsers[1].id,
            count: 1,
            forObject: { id: testUsers[1].id, type: "User" },
            status: ReportStatus.Open,
            withDetails: true,
        });
        seededReport2 = reports2[0];
    });

    afterAll(async () => {
        // Clean up
        await CacheService.get().flushAll();
        await DbProvider.deleteAll();

        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("findOne", () => {
        it("returns a report by id for authenticated user", async () => {
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
            const { req, res } = await mockLoggedOutSession();
            const input: FindByIdInput = { id: seededReport2.id };
            const result = await report.findOne({ input }, { req, res }, report_findOne);
            expect(result).not.toBeNull();
            expect(result.id).toEqual(seededReport2.id);
        });

        it("throws error for non-existent report", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: FindByIdInput = { id: "non-existent-id" };

            await expect(async () => {
                await report.findOne({ input }, { req, res }, report_findOne);
            }).rejects.toThrow();
        });
    });

    describe("findMany", () => {
        it("returns all reports for authenticated user", async () => {
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
            const { req, res } = await mockLoggedOutSession();
            const input: ReportSearchInput = { take: 10 };
            const result = await report.findMany({ input }, { req, res }, report_findMany);
            expect(result.edges).toHaveLength(2);
        });
    });

    describe("createOne", () => {
        it("creates a new report for authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id,
            });

            // Use validation fixtures for API input
            const input: ReportCreateInput = reportTestDataFactory.createMinimal({
                createdForType: ReportFor.User,
                createdForConnect: testUsers[1].id,
                language: "en",
                reason: "Test reason",
                details: "Test details",
            });

            const result = await report.createOne({ input }, { req, res }, report_createOne);
            expect(result.id).toBeDefined();
            expect(result.status).toBe(ReportStatus.Open);
            expect(result.reason).toBe("Test reason");
            expect(result.you?.canRespond).toBe(true);
        });

        it("does not allow duplicate open report on same object", async () => {
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id,
            });

            // Use validation fixtures for API input
            const baseInput: ReportCreateInput = reportTestDataFactory.createMinimal({
                createdForType: ReportFor.User,
                createdForConnect: testUsers[1].id,
                language: "en",
                reason: "Reason A",
            });

            await report.createOne({ input: baseInput }, { req, res }, report_createOne);

            const dupInput: ReportCreateInput = reportTestDataFactory.createMinimal({
                createdForType: ReportFor.User,
                createdForConnect: testUsers[1].id,
                language: "en",
                reason: "Reason A",
            });

            await expect(async () => {
                await report.createOne({ input: dupInput }, { req, res }, report_createOne);
            }).rejects.toThrow(); // Duplicate report should fail
        });

        it("throws error when not authenticated", async () => {
            const { req, res } = await mockLoggedOutSession();

            // Use validation fixtures for API input
            const input: ReportCreateInput = reportTestDataFactory.createMinimal({
                createdForType: ReportFor.User,
                createdForConnect: testUsers[0].id,
                language: "en",
                reason: "No auth",
            });

            await expect(async () => {
                await report.createOne({ input }, { req, res }, report_createOne);
            }).rejects.toThrow(); // Unauthenticated should fail
        });
    });

    describe("updateOne", () => {
        it("denies update for non-admin user", async () => {
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id,
            });
            const input: ReportUpdateInput = { id: seededReport1.id, reason: "Updated" };

            await expect(async () => {
                await report.updateOne({ input }, { req, res }, report_updateOne);
            }).rejects.toThrow(); // Non-admin should fail
        });

        it("allows admin to update closed report", async () => {
            // Close the second report
            await DbProvider.get().report.update({
                where: { id: seededReport2.id },
                data: { status: ReportStatus.ClosedHidden },
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: adminUser.id,
            });

            const input: ReportUpdateInput = { id: seededReport2.id, reason: "Admin updated" };
            const result = await report.updateOne({ input }, { req, res }, report_updateOne);
            expect(result.reason).toBe("Admin updated");
        });

        it("throws error when updating non-existent report", async () => {
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
            const { req, res } = await mockLoggedOutSession();
            const input: ReportUpdateInput = { id: seededReport1.id, reason: "Should fail" };

            await expect(async () => {
                await report.updateOne({ input }, { req, res }, report_updateOne);
            }).rejects.toThrow();
        });
    });
}); 
