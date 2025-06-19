import { type ApiVersionConfigObject } from "../../../shape/configs/api.js";
import { LATEST_CONFIG_VERSION } from "../../../shape/configs/utils.js";
import { type ConfigTestFixtures, mergeWithBaseDefaults } from "./baseConfigFixtures.js";

/**
 * API configuration fixtures for testing API version settings
 */
export const apiConfigFixtures: ConfigTestFixtures<ApiVersionConfigObject> = {
    minimal: {
        __version: LATEST_CONFIG_VERSION,
    },
    
    complete: {
        __version: LATEST_CONFIG_VERSION,
        rateLimiting: {
            requestsPerMinute: 1000,
            burstLimit: 100,
            useGlobalRateLimit: true,
        },
        authentication: {
            type: "apiKey",
            location: "header",
            parameterName: "X-API-Key",
            settings: {
                prefix: "Bearer",
                required: true,
            }
        },
        caching: {
            enabled: true,
            ttl: 3600,
            invalidation: "ttl",
        },
        timeout: {
            request: 30000,
            connection: 5000,
        },
        retry: {
            maxAttempts: 3,
            backoffStrategy: "exponential",
            initialDelay: 1000,
        },
        documentationLink: "https://api.example.com/docs",
        schema: {
            language: "openapi",
            text: `openapi: 3.0.0
info:
  title: Example API
  version: 1.0.0
paths:
  /users:
    get:
      summary: Get all users
      responses:
        '200':
          description: Success`,
        },
        callLink: "https://api.example.com/v1",
        resources: [{
            link: "https://api.example.com",
            usedFor: "OfficialWebsite",
            translations: [{
                language: "en",
                name: "API Documentation",
                description: "Official API documentation and reference"
            }]
        }]
    },
    
    withDefaults: {
        __version: LATEST_CONFIG_VERSION,
        rateLimiting: {
            requestsPerMinute: 1000,
            burstLimit: 100,
            useGlobalRateLimit: true,
        },
        authentication: {
            type: "none",
        },
        caching: {
            enabled: true,
            ttl: 3600,
            invalidation: "ttl",
        },
        timeout: {
            request: 10000,
            connection: 10000,
        },
        retry: {
            maxAttempts: 3,
            backoffStrategy: "exponential",
            initialDelay: 1000,
        },
    },
    
    invalid: {
        missingVersion: {
            // Missing __version
            rateLimiting: {
                requestsPerMinute: 1000,
            },
            authentication: {
                type: "apiKey",
            },
        },
        invalidVersion: {
            __version: "0.1", // Invalid version
            authentication: {
                type: "oauth2",
            },
        },
        malformedStructure: {
            __version: LATEST_CONFIG_VERSION,
            authentication: "string instead of object", // Wrong type
        },
        invalidTypes: {
            __version: LATEST_CONFIG_VERSION,
            rateLimiting: {
                requestsPerMinute: "not a number", // Should be number
                burstLimit: -1, // Should be positive
            },
            timeout: {
                request: "5 seconds", // Should be number
            }
        }
    },
    
    variants: {
        publicApiNoAuth: {
            __version: LATEST_CONFIG_VERSION,
            authentication: {
                type: "none",
            },
            rateLimiting: {
                requestsPerMinute: 60,
                burstLimit: 10,
                useGlobalRateLimit: true,
            },
            caching: {
                enabled: true,
                ttl: 7200,
                invalidation: "ttl",
            },
        },
        
        secureApiKeyAuth: {
            __version: LATEST_CONFIG_VERSION,
            authentication: {
                type: "apiKey",
                location: "header",
                parameterName: "Authorization",
                settings: {
                    prefix: "Bearer",
                    required: true,
                    validatePattern: "^[A-Za-z0-9-_]+$",
                }
            },
            rateLimiting: {
                requestsPerMinute: 1000,
                burstLimit: 100,
                useGlobalRateLimit: false,
            },
            timeout: {
                request: 30000,
                connection: 5000,
            },
            documentationLink: "https://api.secure.com/docs",
        },
        
        oauth2ProtectedApi: {
            __version: LATEST_CONFIG_VERSION,
            authentication: {
                type: "oauth2",
                location: "header",
                parameterName: "Authorization",
                settings: {
                    authorizationUrl: "https://auth.example.com/authorize",
                    tokenUrl: "https://auth.example.com/token",
                    scopes: ["read", "write"],
                    flow: "authorizationCode",
                }
            },
            rateLimiting: {
                requestsPerMinute: 5000,
                burstLimit: 500,
                useGlobalRateLimit: false,
            },
            retry: {
                maxAttempts: 5,
                backoffStrategy: "exponential",
                initialDelay: 500,
            },
        },
        
        basicAuthApi: {
            __version: LATEST_CONFIG_VERSION,
            authentication: {
                type: "basic",
                location: "header",
                parameterName: "Authorization",
                settings: {
                    realm: "API Access",
                }
            },
            timeout: {
                request: 60000,
                connection: 10000,
            },
        },
        
        highPerformanceApi: {
            __version: LATEST_CONFIG_VERSION,
            rateLimiting: {
                requestsPerMinute: 10000,
                burstLimit: 1000,
                useGlobalRateLimit: false,
            },
            caching: {
                enabled: false, // No caching for real-time data
            },
            timeout: {
                request: 5000, // Fast timeout
                connection: 1000,
            },
            retry: {
                maxAttempts: 1, // No retries for speed
            },
        },
        
        webhookApi: {
            __version: LATEST_CONFIG_VERSION,
            authentication: {
                type: "webhook",
                location: "header",
                parameterName: "X-Webhook-Signature",
                settings: {
                    algorithm: "sha256",
                    secret: "webhook_secret",
                }
            },
            timeout: {
                request: 30000,
                connection: 5000,
            },
            retry: {
                maxAttempts: 3,
                backoffStrategy: "linear",
                initialDelay: 5000,
            },
            callLink: "https://webhook.example.com/notify",
        },
        
        graphqlApi: {
            __version: LATEST_CONFIG_VERSION,
            authentication: {
                type: "apiKey",
                location: "header",
                parameterName: "X-API-Key",
            },
            schema: {
                language: "graphql",
                text: `type Query {
  user(id: ID!): User
  users: [User!]!
}

type User {
  id: ID!
  name: String!
  email: String!
}`,
            },
            callLink: "https://api.example.com/graphql",
            documentationLink: "https://api.example.com/graphql/docs",
        },
        
        restApiWithSwagger: {
            __version: LATEST_CONFIG_VERSION,
            schema: {
                language: "openapi",
                text: `openapi: 3.0.0
info:
  title: REST API
  version: 2.0.0
  description: RESTful API with Swagger documentation
servers:
  - url: https://api.example.com/v2
paths:
  /resources:
    get:
      summary: List resources
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
            default: 10
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                type: array`,
            },
            callLink: "https://api.example.com/v2",
            documentationLink: "https://api.example.com/swagger",
            resources: [
                {
                    link: "https://api.example.com/swagger-ui",
                    usedFor: "Interactive",
                    translations: [{
                        language: "en",
                        name: "Swagger UI",
                        description: "Interactive API documentation"
                    }]
                },
                {
                    link: "https://api.example.com/postman",
                    usedFor: "Learning",
                    translations: [{
                        language: "en",
                        name: "Postman Collection",
                        description: "Ready-to-use API collection"
                    }]
                }
            ]
        }
    }
};

/**
 * Create an API config with specific authentication
 */
export function createApiConfigWithAuth(
    authType: string,
    authSettings: Partial<ApiVersionConfigObject["authentication"]> = {}
): ApiVersionConfigObject {
    return mergeWithBaseDefaults<ApiVersionConfigObject>({
        authentication: {
            type: authType,
            location: "header",
            parameterName: "Authorization",
            ...authSettings
        }
    });
}

/**
 * Create an API config with specific rate limiting
 */
export function createApiConfigWithRateLimit(
    requestsPerMinute: number,
    burstLimit?: number
): ApiVersionConfigObject {
    return mergeWithBaseDefaults<ApiVersionConfigObject>({
        rateLimiting: {
            requestsPerMinute,
            burstLimit: burstLimit || Math.floor(requestsPerMinute / 10),
            useGlobalRateLimit: false,
        }
    });
}

/**
 * Create an API config with schema
 */
export function createApiConfigWithSchema(
    language: string,
    text: string,
    callLink?: string
): ApiVersionConfigObject {
    return mergeWithBaseDefaults<ApiVersionConfigObject>({
        schema: {
            language,
            text,
        },
        callLink,
        documentationLink: callLink ? `${callLink}/docs` : undefined,
    });
}