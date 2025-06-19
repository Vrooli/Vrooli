import { 
    TransferObjectType, 
    TransferStatus,
    type TransferRequestSendInput,
    type TransferRequestReceiveInput,
    type TransferUpdateInput,
    type TransferDenyInput
} from "@vrooli/shared";

/**
 * Form data fixtures for transfer-related forms
 * 
 * Transfer forms handle ownership transfer requests between users/teams for resources.
 * Key interactions:
 * 1. Transfer Request Send - Requesting to send an object to another user/team
 * 2. Transfer Request Receive - Requesting to receive an object from another user/team
 * 3. Transfer Update - Updating a pending transfer request
 * 4. Transfer Accept/Deny - Responding to incoming transfer requests
 */

// Valid Snowflake IDs for testing (18-19 digit strings)
const validIds = {
    transfer1: "123456789012345678",
    transfer2: "123456789012345679",
    resource1: "234567890123456789",
    resource2: "234567890123456780",
    user1: "345678901234567890",
    user2: "345678901234567891",
    team1: "456789012345678901",
    team2: "456789012345678902",
};

/**
 * Transfer Request Send Form Data
 */
export const minimalTransferRequestSendFormInput: TransferRequestSendInput = {
    objectType: TransferObjectType.Resource,
    objectConnect: validIds.resource1,
    toUserConnect: validIds.user1,
};

export const completeTransferRequestSendFormInput: TransferRequestSendInput = {
    message: "I would like to transfer this resource to you. Please accept ownership.",
    objectType: TransferObjectType.Resource,
    objectConnect: validIds.resource1,
    toUserConnect: validIds.user1,
};

export const transferRequestSendToTeamFormInput: TransferRequestSendInput = {
    message: "Transferring this resource to your team for better management.",
    objectType: TransferObjectType.Resource,
    objectConnect: validIds.resource1,
    toTeamConnect: validIds.team1,
};

/**
 * Transfer Request Receive Form Data
 */
export const minimalTransferRequestReceiveFormInput: TransferRequestReceiveInput = {
    objectType: TransferObjectType.Resource,
    objectConnect: validIds.resource1,
};

export const completeTransferRequestReceiveFormInput: TransferRequestReceiveInput = {
    message: "Our team would like to take ownership of this resource.",
    objectType: TransferObjectType.Resource,
    objectConnect: validIds.resource1,
    toTeamConnect: validIds.team1,
};

export const transferRequestReceiveWithoutTeamFormInput: TransferRequestReceiveInput = {
    message: "I would like to request ownership of this resource.",
    objectType: TransferObjectType.Resource,
    objectConnect: validIds.resource1,
};

/**
 * Transfer Update Form Data
 */
export const minimalTransferUpdateFormInput: TransferUpdateInput = {
    id: validIds.transfer1,
};

export const completeTransferUpdateFormInput: TransferUpdateInput = {
    id: validIds.transfer1,
    message: "Updated message: Please review this transfer request with updated details.",
};

/**
 * Transfer Deny Form Data
 */
export const minimalTransferDenyFormInput: TransferDenyInput = {
    id: validIds.transfer1,
};

export const completeTransferDenyFormInput: TransferDenyInput = {
    id: validIds.transfer1,
    reason: "We cannot accept this transfer at this time due to resource constraints.",
};

/**
 * Transfer Accept Form Data (Simple)
 */
export const transferAcceptFormInput = {
    id: validIds.transfer1,
};

/**
 * Transfer scenarios for different use cases
 */
