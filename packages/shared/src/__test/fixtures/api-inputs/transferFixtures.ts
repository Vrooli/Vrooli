import type { TransferObjectType, TransferRequestReceiveInput, TransferRequestSendInput, TransferUpdateInput } from "../../../api/types.js";

// Extended types for testing that include the id field (which validation expects but API types don't include)
interface TransferRequestSendInputWithId extends TransferRequestSendInput {
    id: string;
}

interface TransferRequestReceiveInputWithId extends TransferRequestReceiveInput {
    id: string;
}
import { type ModelTestFixtures, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { transferValidation, transferRequestSendValidation, transferRequestReceiveValidation } from "../../../validation/models/transfer.js";

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
// Note: Transfer doesn't have a create operation through normal means
// Only update is supported after creation via request send/receive
export const transferFixtures: ModelTestFixtures<never, TransferUpdateInput> = {
    minimal: {
        create: null as never,  // No create operations for transfers
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: null as never,  // No create operations for transfers
        update: {
            id: validIds.id1,
            message: "Updated transfer message",
        },
    },
    invalid: {
        missingRequired: {
            create: null as never,  // No create operations for transfers
            update: {
                // Missing required id field
                message: "Updated message",
            } as TransferUpdateInput,
        },
        invalidTypes: {
            create: null as never,  // No create operations for transfers
            update: {
                id: validIds.id1,
                // Testing invalid type: message should be string
                message: 123, // Should be string
            } as unknown as TransferUpdateInput,
        },
        invalidId: {
            create: null as never,  // No create operations for transfers
            update: {
                // Testing invalid ID format
                id: "invalid-id",
            } as unknown as TransferUpdateInput,
        },
        invalidObjectType: {
            create: null as never,  // No create operations for transfers
        },
        missingObject: {
            create: null as never,  // No create operations for transfers
        },
        missingRecipient: {
            create: null as never,  // No create operations for transfers
        },
        bothRecipients: {
            create: null as never,  // No create operations for transfers
        },
        invalidObjectConnect: {
            create: null as never,  // No create operations for transfers
        },
        invalidToUserConnect: {
            create: null as never,  // No create operations for transfers
        },
        invalidToTeamConnect: {
            create: null as never,  // No create operations for transfers
        },
    },
    edgeCases: {
        transferToUser: {
            create: null as never,  // No create operations for transfers
        },
        transferToTeam: {
            create: null as never,  // No create operations for transfers
        },
        withMessage: {
            create: null as never,  // No create operations for transfers
        },
        withoutMessage: {
            create: null as never,  // No create operations for transfers
        },
        apiTransfer: {
            create: null as never,  // No create operations for transfers
        },
        routineTransfer: {
            create: null as never,  // No create operations for transfers
        },
        projectTransfer: {
            create: null as never,  // No create operations for transfers
        },
        standardTransfer: {
            create: null as never,  // No create operations for transfers
        },
        teamTransfer: {
            create: null as never,  // No create operations for transfers
        },
        longMessage: {
            create: null as never,  // No create operations for transfers
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
            create: null as never,  // No create operations for transfers
        },
        differentUsers: {
            create: null as never,  // No create operations for transfers
        },
        differentTeams: {
            create: null as never,  // No create operations for transfers
        },
        differentObjects: {
            create: null as never,  // No create operations for transfers
        },
    },
};

// Transfer request send fixtures (for testing transferRequestSendValidation)
export const transferRequestSendFixtures: ModelTestFixtures<TransferRequestSendInputWithId, TransferUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            objectType: "Resource" as TransferObjectType,
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
            objectType: "Resource" as TransferObjectType,
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
                // Missing required fields: id, objectType, objectConnect, and recipient
                message: "Incomplete send request",
            } as TransferRequestSendInputWithId,
            update: {
                // Missing required id field
                message: "Updated message",
            } as TransferUpdateInput,
        },
        invalidTypes: {
            create: {
                // Testing invalid types: id should be string
                id: 123, // Should be string
                // Testing invalid types: message should be string
                message: 456, // Should be string
                // Testing invalid enum value
                objectType: "InvalidType", // Invalid enum value
                // Testing invalid types: objectConnect should be string
                objectConnect: 789, // Should be string
                // Testing invalid types: toUserConnect should be string
                toUserConnect: 101112, // Should be string
            } as unknown as TransferRequestSendInputWithId,
            update: {
                id: validIds.id1,
                // Testing invalid type: message should be string
                message: 123, // Should be string
            } as unknown as TransferUpdateInput,
        },
        bothRecipients: {
            create: {
                id: validIds.id1,
                objectType: "Resource" as TransferObjectType,
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
                objectType: "Resource" as TransferObjectType,
                objectConnect: validIds.id2,
                toUserConnect: validIds.id3,
            },
        },
        sendToTeam: {
            create: {
                id: validIds.id1,
                objectType: "Resource" as TransferObjectType,
                objectConnect: validIds.id2,
                toTeamConnect: validIds.id4,
            },
        },
    },
};

