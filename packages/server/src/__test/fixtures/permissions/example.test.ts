import { beforeEach, describe, expect, it, jest } from "@jest/globals";
import { BookmarkFor } from "@vrooli/shared";
import { DbProvider } from "../../../services/db.js";
import { bookmarkList } from "../../../endpoints/logic/bookmarkList.js";
import { bookmarkList_findOne } from "../../../endpoints/generated/bookmarkList_findOne.js";
import {
    adminUser,
    standardUser,
    guestUser,
    writeApiKey,
    readOnlyPublicApiKey,
    basicTeamScenario,
    quickSession,
    testPermissionMatrix,
    createSession,
    expectPermissionDenied,
    expectPermissionGranted,
} from "./index.js";

/**
 * Example test file showing how to use permission fixtures
 * 
 * This demonstrates various patterns for testing permissions
 * using the centralized fixture system.
 */

describe("Permission Fixtures Example", () => {
    let cleanup: (() => Promise<void>)[] = [];

    beforeEach(async () => {
        // Clean up any test data
        for (const cleanupFn of cleanup) {
            await cleanupFn();
        }
        cleanup = [];
    });

    describe("Basic User Permissions", () => {
        it("should use quickSession for simple tests", async () => {
            // Create a bookmark list owned by standard user
            const list = await DbProvider.get().bookmarkList.create({
                data: {
                    id: "test_list_001",
                    label: "My Bookmarks",
                    user: { connect: { id: standardUser.id } },
                },
            });
            cleanup.push(() => DbProvider.get().bookmarkList.delete({ where: { id: list.id } }));

            // Test owner can access
            const { req, res } = await quickSession.standard();
            const result = await bookmarkList.findOne(
                { input: { id: list.id } },
                { req, res },
                bookmarkList_findOne,
            );
            expect(result).toBeDefined();
            expect(result.id).toBe(list.id);
        });

        it("should test permission matrix for multiple scenarios", async () => {
            // Create a private bookmark list
            const list = await DbProvider.get().bookmarkList.create({
                data: {
                    id: "test_list_002",
                    label: "Private List",
                    user: { connect: { id: adminUser.id } },
                },
            });
            cleanup.push(() => DbProvider.get().bookmarkList.delete({ where: { id: list.id } }));

            // Test access matrix
            await testPermissionMatrix(
                async (session) => {
                    return bookmarkList.findOne(
                        { input: { id: list.id } },
                        session,
                        bookmarkList_findOne,
                    );
                },
                {
                    admin: true,      // Owner can access
                    standard: false,  // Other users cannot
                    guest: false,     // Guests cannot
                    readOnly: false,  // API keys of other users cannot
                },
            );
        });
    });

    describe("API Key Permissions", () => {
        it("should test read vs write API keys", async () => {
            // Create test data
            const userId = standardUser.id;
            const list = await DbProvider.get().bookmarkList.create({
                data: {
                    id: "test_list_003",
                    label: "API Test List",
                    user: { connect: { id: userId } },
                },
            });
            cleanup.push(() => DbProvider.get().bookmarkList.delete({ where: { id: list.id } }));

            // Test read-only API key can read
            const readSession = await quickSession.readOnly();
            await expectPermissionGranted(async () => {
                await bookmarkList.findOne(
                    { input: { id: list.id } },
                    readSession,
                    bookmarkList_findOne,
                );
            });

            // Test write API key can also read (write implies read)
            const writeSession = await quickSession.write();
            await expectPermissionGranted(async () => {
                await bookmarkList.findOne(
                    { input: { id: list.id } },
                    writeSession,
                    bookmarkList_findOne,
                );
            });
        });
    });

    describe("Team Permissions", () => {
        it("should test team member access levels", async () => {
            const scenario = basicTeamScenario;
            
            // Create a team resource
            const teamResource = await DbProvider.get().project.create({
                data: {
                    id: "test_project_001",
                    name: "Team Project",
                    isPrivate: true,
                    team: { connect: { id: scenario.team.id } },
                    createdBy: { connect: { id: scenario.members[0].user.id } },
                },
            });
            cleanup.push(() => DbProvider.get().project.delete({ where: { id: teamResource.id } }));

            // Test each team member's access
            for (const member of scenario.members) {
                const session = await createSession(member.user);
                
                // All team members should be able to read
                await expectPermissionGranted(async () => {
                    // In real test, call the project.findOne endpoint
                    // This is just an example
                    return { success: true };
                });

                // Only certain roles can delete
                if (member.role === "Owner") {
                    await expectPermissionGranted(async () => {
                        // Test delete permission
                        return { success: true };
                    });
                } else {
                    await expectPermissionDenied(async () => {
                        // Test delete permission
                        throw new Error("Permission denied");
                    });
                }
            }
        });
    });

    describe("Edge Cases", () => {
        it("should handle banned users", async () => {
            const { req, res } = await createSession(bannedUser);
            
            // Banned users should be denied most operations
            await expectPermissionDenied(async () => {
                // Most endpoints should reject banned users
                throw new Error("Account is locked");
            }, /locked|banned/i);
        });

        it("should test guest access to public resources", async () => {
            // Create a public resource
            const publicList = await DbProvider.get().bookmarkList.create({
                data: {
                    id: "test_list_public",
                    label: "Public Bookmarks",
                    user: { connect: { id: standardUser.id } },
                    // In real app, there would be an isPublic flag
                },
            });
            cleanup.push(() => DbProvider.get().bookmarkList.delete({ where: { id: publicList.id } }));

            const { req, res } = await quickSession.guest();
            
            // Guests cannot access bookmark lists (they're always private in this example)
            await expectPermissionDenied(async () => {
                await bookmarkList.findOne(
                    { input: { id: publicList.id } },
                    { req, res },
                    bookmarkList_findOne,
                );
            });
        });
    });

    describe("Custom Scenarios", () => {
        it("should create custom user with specific permissions", async () => {
            // Create a user with custom permissions
            const moderator = createUserWithPermissions(
                { handle: "moderator", name: "Content Moderator" },
                ["moderate_content", "view_reports", "ban_users"],
            );

            const session = await createSession(moderator);
            
            // Test that custom permissions are properly set
            expect(session.req.session.data.roles).toHaveLength(1);
            expect(session.req.session.data.roles[0].role.permissions).toContain("moderate_content");
        });
    });
});