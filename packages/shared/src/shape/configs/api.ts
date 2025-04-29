import { ApiVersion } from "../../api/types.js";
import { type PassableLogger } from "../../consts/commonTypes.js";
import { BaseConfig, BaseConfigObject } from "./baseConfig.js";
import { LATEST_CONFIG_VERSION, parseObject, type StringifyMode } from "./utils.js";

const DEFAULT_STRINGIFY_MODE: StringifyMode = "json";

/**
 * Represents all data that can be stored in an API's stringified config.
 * 
 * This includes configuration that doesn't need to be queried while searching
 * (e.g., API rate limiting, authentication settings). Properties like callLink,
 * schemaText, and documentationLink are stored directly in the ApiVersion model.
 */
export interface ApiVersionConfigObject extends BaseConfigObject {
    /** Rate limiting configuration for the API */
    rateLimiting?: {
        /** Maximum requests per minute */
        requestsPerMinute?: number;
        /** Burst limit */
        burstLimit?: number;
        /** Whether to use global rate limiting */
        useGlobalRateLimit?: boolean;
    };
    /** Authentication settings */
    authentication?: {
        /** Authentication type (e.g., "none", "apiKey", "oauth2", "basic") */
        type: string;
        /** Where to place the authentication (header, query, body) */
        location?: string;
        /** Name of the parameter (e.g., "Authorization", "api_key") */
        parameterName?: string;
        /** Additional auth settings as key-value pairs */
        settings?: Record<string, any>;
    };
    /** Caching instructions */
    caching?: {
        /** Whether responses should be cached */
        enabled: boolean;
        /** Time-to-live in seconds */
        ttl?: number;
        /** Cache invalidation strategy */
        invalidation?: string;
    };
    /** Timeout settings */
    timeout?: {
        /** Request timeout in milliseconds */
        request?: number;
        /** Connection timeout in milliseconds */
        connection?: number;
    };
    /** Retry settings */
    retry?: {
        /** Maximum number of retry attempts */
        maxAttempts?: number;
        /** Backoff strategy (e.g., "fixed", "exponential") */
        backoffStrategy?: string;
        /** Initial delay in milliseconds */
        initialDelay?: number;
    };
    /** Where to find the API's documentation */
    documentationLink?: string;
    /** The API's schema */
    schema?: {
        /** The language the schema is written in */
        language?: string;
        /** The schema text */
        text?: string;
    };
    /** The API's call link */
    callLink?: string;
}

/**
 * Top-level API config that encapsulates all API-related configuration data.
 */
export class ApiVersionConfig extends BaseConfig<ApiVersionConfigObject> {
    rateLimiting?: ApiVersionConfigObject["rateLimiting"];
    authentication?: ApiVersionConfigObject["authentication"];
    caching?: ApiVersionConfigObject["caching"];
    timeout?: ApiVersionConfigObject["timeout"];
    retry?: ApiVersionConfigObject["retry"];
    documentationLink?: ApiVersionConfigObject["documentationLink"];
    schema?: ApiVersionConfigObject["schema"];
    callLink?: ApiVersionConfigObject["callLink"];

    constructor(data: ApiVersionConfigObject) {
        super(data);
        this.rateLimiting = data.rateLimiting;
        this.authentication = data.authentication;
        this.caching = data.caching;
        this.timeout = data.timeout;
        this.retry = data.retry;
        this.documentationLink = data.documentationLink;
        this.schema = data.schema;
        this.callLink = data.callLink;
    }

    /**
     * Creates an ApiVersionConfig from an API version object
     */
    static createFromApiVersion(
        apiVersion: {
            data?: string | null;
            callLink: ApiVersion["callLink"];
            schemaText?: ApiVersion["schemaText"];
            schemaLanguage?: ApiVersion["schemaLanguage"];
            documentationLink?: ApiVersion["documentationLink"];
        },
        logger: PassableLogger,
        options: { mode?: StringifyMode } = {},
    ): ApiVersionConfig {
        const mode = options.mode || DEFAULT_STRINGIFY_MODE;
        const obj = apiVersion.data ? parseObject<ApiVersionConfigObject>(apiVersion.data, mode, logger) : null;
        if (!obj) {
            return ApiVersionConfig.default({
                callLink: apiVersion.callLink,
                schemaText: apiVersion.schemaText,
                schemaLanguage: apiVersion.schemaLanguage,
                documentationLink: apiVersion.documentationLink,
            });
        }
        return new ApiVersionConfig({
            data: obj,
            callLink: apiVersion.callLink,
            schemaText: apiVersion.schemaText,
            schemaLanguage: apiVersion.schemaLanguage,
            documentationLink: apiVersion.documentationLink,
        });
    }

    /**
     * Creates a default ApiVersionConfig
     */
    static default({
        callLink,
        schemaText,
        schemaLanguage,
        documentationLink,
    }: {
        callLink: ApiVersion["callLink"],
        schemaText?: ApiVersion["schemaText"],
        schemaLanguage?: ApiVersion["schemaLanguage"],
        documentationLink?: ApiVersion["documentationLink"]
    }): ApiVersionConfig {
        const data: ApiVersionConfigObject = {
            __version: LATEST_CONFIG_VERSION,
            resources: [],
            metadata: {},
        };
        return new ApiVersionConfig({
            data,
            callLink,
            schemaText,
            schemaLanguage,
            documentationLink,
        });
    }

    /**
     * Exports the config to a plain object
     */
    override export(): ApiVersionConfigObject {
        return {
            ...super.export(),
            rateLimiting: this.rateLimiting,
            authentication: this.authentication,
            caching: this.caching,
            timeout: this.timeout,
            retry: this.retry,
        };
    }

    /**
     * Sets rate limiting settings
     */
    setRateLimiting(config: ApiVersionConfigObject["rateLimiting"]): void {
        this.rateLimiting = config;
    }

    /**
     * Sets authentication settings
     */
    setAuthentication(config: ApiVersionConfigObject["authentication"]): void {
        this.authentication = config;
    }

    /**
     * Sets caching settings
     */
    setCaching(config: ApiVersionConfigObject["caching"]): void {
        this.caching = config;
    }

    /**
     * Sets timeout settings
     */
    setTimeout(config: ApiVersionConfigObject["timeout"]): void {
        this.timeout = config;
    }

    /**
     * Sets retry settings
     */
    setRetry(config: ApiVersionConfigObject["retry"]): void {
        this.retry = config;
    }
} 
