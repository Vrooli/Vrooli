import { describe, it, expect, beforeEach } from "vitest";
import { createSimpleTestUser } from "../utils/simple-helpers.js";
import { getPrisma } from "../setup/test-setup.js";
import { generatePK } from "@vrooli/shared";

describe("Working Bookmark Tests", () => {
    let testUser: any;
    let testUser2: any;
    let prisma: any;

    beforeEach(async () => {
        // Get Prisma instance
        prisma = getPrisma();
        
        // Create test users
        const result1 = await createSimpleTestUser();
        testUser = result1.user;
        
        const result2 = await createSimpleTestUser();
        testUser2 = result2.user;
    });

    it("should create a bookmark list and bookmark a user", async () => {
        // Create bookmark list
        const bookmarkList = await prisma.bookmark_list.create({
            data: {
                id: String(generatePK()),
                label: "My Favorite Users",
                user: { connect: { id: testUser.id } },
            },
        });

        expect(bookmarkList).toBeDefined();
        expect(bookmarkList.label).toBe("My Favorite Users");

        // Create bookmark for the second user
        const bookmark = await prisma.bookmark.create({
            data: {
                id: String(generatePK()),
                user: { connect: { id: testUser2.id } }, // User being bookmarked
                list: { connect: { id: bookmarkList.id } },
            },
            include: {
                list: true,
                user: true,
            },
        });

        expect(bookmark).toBeDefined();
        expect(bookmark.list.label).toBe("My Favorite Users");
        expect(bookmark.user.id).toBe(testUser2.id);
    });
});
