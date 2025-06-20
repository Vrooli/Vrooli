import type { WalletUpdateInput } from "../../../api/types.js";
import { type ModelTestFixtures, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";
import { walletValidation } from "../../../validation/models/wallet.js";
import { NAME_MAX_LENGTH } from "../../../validation/utils/validationConstants.js";

// Wallet doesn't support create, so we use a placeholder type
type WalletCreateInput = { id: string };

// Valid Snowflake IDs for testing
const validIds = {
    id1: "400000000000000001",
    id2: "400000000000000002",
    id3: "400000000000000003",
};

// Shared wallet test fixtures
export const walletFixtures: ModelTestFixtures<WalletCreateInput, WalletUpdateInput> = {
    minimal: {
        create: {
            // Wallet doesn't support create
            id: validIds.id1,
        },
        update: {
            id: validIds.id1,
        },
    },
    complete: {
        create: {
            // Wallet doesn't support create
            id: validIds.id2,
        },
        update: {
            id: validIds.id2,
            name: "My Primary Wallet",
        },
    },
    invalid: {
        missingRequired: {
            create: {
                // Not applicable - no create operation
            },
            update: {
                // Missing id
                name: "Wallet Name",
            },
        },
        invalidTypes: {
            create: {
                // Not applicable
            },
            update: {
                id: 123, // Should be string
                name: true, // Should be string
            },
        },
        nameTooShort: {
            update: {
                id: validIds.id1,
                name: "ab", // Less than 3 characters
            },
        },
        nameTooLong: {
            update: {
                id: validIds.id1,
                name: "a".repeat(51), // Exceeds 50 char limit
            },
        },
    },
    edgeCases: {
        minLengthName: {
            update: {
                id: validIds.id1,
                name: "abc", // Exactly 3 characters
            },
        },
        maxLengthName: {
            update: {
                id: validIds.id1,
                name: "a".repeat(NAME_MAX_LENGTH), // Exactly at max length
            },
        },
        nameWithSpecialChars: {
            update: {
                id: validIds.id1,
                name: "Wallet #1 (Primary)",
            },
        },
        nameWithUnicode: {
            update: {
                id: validIds.id1,
                name: "💰 Savings Wallet",
            },
        },
        updateWithoutName: {
            update: {
                id: validIds.id1,
                // name is optional
            },
        },
        emptyStringName: {
            update: {
                id: validIds.id1,
                name: "",
            },
        },
        whitespaceOnlyName: {
            update: {
                id: validIds.id1,
                name: "   ",
            },
        },
    },
};

// Custom factory that always generates valid IDs
const customizers = {
    create: (base: WalletCreateInput): WalletCreateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
    update: (base: WalletUpdateInput): WalletUpdateInput => ({
        ...base,
        id: base.id || validIds.id1,
    }),
};

// Export a factory for creating test data programmatically
export const walletTestDataFactory = new TypedTestDataFactory(walletFixtures, walletValidation, customizers);
export const typedWalletFixtures = createTypedFixtures(walletFixtures, walletValidation);
