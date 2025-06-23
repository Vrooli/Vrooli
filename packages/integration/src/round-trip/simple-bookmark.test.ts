import { describe, it, expect, beforeEach } from "vitest";
import { createTestUser } from "../utils/test-helpers.js";
import { BookmarkFor } from "@vrooli/shared";
import { getPrisma } from "../setup/test-setup.js";

describe("Simple Bookmark Round-Trip Tests", () => {
    let testUser: any;
    let _sessionData: any;
    let prisma: any;

    beforeEach(async () => {
        // Get Prisma instance
        prisma = getPrisma();
        
        // Create a test user for each test
        const result = await createTestUser();
        testUser = result.user;
        _sessionData = result.sessionData;
    });

    it("should create a bookmark directly via Prisma", async () => {
        // Create bookmark list
        const bookmarkList = await prisma.bookmarkList.create({
            data: {
                id: "123456789012345678901234",
                label: "My Reading List",
                user: { connect: { id: testUser.id } },
            },
        });

        // Create bookmark
        const bookmark = await prisma.bookmark.create({
            data: {
                id: "234567890123456789012345",
                bookmarkFor: BookmarkFor.User,
                user: { connect: { id: testUser.id } },
                list: { connect: { id: bookmarkList.id } },
            },
            include: {
                list: true,
            },
        });

        expect(bookmark).toBeDefined();
        expect(bookmark.bookmarkFor).toBe(BookmarkFor.User);
        expect(bookmark.list.label).toBe("My Reading List");
    });
});
