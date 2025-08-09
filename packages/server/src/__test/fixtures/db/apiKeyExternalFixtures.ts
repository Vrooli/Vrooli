/* eslint-disable no-magic-numbers */
import { type Prisma, type PrismaClient } from "@prisma/client";
import { generatePK } from "@vrooli/shared";

/**
 * Database fixtures for ApiKeyExternal model - used for seeding test data
 * These fixtures represent external API keys for third-party services (e.g., OpenAI, GitHub, etc.)
 */

// Consistent IDs for testing - using lazy initialization
const _apiKeyExternalDbIds: { [key: string]: bigint | undefined } = {};

export const apiKeyExternalDbIds = {
    get openai1() { return _apiKeyExternalDbIds.openai1 ??= generatePK(); },
    get openai2() { return _apiKeyExternalDbIds.openai2 ??= generatePK(); },
    get github1() { return _apiKeyExternalDbIds.github1 ??= generatePK(); },
    get stripe1() { return _apiKeyExternalDbIds.stripe1 ??= generatePK(); },
    get twilio1() { return _apiKeyExternalDbIds.twilio1 ??= generatePK(); },
    get disabled1() { return _apiKeyExternalDbIds.disabled1 ??= generatePK(); },
};

// Cached IDs for consistent user/team relationships
const _apiKeyExternalRelationIds: { [key: string]: bigint | undefined } = {};
export const apiKeyExternalRelationIds = {
    get user1() { return _apiKeyExternalRelationIds.user1 ??= generatePK(); },
    get user2() { return _apiKeyExternalRelationIds.user2 ??= generatePK(); },
    get team1() { return _apiKeyExternalRelationIds.team1 ??= generatePK(); },
    get team2() { return _apiKeyExternalRelationIds.team2 ??= generatePK(); },
    get resource1() { return _apiKeyExternalRelationIds.resource1 ??= generatePK(); },
};

/**
 * Minimal API key for basic testing
 */
export const minimalApiKeyExternalDb: Prisma.api_key_externalCreateInput = {
    id: apiKeyExternalDbIds.openai1,
    name: "Test OpenAI Key",
    service: "openai",
    key: "sk-test123456789abcdef",
};

/**
 * Complete API key with user association
 */
export const userApiKeyExternalDb: Prisma.api_key_externalCreateInput = {
    id: apiKeyExternalDbIds.github1,
    name: "GitHub Personal Access Token",
    service: "github",
    key: "ghp_testtoken123456789",
    user: { connect: { id: apiKeyExternalRelationIds.user1 } },
};

/**
 * API key associated with a team
 */
export const teamApiKeyExternalDb: Prisma.api_key_externalCreateInput = {
    id: apiKeyExternalDbIds.stripe1,
    name: "Team Stripe API Key",
    service: "stripe",
    key: "sk_test_fake_stripe_key_for_testing_12345",
    team: { connect: { id: apiKeyExternalRelationIds.team1 } },
};

/**
 * Disabled API key
 */
export const disabledApiKeyExternalDb: Prisma.api_key_externalCreateInput = {
    id: apiKeyExternalDbIds.disabled1,
    name: "Disabled Twilio Key",
    service: "twilio",
    key: "AC_test_disabled_key_123",
    disabledAt: new Date("2024-01-01T00:00:00Z"),
};

/**
 * Factory for creating API key external database fixtures with overrides
 */
export class ApiKeyExternalDbFactory {
    /**
     * Create a minimal API key
     */
    static createMinimal(overrides?: Partial<Prisma.api_key_externalCreateInput>): Prisma.api_key_externalCreateInput {
        return {
            ...minimalApiKeyExternalDb,
            id: generatePK(),
            ...overrides,
        };
    }

    /**
     * Create an API key for a specific user
     */
    static createForUser(
        userId: bigint,
        service: string,
        overrides?: Partial<Prisma.api_key_externalCreateInput>,
    ): Prisma.api_key_externalCreateInput {
        return {
            id: generatePK(),
            name: `${service} API Key`,
            service,
            key: `test_${service}_key_${Date.now()}`,
            user: { connect: { id: userId } },
            ...overrides,
        };
    }

