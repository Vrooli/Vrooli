import type { MemberInviteCreateInput, MemberInviteUpdateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TestDataFactory, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { memberInviteValidation } from "../../../validation/models/memberInvite.js";

/**
 * MemberInvite test fixtures with full TypeScript type safety.
 * 
 * Usage examples:
 * 
 * // Using typed fixtures directly:
 * const minimalInvite = memberInviteFixtures.minimal.create;
 * 
 * // Using the factory to create test data:
 * const invite = memberInviteTestDataFactory.createMinimal({
 *     message: "Custom invitation message"
 * });
 * 
 * // Using the factory with validation:
 * const validatedInvite = await memberInviteTestDataFactory.createMinimalValidated({
 *     message: "Validated invitation"
 * });
 */

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    teamId1: "123456789012345681",
    teamId2: "123456789012345682",
    userId1: "123456789012345683",
    userId2: "123456789012345684",
};

// Shared memberInvite test fixtures
export const memberInviteFixtures: ModelTestFixtures<MemberInviteCreateInput, MemberInviteUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            teamConnect: validIds.teamId1,
            userConnect: validIds.userId1,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id2,
            message: "You've been invited to join our team!",
            willBeAdmin: true,
            willHavePermissions: JSON.stringify(["read", "write", "admin"]),
            teamConnect: validIds.teamId2,
            userConnect: validIds.userId2,
        },
        update: {
            id: validIds.id2,
            message: "Updated invitation message",
            willBeAdmin: false,
            willHavePermissions: JSON.stringify(["read"]),
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing id, teamConnect, and userConnect
                message: "Incomplete invite",
            },
            update: {
                // Missing id
                message: "Updated message",
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                message: false, // Should be string
                willBeAdmin: "not-a-boolean", // Should be boolean
                willHavePermissions: 123, // Should be string (JSON)
                teamConnect: 456, // Should be string
                userConnect: 789, // Should be string
            },
            update: {
                id: validIds.id3,
                message: 123, // Should be string
                willBeAdmin: "true", // Should be boolean, not string
                willHavePermissions: false, // Should be string (JSON)
            },
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                teamConnect: validIds.teamId1,
                userConnect: validIds.userId1,
            },
        },
        invalidTeamConnect: {
            create: {
                id: validIds.id1,
                teamConnect: "not-a-valid-snowflake",
                userConnect: validIds.userId1,
            },
        },
        invalidUserConnect: {
            create: {
                id: validIds.id1,
                teamConnect: validIds.teamId1,
                userConnect: "not-a-valid-snowflake",
            },
        },
        missingTeamConnect: {
            create: {
                id: validIds.id1,
                userConnect: validIds.userId1,
                // Missing required teamConnect
            },
        },
        missingUserConnect: {
            create: {
                id: validIds.id1,
                teamConnect: validIds.teamId1,
                // Missing required userConnect
            },
        },
        invalidPermissionsFormat: {
            create: {
                id: validIds.id1,
                willHavePermissions: "not-valid-json", // Should be valid JSON
                teamConnect: validIds.teamId1,
                userConnect: validIds.userId1,
            },
        },
    },
    edgeCases: {
        adminWithoutPermissions: {
            create: {
                id: validIds.id1,
                willBeAdmin: true,
                // No specific permissions set
                teamConnect: validIds.teamId1,
                userConnect: validIds.userId1,
            },
        },
        nonAdminWithPermissions: {
            create: {
                id: validIds.id1,
                willBeAdmin: false,
                willHavePermissions: JSON.stringify(["read", "write"]),
                teamConnect: validIds.teamId1,
                userConnect: validIds.userId1,
            },
        },
        emptyPermissions: {
            create: {
                id: validIds.id1,
                willHavePermissions: JSON.stringify([]),
                teamConnect: validIds.teamId1,
                userConnect: validIds.userId1,
            },
        },
        complexPermissions: {
            create: {
                id: validIds.id1,
                willHavePermissions: JSON.stringify([
                    "read",
                    "write", 
                    "delete",
                    "admin",
                    "moderate",
                    "invite",
                ]),
                teamConnect: validIds.teamId1,
                userConnect: validIds.userId1,
            },
        },
        longMessage: {
            create: {
                id: validIds.id1,
                message: "x".repeat(1000), // Test long message
                teamConnect: validIds.teamId1,
                userConnect: validIds.userId1,
            },
        },
        emptyMessage: {
            create: {
                id: validIds.id1,
                message: "", // Empty string should be removed
                teamConnect: validIds.teamId1,
                userConnect: validIds.userId1,
            },
        },
        whitespaceMessage: {
            create: {
                id: validIds.id1,
                message: "  Invitation message with whitespace  ",
                teamConnect: validIds.teamId1,
                userConnect: validIds.userId1,
            },
        },
        updateWithoutConnections: {
            update: {
                id: validIds.id1,
                message: "Updated message",
                willBeAdmin: true,
                willHavePermissions: JSON.stringify(["admin"]),
                // No team/user connections in update
            },
        },
        booleanEdgeCases: [
            {
                id: validIds.id1,
                willBeAdmin: true,
                teamConnect: validIds.teamId1,
                userConnect: validIds.userId1,
            },
            {
                id: validIds.id2,
                willBeAdmin: false,
                teamConnect: validIds.teamId1,
                userConnect: validIds.userId1,
            },
        ],
        permissionsEdgeCases: [
            {
                id: validIds.id1,
                willHavePermissions: "[]", // Empty array as JSON string
                teamConnect: validIds.teamId1,
                userConnect: validIds.userId1,
            },
            {
                id: validIds.id2,
                willHavePermissions: JSON.stringify(["read"]), // Single permission
                teamConnect: validIds.teamId1,
                userConnect: validIds.userId1,
            },
            {
                id: validIds.id3,
                willHavePermissions: JSON.stringify({ role: "editor" }), // Object permissions
                teamConnect: validIds.teamId1,
                userConnect: validIds.userId1,
            },
        ],
    },
};

// Custom factory that always generates valid IDs
const customizers = {
    create: (base: Partial<MemberInviteCreateInput>): MemberInviteCreateInput => ({
        ...base,
        id: base.id || validIds.id1,
        teamConnect: base.teamConnect || validIds.teamId1,
        userConnect: base.userConnect || validIds.userId1,
    } as MemberInviteCreateInput),
    update: (base: Partial<MemberInviteUpdateInput>): MemberInviteUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    } as MemberInviteUpdateInput),
};

// Export a factory for creating test data programmatically
export const memberInviteTestDataFactory = new TypedTestDataFactory(memberInviteFixtures, memberInviteValidation, customizers);

// Export typed fixtures with validation methods
export const typedMemberInviteFixtures = createTypedFixtures(memberInviteFixtures, memberInviteValidation);

// Maintain backward compatibility with the original TestDataFactory export if needed
export const memberInviteTestDataFactoryLegacy = new TestDataFactory(memberInviteFixtures, customizers);
