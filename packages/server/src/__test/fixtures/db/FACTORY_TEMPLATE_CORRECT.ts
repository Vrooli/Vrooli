/**
 * CORRECTED Factory Template for Database Fixtures
 * This template uses the verified correct Prisma type patterns
 */

import { generatePK } from "@vrooli/shared";
import { type Prisma, type PrismaClient, type user } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";

/**
 * Example factory using CORRECT type patterns:
 * - Model type: Direct import (user, not Prisma.user)
 * - CreateInput: Prisma namespace (Prisma.userCreateInput)
 * - UpdateInput: Prisma namespace (Prisma.userUpdateInput)
 * - Include: Prisma namespace (Prisma.userInclude)
 */
export class UserDbFactoryTemplate extends EnhancedDatabaseFactory<
    user,                           // Model type - direct import
    Prisma.userCreateInput,         // Create input - Prisma namespace
    Prisma.userInclude,             // Include - Prisma namespace
    Prisma.userUpdateInput          // Update input - Prisma namespace
> {
    constructor(prisma: PrismaClient) {
        super("user", prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.user;
    }

    /**
     * Generate minimal data required for creating a user
     */
    protected generateMinimalData(overrides?: Partial<Prisma.userCreateInput>): Prisma.userCreateInput {
        return {
            id: generatePK(),
            publicId: generatePK().toString(),
            name: "Test User",
            handle: `test_user_${generatePK().toString().slice(-6)}`,
            status: "Unlocked",
            isBot: false,
            isBotDepictingPerson: false,
            isPrivate: false,
            ...overrides,
        };
    }

    /**
     * Generate complete data with all optional fields populated
     */
    protected generateCompleteData(overrides?: Partial<Prisma.userCreateInput>): Prisma.userCreateInput {
        return {
            id: generatePK(),
            publicId: generatePK().toString(),
            name: "Complete Test User",
            handle: `complete_user_${generatePK().toString().slice(-6)}`,
            status: "Unlocked",
            isBot: false,
            isBotDepictingPerson: false,
            isPrivate: false,
            theme: "dark",
            languages: ["en", "es"],
            botSettings: JSON.stringify({ model: "gpt-4", temperature: 0.7 }),
            creditSettings: JSON.stringify({ autoRecharge: false }),
            notificationSettings: JSON.stringify({ email: true, push: false }),
            isPrivateBookmarks: false,
            isPrivateMemberships: false,
            isPrivateResources: false,
            // Add any other optional fields that exist on the user model
            ...overrides,
        };
    }

    /**
     * Optional: Override to apply relationships when using createWithRelations
     * This example shows how to handle common user relationships
     */
    protected async applyRelationships(
        baseData: Prisma.userCreateInput,
        config: any,
        tx: PrismaClient,
    ): Promise<Prisma.userCreateInput> {
        const dataWithRelations = { ...baseData };

        // Example: Add authentication if requested
        if (config.withAuth) {
            (dataWithRelations as any).auths = {
                create: [{
                    id: this.generateId(),
                    provider: "Password",
                    hashed_password: "$2b$10$dummy.hashed.password.for.testing",
                }],
            };
        }

        // Example: Add email if requested
        if (config.withEmail) {
            (dataWithRelations as any).emails = {
                create: [{
                    id: this.generateId(),
                    emailAddress: `test_${Date.now()}@example.com`,
                    verifiedAt: new Date(),
                }],
            };
        }

        // Example: Connect to existing team if provided
        if (config.teamId) {
            (dataWithRelations as any).teams = {
                create: [{
                    id: this.generateId(),
                    role: {
                        create: {
                            id: this.generateId(),
                            name: "Member",
                            permissions: JSON.stringify(["Read"]),
                        },
                    },
                    team: {
                        connect: { id: config.teamId },
                    },
                }],
            };
        }

        return dataWithRelations;
    }

    /**
     * Optional: Define default includes for queries
     */
    protected getDefaultInclude(): Prisma.userInclude {
        return {
            _count: true,
            emails: true,
            auths: {
                select: {
                    id: true,
                    provider: true,
                },
            },
        };
    }

    /**
     * Create a bot user variant
     */
    async createBot(overrides?: Partial<Prisma.userCreateInput>): Promise<user> {
        return this.createMinimal({
            name: "Test Bot",
            handle: `test_bot_${this.generateId().toString().slice(-6)}`,
            isBot: true,
            botSettings: JSON.stringify({ model: "gpt-3.5-turbo" }),
            ...overrides,
        });
    }

    /**
     * Create a private user variant
     */
    async createPrivate(overrides?: Partial<Prisma.userCreateInput>): Promise<user> {
        return this.createMinimal({
            name: "Private User",
            handle: `private_user_${this.generateId().toString().slice(-6)}`,
            isPrivate: true,
            ...overrides,
        });
    }

    /**
     * Create an admin user variant
     */
    async createAdmin(overrides?: Partial<Prisma.userCreateInput>): Promise<user> {
        return this.createComplete({
            name: "Admin User",
            handle: `admin_${this.generateId().toString().slice(-6)}`,
            // Add any admin-specific fields
            ...overrides,
        });
    }

    /**
     * Override to add specific constraint checks for users
     */
    async verifyConstraints(id: string): Promise<{ valid: boolean; violations: string[] }> {
        const result = await super.verifyConstraints(id);
        
        try {
            const bigIntId = BigInt(id);
            const user = await this.findById(bigIntId);
            
            if (user) {
                // Check user-specific constraints
                if (!user.handle || user.handle.length < 3) {
                    result.violations.push("Handle must be at least 3 characters");
                }
                
                if (user.handle && user.handle.length > 30) {
                    result.violations.push("Handle must not exceed 30 characters");
                }
                
                if (!user.name || user.name.trim().length === 0) {
                    result.violations.push("Name cannot be empty");
                }
                
                if (user.isBot && user.isBotDepictingPerson) {
                    result.violations.push("Bot cannot depict a person");
                }
            }
        } catch (error) {
            // Error handling already done in parent
        }
        
        return {
            valid: result.violations.length === 0,
            violations: result.violations,
        };
    }
}

/**
 * Factory function using correct pattern
 */
export function createUserDbFactoryTemplate(prisma: PrismaClient): UserDbFactoryTemplate {
    return new UserDbFactoryTemplate(prisma);
}

/**
 * TEMPLATE PATTERN FOR OTHER MODELS:
 * 
 * 1. Import the model type directly: type api_key from "@prisma/client"
 * 2. Use Prisma namespace for inputs: Prisma.api_keyCreateInput
 * 3. Extend EnhancedDatabaseFactory with correct type parameters:
 *    - Model type (direct import)
 *    - CreateInput type (Prisma namespace)
 *    - Include type (Prisma namespace)
 *    - UpdateInput type (Prisma namespace)
 * 
 * 4. Implement required abstract methods:
 *    - getPrismaDelegate(): Return prisma.[modelName]
 *    - generateMinimalData(): Return minimal valid data
 *    - generateCompleteData(): Return complete data with all fields
 * 
 * 5. Optionally override:
 *    - applyRelationships(): Handle complex relationships
 *    - getDefaultInclude(): Define default query includes
 *    - verifyConstraints(): Add model-specific validation
 * 
 * 6. Add convenience methods for common variants (like createBot, createPrivate)
 * 
 * 7. All ID fields MUST use bigint type (via generatePK())
 * 
 * Example for api_key model:
 * 
 * import { type Prisma, type PrismaClient, type api_key } from "@prisma/client";
 * 
 * export class ApiKeyDbFactory extends EnhancedDatabaseFactory<
 *     api_key,
 *     Prisma.api_keyCreateInput,
 *     Prisma.api_keyInclude,
 *     Prisma.api_keyUpdateInput
 * > {
 *     constructor(prisma: PrismaClient) {
 *         super("api_key", prisma);
 *     }
 * 
 *     protected getPrismaDelegate() {
 *         return this.prisma.api_key;
 *     }
 * 
 *     protected generateMinimalData(overrides?: Partial<Prisma.api_keyCreateInput>): Prisma.api_keyCreateInput {
 *         return {
 *             id: generatePK(),
 *             key: `test_key_${generatePK().toString().slice(-8)}`,
 *             // ... other required fields
 *             ...overrides,
 *         };
 *     }
 * 
 *     // ... implement generateCompleteData and other methods
 * }
 */