// Transfer request receive fixtures (for testing transferRequestReceiveValidation)
export const transferRequestReceiveFixtures: ModelTestFixtures<TransferRequestReceiveInputWithId, TransferUpdateInput> = {
    minimal: {
        create: {
            id: validIds.id1,
            objectType: "Resource" as TransferObjectType,
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
            objectType: "Resource" as TransferObjectType,
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
                // Missing required fields: id, objectType, and objectConnect
                message: "Incomplete receive request",
                toTeamConnect: validIds.id4,
            } as TransferRequestReceiveInputWithId,
            update: {
                // Missing required id field
                message: "Updated message",
            } as TransferUpdateInput,
        },
        invalidTypes: {
            create: {
                // Testing invalid types: id should be string
                id: 123, // Should be string
                // Testing invalid types: message should be string
                message: 456, // Should be string
                // Testing invalid enum value
                objectType: "InvalidType", // Invalid enum value
                // Testing invalid types: objectConnect should be string
                objectConnect: 789, // Should be string
                // Testing invalid types: toTeamConnect should be string
                toTeamConnect: 101112, // Should be string
            } as unknown as TransferRequestReceiveInputWithId,
            update: {
                id: validIds.id1,
                // Testing invalid type: message should be string
                message: 123, // Should be string
            } as unknown as TransferUpdateInput,
        },
    },
    edgeCases: {
        receiveForTeam: {
            create: {
                id: validIds.id1,
                objectType: "Resource" as TransferObjectType,
                objectConnect: validIds.id2,
                toTeamConnect: validIds.id4,
            },
        },
        receiveWithoutTeam: {
            create: {
                id: validIds.id1,
                objectType: "Resource" as TransferObjectType,
                objectConnect: validIds.id2,
                // No toTeamConnect - should be allowed
            },
        },
    },
};

// Custom factory that always generates valid IDs and required fields for transfer updates
const transferUpdateCustomizers = {
    update: (base: TransferUpdateInput): TransferUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Custom factory for transfer request send operations
const transferRequestSendCustomizers = {
    create: (base: TransferRequestSendInputWithId): TransferRequestSendInputWithId => ({
        ...base,
        id: base.id || validIds.id1,
        objectType: base.objectType || ("Resource" as TransferObjectType),
        objectConnect: base.objectConnect || validIds.id2,
        toUserConnect: base.toUserConnect || base.toTeamConnect ? base.toUserConnect : validIds.id3,
    }),
    update: (base: TransferUpdateInput): TransferUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Custom factory for transfer request receive operations
const transferRequestReceiveCustomizers = {
    create: (base: TransferRequestReceiveInputWithId): TransferRequestReceiveInputWithId => ({
        ...base,
        id: base.id || validIds.id1,
        objectType: base.objectType || ("Resource" as TransferObjectType),
        objectConnect: base.objectConnect || validIds.id2,
    }),
    update: (base: TransferUpdateInput): TransferUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export enhanced type-safe factories for creating test data programmatically
export const transferTestDataFactory = new TypedTestDataFactory(transferFixtures, transferValidation, transferUpdateCustomizers);
export const transferRequestSendTestDataFactory = new TypedTestDataFactory(transferRequestSendFixtures, { create: transferRequestSendValidation, update: transferValidation.update }, transferRequestSendCustomizers);
export const transferRequestReceiveTestDataFactory = new TypedTestDataFactory(transferRequestReceiveFixtures, { create: transferRequestReceiveValidation, update: transferValidation.update }, transferRequestReceiveCustomizers);

// Export type-safe fixtures with validation capabilities
export const typedTransferFixtures = createTypedFixtures(transferFixtures, transferValidation);
export const typedTransferRequestSendFixtures = createTypedFixtures(transferRequestSendFixtures, { create: transferRequestSendValidation, update: transferValidation.update });
export const typedTransferRequestReceiveFixtures = createTypedFixtures(transferRequestReceiveFixtures, { create: transferRequestReceiveValidation, update: transferValidation.update });
