import { ResourceSubType } from "../../../api/types.js";
import { type StandardVersionConfigObject } from "../../../shape/configs/standard.js";
import { LATEST_CONFIG_VERSION } from "../../../shape/configs/utils.js";
import { type ConfigTestFixtures, mergeWithBaseDefaults } from "./baseConfigFixtures.js";

/**
 * Standard configuration fixtures for testing standard version configurations
 */
export const standardConfigFixtures: ConfigTestFixtures<StandardVersionConfigObject> = {
    minimal: {
        __version: LATEST_CONFIG_VERSION,
    },

    complete: {
        __version: LATEST_CONFIG_VERSION,
        validation: {
            strictMode: true,
            rules: {
                required: ["field1", "field2"],
                minLength: { field1: 3, field2: 5 },
                pattern: { email: "^[\\w-\\.]+@([\\w-]+\\.)+[\\w-]{2,4}$" },
            },
            errorMessages: {
                required: "This field is required",
                minLength: "Field must be at least {min} characters",
                pattern: "Invalid format",
            },
        },
        format: {
            defaultFormat: "json",
            options: {
                indentation: 2,
                sortKeys: true,
                includeComments: false,
            },
        },
        compatibility: {
            minimumRequirements: {
                runtime: "node >= 16.0.0",
                memory: "512MB",
                diskSpace: "100MB",
            },
            knownIssues: [
                "Does not support Unicode characters in field names",
                "Performance degradation with arrays over 10,000 items",
            ],
            compatibleWith: ["ISO-8601", "RFC-3339", "JSON-Schema-Draft-07"],
        },
        compliance: {
            compliesWith: ["GDPR", "HIPAA", "SOC2"],
            certifications: [
                {
                    name: "ISO 27001",
                    issuer: "International Organization for Standardization",
                    date: "2024-01-01",
                    expiration: "2027-01-01",
                },
                {
                    name: "SOC 2 Type II",
                    issuer: "AICPA",
                    date: "2023-06-15",
                    expiration: "2024-06-15",
                },
            ],
        },
        schema: JSON.stringify({
            "$schema": "http://json-schema.org/draft-07/schema#",
            "type": "object",
            "properties": {
                "field1": { "type": "string", "minLength": 3 },
                "field2": { "type": "string", "minLength": 5 },
                "email": { "type": "string", "format": "email" },
            },
            "required": ["field1", "field2"],
        }),
        schemaLanguage: "json-schema",
        props: {
            customProp1: "value1",
            customProp2: 42,
            customProp3: true,
            nestedProps: {
                prop1: "nested value",
                prop2: [1, 2, 3],
            },
        },
    },

    withDefaults: {
        __version: LATEST_CONFIG_VERSION,
    },

    invalid: {
        missingVersion: {
            // Missing __version
            validation: {
                strictMode: true,
            },
        },
        invalidVersion: {
            __version: "0.1", // Invalid version
            schema: "{}",
        },
        malformedStructure: {
            __version: LATEST_CONFIG_VERSION,
            validation: "string instead of object", // Wrong type
        },
        invalidTypes: {
            __version: LATEST_CONFIG_VERSION,
            validation: {
                // @ts-expect-error - Intentionally invalid type for testing
                strictMode: "yes", // Should be boolean
                // @ts-expect-error - Intentionally invalid type for testing
                rules: "not an object", // Should be object
            },
            compliance: {
                // @ts-expect-error - Intentionally invalid type for testing
                certifications: "not an array", // Should be array
            },
        },
    },

    variants: {
        minimalValidation: {
            __version: LATEST_CONFIG_VERSION,
            validation: {
                strictMode: false,
            },
        },

        dataStructureStandard: {
            __version: LATEST_CONFIG_VERSION,
            validation: {
                strictMode: true,
                rules: {
                    type: "object",
                    required: ["id", "name", "type"],
                    properties: {
                        id: { type: "string", format: "uuid" },
                        name: { type: "string", minLength: 1, maxLength: 255 },
                        type: { type: "string", enum: ["string", "number", "boolean", "object", "array"] },
                    },
                },
            },
            schema: JSON.stringify({
                "$schema": "http://json-schema.org/draft-07/schema#",
                "title": "Data Structure Definition",
                "type": "object",
                "required": ["id", "name", "type"],
                "properties": {
                    "id": { "type": "string", "format": "uuid" },
                    "name": { "type": "string", "minLength": 1, "maxLength": 255 },
                    "type": { "type": "string", "enum": ["string", "number", "boolean", "object", "array"] },
                    "description": { "type": "string" },
                    "properties": { "type": "object" },
                },
            }),
            schemaLanguage: "json-schema",
            compatibility: {
                compatibleWith: ["JSON-Schema-Draft-07", "OpenAPI-3.0"],
            },
        },

        promiseStandard: {
            __version: LATEST_CONFIG_VERSION,
            validation: {
                strictMode: true,
                rules: {
                    maxLength: { description: 500 },
                    required: ["title", "description", "criteria"],
                    dateValidation: {
                        startDate: "future",
                        endDate: "afterStartDate",
                    },
                },
                errorMessages: {
                    required: "Promise {field} is required",
                    maxLength: "Description cannot exceed 500 characters",
                    dateValidation: "End date must be after start date",
                },
            },
            format: {
                defaultFormat: "markdown",
                options: {
                    includeMetadata: true,
                    formatDates: "ISO8601",
                },
            },
            props: {
                templateType: "promise",
                allowPartialCompletion: true,
                trackingEnabled: true,
            },
        },

        technicalStandard: {
            __version: LATEST_CONFIG_VERSION,
            validation: {
                strictMode: true,
                rules: {
                    syntaxValidation: true,
                    linting: {
                        enabled: true,
                        rules: ["no-unused-vars", "no-console", "strict-types"],
                    },
                },
            },
            compatibility: {
                minimumRequirements: {
                    runtime: "node >= 18.0.0",
                    typescript: ">= 5.0.0",
                    memory: "1GB",
                },
                knownIssues: [
                    "TypeScript 4.x requires polyfills for certain features",
                ],
                compatibleWith: ["ES2022", "CommonJS", "ESM"],
            },
            compliance: {
                compliesWith: ["OWASP", "CWE-Top-25"],
                certifications: [{
                    name: "Security Audit",
                    issuer: "Internal Security Team",
                    date: "2024-03-01",
                }],
            },
            schema: JSON.stringify({
                "$schema": "http://json-schema.org/draft-07/schema#",
                "type": "object",
                "properties": {
                    "version": { "type": "string", "pattern": "^\\d+\\.\\d+\\.\\d+$" },
                    "dependencies": { "type": "object" },
                    "scripts": { "type": "object" },
                },
            }),
            schemaLanguage: "json-schema",
        },

        apiContractStandard: {
            __version: LATEST_CONFIG_VERSION,
            validation: {
                strictMode: true,
                rules: {
                    validateEndpoints: true,
                    validateResponses: true,
                    allowAdditionalProperties: false,
                },
            },
            format: {
                defaultFormat: "openapi",
                options: {
                    version: "3.1.0",
                    includeExamples: true,
                },
            },
            compatibility: {
                compatibleWith: ["OpenAPI-3.1", "JSON-Schema-2020-12", "REST", "GraphQL"],
            },
            compliance: {
                compliesWith: ["REST-Best-Practices", "API-Security-Top-10"],
            },
            schema: JSON.stringify({
                "openapi": "3.1.0",
                "info": {
                    "title": "Standard API",
                    "version": "1.0.0",
                },
                "paths": {},
                "components": {
                    "schemas": {},
                },
            }),
            schemaLanguage: "openapi",
            props: {
                authentication: ["bearer", "api-key"],
                rateLimiting: {
                    enabled: true,
                    requests: 100,
                    window: "1h",
                },
            },
        },

        formStandard: {
            __version: LATEST_CONFIG_VERSION,
            validation: {
                strictMode: false,
                rules: {
                    clientSideValidation: true,
                    serverSideValidation: true,
                    realTimeValidation: false,
                },
                errorMessages: {
                    email: "Please enter a valid email address",
                    required: "This field cannot be empty",
                    minLength: "Must be at least {min} characters long",
                },
            },
            format: {
                defaultFormat: "json",
                options: {
                    preserveOrder: true,
                    includeHelpText: true,
                },
            },
            props: {
                formType: "dynamic",
                fields: [
                    { name: "email", type: "email", required: true },
                    { name: "name", type: "text", required: true, minLength: 2 },
                    { name: "message", type: "textarea", required: false, maxLength: 1000 },
                ],
                submitAction: "POST /api/contact",
            },
        },
    },
};

