import { generatePK, generatePublicId, generateSnowflake, nanoid } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { DbTestFixtures, BulkSeedOptions, BulkSeedResult, DbErrorScenarios } from "./types.js";

/**
 * Database fixtures for ApiKey model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const apiKeyDbIds = {
    apiKey1: generatePK(),
    apiKey2: generatePK(),
    apiKey3: generatePK(),
    apiKey4: generatePK(),
    apiKey5: generatePK(),
};

/**
 * Enhanced test fixtures for ApiKey model following standard structure
 */
export const apiKeyDbFixtures: DbTestFixtures<Prisma.api_keyCreateInput> = {
    minimal: {
        id: generatePK(),
        key: `vrooli_test_${generateSnowflake()}`,
        name: "Test API Key",
        permissions: {},
        limitHard: BigInt(25000000000),
        stopAtLimit: true,
    },
    complete: {
        id: generatePK(),
        key: `vrooli_test_${generateSnowflake()}`,
        name: "Complete Test API Key",
        permissions: {
            endpoints: ["user:read", "user:write", "project:read", "project:write", "routine:read", "routine:write"],
            rateLimit: 100,
            rateLimitWindow: 60,
        },
        limitHard: BigInt(50000000000),
        limitSoft: BigInt(40000000000),
        creditsUsed: BigInt(15000000000),
        stopAtLimit: true,
    },
    invalid: {
        missingRequired: {
            // Missing required id, key, name, limitHard
            permissions: {},
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            key: 123, // Should be string
            name: true, // Should be string
            permissions: "invalid-json", // Should be object
            limitHard: "not-a-bigint", // Should be bigint
            limitSoft: "not-a-bigint", // Should be bigint
            creditsUsed: "not-a-bigint", // Should be bigint
            stopAtLimit: "yes", // Should be boolean
        },
        duplicateKey: {
            id: generatePK(),
            key: "duplicate_api_key", // Same key as another API key
            name: "Duplicate Key API Key",
            permissions: {},
            limitHard: BigInt(25000000000),
            stopAtLimit: true,
        },
        softLimitExceedsHard: {
            id: generatePK(),
            key: `vrooli_test_${generateSnowflake()}`,
            name: "Invalid Limits API Key",
            permissions: {},
            limitHard: BigInt(25000000000),
            limitSoft: BigInt(50000000000), // Soft limit exceeds hard limit
            stopAtLimit: true,
        },
        negativeCredits: {
            id: generatePK(),
            key: `vrooli_test_${generateSnowflake()}`,
            name: "Negative Credits API Key",
            permissions: {},
            limitHard: BigInt(25000000000),
            creditsUsed: BigInt(-1000), // Negative credits
            stopAtLimit: true,
        },
    },
    edgeCases: {
        maxLengthName: {
            id: generatePK(),
            key: `vrooli_test_${generateSnowflake()}`,
            name: "a".repeat(255), // Maximum name length
            permissions: {},
            limitHard: BigInt(25000000000),
            stopAtLimit: true,
        },
        zeroLimits: {
            id: generatePK(),
            key: `vrooli_test_${generateSnowflake()}`,
            name: "Zero Limits API Key",
            permissions: {},
            limitHard: BigInt(0),
            limitSoft: BigInt(0),
            creditsUsed: BigInt(0),
            stopAtLimit: true,
        },
        maxLimits: {
            id: generatePK(),
            key: `vrooli_test_${generateSnowflake()}`,
            name: "Max Limits API Key",
            permissions: {},
            limitHard: BigInt(Number.MAX_SAFE_INTEGER), // Max safe bigint value
            limitSoft: BigInt(Number.MAX_SAFE_INTEGER - 1),
            creditsUsed: BigInt(Number.MAX_SAFE_INTEGER - 2),
            stopAtLimit: false,
        },
        complexPermissions: {
            id: generatePK(),
            key: `vrooli_test_${generateSnowflake()}`,
            name: "Complex Permissions API Key",
            permissions: {
                endpoints: [
                    "user:read", "user:write", "user:delete",
                    "project:read", "project:write", "project:delete",
                    "routine:read", "routine:write", "routine:delete",
                    "team:read", "team:write", "team:delete",
                    "admin:read", "admin:write"
                ],
                rateLimit: 1000,
                rateLimitWindow: 3600,
                ipWhitelist: ["192.168.1.1", "10.0.0.0/8"],
                allowedOrigins: ["https://example.com", "https://api.example.com"],
                scopes: ["read", "write", "admin"],
                metadata: {
                    purpose: "Integration testing",
                    environment: "test",
                    version: "1.0.0"
                }
            },
            limitHard: BigInt(100000000000),
            limitSoft: BigInt(80000000000),
            stopAtLimit: false,
        },
        disabledApiKey: {
            id: generatePK(),
            key: `vrooli_test_${generateSnowflake()}`,
            name: "Disabled API Key",
            permissions: {},
            limitHard: BigInt(25000000000),
            stopAtLimit: true,
            disabledAt: new Date(),
        },
        nearSoftLimit: {
            id: generatePK(),
            key: `vrooli_test_${generateSnowflake()}`,
            name: "Near Soft Limit API Key",
            permissions: {},
            limitHard: BigInt(25000000000),
            limitSoft: BigInt(20000000000),
            creditsUsed: BigInt(19900000000), // 99.5% of soft limit
            stopAtLimit: false,
        },
        nearHardLimit: {
            id: generatePK(),
            key: `vrooli_test_${generateSnowflake()}`,
            name: "Near Hard Limit API Key",
            permissions: {},
            limitHard: BigInt(25000000000),
            creditsUsed: BigInt(24900000000), // 99.6% of hard limit
            stopAtLimit: true,
        },
        exceededSoftLimit: {
            id: generatePK(),
            key: `vrooli_test_${generateSnowflake()}`,
            name: "Exceeded Soft Limit API Key",
            permissions: {},
            limitHard: BigInt(25000000000),
            limitSoft: BigInt(20000000000),
            creditsUsed: BigInt(22000000000), // Exceeded soft but under hard
            stopAtLimit: false,
        },
    },
};

