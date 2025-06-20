import { generatePK } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
} from "./types.js";

interface CreditAccountRelationConfig extends RelationConfig {
    withUser?: boolean;
    withTeam?: boolean;
    userId?: string;
    teamId?: string;
    initialBalance?: bigint;
}

/**
 * Enhanced database fixture factory for Credit Account model
 * Provides comprehensive testing capabilities for credit accounts
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for user and team credit accounts
 * - Balance testing scenarios
 * - Credit transaction testing
 * - Predefined test scenarios
 */
export class CreditAccountDbFactory extends EnhancedDatabaseFactory<
    Prisma.credit_account,
    Prisma.credit_accountCreateInput,
    Prisma.credit_accountInclude,
    Prisma.credit_accountUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('credit_account', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.credit_account;
    }

    /**
     * Get complete test fixtures for Credit Account model
     */
    protected getFixtures(): DbTestFixtures<Prisma.credit_accountCreateInput, Prisma.credit_accountUpdateInput> {
        const userId = generatePK();
        const teamId = generatePK();
        
        return {
            minimal: {
                id: generatePK(),
                currentBalance: 1000n,
                user: { connect: { id: userId } },
            },
            
            complete: {
                id: generatePK(),
                currentBalance: 100000n,
                user: { connect: { id: userId } },
            },
            
            variants: {
                zeroBalance: {
                    id: generatePK(),
                    currentBalance: 0n,
                    user: { connect: { id: userId } },
                },
                
                highBalance: {
                    id: generatePK(),
                    currentBalance: 10000000n, // 10 million credits
                    user: { connect: { id: userId } },
                },
                
                teamAccount: {
                    id: generatePK(),
                    currentBalance: 50000n,
                    team: { connect: { id: teamId } },
                },
                
                userWithLowBalance: {
                    id: generatePK(),
                    currentBalance: 10n,
                    user: { connect: { id: userId } },
                },
            },
            
            invalid: {
                missingOwner: {
                    id: generatePK(),
                    currentBalance: 1000n,
                    // Missing user or team connection
                },
                
                negativeBalance: {
                    id: generatePK(),
                    currentBalance: -1000n,
                    user: { connect: { id: userId } },
                },
            },
            
            edgeCase: {
                maxBalance: {
                    id: generatePK(),
                    currentBalance: 9223372036854775807n, // Max bigint value
                    user: { connect: { id: userId } },
                },
                
                bothUserAndTeam: {
                    id: generatePK(),
                    currentBalance: 1000n,
                    user: { connect: { id: userId } },
                    team: { connect: { id: teamId } },
                },
            },
            
            update: {
                addCredits: {
                    currentBalance: 5000n,
                },
                
                subtractCredits: {
                    currentBalance: 500n,
                },
                
                zeroOut: {
                    currentBalance: 0n,
                },
            },
        };
    }

    /**
     * Create credit account with specific balance
     */
    async createWithBalance(balance: bigint, overrides?: Partial<Prisma.credit_accountCreateInput>) {
        const data = {
            ...this.getFixtures().minimal,
            currentBalance: balance,
            ...overrides,
        };
        
        return this.createMinimal(data);
    }

    /**
     * Create credit account for specific user
     */
    async createForUser(userId: string, balance = 1000n, overrides?: Partial<Prisma.credit_accountCreateInput>) {
        const data = {
            ...this.getFixtures().minimal,
            currentBalance: balance,
            user: { connect: { id: userId } },
            ...overrides,
        };
        
        return this.createMinimal(data);
    }

    /**
     * Create credit account for specific team
     */
    async createForTeam(teamId: string, balance = 1000n, overrides?: Partial<Prisma.credit_accountCreateInput>) {
        const data = {
            id: generatePK(),
            currentBalance: balance,
            team: { connect: { id: teamId } },
            ...overrides,
        };
        
        return this.createMinimal(data);
    }

    /**
     * Create credit account with relationships
     */
    async createWithRelations(config: CreditAccountRelationConfig) {
        return this.prisma.$transaction(async (tx) => {
            const data: Prisma.credit_accountCreateInput = {
                id: generatePK(),
                currentBalance: config.initialBalance || 1000n,
            };

            if (config.userId) {
                data.user = { connect: { id: config.userId } };
            } else if (config.withUser) {
                // Create a new user if needed
                const user = await tx.user.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePK(),
                        name: "Test User",
                        handle: `test_user_${generatePK().slice(-6)}`,
                        status: "Unlocked",
                        isBot: false,
                        isBotDepictingPerson: false,
                        isPrivate: false,
                    },
                });
                data.user = { connect: { id: user.id } };
            }

            if (config.teamId) {
                data.team = { connect: { id: config.teamId } };
            } else if (config.withTeam) {
                // Create a new team if needed
                const team = await tx.team.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePK(),
                        name: "Test Team",
                        handle: `test_team_${generatePK().slice(-6)}`,
                        isPrivate: false,
                        createdBy: { connect: { id: generatePK() } },
                    },
                });
                data.team = { connect: { id: team.id } };
            }

            return tx.credit_account.create({
                data,
                include: {
                    user: true,
                    team: true,
                    ledgerEntries: true,
                },
            });
        });
    }

    /**
     * Verify credit account balance
     */
    async verifyBalance(accountId: string, expectedBalance: bigint) {
        const account = await this.prisma.credit_account.findUnique({
            where: { id: accountId },
        });
        
        if (!account) {
            throw new Error(`Credit account ${accountId} not found`);
        }
        
        if (account.currentBalance !== expectedBalance) {
            throw new Error(
                `Balance mismatch: expected ${expectedBalance}, got ${account.currentBalance}`
            );
        }
        
        return account;
    }
}

/**
 * Factory function to create CreditAccountDbFactory instance
 */
export function createCreditAccountDbFactory(prisma: PrismaClient): CreditAccountDbFactory {
    return new CreditAccountDbFactory(prisma);
}