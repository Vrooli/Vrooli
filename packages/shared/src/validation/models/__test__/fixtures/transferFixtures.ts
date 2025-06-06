import { type ModelTestFixtures, TestDataFactory } from "../validationTestUtils.js";

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    id1: "123456789012345678",
    id2: "123456789012345679",
    id3: "123456789012345680",
    id4: "123456789012345681",
    id5: "123456789012345682",
    id6: "123456789012345683",
};

// Shared transfer test fixtures
export const transferFixtures: ModelTestFixtures = {
    minimal: {
        create: {
            id: validIds.id1,
            objectType: "Resource",
            objectConnect: validIds.id2,
            toUserConnect: validIds.id3,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id1,
            message: "I would like to transfer this API to you.",
            objectType: "Resource",
            objectConnect: validIds.id2,
            toTeamConnect: validIds.id4,
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 123,
            unknownField3: true,
        },
        update: {
            id: validIds.id1,
            message: "Updated transfer message",
            // Add some extra fields that will be stripped
            unknownField1: "should be stripped",
            unknownField2: 456,
        },
    },
    invalid: {
        missingRequired: {
            create: {
                id: validIds.id1,
                // Missing required objectType, objectConnect, and recipient
                message: "Incomplete transfer",
            },
            update: {
                // Missing required id
                message: "Updated message",
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                message: 456, // Should be string
                objectType: "InvalidType", // Invalid enum value
                objectConnect: 789, // Should be string
                toUserConnect: 101112, // Should be string
            },
            update: {
                id: validIds.id1,
                message: 123, // Should be string
            },
        },
        invalidId: {
            create: {
                id: "not-a-valid-snowflake",
                objectType: "Resource",
                objectConnect: validIds.id2,
                toUserConnect: validIds.id3,
            },
            update: {
                id: "invalid-id",
            },
        },
        invalidObjectType: {
            create: {
                id: validIds.id1,
                objectType: "UnknownType", // Not a valid enum value
                objectConnect: validIds.id2,
                toUserConnect: validIds.id3,
            },
        },
        missingObject: {
            create: {
                id: validIds.id1,
                objectType: "Resource",
                // Missing required objectConnect
                toUserConnect: validIds.id3,
            },
        },
        missingRecipient: {
            create: {
                id: validIds.id1,
                objectType: "Resource",
                objectConnect: validIds.id2,
                // Missing both toUserConnect and toTeamConnect
            },
        },
        bothRecipients: {
            create: {
                id: validIds.id1,
                objectType: "Resource",
                objectConnect: validIds.id2,
                toUserConnect: validIds.id3,
                toTeamConnect: validIds.id4, // Should only have one recipient
            },
        },
        invalidObjectConnect: {
            create: {
                id: validIds.id1,
                objectType: "Resource",
                objectConnect: "invalid-object-id",
                toUserConnect: validIds.id3,
            },
        },
        invalidToUserConnect: {
            create: {
                id: validIds.id1,
                objectType: "Resource",
                objectConnect: validIds.id2,
                toUserConnect: "invalid-user-id",
            },
        },
        invalidToTeamConnect: {
            create: {
                id: validIds.id1,
                objectType: "Resource",
                objectConnect: validIds.id2,
                toTeamConnect: "invalid-team-id",
            },
        },
    },
    edgeCases: {
        transferToUser: {
            create: {
                id: validIds.id1,
                objectType: "Resource",
                objectConnect: validIds.id2,
                toUserConnect: validIds.id3,
                // No message
            },
        },
        transferToTeam: {
            create: {
                id: validIds.id1,
                objectType: "Resource",
                objectConnect: validIds.id2,
                toTeamConnect: validIds.id4,
                // No message
            },
        },
        withMessage: {
            create: {
                id: validIds.id1,
                message: "Please accept this transfer. It includes important work.",
                objectType: "Resource",
                objectConnect: validIds.id2,
                toUserConnect: validIds.id3,
            },
        },
        withoutMessage: {
            create: {
                id: validIds.id1,
                objectType: "Resource",
                objectConnect: validIds.id2,
                toUserConnect: validIds.id3,
                // No message field
            },
        },
        apiTransfer: {
            create: {
                id: validIds.id1,
                objectType: "Resource",
                objectConnect: validIds.id2,
                toUserConnect: validIds.id3,
                message: "Transferring API ownership",
            },
        },
        routineTransfer: {
            create: {
                id: validIds.id1,
                objectType: "Resource",
                objectConnect: validIds.id2,
                toTeamConnect: validIds.id4,
                message: "Transferring routine to team",
            },
        },
        projectTransfer: {
            create: {
                id: validIds.id1,
                objectType: "Resource",
                objectConnect: validIds.id2,
                toUserConnect: validIds.id3,
                message: "Project ownership transfer",
            },
        },
        standardTransfer: {
            create: {
                id: validIds.id1,
                objectType: "Resource",
                objectConnect: validIds.id2,
                toTeamConnect: validIds.id4,
                message: "Standard transfer to team",
            },
        },
        teamTransfer: {
            create: {
                id: validIds.id1,
                objectType: "Resource",
                objectConnect: validIds.id2,
                toUserConnect: validIds.id3,
                message: "Team ownership transfer",
            },
        },
        longMessage: {
            create: {
                id: validIds.id1,
                message: "This is a very long message explaining the reasons for this transfer. The object being transferred contains important work that would benefit from being managed by the recipient. Please review the contents carefully before accepting.",
                objectType: "Resource",
                objectConnect: validIds.id2,
                toUserConnect: validIds.id3,
            },
        },
        updateWithMessage: {
            update: {
                id: validIds.id1,
                message: "Updated transfer message with new details",
            },
        },
        updateWithoutMessage: {
            update: {
                id: validIds.id1,
                // Only ID, no message
            },
        },
        differentObjectTypes: {
            create: {
                id: validIds.id1,
                objectType: "Resource",
                objectConnect: validIds.id2,
                toUserConnect: validIds.id3,
            },
        },
        differentUsers: {
            create: {
                id: validIds.id1,
                objectType: "Resource",
                objectConnect: validIds.id2,
                toUserConnect: validIds.id6, // Different user
            },
        },
        differentTeams: {
            create: {
                id: validIds.id1,
                objectType: "Resource",
                objectConnect: validIds.id2,
                toTeamConnect: validIds.id6, // Different team
            },
        },
        differentObjects: {
            create: {
                id: validIds.id1,
                objectType: "Resource",
                objectConnect: validIds.id6, // Different object
                toUserConnect: validIds.id3,
            },
        },
    },
};

