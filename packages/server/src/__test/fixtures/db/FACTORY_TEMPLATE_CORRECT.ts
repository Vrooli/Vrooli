/**
 * CORRECTED Factory Template for Database Fixtures
 * This template uses the verified correct Prisma type patterns
 */

import { generatePK } from "@vrooli/shared";
import { type Prisma, type PrismaClient, type user } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { DbTestFixtures } from "./types.js";

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
     * Generate test fixtures using correct types
     */
    protected getFixtures(): DbTestFixtures<Prisma.userCreateInput, Prisma.userUpdateInput> {
        return {
            minimal: {
                id: generatePK(),
                publicId: generatePK().toString(),
                name: "Test User",
                handle: `test_user_${generatePK().toString().slice(-6)}`,
                status: "Unlocked",
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
            },
            
            complete: {
                id: generatePK(),
                publicId: generatePK().toString(),
                name: "Complete Test User",
                handle: `complete_user_${generatePK().toString().slice(-6)}`,
                status: "Unlocked",
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
                // Add additional optional fields for complete version
            },
            
            variants: {
                bot: {
                    id: generatePK(),
                    publicId: generatePK().toString(),
                    name: "Test Bot",
                    handle: `test_bot_${generatePK().toString().slice(-6)}`,
                    status: "Unlocked",
                    isBot: true,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                },
                
                private: {
                    id: generatePK(),
                    publicId: generatePK().toString(),
                    name: "Private User",
                    handle: `private_user_${generatePK().toString().slice(-6)}`,
                    status: "Unlocked",
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: true,
                },
            },
            
            invalid: {
                missingRequired: {
                    // Missing required fields like name, handle
                    id: generatePK(),
                } as any, // Cast to bypass type checking for invalid data
                
                invalidTypes: {
                    id: "invalid_id_string" as any,
                    publicId: 12345 as any,
                    name: true as any,
                    handle: null as any,
                    status: "InvalidStatus" as any,
                    isBot: "not_boolean" as any,
                    isBotDepictingPerson: 1 as any,
                    isPrivate: {} as any,
                },
            },
            
            edgeCase: {
                longHandle: {
                    id: generatePK(),
                    publicId: generatePK().toString(),
                    name: "User with very long handle",
                    handle: "a".repeat(100), // Test handle length limits
                    status: "Unlocked",
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                },
            },
            
            update: {
                updateName: {
                    name: "Updated User Name",
                },
                
                makePrivate: {
                    isPrivate: true,
                },
                
                changeStatus: {
                    status: "Deleted",
                },
            },
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
 * 3. Use EnhancedDatabaseFactory with correct type parameters
 * 4. Implement getPrismaDelegate() to return prisma.[modelName]
 * 5. Implement getFixtures() with proper test data structure
 */
