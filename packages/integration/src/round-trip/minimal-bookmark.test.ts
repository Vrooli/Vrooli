import { describe, it, expect, beforeEach } from "vitest";
import { createSimpleTestUser } from "../utils/simple-helpers.js";
import { BookmarkFor } from "@vrooli/shared";
import { getPrisma } from "../setup/test-setup.js";
import { generatePK } from "@vrooli/shared";

describe("Minimal Bookmark Round-Trip Tests", () => {
    let testUser: any;
    let _sessionData: any;
    let prisma: any;

    beforeEach(async () => {
        // Get Prisma instance
        prisma = getPrisma();
        
        // Create a test user for each test
        const result = await createSimpleTestUser();
        testUser = result.user;
        _sessionData = result.sessionData;
    });

    it("should create a bookmark directly via Prisma", async () => {
        // Create bookmark list
        const bookmarkList = await prisma.bookmark_list.create({
            data: {
                id: String(generatePK()),
                label: "My Reading List",
                user: { connect: { id: testUser.id } },
            },
        });

        // Create bookmark
        const bookmark = await prisma.bookmark.create({
            data: {
                id: String(generatePK()),
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
