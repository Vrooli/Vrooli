/**
 * API Key Service for External API Authentication
 * 
 * Manages retrieval and decryption of external API keys stored in the database.
 * Features:
 * - Secure key decryption using ApiKeyEncryptionService
 * - Service auto-detection from URLs
 * - Temporary caching for performance
 * - Support for user and team-level API keys
 */
import { MINUTES_5_MS } from "@vrooli/shared";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import type { AuthConfig } from "./httpClient.js";

// Cache for decrypted API keys (short-lived for security)
const API_KEY_CACHE = new Map<string, { key: string; expiry: number }>();
const CACHE_TTL_MS = MINUTES_5_MS;

export interface DecryptedApiKey {
    id: string;
    service: string;
    name: string;
    decryptedKey: string;
    authType: AuthConfig["type"];
    headerName?: string;
    isTeamKey: boolean;
}

export interface UserContext {
    userId: string;
    teamId?: string;
}

export class APIKeyService {
    constructor() {
        // ApiKeyEncryptionService is a singleton, accessed via get()
    }

    /**
     * Get API key for a specific service and user context
     */
    async getApiKey(serviceName: string, context: UserContext): Promise<DecryptedApiKey | null> {
        try {
            // Look for team key first, then user key
            const apiKey = await this.findApiKeyInDatabase(serviceName, context);

            if (!apiKey) {
                logger.debug("[APIKeyService] No API key found", { serviceName, userId: context.userId });
                return null;
            }

            // Convert to format expected by decryptAndCache
            const keyData = {
                id: apiKey.id.toString(),
                key: apiKey.key,
                updatedAt: apiKey.updatedAt,
            };

            // Decrypt the key
            const decryptedKey = await this.decryptAndCache(keyData);

            return {
                id: apiKey.id.toString(),
                service: apiKey.service,
                name: apiKey.name,
                decryptedKey,
                authType: "bearer", // Default, should be overridden by config
                headerName: undefined, // Should be provided by config
                isTeamKey: !!apiKey.teamId,
            };

        } catch (error) {
            logger.error("[APIKeyService] Failed to retrieve API key", {
                serviceName,
                userId: context.userId,
                error: error instanceof Error ? error.message : String(error),
            });
            return null;
        }
    }

    /**
     * Get all API keys for a service (useful for fallback scenarios)
     */
    async getApiKeysByService(serviceName: string, context: UserContext): Promise<DecryptedApiKey[]> {
        try {
            // Build OR conditions
            const orConditions: any[] = [{ userId: context.userId }];
            if (context.teamId) {
                try {
                    orConditions.push({ teamId: BigInt(context.teamId) });
                } catch (e) {
                    logger.warn("[APIKeyService] Invalid teamId format", { teamId: context.teamId });
                }
            }

            const apiKeys = await DbProvider.get().api_key_external.findMany({
                where: {
                    service: serviceName,
                    OR: orConditions,
                },
                orderBy: [
                    { teamId: { sort: "desc", nulls: "last" } }, // Team keys first
                    { createdAt: "desc" }, // Most recent first
                ],
            });

            const decryptedKeys: DecryptedApiKey[] = [];

            for (const apiKey of apiKeys) {
                try {
                    // Convert to format expected by decryptAndCache
                    const keyData = {
                        id: apiKey.id.toString(),
                        key: apiKey.key,
                        updatedAt: apiKey.updatedAt,
                    };
                    
                    const decryptedKey = await this.decryptAndCache(keyData);
                    decryptedKeys.push({
                        id: apiKey.id.toString(),
                        service: apiKey.service,
                        name: apiKey.name,
                        decryptedKey,
                        authType: "bearer", // Default, should be overridden by config
                        headerName: undefined, // Should be provided by config
                        isTeamKey: !!apiKey.teamId,
                    });
                } catch (decryptError) {
                    logger.warn("[APIKeyService] Failed to decrypt API key", {
                        keyId: apiKey.id.toString(),
                        service: serviceName,
                        error: decryptError instanceof Error ? decryptError.message : String(decryptError),
                    });
                }
            }

            return decryptedKeys;

        } catch (error) {
            logger.error("[APIKeyService] Failed to retrieve API keys", {
                serviceName,
                userId: context.userId,
                error: error instanceof Error ? error.message : String(error),
            });
            return [];
        }
    }


    /**
     * Convert DecryptedApiKey to AuthConfig for HTTPClient
     * Note: Auth type and header names should come from configuration, not hard-coded logic
     */
    createAuthConfig(apiKey: DecryptedApiKey): AuthConfig {
        const authConfig: AuthConfig = {
            type: apiKey.authType,
            token: apiKey.decryptedKey,
        };

        if (apiKey.headerName) {
            authConfig.headerName = apiKey.headerName;
        }

        // For basic auth, check if the key contains username:password format
        if (apiKey.authType === "basic" && apiKey.decryptedKey.includes(":")) {
            const [username, password] = apiKey.decryptedKey.split(":", 2);
            if (username && password) {
                authConfig.username = username;
                authConfig.password = password;
                delete authConfig.token; // Basic auth doesn't use token
            }
        } else if (apiKey.authType === "custom") {
            authConfig.value = apiKey.decryptedKey;
            authConfig.headerName = apiKey.headerName || "Authorization";
        }

        return authConfig;
    }


    /**
     * Find API key in database with proper priority (team > user)
     */
    private async findApiKeyInDatabase(serviceName: string, context: UserContext) {
        // Build OR conditions
        const orConditions: any[] = [{ userId: context.userId }];
        if (context.teamId) {
            try {
                orConditions.push({ teamId: BigInt(context.teamId) });
            } catch (e) {
                logger.warn("[APIKeyService] Invalid teamId format", { teamId: context.teamId });
            }
        }

        return await DbProvider.get().api_key_external.findFirst({
            where: {
                service: serviceName,
                OR: orConditions,
            },
            orderBy: [
                { teamId: { sort: "desc", nulls: "last" } }, // Team keys first
                { createdAt: "desc" }, // Most recent first
            ],
        });
    }

    /**
     * Decrypt API key with caching for performance
     */
    private async decryptAndCache(apiKey: { id: string; key: string; updatedAt?: Date | null }): Promise<string> {
        const cacheKey = `${apiKey.id}:${apiKey.updatedAt?.getTime() || 0}`;

        // Check cache first
        const cached = API_KEY_CACHE.get(cacheKey);
        if (cached && cached.expiry > Date.now()) {
            return cached.key;
        }

        // Decrypt the key
        const decryptedKey = ApiKeyEncryptionService.get().decryptExternal(apiKey.key);

        // Cache the result
        API_KEY_CACHE.set(cacheKey, {
            key: decryptedKey,
            expiry: Date.now() + CACHE_TTL_MS,
        });

        // Clean up expired cache entries
        this.cleanupCache();

        return decryptedKey;
    }

    /**
     * Clean up expired cache entries
     */
    private cleanupCache(): void {
        const now = Date.now();
        const keysToDelete: string[] = [];

        API_KEY_CACHE.forEach((value, key) => {
            if (value.expiry <= now) {
                keysToDelete.push(key);
            }
        });

        keysToDelete.forEach(key => API_KEY_CACHE.delete(key));
    }

    /**
     * Clear all cached keys (for security)
     */
    static clearCache(): void {
        API_KEY_CACHE.clear();
    }
}
