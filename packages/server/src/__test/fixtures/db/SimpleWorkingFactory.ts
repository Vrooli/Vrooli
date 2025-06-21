/**
 * Simple Working Factory Template
 * This bypasses the complex EnhancedDatabaseFactory to focus on core functionality
 */

import { generatePK } from "@vrooli/shared";
import { type Prisma, type PrismaClient, type user } from "@prisma/client";

/**
 * Simple factory that works with correct Prisma types
 * This demonstrates the correct pattern without complex inheritance
 */
export class SimpleUserDbFactory {
    constructor(private prisma: PrismaClient) {}

    /**
     * Create minimal user with only required fields
     */
    async createMinimal(overrides?: Partial<Prisma.userCreateInput>): Promise<user> {
        const data: Prisma.userCreateInput = {
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

        return this.prisma.user.create({
            data,
        });
    }

    /**
     * Create complete user with additional optional fields
     */
    async createComplete(overrides?: Partial<Prisma.userCreateInput>): Promise<user> {
        const data: Prisma.userCreateInput = {
            id: generatePK(),
            publicId: generatePK().toString(),
            name: "Complete Test User",
            handle: `complete_user_${generatePK().toString().slice(-6)}`,
            status: "Unlocked",
            isBot: false,
            isBotDepictingPerson: false,
            isPrivate: false,
            // Add optional fields if they exist
            // bannerImage: "",
            // profileImage: "",
            ...overrides,
        };

        return this.prisma.user.create({
            data,
        });
    }

    /**
     * Create user with relationships
     */
    async createWithAuth(password = "test123", overrides?: Partial<Prisma.userCreateInput>): Promise<user> {
        return this.prisma.$transaction(async (tx) => {
            const user = await tx.user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePK().toString(),
                    name: "User with Auth",
                    handle: `auth_user_${generatePK().toString().slice(-6)}`,
                    status: "Unlocked",
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                    ...overrides,
                },
            });

            // Create authentication record
            await tx.user_auth.create({
                data: {
                    id: generatePK(),
                    user_id: user.id,
                    provider: "Password",
                    // Note: In real usage, password should be hashed
                    hashed_password: `hashed_${password}`,
                },
            });

            return user;
        });
    }

    /**
     * Create bot user
     */
    async createBot(overrides?: Partial<Prisma.userCreateInput>): Promise<user> {
        return this.createMinimal({
            name: "Test Bot",
            handle: `test_bot_${generatePK().toString().slice(-6)}`,
            isBot: true,
            ...overrides,
        });
    }

    /**
     * Create private user
     */
    async createPrivate(overrides?: Partial<Prisma.userCreateInput>): Promise<user> {
        return this.createMinimal({
            name: "Private User",
            handle: `private_user_${generatePK().toString().slice(-6)}`,
            isPrivate: true,
            ...overrides,
        });
    }

    /**
     * Update user
     */
    async update(id: bigint, data: Prisma.userUpdateInput): Promise<user> {
        return this.prisma.user.update({
            where: { id },
            data,
        });
    }

    /**
     * Delete user
     */
    async delete(id: bigint): Promise<user> {
        return this.prisma.user.delete({
            where: { id },
        });
    }

    /**
     * Find user with relationships
     */
    async findWithRelations(id: bigint): Promise<user | null> {
        return this.prisma.user.findUnique({
            where: { id },
            include: {
                auths: true,
                // Add other relationships as needed
            },
        });
    }
}

/**
 * Factory function for easy instantiation
 */
export function createSimpleUserDbFactory(prisma: PrismaClient): SimpleUserDbFactory {
    return new SimpleUserDbFactory(prisma);
}

/**
 * TEMPLATE PATTERN FOR OTHER MODELS:
 * 
 * 1. Import the model type directly: type api_key from "@prisma/client"
 * 2. Use Prisma namespace for inputs: Prisma.api_keyCreateInput, Prisma.api_keyUpdateInput
 * 3. Use constructor injection for PrismaClient
 * 4. Implement createMinimal, createComplete, and specialized creation methods
 * 5. Use transactions for complex relationship creation
 * 6. Provide update, delete, and relationship query methods
 * 
 * Example for api_key:
 * 
 * export class SimpleApiKeyDbFactory {
 *     constructor(private prisma: PrismaClient) {}
 *     
 *     async createMinimal(overrides?: Partial<Prisma.api_keyCreateInput>): Promise<api_key> {
 *         const data: Prisma.api_keyCreateInput = {
 *             id: generatePK(),
 *             // ... required fields for api_key
 *             ...overrides,
 *         };
 *         return this.prisma.api_key.create({ data });
 *     }
 * }
 */