/**
 * Legacy API key fixtures for backward compatibility
 */
export const minimalApiKeyDb: Prisma.api_keyCreateInput = {
    id: apiKeyDbIds.apiKey1,
    key: `vrooli_test_${generateSnowflake()}`,
    name: "Test API Key",
    permissions: {},
    limitHard: BigInt(25000000000),
    stopAtLimit: true,
};

export const apiKeyWithSoftLimitDb: Prisma.api_keyCreateInput = {
    id: apiKeyDbIds.apiKey2,
    key: `vrooli_test_${generateSnowflake()}`,
    name: "API Key with Soft Limit",
    permissions: {},
    limitHard: BigInt(50000000000),
    limitSoft: BigInt(40000000000),
    stopAtLimit: true,
};

export const apiKeyWithPermissionsDb: Prisma.api_keyCreateInput = {
    id: apiKeyDbIds.apiKey3,
    key: `vrooli_test_${generateSnowflake()}`,
    name: "API Key with Permissions",
    permissions: {
        endpoints: ["user:read", "project:read", "routine:read"],
        rateLimit: 100,
        rateLimitWindow: 60,
    },
    limitHard: BigInt(25000000000),
    stopAtLimit: true,
};

export const disabledApiKeyDb: Prisma.api_keyCreateInput = {
    id: apiKeyDbIds.apiKey4,
    key: `vrooli_test_${generateSnowflake()}`,
    name: "Disabled API Key",
    permissions: {},
    limitHard: BigInt(25000000000),
    stopAtLimit: true,
    disabledAt: new Date(),
};

export const apiKeyWithUsageDb: Prisma.api_keyCreateInput = {
    id: apiKeyDbIds.apiKey5,
    key: `vrooli_test_${generateSnowflake()}`,
    name: "API Key with Usage",
    permissions: {},
    limitHard: BigInt(25000000000),
    limitSoft: BigInt(20000000000),
    creditsUsed: BigInt(15000000000), // 60% of hard limit, 75% of soft limit
    stopAtLimit: false,
};

/**
 * Enhanced factory for creating ApiKey database fixtures
 */
export class ApiKeyDbFactory extends EnhancedDbFactory<Prisma.api_keyCreateInput> {
    
    /**
     * Get the test fixtures for ApiKey model
     */
    protected getFixtures(): DbTestFixtures<Prisma.api_keyCreateInput> {
        return apiKeyDbFixtures;
    }

