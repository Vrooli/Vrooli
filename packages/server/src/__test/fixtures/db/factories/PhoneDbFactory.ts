import { generatePK, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface PhoneRelationConfig extends RelationConfig {
    withUser?: boolean;
    withTeam?: boolean;
    verified?: boolean;
    userId?: string | bigint;
    teamId?: string | bigint;
}

/**
 * Database fixture factory for Phone model
 * Handles phone verification for both users and teams
 */
export class PhoneDbFactory extends DatabaseFixtureFactory<
    Prisma.Phone,
    Prisma.PhoneCreateInput,
    Prisma.PhoneInclude,
    Prisma.PhoneUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('Phone', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.phone;
    }

    protected getMinimalData(overrides?: Partial<Prisma.PhoneCreateInput>): Prisma.PhoneCreateInput {
        const uniquePhone = this.generateUniquePhoneNumber();
        
        return {
            id: generatePK(),
            phoneNumber: uniquePhone,
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.PhoneCreateInput>): Prisma.PhoneCreateInput {
        return {
            id: generatePK(),
            phoneNumber: this.generateUniquePhoneNumber(),
            verifiedAt: new Date(),
            verificationCode: this.generateVerificationCode(),
            lastVerificationCodeRequestAttempt: new Date(Date.now() - 5 * 60 * 1000), // 5 minutes ago
            ...overrides,
        };
    }

    protected getDefaultInclude(): Prisma.PhoneInclude {
        return {
            user: {
                select: {
                    id: true,
                    name: true,
                    handle: true,
                },
            },
            team: {
                select: {
                    id: true,
                    name: true,
                    handle: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.PhoneCreateInput,
        config: PhoneRelationConfig,
        tx: any
    ): Promise<Prisma.PhoneCreateInput> {
        let data = { ...baseData };

        // Handle user relationship
        if (config.withUser || config.userId) {
            const userId = config.userId 
                ? (typeof config.userId === 'string' ? BigInt(config.userId) : config.userId)
                : generatePK();
            data.user = {
                connect: { id: userId }
            };
        }

        // Handle team relationship
        if (config.withTeam || config.teamId) {
            const teamId = config.teamId
                ? (typeof config.teamId === 'string' ? BigInt(config.teamId) : config.teamId)
                : generatePK();
            data.team = {
                connect: { id: teamId }
            };
        }

        // Handle verification status
        if (config.verified) {
            data.verifiedAt = new Date();
            data.verificationCode = null; // Clear code after verification
        } else if (config.verified === false) {
            data.verifiedAt = null;
            data.verificationCode = this.generateVerificationCode();
            data.lastVerificationCodeRequestAttempt = new Date();
        }

        return data;
    }

    protected async checkModelConstraints(record: Prisma.Phone): Promise<string[]> {
        const violations: string[] = [];
        
        // Check phone number format
        if (!this.isValidPhoneNumber(record.phoneNumber)) {
            violations.push('Phone number format is invalid');
        }

        // Check phone number uniqueness
        const duplicate = await this.prisma.phone.findFirst({
            where: { 
                phoneNumber: record.phoneNumber,
                id: { not: record.id },
            },
        });
        if (duplicate) {
            violations.push('Phone number must be unique');
        }

        // Check that phone belongs to either user or team, not both
        if (record.userId && record.teamId) {
            violations.push('Phone cannot belong to both user and team');
        }

        // Check that phone belongs to someone
        if (!record.userId && !record.teamId) {
            violations.push('Phone must belong to either a user or team');
        }

        // Check verification code constraints
        if (record.verificationCode && record.verificationCode.length !== 6) {
            violations.push('Verification code must be 6 digits');
        }

        // Check that verified phones don't have verification codes
        if (record.verifiedAt && record.verificationCode) {
            violations.push('Verified phones should not have pending verification codes');
        }

        return violations;
    }

    /**
     * Create a verified phone for a user
     */
    async createVerifiedUserPhone(userId: string | bigint): Promise<Prisma.Phone> {
        return this.createWithRelations({
            userId,
            verified: true,
        });
    }

    /**
     * Create a verified phone for a team
     */
    async createVerifiedTeamPhone(teamId: string | bigint): Promise<Prisma.Phone> {
        return this.createWithRelations({
            teamId,
            verified: true,
        });
    }

    /**
     * Create an unverified phone with pending verification
     */
    async createUnverifiedPhone(ownerId: string | bigint, isTeam: boolean = false): Promise<Prisma.Phone> {
        return this.createWithRelations({
            [isTeam ? 'teamId' : 'userId']: ownerId,
            verified: false,
        });
    }

    /**
     * Create a phone with expired verification attempt
     */
    async createExpiredVerificationPhone(userId: string | bigint): Promise<Prisma.Phone> {
        return this.createMinimal({
            user: { connect: { id: typeof userId === 'string' ? BigInt(userId) : userId } },
            verificationCode: this.generateVerificationCode(),
            lastVerificationCodeRequestAttempt: new Date(Date.now() - 24 * 60 * 60 * 1000), // 24 hours ago
        });
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing phoneNumber
                id: generatePK(),
            },
            invalidTypes: {
                id: "not-a-bigint",
                phoneNumber: 123456789, // Should be string
                verifiedAt: "invalid-date", // Should be Date
                verificationCode: 123456, // Should be string
            },
            invalidPhoneNumber: {
                id: generatePK(),
                phoneNumber: "invalid-phone", // Invalid format
            },
            bothUserAndTeam: {
                id: generatePK(),
                phoneNumber: this.generateUniquePhoneNumber(),
                user: { connect: { id: generatePK() } },
                team: { connect: { id: generatePK() } },
            },
            noOwner: {
                id: generatePK(),
                phoneNumber: this.generateUniquePhoneNumber(),
                // Neither user nor team specified
            },
            invalidVerificationCode: {
                id: generatePK(),
                phoneNumber: this.generateUniquePhoneNumber(),
                verificationCode: "12345", // Too short
            },
            verifiedWithCode: {
                id: generatePK(),
                phoneNumber: this.generateUniquePhoneNumber(),
                verifiedAt: new Date(),
                verificationCode: "123456", // Should be null if verified
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.PhoneCreateInput> {
        return {
            internationalPhone: {
                ...this.getMinimalData(),
                phoneNumber: "+81-90-1234-5678", // Japanese format
            },
            maxLengthPhone: {
                ...this.getMinimalData(),
                phoneNumber: "+1234567890123456", // Maximum length
            },
            recentVerificationAttempt: {
                ...this.getMinimalData(),
                verificationCode: this.generateVerificationCode(),
                lastVerificationCodeRequestAttempt: new Date(Date.now() - 1000), // 1 second ago
            },
            expiredVerification: {
                ...this.getMinimalData(),
                verificationCode: this.generateVerificationCode(),
                lastVerificationCodeRequestAttempt: new Date(Date.now() - 15 * 60 * 1000), // 15 minutes ago
            },
            justVerified: {
                ...this.getMinimalData(),
                verifiedAt: new Date(),
                verificationCode: null,
            },
        };
    }

    // Helper methods

    /**
     * Generate a unique phone number for testing
     */
    private generateUniquePhoneNumber(): string {
        const areaCode = Math.floor(Math.random() * 900) + 100; // 100-999
        const exchange = Math.floor(Math.random() * 900) + 100; // 100-999
        const number = Math.floor(Math.random() * 9000) + 1000; // 1000-9999
        const uniqueSuffix = nanoid(4); // Add uniqueness
        return `+1${areaCode}${exchange}${number.toString().slice(0, 4)}`;
    }

    /**
     * Generate a 6-digit verification code
     */
    private generateVerificationCode(): string {
        return Math.floor(100000 + Math.random() * 900000).toString();
    }

    /**
     * Validate phone number format (basic validation)
     */
    private isValidPhoneNumber(phoneNumber: string): boolean {
        // Basic international phone number validation
        const phoneRegex = /^\+?[1-9]\d{1,14}$/;
        return phoneRegex.test(phoneNumber.replace(/[-\s()]/g, ''));
    }

    protected getCascadeInclude(): any {
        return {
            user: true,
            team: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.Phone,
        remainingDepth: number,
        tx: any
    ): Promise<void> {
        // Phone has no dependent records to delete
        // The phone record itself will be deleted by the caller
    }
}

// Export factory creator function
export const createPhoneDbFactory = (prisma: PrismaClient) => new PhoneDbFactory(prisma);