/**
 * Create a standard config with specific validation rules
 */
export function createStandardConfigWithValidation(
    rules: Record<string, unknown>,
    strictMode = true,
): StandardVersionConfigObject {
    return mergeWithBaseDefaults<StandardVersionConfigObject>({
        validation: {
            strictMode,
            rules,
            errorMessages: {
                default: "Validation failed",
            },
        },
    });
}

/**
 * Create a standard config for a specific schema language
 */
export function createStandardConfigWithSchema(
    schema: string | object,
    schemaLanguage = "json-schema",
): StandardVersionConfigObject {
    return {
        __version: LATEST_CONFIG_VERSION,
        schema: typeof schema === "string" ? schema : JSON.stringify(schema),
        schemaLanguage,
    };
}

/**
 * Create a standard config for a specific resource sub-type
 */
export function createStandardConfigForSubType(
    subType: ResourceSubType,
    overrides: Partial<StandardVersionConfigObject> = {},
): StandardVersionConfigObject {
    const baseConfigs: Record<ResourceSubType, Partial<StandardVersionConfigObject>> = {
        [ResourceSubType.StandardDataStructure]: {
            validation: {
                strictMode: true,
                rules: {
                    type: "object",
                    additionalProperties: false,
                },
            },
            schemaLanguage: "json-schema",
        },
        [ResourceSubType.StandardPrompt]: {
            format: {
                defaultFormat: "markdown",
            },
            props: {
                templateType: "promise",
                trackingEnabled: true,
            },
        },
        // Routine types
        [ResourceSubType.RoutineInternalAction]: {
            format: { defaultFormat: "json" },
        },
        [ResourceSubType.RoutineCode]: {
            format: { defaultFormat: "javascript" },
        },
        [ResourceSubType.RoutineData]: {
            validation: { strictMode: false },
        },
        [ResourceSubType.RoutineGenerate]: {
            format: { defaultFormat: "template" },
        },
        [ResourceSubType.RoutineInformational]: {
            format: { defaultFormat: "markdown" },
        },
        [ResourceSubType.RoutineMultiStep]: {
            format: { defaultFormat: "bpmn" },
        },
        [ResourceSubType.RoutineSmartContract]: {
            format: { defaultFormat: "solidity" },
        },
        [ResourceSubType.RoutineWeb]: {
            format: { defaultFormat: "html" },
        },
        // Code types
        [ResourceSubType.CodeSmartContract]: {
            format: { defaultFormat: "solidity" },
        },
        [ResourceSubType.RoutineApi]: {
            format: {
                defaultFormat: "openapi",
                options: {
                    version: "3.1.0",
                },
            },
            schemaLanguage: "openapi",
        },
        [ResourceSubType.CodeDataConverter]: {
            props: {
                widgetType: "component",
                reactive: true,
            },
        },
    };

    return mergeWithBaseDefaults<StandardVersionConfigObject>({
        ...baseConfigs[subType],
        ...overrides,
    });
}
