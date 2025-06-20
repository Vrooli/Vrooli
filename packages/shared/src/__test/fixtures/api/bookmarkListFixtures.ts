import { BookmarkFor, type BookmarkListCreateInput, type BookmarkListUpdateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { bookmarkListValidation } from "../../../validation/models/bookmarkList.js";

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    bookmarkId1: "123456789012345682",
    bookmarkId2: "123456789012345683",
    forId1: "123456789012345684",
    forId2: "123456789012345685",
};

// Shared bookmarkList test fixtures
export const bookmarkListFixtures: ModelTestFixtures<BookmarkListCreateInput, BookmarkListUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            label: "My List",
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id2,
            label: "Complete List",
            bookmarksCreate: [
                {
                    id: validIds.bookmarkId1,
                    bookmarkFor: BookmarkFor.Tag,
                    forConnect: validIds.forId1,
                },
            ],
        },
        update: {
            id: validIds.id2,
            label: "Updated List",
            bookmarksConnect: [validIds.bookmarkId2],
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing id and label
                bookmarksCreate: [],
            },
            update: {
                // Missing id
                label: "Updated",
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                label: 456, // Should be string
            },
            update: {
                id: validIds.id3,
                label: false, // Should be string
            },
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                label: "Valid Label",
            },
        },
        emptyLabel: {
            create: {
                id: validIds.id1,
                label: "", // Empty string should be removed and cause required error
            },
        },
        labelTooLong: {
            create: {
                id: validIds.id1,
                label: "x".repeat(129), // Max 128 characters
            },
        },
    },
    edgeCases: {
        maxLengthLabel: {
            create: {
                id: validIds.id1,
                label: "x".repeat(128), // Exactly at max length
            },
        },
        withWhitespaceLabel: {
            create: {
                id: validIds.id1,
                label: "  Valid Label  ", // Should be trimmed
            },
        },
        multipleBookmarks: {
            create: {
                id: validIds.id1,
                label: "Multi Bookmark List",
                bookmarksCreate: [
                    {
                        id: validIds.bookmarkId1,
                        bookmarkFor: BookmarkFor.Tag,
                        forConnect: validIds.forId1,
                    },
                    {
                        id: validIds.bookmarkId2,
                        bookmarkFor: BookmarkFor.Resource,
                        forConnect: validIds.forId2,
                    },
                ],
            },
        },
        updateWithAllBookmarkOperations: {
            update: {
                id: validIds.id1,
                label: "Updated List",
                bookmarksConnect: [validIds.bookmarkId1],
                bookmarksCreate: [
                    {
                        id: validIds.bookmarkId2,
                        bookmarkFor: BookmarkFor.User,
                        forConnect: validIds.forId2,
                    },
                ],
                bookmarksUpdate: [
                    {
                        id: validIds.bookmarkId1,
                    },
                ],
                bookmarksDelete: [validIds.bookmarkId2],
            },
        },
    },
};

// Custom factory that always generates valid IDs
const customizers = {
    create: (base: BookmarkListCreateInput): BookmarkListCreateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
    update: (base: BookmarkListUpdateInput): BookmarkListUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const bookmarkListTestDataFactory = new TypedTestDataFactory(bookmarkListFixtures, bookmarkListValidation, customizers);
export const typedBookmarkListFixtures = createTypedFixtures(bookmarkListFixtures, bookmarkListValidation);