export const transferFormScenarios = {
    // User-to-user resource transfer
    userToUser: {
        send: {
            message: "Passing this resource to you as agreed.",
            objectType: TransferObjectType.Resource,
            objectConnect: validIds.resource1,
            toUserConnect: validIds.user2,
        } as TransferRequestSendInput,
        receive: {
            message: "I would like to take over this resource.",
            objectType: TransferObjectType.Resource,
            objectConnect: validIds.resource1,
        } as TransferRequestReceiveInput,
    },

    // User-to-team resource transfer
    userToTeam: {
        send: {
            message: "Contributing this resource to your team.",
            objectType: TransferObjectType.Resource,
            objectConnect: validIds.resource1,
            toTeamConnect: validIds.team1,
        } as TransferRequestSendInput,
        receive: {
            message: "Our team wants to adopt this resource.",
            objectType: TransferObjectType.Resource,
            objectConnect: validIds.resource1,
            toTeamConnect: validIds.team1,
        } as TransferRequestReceiveInput,
    },

    // Team-to-user resource transfer
    teamToUser: {
        send: {
            message: "Transferring resource ownership to individual maintainer.",
            objectType: TransferObjectType.Resource,
            objectConnect: validIds.resource1,
            toUserConnect: validIds.user1,
        } as TransferRequestSendInput,
        receive: {
            message: "I would like to maintain this resource individually.",
            objectType: TransferObjectType.Resource,
            objectConnect: validIds.resource1,
        } as TransferRequestReceiveInput,
    },

    // Emergency transfer (no message)
    emergency: {
        send: {
            objectType: TransferObjectType.Resource,
            objectConnect: validIds.resource1,
            toUserConnect: validIds.user1,
        } as TransferRequestSendInput,
        update: {
            id: validIds.transfer1,
            message: "Emergency transfer - please accept immediately.",
        } as TransferUpdateInput,
    },
};

/**
 * Transfer form validation test cases
 */
export const invalidTransferFormInputs = {
    // Transfer Request Send - missing required fields
    sendMissingObjectType: {
        // @ts-expect-error - Testing invalid input
        objectType: undefined,
        objectConnect: validIds.resource1,
        toUserConnect: validIds.user1,
    },
    sendMissingObjectConnect: {
        objectType: TransferObjectType.Resource,
        // @ts-expect-error - Testing invalid input
        objectConnect: undefined,
        toUserConnect: validIds.user1,
    },
    sendMissingRecipient: {
        objectType: TransferObjectType.Resource,
        objectConnect: validIds.resource1,
        // No toUserConnect or toTeamConnect
    },
    sendBothRecipients: {
        objectType: TransferObjectType.Resource,
        objectConnect: validIds.resource1,
        toUserConnect: validIds.user1,
        toTeamConnect: validIds.team1, // Should only have one recipient
    },
    sendInvalidObjectId: {
        objectType: TransferObjectType.Resource,
        objectConnect: "invalid-id", // Invalid snowflake ID
        toUserConnect: validIds.user1,
    },
    sendInvalidUserId: {
        objectType: TransferObjectType.Resource,
        objectConnect: validIds.resource1,
        toUserConnect: "invalid-user-id", // Invalid snowflake ID
    },
    sendInvalidTeamId: {
        objectType: TransferObjectType.Resource,
        objectConnect: validIds.resource1,
        toTeamConnect: "invalid-team-id", // Invalid snowflake ID
    },

    // Transfer Request Receive - missing required fields
    receiveMissingObjectType: {
        // @ts-expect-error - Testing invalid input
        objectType: undefined,
        objectConnect: validIds.resource1,
    },
    receiveMissingObjectConnect: {
        objectType: TransferObjectType.Resource,
        // @ts-expect-error - Testing invalid input
        objectConnect: undefined,
    },
    receiveInvalidObjectId: {
        objectType: TransferObjectType.Resource,
        objectConnect: "invalid-id", // Invalid snowflake ID
    },
    receiveInvalidTeamId: {
        objectType: TransferObjectType.Resource,
        objectConnect: validIds.resource1,
        toTeamConnect: "invalid-team-id", // Invalid snowflake ID
    },

    // Transfer Update - missing required fields
    updateMissingId: {
        // @ts-expect-error - Testing invalid input
        id: undefined,
        message: "Updated message",
    },
    updateInvalidId: {
        id: "invalid-transfer-id", // Invalid snowflake ID
        message: "Updated message",
    },

    // Transfer Deny - missing required fields
    denyMissingId: {
        // @ts-expect-error - Testing invalid input
        id: undefined,
        reason: "Cannot accept",
    },
    denyInvalidId: {
        id: "invalid-transfer-id", // Invalid snowflake ID
        reason: "Cannot accept",
    },
};

/**
 * Form state fixtures for UI testing
 */
