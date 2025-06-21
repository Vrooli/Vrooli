import { generatePK, generatePublicId } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
} from "./types.js";

interface ModelNameRelationConfig extends RelationConfig {
    // Add model-specific relation configuration options
    withUser?: boolean;
    withTeam?: boolean;
    userId?: string;
    teamId?: string;
}

/**
 * Template for creating model-specific database factories
 * Replace MODEL_NAME with actual model name (e.g., User, CreditAccount)
 * Replace model_name with actual schema model name (e.g., user, credit_account)
 */
export class ModelNameDbFactory extends EnhancedDatabaseFactory<
    Prisma.model_nameCreateInput,   // ✅ Use CreateInput as model type parameter
    Prisma.model_nameCreateInput,   // Create input type  
    Prisma.model_nameInclude,       // Include type
    Prisma.model_nameUpdateInput    // Update input type
> {
    constructor(prisma: PrismaClient) {
        super('model_name', prisma);
    }

    // ✅ Required: Get the Prisma delegate for database operations
    protected getPrismaDelegate() {
        return this.prisma.model_name;
    }

    // ✅ Required: Generate minimal valid data
    protected generateMinimalData(overrides?: Partial<Prisma.model_nameCreateInput>): Prisma.model_nameCreateInput {
        const baseData: Prisma.model_nameCreateInput = {
            id: generatePK(),
            // Add other required fields based on schema
            // Example for user:
            // name: "Test User",
            // handle: `test_${generatePK().toString().slice(-6)}`,
            // status: "Unlocked",
        };
        
        return {
            ...baseData,
            ...overrides,
        };
    }

    // ✅ Required: Generate complete data with all common fields
    protected generateCompleteData(overrides?: Partial<Prisma.model_nameCreateInput>): Prisma.model_nameCreateInput {
        const baseData = this.generateMinimalData();
        const completeData: Prisma.model_nameCreateInput = {
            ...baseData,
            // Add additional fields that are commonly used in tests
            // Example for user:
            // publicId: generatePublicId(),
            // isBot: false,
            // isPrivate: false,
        };
        
        return {
            ...completeData,
            ...overrides,
        };
    }

    // ✅ Required: Get complete test fixtures
    protected getFixtures(): DbTestFixtures<Prisma.model_nameCreateInput, Prisma.model_nameUpdateInput> {
        return {
            minimal: this.generateMinimalData(),
            
            complete: this.generateCompleteData(),
            
            variants: {
                // Named variations for specific test scenarios
                variant1: {
                    ...this.generateMinimalData(),
                    // Specific field overrides
                },
                
                variant2: {
                    ...this.generateMinimalData(),
                    // Different field values
                },
            },
            
            invalid: {
                // Invalid data for error testing
                missingRequired: {
                    // Missing required fields - structure depends on model
                    id: generatePK(),
                    // Intentionally omit required fields
                } as any, // Type assertion since this is intentionally invalid
                
                invalidTypes: {
                    // Invalid field types
                    id: "invalid-id" as any,  // Wrong type for id
                    // Other invalid type examples
                },
            },
            
            edgeCase: {
                // Edge cases and boundary conditions
                maxLengthFields: {
                    ...this.generateMinimalData(),
                    // Test maximum field lengths
                },
                
                specialCharacters: {
                    ...this.generateMinimalData(),
                    // Test special characters in string fields
                },
            },
            
            update: {
                // Update operations
                standardUpdate: {
                    // Standard field updates
                },
                
                statusChange: {
                    // Status or state changes
                },
            },
        };
    }

    // ✅ Optional: Get default include configuration
    protected getDefaultInclude(): Prisma.model_nameInclude {
        return {
            // Define which relations to include by default
            // Example:
            // user: true,
            // team: true,
        } as Prisma.model_nameInclude;
    }

    // Helper methods for common operations

    /**
     * Create record with specific overrides
     */
    async createWithOverrides(overrides: Partial<Prisma.model_nameCreateInput>) {
        return this.createMinimal(overrides);
    }

    /**
     * Create record with relationships
     */
    async createWithRelations(config: ModelNameRelationConfig) {
        return this.prisma.$transaction(async (tx) => {
            const data = this.generateMinimalData();
            
            // Handle relationship creation based on config
            if (config.userId) {
                // Connect to existing user
                (data as any).user = { connect: { id: config.userId } };
            } else if (config.withUser) {
                // Create new user and connect
                const user = await tx.user.create({
                    data: {
                        id: generatePK(),
                        name: "Related User",
                        // Other required user fields
                    } as any, // Simplified for template
                });
                (data as any).user = { connect: { id: user.id } };
            }

            // Add more relationship handling as needed

            return tx.model_name.create({
                data,
                include: this.getDefaultInclude(),
            });
        });
    }
}

/**
 * Factory function to create ModelNameDbFactory instance
 */
export function createModelNameDbFactory(prisma: PrismaClient): ModelNameDbFactory {
    return new ModelNameDbFactory(prisma);
}

// Export type for external use
export type { ModelNameRelationConfig };