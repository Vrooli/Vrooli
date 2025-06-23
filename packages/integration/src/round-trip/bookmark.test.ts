import { Bookmark } from "@vrooli/server";
import { BookmarkCreateInput, BookmarkFor, shape, transformInput } from "@vrooli/shared";
import { beforeEach, describe, expect, it } from "vitest";
import { createTestUser } from "../utils/test-helpers.js";

describe("Bookmark Round-Trip Tests", () => {
    let testUser: any;
    let sessionData: any;

    beforeEach(async () => {
        // Create a test user for each test
        const result = await createTestUser();
        testUser = result.user;
        sessionData = result.sessionData;
    });

    it("should create a bookmark through the full stack", async () => {
        // Step 1: Create form data as if from UI
        const formData = {
            bookmarkFor: BookmarkFor.User,
            forId: testUser.id,
            list: {
                id: "new",
                label: "My Reading List",
            },
        };

        // Step 2: Shape the data using UI shaping logic
        const shapedData = shape(
            formData,
            ["id", "list"],
            ["bookmarkFor", "forId"],
            "Bookmark",
            ["Create"],
        );

        // Step 3: Validate and transform using shared validation
        const validatedInput = transformInput<BookmarkCreateInput>(
            shapedData,
            "BookmarkCreateInput",
        );

        // Step 4: Call the endpoint logic directly (simulating API call)
        const result = await Bookmark.Create.performLogic({
            body: validatedInput,
            session: {
                id: sessionData.users[0].id,
                languages: sessionData.languages,
            },
        });

        // Step 5: Verify the result
        expect(result).toBeDefined();
        expect(result.id).toBeDefined();
        expect(result.bookmarkFor).toBe(BookmarkFor.User);
        expect(result.forId).toBe(testUser.id);
        expect(result.list).toBeDefined();
        expect(result.list.label).toBe("My Reading List");

        // Step 6: Verify database state
        const prisma = (global as any).__PRISMA__;
        const dbBookmark = await prisma.bookmark.findUnique({
            where: { id: result.id },
            include: { list: true },
        });

        expect(dbBookmark).toBeDefined();
        expect(dbBookmark.bookmarkFor).toBe(BookmarkFor.User);
        expect(dbBookmark.user.id).toBe(testUser.id);
        expect(dbBookmark.list.label).toBe("My Reading List");
    });

    it("should handle validation errors correctly", async () => {
        // Test with invalid data
        const invalidData = {
            bookmarkFor: "InvalidType", // Invalid enum value
            forId: "not-a-valid-id",
        };

        // This should throw a validation error
        await expect(async () => {
            const shapedData = shape(
                invalidData,
                ["id", "list"],
                ["bookmarkFor", "forId"],
                "Bookmark",
                ["Create"],
            );

            transformInput<BookmarkCreateInput>(
                shapedData,
                "BookmarkCreateInput",
            );
        }).rejects.toThrow();
    });

    it("should enforce permissions correctly", async () => {
        // Create another user
        const { user: otherUser } = await createTestUser();

        // Try to bookmark a private user
        await (global as any).__PRISMA__.user.update({
            where: { id: otherUser.id },
            data: { isPrivate: true },
        });

        const formData = {
            bookmarkFor: BookmarkFor.User,
            forId: otherUser.id,
            list: {
                id: "new",
                label: "My Reading List",
            },
        };

        const shapedData = shape(
            formData,
            ["id", "list"],
            ["bookmarkFor", "forId"],
            "Bookmark",
            ["Create"],
        );

        const validatedInput = transformInput<BookmarkCreateInput>(
            shapedData,
            "BookmarkCreateInput",
        );

        // This should fail with permission error
        await expect(
            Bookmark.Create.performLogic({
                body: validatedInput,
                session: {
                    id: sessionData.users[0].id,
                    languages: sessionData.languages,
                },
            }),
        ).rejects.toThrow();
    });
});