export const transferFormStates = {
    // Initial empty state
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
        isDirty: false,
    },

    // State with validation errors
    withErrors: {
        values: {
            objectType: "",
            objectConnect: "",
            toUserConnect: "",
            message: "",
        },
        errors: {
            objectType: "Object type is required",
            objectConnect: "Object to transfer is required",
            toUserConnect: "Recipient is required",
        },
        touched: {
            objectType: true,
            objectConnect: true,
            toUserConnect: true,
        },
        isValid: false,
        isSubmitting: false,
        isDirty: true,
    },

    // Valid form ready for submission
    valid: {
        values: completeTransferRequestSendFormInput,
        errors: {},
        touched: {
            objectType: true,
            objectConnect: true,
            toUserConnect: true,
            message: true,
        },
        isValid: true,
        isSubmitting: false,
        isDirty: true,
    },

    // Form being submitted
    submitting: {
        values: completeTransferRequestSendFormInput,
        errors: {},
        touched: {
            objectType: true,
            objectConnect: true,
            toUserConnect: true,
            message: true,
        },
        isValid: true,
        isSubmitting: true,
        isDirty: true,
    },
};

/**
 * Mock data for transfer-related UI components
 */
export const mockTransferData = {
    // Existing transfer objects for display
    pendingTransfer: {
        id: validIds.transfer1,
        status: TransferStatus.Pending,
        message: "Please accept this resource transfer",
        objectType: TransferObjectType.Resource,
        objectConnect: validIds.resource1,
        fromOwner: { id: validIds.user1, name: "John Doe" },
        toOwner: { id: validIds.user2, name: "Jane Smith" },
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
    },
    
    acceptedTransfer: {
        id: validIds.transfer2,
        status: TransferStatus.Accepted,
        message: "Resource transfer completed successfully",
        objectType: TransferObjectType.Resource,
        objectConnect: validIds.resource2,
        fromOwner: { id: validIds.user1, name: "John Doe" },
        toOwner: { id: validIds.team1, name: "Development Team" },
        createdAt: new Date(Date.now() - 86400000).toISOString(), // 1 day ago
        updatedAt: new Date().toISOString(),
        closedAt: new Date().toISOString(),
    },

    // Resource objects for transfer
    transferableResource: {
        id: validIds.resource1,
        name: "API Documentation",
        description: "Comprehensive API documentation for the platform",
        resourceType: "Document",
        owner: { id: validIds.user1, name: "John Doe" },
        canTransfer: true,
    },

    // Potential recipients
    potentialRecipients: {
        users: [
            { id: validIds.user1, name: "John Doe", handle: "johndoe" },
            { id: validIds.user2, name: "Jane Smith", handle: "janesmith" },
        ],
        teams: [
            { id: validIds.team1, name: "Development Team", handle: "dev-team" },
            { id: validIds.team2, name: "Documentation Team", handle: "docs-team" },
        ],
    },
};

/**
 * Helper functions for form data transformation
 */
export const transferFormHelpers = {
    /**
     * Validates transfer request send input
     */
    validateSendRequest: (input: Partial<TransferRequestSendInput>): string[] => {
        const errors: string[] = [];
        
        if (!input.objectType) errors.push("Object type is required");
        if (!input.objectConnect) errors.push("Object to transfer is required");
        if (!input.toUserConnect && !input.toTeamConnect) {
            errors.push("Recipient (user or team) is required");
        }
        if (input.toUserConnect && input.toTeamConnect) {
            errors.push("Cannot specify both user and team recipients");
        }
        
        return errors;
    },

    /**
     * Validates transfer request receive input
     */
    validateReceiveRequest: (input: Partial<TransferRequestReceiveInput>): string[] => {
        const errors: string[] = [];
        
        if (!input.objectType) errors.push("Object type is required");
        if (!input.objectConnect) errors.push("Object to receive is required");
        
        return errors;
    },

    /**
     * Creates form initial values from existing transfer data
     */
    createUpdateFormInitialValues: (transferData?: Partial<any>) => ({
        id: transferData?.id || "",
        message: transferData?.message || "",
        ...transferData,
    }),

    /**
     * Transforms form data for API submission
     */
    transformSendRequestForApi: (formData: TransferRequestSendInput) => ({
        ...formData,
        // Ensure only one recipient is set
        toUserConnect: formData.toTeamConnect ? undefined : formData.toUserConnect,
        toTeamConnect: formData.toUserConnect ? undefined : formData.toTeamConnect,
    }),
};