// Tests for the ReportResponse endpoint (findOne, findMany, createOne, updateOne)
import { FindByIdInput, ReportResponseCreateInput, ReportResponseSearchInput, ReportResponseUpdateInput, ReportStatus, ReportSuggestedAction, SEEDED_IDS, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockReadPublicPermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { reportResponse_createOne } from "../generated/reportResponse_createOne.js";
import { reportResponse_findMany } from "../generated/reportResponse_findMany.js";
import { reportResponse_findOne } from "../generated/reportResponse_findOne.js";
import { reportResponse_updateOne } from "../generated/reportResponse_updateOne.js";
import { reportResponse } from "./reportResponse.js";

// Test users
const user1Id = uuid();
const adminId = SEEDED_IDS.User.Admin; // Use seeded admin ID
// Test reports and responses
let report1: any;
let response1: any;

describe("EndpointsReportResponse", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        // stub logger
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async () => {
        // Reset DBs
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Seed users (user1 and admin)
        await DbProvider.get().user.create({ data: { id: user1Id, name: "User 1", handle: "user1", status: "Unlocked", isBot: false, isBotDepictingPerson: false, isPrivate: false, auths: { create: [{ provider: "Password", hashed_password: "hash" }] } } });
        // Assume admin user exists or create if needed
        await DbProvider.get().user.upsert({
            where: { id: adminId },
            update: { name: "Admin User", handle: "admin", status: "Unlocked" },
            create: { id: adminId, name: "Admin User", handle: "admin", status: "Unlocked", isBot: false, isBotDepictingPerson: false, isPrivate: false, auths: { create: [{ provider: "Password", hashed_password: "hash" }] } },
        });

        // Seed a report created by user1 targeting themselves
        report1 = await DbProvider.get().report.create({
            data: {
                id: uuid(), language: "en", reason: "Test Report", details: "Details", status: ReportStatus.Open,
                createdBy: { connect: { id: user1Id } }, user: { connect: { id: user1Id } },
            },
        });

        // Seed a response from the admin
        response1 = await DbProvider.get().report_response.create({
            data: {
                id: uuid(), details: "Admin response", report: { connect: { id: report1.id } },
                actionSuggested: ReportSuggestedAction.NonIssue, // Use enum
                createdBy: { connect: { id: adminId } }, // Explicitly connect creator for seeding
            },
        });
    });

    after(async () => {
        // Cleanup and restore
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        it("admin can find a report response", async () => {
            const adminUser = { ...loggedInUserNoPremiumData(), id: adminId };
            const { req, res } = await mockAuthenticatedSession(adminUser);
            const input: FindByIdInput = { id: response1.id };
            const result = await reportResponse.findOne({ input }, { req, res }, reportResponse_findOne);
            expect(result.id).to.equal(response1.id);
            expect(result.details).to.equal("Admin response");
        });

        it("API key with public read can find response", async () => {
            const perms = mockReadPublicPermissions();
            const token = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(token, perms, loggedInUserNoPremiumData);
            const input: FindByIdInput = { id: response1.id };
            const result = await reportResponse.findOne({ input }, { req, res }, reportResponse_findOne);
            expect(result.id).to.equal(response1.id);
        });

        it("throws for non-admin user", async () => {
            const user = { ...loggedInUserNoPremiumData(), id: user1Id };
            const { req, res } = await mockAuthenticatedSession(user);
            const input: FindByIdInput = { id: response1.id };
            try {
                await reportResponse.findOne({ input }, { req, res }, reportResponse_findOne);
                expect.fail("Expected permission error");
            } catch (err) {
                // expected
            }
        });
    });

    describe("findMany", () => {
        it("admin can find report responses", async () => {
            const adminUser = { ...loggedInUserNoPremiumData(), id: adminId };
            const { req, res } = await mockAuthenticatedSession(adminUser);
            const input: ReportResponseSearchInput = { take: 10, reportId: report1.id };
            const result = await reportResponse.findMany({ input }, { req, res }, reportResponse_findMany);
            const ids = result.edges!.map(e => e!.node!.id);
            expect(ids).to.include(response1.id);
        });

        it("throws for non-admin user", async () => {
            const user = { ...loggedInUserNoPremiumData(), id: user1Id };
            const { req, res } = await mockAuthenticatedSession(user);
            const input: ReportResponseSearchInput = { take: 10, reportId: report1.id };
            try {
                await reportResponse.findMany({ input }, { req, res }, reportResponse_findMany);
                expect.fail("Expected permission error");
            } catch (err) {
                // expected
            }
        });
    });

    describe("createOne", () => {
        it("admin can create a report response", async () => {
            const adminUser = { ...loggedInUserNoPremiumData(), id: adminId };
            const { req, res } = await mockAuthenticatedSession(adminUser);
            const newResponseId = uuid();
            const input: ReportResponseCreateInput = {
                id: newResponseId,
                details: "New admin response",
                reportConnect: report1.id,
                actionSuggested: ReportSuggestedAction.NonIssue, // Use enum
            };
            const result = await reportResponse.createOne({ input }, { req, res }, reportResponse_createOne);
            expect(result.id).to.equal(newResponseId);
            expect(result.details).to.equal("New admin response");
        });

        it("throws for non-admin user", async () => {
            const user = { ...loggedInUserNoPremiumData(), id: user1Id };
            const { req, res } = await mockAuthenticatedSession(user);
            const input: ReportResponseCreateInput = { id: uuid(), details: "User response", reportConnect: report1.id, actionSuggested: ReportSuggestedAction.NonIssue }; // Use enum
            try {
                await reportResponse.createOne({ input }, { req, res }, reportResponse_createOne);
                expect.fail("Expected permission error");
            } catch (err) {
                // expected
            }
        });
    });

    describe("updateOne", () => {
        it("admin can update a report response", async () => {
            const adminUser = { ...loggedInUserNoPremiumData(), id: adminId };
            const { req, res } = await mockAuthenticatedSession(adminUser);
            const input: ReportResponseUpdateInput = { id: response1.id, details: "Updated admin response" };
            const result = await reportResponse.updateOne({ input }, { req, res }, reportResponse_updateOne);
            expect(result.id).to.equal(response1.id);
            expect(result.details).to.equal("Updated admin response");
        });

        it("throws for non-admin user", async () => {
            const user = { ...loggedInUserNoPremiumData(), id: user1Id };
            const { req, res } = await mockAuthenticatedSession(user);
            const input: ReportResponseUpdateInput = { id: response1.id, details: "User updated response" };
            try {
                await reportResponse.updateOne({ input }, { req, res }, reportResponse_updateOne);
                expect.fail("Expected permission error");
            } catch (err) {
                // expected
            }
        });
    });
}); 