    /**
     * Generate fresh identifiers for ApiKey
     */
    protected generateFreshIdentifiers(): Record<string, any> {
        return {
            id: generatePK(),
            key: `vrooli_test_${generateSnowflake()}`,
        };
    }

    /**
     * Get ApiKey-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: apiKeyDbIds.apiKey1, // Duplicate ID
                    key: `vrooli_test_${generateSnowflake()}`,
                    name: "Duplicate ID API Key",
                    permissions: {},
                    limitHard: BigInt(25000000000),
                    stopAtLimit: true,
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    key: `vrooli_test_${generateSnowflake()}`,
                    name: "Foreign Key API Key",
                    permissions: {},
                    limitHard: BigInt(25000000000),
                    stopAtLimit: true,
                    user: { connect: { id: BigInt(999999999999999) } },
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    key: "", // Empty key violates constraint
                    name: "Check Constraint API Key",
                    permissions: {},
                    limitHard: BigInt(25000000000),
                    stopAtLimit: true,
                },
            },
            validation: {
                requiredFieldMissing: apiKeyDbFixtures.invalid.missingRequired,
                invalidDataType: apiKeyDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    key: `vrooli_test_${generateSnowflake()}`,
                    name: "a".repeat(1000), // Name too long
                    permissions: {},
                    limitHard: BigInt(0), // Zero limit instead of negative
                    stopAtLimit: true,
                },
            },
            businessLogic: {
                softLimitExceedsHard: apiKeyDbFixtures.invalid.softLimitExceedsHard,
                negativeCredits: apiKeyDbFixtures.invalid.negativeCredits,
                creditsExceedHardLimit: {
                    id: generatePK(),
                    key: `vrooli_test_${generateSnowflake()}`,
                    name: "Credits Exceed Hard Limit API Key",
                    permissions: {},
                    limitHard: BigInt(25000000000),
                    creditsUsed: BigInt(30000000000), // Credits exceed hard limit
                    stopAtLimit: true,
                },
                disabledKeyWithUsage: {
                    id: generatePK(),
                    key: `vrooli_test_${generateSnowflake()}`,
                    name: "Disabled Key with Usage",
                    permissions: {},
                    limitHard: BigInt(25000000000),
                    creditsUsed: BigInt(5000000000),
                    disabledAt: new Date(),
                    stopAtLimit: true,
                },
            },
        };
    }

    /**
     * Add user relationship to an ApiKey fixture
     */
    protected addAuthentication(data: Prisma.api_keyCreateInput): Prisma.api_keyCreateInput {
        // For ApiKey, authentication means associating with a user
        return {
            ...data,
            user: { connect: { id: BigInt(generatePK()) } },
        };
    }

    /**
     * Add team relationship to an ApiKey fixture
     */
    protected addTeamMemberships(data: Prisma.api_keyCreateInput, teams: Array<{ teamId: string; role: string }>): Prisma.api_keyCreateInput {
        // For ApiKey, we can only associate with one team
        const firstTeam = teams[0];
        if (firstTeam) {
            return {
                ...data,
                team: { connect: { id: BigInt(firstTeam.teamId) } },
            };
        }
        return data;
    }

