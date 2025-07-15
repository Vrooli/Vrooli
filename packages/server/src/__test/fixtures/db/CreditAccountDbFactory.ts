// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed type safety issues: removed temporary any type parameter, improved invalid type casts
import type { Prisma, PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { DbTestFixtures } from "./types.js";

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
    Prisma.credit_accountCreateInput,
    Prisma.credit_accountCreateInput,
    Prisma.credit_accountInclude,
    Prisma.credit_accountUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("credit_account", prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.credit_account;
    }

    protected generateMinimalData(overrides?: Partial<Prisma.credit_accountCreateInput>): Prisma.credit_accountCreateInput {
        return {
            id: this.generateId(),
            currentBalance: BigInt(1000),
            user: { connect: { id: this.generateId() } },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.credit_accountCreateInput>): Prisma.credit_accountCreateInput {
        return {
            ...this.generateMinimalData(),
            currentBalance: BigInt(100000),
            ...overrides,
        };
    }

    /**
     * Get complete test fixtures for Credit Account model
     */
    protected getFixtures(): DbTestFixtures<Prisma.credit_accountCreateInput, Prisma.credit_accountUpdateInput> {
        const userId = this.generateId();
        const teamId = this.generateId();
        
        return {
            minimal: this.generateMinimalData(),
            
            complete: this.generateCompleteData(),
            
            edgeCases: {
                zeroBalance: {
                    id: this.generateId(),
                    currentBalance: BigInt(0),
                    user: { connect: { id: userId } },
                },
                
                highBalance: {
                    id: this.generateId(),
                    currentBalance: BigInt(10000000), // 10 million credits
                    user: { connect: { id: userId } },
                },
                
                teamAccount: {
                    id: this.generateId(),
                    currentBalance: BigInt(50000),
                    team: { connect: { id: teamId } },
                },
                
                userWithLowBalance: {
                    id: this.generateId(),
                    currentBalance: BigInt(10),
                    user: { connect: { id: userId } },
                },
                
                maxBalance: {
                    id: this.generateId(),
                    currentBalance: BigInt("9223372036854775807"), // Max bigint value
                    user: { connect: { id: userId } },
                },
                
                bothUserAndTeam: {
                    id: this.generateId(),
                    currentBalance: BigInt(1000),
                    user: { connect: { id: userId } },
                    team: { connect: { id: teamId } },
                },
            },
            
            invalid: {
                missingRequired: {
                    id: this.generateId(),
                    currentBalance: BigInt(1000),
                    // Missing user or team connection
                } as Prisma.credit_accountCreateInput,
                
                invalidTypes: {
                    id: "not-a-bigint" as unknown as bigint,
                    currentBalance: "not-a-number" as unknown as bigint,
                    user: { connect: { id: userId } },
                } as Prisma.credit_accountCreateInput,
            },
            
            updates: {
                minimal: {
                    currentBalance: BigInt(5000),
                },
                
                complete: {
                    currentBalance: BigInt(100000),
                },
                
                addCredits: {
                    currentBalance: BigInt(5000),
                },
                
                subtractCredits: {
                    currentBalance: BigInt(500),
                },
                
                zeroOut: {
                    currentBalance: BigInt(0),
                },
            },
        };
    }

    /**
     * Create credit account with specific balance
     */
    async createWithBalance(balance: bigint, overrides?: Partial<Prisma.credit_accountCreateInput>) {
        const data = {
            ...this.generateMinimalData(),
            currentBalance: balance,
            ...overrides,
        };
        
        return this.createMinimal(data);
    }

    /**
     * Create credit account for specific user
     */
    async createForUser(userId: bigint, balance = BigInt(1000), overrides?: Partial<Prisma.credit_accountCreateInput>) {
        const data = {
            ...this.generateMinimalData(),
            currentBalance: balance,
            user: { connect: { id: userId } },
            team: undefined, // Clear team connection
            ...overrides,
        };
        
        return this.createMinimal(data);
    }

    /**
     * Create credit account for specific team
     */
    async createForTeam(teamId: bigint, balance = BigInt(1000), overrides?: Partial<Prisma.credit_accountCreateInput>) {
        const data = {
            id: this.generateId(),
            currentBalance: balance,
            team: { connect: { id: teamId } },
            user: undefined, // Clear user connection
            ...overrides,
        };
        
        return this.createMinimal(data);
    }
}

/**
 * Factory function to create CreditAccountDbFactory instance
 */
export function createCreditAccountDbFactory(prisma: PrismaClient): CreditAccountDbFactory {
    return new CreditAccountDbFactory(prisma);
}
