import {
    memberInviteFormConfig,
    memberInviteValidation,
    type MemberInviteCreateInput,
    type MemberInviteShape,
    type MemberInviteUpdateInput,
    type Session,
} from "@vrooli/shared";
import { transformMemberInviteValues } from "../../../views/objects/memberInvite/MemberInvitesUpsert.js";
import { createUIFormTestFactory, type UIFormTestConfig } from "./UIFormTestFactory.js";

/**
 * Configuration for MemberInvite form testing with data-driven test scenarios
 */
const memberInviteFormTestConfig: UIFormTestConfig<MemberInviteShape, MemberInviteShape, MemberInviteCreateInput, MemberInviteUpdateInput, MemberInviteShape> = {
    // Form metadata
    objectType: "MemberInvite",
    formFixtures: {
        minimal: {
            __typename: "MemberInvite" as const,
            id: "memberinvite_minimal",
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            message: null,
            willBeAdmin: false,
            willHavePermissions: [],
            user: {
                __typename: "User" as const,
                id: "user_123",
                name: "Test User",
                handle: "testuser",
            },
            team: {
                __typename: "Team" as const,
                id: "team_123",
            },
            you: {
                __typename: "MemberInviteYou" as const,
                canDelete: true,
                canUpdate: true,
            },
        },
        complete: {
            __typename: "MemberInvite" as const,
            id: "memberinvite_complete",
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            message: "Welcome to our team! We're excited to have you contribute to our projects.",
            willBeAdmin: true,
            willHavePermissions: ["Create", "Update", "Delete"],
            user: {
                __typename: "User" as const,
                id: "user_456",
                name: "John Doe",
                handle: "johndoe",
            },
            team: {
                __typename: "Team" as const,
                id: "team_456",
            },
            you: {
                __typename: "MemberInviteYou" as const,
                canDelete: true,
                canUpdate: true,
            },
        },
        invalid: {
            __typename: "MemberInvite" as const,
            id: "memberinvite_invalid",
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            message: "A".repeat(2049), // Invalid: exceeds character limit
            willBeAdmin: false,
            willHavePermissions: [],
            user: {
                __typename: "User" as const,
                id: "", // Invalid: empty user ID
                name: "",
                handle: "",
            },
            team: {
                __typename: "Team" as const,
                id: "", // Invalid: empty team ID
            },
            you: {
                __typename: "MemberInviteYou" as const,
                canDelete: true,
                canUpdate: true,
            },
        },
        edgeCase: {
            __typename: "MemberInvite" as const,
            id: "memberinvite_edge",
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            message: "A".repeat(2048), // Edge case: at character limit
            willBeAdmin: false,
            willHavePermissions: ["Create", "Read", "Update", "Delete", "Comment", "React"],
            user: {
                __typename: "User" as const,
                id: "user_edge",
                name: "Edge Case User",
                handle: "edgeuser",
            },
            team: {
                __typename: "Team" as const,
                id: "team_edge",
            },
            you: {
                __typename: "MemberInviteYou" as const,
                canDelete: true,
                canUpdate: true,
            },
        },
    },

    // Validation schemas from shared package
    validation: memberInviteValidation,

    // API endpoints from shared package
    endpoints: {
        create: memberInviteFormConfig.endpoints.createOne,
        update: memberInviteFormConfig.endpoints.updateOne,
    },

    // Transform functions - form already uses MemberInviteShape, so no transformation needed
    formToShape: (formData: MemberInviteShape) => formData,

    transformFunction: (shape: MemberInviteShape, existing: MemberInviteShape, isCreate: boolean) => {
        const result = transformMemberInviteValues(shape, existing, isCreate);
        if (!result) {
            throw new Error("Transform function returned undefined");
        }
        return result;
    },

    initialValuesFunction: (session?: Session, existing?: Partial<MemberInviteShape>): MemberInviteShape => {
        return memberInviteFormConfig.transformations.getInitialValues(session, existing);
    },

    // DATA-DRIVEN TEST SCENARIOS - replaces all custom wrapper methods
    testScenarios: {
        messageValidation: {
            description: "Test invite message validation and length limits",
            testCases: [
                {
                    name: "Valid message",
                    field: "message",
                    value: "Welcome to our team!",
                    shouldPass: true,
                },
                {
                    name: "Empty message",
                    field: "message",
                    value: null,
                    shouldPass: true, // Message is optional
                },
                {
                    name: "Long valid message",
                    field: "message",
                    value: "Welcome to our team! We're excited to have you contribute to our projects and collaborate with us.",
                    shouldPass: true,
                },
                {
                    name: "At character limit",
                    field: "message",
                    value: "A".repeat(2048),
                    shouldPass: true,
                },
                {
                    name: "Over character limit",
                    field: "message",
                    value: "A".repeat(2049),
                    shouldPass: false,
                },
            ],
        },

        adminRights: {
            description: "Test admin privileges assignment",
            testCases: [
                {
                    name: "Regular member",
                    data: { willBeAdmin: false },
                    shouldPass: true,
                },
                {
                    name: "Admin member",
                    data: { willBeAdmin: true },
                    shouldPass: true,
                },
            ],
        },

        permissions: {
            description: "Test permission assignment validation",
            testCases: [
                {
                    name: "No permissions",
                    data: { willHavePermissions: [] },
                    shouldPass: true,
                },
                {
                    name: "Read only",
                    data: { willHavePermissions: ["Read"] },
                    shouldPass: true,
                },
                {
                    name: "Standard permissions",
                    data: { willHavePermissions: ["Create", "Read", "Update"] },
                    shouldPass: true,
                },
                {
                    name: "Full permissions",
                    data: { willHavePermissions: ["Create", "Read", "Update", "Delete", "Comment", "React"] },
                    shouldPass: true,
                },
            ],
        },

        userValidation: {
            description: "Test user reference validation",
            testCases: [
                {
                    name: "Valid user",
                    data: {
                        user: {
                            __typename: "User",
                            id: "user_123",
                            name: "Test User",
                            handle: "testuser",
                        },
                    },
                    shouldPass: true,
                },
                {
                    name: "Missing user ID",
                    data: {
                        user: {
                            __typename: "User",
                            id: "",
                            name: "Test User",
                            handle: "testuser",
                        },
                    },
                    shouldPass: false,
                },
            ],
        },

        teamValidation: {
            description: "Test team reference validation",
            testCases: [
                {
                    name: "Valid team",
                    data: {
                        team: {
                            __typename: "Team",
                            id: "team_123",
                        },
                    },
                    shouldPass: true,
                },
                {
                    name: "Missing team ID",
                    data: {
                        team: {
                            __typename: "Team",
                            id: "",
                        },
                    },
                    shouldPass: false,
                },
            ],
        },

        inviteWorkflow: {
            description: "Test different member invite scenarios",
            testCases: [
                {
                    name: "Admin invite with full permissions",
                    data: {
                        willBeAdmin: true,
                        willHavePermissions: ["Create", "Read", "Update", "Delete"],
                        message: "Welcome as an admin!",
                    },
                    shouldPass: true,
                },
                {
                    name: "Contributor invite",
                    data: {
                        willBeAdmin: false,
                        willHavePermissions: ["Create", "Read", "Update"],
                        message: "Looking forward to your contributions!",
                    },
                    shouldPass: true,
                },
                {
                    name: "Observer invite",
                    data: {
                        willBeAdmin: false,
                        willHavePermissions: ["Read"],
                        message: "Welcome as an observer!",
                    },
                    shouldPass: true,
                },
            ],
        },
    },
};

/**
 * SIMPLIFIED: Direct factory export - no wrapper function needed!
 */
export const memberInviteFormTestFactory = createUIFormTestFactory(memberInviteFormTestConfig);

/**
 * Type exports for use in other test files
 */
export { memberInviteFormTestConfig };
