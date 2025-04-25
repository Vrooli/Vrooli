// Tests for the Role endpoint (findOne, findMany, createOne, updateOne)
import { FindByIdInput, RoleCreateInput, RoleSearchInput, RoleUpdateInput, SEEDED_IDS, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockReadPublicPermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { role_createOne } from "../generated/role_createOne.js";
import { role_findMany } from "../generated/role_findMany.js";
import { role_findOne } from "../generated/role_findOne.js";
import { role_updateOne } from "../generated/role_updateOne.js";
import { role } from "./role.js";

// Test users and roles
const user1Id = uuid();
const adminId = SEEDED_IDS.User.Admin;
let role1: any;
let team1Id = uuid();

describe("EndpointsRole", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        // stub logger
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async () => {
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Seed users
        await DbProvider.get().user.create({ data: { id: user1Id, name: "User 1", handle: "user1", status: "Unlocked", isBot: false, isBotDepictingPerson: false, isPrivate: false, auths: { create: [{ provider: "Password", hashed_password: "hash" }] } } });
        await DbProvider.get().user.upsert({
            where: { id: adminId }, update: {}, create: { id: adminId, name: "Admin", handle: "admin", status: "Unlocked", isBot: false, isBotDepictingPerson: false, isPrivate: false, auths: { create: [{ provider: "Password", hashed_password: "hash" }] } }
        });

        // Seed team
        await DbProvider.get().team.create({
            data: {
                id: team1Id, handle: "team1", isPrivate: false, isOpenToNewMembers: true, permissions: "",
                translations: { create: { language: "en", name: "Team 1" } }
            }
        });

        // Seed a role
        role1 = await DbProvider.get().role.create({
            data: {
                id: uuid(), name: "member", permissions: "read", team: { connect: { id: team1Id } },
                translations: { create: [{ id: uuid(), language: "en", name: "Member", description: "Default member role" }] }
            }
        });
    });

    after(async () => {
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        it("admin can find a role", async () => {
            const adminUser = { ...loggedInUserNoPremiumData, id: adminId };
            const { req, res } = await mockAuthenticatedSession(adminUser);
            const input: FindByIdInput = { id: role1.id };
            const result = await role.findOne({ input }, { req, res }, role_findOne);
            expect(result.id).to.equal(role1.id);
            expect(result.name).to.equal("member");
        });

        it("API key with public read can find role", async () => {
            const perms = mockReadPublicPermissions();
            const token = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(token, perms, loggedInUserNoPremiumData);
            const input: FindByIdInput = { id: role1.id };
            const result = await role.findOne({ input }, { req, res }, role_findOne);
            expect(result.id).to.equal(role1.id);
        });

        it("regular user can find role", async () => {
            const user = { ...loggedInUserNoPremiumData, id: user1Id };
            const { req, res } = await mockAuthenticatedSession(user);
            const input: FindByIdInput = { id: role1.id };
            const result = await role.findOne({ input }, { req, res }, role_findOne);
            expect(result.id).to.equal(role1.id);
        });
    });

    describe("findMany", () => {
        it("admin can find roles", async () => {
            const adminUser = { ...loggedInUserNoPremiumData, id: adminId };
            const { req, res } = await mockAuthenticatedSession(adminUser);
            const input: RoleSearchInput = { take: 10, teamId: team1Id };
            const result = await role.findMany({ input }, { req, res }, role_findMany);
            const ids = result.edges!.map(e => e!.node!.id);
            expect(ids).to.include(role1.id);
        });

        it("regular user can find roles", async () => {
            const user = { ...loggedInUserNoPremiumData, id: user1Id };
            const { req, res } = await mockAuthenticatedSession(user);
            const input: RoleSearchInput = { take: 10, teamId: team1Id };
            const result = await role.findMany({ input }, { req, res }, role_findMany);
            const ids = result.edges!.map(e => e!.node!.id);
            expect(ids).to.include(role1.id);
        });
    });

    describe("createOne", () => {
        it("admin can create a role", async () => {
            const adminUser = { ...loggedInUserNoPremiumData, id: adminId };
            const { req, res } = await mockAuthenticatedSession(adminUser);
            const newRoleId = uuid();
            const input: RoleCreateInput = {
                id: newRoleId,
                name: "admin",
                permissions: "all",
                teamConnect: team1Id,
                translationsCreate: [{ id: uuid(), language: "en", description: "Administrator role" }]
            };
            const result = await role.createOne({ input }, { req, res }, role_createOne);
            expect(result.id).to.equal(newRoleId);
            expect(result.name).to.equal("admin");
        });

        it("throws for non-admin user", async () => {
            const user = { ...loggedInUserNoPremiumData, id: user1Id };
            const { req, res } = await mockAuthenticatedSession(user);
            const input: RoleCreateInput = {
                id: uuid(), name: "hacker", permissions: "none", teamConnect: team1Id,
                translationsCreate: [{ id: uuid(), language: "en", description: "Hacker role (should fail)" }]
            };
            try {
                await role.createOne({ input }, { req, res }, role_createOne);
                expect.fail("Expected permission error");
            } catch (err) {
                // expected
            }
        });
    });

    describe("updateOne", () => {
        it("admin can update a role", async () => {
            const adminUser = { ...loggedInUserNoPremiumData, id: adminId };
            const { req, res } = await mockAuthenticatedSession(adminUser);
            const input: RoleUpdateInput = { id: role1.id, permissions: "read,write" };
            const result = await role.updateOne({ input }, { req, res }, role_updateOne);
            expect(result.id).to.equal(role1.id);
            expect(result.permissions).to.equal("read,write");
        });

        it("throws for non-admin user", async () => {
            const user = { ...loggedInUserNoPremiumData, id: user1Id };
            const { req, res } = await mockAuthenticatedSession(user);
            const input: RoleUpdateInput = { id: role1.id, permissions: "hacked" };
            try {
                await role.updateOne({ input }, { req, res }, role_updateOne);
                expect.fail("Expected permission error");
            } catch (err) {
                // expected
            }
        });
    });
}); 