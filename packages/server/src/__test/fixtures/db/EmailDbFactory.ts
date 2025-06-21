import { generatePK, generatePublicId } from "./idHelpers.js";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
} from "./types.js";

interface EmailRelationConfig extends RelationConfig {
    withUser?: boolean;
    userId?: bigint;
    verified?: boolean;
}

/**
 * Enhanced database fixture factory for Email model
 * Provides comprehensive testing capabilities for email addresses
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for user and team emails
 * - Email verification testing
 * - Verification code management
 * - Case-insensitive email handling
 * - Predefined test scenarios
 */
export class EmailDbFactory extends EnhancedDatabaseFactory<
    any, // Using any temporarily to avoid type issues
    Prisma.emailCreateInput,
    Prisma.emailInclude,
    Prisma.emailUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('email', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.email;
    }

    protected generateMinimalData(overrides?: Partial<Prisma.emailCreateInput>): Prisma.emailCreateInput {
        return {
            id: generatePK(),
            emailAddress: `test_${generatePublicId()}@example.com`,
            user: { connect: { id: generatePK() } },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.emailCreateInput>): Prisma.emailCreateInput {
        return {
            ...this.generateMinimalData(),
            emailAddress: `complete_${generatePublicId()}@example.com`,
            verifiedAt: new Date(),
            verificationCode: `verify_${generatePublicId()}`,
            lastVerificationCodeRequestAttempt: new Date(Date.now() - 60000), // 1 minute ago
            ...overrides,
        };
    }
    /**
     * Get complete test fixtures for Email model
     */
    protected getFixtures(): DbTestFixtures<Prisma.emailCreateInput, Prisma.emailUpdateInput> {
        const userId = generatePK();
        
        return {
            minimal: this.generateMinimalData(),
            complete: this.generateCompleteData(),
            invalid: {
                missingRequired: {
                    // Missing id and emailAddress
                    user: {
                        connect: { id: userId }
                    },
                },
                invalidTypes: {
                    id: 123 as any, // Should be bigint
                    emailAddress: null as any, // Should be string
                    verifiedAt: "not-a-date" as any, // Should be Date
                },
                duplicateEmail: {
                    id: generatePK(),
                    emailAddress: "existing@example.com", // Assumes this exists
                    user: {
                        connect: { id: userId }
                    },
                },
            },
            edgeCases: {
                unverifiedWithCode: {
                    id: generatePK(),
                    emailAddress: `unverified_${nanoid()}@example.com`,
                    verificationCode: `verify_${nanoid()}`,
                    lastVerificationCodeRequestAttempt: new Date(),
                    user: {
                        connect: { id: userId }
                    },
                },
                verifiedNoCode: {
                    id: generatePK(),
                    emailAddress: `verified_${nanoid()}@example.com`,
                    verifiedAt: new Date(),
                    verificationCode: null,
                    user: {
                        connect: { id: userId }
                    },
                },
                teamEmail: {
                    id: generatePK(),
                    emailAddress: `team_${nanoid()}@company.com`,
                    verifiedAt: new Date(),
                    team: {
                        connect: { id: generatePK() }
                    },
                },
                caseInsensitiveEmail: {
                    id: generatePK(),
                    emailAddress: `MiXeD.CaSe_${nanoid()}@ExAmPlE.CoM`,
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
export const createEmailDbFactory = (prisma: PrismaClient) => new EmailDbFactory(prisma);

// Export the class for type usage
export { EmailDbFactory as EmailDbFactoryClass };