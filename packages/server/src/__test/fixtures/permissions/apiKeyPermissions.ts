import { ApiKeyType, generatePK } from "@vrooli/shared";

// Type definition for API key authentication data
export interface ApiKeyFullAuthData {
    __type: ApiKeyType;
    id: string;
    userId: string;
    permissions: {
        read: "None" | "Public" | "Private" | "Auth";
        write: "None" | "Public" | "Private" | "Auth";
        bot: boolean;
        daily_credits: number;
    };
    csrfToken: string;
    isLoggedIn: boolean;
    languages: string[];
    timeZone: string;
    theme: string;
    isExpired?: boolean;
    isRevoked?: boolean;
}

/**
 * API Key permission fixtures for testing various API access scenarios
 */

// Base API key data
const baseApiKeyData = {
    csrfToken: "test-csrf-token",
    isLoggedIn: true,
    languages: ["en"],
    timeZone: "UTC",
    theme: "light",
} as const;

/**
 * Read-only API key - can only read public data
 */
export const readOnlyPublicApiKey: ApiKeyFullAuthData = {
    ...baseApiKeyData,
    __type: ApiKeyType.Internal,
    id: "api_100000000000000001",
    userId: "222222222222222222", // standardUser
    permissions: {
        read: "Public",
        write: "None",
        bot: false,
        daily_credits: 100,
    },
};

/**
 * Read private API key - can read public and private user data
 */
export const readPrivateApiKey: ApiKeyFullAuthData = {
    ...baseApiKeyData,
    __type: ApiKeyType.Internal,
    id: "api_200000000000000002",
    userId: "222222222222222222", // standardUser
    permissions: {
        read: "Private",
        write: "None",
        bot: false,
        daily_credits: 1000,
    },
};

/**
 * Write-enabled API key - full CRUD permissions on user data
 */
export const writeApiKey: ApiKeyFullAuthData = {
    ...baseApiKeyData,
    __type: ApiKeyType.Internal,
    id: "api_300000000000000003",
    userId: "333333333333333333", // premiumUser
    permissions: {
        read: "Private",
        write: "Private",
        bot: false,
        daily_credits: 10000,
    },
};

/**
 * Bot API key - special permissions for automated operations
 */
export const botApiKey: ApiKeyFullAuthData = {
    ...baseApiKeyData,
    __type: ApiKeyType.Internal,
    id: "api_400000000000000004",
    userId: "666666666666666666", // botUser
    permissions: {
        read: "Private",
        write: "Private",
        bot: true,
        daily_credits: 100000,
    },
};

/**
 * External API key - third-party integration with limited scope
 */
export const externalApiKey: ApiKeyFullAuthData = {
    ...baseApiKeyData,
    __type: ApiKeyType.External,
    id: "api_500000000000000005",
    userId: "222222222222222222",
    permissions: {
        read: "Public",
        write: "None",
        bot: false,
        daily_credits: 500,
    },
};

/**
 * Rate-limited API key - for testing quota enforcement
 */
export const rateLimitedApiKey: ApiKeyFullAuthData = {
    ...baseApiKeyData,
    __type: ApiKeyType.Internal,
    id: "api_600000000000000006",
    userId: "222222222222222222",
    permissions: {
        read: "Public",
        write: "None",
        bot: false,
        daily_credits: 10, // Very low limit for testing
    },
};

/**
 * API key with auth permissions - can read auth-specific data
 */
export const authReadApiKey: ApiKeyFullAuthData = {
    ...baseApiKeyData,
    __type: ApiKeyType.Internal,
    id: "api_700000000000000007",
    userId: "333333333333333333",
    permissions: {
        read: "Auth",
        write: "None",
        bot: false,
        daily_credits: 5000,
    },
};

/**
 * API key with auth write permissions - full access including auth operations
 */
export const authWriteApiKey: ApiKeyFullAuthData = {
    ...baseApiKeyData,
    __type: ApiKeyType.Internal,
    id: "api_800000000000000008",
    userId: "111111111111111111", // adminUser
    permissions: {
        read: "Auth",
        write: "Auth",
        bot: false,
        daily_credits: 50000,
    },
};

/**
 * Expired API key - should be rejected for all operations
 */
export const expiredApiKey: ApiKeyFullAuthData = {
    ...baseApiKeyData,
    __type: ApiKeyType.Internal,
    id: "api_900000000000000009",
    userId: "222222222222222222",
    permissions: {
        read: "Private",
        write: "Private",
        bot: false,
        daily_credits: 0, // Expired keys have no credits
    },
    isExpired: true,
};

/**
 * Revoked API key - manually disabled, should be rejected
 */
export const revokedApiKey: ApiKeyFullAuthData = {
    ...baseApiKeyData,
    __type: ApiKeyType.Internal,
    id: "api_101010101010101010",
    userId: "333333333333333333",
    permissions: {
        read: "None",
        write: "None",
        bot: false,
        daily_credits: 0,
    },
    isRevoked: true,
};

/**
 * Helper to create custom API key configurations
 */
export function createApiKeyWithPermissions(
    userId: string,
    permissions: Partial<ApiKeyFullAuthData["permissions"]>,
    overrides: Partial<ApiKeyFullAuthData> = {},
): ApiKeyFullAuthData {
    return {
        ...baseApiKeyData,
        __type: ApiKeyType.Internal,
        id: `api_${generatePK()}`,
        userId,
        permissions: {
            read: "Public",
            write: "None",
            bot: false,
            daily_credits: 1000,
            ...permissions,
        },
        ...overrides,
    };
}

/**
 * Create a set of API keys for testing different permission levels
 */
export function createApiKeySet(userId: string): {
    none: ApiKeyFullAuthData;
    readPublic: ApiKeyFullAuthData;
    readPrivate: ApiKeyFullAuthData;
    writePrivate: ApiKeyFullAuthData;
    full: ApiKeyFullAuthData;
} {
    return {
        none: createApiKeyWithPermissions(userId, { read: "None", write: "None" }),
        readPublic: createApiKeyWithPermissions(userId, { read: "Public", write: "None" }),
        readPrivate: createApiKeyWithPermissions(userId, { read: "Private", write: "None" }),
        writePrivate: createApiKeyWithPermissions(userId, { read: "Private", write: "Private" }),
        full: createApiKeyWithPermissions(userId, { read: "Auth", write: "Auth" }),
    };
}