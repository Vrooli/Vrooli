import { generatePK, nanoid } from "@vrooli/shared";
import { type Prisma, type phone } from "@prisma/client";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { 
    DbTestFixtures, 
} from "./types.js";

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
                phoneNumber: `+1555${Math.floor(Math.random() * 10000000).toString().padStart(7, '0')}`,
                user: {
                    connect: { id: userId }
                },
            },
            complete: {
                id: generatePK(),
                phoneNumber: `+1555${Math.floor(Math.random() * 10000000).toString().padStart(7, '0')}`,
                verifiedAt: new Date(),
                verificationCode: `${Math.floor(Math.random() * 1000000).toString().padStart(6, '0')}`,
                lastVerificationCodeRequestAttempt: new Date(Date.now() - 60000), // 1 minute ago
                user: {
                    connect: { id: userId }
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id and phoneNumber
                    user: {
                        connect: { id: userId }
                    },
                },
                invalidTypes: {
                    id: 123 as any, // Should be bigint
                    phoneNumber: null as any, // Should be string
                    verifiedAt: "not-a-date" as any, // Should be Date
                },
                invalidFormat: {
                    id: generatePK(),
                    phoneNumber: "not-a-phone-number",
                    user: {
                        connect: { id: userId }
                    },
                },
            },
            edgeCases: {
                unverifiedWithCode: {
                    id: generatePK(),
                    phoneNumber: `+1555${Math.floor(Math.random() * 10000000).toString().padStart(7, '0')}`,
                    verificationCode: "123456",
                    lastVerificationCodeRequestAttempt: new Date(),
                    user: {
                        connect: { id: userId }
                    },
                },
                internationalPhone: {
                    id: generatePK(),
                    phoneNumber: `+44${Math.floor(Math.random() * 1000000000).toString().padStart(9, '0')}`, // UK
                    user: {
                        connect: { id: userId }
                    },
                },
                teamPhone: {
                    id: generatePK(),
                    phoneNumber: `+1800${Math.floor(Math.random() * 10000000).toString().padStart(7, '0')}`,
                    verifiedAt: new Date(),
                    team: {
                        connect: { id: generatePK() }
                    },
                },
                shortCode: {
                    id: generatePK(),
                    phoneNumber: "+12345", // Short codes are typically 5-6 digits
                    verifiedAt: new Date(),
                    user: {
                        connect: { id: userId }
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
export const createPhoneDbFactory = () => new PhoneDbFactory();

// Export the class for type usage
export { PhoneDbFactory as PhoneDbFactoryClass };