// AI_CHECK: TEST_COVERAGE=2 | LAST: 2025-06-18
import { describe, expect, it, vi } from "vitest";
import { ResourceSubType } from "../../api/types.js";
import { standardConfigFixtures } from "../../__test/fixtures/config/standardConfigFixtures.js";
import { StandardVersionConfig, type StandardVersionConfigObject } from "./standard.js";

describe("StandardVersionConfig", () => {
    // Basic fixture integrity tests
    describe("fixture integrity", () => {
        it("should create valid configs from all fixtures", () => {
            // Test minimal fixture
            const minimalConfig = new StandardVersionConfig({
                config: standardConfigFixtures.minimal,
                resourceSubType: ResourceSubType.StandardDataStructure,
            });
            expect(minimalConfig).toBeDefined();
            expect(minimalConfig.__version).toBeDefined();
            
            // Test complete fixture
            const completeConfig = new StandardVersionConfig({
                config: standardConfigFixtures.complete,
                resourceSubType: ResourceSubType.StandardPrompt,
            });
            expect(completeConfig).toBeDefined();
            expect(completeConfig.__version).toBeDefined();
            
            // Test all variants
            Object.entries(standardConfigFixtures.variants).forEach(([variantName, variantData]) => {
                const variantConfig = new StandardVersionConfig({
                    config: variantData,
                    resourceSubType: ResourceSubType.StandardDataStructure,
                });
                expect(variantConfig).toBeDefined();
                expect(variantConfig.__version).toBeDefined();
            });
        });

        it("should maintain round-trip consistency for all fixtures", () => {
            const allFixtures = [
                { name: "minimal", data: standardConfigFixtures.minimal },
                { name: "complete", data: standardConfigFixtures.complete },
                { name: "withDefaults", data: standardConfigFixtures.withDefaults },
                ...Object.entries(standardConfigFixtures.variants).map(([name, data]) => ({ name: `variant:${name}`, data })),
            ];

            allFixtures.forEach(({ name, data }) => {
                const config1 = new StandardVersionConfig({
                    config: data,
                    resourceSubType: ResourceSubType.StandardDataStructure,
                });
                const exported1 = config1.export();
                
                const config2 = new StandardVersionConfig({
                    config: exported1,
                    resourceSubType: ResourceSubType.StandardDataStructure,
                });
                const exported2 = config2.export();
                
                expect(exported1).toEqual(exported2);
            });
        });
    });

    // Standard-specific business logic tests
    describe("standard-specific functionality", () => {
        const mockLogger = {
            log: vi.fn(),
            info: vi.fn(),
            warn: vi.fn(),
            error: vi.fn(),
        };

    describe("constructor", () => {
        it("should create config with provided values", () => {
            const configObj = {
                ...standardConfigFixtures.complete,
                resources: [],
            };

            const config = new StandardVersionConfig({
                config: configObj,
                resourceSubType: ResourceSubType.StandardDataStructure,
            });

            expect(config.validation).toEqual(configObj.validation);
            expect(config.format).toEqual(configObj.format);
            expect(config.compatibility).toEqual(configObj.compatibility);
            expect(config.compliance).toEqual(configObj.compliance);
            expect(config.schema).toBe(configObj.schema);
            expect(config.schemaLanguage).toBe(configObj.schemaLanguage);
            expect(config.props).toEqual(configObj.props);
            expect(config.resourceSubType).toBe(ResourceSubType.StandardDataStructure);
        });

        it("should handle minimal config", () => {
            const configObj = {
                ...standardConfigFixtures.minimal,
                resources: [],
            };

            const config = new StandardVersionConfig({
                config: configObj,
                resourceSubType: ResourceSubType.StandardPrompt,
            });

            expect(config.validation).toBeUndefined();
            expect(config.format).toBeUndefined();
            expect(config.compatibility).toBeUndefined();
            expect(config.compliance).toBeUndefined();
            expect(config.schema).toBeUndefined();
            expect(config.schemaLanguage).toBeUndefined();
            expect(config.props).toBeUndefined();
            expect(config.resourceSubType).toBe(ResourceSubType.StandardPrompt);
        });
    });

    describe("static methods", () => {
        describe("parse", () => {
            it("should parse valid config", () => {
                const version = {
                    config: {
                        ...standardConfigFixtures.variants.minimalValidation,
                        resources: [],
                        schema: "{\"type\": \"string\"}",
                    },
                    resourceSubType: ResourceSubType.StandardPrompt,
                };

                const config = StandardVersionConfig.parse(version, mockLogger);

                expect(config.validation?.strictMode).toBe(false);
                expect(config.schema).toBe("{\"type\": \"string\"}");
                expect(config.resourceSubType).toBe(ResourceSubType.StandardPrompt);
            });

            it("should handle missing config with fallback resourceSubType", () => {
                const version = {
                    config: null,
                    resourceSubType: undefined as any,
                };

                const config = StandardVersionConfig.parse(version, mockLogger);

                expect(config.resourceSubType).toBe(ResourceSubType.StandardDataStructure);
                expect(config.__version).toBeDefined();
                expect(config.resources).toEqual([]);
            });
        });

        describe("manual creation", () => {
            it("should create config with manual constructor call", () => {
                const config = new StandardVersionConfig({
                    config: {
                        ...standardConfigFixtures.minimal,
                        resources: [],
                    },
                    resourceSubType: ResourceSubType.StandardDataStructure,
                });

                expect(config.__version).toBe(standardConfigFixtures.minimal.__version);
                expect(config.resources).toEqual([]);
                expect(config.resourceSubType).toBe(ResourceSubType.StandardDataStructure);
                expect(config.validation).toBeUndefined();
                expect(config.format).toBeUndefined();
                expect(config.compatibility).toBeUndefined();
                expect(config.compliance).toBeUndefined();
                expect(config.schema).toBeUndefined();
                expect(config.schemaLanguage).toBeUndefined();
                expect(config.props).toBeUndefined();
            });
        });
    });

    describe("export", () => {
        it("should export all config properties", () => {
            const originalConfig = {
                ...standardConfigFixtures.complete,
                resources: [{ link: "https://example.com", translations: [{ language: "en", name: "Example" }] }],
                validation: {
                    strictMode: true,
                    rules: { pattern: "^[a-z]+$" },
                },
                format: {
                    defaultFormat: "yaml",
                },
                schema: "{\"type\": \"array\"}",
            };

            const config = new StandardVersionConfig({
                config: originalConfig,
                resourceSubType: ResourceSubType.StandardDataStructure,
            });
            const exported = config.export();

            expect(exported.validation).toEqual(originalConfig.validation);
            expect(exported.format).toEqual(originalConfig.format);
            expect(exported.schema).toBe(originalConfig.schema);
            expect(exported.schemaLanguage).toBe(originalConfig.schemaLanguage);
            expect(exported.__version).toBe(standardConfigFixtures.complete.__version);
            expect(exported.resources).toEqual(originalConfig.resources);
        });

        it("should export undefined values correctly", () => {
            const minimalConfig = {
                ...standardConfigFixtures.minimal,
                resources: [],
            };

            const config = new StandardVersionConfig({
                config: minimalConfig,
                resourceSubType: ResourceSubType.StandardPrompt,
            });
            const exported = config.export();

            expect(exported.validation).toBeUndefined();
            expect(exported.format).toBeUndefined();
            expect(exported.compatibility).toBeUndefined();
            expect(exported.compliance).toBeUndefined();
            expect(exported.schema).toBeUndefined();
            expect(exported.schemaLanguage).toBeUndefined();
            expect(exported.props).toBeUndefined();
        });
    });

    describe("validation configuration", () => {
        it("should handle strict mode validation", () => {
            const config = new StandardVersionConfig({
                config: {
                    ...standardConfigFixtures.variants.dataStructureStandard,
                    resources: [],
                    validation: {
                        strictMode: true,
                        rules: {
                            required: ["name", "type"],
                            properties: {
                                name: { type: "string" },
                                type: { enum: ["object", "array"] },
                            },
                        },
                        errorMessages: {
                            required: "Field is required",
                            type: "Invalid type",
                        },
                    },
                },
                resourceSubType: ResourceSubType.StandardDataStructure,
            });

            expect(config.validation?.strictMode).toBe(true);
            expect(config.validation?.rules?.required).toEqual(["name", "type"]);
            expect(config.validation?.errorMessages?.required).toBe("Field is required");
        });

        it("should handle non-strict validation", () => {
            const config = new StandardVersionConfig({
                config: {
                    ...standardConfigFixtures.variants.minimalValidation,
                    resources: [],
                },
                resourceSubType: ResourceSubType.StandardPrompt,
            });

            expect(config.validation?.strictMode).toBe(false);
        });
    });

    describe("format configuration", () => {
        it("should handle various format settings", () => {
            const config = new StandardVersionConfig({
                config: {
                    ...standardConfigFixtures.complete,
                    resources: [],
                    format: {
                        defaultFormat: "xml",
                        options: {
                            encoding: "utf-8",
                            indent: 4,
                            preserveWhitespace: true,
                        },
                    },
                },
                resourceSubType: ResourceSubType.StandardDataStructure,
            });

            expect(config.format?.defaultFormat).toBe("xml");
            expect(config.format?.options?.encoding).toBe("utf-8");
            expect(config.format?.options?.indent).toBe(4);
        });
    });

    describe("compatibility configuration", () => {
        it("should handle compatibility settings", () => {
            const config = new StandardVersionConfig({
                config: {
                    ...standardConfigFixtures.variants.technicalStandard,
                    resources: [],
                    compatibility: {
                        minimumRequirements: {
                            nodeVersion: ">=16.0.0",
                            browser: "Chrome 90+",
                        },
                        knownIssues: [
                            "Performance issue with large datasets",
                            "Memory leak in Safari",
                        ],
                        compatibleWith: [
                            "OpenAPI 3.0",
                            "JSON Schema Draft 7",
                        ],
                    },
                },
                resourceSubType: ResourceSubType.StandardPrompt,
            });

            expect(config.compatibility?.minimumRequirements?.nodeVersion).toBe(">=16.0.0");
            expect(config.compatibility?.knownIssues).toHaveLength(2);
            expect(config.compatibility?.compatibleWith).toContain("OpenAPI 3.0");
        });
    });

    describe("compliance configuration", () => {
        it("should handle compliance settings", () => {
            const config = new StandardVersionConfig({
                config: {
                    ...standardConfigFixtures.complete,
                    resources: [],
                    compliance: {
                        compliesWith: ["SOX", "HIPAA", "PCI DSS"],
                        certifications: [
                            {
                                name: "ISO 27001",
                                issuer: "ISO",
                                date: "2023-06-15",
                                expiration: "2026-06-15",
                            },
                            {
                                name: "SOC 2 Type II",
                                issuer: "AICPA",
                                date: "2023-12-01",
                            },
                        ],
                    },
                },
                resourceSubType: ResourceSubType.StandardPrompt,
            });

            expect(config.compliance?.compliesWith).toContain("SOX");
            expect(config.compliance?.certifications).toHaveLength(2);
            expect(config.compliance?.certifications?.[0].name).toBe("ISO 27001");
            expect(config.compliance?.certifications?.[1].expiration).toBeUndefined();
        });
    });

    describe("schema configuration", () => {
        it("should handle JSON schema", () => {
            const jsonSchema = JSON.stringify({
                type: "object",
                properties: {
                    name: { type: "string" },
                    age: { type: "number", minimum: 0 },
                },
                required: ["name"],
            });

            const config = new StandardVersionConfig({
                config: {
                    ...standardConfigFixtures.minimal,
                    resources: [],
                    schema: jsonSchema,
                    schemaLanguage: "json-schema",
                },
                resourceSubType: ResourceSubType.StandardDataStructure,
            });

            expect(config.schema).toBe(jsonSchema);
            expect(config.schemaLanguage).toBe("json-schema");
        });

        it("should handle other schema languages", () => {
            const xmlSchema = "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\"></xs:schema>";

            const config = new StandardVersionConfig({
                config: {
                    ...standardConfigFixtures.minimal,
                    resources: [],
                    schema: xmlSchema,
                    schemaLanguage: "xml-schema",
                },
                resourceSubType: ResourceSubType.StandardPrompt,
            });

            expect(config.schema).toBe(xmlSchema);
            expect(config.schemaLanguage).toBe("xml-schema");
        });
    });

    describe("props configuration", () => {
        it("should handle custom props", () => {
            const config = new StandardVersionConfig({
                config: {
                    ...standardConfigFixtures.complete,
                    resources: [],
                    props: {
                        generator: "custom-tool-v1.2.3",
                        generatedAt: "2024-01-15T10:30:00Z",
                        metadata: {
                            author: "John Doe",
                            purpose: "Data validation",
                        },
                        flags: ["experimental", "beta"],
                    },
                },
                resourceSubType: ResourceSubType.StandardDataStructure,
            });

            expect(config.props?.generator).toBe("custom-tool-v1.2.3");
            expect(config.props?.metadata).toEqual({
                author: "John Doe",
                purpose: "Data validation",
            });
            expect(config.props?.flags).toEqual(["experimental", "beta"]);
        });
    });

    describe("setter methods", () => {
        it("should set validation", () => {
            const config = new StandardVersionConfig({
                config: {
                    ...standardConfigFixtures.minimal,
                    resources: [],
                },
                resourceSubType: ResourceSubType.StandardDataStructure,
            });

            const validation = {
                strictMode: true,
                rules: { maxLength: 100 },
                errorMessages: { maxLength: "Too long" },
            };

            config.setValidation(validation);
            expect(config.validation).toEqual(validation);

            const exported = config.export();
            expect(exported.validation).toEqual(validation);
        });

        it("should set format", () => {
            const config = new StandardVersionConfig({
                config: {
                    ...standardConfigFixtures.minimal,
                    resources: [],
                },
                resourceSubType: ResourceSubType.StandardPrompt,
            });

            const format = {
                defaultFormat: "csv",
                options: { delimiter: "," },
            };

            config.setFormat(format);
            expect(config.format).toEqual(format);

            const exported = config.export();
            expect(exported.format).toEqual(format);
        });

        it("should set compatibility", () => {
            const config = new StandardVersionConfig({
                config: {
                    ...standardConfigFixtures.minimal,
                    resources: [],
                },
                resourceSubType: ResourceSubType.StandardDataStructure,
            });

            const compatibility = {
                minimumRequirements: { ram: "8GB" },
                knownIssues: ["Memory leak"],
                compatibleWith: ["v1.0", "v2.0"],
            };

            config.setCompatibility(compatibility);
            expect(config.compatibility).toEqual(compatibility);

            const exported = config.export();
            expect(exported.compatibility).toEqual(compatibility);
        });

        it("should set compliance", () => {
            const config = new StandardVersionConfig({
                config: {
                    ...standardConfigFixtures.minimal,
                    resources: [],
                },
                resourceSubType: ResourceSubType.StandardPrompt,
            });

            const compliance = {
                compliesWith: ["GDPR"],
                certifications: [{
                    name: "Privacy Shield",
                    issuer: "US Dept of Commerce",
                    date: "2024-01-01",
                }],
            };

            config.setCompliance(compliance);
            expect(config.compliance).toEqual(compliance);

            const exported = config.export();
            expect(exported.compliance).toEqual(compliance);
        });

        it("should set schema", () => {
            const config = new StandardVersionConfig({
                config: {
                    ...standardConfigFixtures.minimal,
                    resources: [],
                },
                resourceSubType: ResourceSubType.StandardDataStructure,
            });

            const schema = "{\"type\": \"array\", \"items\": {\"type\": \"string\"}}";

            config.setSchema(schema);
            expect(config.schema).toBe(schema);

            const exported = config.export();
            expect(exported.schema).toBe(schema);
        });

        it("should set schemaLanguage", () => {
            const config = new StandardVersionConfig({
                config: {
                    ...standardConfigFixtures.minimal,
                    resources: [],
                },
                resourceSubType: ResourceSubType.StandardPrompt,
            });

            const schemaLanguage = "avro-schema";

            config.setSchemaLanguage(schemaLanguage);
            expect(config.schemaLanguage).toBe(schemaLanguage);

            const exported = config.export();
            expect(exported.schemaLanguage).toBe(schemaLanguage);
        });

        it("should handle setting undefined values", () => {
            const config = new StandardVersionConfig({
                config: {
                    ...standardConfigFixtures.minimal,
                    resources: [],
                    validation: { strictMode: true },
                    format: { defaultFormat: "json" },
                },
                resourceSubType: ResourceSubType.StandardDataStructure,
            });

            config.setValidation(undefined);
            config.setFormat(undefined);

            expect(config.validation).toBeUndefined();
            expect(config.format).toBeUndefined();
        });

        it("should handle chaining setters", () => {
            const config = new StandardVersionConfig({
                config: {
                    ...standardConfigFixtures.minimal,
                    resources: [],
                },
                resourceSubType: ResourceSubType.StandardPrompt,
            });

            config.setValidation({ strictMode: false });
            config.setFormat({ defaultFormat: "yaml" });
            config.setSchema("{\"type\": \"number\"}");
            config.setSchemaLanguage("json-schema");

            const exported = config.export();
            expect(exported.validation?.strictMode).toBe(false);
            expect(exported.format?.defaultFormat).toBe("yaml");
            expect(exported.schema).toBe("{\"type\": \"number\"}");
            expect(exported.schemaLanguage).toBe("json-schema");
        });
    });

    describe("integration scenarios", () => {
        it("should handle complete standard configuration", () => {
            const completeConfig: StandardVersionConfigObject = {
                __version: "1.0",
                resources: [],
                validation: {
                    strictMode: true,
                    rules: { type: "object" },
                    errorMessages: { type: "Must be an object" },
                },
                format: {
                    defaultFormat: "json",
                    options: { pretty: true },
                },
                compatibility: {
                    minimumRequirements: { version: "2.0" },
                    knownIssues: [],
                    compatibleWith: ["Standard A", "Standard B"],
                },
                compliance: {
                    compliesWith: ["Regulation X"],
                    certifications: [],
                },
                schema: "{\"type\": \"object\"}",
                schemaLanguage: "json-schema",
                props: { custom: "value" },
            };

            const config = new StandardVersionConfig({
                config: completeConfig,
                resourceSubType: ResourceSubType.StandardPrompt,
            });

            expect(config.validation?.strictMode).toBe(true);
            expect(config.format?.defaultFormat).toBe("json");
            expect(config.compatibility?.compatibleWith).toContain("Standard A");
            expect(config.compliance?.compliesWith).toContain("Regulation X");
            expect(config.schema).toBe("{\"type\": \"object\"}");
            expect(config.schemaLanguage).toBe("json-schema");
            expect(config.props?.custom).toBe("value");
            expect(config.resourceSubType).toBe(ResourceSubType.StandardPrompt);
        });

        it("should maintain consistency through parse and export cycle", () => {
            const originalConfig = {
                __version: "1.0",
                resources: [],
                validation: { strictMode: true },
                schema: "{\"type\": \"string\"}",
                schemaLanguage: "json-schema",
            };

            const version = {
                config: originalConfig,
                resourceSubType: ResourceSubType.StandardPrompt,
            };

            const parsed = StandardVersionConfig.parse(version, mockLogger);
            const exported = parsed.export();

            expect(exported.validation).toEqual(originalConfig.validation);
            expect(exported.schema).toBe(originalConfig.schema);
            expect(exported.schemaLanguage).toBe(standardConfigFixtures.complete.schemaLanguage);
        });
    });
    });
});
