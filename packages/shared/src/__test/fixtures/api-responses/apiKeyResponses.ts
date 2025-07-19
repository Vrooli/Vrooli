/* c8 ignore start */
/**
 * API Key Response Fixtures
 * 
 * Comprehensive fixtures for API key management including
 * key creation, external service integration, and access control.
 */

import type {
    ApiKey,
    ApiKeyCreated,
    ApiKeyCreateInput,
    ApiKeyExternal,
    ApiKeyExternalCreateInput,
    ApiKeyExternalUpdateInput,
    ApiKeyUpdateInput,
} from "../../../api/types.js";
import { generatePK } from "../../../id/index.js";
import { BaseAPIResponseFactory } from "./base.js";
import type { MockDataOptions } from "./types.js";
import { userResponseFactory } from "./userResponses.js";
import {
    API_KEY_LENGTH,
    DEFAULT_COUNT,
    DEFAULT_DELAY_MS,
    DEFAULT_ERROR_RATE,
    MAX_KEYS_PER_USER,
    MAX_NAME_LENGTH,
    MILLISECONDS_PER_HOUR,
    MIN_KEY_LENGTH_FOR_MASKING,
    ONE_DAY_MS,
    THREE_DAYS_MS,
    SEVEN_DAYS_MS,
    NINETY_DAYS_MS,
    TWO_HOURS_TO_MS,
} from "../constants.js";

// API Key specific constants
const USAGE_COUNT_HIGH = 5432;
const USAGE_COUNT_MED = 1547;
const USAGE_COUNT_LOW = 234;
const USAGE_COUNT_VERY_LOW = 156;
const USAGE_COUNT_MINIMAL = 45;
const USAGE_COUNT_LEGACY = 12;
const MASKED_LENGTH = 24;
const PREFIX_LENGTH = 8;
const SERVICE_KEY_MIN_LENGTH = 10;
const AWS_KEY_SUFFIX_LENGTH = 16;
const OPENAI_KEY_SUFFIX_LENGTH = 48;
const STRIPE_KEY_MIN_SUFFIX_LENGTH = 24;
const GOOGLE_KEY_SUFFIX_LENGTH = 35;

// Supported external services
const EXTERNAL_SERVICES = ["openai", "anthropic", "google", "microsoft", "aws", "stripe"] as const;

/**
 * API Key response factory
 */
export class ApiKeyResponseFactory extends BaseAPIResponseFactory<
    ApiKey,
    ApiKeyCreateInput,
    ApiKeyUpdateInput
