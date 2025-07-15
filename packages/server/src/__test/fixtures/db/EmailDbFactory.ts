// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed type safety issues: removed temporary any type parameter, improved invalid type casts
import { nanoid } from "@vrooli/shared";
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
    email,
    Prisma.emailCreateInput,
    Prisma.emailInclude,
    Prisma.emailUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("email", prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.email;
    }

    protected generateMinimalData(overrides?: Partial<Prisma.emailCreateInput>): Prisma.emailCreateInput {
        return {
            id: this.generateId(),
            emailAddress: `test_${this.generatePublicId()}@example.com`,
            user: { connect: { id: this.generateId() } },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.emailCreateInput>): Prisma.emailCreateInput {
        return {
            ...this.generateMinimalData(),
            emailAddress: `complete_${this.generatePublicId()}@example.com`,
            verifiedAt: new Date(),
            verificationCode: `verify_${this.generatePublicId()}`,
            lastVerificationCodeRequestAttempt: new Date(Date.now() - 60000), // 1 minute ago
            ...overrides,
        };
    }
    /**
     * Get complete test fixtures for Email model
     */
    protected getFixtures(): DbTestFixtures<Prisma.emailCreateInput, Prisma.emailUpdateInput> {
        const userId = this.generateId();
        
        return {
            minimal: this.generateMinimalData(),
            complete: this.generateCompleteData(),
            invalid: {
                missingRequired: {
                    // Missing id and emailAddress
                    user: {
                        connect: { id: userId },
                    },
                },
                invalidTypes: {
                    id: 123 as unknown as bigint, // Should be bigint
                    emailAddress: null as unknown as string, // Should be string
                    verifiedAt: "not-a-date" as unknown as Date, // Should be Date
                },
                duplicateEmail: {
                    id: this.generateId(),
                    emailAddress: "existing@example.com", // Assumes this exists
                    user: {
                        connect: { id: userId },
                    },
                },
            },
            edgeCases: {
                unverifiedWithCode: {
                    id: this.generateId(),
                    emailAddress: `unverified_${nanoid()}@example.com`,
                    verificationCode: `verify_${nanoid()}`,
                    lastVerificationCodeRequestAttempt: new Date(),
                    user: {
                        connect: { id: userId },
                    },
                },
                verifiedNoCode: {
                    id: this.generateId(),
                    emailAddress: `verified_${nanoid()}@example.com`,
                    verifiedAt: new Date(),
                    verificationCode: null,
                    user: {
                        connect: { id: userId },
                    },
                },
                teamEmail: {
                    id: this.generateId(),
                    emailAddress: `team_${nanoid()}@company.com`,
                    verifiedAt: new Date(),
                    team: {
                        connect: { id: this.generateId() },
                    },
                },
                caseInsensitiveEmail: {
                    id: this.generateId(),
                    emailAddress: `MiXeD.CaSe_${nanoid()}@ExAmPlE.CoM`,
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
export const createEmailDbFactory = (prisma: PrismaClient) => new EmailDbFactory(prisma);

// Export the class for type usage
export { EmailDbFactory as EmailDbFactoryClass };
