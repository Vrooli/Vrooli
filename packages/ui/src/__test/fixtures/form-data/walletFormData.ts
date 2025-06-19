/**
 * Form data fixtures for wallet-related forms
 * These represent data as it appears in form state before submission
 */

import { DUMMY_ID, type WalletShape } from "@vrooli/shared";

/**
 * Wallet update form data
 * Note: Wallets cannot be created through forms - they are created via handshake
 */
export const minimalWalletUpdateFormInput = {
    id: "400000000000000001",
    name: "My Wallet",
};

export const completeWalletUpdateFormInput = {
    id: "400000000000000002",
    name: "My Primary Crypto Wallet",
};

/**
 * Form data for different wallet scenarios
 */
export const walletFormScenarios = {
    personalWallet: {
        id: "400000000000000003",
        name: "Personal Wallet",
    },
    teamWallet: {
        id: "400000000000000004", 
        name: "Team Treasury Wallet",
    },
    stakingWallet: {
        id: "400000000000000005",
        name: "💰 Staking Rewards",
    },
    shortName: {
        id: "400000000000000006",
        name: "BTC", // Minimum 3 characters
    },
    maxLengthName: {
        id: "400000000000000007",
        name: "a".repeat(50), // Maximum 50 characters
    },
    specialCharsName: {
        id: "400000000000000008",
        name: "Wallet #1 (Primary)",
    },
    unicodeName: {
        id: "400000000000000009",
        name: "💎 Diamond Hands Wallet 🚀",
    },
};

/**
 * Form data for wallet name clearing
 */
export const clearWalletNameFormInput = {
    id: "400000000000000010",
    name: undefined, // Clears the name
};

/**
 * Form validation states
 */
export const walletFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            id: "", // Required but empty
            name: "ab", // Too short (less than 3 characters)
        },
        errors: {
            id: "Wallet ID is required",
            name: "Name must be at least 3 characters",
        },
        touched: {
            id: true,
            name: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalWalletUpdateFormInput,
        errors: {},
        touched: {
            id: true,
            name: true,
        },
        isValid: true,
        isSubmitting: false,
    },
    submitting: {
        values: minimalWalletUpdateFormInput,
        errors: {},
        touched: {},
        isValid: true,
        isSubmitting: true,
    },
};

/**
 * Helper function to create wallet form initial values
 */
export const createWalletFormInitialValues = (walletData?: Partial<any>) => ({
    id: walletData?.id || "",
    name: walletData?.name || "",
    ...walletData,
});

/**
 * Helper function to transform form data to API format
 */
export const transformWalletFormToApiInput = (formData: any) => ({
    id: formData.id,
    name: formData.name || null, // Convert empty string to null
});

/**
 * Transform form data to WalletShape for API operations
 */
export const walletFormDataToShape = (formData: any): WalletShape => ({
    __typename: "Wallet",
    id: formData.id || DUMMY_ID,
});

/**
 * Invalid form inputs for testing validation
 */
export const invalidWalletFormInputs = {
    missingId: {
        // @ts-expect-error - Testing invalid input
        id: undefined,
        name: "Test Wallet",
    },
    emptyId: {
        id: "",
        name: "Test Wallet",
    },
    invalidId: {
        id: "invalid-id", // Not a valid snowflake ID
        name: "Test Wallet",
    },
    nameTooShort: {
        id: "400000000000000001",
        name: "ab", // Less than 3 characters
    },
    nameTooLong: {
        id: "400000000000000001",
        name: "a".repeat(51), // More than 50 characters
    },
    emptyStringName: {
        id: "400000000000000001",
        name: "", // Empty string (should be converted to null)
    },
    whitespaceOnlyName: {
        id: "400000000000000001",
        name: "   ", // Only whitespace
    },
};

/**
 * Edge cases for wallet form testing
 */
export const walletFormEdgeCases = {
    nameWithLeadingTrailingSpaces: {
        id: "400000000000000001",
        name: "  My Wallet  ", // Should be trimmed
    },
    nameWithSpecialCharacters: {
        id: "400000000000000001",
        name: "Wallet-1_Test (Primary)",
    },
    nameWithNumbers: {
        id: "400000000000000001",
        name: "Wallet 2024",
    },
    nameWithEmojis: {
        id: "400000000000000001",
        name: "🔐 Secure Wallet 🔒",
    },
    minimalValidName: {
        id: "400000000000000001",
        name: "ABC", // Exactly 3 characters
    },
    maximalValidName: {
        id: "400000000000000001",
        name: "a".repeat(50), // Exactly 50 characters
    },
    updateWithoutName: {
        id: "400000000000000001",
        // name is optional in update
    },
};