// Transfer request send fixtures (for testing transferRequestSendValidation)
export const transferRequestSendFixtures: ModelTestFixtures = {
    minimal: {
        create: {
            id: validIds.id1,
            objectType: "Resource",
            objectConnect: validIds.id2,
            toUserConnect: validIds.id3,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id1,
            message: "Sending transfer request",
            objectType: "Resource",
            objectConnect: validIds.id2,
            toTeamConnect: validIds.id4,
        },
        update: {
            id: validIds.id1,
            message: "Updated send request",
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required id, objectType, objectConnect, and recipient
                message: "Incomplete send request",
            },
            update: {
                // Missing required id
                message: "Updated message",
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                message: 456, // Should be string
                objectType: "InvalidType", // Invalid enum value
                objectConnect: 789, // Should be string
                toUserConnect: 101112, // Should be string
            },
            update: {
                id: validIds.id1,
                message: 123, // Should be string
            },
        },
        bothRecipients: {
            create: {
                id: validIds.id1,
                objectType: "Resource",
                objectConnect: validIds.id2,
                toUserConnect: validIds.id3,
                toTeamConnect: validIds.id4, // Violates exclusivity rule
            },
        },
    },
    edgeCases: {
        sendToUser: {
            create: {
                id: validIds.id1,
                objectType: "Resource",
                objectConnect: validIds.id2,
                toUserConnect: validIds.id3,
            },
        },
        sendToTeam: {
            create: {
                id: validIds.id1,
                objectType: "Resource",
                objectConnect: validIds.id2,
                toTeamConnect: validIds.id4,
            },
        },
    },
};

// Transfer request receive fixtures (for testing transferRequestReceiveValidation)
export const transferRequestReceiveFixtures: ModelTestFixtures = {
    minimal: {
        create: {
            id: validIds.id1,
            objectType: "Resource",
            objectConnect: validIds.id2,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            id: validIds.id1,
            message: "Receiving transfer request",
            objectType: "Resource",
            objectConnect: validIds.id2,
            toTeamConnect: validIds.id4,
        },
        update: {
            id: validIds.id1,
            message: "Updated receive request",
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required id, objectType, and objectConnect
                message: "Incomplete receive request",
                toTeamConnect: validIds.id4,
            },
            update: {
                // Missing required id
                message: "Updated message",
            },
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                message: 456, // Should be string
                objectType: "InvalidType", // Invalid enum value
                objectConnect: 789, // Should be string
                toTeamConnect: 101112, // Should be string
            },
            update: {
                id: validIds.id1,
                message: 123, // Should be string
            },
        },
    },
    edgeCases: {
        receiveForTeam: {
            create: {
                id: validIds.id1,
                objectType: "Resource",
                objectConnect: validIds.id2,
                toTeamConnect: validIds.id4,
            },
        },
        receiveWithoutTeam: {
            create: {
                id: validIds.id1,
                objectType: "Resource",
                objectConnect: validIds.id2,
                // No toTeamConnect - should be allowed
            },
        },
    },
};

// Custom factory that always generates valid IDs and required fields
const customizers = {
    create: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
        objectType: base.objectType || "Resource",
        objectConnect: base.objectConnect || validIds.id2,
        toUserConnect: base.toUserConnect || validIds.id3,
    }),
    update: (base: any) => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export factories for creating test data programmatically
export const transferTestDataFactory = new TestDataFactory(transferFixtures, customizers);
export const transferRequestSendTestDataFactory = new TestDataFactory(transferRequestSendFixtures, customizers);
export const transferRequestReceiveTestDataFactory = new TestDataFactory(transferRequestReceiveFixtures, customizers);