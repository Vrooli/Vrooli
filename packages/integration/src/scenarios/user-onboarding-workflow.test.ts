/**
 * User Onboarding Workflow Scenario Test
 * 
 * This scenario tests a complete user onboarding workflow:
 * 1. User creates account
 * 2. User completes profile setup
 * 3. User creates their first bookmark list
 * 4. User bookmarks another user
 * 5. Verify the complete workflow worked end-to-end
 */

import { describe, it, expect, beforeEach } from "vitest";
import { createSimpleTestUser } from "../utils/simple-helpers.js";
import { getPrisma } from "../setup/test-setup.js";
import { generatePK } from "@vrooli/shared";

describe("User Onboarding Workflow Scenario", () => {
    let prisma: any;

    beforeEach(async () => {
        prisma = getPrisma();
    });

    it("should complete full user onboarding workflow", async () => {
        console.log("🎬 Starting user onboarding scenario...");

        // Step 1: Create first user (representing new signup)
        console.log("📝 Step 1: User signup");
        const { user: newUser, sessionData: _sessionData } = await createSimpleTestUser();
        
        expect(newUser).toBeDefined();
        expect(newUser.name).toBe("Test User");
        expect(newUser.emails).toHaveLength(1);
        expect(newUser.emails[0].verifiedAt).toBeDefined();
        
        console.log(`✅ User created: ${newUser.handle} (${newUser.emails[0].emailAddress})`);

        // Step 2: Complete profile setup (simulated by updating user)
        console.log("📝 Step 2: Profile completion");
        const updatedUser = await prisma.user.update({
            where: { id: newUser.id },
            data: {
                name: "John Developer",
                theme: "dark",
                languages: ["en", "es"],
            },
        });

        expect(updatedUser.name).toBe("John Developer");
        expect(updatedUser.theme).toBe("dark");
        console.log(`✅ Profile completed for ${updatedUser.name}`);

        // Step 3: Create another user to bookmark
        console.log("📝 Step 3: Create target user for bookmarking");
        const { user: targetUser } = await createSimpleTestUser();
        
        expect(targetUser).toBeDefined();
        console.log(`✅ Target user created: ${targetUser.handle}`);

        // Step 4: Create bookmark list
        console.log("📝 Step 4: Create bookmark list");
        const bookmarkList = await prisma.bookmark_list.create({
            data: {
                id: String(generatePK()),
                label: "My Favorite Developers",
                user: { connect: { id: newUser.id } },
            },
        });

        expect(bookmarkList).toBeDefined();
        expect(bookmarkList.label).toBe("My Favorite Developers");
        console.log(`✅ Bookmark list created: "${bookmarkList.label}"`);

        // Step 5: Bookmark the target user
        console.log("📝 Step 5: Bookmark target user");
        const bookmark = await prisma.bookmark.create({
            data: {
                id: String(generatePK()),
                user: { connect: { id: targetUser.id } }, // User being bookmarked
                list: { connect: { id: bookmarkList.id } },
            },
            include: {
                user: {
                    select: {
                        id: true,
                        handle: true,
                        name: true,
                    },
                },
                list: {
                    select: {
                        id: true,
                        label: true,
                    },
                },
            },
        });

        expect(bookmark).toBeDefined();
        expect(bookmark.user.id).toBe(targetUser.id);
        expect(bookmark.list.id).toBe(bookmarkList.id);
        console.log(`✅ Bookmarked user: ${bookmark.user.handle} in list "${bookmark.list.label}"`);

        // Step 6: Verify complete workflow state
        console.log("📝 Step 6: Verify workflow completion");
        
        // Check user has completed onboarding
        const finalUser = await prisma.user.findUnique({
            where: { id: newUser.id },
            include: {
                emails: true,
                bookmakLists: {
                    include: {
                        bookmarks: {
                            include: {
                                user: {
                                    select: {
                                        handle: true,
                                        name: true,
                                    },
                                },
                            },
                        },
                    },
                },
            },
        });

        // Verify onboarding completion state
        expect(finalUser).toBeDefined();
        expect(finalUser.name).toBe("John Developer"); // Profile completed
        expect(finalUser.emails).toHaveLength(1); // Email verified
        expect(finalUser.emails[0].verifiedAt).toBeDefined(); // Email verified
        expect(finalUser.bookmakLists).toHaveLength(1); // Has bookmark lists
        expect(finalUser.bookmakLists[0].bookmarks).toHaveLength(1); // Has bookmarks
        expect(finalUser.bookmakLists[0].bookmarks[0].user.handle).toBe(targetUser.handle); // Correct user bookmarked

        console.log("📊 Workflow completion metrics:");
        console.log("   - User profile completed: ✅");
        console.log("   - Email verified: ✅");
        console.log(`   - Bookmark lists created: ${finalUser.bookmakLists.length}`);
        console.log(`   - Total bookmarks: ${finalUser.bookmakLists[0].bookmarks.length}`);
        console.log(`   - Bookmarked users: ${finalUser.bookmakLists[0].bookmarks.map(b => b.user.handle).join(", ")}`);

        // Verify workflow business logic
        const workflowMetrics = {
            userSignedUp: true,
            profileCompleted: finalUser.name !== "Test User",
            emailVerified: finalUser.emails[0].verifiedAt !== null,
            hasBookmarkLists: finalUser.bookmakLists.length > 0,
            hasBookmarks: finalUser.bookmakLists.some(list => list.bookmarks.length > 0),
            socialInteraction: finalUser.bookmakLists[0].bookmarks.length > 0,
        };

        expect(workflowMetrics.userSignedUp).toBe(true);
        expect(workflowMetrics.profileCompleted).toBe(true);
        expect(workflowMetrics.emailVerified).toBe(true);
        expect(workflowMetrics.hasBookmarkLists).toBe(true);
        expect(workflowMetrics.hasBookmarks).toBe(true);
        expect(workflowMetrics.socialInteraction).toBe(true);

        console.log("🎉 User onboarding workflow completed successfully!");
    });

    it("should handle partial onboarding and allow resume", async () => {
        console.log("🎬 Starting partial onboarding scenario...");

        // Create user but don't complete all steps
        const { user: partialUser } = await createSimpleTestUser();
        
        // User completes profile but doesn't create bookmark lists yet
        await prisma.user.update({
            where: { id: partialUser.id },
            data: {
                name: "Partial User",
                theme: "light",
            },
        });

        // Check onboarding state
        const userState = await prisma.user.findUnique({
            where: { id: partialUser.id },
            include: {
                emails: true,
                bookmakLists: true,
            },
        });

        const onboardingProgress = {
            profileCompleted: userState.name !== "Test User",
            emailVerified: userState.emails[0].verifiedAt !== null,
            hasBookmarkLists: userState.bookmakLists.length > 0,
        };

        expect(onboardingProgress.profileCompleted).toBe(true);
        expect(onboardingProgress.emailVerified).toBe(true);
        expect(onboardingProgress.hasBookmarkLists).toBe(false); // Not completed yet

        console.log("📊 Partial onboarding state:");
        console.log(`   - Profile completed: ${onboardingProgress.profileCompleted ? "✅" : "❌"}`);
        console.log(`   - Email verified: ${onboardingProgress.emailVerified ? "✅" : "❌"}`);
        console.log(`   - Bookmark lists created: ${onboardingProgress.hasBookmarkLists ? "✅" : "❌"}`);

        // User can resume later and complete onboarding
        const bookmarkList = await prisma.bookmark_list.create({
            data: {
                id: String(generatePK()),
                label: "Resume Later List",
                user: { connect: { id: partialUser.id } },
            },
        });

        expect(bookmarkList).toBeDefined();
        console.log("✅ User resumed and completed onboarding");
    });
});
