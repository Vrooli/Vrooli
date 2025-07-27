import { CopyType, DeleteType, generatePK, generatePublicId, type CopyInput, type DeleteAccountInput, type DeleteAllInput, type DeleteManyInput, type DeleteOneInput } from "@vrooli/shared";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { assertRequiresApiKeyWritePermissions, assertRequiresAuth } from "../../__test/authTestUtils.js";
import { expectCustomErrorAsync } from "../../__test/errorTestUtils.js";
import { mockApiSession, mockAuthenticatedSession, mockWriteAuthPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { actions_copy } from "../generated/actions_copy.js";
import { actions_deleteAccount } from "../generated/actions_deleteAccount.js";
import { actions_deleteAll } from "../generated/actions_deleteAll.js";
import { actions_deleteOne } from "../generated/actions_deleteOne.js";
import { actions } from "./actions.js";
// Import database fixtures
import { AUTH_PROVIDERS } from "@vrooli/shared";
import { createResourceDbFactory } from "../../__test/fixtures/db/index.js";
import { createRunDbFactory } from "../../__test/fixtures/db/RunDbFactory.js";
import { TeamDbFactory } from "../../__test/fixtures/db/teamFixtures.js";
import { seedTestUsers, UserDbFactory } from "../../__test/fixtures/db/userFixtures.js";
import { cleanupGroups } from "../../__test/helpers/testCleanupHelpers.js";
import { validateCleanup } from "../../__test/helpers/testValidation.js";
import { PasswordAuthService } from "../../auth/email.js";

describe("EndpointsActions", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warning").mockImplementation(() => logger);
    });

    afterEach(async () => {
        // Perform cleanup using dependency-ordered cleanup helpers
        await cleanupGroups.userAuth(DbProvider.get());

        // Validate cleanup to detect any missed records
        const orphans = await validateCleanup(DbProvider.get(), {
            tables: ["user", "user_auth", "email", "phone", "push_device", "session"],
            logOrphans: true,
        });
        if (orphans.length > 0) {
            console.warn("Test cleanup incomplete:", orphans);
        }
    });

    beforeEach(async () => {
        // Clean up using dependency-ordered cleanup helpers
        await cleanupGroups.userAuth(DbProvider.get());
        // Clear Redis cache to reset rate limiting
        await CacheService.get().flushAll();
    });

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("copy", () => {
        describe("authentication", () => {
            it("not logged in", async () => {
                await assertRequiresAuth(
                    actions.copy,
                    { objectType: CopyType.Resource, id: generatePK().toString(), intendToPullRequest: false },
                    actions_copy,
                );
            });

            it("API key - no write permissions", async () => {
                await assertRequiresApiKeyWritePermissions(
                    actions.copy,
                    { objectType: CopyType.Resource, id: generatePK().toString(), intendToPullRequest: false },
                    actions_copy,
                );
            });
        });

        describe("valid", () => {
            it("copies own note", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                // Create factory instance for note resources
                const resourceFactory = createResourceDbFactory(DbProvider.get());

                // Create a note to copy
                const { resource: originalNote, versions: _versions } = await resourceFactory.createWithVersions(1, {
                    resourceType: "Note",
                    ownedByUser: { connect: { id: testUser[0].id } },
                    isPrivate: false,
                });

                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: CopyInput = {
                    objectType: CopyType.Resource,
                    id: originalNote.id.toString(),
                    intendToPullRequest: false,
                };

                const result = await actions.copy({ input }, { req, res }, actions_copy);

                expect(result).toBeDefined();
                expect(result.__typename).toBe("CopyResult");
                expect(result.resourceVersion).toBeDefined();
                expect(result.resourceVersion?.id).not.toBe(originalNote.id.toString());
                expect(result.resourceVersion?.isPrivate).toBe(originalNote.isPrivate);

                // Verify copy was created in database
                const copiedNote = await DbProvider.get().resource.findUnique({
                    where: { id: BigInt(result.resourceVersion!.id) },
                    include: { versions: true },
                });
                expect(copiedNote).toBeDefined();
                expect(copiedNote?.resourceType).toBe("Note");
                expect(copiedNote?.versions[0].versionLabel).toContain("Copy of");
            });

            it("copies own project", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                // Create factory instance for project resources
                const resourceFactory = createResourceDbFactory(DbProvider.get());

                // Create a project to copy
                const { resource: originalProject, versions: _versions } = await resourceFactory.createWithVersions(1, {
                    resourceType: "Project",
                    ownedByUser: { connect: { id: testUser[0].id } },
                    isPrivate: false,
                });

                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: CopyInput = {
                    objectType: CopyType.Resource,
                    id: originalProject.id.toString(),
                    intendToPullRequest: false,
                };

                const result = await actions.copy({ input }, { req, res }, actions_copy);

                expect(result).toBeDefined();
                expect(result.__typename).toBe("CopyResult");
                expect(result.resourceVersion).toBeDefined();
                expect(result.resourceVersion?.id).not.toBe(originalProject.id.toString());
                expect(result.resourceVersion?.id).not.toBe(originalProject.id.toString());
            });

            it("copies team project as team member", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                // Create team with user as member
                const teamFactory = new TeamDbFactory();
                const team = await teamFactory.createMinimal({
                    ownedByUser: { connect: { id: testUser[0].id } },
                });

                // Create team project
                const resourceFactory = createResourceDbFactory(DbProvider.get());
                const { resource: teamProject, versions: _versions } = await resourceFactory.createWithVersions(1, {
                    resourceType: "Project",
                    ownedByUser: { connect: { id: testUser[0].id } },
                    ownedByTeam: { connect: { id: team.id } },
                    isPrivate: false,
                });

                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: CopyInput = {
                    objectType: CopyType.Resource,
                    id: teamProject.id.toString(),
                    intendToPullRequest: false,
                };

                const result = await actions.copy({ input }, { req, res }, actions_copy);

                expect(result).toBeDefined();
                expect(result.resourceVersion).toBeDefined();
                expect(result.resourceVersion?.id).not.toBe(teamProject.id.toString());
            });
        });

        describe("invalid", () => {
            it("cannot copy private object from another user", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

                // Create factory instance for note resources
                const resourceFactory = createResourceDbFactory(DbProvider.get());

                // Create private note by user 1
                const { resource: privateNote } = await resourceFactory.createWithVersions(1, {
                    resourceType: "Note",
                    ownedByUser: { connect: { id: testUsers[0].id } },
                    isPrivate: true,
                });

                // Try to copy as user 2
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[1].id.toString(),
                });

                const input: CopyInput = {
                    objectType: CopyType.Resource,
                    id: privateNote.id.toString(),
                    intendToPullRequest: false,
                };

                await expectCustomErrorAsync(
                    actions.copy({ input }, { req, res }, actions_copy),
                    "Unauthorized",
                    "0323",
                );
            });

            it("cannot copy non-existent object", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: CopyInput = {
                    objectType: CopyType.Resource,
                    id: generatePK().toString(),
                    intendToPullRequest: false,
                };

                await expectCustomErrorAsync(
                    actions.copy({ input }, { req, res }, actions_copy),
                    "Unauthorized",
                    "0323",
                );
            });
        });
    });

    describe("deleteOne", () => {
        describe("authentication", () => {
            it("not logged in", async () => {
                await assertRequiresAuth(
                    actions.deleteOne,
                    { objectType: DeleteType.Resource, id: generatePK().toString() },
                    actions_deleteOne,
                );
            });

            it("API key - no write permissions", async () => {
                await assertRequiresApiKeyWritePermissions(
                    actions.deleteOne,
                    { objectType: DeleteType.Resource, id: generatePK().toString() },
                    actions_deleteOne,
                );
            });
        });

        describe("valid", () => {
            it("deletes own note", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                // Create a note to delete
                const resourceFactory = createResourceDbFactory(DbProvider.get());
                const { resource: note } = await resourceFactory.createWithVersions(1, {
                    resourceType: "Note",
                    ownedByUser: { connect: { id: testUser[0].id } },
                });

                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: DeleteOneInput = {
                    objectType: DeleteType.Resource,
                    id: note.id.toString(),
                };

                const result = await actions.deleteOne({ input }, { req, res }, actions_deleteOne);

                expect(result).toBeDefined();
                expect(result.__typename).toBe("Success");
                expect(result.success).toBe(true);

                // Verify deletion
                const deletedNote = await DbProvider.get().resource.findUnique({
                    where: { id: note.id },
                });
                expect(deletedNote).toBeNull();
            });

            it("deletes team project as team owner", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                // Create team with user as owner
                const teamFactory = new TeamDbFactory();
                const team = await teamFactory.createMinimal({
                    ownedByUser: { connect: { id: testUser[0].id } },
                });

                // Create team project
                const resourceFactory = createResourceDbFactory(DbProvider.get());
                const { resource: project } = await resourceFactory.createWithVersions(1, {
                    resourceType: "Project",
                    ownedByUser: { connect: { id: testUser[0].id } },
                    ownedByTeam: { connect: { id: team.id } },
                });

                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: DeleteOneInput = {
                    objectType: DeleteType.Resource,
                    id: project.id.toString(),
                };

                const result = await actions.deleteOne({ input }, { req, res }, actions_deleteOne);

                expect(result.success).toBe(true);
            });
        });

        describe("invalid", () => {
            it("cannot delete another user's object", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

                // Create note by user 1
                const resourceFactory = createResourceDbFactory(DbProvider.get());
                const { resource: note } = await resourceFactory.createWithVersions(1, {
                    resourceType: "Note",
                    ownedByUser: { connect: { id: testUsers[0].id } },
                });

                // Try to delete as user 2
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[1].id.toString(),
                });

                const input: DeleteOneInput = {
                    objectType: DeleteType.Resource,
                    id: note.id.toString(),
                };

                await expectCustomErrorAsync(
                    actions.deleteOne({ input }, { req, res }, actions_deleteOne),
                    "Unauthorized",
                    "0323",
                );
            });

            it("cannot delete team object without permission", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

                // Create team with user 1 as owner, user 2 as member without delete permission
                const teamFactory = new TeamDbFactory();
                const team = await teamFactory.createMinimal({
                    ownedByUser: { connect: { id: testUsers[0].id } },
                });

                // Add user 2 as member without delete permission
                await DbProvider.get().member.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        teamId: team.id,
                        userId: testUsers[1].id,
                        permissions: "[\"viewChildObjects\"]", // No delete permission
                    },
                });

                // Create team project
                const resourceFactory = createResourceDbFactory(DbProvider.get());
                const { resource: project } = await resourceFactory.createWithVersions(1, {
                    resourceType: "Project",
                    ownedByUser: { connect: { id: testUsers[0].id } },
                    ownedByTeam: { connect: { id: team.id } },
                });

                // Try to delete as user 2
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[1].id.toString(),
                });

                const input: DeleteOneInput = {
                    objectType: DeleteType.Resource,
                    id: project.id.toString(),
                };

                await expectCustomErrorAsync(
                    actions.deleteOne({ input }, { req, res }, actions_deleteOne),
                    "Unauthorized",
                    "0323",
                );
            });
        });
    });

    describe("deleteMany", () => {
        describe("authentication", () => {
            it("not logged in", async () => {
                await assertRequiresAuth(
                    actions.deleteMany,
                    { objectType: DeleteType.Note, ids: [generatePK()] },
                    actions_deleteMany,
                );
            });

            it("API key - no write permissions", async () => {
                await assertRequiresApiKeyWritePermissions(
                    actions.deleteMany,
                    { objectType: DeleteType.Note, ids: [generatePK()] },
                    actions_deleteMany,
                );
            });
        });

        describe("valid", () => {
            it("deletes multiple own notes", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                // Create multiple notes
                const resourceFactory = createResourceDbFactory(DbProvider.get());
                const notes = await Promise.all([
                    resourceFactory.createNoteResource({
                        ownedByUser: { connect: { id: testUser[0].id } },
                    }),
                    resourceFactory.createNoteResource({
                        ownedByUser: { connect: { id: testUser[0].id } },
                    }),
                    resourceFactory.createNoteResource({
                        ownedByUser: { connect: { id: testUser[0].id } },
                    }),
                ]);

                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: DeleteManyInput = {
                    objectType: DeleteType.Note,
                    ids: notes.map(n => n.id.toString()),
                };

                const result = await actions.deleteMany({ input }, { req, res }, actions_deleteMany);

                expect(result).toBeDefined();
                expect(result.__typename).toBe("Count");
                expect(result.count).toBe(3);

                // Verify all were deleted
                const remainingNotes = await DbProvider.get().resource.findMany({
                    where: { id: { in: notes.map(n => n.id) } },
                });
                expect(remainingNotes).toHaveLength(0);
            });

            it("deletes only authorized objects", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

                // Create notes by different users
                const resourceFactory = createResourceDbFactory(DbProvider.get());
                const notes = await Promise.all([
                    // User 1's notes
                    resourceFactory.createNoteResource({
                        ownedByUser: { connect: { id: testUsers[0].id } },
                    }),
                    resourceFactory.createNoteResource({
                        ownedByUser: { connect: { id: testUsers[0].id } },
                    }),
                    // User 2's note
                    resourceFactory.createNoteResource({
                        ownedByUser: { connect: { id: testUsers[1].id } },
                    }),
                ]);

                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: DeleteManyInput = {
                    objectType: DeleteType.Note,
                    ids: notes.map(n => n.id.toString()),
                };

                const result = await actions.deleteMany({ input }, { req, res }, actions_deleteMany);

                expect(result.count).toBe(2); // Only user 1's notes

                // Verify user 2's note still exists
                const remainingNote = await DbProvider.get().note.findUnique({
                    where: { id: notes[2].id },
                });
                expect(remainingNote).toBeDefined();
            });
        });

        describe("invalid", () => {
            it("returns 0 when no objects can be deleted", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

                // Create notes by user 1
                const resourceFactory = createResourceDbFactory(DbProvider.get());
                const notes = await Promise.all([
                    resourceFactory.createNoteResource({
                        ownedByUser: { connect: { id: testUsers[0].id } },
                    }),
                    resourceFactory.createNoteResource({
                        ownedByUser: { connect: { id: testUsers[0].id } },
                    }),
                ]);

                // Try to delete as user 2
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[1].id.toString(),
                });

                const input: DeleteManyInput = {
                    objectType: DeleteType.Note,
                    ids: notes.map(n => n.id.toString()),
                };

                const result = await actions.deleteMany({ input }, { req, res }, actions_deleteMany);

                expect(result.count).toBe(0);
            });
        });
    });

    describe("deleteAll", () => {
        describe("authentication", () => {
            it("not logged in", async () => {
                await assertRequiresAuth(
                    actions.deleteAll,
                    { objectTypes: [DeleteType.Run] },
                    actions_deleteAll,
                );
            });

            it("requires API key with auth write permissions", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWritePrivatePermissions(["Run"]), // Wrong permission type
                });

                const input: DeleteAllInput = {
                    objectTypes: [DeleteType.RunProject],
                };

                await expectCustomErrorAsync(
                    actions.deleteAll({ input }, { req, res }, actions_deleteAll),
                    "Unauthorized",
                    "0323",
                );
            });
        });

        describe("valid", () => {
            it("deletes all Run for user", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                // Create resource and runs
                const resourceFactory = createResourceDbFactory(DbProvider.get());
                const runFactory = createRunDbFactory(DbProvider.get());

                const { resource: _project, versions: _versions } = await resourceFactory.createWithVersions(1, {
                    resourceType: "Project",
                    ownedByUser: { connect: { id: testUser[0].id } },
                });

                await Promise.all([
                    runFactory.createMinimal({
                        name: "Run 1",
                        status: "Completed",
                        isPrivate: false,
                        resourceVersion: { connect: { id: versions[0].id } },
                        user: { connect: { id: testUser[0].id } },
                        completedComplexity: 0,
                        contextSwitches: 0,
                        timeElapsed: 0,
                    }),
                    runFactory.createMinimal({
                        name: "Run 2",
                        status: "InProgress",
                        isPrivate: false,
                        resourceVersion: { connect: { id: versions[0].id } },
                        user: { connect: { id: testUser[0].id } },
                        completedComplexity: 0,
                        contextSwitches: 0,
                        timeElapsed: 0,
                    }),
                ]);

                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: DeleteAllInput = {
                    objectTypes: [DeleteType.Run],
                };

                const result = await actions.deleteAll({ input }, { req, res }, actions_deleteAll);

                expect(result).toBeDefined();
                expect(result.__typename).toBe("Count");
                expect(result.count).toBe(2);

                // Verify all were deleted
                const remaining = await DbProvider.get().run.findMany({
                    where: { userId: testUser[0].id },
                });
                expect(remaining).toHaveLength(0);
            });

            it("deletes multiple object types", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                // Create run projects and routines
                const resourceFactory = createResourceDbFactory(DbProvider.get());
                const runFactory = createRunDbFactory(DbProvider.get());

                const { resource: _project, versions: projectVersions } = await resourceFactory.createWithVersions(1, {
                    resourceType: "Project",
                    ownedByUser: { connect: { id: testUser[0].id } },
                });

                const { resource: _routine, versions: routineVersions } = await resourceFactory.createWithVersions(1, {
                    resourceType: "Routine",
                    ownedByUser: { connect: { id: testUser[0].id } },
                });

                await runFactory.createMinimal({
                    name: "Run Project",
                    status: "Completed",
                    isPrivate: false,
                    resourceVersion: { connect: { id: projectVersions[0].id } },
                    user: { connect: { id: testUser[0].id } },
                    completedComplexity: 0,
                    contextSwitches: 0,
                    timeElapsed: 0,
                });

                await runFactory.createMinimal({
                    name: "Run Routine",
                    status: "Completed",
                    isPrivate: false,
                    resourceVersion: { connect: { id: routineVersions[0].id } },
                    user: { connect: { id: testUser[0].id } },
                    completedComplexity: 0,
                    contextSwitches: 0,
                    timeElapsed: 0,
                });

                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: DeleteAllInput = {
                    objectTypes: [DeleteType.Run],
                };

                const result = await actions.deleteAll({ input }, { req, res }, actions_deleteAll);

                expect(result.count).toBe(2);
            });
        });
    });

    describe("deleteAccount", () => {
        describe("authentication", () => {
            it("not logged in", async () => {
                await assertRequiresAuth(
                    actions.deleteAccount,
                    { password: "password123", deletePublicData: true },
                    actions_deleteAccount,
                );
            });

            it("requires auth write permissions", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockApiSession({
                    userId: testUser[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWritePrivatePermissions(["User"]), // Wrong permission type
                });

                const input: DeleteAccountInput = {
                    password: "password123",
                    deletePublicData: true,
                };

                await expectCustomErrorAsync(
                    actions.deleteAccount({ input }, { req, res }, actions_deleteAccount),
                    "Unauthorized",
                    "0323",
                );
            });
        });

        describe("valid", () => {
            it("deletes account with correct password", async () => {
                const password = "SecurePassword123!";
                const hashedPassword = PasswordAuthService.hashPassword(password);

                // Create user with password
                const testUser = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        auths: {
                            create: {
                                id: generatePK(),
                                provider: AUTH_PROVIDERS.Password,
                                hashed_password: hashedPassword,
                            },
                        },
                    }),
                });

                const { req, res } = await mockApiSession({
                    userId: testUser.id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: DeleteAccountInput = {
                    password,
                    deletePublicData: true,
                };

                const result = await actions.deleteAccount({ input }, { req, res }, actions_deleteAccount);

                expect(result).toBeDefined();
                expect(result.__typename).toBe("Session");
                expect(result.isLoggedIn).toBe(false);

                // Verify user was deleted
                const deletedUser = await DbProvider.get().user.findUnique({
                    where: { id: testUser.id },
                });
                expect(deletedUser).toBeNull();
            });
        });

        describe("invalid", () => {
            it("fails with incorrect password", async () => {
                const password = "SecurePassword123!";
                const hashedPassword = PasswordAuthService.hashPassword(password);

                // Create user with password
                const testUser = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        auths: {
                            create: {
                                id: generatePK(),
                                provider: AUTH_PROVIDERS.Password,
                                hashed_password: hashedPassword,
                            },
                        },
                    }),
                });

                const { req, res } = await mockApiSession({
                    userId: testUser.id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: DeleteAccountInput = {
                    password: "WrongPassword123!",
                    deletePublicData: true,
                };

                await expectCustomErrorAsync(
                    actions.deleteAccount({ input }, { req, res }, actions_deleteAccount),
                    "Unauthorized",
                    "0323",
                );

                // Verify user still exists
                const user = await DbProvider.get().user.findUnique({
                    where: { id: testUser.id },
                });
                expect(user).toBeDefined();
            });

            it("requires password reset if no password set", async () => {
                // Create user without password
                const testUser = await DbProvider.get().user.create({
                    data: UserDbFactory.createMinimal(),
                });

                const { req, res } = await mockApiSession({
                    userId: testUser.id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockWriteAuthPermissions(),
                });

                const input: DeleteAccountInput = {
                    password: "AnyPassword123!",
                    deletePublicData: true,
                };

                await expectCustomErrorAsync(
                    actions.deleteAccount({ input }, { req, res }, actions_deleteAccount),
                    "Unauthorized",
                    "0323",
                );
            });
        });
    });
});
