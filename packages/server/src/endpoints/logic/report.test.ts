import { FindByIdInput, generatePK, generatePublicId, ReportCreateInput, ReportFor, ReportSearchInput, ReportStatus, ReportUpdateInput } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { assertFindManyResultIds } from "../../__test/helpers.js";
import { defaultPublicUserData, loggedInUserNoPremiumData, mockAuthenticatedSession, mockLoggedOutSession, seedMockAdminUser } from "../../__test/session.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { report_createOne } from "../generated/report_createOne.js";
import { report_findMany } from "../generated/report_findMany.js";
import { report_findOne } from "../generated/report_findOne.js";
import { report_updateOne } from "../generated/report_updateOne.js";
import { report } from "./report.js";

// Test users
let adminId: string;
const user1Id = generatePK();
const user2Id = generatePK();

// Holds seeded report records
let seededReport1: any;
let seededReport2: any;

describe("EndpointsReport", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        // Stub logger to avoid noise
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async () => {
        // Clear databases
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        const admin = await seedMockAdminUser();
        adminId = admin.id.toString();

        // Create two users
        await DbProvider.get().user.create({
            data: {
                ...defaultPublicUserData(),
                id: user1Id,
                name: "Test User 1",
            },
        });
        await DbProvider.get().user.create({
            data: {
                ...defaultPublicUserData(),
                id: user2Id,
                name: "Test User 2",
            },
        });

        // Seed two reports: one by each user, targeting themselves
        const data1 = {
            id: generatePK().toString(),
            publicId: generatePublicId(),
            language: "en",
            reason: "Reason 1",
            details: "Details 1",
            status: ReportStatus.Open,
            createdBy: { connect: { id: user1Id } },
            user: { connect: { id: user1Id } }, // createdForType: User
        };
        const data2 = {
            id: generatePK().toString(),
            publicId: generatePublicId(),
            language: "en",
            reason: "Reason 2",
            details: "Details 2",
            status: ReportStatus.Open,
            createdBy: { connect: { id: user2Id } },
            user: { connect: { id: user2Id } },
        };
        seededReport1 = await DbProvider.get().report.create({ data: data1 });
        seededReport2 = await DbProvider.get().report.create({ data: data2 });
    });

    after(async () => {
        // Cleanup
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        it("returns a report by id for authenticated user", async () => {
            const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
            const { req, res } = await mockAuthenticatedSession(testUser);
            const input: FindByIdInput = { id: seededReport1.id };
            const result = await report.findOne({ input }, { req, res }, report_findOne);
            expect(result).to.not.be.null;
            expect(result.id).to.equal(seededReport1.id);
            expect(result.reason).to.equal(seededReport1.reason);
        });

        it("returns a report by id when not authenticated", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: FindByIdInput = { id: seededReport2.id };
            const result = await report.findOne({ input }, { req, res }, report_findOne);
            expect(result).to.not.be.null;
            expect(result.id).to.equal(seededReport2.id);
        });

        it("throws error for non-existent report", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: FindByIdInput = { id: generatePK().toString() };
            try {
                await report.findOne({ input }, { req, res }, report_findOne);
                expect.fail("Expected error for non-existent report");
            } catch (error) {
                // Error expected
            }
        });
    });

    describe("findMany", () => {
        it("returns all reports for authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
            const input: ReportSearchInput = { take: 10 };
            const expectedIds = [
                seededReport1.id,
                seededReport2.id,
            ];
            const result = await report.findMany({ input }, { req, res }, report_findMany);
            expect(result).to.not.be.null;
            expect(result).to.have.property("edges").that.is.an("array");
            assertFindManyResultIds(expect, result, expectedIds);
        });

        it("filters reports by userId", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
            const input: ReportSearchInput = { take: 10, userId: user1Id.toString() };
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
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: user1Id });
            const result = await report.findMany({ input }, { req, res }, report_findMany);
            expect(result.edges).to.have.lengthOf(1);
            expect(result.edges![0]!.node!.id).to.equal(seededReport1.id);
        });

        it("allows anonymous access to all reports", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: ReportSearchInput = { take: 10 };
            const result = await report.findMany({ input }, { req, res }, report_findMany);
            expect(result.edges).to.have.lengthOf(2);
        });
    });

    describe("createOne", () => {
        it("creates a new report for authenticated user", async () => {
            const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
            const { req, res } = await mockAuthenticatedSession(testUser);
            const newId = generatePK();
            const input: ReportCreateInput = {
                id: newId.toString(),
                createdForType: ReportFor.User,
                createdForConnect: user2Id.toString(),
                language: "en",
                reason: "Test reason",
                details: "Test details",
            };
            const result = await report.createOne({ input }, { req, res }, report_createOne);
            console.log("[report fail fix] result.you:", result.you, "status:", result.status);
            expect(result.id).to.equal(newId);
            expect(result.status).to.equal(ReportStatus.Open);
            expect(result.you?.canRespond).to.be.true;
        });

        it("does not allow duplicate open report on same object", async () => {
            const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
            const { req, res } = await mockAuthenticatedSession(testUser);
            const id1 = generatePK();
            const baseInput = {
                id: id1.toString(),
                createdForType: ReportFor.User,
                createdForConnect: user2Id.toString(),
                language: "en",
                reason: "Reason A",
            };
            await report.createOne({ input: baseInput }, { req, res }, report_createOne);
            const dupInput = { ...baseInput, id: generatePK().toString() };
            try {
                await report.createOne({ input: dupInput }, { req, res }, report_createOne);
                expect.fail("Expected error for duplicate open report");
            } catch (error) {
                // Error expected
            }
        });

        it("throws error when not authenticated", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: ReportCreateInput = {
                id: generatePK().toString(),
                createdForType: ReportFor.User,
                createdForConnect: user1Id.toString(),
                language: "en",
                reason: "No auth",
            };
            try {
                await report.createOne({ input }, { req, res }, report_createOne);
                expect.fail("Expected error for unauthenticated create");
            } catch (error) {
                // Error expected
            }
        });
    });

    describe("updateOne", () => {
        it("denies update for non-admin user", async () => {
            const testUser = { ...loggedInUserNoPremiumData(), id: user1Id };
            const { req, res } = await mockAuthenticatedSession(testUser);
            const input: ReportUpdateInput = { id: seededReport1.id, reason: "Updated" };
            try {
                await report.updateOne({ input }, { req, res }, report_updateOne);
                expect.fail("Expected error for non-admin update");
            } catch (error) {
                // Error expected
            }
        });

        it("allows admin to update closed report", async () => {
            // Close the second report
            await DbProvider.get().report.update({ where: { id: seededReport2.id }, data: { status: ReportStatus.ClosedHidden } });
            // Seed admin user and auth
            const adminUser = { ...loggedInUserNoPremiumData(), id: adminId };
            await DbProvider.get().user_auth.upsert({
                where: { id: adminUser.auths[0].id },
                create: {
                    id: adminUser.auths[0].id,
                    provider: "Password",
                    hashed_password: "dummy-hash",
                    user: { connect: { id: BigInt(adminUser.id) } },
                },
                update: { hashed_password: "dummy-hash" },
            });
            const { req, res } = await mockAuthenticatedSession(adminUser);
            const input: ReportUpdateInput = { id: seededReport2.id, reason: "Admin updated" };
            const result = await report.updateOne({ input }, { req, res }, report_updateOne);
            expect(result.reason).to.equal("Admin updated");
        });

        it("throws error when updating non-existent report", async () => {
            // Seed admin user and auth
            const adminUser = { ...loggedInUserNoPremiumData(), id: adminId };
            await DbProvider.get().user_auth.upsert({
                where: { id: adminUser.auths[0].id },
                create: {
                    id: adminUser.auths[0].id,
                    provider: "Password",
                    hashed_password: "dummy-hash",
                    user: { connect: { id: BigInt(adminUser.id) } },
                },
                update: { hashed_password: "dummy-hash" },
            });
            const { req, res } = await mockAuthenticatedSession(adminUser);
            const input: ReportUpdateInput = { id: generatePK().toString(), reason: "No such" };
            try {
                await report.updateOne({ input }, { req, res }, report_updateOne);
                expect.fail("Expected error for non-existent update");
            } catch (error) {
                // Error expected
            }
        });

        it("throws when not authenticated", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: ReportUpdateInput = { id: seededReport1.id, reason: "Should fail" };
            try {
                await report.updateOne({ input }, { req, res }, report_updateOne);
                expect.fail("Expected error for unauthenticated update");
            } catch (error) {
                // Error expected
            }
        });
    });
}); 
