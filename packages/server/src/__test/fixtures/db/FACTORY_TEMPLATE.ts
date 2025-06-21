import { generatePK } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { DbTestFixtures } from "./types.js";
import { PRISMA_TYPE_MAP } from "./PRISMA_TYPE_MAP.js";

/**
 * Template for creating model-specific database factories
 * 
 * USAGE INSTRUCTIONS:
 * 1. Copy this file and rename to [ModelName]DbFactory.ts
 * 2. Replace all instances of 'MODEL_NAME' with your actual model name (e.g., 'user', 'team')
 * 3. Replace 'ModelName' with PascalCase version (e.g., 'User', 'Team')
 * 4. Update the fixtures to match your model's requirements
 * 5. Implement model-specific helper methods as needed
 * 
 * EXAMPLE:
 * For model 'credit_account':
 * - MODEL_NAME -> credit_account
 * - ModelName -> CreditAccount
 * - File name -> CreditAccountDbFactory.ts
 */

// Get the correct Prisma types from our mapping
const MODEL_TYPES = PRISMA_TYPE_MAP['MODEL_NAME'];

export class ModelNameDbFactory extends EnhancedDatabaseFactory<
    Prisma.MODEL_NAME,                                     // Model type (from Prisma namespace)
    Prisma[typeof MODEL_TYPES.createInput],                // Create input type
    Prisma[typeof MODEL_TYPES.include],                    // Include type
    Prisma[typeof MODEL_TYPES.updateInput]                 // Update input type
> {
    constructor(prisma: PrismaClient) {
        super('MODEL_NAME', prisma);
    }

    /**
     * Get the Prisma delegate for this model
     */
    protected getPrismaDelegate() {
        return this.prisma.MODEL_NAME;
    }

    /**
     * Get minimal valid data for this model
     */
    protected generateMinimalData(overrides?: Partial<Prisma[typeof MODEL_TYPES.createInput]>): Prisma[typeof MODEL_TYPES.createInput] {
        return {
            id: generatePK(),
            // TODO: Add minimal required fields for this model
            // Example for a user model:
            // name: "Test User",
            // handle: `test_user_${generatePK().slice(-6)}`,
            ...overrides,
        } as Prisma[typeof MODEL_TYPES.createInput];
    }

    /**
     * Get complete data with all fields populated
     */
    protected generateCompleteData(overrides?: Partial<Prisma[typeof MODEL_TYPES.createInput]>): Prisma[typeof MODEL_TYPES.createInput] {
        return {
            ...this.generateMinimalData(),
            // TODO: Add all optional fields with reasonable defaults
            // Example:
            // bio: "A complete test user with all fields",
            // isPrivate: false,
            // created: new Date(),
            ...overrides,
        } as Prisma[typeof MODEL_TYPES.createInput];
    }

    /**
     * Get complete test fixtures for this model
     */
    protected getFixtures(): DbTestFixtures<Prisma[typeof MODEL_TYPES.createInput], Prisma[typeof MODEL_TYPES.updateInput]> {
        return {
            minimal: this.generateMinimalData(),
            
            complete: this.generateCompleteData(),
            
            variants: {
                // TODO: Add named variations for specific test scenarios
                // Example:
                // privateUser: this.generateMinimalData({ isPrivate: true }),
                // adminUser: this.generateMinimalData({ role: "Admin" }),
            },
            
            invalid: {
                missingRequired: {
                    // TODO: Create data missing required fields
                    // This should fail validation
                } as any,
                
                invalidTypes: {
                    // TODO: Create data with invalid field types
                    // Example: id: "not-a-number" when expecting number
                } as any,
            },
            
            edgeCase: {
                // TODO: Add edge cases specific to this model
                // Example:
                // maxLengthName: this.generateMinimalData({ 
                //     name: "A".repeat(255) // Max length 
                // }),
            },
            
            update: {
                // TODO: Add common update scenarios
                // Example:
                // changeName: { name: "Updated Name" },
                // makePrivate: { isPrivate: true },
            },
        };
    }

    /**
     * Get transaction-scoped Prisma delegate
     */
    protected getTxPrismaDelegate(tx: any) {
        return tx.MODEL_NAME;
    }

    /**
     * Apply relationships to the data
     */
    protected async applyRelationships(
        data: Prisma[typeof MODEL_TYPES.createInput],
        config: any,
        tx: any
    ): Promise<Prisma[typeof MODEL_TYPES.createInput]> {
        // TODO: Implement relationship handling specific to this model
        // Example for a model with a user relationship:
        // if (config.withUser && !data.user) {
        //     const user = await tx.user.create({
        //         data: { /* user data */ }
        //     });
        //     data.user = { connect: { id: user.id } };
        // }
        return data;
    }

    /**
     * Get default include configuration
     */
    protected getDefaultInclude(): Prisma[typeof MODEL_TYPES.include] | undefined {
        // TODO: Return default relationships to include
        // Example:
        // return {
        //     user: true,
        //     tags: true,
        // };
        return undefined;
    }

    /**
     * Make data unique for bulk operations
     */
    protected makeUnique(data: Prisma[typeof MODEL_TYPES.createInput], index: number): Prisma[typeof MODEL_TYPES.createInput] {
        // TODO: Ensure unique fields are actually unique
        // Example:
        // return {
        //     ...data,
        //     handle: `${data.handle}_${index}`,
        //     email: `user_${index}@example.com`,
        // };
        return {
            ...data,
            id: generatePK(), // Always generate new ID
        };
    }

    // Model-specific helper methods
    // TODO: Add any model-specific helper methods here
    // Example:
    // async createWithPremium(overrides?: Partial<Prisma.userCreateInput>) {
    //     return this.createComplete({
    //         premium: { /* premium data */ },
    //         ...overrides,
    //     });
    // }
}

/**
 * Factory function to create ModelNameDbFactory instance
 */
export function createModelNameDbFactory(prisma: PrismaClient): ModelNameDbFactory {
    return new ModelNameDbFactory(prisma);
}