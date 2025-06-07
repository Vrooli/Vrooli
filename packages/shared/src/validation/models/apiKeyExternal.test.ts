import { describe, it, expect } from "vitest";
import { apiKeyExternalValidation } from "./apiKeyExternal.js";
import { apiKeyExternalFixtures } from "./__test__/fixtures/apiKeyExternalFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test__/validationTestUtils.js";
import { API_KEY_EXTERNAL_MAX_LENGTH, NAME_MAX_LENGTH } from "../utils/validationConstants.js";

describe("apiKeyExternalValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        apiKeyExternalValidation,
        apiKeyExternalFixtures,
        "apiKeyExternal",
    );

    describe("specific field validations", () => {
        const defaultParams = { omitFields: [] };

        describe("key field", () => {
            const createSchema = apiKeyExternalValidation.create(defaultParams);

            it("should accept various key formats", async () => {
                const scenarios = [
                    {
                        data: { ...apiKeyExternalFixtures.minimal.create, key: "sk-test-123" },
                        shouldPass: true,
                        description: "simple key format",
                    },
                    {
                        data: { ...apiKeyExternalFixtures.minimal.create, key: "api_key_12345" },
                        shouldPass: true,
                        description: "underscore format",
                    },
                    {
                        data: { ...apiKeyExternalFixtures.minimal.create, key: "Bearer abcd1234" },
                        shouldPass: true,
                        description: "bearer token format",
                    },
                    {
                        data: { ...apiKeyExternalFixtures.minimal.create, key: "a".repeat(API_KEY_EXTERNAL_MAX_LENGTH) },
                        shouldPass: true,
                        description: "maximum length key",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject empty or whitespace-only keys", async () => {
                const scenarios = [
                    {
                        data: { ...apiKeyExternalFixtures.minimal.create, key: "" },
                        shouldPass: false,
                        expectedError: /required/i,
                        description: "empty key",
                    },
                    {
                        data: { ...apiKeyExternalFixtures.minimal.create, key: "   " },
                        shouldPass: false,
                        expectedError: /required/i,
                        description: "whitespace-only key",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject keys that are too long", async () => {
                await testValidation(
                    createSchema,
                    apiKeyExternalFixtures.invalid.tooLongKey.create,
                    false,
                    /over the limit/,
                );
            });
        });

        describe("service field", () => {
            const createSchema = apiKeyExternalValidation.create(defaultParams);

            it("should accept various service names", async () => {
                const scenarios = [
                    {
                        data: { ...apiKeyExternalFixtures.minimal.create, service: "OpenAI" },
                        shouldPass: true,
                        description: "OpenAI service",
                    },
                    {
                        data: { ...apiKeyExternalFixtures.minimal.create, service: "Anthropic" },
                        shouldPass: true,
                        description: "Anthropic service",
                    },
                    {
                        data: { ...apiKeyExternalFixtures.minimal.create, service: "Google" },
                        shouldPass: true,
                        description: "Google service",
                    },
                    {
                        data: { ...apiKeyExternalFixtures.minimal.create, service: "Mistral" },
                        shouldPass: true,
                        description: "Mistral service",
                    },
                    {
                        data: { ...apiKeyExternalFixtures.minimal.create, service: "custom-service" },
                        shouldPass: true,
                        description: "custom service name",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject too long service names", async () => {
                await testValidation(
                    createSchema,
                    apiKeyExternalFixtures.invalid.tooLongService.create,
                    false,
                    /over the limit/,
                );
            });

            it("should reject empty service names", async () => {
                await testValidation(
                    createSchema,
                    { ...apiKeyExternalFixtures.minimal.create, service: "" },
                    false,
                    /required/i,
                );
            });
        });

        describe("name field", () => {
            const createSchema = apiKeyExternalValidation.create(defaultParams);

            it("should accept various name formats", async () => {
                const scenarios = [
                    {
                        data: { ...apiKeyExternalFixtures.minimal.create, name: "My API Key" },
                        shouldPass: true,
                        description: "simple name",
                    },
                    {
                        data: { ...apiKeyExternalFixtures.minimal.create, name: "OpenAI GPT-4 Key #1" },
                        shouldPass: true,
                        description: "detailed name with symbols",
                    },
                    {
                        data: { ...apiKeyExternalFixtures.minimal.create, name: "a".repeat(NAME_MAX_LENGTH) },
                        shouldPass: true,
                        description: "maximum length name",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject names that are too long", async () => {
                await testValidation(
                    createSchema,
                    apiKeyExternalFixtures.invalid.tooLongName.create,
                    false,
                    /over the limit/,
                );
            });
        });

        describe("disabled field", () => {
            const createSchema = apiKeyExternalValidation.create(defaultParams);
            const updateSchema = apiKeyExternalValidation.update(defaultParams);

            it("should accept boolean values for disabled field", async () => {
                const scenarios = [
                    {
                        data: { ...apiKeyExternalFixtures.minimal.create, disabled: true },
                        shouldPass: true,
                        description: "disabled true",
                    },
                    {
                        data: { ...apiKeyExternalFixtures.minimal.create, disabled: false },
                        shouldPass: true,
                        description: "disabled false",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should be optional in create", async () => {
                const dataWithoutDisabled = { ...apiKeyExternalFixtures.minimal.create };
                delete dataWithoutDisabled.disabled;
                
                await testValidation(createSchema, dataWithoutDisabled, true);
            });

            it("should be optional in update", async () => {
                const scenarios = [
                    {
                        data: { ...apiKeyExternalFixtures.minimal.update, disabled: true },
                        shouldPass: true,
                        description: "update with disabled field",
                    },
                    {
                        data: apiKeyExternalFixtures.minimal.update,
                        shouldPass: true,
                        description: "update without disabled field",
                    },
                ];

                await testValidationBatch(updateSchema, scenarios);
            });
        });
    });

    describe("update specific validations", () => {
        const updateSchema = apiKeyExternalValidation.update({ omitFields: [] });

        it("should allow partial updates", async () => {
            const scenarios = [
                {
                    data: { id: "123456789012345678", name: "Updated Name" },
                    shouldPass: true,
                    description: "update only name",
                },
                {
                    data: { id: "123456789012345678", disabled: true },
                    shouldPass: true,
                    description: "update only disabled status",
                },
                {
                    data: { id: "123456789012345678", key: "new-key-123" },
                    shouldPass: true,
                    description: "update only key",
                },
                {
                    data: { id: "123456789012345678", service: "NewService" },
                    shouldPass: true,
                    description: "update only service",
                },
            ];

            await testValidationBatch(updateSchema, scenarios);
        });

        it("should allow complete updates", async () => {
            await testValidation(
                updateSchema,
                apiKeyExternalFixtures.complete.update,
                true,
            );
        });
    });

    describe("field trimming and sanitization", () => {
        const createSchema = apiKeyExternalValidation.create({ omitFields: [] });

        it("should trim whitespace from string fields", async () => {
            const dataWithWhitespace = {
                id: "123456789012345678",
                key: "  sk-test-key  ",
                name: "  Test Key  ",
                service: "  OpenAI  ",
            };

            const result = await testValidation(createSchema, dataWithWhitespace, true);
            
            // Check that whitespace was trimmed
            if (result.key) expect(result.key).to.equal("sk-test-key");
            if (result.name) expect(result.name).to.equal("Test Key");
            if (result.service) expect(result.service).to.equal("OpenAI");
        });

        it("should handle empty strings properly", async () => {
            await testValidation(
                createSchema,
                apiKeyExternalFixtures.edgeCases.emptyStrings.create,
                false,
                /required/i,
            );
        });
    });
});
