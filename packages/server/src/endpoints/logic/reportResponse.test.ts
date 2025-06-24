import { type FindByIdInput, type ReportResponseCreateInput, type ReportResponseSearchInput, type ReportResponseUpdateInput, ReportStatus, ReportSuggestedAction, generatePK } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockReadPublicPermissions, seedMockAdminUser } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { reportResponse_createOne } from "../generated/reportResponse_createOne.js";
import { reportResponse_findMany } from "../generated/reportResponse_findMany.js";
import { reportResponse_findOne } from "../generated/reportResponse_findOne.js";
import { reportResponse_updateOne } from "../generated/reportResponse_updateOne.js";
import { reportResponse } from "./reportResponse.js";
// Import database fixtures for seeding
import { ReportDbFactory, seedTestReports } from "../../__test/fixtures/db/reportFixtures.js";
import { ReportResponseDbFactory, seedTestReportResponses } from "../../__test/fixtures/db/reportResponseFixtures.js";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
// Import validation fixtures for API input testing
import { reportResponseTestDataFactory } from "@vrooli/shared";

describe("EndpointsReportResponse", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warning").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Clean up tables used in tests
        const prisma = DbProvider.get();
        await prisma.report_response.deleteMany();
        await prisma.report.deleteMany();
        await prisma.user.deleteMany();
        // Clear Redis cache
        await CacheService.get().flushAll();
    });

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    // Helper function to create test data
    const createTestData = async () => {
        // Create test users
        const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });
        
        // Ensure admin user exists for report responses
        const adminUser = await seedMockAdminUser();
        
        // Create test reports
        const reports = await seedTestReports(DbProvider.get(), {
            createdById: testUsers[0].id,
            targetUserId: testUsers[1].id,
            count: 2,
            withResponses: false,
        });
        
        // Create test report responses
        const reportResponses = await seedTestReportResponses(DbProvider.get(), {
            reportId: reports[0].id,
            createdById: adminUser.id,
            count: 2,
        });
        
        return { testUsers, adminUser, reports, reportResponses };
    };

    describe("findOne", () => {
        it("admin can find a report response", async () => {
            const { adminUser, reportResponses } = await createTestData();
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: adminUser.id,
            });
            
            const input: FindByIdInput = { id: reportResponses[0].id };
            const result = await reportResponse.findOne({ input }, { req, res }, reportResponse_findOne);
            
            expect(result).not.toBeNull();
            expect(result.id).toEqual(reportResponses[0].id);
            expect(result.details).toBeDefined();
        });

        it("API key with public read can find response", async () => {
            const { reportResponses } = await createTestData();
            const perms = mockReadPublicPermissions();
            const token = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(token, perms, loggedInUserNoPremiumData());
            
            const input: FindByIdInput = { id: reportResponses[0].id };
            const result = await reportResponse.findOne({ input }, { req, res }, reportResponse_findOne);
            
            expect(result).not.toBeNull();
            expect(result.id).toEqual(reportResponses[0].id);
        });

        it("throws for non-admin user", async () => {
            const { testUsers, reportResponses } = await createTestData();
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id,
            });
            
            const input: FindByIdInput = { id: reportResponses[0].id };
            
            await expect(async () => {
                await reportResponse.findOne({ input }, { req, res }, reportResponse_findOne);
            }).rejects.toThrow();
        });
    });

    describe("findMany", () => {
        it("admin can find report responses", async () => {
            const { adminUser, reports, reportResponses } = await createTestData();
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: adminUser.id,
            });
            
            const input: ReportResponseSearchInput = { 
                take: 10, 
                reportId: reports[0].id 
            };
            const result = await reportResponse.findMany({ input }, { req, res }, reportResponse_findMany);
            
            expect(result.edges).toBeDefined();
            const ids = result.edges!.map(e => e!.node!.id);
            expect(ids).toContain(reportResponses[0].id);
        });

        it("throws for non-admin user", async () => {
            const { testUsers, reports } = await createTestData();
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id,
            });
            
            const input: ReportResponseSearchInput = { 
                take: 10, 
                reportId: reports[0].id 
            };
            
            await expect(async () => {
                await reportResponse.findMany({ input }, { req, res }, reportResponse_findMany);
            }).rejects.toThrow();
        });
    });

    describe("createOne", () => {
        it("admin can create a report response", async () => {
            const { adminUser, reports } = await createTestData();
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: adminUser.id,
            });
            
            const input: ReportResponseCreateInput = reportResponseTestDataFactory.createMinimal({
                id: generatePK(),
                details: "New admin response",
                reportConnect: reports[0].id,
                actionSuggested: ReportSuggestedAction.NonIssue,
            });
            
            const result = await reportResponse.createOne({ input }, { req, res }, reportResponse_createOne);
            
            expect(result).not.toBeNull();
            expect(result.id).toBe(input.id);
            expect(result.details).toBe(input.details);
            expect(result.actionSuggested).toBe(input.actionSuggested);
        });

        it("throws for non-admin user", async () => {
            const { testUsers, reports } = await createTestData();
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id,
            });
            
            const input: ReportResponseCreateInput = reportResponseTestDataFactory.createMinimal({
                details: "User response",
                reportConnect: reports[0].id,
                actionSuggested: ReportSuggestedAction.NonIssue,
            });
            
            await expect(async () => {
                await reportResponse.createOne({ input }, { req, res }, reportResponse_createOne);
            }).rejects.toThrow();
        });

        it("validates input requirements", async () => {
            const { adminUser, reports } = await createTestData();
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: adminUser.id,
            });
            
            // Test missing required fields
            const invalidInput = {
                // details is missing
                reportConnect: reports[0].id,
                actionSuggested: ReportSuggestedAction.NonIssue,
            } as ReportResponseCreateInput;
            
            await expect(async () => {
                await reportResponse.createOne({ input: invalidInput }, { req, res }, reportResponse_createOne);
            }).rejects.toThrow();
        });
    });

    describe("updateOne", () => {
        it("admin can update a report response", async () => {
            const { adminUser, reportResponses } = await createTestData();
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: adminUser.id,
            });
            
            const input: ReportResponseUpdateInput = {
                id: reportResponses[0].id,
                details: "Updated admin response",
                actionSuggested: ReportSuggestedAction.Resolved,
            };
            
            const result = await reportResponse.updateOne({ input }, { req, res }, reportResponse_updateOne);
            
            expect(result).not.toBeNull();
            expect(result.id).toBe(input.id);
            expect(result.details).toBe(input.details);
            expect(result.actionSuggested).toBe(input.actionSuggested);
        });

        it("throws for non-admin user", async () => {
            const { testUsers, reportResponses } = await createTestData();
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: testUsers[0].id,
            });
            
            const input: ReportResponseUpdateInput = {
                id: reportResponses[0].id,
                details: "User updated response",
            };
            
            await expect(async () => {
                await reportResponse.updateOne({ input }, { req, res }, reportResponse_updateOne);
            }).rejects.toThrow();
        });

        it("throws for non-existent report response", async () => {
            const { adminUser } = await createTestData();
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: adminUser.id,
            });
            
            const input: ReportResponseUpdateInput = {
                id: generatePK(), // Non-existent ID
                details: "Updated response",
            };
            
            await expect(async () => {
                await reportResponse.updateOne({ input }, { req, res }, reportResponse_updateOne);
            }).rejects.toThrow();
        });
    });
});