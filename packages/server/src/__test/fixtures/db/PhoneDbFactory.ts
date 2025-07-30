import { type Prisma } from "@prisma/client";
import { generatePK } from "@vrooli/shared";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type {
    DbTestFixtures,
} from "./types.js";

// Constants for phone number generation
const PHONE_NUMBER_MAX = 10000000;
const PHONE_NUMBER_MAX_UK = 1000000000;
const PHONE_NUMBER_PADDING = 7;
const PHONE_NUMBER_PADDING_UK = 9;
const VERIFICATION_CODE_MAX = 1000000;
const VERIFICATION_CODE_PADDING = 6;
const VERIFICATION_DELAY_MS = 60000; // 1 minute
const INVALID_ID_NUMBER = 123;

/**
 * Enhanced database fixture factory for Phone model
 * Provides comprehensive testing capabilities for phone numbers
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for user and team phones
 * - Phone verification testing
 * - Verification code management
 * - International phone number formats
 * - Predefined test scenarios
 */
export class PhoneDbFactory extends EnhancedDbFactory<
    Prisma.phoneCreateInput,
    Prisma.phoneUpdateInput
> {
    /**
     * Get complete test fixtures for Phone model
     */
    protected getFixtures(): DbTestFixtures<Prisma.phoneCreateInput, Prisma.phoneUpdateInput> {
        const userId = generatePK();

        return {
            minimal: {
                id: generatePK(),
                phoneNumber: `+1555${Math.floor(Math.random() * PHONE_NUMBER_MAX).toString().padStart(PHONE_NUMBER_PADDING, "0")}`,
                user: {
                    connect: { id: userId },
                },
            },
            complete: {
                id: generatePK(),
                phoneNumber: `+1555${Math.floor(Math.random() * PHONE_NUMBER_MAX).toString().padStart(PHONE_NUMBER_PADDING, "0")}`,
                verifiedAt: new Date(),
                verificationCode: `${Math.floor(Math.random() * VERIFICATION_CODE_MAX).toString().padStart(VERIFICATION_CODE_PADDING, "0")}`,
                lastVerificationCodeRequestAttempt: new Date(Date.now() - VERIFICATION_DELAY_MS), // 1 minute ago
                user: {
                    connect: { id: userId },
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id and phoneNumber
                    user: {
                        connect: { id: userId },
                    },
                },
                invalidTypes: {
                    id: INVALID_ID_NUMBER as any, // Should be bigint
                    phoneNumber: null as any, // Should be string
                    verifiedAt: "not-a-date" as any, // Should be Date
                },
                invalidFormat: {
                    id: generatePK(),
                    phoneNumber: "not-a-phone-number",
                    user: {
                        connect: { id: userId },
                    },
                },
            },
            edgeCases: {
                unverifiedWithCode: {
                    id: generatePK(),
                    phoneNumber: `+1555${Math.floor(Math.random() * PHONE_NUMBER_MAX).toString().padStart(PHONE_NUMBER_PADDING, "0")}`,
                    verificationCode: "123456",
                    lastVerificationCodeRequestAttempt: new Date(),
                    user: {
                        connect: { id: userId },
                    },
                },
                internationalPhone: {
                    id: generatePK(),
                    phoneNumber: `+44${Math.floor(Math.random() * PHONE_NUMBER_MAX_UK).toString().padStart(PHONE_NUMBER_PADDING_UK, "0")}`, // UK
                    user: {
                        connect: { id: userId },
                    },
                },
                teamPhone: {
                    id: generatePK(),
                    phoneNumber: `+1800${Math.floor(Math.random() * PHONE_NUMBER_MAX).toString().padStart(PHONE_NUMBER_PADDING, "0")}`,
                    verifiedAt: new Date(),
                    team: {
                        connect: { id: generatePK() },
                    },
                },
                shortCode: {
                    id: generatePK(),
                    phoneNumber: "+12345", // Short codes are typically 5-6 digits
                    verifiedAt: new Date(),
                    user: {
                        connect: { id: userId },
                    },
                },
            },
            updates: {
                minimal: {
                    verifiedAt: new Date(),
                },
                complete: {
                    verifiedAt: new Date(),
                    verificationCode: null,
                    lastVerificationCodeRequestAttempt: null,
                },
            },
        };
    }
}

// Export factory creator function
export function createPhoneDbFactory() { return new PhoneDbFactory(); }

// Export the class for type usage
export { PhoneDbFactory as PhoneDbFactoryClass };
