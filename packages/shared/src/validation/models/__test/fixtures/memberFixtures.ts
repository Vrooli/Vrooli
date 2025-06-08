import { type ModelTestFixtures, TestDataFactory } from "../validationTestUtils.js";

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
};

// Shared member test fixtures
export const memberFixtures: ModelTestFixtures = {
    minimal: {
        create: null, // No create operation supported
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: null, // No create operation supported
        update: {
            id: validIds.id2,
            isAdmin: true,
            permissions: JSON.stringify(["Create", "Read", "Update", "Delete"]),
        },
    },
    invalid: {
        missingRequired: {
            create: null, // No create operation supported
            update: {
                // Missing id
                isAdmin: true,
            },
        },
        invalidTypes: {
            create: null, // No create operation supported
            update: {
                id: 123, // Should be string
                isAdmin: "yes", // Should be boolean
                permissions: "invalid", // Should be array
            },
        },
        invalidId: {
            create: null, // No create operation supported
            update: {
                id: "not-a-valid-snowflake",
                isAdmin: false,
            },
        },
    },
    edgeCases: {
        onlyAdminChange: {
            update: {
                id: validIds.id1,
                isAdmin: false,
            },
        },
        onlyPermissionsChange: {
            update: {
                id: validIds.id1,
                permissions: JSON.stringify(["Read"]),
            },
        },
        emptyPermissions: {
            update: {
                id: validIds.id1,
                permissions: JSON.stringify([]),
            },
        },
        fullPermissions: {
            update: {
                id: validIds.id1,
                permissions: JSON.stringify(["Create", "Read", "Update", "Delete", "UseApi", "Manage"]),
            },
        },
    },
};

// Custom factory that always generates valid IDs
const customizers = {
    create: (_base: any) => null, // No create operation
    update: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const memberTestDataFactory = new TestDataFactory(memberFixtures, customizers);
