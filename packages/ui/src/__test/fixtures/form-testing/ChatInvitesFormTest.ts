import {
    chatInviteFormConfig,
    chatInviteValidation,
    createTransformFunction,
    endpointsChatInvite,
    type ChatInviteCreateInput,
    type ChatInviteShape,
    type ChatInviteUpdateInput,
    type Session,
} from "@vrooli/shared";
import { transformChatInviteValues } from "../../../views/objects/chatInvite/ChatInvitesUpsert.js";
import { createUIFormTestFactory, type UIFormTestConfig } from "./UIFormTestFactory.js";

/**
 * Configuration for ChatInvite form testing with data-driven test scenarios
 */
const chatInviteFormTestConfig: UIFormTestConfig<ChatInviteShape, ChatInviteShape, ChatInviteCreateInput, ChatInviteUpdateInput, ChatInviteShape> = {
    // Form metadata
    objectType: "ChatInvite",
    formFixtures: {
        minimal: {
            __typename: "ChatInvite" as const,
            id: "chatinvite_minimal",
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            message: null,
            user: {
                __typename: "User" as const,
                id: "user_123",
                name: "Test User",
                handle: "testuser",
            },
            chat: {
                __typename: "Chat" as const,
                id: "chat_123",
            },
            you: {
                __typename: "ChatInviteYou" as const,
                canDelete: true,
                canUpdate: true,
            },
        },
        complete: {
            __typename: "ChatInvite" as const,
            id: "chatinvite_complete",
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            message: "Welcome to our team chat! We're excited to have you join our discussions.",
            user: {
                __typename: "User" as const,
                id: "user_456",
                name: "John Doe",
                handle: "johndoe",
            },
            chat: {
                __typename: "Chat" as const,
                id: "chat_456",
            },
            you: {
                __typename: "ChatInviteYou" as const,
                canDelete: true,
                canUpdate: true,
            },
        },
        invalid: {
            __typename: "ChatInvite" as const,
            id: "chatinvite_invalid",
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            message: "A".repeat(2049), // Invalid: exceeds character limit
            user: {
                __typename: "User" as const,
                id: "", // Invalid: empty user ID
                name: "",
                handle: "",
            },
            chat: {
                __typename: "Chat" as const,
                id: "", // Invalid: empty chat ID
            },
            you: {
                __typename: "ChatInviteYou" as const,
                canDelete: true,
                canUpdate: true,
            },
        },
        edgeCase: {
            __typename: "ChatInvite" as const,
            id: "chatinvite_edge",
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            message: "A".repeat(2048), // Edge case: at character limit
            user: {
                __typename: "User" as const,
                id: "user_edge",
                name: "Edge Case User",
                handle: "edgeuser",
            },
            chat: {
                __typename: "Chat" as const,
                id: "chat_edge",
            },
            you: {
                __typename: "ChatInviteYou" as const,
                canDelete: true,
                canUpdate: true,
            },
        },
    },

    // Validation schemas from shared package
    validation: chatInviteValidation,

    // API endpoints from shared package
    endpoints: {
        create: endpointsChatInvite.createOne,
        update: endpointsChatInvite.updateOne,
    },

    // Transform functions - form already uses ChatInviteShape, so no transformation needed
    formToShape: (formData: ChatInviteShape) => formData,

    transformFunction: (shape: ChatInviteShape, existing: ChatInviteShape, isCreate: boolean) => {
        const result = transformChatInviteValues(shape, existing, isCreate);
        if (!result) {
            throw new Error("Transform function returned undefined");
        }
        return result;
    },

    initialValuesFunction: (session?: Session, existing?: Partial<ChatInviteShape>): ChatInviteShape => {
        return chatInviteFormConfig.transformations.getInitialValues(session, existing);
    },

    // DATA-DRIVEN TEST SCENARIOS - replaces all custom wrapper methods
    testScenarios: {
        messageValidation: {
            description: "Test invite message validation and length limits",
            testCases: [
                {
                    name: "Valid message",
                    field: "message",
                    value: "Welcome to our chat!",
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
                    value: "Welcome to our team chat! We're excited to have you join our discussions and collaborate with us.",
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

        chatValidation: {
            description: "Test chat reference validation",
            testCases: [
                {
                    name: "Valid chat",
                    data: {
                        chat: {
                            __typename: "Chat",
                            id: "chat_123",
                        },
                    },
                    shouldPass: true,
                },
                {
                    name: "Missing chat ID",
                    data: {
                        chat: {
                            __typename: "Chat",
                            id: "",
                        },
                    },
                    shouldPass: false,
                },
            ],
        },

        inviteWorkflow: {
            description: "Test different invite workflow scenarios",
            testCases: [
                {
                    name: "Invite with personal message",
                    data: {
                        message: "Looking forward to working with you!",
                        user: { __typename: "User", id: "user_123", name: "Test User", handle: "testuser" },
                    },
                    shouldPass: true,
                },
                {
                    name: "Standard invite without message",
                    data: {
                        message: null,
                        user: { __typename: "User", id: "user_456", name: "Another User", handle: "anotheruser" },
                    },
                    shouldPass: true,
                },
                {
                    name: "Bulk invite scenario",
                    data: {
                        message: "Welcome to the team chat!",
                        user: { __typename: "User", id: "user_789", name: "Bulk User", handle: "bulkuser" },
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
export const chatInviteFormTestFactory = createUIFormTestFactory(chatInviteFormTestConfig);

/**
 * Type exports for use in other test files
 */
export { chatInviteFormTestConfig };
