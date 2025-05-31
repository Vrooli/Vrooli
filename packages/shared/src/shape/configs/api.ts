import { type ResourceVersion } from "../../api/types.js";
import { type PassableLogger } from "../../consts/commonTypes.js";
import { BaseConfig, type BaseConfigObject } from "./base.js";

const LATEST_CONFIG_VERSION = "1.0";

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

    constructor({ config }: { config: ApiVersionConfigObject }) {
        super(config);
        this.rateLimiting = config.rateLimiting;
        this.authentication = config.authentication;
        this.caching = config.caching;
        this.timeout = config.timeout;
        this.retry = config.retry;
        this.documentationLink = config.documentationLink;
        this.schema = config.schema;
        this.callLink = config.callLink;
    }

    static parse(
        version: Pick<ResourceVersion, "config">,
        logger: PassableLogger,
        opts?: { useFallbacks?: boolean },
    ): ApiVersionConfig {
        return super.parseBase<ApiVersionConfigObject, ApiVersionConfig>(
            version.config,
            logger,
            (cfg) => {
                // ensure defaults for input/output/testcases
                if (opts?.useFallbacks ?? true) {
                    cfg.rateLimiting ??= ApiVersionConfig.defaultRateLimiting();
                    cfg.authentication ??= ApiVersionConfig.defaultAuthentication();
                    cfg.caching ??= ApiVersionConfig.defaultCaching();
                    cfg.timeout ??= ApiVersionConfig.defaultTimeout();
                    cfg.retry ??= ApiVersionConfig.defaultRetry();
                }
                return new ApiVersionConfig({ config: cfg });
            },
        );
    }

    /**
     * Creates a default ApiVersionConfig
     */
    static default(): ApiVersionConfig {
        const config: ApiVersionConfigObject = {
            __version: LATEST_CONFIG_VERSION,
            resources: [],
            rateLimiting: ApiVersionConfig.defaultRateLimiting(),
            authentication: ApiVersionConfig.defaultAuthentication(),
            caching: ApiVersionConfig.defaultCaching(),
            timeout: ApiVersionConfig.defaultTimeout(),
            retry: ApiVersionConfig.defaultRetry(),
        };
        return new ApiVersionConfig({ config });
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

    static defaultRateLimiting(): ApiVersionConfigObject["rateLimiting"] {
        return {
            requestsPerMinute: 1000,
            burstLimit: 100,
            useGlobalRateLimit: true,
        };
    }

    static defaultAuthentication(): ApiVersionConfigObject["authentication"] {
        return {
            type: "none",
        };
    }

    static defaultCaching(): ApiVersionConfigObject["caching"] {
        return {
            enabled: true,
            ttl: 3600,
            invalidation: "ttl",
        };
    }

    static defaultTimeout(): ApiVersionConfigObject["timeout"] {
        return {
            request: 10000,
            connection: 10000,
        };
    }

    static defaultRetry(): ApiVersionConfigObject["retry"] {
        return {
            maxAttempts: 3,
            backoffStrategy: "exponential",
            initialDelay: 1000,
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