    /**
     * Create an API key for a specific team
     */
    static createForTeam(
        teamId: bigint,
        service: string,
        overrides?: Partial<Prisma.api_key_externalCreateInput>,
    ): Prisma.api_key_externalCreateInput {
        return {
            id: generatePK(),
            name: `Team ${service} API Key`,
            service,
            key: `test_team_${service}_key_${Date.now()}`,
            team: { connect: { id: teamId } },
            ...overrides,
        };
    }

    /**
     * Create a disabled API key
     */
    static createDisabled(
        service: string,
        overrides?: Partial<Prisma.api_key_externalCreateInput>,
    ): Prisma.api_key_externalCreateInput {
        return {
            id: generatePK(),
            name: `Disabled ${service} Key`,
            service,
            key: `test_disabled_${service}_key_${Date.now()}`,
            disabledAt: new Date(),
            ...overrides,
        };
    }

    /**
     * Create an API key with a linked resource
     */
    static createWithResource(
        resourceId: bigint,
        service: string,
        overrides?: Partial<Prisma.api_key_externalCreateInput>,
    ): Prisma.api_key_externalCreateInput {
        return {
            id: generatePK(),
            name: `${service} Key with Resource`,
            service,
            key: `test_${service}_resource_key_${Date.now()}`,
            resource: { connect: { id: resourceId } },
            ...overrides,
        };
    }

    /**
     * Create multiple API keys for common services
     */
    static createCommonServices(
        ownerId: bigint,
        ownerType: "user" | "team" = "user",
    ): Prisma.api_key_externalCreateInput[] {
        const services = [
            { service: "openai", name: "OpenAI API Key", prefix: "sk-" },
            { service: "github", name: "GitHub Personal Access Token", prefix: "ghp_" },
            { service: "stripe", name: "Stripe API Key", prefix: "sk_test_" },
            { service: "twilio", name: "Twilio Account SID", prefix: "AC" },
            { service: "sendgrid", name: "SendGrid API Key", prefix: "SG." },
        ];

        return services.map(({ service, name, prefix }) => ({
            id: generatePK(),
            name,
            service,
            key: `${prefix}test_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
            ...(ownerType === "user"
                ? { user: { connect: { id: ownerId } } }
                : { team: { connect: { id: ownerId } } }
            ),
        }));
    }
}

/**
 * Helper to seed API keys for testing
 */
export async function seedApiKeysExternal(
    prisma: PrismaClient,
    options: {
        userId?: bigint;
        teamId?: bigint;
        services?: string[];
        count?: number;
        includeDisabled?: boolean;
    },
) {
    const apiKeys = [];
    const servicesToCreate = options.services || ["openai", "github"];
    const count = options.count || servicesToCreate.length;

    if (options.userId) {
        // Create user API keys
        for (let i = 0; i < Math.min(count, servicesToCreate.length); i++) {
            const apiKey = await prisma.api_key_external.create({
                data: ApiKeyExternalDbFactory.createForUser(
                    options.userId,
                    servicesToCreate[i],
                    { name: `User ${servicesToCreate[i]} Key ${i + 1}` },
                ),
            });
            apiKeys.push(apiKey);
        }
    }

    if (options.teamId) {
        // Create team API keys
        for (let i = 0; i < Math.min(count, servicesToCreate.length); i++) {
            const apiKey = await prisma.api_key_external.create({
                data: ApiKeyExternalDbFactory.createForTeam(
                    options.teamId,
                    servicesToCreate[i],
                    { name: `Team ${servicesToCreate[i]} Key ${i + 1}` },
                ),
            });
            apiKeys.push(apiKey);
        }
    }

    if (options.includeDisabled) {
        // Add a disabled key
        const disabledKey = await prisma.api_key_external.create({
            data: ApiKeyExternalDbFactory.createDisabled(
                servicesToCreate[0],
                options.userId ? { user: { connect: { id: options.userId } } } : undefined,
            ),
        });
        apiKeys.push(disabledKey);
    }

    return apiKeys;
}

/**
 * Helper to clean up API keys after tests
 */
export async function cleanupApiKeysExternal(
    prisma: PrismaClient,
    apiKeyIds: bigint[],
) {
    await prisma.api_key_external.deleteMany({
        where: { id: { in: apiKeyIds } },
    });
}
