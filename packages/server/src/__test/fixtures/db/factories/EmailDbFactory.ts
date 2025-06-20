import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface EmailRelationConfig extends RelationConfig {
    withUser?: boolean;
    withTeam?: boolean;
    verified?: boolean;
    withVerificationCode?: boolean;
}

/**
 * Database fixture factory for email model
 * Handles email addresses for users and teams with verification support
 */
export class EmailDbFactory extends DatabaseFixtureFactory<
    Prisma.email,
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

    protected getMinimalData(overrides?: Partial<Prisma.emailCreateInput>): Prisma.emailCreateInput {
        const uniqueEmail = `test_${nanoid(8)}@example.com`.toLowerCase();
        
        return {
            id: generatePK(),
            emailAddress: uniqueEmail,
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.emailCreateInput>): Prisma.emailCreateInput {
        const uniqueEmail = `complete_${nanoid(8)}@example.com`.toLowerCase();
        
        return {
            id: generatePK(),
            emailAddress: uniqueEmail,
            verifiedAt: new Date(),
            verificationCode: null, // Cleared after verification
            lastVerificationCodeRequestAttempt: new Date(Date.now() - 24 * 60 * 60 * 1000), // 1 day ago
            ...overrides,
        };
    }

    /**
     * Create verified email
     */
    async createVerified(overrides?: Partial<Prisma.emailCreateInput>): Promise<Prisma.email> {
        return this.createComplete({
            verifiedAt: new Date(),
            verificationCode: null,
            ...overrides,
        });
    }

    /**
     * Create unverified email with pending verification
     */
    async createUnverified(overrides?: Partial<Prisma.emailCreateInput>): Promise<Prisma.email> {
        return this.createMinimal({
            verificationCode: `verify_${nanoid(32)}`,
            lastVerificationCodeRequestAttempt: new Date(),
            verifiedAt: null,
            ...overrides,
        });
    }

    /**
     * Create email with expired verification code
     */
    async createExpiredVerification(overrides?: Partial<Prisma.emailCreateInput>): Promise<Prisma.email> {
        return this.createMinimal({
            verificationCode: `expired_${nanoid(32)}`,
            lastVerificationCodeRequestAttempt: new Date(Date.now() - 25 * 60 * 60 * 1000), // 25 hours ago
            verifiedAt: null,
            ...overrides,
        });
    }

    /**
     * Create team email
     */
    async createTeamEmail(overrides?: Partial<Prisma.emailCreateInput>): Promise<Prisma.email> {
        const teamEmail = `team_${nanoid(8)}@company.com`.toLowerCase();
        
        return this.createVerified({
            emailAddress: teamEmail,
            ...overrides,
        });
    }

    protected getDefaultInclude(): Prisma.emailInclude {
        return {
            user: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                },
            },
            team: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.emailCreateInput,
        config: EmailRelationConfig,
        tx: any
    ): Promise<Prisma.emailCreateInput> {
        let data = { ...baseData };

        // Handle user connection
        if (config.withUser && !data.team) {
            const user = await tx.user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Email Test User",
                    handle: `email_user_${nanoid(8)}`,
                    status: "Unlocked",
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                },
            });
            data.user = { connect: { id: user.id } };
        }

        // Handle team connection
        if (config.withTeam && !data.user) {
            const team = await tx.team.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Email Test Team",
                    handle: `email_team_${nanoid(8)}`,
                    isPrivate: false,
                },
            });
            data.team = { connect: { id: team.id } };
        }

        // Handle verification status
        if (config.verified !== undefined) {
            if (config.verified) {
                data.verifiedAt = new Date();
                data.verificationCode = null;
            } else {
                data.verifiedAt = null;
                data.verificationCode = config.withVerificationCode ? `verify_${nanoid(32)}` : null;
                if (config.withVerificationCode) {
                    data.lastVerificationCodeRequestAttempt = new Date();
                }
            }
        }

        return data;
    }

    /**
     * Create test scenarios
     */
    async createUserWithMultipleEmails(count: number = 3): Promise<Prisma.email[]> {
        const user = await this.prisma.user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Multi-Email User",
                handle: `multiemail_${nanoid(8)}`,
                status: "Unlocked",
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
            },
        });

        const emails = await Promise.all(
            Array.from({ length: count }, async (_, i) => {
                const email = `user_email_${i}_${nanoid(6)}@example.com`.toLowerCase();
                return this.createWithRelations({
                    overrides: {
                        emailAddress: email,
                    },
                    withUser: false, // We'll connect manually
                    verified: i === 0, // First email is verified
                    user: { connect: { id: user.id } },
                });
            }),
        );

        return emails;
    }

    async createProfessionalEmails(): Promise<Prisma.email[]> {
        const domains = ['gmail.com', 'outlook.com', 'company.com', 'university.edu'];
        
        return Promise.all(
            domains.map(domain => {
                const localPart = `professional_${nanoid(6)}`;
                return this.createVerified({
                    emailAddress: `${localPart}@${domain}`.toLowerCase(),
                });
            }),
        );
    }

    protected async checkModelConstraints(record: Prisma.email): Promise<string[]> {
        const violations: string[] = [];
        
        // Check email format
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        if (!emailRegex.test(record.emailAddress)) {
            violations.push('Invalid email format');
        }

        // Check email uniqueness
        const duplicate = await this.prisma.email.findFirst({
            where: { 
                emailAddress: record.emailAddress,
                id: { not: record.id },
            },
        });
        if (duplicate) {
            violations.push('Email address must be unique');
        }

        // Check that email belongs to either user or team, not both
        if (record.userId && record.teamId) {
            violations.push('Email cannot belong to both user and team');
        }

        // Check that email belongs to at least one entity
        if (!record.userId && !record.teamId) {
            violations.push('Email must belong to either a user or team');
        }

        // Check verification code timing
        if (record.verificationCode && record.lastVerificationCodeRequestAttempt) {
            const hoursSinceRequest = (Date.now() - record.lastVerificationCodeRequestAttempt.getTime()) / (1000 * 60 * 60);
            if (hoursSinceRequest > 24) {
                violations.push('Verification code may be expired (>24 hours old)');
            }
        }

        // Check verified emails don't have verification codes
        if (record.verifiedAt && record.verificationCode) {
            violations.push('Verified emails should not have verification codes');
        }

        return violations;
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing id and emailAddress
                verifiedAt: new Date(),
            },
            invalidTypes: {
                id: "not-a-snowflake",
                emailAddress: 123, // Should be string
                verifiedAt: "not-a-date", // Should be Date
                userId: "not-a-bigint", // Should be BigInt
            },
            invalidEmailFormat: {
                id: generatePK(),
                emailAddress: "not-an-email",
                user: { connect: { id: generatePK() } },
            },
            duplicateEmail: {
                id: generatePK(),
                emailAddress: "existing@example.com", // Assumes this exists
                user: { connect: { id: generatePK() } },
            },
            bothUserAndTeam: {
                id: generatePK(),
                emailAddress: `conflict_${nanoid(8)}@example.com`,
                user: { connect: { id: generatePK() } },
                team: { connect: { id: generatePK() } },
            },
            noOwner: {
                id: generatePK(),
                emailAddress: `orphan_${nanoid(8)}@example.com`,
                // No user or team connection
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.emailCreateInput> {
        return {
            maxLengthEmail: {
                ...this.getMinimalData(),
                // Max reasonable email length (320 chars total as per RFC)
                emailAddress: `${'a'.repeat(64)}@${'b'.repeat(63)}.${'c'.repeat(63)}.${'d'.repeat(63)}.${'e'.repeat(61)}.com`.toLowerCase(),
            },
            specialCharsEmail: {
                ...this.getMinimalData(),
                emailAddress: `test.user+tag_${nanoid(4)}@sub-domain.example.com`.toLowerCase(),
            },
            internationalDomain: {
                ...this.getMinimalData(),
                emailAddress: `user_${nanoid(6)}@εταιρεία.ελ`.toLowerCase(), // Greek domain
            },
            caseInsensitiveEmail: {
                ...this.getMinimalData(),
                emailAddress: `TeSt_UsEr_${nanoid(6)}@ExAmPlE.CoM`.toLowerCase(),
            },
            recentVerificationRequest: {
                ...this.getMinimalData(),
                verificationCode: `urgent_${nanoid(32)}`,
                lastVerificationCodeRequestAttempt: new Date(Date.now() - 30 * 1000), // 30 seconds ago
                verifiedAt: null,
            },
            oldUnverifiedEmail: {
                ...this.getMinimalData(),
                emailAddress: `old_unverified_${nanoid(6)}@example.com`.toLowerCase(),
                verificationCode: `old_${nanoid(32)}`,
                lastVerificationCodeRequestAttempt: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000), // 30 days ago
                verifiedAt: null,
            },
            quicklyVerifiedEmail: {
                ...this.getCompleteData(),
                verifiedAt: new Date(),
                lastVerificationCodeRequestAttempt: new Date(Date.now() - 5 * 60 * 1000), // Verified 5 minutes after request
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            user: true,
            team: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.email,
        remainingDepth: number,
        tx: any
    ): Promise<void> {
        // Emails don't have dependent records to delete
        // The user/team relationship is handled by the parent
    }

    /**
     * Helper to generate various email formats
     */
    static generateEmailVariants(baseUser: string): string[] {
        const domains = ['example.com', 'test.org', 'demo.net'];
        const variants: string[] = [];
        
        domains.forEach(domain => {
            variants.push(
                `${baseUser}@${domain}`,
                `${baseUser}+test@${domain}`,
                `${baseUser}.backup@${domain}`,
                `no-reply+${baseUser}@${domain}`,
            );
        });
        
        return variants.map(e => e.toLowerCase());
    }
}

// Export factory creator function
export const createEmailDbFactory = (prisma: PrismaClient) => new EmailDbFactory(prisma);