> {
    protected readonly entityName = "api_key";

    /**
     * Create mock API key data
     */
    createMockData(options?: MockDataOptions): ApiKey {
        const scenario = options?.scenario || "minimal";
        const keyId = String(options?.overrides?.id || generatePK());

        const baseApiKey: ApiKey = {
            __typename: "ApiKey",
            id: keyId,
            creditsUsed: "0",
            limitHard: "100000",
            name: "Test API Key",
            permissions: "read",
            stopAtLimit: false,
        };

        if (scenario === "complete" || scenario === "edge-case") {
            return {
                ...baseApiKey,
                name: scenario === "edge-case" ? "A".repeat(MAX_NAME_LENGTH) : "Production API Key",
                // Note: lastUsed and timesUsed are not part of the ApiKey type
                ...options?.overrides,
            };
        }

        return {
            ...baseApiKey,
            ...options?.overrides,
        };
    }

    /**
     * Create API key from input
     */
    createFromInput(input: ApiKeyCreateInput): ApiKey {
        const keyId = generatePK().toString();

        return {
            __typename: "ApiKey",
            id: keyId,
            name: input.name || "Unnamed API Key",
            creditsUsed: "0",
            limitHard: input.limitHard,
            limitSoft: input.limitSoft,
            permissions: input.permissions,
            stopAtLimit: input.stopAtLimit,
        };
    }

    /**
     * Create API key creation response (with full key)
     */
    createApiKeyCreatedResponse(input: ApiKeyCreateInput): ApiKeyCreated {
        const apiKey = this.createFromInput(input);

        return {
            ...apiKey,
            key: this.generateFullKey(), // Full key is only shown once
        };
    }

    /**
     * Update API key from input
     */
    updateFromInput(existing: ApiKey, input: ApiKeyUpdateInput): ApiKey {
        const updates: Partial<ApiKey> = {};

        if (input.name !== undefined) updates.name = input.name;

        return {
            ...existing,
            ...updates,
        };
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: ApiKeyCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.id) {
            errors.id = "ID is required";
        }

        if (input.name && input.name.length > MAX_NAME_LENGTH) {
            errors.name = `Name must be ${MAX_NAME_LENGTH} characters or less`;
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: ApiKeyUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (input.name !== undefined && input.name && input.name.length > MAX_NAME_LENGTH) {
            errors.name = `Name must be ${MAX_NAME_LENGTH} characters or less`;
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create API keys for a user
     */
    createApiKeysForUser(userId: string, count = 3): ApiKey[] {
        const user = userResponseFactory.createMockData({ overrides: { id: userId } });

        return Array.from({ length: count }, (_, index) =>
            this.createMockData({
                overrides: {
                    id: `api_key_${userId}_${index}`,
                    user,
                    name: `API Key ${index + 1}`,
                    // Note: lastUsed and timesUsed are not part of the ApiKey type
                },
            }),
        );
    }

    /**
     * Create API keys with different usage patterns
     */
    createApiKeysWithUsagePatterns(): ApiKey[] {
        const baseTime = Date.now();

        return [
            // Frequently used key
            this.createMockData({
                overrides: {
                    name: "Production API Key",
                    // Note: lastUsed and timesUsed are not part of the ApiKey type
                },
            }),

            // Occasionally used key
            this.createMockData({
                overrides: {
                    name: "Development API Key",
                    // Note: lastUsed and timesUsed are not part of the ApiKey type
                },
            }),

            // Unused key
            this.createMockData({
                overrides: {
                    name: "Staging API Key",
                    // Note: lastUsed and timesUsed are not part of the ApiKey type
                },
            }),

            // Old unused key
            this.createMockData({
                overrides: {
                    name: "Legacy API Key",
                    // Note: lastUsed and timesUsed are not part of the ApiKey type
                },
            }),
        ];
    }

    /**
     * Create key limit error response
     */
    createKeyLimitErrorResponse(currentCount = MAX_KEYS_PER_USER) {
        return this.createBusinessErrorResponse("limit", {
            resource: "api_key",
            limit: MAX_KEYS_PER_USER,
            current: currentCount,
            message: `Maximum number of API keys (${MAX_KEYS_PER_USER}) reached`,
        });
    }

    /**
     * Generate a masked API key (for display)
     */
    private generateMaskedKey(): string {
        const prefix = "vk_";
        const visible = this.generateRandomString(PREFIX_LENGTH);
        const masked = "•".repeat(MASKED_LENGTH);
        return `${prefix}${visible}${masked}`;
    }

    /**
     * Generate a full API key (for creation response)
     */
    private generateFullKey(): string {
        const prefix = "vk_";
        const key = this.generateRandomString(API_KEY_LENGTH - prefix.length);
        return `${prefix}${key}`;
    }

    /**
     * Generate random string for keys
     */
    private generateRandomString(length: number): string {
        const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
        let result = "";
        for (let i = 0; i < length; i++) {
            result += chars.charAt(Math.floor(Math.random() * chars.length));
        }
        return result;
    }
}

/**
 * External API Key response factory
 */
export class ApiKeyExternalResponseFactory extends BaseAPIResponseFactory<
    ApiKeyExternal,
    ApiKeyExternalCreateInput,
    ApiKeyExternalUpdateInput
> {
    protected readonly entityName = "api_key_external";

    /**
     * Create mock external API key data
     */
    createMockData(options?: MockDataOptions): ApiKeyExternal {
        const scenario = options?.scenario || "minimal";
        const keyId = String(options?.overrides?.id || generatePK());

        const baseExternalKey: ApiKeyExternal = {
            __typename: "ApiKeyExternal",
            id: keyId,
            name: "OpenAI API Key",
            service: "openai",
        };

        if (scenario === "complete" || scenario === "edge-case") {
            return {
                ...baseExternalKey,
                name: scenario === "edge-case" ? "Edge Case Service Key" : "Production OpenAI Key",
                service: scenario === "edge-case" ? "unknown_service" : "anthropic",
                ...options?.overrides,
            };
        }

        return {
            ...baseExternalKey,
            ...options?.overrides,
        };
    }

    /**
     * Create external API key from input
     */
    createFromInput(input: ApiKeyExternalCreateInput): ApiKeyExternal {
        const keyId = generatePK().toString();

        return {
            __typename: "ApiKeyExternal",
            id: keyId,
            name: input.name || `${input.service} API Key`,
            service: input.service,
        };
    }

    /**
     * Update external API key from input
     */
    updateFromInput(existing: ApiKeyExternal, input: ApiKeyExternalUpdateInput): ApiKeyExternal {
        const updates: Partial<ApiKeyExternal> = {};

        if (input.name !== undefined) updates.name = input.name;
        if (input.key !== undefined) {
            // Note: ApiKeyExternal type doesn't include the key field in the response
            // Key updates are processed but not returned in the public API
        }

        return {
            ...existing,
            ...updates,
        };
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: ApiKeyExternalCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        // Note: id is not part of ApiKeyExternalCreateInput - it's generated automatically

        if (!input.service) {
            errors.service = "Service is required";
        } else if (!EXTERNAL_SERVICES.includes(input.service as any)) {
            errors.service = `Service must be one of: ${EXTERNAL_SERVICES.join(", ")}`;
        }

        if (!input.key) {
            errors.key = "API key is required";
        } else if (!this.validateKeyFormat(input.key, input.service)) {
            errors.key = `Invalid API key format for ${input.service}`;
        }

        if (input.name && input.name.length > MAX_NAME_LENGTH) {
            errors.name = `Name must be ${MAX_NAME_LENGTH} characters or less`;
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: ApiKeyExternalUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (input.name !== undefined && input.name && input.name.length > MAX_NAME_LENGTH) {
            errors.name = `Name must be ${MAX_NAME_LENGTH} characters or less`;
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create external keys for all services
     */
    createExternalKeysForAllServices(): ApiKeyExternal[] {
        return EXTERNAL_SERVICES.map((service, index) =>
            this.createMockData({
                overrides: {
                    id: `external_key_${service}_${index}`,
                    service,
                    name: `${service.charAt(0).toUpperCase() + service.slice(1)} API Key`,
                        },
            }),
        );
    }

    /**
     * Create invalid service key error response
     */
    createInvalidServiceErrorResponse(service: string) {
        return this.createValidationErrorResponse({
            service: `Unsupported service: ${service}. Supported services: ${EXTERNAL_SERVICES.join(", ")}`,
        });
    }

    /**
     * Create invalid key format error response
     */
    createInvalidKeyFormatErrorResponse(service: string) {
        return this.createValidationErrorResponse({
            key: `Invalid API key format for ${service}`,
        });
    }

    /**
     * Mask external API key based on service
     */
    private maskExternalKey(key: string, service: string): string {
        if (key.length < MIN_KEY_LENGTH_FOR_MASKING) return key;

        const prefixLength = this.getServicePrefixLength(service);
        const prefix = key.substring(0, prefixLength);
        const suffix = key.substring(key.length - 4);
        const masked = "•".repeat(Math.max(0, key.length - prefixLength - 4));

        return `${prefix}${masked}${suffix}`;
    }

    /**
     * Generate masked key for service
     */
    private generateMaskedKeyForService(service: string): string {
        const patterns = {
            openai: "sk-•••••••••••••••••••••••••••••••••••••••••••••••••••",
            anthropic: "claude-•••••••••••••••••••••••••••••••••••••••••••••",
            google: "AIza•••••••••••••••••••••••••••••••••••••",
            microsoft: "ms-•••••••••••••••••••••••••••••••••••••••••••••••",
            aws: "AKIA•••••••••••••••••••••••••••••••••••••",
            stripe: "sk_live_•••••••••••••••••••••••••••••••••••••••••••",
        };

        return patterns[service as keyof typeof patterns] || "key-••••••••••••••••••••••••••••••••••••••";
    }

    /**
     * Get service prefix length for masking
     */
    private getServicePrefixLength(service: string): number {
        const prefixLengths = {
            openai: 3, // sk-
            anthropic: 7, // claude-
            google: 4, // AIza
            microsoft: 3, // ms-
            aws: 4, // AKIA
            stripe: 8, // sk_live_
        };

        return prefixLengths[service as keyof typeof prefixLengths] || 4;
    }

    /**
     * Validate key format for service
     */
    private validateKeyFormat(key: string, service: string): boolean {
        const patterns = {
            openai: new RegExp(`^sk-[a-zA-Z0-9]{${OPENAI_KEY_SUFFIX_LENGTH}}$`),
            anthropic: /^claude-[a-zA-Z0-9-]+$/,
            google: new RegExp(`^AIza[a-zA-Z0-9_-]{${GOOGLE_KEY_SUFFIX_LENGTH}}$`),
            microsoft: /^ms-[a-zA-Z0-9]{32,}$/,
            aws: new RegExp(`^AKIA[a-zA-Z0-9]{${AWS_KEY_SUFFIX_LENGTH}}$`),
            stripe: new RegExp(`^sk_(live|test)_[a-zA-Z0-9]{${STRIPE_KEY_MIN_SUFFIX_LENGTH},}$`),
        };

        const pattern = patterns[service as keyof typeof patterns];
        return pattern ? pattern.test(key) : key.length > SERVICE_KEY_MIN_LENGTH; // Generic validation
    }
}

/**
 * Pre-configured API key response scenarios
 */
export const apiKeyResponseScenarios = {
    // Success scenarios
    createSuccess: (input?: Partial<ApiKeyCreateInput>) => {
        const factory = new ApiKeyResponseFactory();
        const defaultInput: ApiKeyCreateInput = {
            id: generatePK().toString(),
            name: "Test API Key",
            limitHard: "100000",
            stopAtLimit: false,
            permissions: "read",
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createFromInput(defaultInput),
        );
    },

    createWithFullKeySuccess: (input?: Partial<ApiKeyCreateInput>) => {
        const factory = new ApiKeyResponseFactory();
        const defaultInput: ApiKeyCreateInput = {
            id: generatePK().toString(),
            name: "Test API Key",
            limitHard: "100000",
            stopAtLimit: false,
            permissions: "read",
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createApiKeyCreatedResponse(defaultInput),
        );
    },

    findSuccess: (apiKey?: ApiKey) => {
        const factory = new ApiKeyResponseFactory();
        return factory.createSuccessResponse(
            apiKey || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new ApiKeyResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    updateSuccess: (existing?: ApiKey, updates?: Partial<ApiKeyUpdateInput>) => {
        const factory = new ApiKeyResponseFactory();
        const apiKey = existing || factory.createMockData({ scenario: "complete" });
        const input: ApiKeyUpdateInput = {
            id: apiKey.id,
            ...updates,
        };
        return factory.createSuccessResponse(
            factory.updateFromInput(apiKey, input),
        );
    },

    listSuccess: (apiKeys?: ApiKey[]) => {
        const factory = new ApiKeyResponseFactory();
        return factory.createPaginatedResponse(
            apiKeys || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: apiKeys?.length || DEFAULT_COUNT },
        );
    },

    userKeysSuccess: (userId?: string) => {
        const factory = new ApiKeyResponseFactory();
        const keys = factory.createApiKeysForUser(userId || generatePK().toString());
        return factory.createPaginatedResponse(
            keys,
            { page: 1, totalCount: keys.length },
        );
    },

    usagePatternsSuccess: () => {
        const factory = new ApiKeyResponseFactory();
        const keys = factory.createApiKeysWithUsagePatterns();
        return factory.createPaginatedResponse(
            keys,
            { page: 1, totalCount: keys.length },
        );
    },

    // External API Key scenarios
    externalCreateSuccess: (input?: Partial<ApiKeyExternalCreateInput>) => {
        const factory = new ApiKeyExternalResponseFactory();
        const defaultInput: ApiKeyExternalCreateInput = {
            name: "OpenAI API Key",
            service: "openai",
            key: "sk-1234567890abcdef1234567890abcdef1234567890abcdef",
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createFromInput(defaultInput),
        );
    },

    externalListSuccess: (externalKeys?: ApiKeyExternal[]) => {
        const factory = new ApiKeyExternalResponseFactory();
        return factory.createPaginatedResponse(
            externalKeys || factory.createExternalKeysForAllServices(),
            { page: 1, totalCount: externalKeys?.length || EXTERNAL_SERVICES.length },
        );
    },

    allServicesSuccess: () => {
        const factory = new ApiKeyExternalResponseFactory();
        const keys = factory.createExternalKeysForAllServices();
        return factory.createPaginatedResponse(
            keys,
            { page: 1, totalCount: keys.length },
        );
    },

    // Error scenarios
    createValidationError: () => {
        const factory = new ApiKeyResponseFactory();
        return factory.createValidationErrorResponse({
            id: "ID is required",
            name: "Name is too long",
        });
    },

    notFoundError: (keyId?: string) => {
        const factory = new ApiKeyResponseFactory();
        return factory.createNotFoundErrorResponse(
            keyId || "non-existent-key",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new ApiKeyResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
            ["api_key:write"],
        );
    },

    keyLimitError: (currentCount?: number) => {
        const factory = new ApiKeyResponseFactory();
        return factory.createKeyLimitErrorResponse(currentCount);
    },

    invalidServiceError: (service = "unknown") => {
        const factory = new ApiKeyExternalResponseFactory();
        return factory.createInvalidServiceErrorResponse(service);
    },

    invalidKeyFormatError: (service = "openai") => {
        const factory = new ApiKeyExternalResponseFactory();
        return factory.createInvalidKeyFormatErrorResponse(service);
    },

    externalValidationError: () => {
        const factory = new ApiKeyExternalResponseFactory();
        return factory.createValidationErrorResponse({
            service: "Service is required",
            key: "API key is required",
            id: "ID is required",
        });
    },

    // MSW handlers
    handlers: {
        success: () => new ApiKeyResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new ApiKeyResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new ApiKeyResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },

    externalHandlers: {
        success: () => new ApiKeyExternalResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new ApiKeyExternalResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new ApiKeyExternalResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instances for direct use
export const apiKeyResponseFactory = new ApiKeyResponseFactory();
export const apiKeyExternalResponseFactory = new ApiKeyExternalResponseFactory();
