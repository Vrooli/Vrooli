import type { MemberUpdateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { memberValidation } from "../../../validation/models/member.js";

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
};

// Shared member test fixtures (Member has no create operation)
export const memberFixtures: ModelTestFixtures<never, MemberUpdateInput> = {
    minimal: {
        create: null as never, // No create operation supported
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: null as never, // No create operation supported
        update: {
            id: validIds.id2,
            isAdmin: true,
            permissions: JSON.stringify(["Create", "Read", "Update", "Delete"]),
        },
    },
    invalid: {
        missingRequired: {
            create: null as never, // No create operation supported
            update: {
                // Missing id
                isAdmin: true,
            } as MemberUpdateInput,
        },
        invalidTypes: {
            create: null as never, // No create operation supported
            update: {
                id: 123, // Should be string
                isAdmin: "yes", // Should be boolean
                permissions: "invalid", // Should be array
            } as unknown as MemberUpdateInput,
        },
        invalidId: {
            create: null as never, // No create operation supported
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
    create: (_base: never): never => null as never, // No create operation
    update: (base: Partial<MemberUpdateInput>): MemberUpdateInput => ({
        id: validIds.id1,
        ...base,
    }),
};

// Export a factory for creating test data programmatically
export const memberTestDataFactory = new TypedTestDataFactory(memberFixtures, memberValidation, customizers);
export const typedMemberFixtures = createTypedFixtures(memberFixtures, memberValidation);
