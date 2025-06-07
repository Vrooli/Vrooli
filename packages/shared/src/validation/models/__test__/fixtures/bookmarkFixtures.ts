import { BookmarkFor } from "../../../../api/types.js";
import { type ModelTestFixtures, TestDataFactory } from "../validationTestUtils.js";

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    id5: "123456789012345682",
    listId1: "123456789012345683",
    listId2: "123456789012345684",
    forId1: "123456789012345685",
    forId2: "123456789012345686",
};

// Shared bookmark test fixtures
export const bookmarkFixtures: ModelTestFixtures = {
    minimal: {
        create: {
            id: validIds.id1,
            bookmarkFor: BookmarkFor.Tag,
            forConnect: validIds.forId1,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id2,
            bookmarkFor: BookmarkFor.Resource,
            forConnect: validIds.forId2,
            listCreate: {
                id: validIds.listId1,
                label: "My Bookmarks",
            },
        },
        update: {
            id: validIds.id2,
            listConnect: validIds.listId2,
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing id, bookmarkFor, and forConnect
                listConnect: validIds.listId1,
            },
            update: {
                // Missing id
                listConnect: validIds.listId1,
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                bookmarkFor: "InvalidType", // Should be valid BookmarkFor enum
                forConnect: 123, // Should be string
            },
            update: {
                id: validIds.id3,
                listConnect: 123, // Should be string
            },
        },
        invalidBookmarkFor: {
            create: {
                id: validIds.id1,
                bookmarkFor: "NotAValidEnum",
                forConnect: validIds.forId1,
            },
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                bookmarkFor: BookmarkFor.Tag,
                forConnect: validIds.forId1,
            },
        },
        missingForConnection: {
            create: {
                id: validIds.id1,
                bookmarkFor: BookmarkFor.Tag,
                // Missing required 'forConnect' relationship
            },
        },
    },
    edgeCases: {
        allBookmarkForTypes: [
            {
                id: validIds.id1,
                bookmarkFor: BookmarkFor.Comment,
                forConnect: validIds.forId1,
            },
            {
                id: validIds.id2,
                bookmarkFor: BookmarkFor.Issue,
                forConnect: validIds.forId1,
            },
            {
                id: validIds.id3,
                bookmarkFor: BookmarkFor.Resource,
                forConnect: validIds.forId1,
            },
            {
                id: validIds.id4,
                bookmarkFor: BookmarkFor.Tag,
                forConnect: validIds.forId1,
            },
            {
                id: validIds.id5,
                bookmarkFor: BookmarkFor.Team,
                forConnect: validIds.forId1,
            },
        ],
        listCreateAndConnect: {
            create: {
                id: validIds.id1,
                bookmarkFor: BookmarkFor.User,
                forConnect: validIds.forId1,
                listCreate: {
                    id: validIds.listId1,
                    label: "Created List",
                },
            },
            update: {
                id: validIds.id1,
                listConnect: validIds.listId2,
            },
        },
        updateWithListUpdate: {
            update: {
                id: validIds.id1,
                listUpdate: {
                    id: validIds.listId1,
                    label: "Updated List Label",
                },
            },
        },
    },
};

// Custom factory that always generates valid IDs
const customizers = {
    create: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
    }),
    update: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const bookmarkTestDataFactory = new TestDataFactory(bookmarkFixtures, customizers);
