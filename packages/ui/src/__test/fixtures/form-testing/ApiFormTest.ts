import {
    endpointsResource,
    resourceVersionTranslationValidation,
    resourceVersionValidation,
    type ResourceVersionCreateInput,
    type ResourceVersionShape,
    type ResourceVersionUpdateInput,
    type Session,
} from "@vrooli/shared";
import { apiInitialValues, transformResourceVersionValues } from "../../../../views/objects/api/ApiUpsert.js";
import { createUIFormTestFactory, type UIFormTestConfig } from "../UIFormTestFactory.js";

/**
 * Configuration for API form testing with data-driven test scenarios
 */
const apiFormTestConfig: UIFormTestConfig<ResourceVersionShape, ResourceVersionShape, ResourceVersionCreateInput, ResourceVersionUpdateInput, ResourceVersionShape> = {
    // Form metadata
    objectType: "ResourceVersion",
    formFixtures: {
        minimal: {
            __typename: "ResourceVersion" as const,
            id: "api_minimal",
            codeLanguage: "Yaml",
            config: {
                callLink: "https://api.weather.com/v1",
                documentationLink: null,
                schemaText: `openapi: 3.1.0
info:
  title: Weather API
  version: 1.0.0
paths:
  /weather:
    get:
      summary: Get weather data
      responses:
        '200':
          description: Weather data`,
            },
            isAutomatable: false,
            isComplete: false,
            isPrivate: false,
            resourceSubType: "Api",
            versionLabel: "1.0.0",
            versionNotes: null,
            translations: [{
                __typename: "ResourceVersionTranslation" as const,
                id: "trans_minimal",
                language: "en",
                name: "Weather API",
                description: "Simple weather data API",
                instructions: "Provides basic weather information",
                details: null,
            }],
        },
        complete: {
            __typename: "ResourceVersion" as const,
            id: "api_complete",
            codeLanguage: "Json",
            config: {
                callLink: "https://api.advancedweather.com/v2",
                documentationLink: "https://docs.advancedweather.com/api/v2",
                schemaText: `{
  "openapi": "3.1.0",
  "info": {
    "title": "Advanced Weather Data API",
    "description": "Comprehensive weather forecasting and historical data API",
    "version": "2.1.0",
    "contact": {
      "name": "API Support",
      "email": "support@advancedweather.com"
    }
  },
  "servers": [
    {
      "url": "https://api.advancedweather.com/v2",
      "description": "Production server"
    }
  ],
  "paths": {
    "/weather/current": {
      "get": {
        "summary": "Get current weather conditions",
        "parameters": [
          {
            "name": "location",
            "in": "query",
            "required": true,
            "schema": { "type": "string" }
          }
        ],
        "responses": {
          "200": {
            "description": "Current weather data",
            "content": {
              "application/json": {
                "schema": { "$ref": "#/components/schemas/CurrentWeather" }
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "CurrentWeather": {
        "type": "object",
        "properties": {
          "temperature": { "type": "number" },
          "humidity": { "type": "number" },
          "windSpeed": { "type": "number" }
        }
      }
    }
  }
}`,
            },
            isAutomatable: true,
            isComplete: true,
            isPrivate: false,
            resourceSubType: "Api",
            versionLabel: "2.1.0",
            versionNotes: "Enhanced with comprehensive schema and documentation",
            translations: [{
                __typename: "ResourceVersionTranslation" as const,
                id: "trans_complete",
                language: "en",
                name: "Advanced Weather Data API",
                description: "Comprehensive weather forecasting and historical data API",
                instructions: "Provides detailed weather information including current conditions, hourly forecasts, daily summaries, historical data, and severe weather alerts. Supports multiple data formats and international locations.",
                details: "Production-ready API with complete OpenAPI 3.1 specification",
            }],
        },
        invalid: {
            __typename: "ResourceVersion" as const,
            id: "api_invalid",
            codeLanguage: "Json",
            config: {
                callLink: "invalid-url", // Invalid URL format
                documentationLink: "also-invalid-url",
                schemaText: "invalid json {", // Invalid JSON
            },
            isAutomatable: false,
            isComplete: false,
            isPrivate: false,
            resourceSubType: "Api",
            versionLabel: "not-a-valid-version", // Invalid version format
            versionNotes: null,
            translations: [{
                __typename: "ResourceVersionTranslation" as const,
                id: "trans_invalid",
                language: "en",
                name: "", // Invalid: required field is empty
                description: "",
                instructions: "",
                details: null,
            }],
        },
        edgeCase: {
            __typename: "ResourceVersion" as const,
            id: "api_edge",
            codeLanguage: "Graphql",
            config: {
                callLink: "https://api.userservice.com/graphql",
                documentationLink: "https://docs.userservice.com/graphql",
                schemaText: `type Query {
  user(id: ID!): User
  users(limit: Int = 10, offset: Int = 0): [User!]!
}

type Mutation {
  createUser(input: CreateUserInput!): User!
  updateUser(id: ID!, input: UpdateUserInput!): User!
  deleteUser(id: ID!): Boolean!
}

type User {
  id: ID!
  email: String!
  name: String!
  createdAt: DateTime!
  updatedAt: DateTime!
  profile: UserProfile
}

type UserProfile {
  bio: String
  avatarUrl: String
  website: String
}

input CreateUserInput {
  email: String!
  name: String!
  profile: UserProfileInput
}

input UpdateUserInput {
  email: String
  name: String
  profile: UserProfileInput
}

input UserProfileInput {
  bio: String
  avatarUrl: String
  website: String
}

scalar DateTime`,
            },
            isAutomatable: true,
            isComplete: true,
            isPrivate: false,
            resourceSubType: "Api",
            versionLabel: "1.5.0",
            versionNotes: "GraphQL API with comprehensive schema",
            translations: [{
                __typename: "ResourceVersionTranslation" as const,
                id: "trans_edge",
                language: "en",
                name: "GraphQL User Management API",
                description: "User management GraphQL API with special characters: @#$%^&*()[]{}|\\:;\"'<>,.?/~`\n\nMultiple\nlines\n\nWith emoji: 🚀 📊 🔗",
                instructions: "GraphQL API for managing user profiles, authentication, and permissions. Supports complex queries and mutations.",
                details: "Complete GraphQL schema with custom scalars and nested types",
            }],
        },
    },

    // Validation schemas from shared package
    validation: resourceVersionValidation,
    translationValidation: resourceVersionTranslationValidation,

    // API endpoints from shared package
    endpoints: {
        create: endpointsResource.createOne,
        update: endpointsResource.updateOne,
    },

    // Transform functions - form already uses ResourceVersionShape, so no transformation needed
    formToShape: (formData: ResourceVersionShape) => formData,

    transformFunction: (shape: ResourceVersionShape, existing: ResourceVersionShape, isCreate: boolean) => {
        const result = transformResourceVersionValues(shape, existing, isCreate);
        if (!result) {
            throw new Error("Transform function returned undefined");
        }
        return result;
    },

    initialValuesFunction: (session?: Session, existing?: Partial<ResourceVersionShape>): ResourceVersionShape => {
        return apiInitialValues(session, existing || {});
    },

    // DATA-DRIVEN TEST SCENARIOS - replaces all custom wrapper methods
    testScenarios: {
        schemaLanguageSupport: {
            description: "Test different API schema language support",
            testCases: [
                {
                    name: "YAML/OpenAPI schema",
                    data: { codeLanguage: "Yaml" },
                    shouldPass: true,
                },
                {
                    name: "JSON/OpenAPI schema",
                    data: { codeLanguage: "Json" },
                    shouldPass: true,
                },
                {
                    name: "GraphQL schema",
                    data: { codeLanguage: "Graphql" },
                    shouldPass: true,
                },
                {
                    name: "XML schema",
                    data: { codeLanguage: "Xml" },
                    shouldPass: true,
                },
            ],
        },

        urlValidation: {
            description: "Test API endpoint URL validation",
            testCases: [
                {
                    name: "Valid HTTPS URL",
                    data: {
                        config: { callLink: "https://api.example.com/v1" },
                    },
                    shouldPass: true,
                },
                {
                    name: "Valid HTTP URL",
                    data: {
                        config: { callLink: "http://localhost:3000/api" },
                    },
                    shouldPass: true,
                },
                {
                    name: "URL with path and query parameters",
                    data: {
                        config: { callLink: "https://api.example.com/v2/data?format=json" },
                    },
                    shouldPass: true,
                },
                {
                    name: "Invalid URL format",
                    data: {
                        config: { callLink: "not-a-url" },
                    },
                    shouldPass: false,
                },
                {
                    name: "Empty URL",
                    data: {
                        config: { callLink: "" },
                    },
                    shouldPass: false,
                },
                {
                    name: "FTP URL (should be invalid for API)",
                    data: {
                        config: { callLink: "ftp://files.example.com/api" },
                    },
                    shouldPass: false,
                },
            ],
        },

        schemaValidation: {
            description: "Test API schema content validation",
            testCases: [
                {
                    name: "Valid OpenAPI YAML",
                    data: {
                        codeLanguage: "Yaml",
                        config: {
                            schemaText: `openapi: 3.1.0
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: Success`,
                        },
                    },
                    shouldPass: true,
                },
                {
                    name: "Valid GraphQL schema",
                    data: {
                        codeLanguage: "Graphql",
                        config: {
                            schemaText: `type Query {
  hello: String
}

type User {
  id: ID!
  name: String!
}`,
                        },
                    },
                    shouldPass: true,
                },
                {
                    name: "Invalid JSON schema",
                    data: {
                        codeLanguage: "Json",
                        config: {
                            schemaText: "invalid json {",
                        },
                    },
                    shouldPass: false,
                },
                {
                    name: "Empty schema",
                    data: {
                        config: {
                            schemaText: "",
                        },
                    },
                    shouldPass: false,
                },
            ],
        },

        documentationLinks: {
            description: "Test API documentation link validation",
            testCases: [
                {
                    name: "Valid documentation URL",
                    data: {
                        config: {
                            documentationLink: "https://docs.example.com/api",
                        },
                    },
                    shouldPass: true,
                },
                {
                    name: "No documentation link",
                    data: {
                        config: {
                            documentationLink: null,
                        },
                    },
                    shouldPass: true,
                },
                {
                    name: "Invalid documentation URL",
                    data: {
                        config: {
                            documentationLink: "not-a-valid-url",
                        },
                    },
                    shouldPass: false,
                },
            ],
        },

        versionValidation: {
            description: "Test API version label validation",
            testCases: [
                {
                    name: "Valid semantic version",
                    field: "versionLabel",
                    value: "1.0.0",
                    shouldPass: true,
                },
                {
                    name: "Version with pre-release",
                    field: "versionLabel",
                    value: "1.0.0-alpha.1",
                    shouldPass: true,
                },
                {
                    name: "Version with build metadata",
                    field: "versionLabel",
                    value: "1.0.0+20231201",
                    shouldPass: true,
                },
                {
                    name: "Invalid version format",
                    field: "versionLabel",
                    value: "not-a-version",
                    shouldPass: false,
                },
            ],
        },

        apiFeatures: {
            description: "Test different API feature configurations",
            testCases: [
                {
                    name: "Public API",
                    data: {
                        isPrivate: false,
                        isComplete: true,
                        isAutomatable: true,
                        resourceSubType: "Api",
                    },
                    shouldPass: true,
                },
                {
                    name: "Private API",
                    data: {
                        isPrivate: true,
                        isComplete: false,
                        isAutomatable: false,
                        resourceSubType: "Api",
                    },
                    shouldPass: true,
                },
                {
                    name: "Beta API",
                    data: {
                        isPrivate: false,
                        isComplete: false,
                        isAutomatable: false,
                        resourceSubType: "Api",
                    },
                    shouldPass: true,
                },
            ],
        },

        complexSchemas: {
            description: "Test handling of complex API schemas",
            testCases: [
                {
                    name: "Large OpenAPI schema",
                    data: {
                        codeLanguage: "Json",
                        config: {
                            schemaText: JSON.stringify({
                                openapi: "3.1.0",
                                info: { title: "Complex API", version: "1.0.0" },
                                paths: {
                                    "/users": {
                                        get: { responses: { "200": { description: "Users list" } } },
                                        post: { responses: { "201": { description: "User created" } } },
                                    },
                                    "/users/{id}": {
                                        get: { responses: { "200": { description: "User details" } } },
                                        put: { responses: { "200": { description: "User updated" } } },
                                        delete: { responses: { "204": { description: "User deleted" } } },
                                    },
                                },
                            }),
                        },
                    },
                    shouldPass: true,
                },
                {
                    name: "GraphQL with complex types",
                    data: {
                        codeLanguage: "Graphql",
                        config: {
                            schemaText: `interface Node {
  id: ID!
}

type User implements Node {
  id: ID!
  email: String!
  posts: [Post!]!
}

type Post implements Node {
  id: ID!
  title: String!
  content: String!
  author: User!
  tags: [Tag!]!
}

type Tag {
  name: String!
  posts: [Post!]!
}

type Query {
  node(id: ID!): Node
  users: [User!]!
  posts: [Post!]!
}`,
                        },
                    },
                    shouldPass: true,
                },
            ],
        },

        apiWorkflows: {
            description: "Test different API development workflows",
            testCases: [
                {
                    name: "REST API with OpenAPI",
                    data: {
                        codeLanguage: "Yaml",
                        config: {
                            callLink: "https://api.example.com/v1",
                            documentationLink: "https://docs.example.com",
                            schemaText: `openapi: 3.1.0
info:
  title: Example REST API
  version: 1.0.0
paths:
  /users:
    get:
      summary: List users
      responses:
        '200':
          description: Users list`,
                        },
                        isComplete: true,
                        isAutomatable: true,
                        resourceSubType: "Api",
                    },
                    shouldPass: true,
                },
                {
                    name: "GraphQL API",
                    data: {
                        codeLanguage: "Graphql",
                        config: {
                            callLink: "https://api.example.com/graphql",
                            documentationLink: "https://docs.example.com/graphql",
                            schemaText: `type Query {
  users: [User!]!
}

type User {
  id: ID!
  name: String!
}`,
                        },
                        isComplete: true,
                        isAutomatable: true,
                        resourceSubType: "Api",
                    },
                    shouldPass: true,
                },
                {
                    name: "Incomplete API draft",
                    data: {
                        codeLanguage: "Yaml",
                        config: {
                            callLink: "https://api.example.com/v1",
                            schemaText: `openapi: 3.1.0
info:
  title: Draft API
  version: 0.1.0`,
                        },
                        isComplete: false,
                        isAutomatable: false,
                        resourceSubType: "Api",
                    },
                    shouldPass: true,
                },
            ],
        },
    },
};

/**
 * SIMPLIFIED: Direct factory export - no wrapper function needed!
 */
export const apiFormTestFactory = createUIFormTestFactory(apiFormTestConfig);

/**
 * Type exports for use in other test files
 */
export { apiFormTestConfig };
