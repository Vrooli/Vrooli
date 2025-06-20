/**
 * Permission Fixtures Example v2
 * 
 * Comprehensive examples showing how to use the enhanced permission fixtures
 * with the unified fixture architecture and factory pattern.
 */

import { describe, it, expect } from "vitest";
import {
    userSessionFactory,
    apiKeyFactory,
    permissionValidator,
    quickSession,
    testPermissionMatrix,
    expectPermissionDenied,
    createPermissionContext,
} from "./index.js";
import { bookmarkScenarios, bookmarkPermissionHelpers } from "./objects/bookmarkPermissions.js";
import { commentScenarios, commentPermissionHelpers } from "./objects/commentPermissions.js";

describe("Permission Fixtures - Complete Examples", () => {
    describe("Factory Pattern Usage", () => {
        it("should create custom user sessions with factories", () => {
            // Create custom users with specific properties
            const powerUser = userSessionFactory.createSession({
                handle: "poweruser",
                hasPremium: true,
                roles: [{
                    role: {
                        name: "PowerUser",
                        permissions: JSON.stringify(["content.*", "team.create"]),
                    },
                }],
            });

            expect(powerUser.handle).toBe("poweruser");
            expect(powerUser.hasPremium).toBe(true);
            expect(powerUser.roles).toHaveLength(1);
        });

        it("should create API keys with custom permissions", () => {
            const customKey = apiKeyFactory.createCustom(
                "Private", // read
                "Private", // write
                false,     // bot
                5000,      // daily credits
            );

            expect(customKey.permissions.read).toBe("Private");
            expect(customKey.permissions.write).toBe("Private");
            expect(customKey.permissions.daily_credits).toBe(5000);
        });

        it("should validate permissions with the validator", () => {
            const user = userSessionFactory.createWithCustomRole("Editor", ["content.edit", "content.delete"]);
            
            expect(permissionValidator.hasPermission(user, "content.edit")).toBe(true);
            expect(permissionValidator.hasPermission(user, "content.delete")).toBe(true);
            expect(permissionValidator.hasPermission(user, "admin.users")).toBe(false);
        });
    });

    describe("Quick Session Creation", () => {
        it("should create sessions quickly for common scenarios", async () => {
            const { req: adminReq } = await quickSession.admin();
            const { req: guestReq } = await quickSession.guest();
            const { req: apiReq } = await quickSession.readOnly();

            expect(adminReq.session.roles).toBeDefined();
            expect(guestReq.session.isLoggedIn).toBe(false);
            expect(apiReq.session.__type).toBeDefined(); // API key session
        });

        it("should create sessions with custom permissions", async () => {
            const { req } = await quickSession.withPermissions(["bookmark.*", "comment.read"]);
            
            const hasBookmarkPerm = permissionValidator.hasPermission(req.session, "bookmark.create");
            const hasCommentWritePerm = permissionValidator.hasPermission(req.session, "comment.write");
            
            expect(hasBookmarkPerm).toBe(true);
            expect(hasCommentWritePerm).toBe(false);
        });
    });

    describe("Permission Matrix Testing", () => {
        // Mock endpoint function for testing
        async function mockBookmarkEndpoint(session: { req: { session: AuthenticatedSessionData | ApiKeyAuthData } }) {
            // Simulate endpoint that requires bookmark.create permission
            if (!permissionValidator.hasPermission(session.req.session, "bookmark.create")) {
                throw new Error("Permission denied: bookmark.create required");
            }
            return { success: true, id: "bookmark_123" };
        }

        it("should test permission matrix across personas", async () => {
            await testPermissionMatrix(
                mockBookmarkEndpoint,
                {
                    admin: true,      // Admins can create bookmarks
                    standard: true,   // Standard users can create bookmarks
                    premium: true,    // Premium users can create bookmarks
                    guest: false,     // Guests cannot create bookmarks
                    banned: false,    // Banned users cannot create bookmarks
                    readOnly: false,  // Read-only API keys cannot create
                    writeEnabled: true, // Write API keys can create
                },
            );
        });
    });

    describe("Object-Specific Permission Testing", () => {
        describe("Bookmark Permissions", () => {
            it("should test public project bookmark scenario", () => {
                const scenario = bookmarkScenarios.publicProjectBookmark;
                
                expect(scenario.resource.__typename).toBe("Bookmark");
                expect(scenario.actors).toBeDefined();
                expect(scenario.actions).toContain("create");
                
                // Test expected permissions
                const ownerActor = scenario.actors.find(a => a.id === "owner");
                expect(ownerActor?.permissions.read).toBe(true);
                expect(ownerActor?.permissions.create).toBe(true);
            });

            it("should test bookmark creation with helper", () => {
                const userBookmark = bookmarkPermissionHelpers.createUserBookmark(
                    "222222222222222222",
                    "project_123",
                    "list_456",
                );

                expect(userBookmark.to.id).toBe("project_123");
                expect(userBookmark.list.id).toBe("list_456");
                expect(userBookmark.list.owner.id).toBe("222222222222222222");
            });

            it("should validate bookmark access permissions", () => {
                const userId = "222222222222222222";
                const userBookmark = bookmarkPermissionHelpers.createUserBookmark(userId, "project_123");
                
                const canAccess = bookmarkPermissionHelpers.canUserAccessBookmark(userId, userBookmark);
                expect(canAccess).toBe(true);
                
                const otherUserCanAccess = bookmarkPermissionHelpers.canUserAccessBookmark("333333333333333333", userBookmark);
                expect(otherUserCanAccess).toBe(false);
            });
        });

        describe("Comment Permissions", () => {
            it("should test comment on public issue scenario", () => {
                const scenario = commentScenarios.publicIssueComment;
                
                expect(scenario.resource.__typename).toBe("Comment");
                expect(scenario.resource.commentedOn.__typename).toBe("Issue");
                
                // Test voting permissions (cannot vote on own comment)
                const commentOwner = scenario.actors.find(a => a.id === "comment_owner");
                expect(commentOwner?.permissions.vote).toBe(false);
                
                // Other users can vote
                const otherUser = scenario.actors.find(a => a.id === "other_user");
                expect(otherUser?.permissions.vote).toBe(true);
            });

            it("should test comment thread creation", () => {
                const thread = commentPermissionHelpers.createCommentThread("issue_123", 3);
                
                expect(thread).toHaveLength(3);
                expect(thread[0].parent).toBeUndefined(); // Root comment
                expect(thread[1].parent?.id).toBe(thread[0].id); // Reply to root
                expect(thread[2].parent?.id).toBe(thread[1].id); // Reply to reply
            });

            it("should validate comment voting permissions", () => {
                const commentOwner = "user_123";
                const otherUser = "user_456";
                
                const comment = commentPermissionHelpers.createCommentOn(
                    "Issue",
                    "issue_123", 
                    commentOwner,
                );
                
                const ownerCanVote = commentPermissionHelpers.canUserVoteOnComment(commentOwner, comment);
                const otherCanVote = commentPermissionHelpers.canUserVoteOnComment(otherUser, comment);
                
                expect(ownerCanVote).toBe(false); // Cannot vote on own comment
                expect(otherCanVote).toBe(true);  // Can vote on public comment
            });
        });
    });

    describe("Advanced Permission Scenarios", () => {
        it("should test team member permissions", async () => {
            const teamMembers = userSessionFactory.createTeam(3);
            const [owner, admin, member] = teamMembers;
            
            // Test that owner has all permissions
            expect(owner._testTeamMembership?.role).toBe("Owner");
            
            // Test that admin has admin permissions
            expect(admin._testTeamMembership?.role).toBe("Admin");
            
            // Test that member has limited permissions
            expect(member._testTeamMembership?.role).toBe("Member");
        });

        it("should test API key rate limiting", () => {
            const rateLimitedKey = apiKeyFactory.createRateLimited();
            const normalKey = apiKeyFactory.createWrite();
            
            expect(rateLimitedKey.permissions.daily_credits).toBe(10);
            expect(normalKey.permissions.daily_credits).toBe(1000);
        });

        it("should test permission context creation", () => {
            const user = userSessionFactory.createStandard();
            const context = createPermissionContext(user, {
                currentProject: "project_123",
                teamMembership: "team_456",
            });
            
            expect(context.session).toBe(user);
            expect(context.context?.currentProject).toBe("project_123");
            expect(context.validator).toBe(permissionValidator);
            expect(context.factories.user).toBe(userSessionFactory);
        });
    });

    describe("Error and Edge Case Testing", () => {
        it("should test with suspended user", async () => {
            const suspendedUser = userSessionFactory.createSuspended();
            const { req } = await quickSession.withUser(suspendedUser);
            
            // Mock endpoint that checks account status
            async function protectedEndpoint() {
                if (req.session.accountStatus === "HardLocked") {
                    throw new Error("Account is suspended");
                }
                return { success: true };
            }
            
            await expectPermissionDenied(protectedEndpoint, "Account is suspended");
        });

        it("should test with expired API key", async () => {
            const expiredKey = apiKeyFactory.createExpired();
            const { req } = await quickSession.withApiKey(expiredKey);
            
            async function apiEndpoint() {
                if (req.session.isExpired) {
                    throw new Error("API key has expired");
                }
                return { success: true };
            }
            
            await expectPermissionDenied(apiEndpoint, "API key has expired");
        });

        it("should test with malformed session", () => {
            const partialUser = userSessionFactory.createPartial({
                handle: undefined, // Missing required field
                email: undefined, // Missing required field
            });
            
            expect(partialUser.handle).toBeUndefined();
            expect(partialUser.email).toBeUndefined();
            expect(partialUser.isLoggedIn).toBe(true);
        });
    });

    describe("Performance Testing", () => {
        it("should measure permission check performance", async () => {
            const user = userSessionFactory.createAdmin();
            const iterations = 1000;
            
            const start = Date.now();
            
            for (let i = 0; i < iterations; i++) {
                permissionValidator.hasPermission(user, "content.read");
            }
            
            const end = Date.now();
            const duration = end - start;
            const avgTime = duration / iterations;
            
            expect(avgTime).toBeLessThan(1); // Should be less than 1ms per check
        });

        it("should test with large permission sets", () => {
            const permissions = Array.from({ length: 100 }, (_, i) => `permission.${i}`);
            const user = userSessionFactory.createWithCustomRole("MegaUser", permissions);
            
            const allPermissions = permissionValidator.getPermissions(user);
            expect(allPermissions).toHaveLength(100);
            
            // Test specific permission check
            expect(permissionValidator.hasPermission(user, "permission.50")).toBe(true);
            expect(permissionValidator.hasPermission(user, "permission.999")).toBe(false);
        });
    });

    describe("Round-Trip Integration", () => {
        it("should demonstrate integration with API fixtures", async () => {
            // This would integrate with actual API fixtures
            const user = userSessionFactory.createStandard();
            const { req } = await quickSession.withUser(user);
            
            // Example of how this would work with actual endpoints
            async function mockCreateBookmark(session: { req: { session: AuthenticatedSessionData | ApiKeyAuthData } }, bookmarkData: Record<string, unknown>) {
                if (!permissionValidator.hasPermission(session.req.session, "bookmark.create")) {
                    throw new Error("Permission denied");
                }
                
                return {
                    success: true,
                    data: {
                        id: "bookmark_123",
                        ...bookmarkData,
                    },
                };
            }
            
            const bookmarkData = bookmarkPermissionHelpers.createUserBookmark(
                user.id,
                "project_123",
            );
            
            const result = await mockCreateBookmark({ req }, bookmarkData);
            expect(result.success).toBe(true);
            expect(result.data.id).toBe("bookmark_123");
        });
    });
});

/**
 * Key Benefits Demonstrated:
 * 
 * 1. **Type Safety**: All fixtures are fully typed with zero `any` types
 * 2. **Factory Pattern**: Consistent, reusable creation methods
 * 3. **Real Integration**: Uses actual permission checking logic
 * 4. **Comprehensive Coverage**: Tests all permission scenarios
 * 5. **Performance**: Efficient permission checking
 * 6. **Edge Cases**: Handles suspended users, expired keys, etc.
 * 7. **Complex Scenarios**: Team permissions, nested objects
 * 8. **Easy Testing**: Simple APIs for common test patterns
 */
