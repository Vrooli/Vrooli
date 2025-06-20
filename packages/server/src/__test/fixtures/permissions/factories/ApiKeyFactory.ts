/**
 * API Key Factory
 * 
 * Factory for creating API key fixtures with various permission scopes
 * and configurations.
 */

import { type ApiKeyAuthData, ApiKeyType } from "../types.js";
import { BasePermissionFactory } from "./BasePermissionFactory.js";

/**
 * Constants for API key generation
 */
const API_KEY_ID_PADDING = 17;
const DEFAULT_DAILY_CREDITS = 100;
const SERVICE_NAME_PADDING = 12;

/**
 * Factory for creating API key fixtures
 */
export class ApiKeyFactory extends BasePermissionFactory<ApiKeyAuthData> {
    
    /**
     * ID counter for generating consistent test IDs
     */
    private idCounter = 0;

    /**
     * Base API key data
     */
    protected readonly baseApiKeyData = {
        csrfToken: "test-csrf-token",
        isLoggedIn: true,
        languages: ["en"],
        timeZone: "UTC",
        theme: "light" as const,
    };

    /**
     * Create a session with specific overrides
     */
    createSession(overrides?: Partial<ApiKeyAuthData>): ApiKeyAuthData {
        this.idCounter++;
        
        const baseKey: ApiKeyAuthData = {
            ...this.baseApiKeyData,
            __type: ApiKeyType.Internal,
            id: `api_${String(this.idCounter).padStart(API_KEY_ID_PADDING, "0")}`,
            userId: "222222222222222222", // Default to standard user
            permissions: {
                read: "None",
                write: "None",
                bot: false,
                daily_credits: DEFAULT_DAILY_CREDITS,
            },
        };

        return this.mergeWithDefaults(baseKey, overrides);
    }

    /**
     * Add API key specific permissions
     */
    withApiKeyPermissions(
        key: ApiKeyAuthData,
        permissions: Partial<ApiKeyAuthData["permissions"]>,
    ): ApiKeyAuthData {
        return {
            ...key,
            permissions: {
                ...key.permissions,
                ...permissions,
            },
        };
    }

    /**
     * Create a read-only public API key
     */
    createReadOnlyPublic(overrides?: Partial<ApiKeyAuthData>): ApiKeyAuthData {
        return this.createSession({
            id: "api_100000000000000001",
            permissions: {
                read: "Public",
                write: "None",
                bot: false,
                daily_credits: 100,
            },
            ...overrides,
        });
    }

    /**
     * Create a read private API key
     */
    createReadPrivate(overrides?: Partial<ApiKeyAuthData>): ApiKeyAuthData {
        return this.createSession({
            id: "api_100000000000000002",
            permissions: {
                read: "Private",
                write: "None",
                bot: false,
                daily_credits: 500,
            },
            ...overrides,
        });
    }

    /**
     * Create a write-enabled API key
     */
    createWrite(overrides?: Partial<ApiKeyAuthData>): ApiKeyAuthData {
        return this.createSession({
            id: "api_100000000000000003",
            permissions: {
                read: "Private",
                write: "Private",
                bot: false,
                daily_credits: 1000,
            },
            ...overrides,
        });
    }

    /**
     * Create a bot API key
     */
    createBot(overrides?: Partial<ApiKeyAuthData>): ApiKeyAuthData {
        return this.createSession({
            id: "api_100000000000000004",
            __type: ApiKeyType.Internal,
            userId: "777777777777777777", // Bot user
            permissions: {
                read: "Auth",
                write: "Auth",
                bot: true,
                daily_credits: 10000,
            },
            ...overrides,
        });
    }

    /**
     * Create an external API key
     */
    createExternal(overrides?: Partial<ApiKeyAuthData>): ApiKeyAuthData {
        return this.createSession({
            id: "api_100000000000000005",
            __type: ApiKeyType.External,
            permissions: {
                read: "Public",
                write: "None",
                bot: false,
                daily_credits: 50,
            },
            ...overrides,
        });
    }