    /**
     * ApiKey-specific validation
     */
    protected validateSpecific(data: Prisma.api_keyCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to ApiKey
        if (!data.key) errors.push("API key is required");
        if (!data.name) errors.push("API key name is required");
        if (!data.limitHard) errors.push("Hard limit is required");

        // Check business logic
        if (data.limitSoft && data.limitHard && data.limitSoft > data.limitHard) {
            errors.push("Soft limit cannot exceed hard limit");
        }

        if (data.creditsUsed && data.limitHard && data.creditsUsed > data.limitHard) {
            warnings.push("Credits used exceeds hard limit - key should be disabled");
        }

        if (data.creditsUsed && data.creditsUsed < BigInt(0)) {
            errors.push("Credits used cannot be negative");
        }

        if (data.limitHard && data.limitHard < BigInt(0)) {
            errors.push("Hard limit cannot be negative");
        }

        if (data.limitSoft && data.limitSoft < BigInt(0)) {
            errors.push("Soft limit cannot be negative");
        }

        // Check key format
        if (data.key && !data.key.startsWith("vrooli_")) {
            warnings.push("API key should start with 'vrooli_' prefix");
        }

        // Check permissions structure
        if (data.permissions && typeof data.permissions !== 'object') {
            errors.push("Permissions must be an object");
        }

        // Check disabled state consistency
        if (data.disabledAt && data.stopAtLimit === false) {
            warnings.push("Disabled keys should probably have stopAtLimit=true");
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(overrides?: Partial<Prisma.api_keyCreateInput>): Prisma.api_keyCreateInput {
        const factory = new ApiKeyDbFactory();
        return factory.createMinimal(overrides);
    }

    static createForUser(
        userId: string,
        overrides?: Partial<Prisma.api_keyCreateInput>
    ): Prisma.api_keyCreateInput {
        const factory = new ApiKeyDbFactory();
        return factory.createMinimal({
            ...overrides,
            user: { connect: { id: BigInt(userId) } },
        });
    }

    static createForTeam(
        teamId: string,
        overrides?: Partial<Prisma.api_keyCreateInput>
    ): Prisma.api_keyCreateInput {
        const factory = new ApiKeyDbFactory();
        return factory.createMinimal({
            ...overrides,
            team: { connect: { id: BigInt(teamId) } },
        });
    }

    static createWithPermissions(
        permissions: Record<string, any>,
        overrides?: Partial<Prisma.api_keyCreateInput>
    ): Prisma.api_keyCreateInput {
        const factory = new ApiKeyDbFactory();
        return factory.createMinimal({
            ...overrides,
            permissions,
        });
    }

    static createDisabled(
        overrides?: Partial<Prisma.api_keyCreateInput>
    ): Prisma.api_keyCreateInput {
        const factory = new ApiKeyDbFactory();
        return factory.createEdgeCase("disabledApiKey");
    }

    static createWithUsage(
        creditsUsed: bigint,
        overrides?: Partial<Prisma.api_keyCreateInput>
    ): Prisma.api_keyCreateInput {
        const factory = new ApiKeyDbFactory();
        return factory.createMinimal({
            ...overrides,
            creditsUsed,
        });
    }

    static createNearLimit(
        percentageUsed: number = 90,
        overrides?: Partial<Prisma.api_keyCreateInput>
    ): Prisma.api_keyCreateInput {
        const factory = new ApiKeyDbFactory();
        if (percentageUsed >= 99) {
            return factory.createEdgeCase("nearHardLimit");
        } else if (percentageUsed >= 95) {
            return factory.createEdgeCase("nearSoftLimit");
        } else {
            const limitHard = BigInt(25000000000);
            const creditsUsed = BigInt(Math.floor(Number(limitHard) * (percentageUsed / 100)));
            return factory.createMinimal({
                ...overrides,
                limitHard,
                creditsUsed,
                stopAtLimit: true,
            });
        }
    }

    static createMultiple(
        count: number,
        userId?: string,
        teamId?: string
    ): Prisma.api_keyCreateInput[] {
        const factory = new ApiKeyDbFactory();
        const keys = [];
        
        for (let i = 0; i < count; i++) {
            const overrides: Partial<Prisma.api_keyCreateInput> = {
                name: `Test API Key ${i + 1}`,
            };
            
            if (userId) {
                overrides.user = { connect: { id: BigInt(userId) } };
            }
            if (teamId) {
                overrides.team = { connect: { id: BigInt(teamId) } };
            }
            
            keys.push(factory.createMinimal(overrides));
        }
        
        return keys;
    }
}

/**
 * Enhanced helper to seed API keys with comprehensive options
 */
export async function seedApiKeys(
    prisma: any,
    options: {
        userId?: string;
        teamId?: string;
        count?: number;
        withPermissions?: boolean;
        withUsage?: boolean;
        includeDisabled?: boolean;
    }
): Promise<BulkSeedResult<any>> {
    const factory = new ApiKeyDbFactory();
    const keys = [];
    const count = options.count || 1;
    let permissionCount = 0;
    let usageCount = 0;
    let disabledCount = 0;

    for (let i = 0; i < count; i++) {
        let keyData: Prisma.api_keyCreateInput;

        if (options.withPermissions && i === 0) {
            // First key has custom permissions
            keyData = factory.createEdgeCase("complexPermissions");
            keyData.name = `API Key ${i + 1} (Permissions)`;
            permissionCount++;
        } else if (options.withUsage && i === 1) {
            // Second key has usage
            keyData = factory.createEdgeCase("nearSoftLimit");
            keyData.name = `API Key ${i + 1} (Usage)`;
            usageCount++;
        } else if (options.includeDisabled && i === count - 1) {
            // Last key is disabled
            keyData = factory.createEdgeCase("disabledApiKey");
            keyData.name = `API Key ${i + 1} (Disabled)`;
            disabledCount++;
        } else {
            // Regular key
            keyData = factory.createMinimal({
                name: `API Key ${i + 1}`,
            });
        }

        // Add user or team connection
        if (options.userId) {
            keyData.user = { connect: { id: BigInt(options.userId) } };
        }
        if (options.teamId) {
            keyData.team = { connect: { id: BigInt(options.teamId) } };
        }

        const apiKey = await prisma.api_key.create({ data: keyData });
        keys.push(apiKey);
    }

    return {
        records: keys,
        summary: {
            total: keys.length,
            withAuth: 0, // ApiKeys don't have auth directly
            bots: 0, // ApiKeys aren't bots
            teams: options.teamId ? keys.length : 0,
        },
    };
}

/**
 * Enhanced helper to create an API key that can be used for authentication testing
 */
export async function createTestApiKey(
    prisma: any,
    userId: string,
    permissions?: Record<string, any>
): Promise<{ id: string; key: string; permissions: any }> {
    const factory = new ApiKeyDbFactory();
    const keyData = permissions 
        ? ApiKeyDbFactory.createWithPermissions(permissions, { user: { connect: { id: BigInt(userId) } } })
        : ApiKeyDbFactory.createForUser(userId);

    const apiKey = await prisma.api_key.create({ data: keyData });
    
    return {
        id: apiKey.id,
        key: apiKey.key,
        permissions: apiKey.permissions,
    };
}

/**
 * Enhanced helper to simulate API key usage over time with validation
 */
export async function simulateApiKeyUsage(
    prisma: any,
    apiKeyId: string,
    operations: Array<{ credits: bigint; timestamp?: Date; validateLimits?: boolean }>
): Promise<{ totalCredits: bigint; limitExceeded: boolean; warnings: string[] }> {
    let totalCredits = BigInt(0);
    let limitExceeded = false;
    const warnings: string[] = [];

    // Get current API key state
    const apiKey = await prisma.api_key.findUnique({
        where: { id: apiKeyId },
        select: { creditsUsed: true, limitHard: true, limitSoft: true, stopAtLimit: true }
    });

    if (!apiKey) {
        throw new Error(`API key with ID ${apiKeyId} not found`);
    }

    let currentCredits = apiKey.creditsUsed || BigInt(0);

    for (const op of operations) {
        if (op.credits < BigInt(0)) {
            warnings.push(`Negative credits in operation: ${op.credits}`);
            continue;
        }

        const newCredits = currentCredits + op.credits;
        totalCredits += op.credits;

        // Check limits if validation is enabled
        if (op.validateLimits !== false) {
            if (apiKey.limitSoft && newCredits > apiKey.limitSoft) {
                warnings.push(`Soft limit exceeded: ${newCredits} > ${apiKey.limitSoft}`);
            }
            if (apiKey.limitHard && newCredits > apiKey.limitHard) {
                warnings.push(`Hard limit exceeded: ${newCredits} > ${apiKey.limitHard}`);
                limitExceeded = true;
                if (apiKey.stopAtLimit) {
                    warnings.push("Operation would be blocked due to stopAtLimit=true");
                    break;
                }
            }
        }
        
        await prisma.api_key.update({
            where: { id: apiKeyId },
            data: {
                creditsUsed: {
                    increment: op.credits,
                },
                updatedAt: op.timestamp || new Date(),
            },
        });

        currentCredits = newCredits;
    }

    return { totalCredits, limitExceeded, warnings };
}

/**
 * Helper to create API key usage scenarios for testing
 */
export function createUsageScenarios() {
    return {
        lightUsage: [
            { credits: BigInt(1000000), timestamp: new Date(Date.now() - 86400000) }, // 1 day ago
            { credits: BigInt(500000), timestamp: new Date(Date.now() - 43200000) }, // 12 hours ago
            { credits: BigInt(250000), timestamp: new Date() }, // now
        ],
        moderateUsage: [
            { credits: BigInt(5000000000), timestamp: new Date(Date.now() - 86400000) },
            { credits: BigInt(3000000000), timestamp: new Date(Date.now() - 43200000) },
            { credits: BigInt(2000000000), timestamp: new Date() },
        ],
        heavyUsage: [
            { credits: BigInt(10000000000), timestamp: new Date(Date.now() - 86400000) },
            { credits: BigInt(8000000000), timestamp: new Date(Date.now() - 43200000) },
            { credits: BigInt(6000000000), timestamp: new Date() },
        ],
        burstUsage: [
            { credits: BigInt(20000000000), timestamp: new Date(Date.now() - 3600000) }, // 1 hour ago - huge burst
            { credits: BigInt(100000), timestamp: new Date() }, // now - small usage
        ],
        gradualIncrease: Array.from({ length: 10 }, (_, i) => ({
            credits: BigInt((i + 1) * 1000000000), // Increasing credits
            timestamp: new Date(Date.now() - (10 - i) * 3600000), // Over 10 hours
        })),
    };
}

/**
 * Helper to validate API key fixture data
 */
export function validateApiKeyFixture(data: Prisma.api_keyCreateInput): {
    isValid: boolean;
    errors: string[];
    warnings: string[];
} {
    const factory = new ApiKeyDbFactory();
    return factory.validateFixture(data);
}

/**
 * Helper to get all available API key test scenarios
 */
export function getApiKeyTestScenarios() {
    const factory = new ApiKeyDbFactory();
    return {
        edgeCases: factory.getAvailableEdgeCases(),
        invalidScenarios: factory.getAvailableInvalidScenarios(),
        usageScenarios: Object.keys(createUsageScenarios()),
    };
}

/**
 * Helper to create comprehensive API key test suite
 */
export async function createApiKeyTestSuite(
    prisma: any,
    options: {
        userId?: string;
        teamId?: string;
        includeEdgeCases?: boolean;
        includeInvalidScenarios?: boolean;
        includeUsageScenarios?: boolean;
    } = {}
): Promise<{
    valid: any[];
    edgeCases: any[];
    invalid: any[];
    withUsage: any[];
}> {
    const factory = new ApiKeyDbFactory();
    const results = {
        valid: [],
        edgeCases: [],
        invalid: [],
        withUsage: [],
    };

    // Create basic valid API keys
    const minimalKey = await prisma.api_key.create({
        data: factory.createMinimal({
            user: options.userId ? { connect: { id: BigInt(options.userId) } } : undefined,
            team: options.teamId ? { connect: { id: BigInt(options.teamId) } } : undefined,
        }),
    });
    results.valid.push(minimalKey);

    const completeKey = await prisma.api_key.create({
        data: factory.createComplete({
            user: options.userId ? { connect: { id: BigInt(options.userId) } } : undefined,
            team: options.teamId ? { connect: { id: BigInt(options.teamId) } } : undefined,
        }),
    });
    results.valid.push(completeKey);

    // Create edge cases if requested
    if (options.includeEdgeCases) {
        for (const edgeCase of factory.getAvailableEdgeCases()) {
            try {
                const key = await prisma.api_key.create({
                    data: factory.createEdgeCase(edgeCase),
                });
                results.edgeCases.push(key);
            } catch (error) {
                // Some edge cases might not be valid for actual creation
                console.warn(`Edge case '${edgeCase}' failed to create:`, error);
            }
        }
    }

    // Create usage scenarios if requested
    if (options.includeUsageScenarios) {
        const usageScenarios = createUsageScenarios();
        for (const [scenarioName, operations] of Object.entries(usageScenarios)) {
            const key = await prisma.api_key.create({
                data: factory.createMinimal({
                    name: `Usage Test - ${scenarioName}`,
                    user: options.userId ? { connect: { id: BigInt(options.userId) } } : undefined,
                }),
            });
            
            try {
                const usageResult = await simulateApiKeyUsage(prisma, key.id, operations);
                results.withUsage.push({ ...key, usageResult });
            } catch (error) {
                console.warn(`Usage scenario '${scenarioName}' failed:`, error);
            }
        }
    }

    return results;
}