    /**
     * Create a rate-limited API key
     */
    createRateLimited(overrides?: Partial<ApiKeyAuthData>): ApiKeyAuthData {
        return this.createSession({
            id: "api_100000000000000006",
            permissions: {
                read: "Public",
                write: "None",
                bot: false,
                daily_credits: 10, // Very low limit
            },
            ...overrides,
        });
    }

    /**
     * Create an auth read API key
     */
    createAuthRead(overrides?: Partial<ApiKeyAuthData>): ApiKeyAuthData {
        return this.createSession({
            id: "api_100000000000000007",
            permissions: {
                read: "Auth",
                write: "None",
                bot: false,
                daily_credits: 1000,
            },
            ...overrides,
        });
    }

    /**
     * Create an auth write API key
     */
    createAuthWrite(overrides?: Partial<ApiKeyAuthData>): ApiKeyAuthData {
        return this.createSession({
            id: "api_100000000000000008",
            permissions: {
                read: "Auth",
                write: "Auth",
                bot: false,
                daily_credits: 5000,
            },
            ...overrides,
        });
    }

    /**
     * Create an expired API key
     */
    createExpired(overrides?: Partial<ApiKeyAuthData>): ApiKeyAuthData {
        return this.createSession({
            id: "api_100000000000000009",
            isExpired: true,
            permissions: {
                read: "Private",
                write: "Private",
                bot: false,
                daily_credits: 1000,
            },
            ...overrides,
        });
    }

    /**
     * Create a revoked API key
     */
    createRevoked(overrides?: Partial<ApiKeyAuthData>): ApiKeyAuthData {
        return this.createSession({
            id: "api_100000000000000010",
            isRevoked: true,
            permissions: {
                read: "Private",
                write: "Private",
                bot: false,
                daily_credits: 1000,
            },
            ...overrides,
        });
    }

    /**
     * Create an API key with no permissions
     */
    createNoPermissions(overrides?: Partial<ApiKeyAuthData>): ApiKeyAuthData {
        return this.createSession({
            id: "api_100000000000000011",
            permissions: {
                read: "None",
                write: "None",
                bot: false,
                daily_credits: 0,
            },
            ...overrides,
        });
    }

    /**
     * Create an API key with maximum permissions
     */
    createMaxPermissions(overrides?: Partial<ApiKeyAuthData>): ApiKeyAuthData {
        return this.createSession({
            id: "api_100000000000000012",
            permissions: {
                read: "Auth",
                write: "Auth",
                bot: true,
                daily_credits: 999999,
            },
            ...overrides,
        });
    }

    /**
     * Create a set of API keys for comprehensive testing
     */
    createSet(userId: string): ApiKeyAuthData[] {
        return [
            this.createReadOnlyPublic({ userId }),
            this.createReadPrivate({ userId }),
            this.createWrite({ userId }),
            this.createBot({ userId }),
            this.createExternal({ userId }),
        ];
    }

    /**
     * Create API key with custom permission combination
     */
    createCustom(
        read: "None" | "Public" | "Private" | "Auth",
        write: "None" | "Public" | "Private" | "Auth",
        bot = false,
        dailyCredits = 100,
        overrides?: Partial<ApiKeyAuthData>,
    ): ApiKeyAuthData {
        return this.createSession({
            permissions: {
                read,
                write,
                bot,
                daily_credits: dailyCredits,
            },
            ...overrides,
        });
    }

    /**
     * Create an API key for a specific service/integration
     */
    createServiceKey(
        serviceName: string,
        permissions: Partial<ApiKeyAuthData["permissions"]>,
        overrides?: Partial<ApiKeyAuthData>,
    ): ApiKeyAuthData {
        return this.createSession({
            id: `api_svc_${serviceName.toLowerCase().padEnd(SERVICE_NAME_PADDING, "0")}`,
            __type: ApiKeyType.External,
            permissions: {
                read: "Public",
                write: "None",
                bot: false,
                daily_credits: 1000,
                ...permissions,
            },
            ...overrides,
        });
    }
}
