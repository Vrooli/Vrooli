import { describe, expect, it } from "vitest";
import { apiKeyFixtures, apiKeyTestDataFactory } from "./__test/fixtures/apiKeyFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test/validationTestUtils.js";
import { apiKeyValidation } from "./apiKey.js";

describe("apiKeyValidation", () => {
    // Run standard test suite
    runStandardValidationTests(apiKeyValidation, apiKeyFixtures, "apiKey");

    // API Key-specific tests
    describe("apiKey-specific validation", () => {
        const defaultParams = { omitFields: [] };

        describe("create validation", () => {
            const createSchema = apiKeyValidation.create(defaultParams);

            describe("name field validation", () => {
                it("should reject name shorter than 3 characters", async () => {
                    await testValidation(
                        createSchema,
                        apiKeyFixtures.invalid.nameTooShort.create,
                        false,
                        /character under the limit/i,
                    );
                });

                it("should reject name longer than 50 characters", async () => {
                    await testValidation(
                        createSchema,
                        apiKeyFixtures.invalid.nameTooLong.create,
                        false,
                        /character over the limit/i,
                    );
                });

                it("should accept name with exactly 3 characters", async () => {
                    const result = await testValidation(
                        createSchema,
                        apiKeyFixtures.edgeCases.minLengthName.create,
                        true,
                    );
                    expect(result.name).toBe("abc");
                });

                it("should accept name with exactly 50 characters", async () => {
                    const result = await testValidation(
                        createSchema,
                        apiKeyFixtures.edgeCases.maxLengthName.create,
                        true,
                    );
                    expect(result.name).toHaveLength(50);
                });
            });

            describe("limitHard field validation", () => {
                it("should accept negative limitHard as bigint string", async () => {
                    // bigIntString validation accepts negative values
                    const result = await testValidation(
                        createSchema,
                        apiKeyFixtures.invalid.negativeLimitHard.create,
                        true,
                    );
                    expect(result.limitHard).toBe("-1000");
                });

                it("should reject non-integer limitHard", async () => {
                    await testValidation(
                        createSchema,
                        apiKeyFixtures.invalid.invalidLimitHard.create,
                        false,
                        /Must be a valid integer/i,
                    );
                });

                it("should accept zero limitHard", async () => {
                    const result = await testValidation(
                        createSchema,
                        apiKeyFixtures.edgeCases.zeroLimitHard.create,
                        true,
                    );
                    expect(result.limitHard).toBe("0");
                });

                it("should accept large bigint as string", async () => {
                    const result = await testValidation(
                        createSchema,
                        apiKeyFixtures.edgeCases.bigIntAsString.create,
                        true,
                    );
                    expect(result.limitHard).toBe("9223372036854775807");
                    expect(result.limitSoft).toBe("9223372036854775806");
                });
            });

            describe("absoluteMax field validation", () => {
                it("should reject negative absoluteMax", async () => {
                    await testValidation(
                        createSchema,
                        apiKeyFixtures.invalid.negativeAbsoluteMax.create,
                        false,
                        /must be greater than or equal to 0/i,
                    );
                });

                it("should reject absoluteMax over 1000000", async () => {
                    await testValidation(
                        createSchema,
                        apiKeyFixtures.invalid.absoluteMaxTooLarge.create,
                        false,
                        /must be less than or equal to 1000000/i,
                    );
                });

                it("should accept zero absoluteMax", async () => {
                    const result = await testValidation(
                        createSchema,
                        apiKeyFixtures.edgeCases.zeroAbsoluteMax.create,
                        true,
                    );
                    expect(result.absoluteMax).toBe(0);
                });

                it("should accept maximum absoluteMax", async () => {
                    const result = await testValidation(
                        createSchema,
                        apiKeyFixtures.edgeCases.maxAbsoluteMax.create,
                        true,
                    );
                    expect(result.absoluteMax).toBe(1000000);
                });

                it("should reject float values for absoluteMax", async () => {
                    await testValidation(
                        createSchema,
                        apiKeyFixtures.edgeCases.floatAbsoluteMax.create,
                        false,
                        /must be an integer/i,
                    );
                });
            });

            describe("stopAtLimit field validation", () => {
                it("should convert string boolean to boolean", async () => {
                    const result = await testValidation(
                        createSchema,
                        apiKeyFixtures.edgeCases.booleanStringStopAtLimit.create,
                        true,
                    );
                    expect(result.stopAtLimit).toBe(true);
                    expect(result.stopAtLimit).to.be.a("boolean");
                });

                it("should accept boolean values", async () => {
                    const result = await testValidation(
                        createSchema,
                        apiKeyFixtures.minimal.create,
                        true,
                    );
                    expect(result.stopAtLimit).toBe(true);
                });
            });

            describe("permissions field validation", () => {
                it("should reject permissions longer than 4096 characters", async () => {
                    await testValidation(
                        createSchema,
                        apiKeyFixtures.invalid.permissionsTooLong.create,
                        false,
                        /character over the limit/i,
                    );
                });

                it("should accept empty permissions", async () => {
                    const result = await testValidation(
                        createSchema,
                        apiKeyFixtures.edgeCases.emptyPermissions.create,
                        true,
                    );
                    expect(result).toHaveProperty("permissions");
                    expect(result.permissions).toBe("");
                });

                it("should accept permissions with exactly 4096 characters", async () => {
                    const result = await testValidation(
                        createSchema,
                        apiKeyFixtures.edgeCases.maxLengthPermissions.create,
                        true,
                    );
                    expect(result.permissions).toHaveLength(4096);
                });

                it("should accept JSON string permissions", async () => {
                    const result = await testValidation(
                        createSchema,
                        apiKeyFixtures.complete.create,
                        true,
                    );
                    expect(result.permissions).to.be.a("string");
                    const parsed = JSON.parse(result.permissions);
                    expect(parsed).toHaveProperty("read", true);
                    expect(parsed).toHaveProperty("write", true);
                    expect(parsed).toHaveProperty("delete", false);
                });
            });

            describe("disabled field validation", () => {
                it("should accept boolean disabled field", async () => {
                    const result = await testValidation(
                        createSchema,
                        apiKeyFixtures.complete.create,
                        true,
                    );
                    expect(result.disabled).toBe(false);
                });

                it("should transform string to boolean for disabled", async () => {
                    const data = {
                        ...apiKeyFixtures.minimal.create,
                        disabled: "true",
                    };
                    const result = await testValidation(createSchema, data, true);
                    expect(result.disabled).toBe(true);
                    expect(result.disabled).to.be.a("boolean");
                });
            });
        });

        describe("update validation", () => {
            const updateSchema = apiKeyValidation.update(defaultParams);

            it("should accept update with only name", async () => {
                const result = await testValidation(
                    updateSchema,
                    apiKeyFixtures.edgeCases.updateOnlyName.update,
                    true,
                );
                expect(result).toHaveProperty("id");
                expect(result).toHaveProperty("name", "Updated Name Only");
                expect(Object.keys(result)).toHaveLength(2);
            });

            it("should accept update with only disabled flag", async () => {
                const result = await testValidation(
                    updateSchema,
                    apiKeyFixtures.edgeCases.updateOnlyDisabled.update,
                    true,
                );
                expect(result).toHaveProperty("id");
                expect(result).toHaveProperty("disabled", true);
                expect(Object.keys(result)).toHaveLength(2);
            });

            it("should accept complete update", async () => {
                const result = await testValidation(
                    updateSchema,
                    apiKeyFixtures.complete.update,
                    true,
                );
                expect(result).toHaveProperty("id");
                expect(result).toHaveProperty("disabled", true);
                expect(result).toHaveProperty("limitHard", "6000000");
                expect(result).toHaveProperty("limitSoft", "5000000");
                expect(result).toHaveProperty("name", "Updated API Key");
                expect(result).toHaveProperty("stopAtLimit", true);
                expect(result).toHaveProperty("absoluteMax", 600000);
                expect(result).toHaveProperty("permissions");
            });

            it("should allow all fields to be optional except id", async () => {
                const result = await testValidation(
                    updateSchema,
                    { id: "500000000000000001" },
                    true,
                );
                expect(result).toEqual({ id: "500000000000000001" });
            });
        });
    });

    describe("batch validation scenarios", () => {
        it("should validate multiple apiKey create scenarios", async () => {
            const defaultParams = { omitFields: [] };
            const schema = apiKeyValidation.create(defaultParams);

            await testValidationBatch(schema, [
                {
                    data: apiKeyTestDataFactory.createMinimal(),
                    shouldPass: true,
                    description: "minimal valid apiKey",
                },
                {
                    data: apiKeyTestDataFactory.createComplete(),
                    shouldPass: true,
                    description: "complete valid apiKey",
                },
                {
                    data: apiKeyTestDataFactory.createMinimal({ name: "a" }),
                    shouldPass: false,
                    expectedError: /characters under the limit/i,
                    description: "name too short",
                },
                {
                    data: apiKeyTestDataFactory.createMinimal({ absoluteMax: -1 }),
                    shouldPass: false,
                    expectedError: /must be greater than or equal to 0/i,
                    description: "negative absoluteMax",
                },
                {
                    data: apiKeyTestDataFactory.createMinimal({ limitHard: "not-a-number" }),
                    shouldPass: false,
                    expectedError: /Must be a valid integer/i,
                    description: "invalid limitHard",
                },
                {
                    data: { name: "Test Key" }, // Missing required fields
                    shouldPass: false,
                    expectedError: /required/i,
                    description: "missing required fields",
                },
            ]);
        });

        it("should validate multiple apiKey update scenarios", async () => {
            const defaultParams = { omitFields: [] };
            const schema = apiKeyValidation.update(defaultParams);

            await testValidationBatch(schema, [
                {
                    data: apiKeyTestDataFactory.updateMinimal(),
                    shouldPass: true,
                    description: "minimal valid update",
                },
                {
                    data: apiKeyTestDataFactory.updateComplete(),
                    shouldPass: true,
                    description: "complete valid update",
                },
                {
                    data: apiKeyTestDataFactory.updateMinimal({ name: "ab" }),
                    shouldPass: false,
                    expectedError: /character under the limit/i,
                    description: "name too short",
                },
                {
                    data: { name: "Valid Name" }, // Missing id
                    shouldPass: false,
                    expectedError: "This field is required",
                    description: "missing required id",
                },
            ]);
        });
    });

    describe("omitFields functionality", () => {
        it("should omit specified fields in create", async () => {
            const schema = apiKeyValidation.create({ omitFields: ["permissions", "disabled"] });
            const data = {
                ...apiKeyFixtures.complete.create,
            };
            const result = await schema.validate(data, { stripUnknown: true });
            expect(result).to.not.have.property("permissions");
            expect(result).to.not.have.property("disabled");
            expect(result).toHaveProperty("id");
            expect(result).toHaveProperty("name");
        });

        it("should omit nested fields if supported", async () => {
            const schema = apiKeyValidation.update({ omitFields: ["name", "absoluteMax"] });
            const data = apiKeyFixtures.complete.update;
            const result = await schema.validate(data, { stripUnknown: true });
            expect(result).to.not.have.property("name");
            expect(result).to.not.have.property("absoluteMax");
            expect(result).toHaveProperty("id");
        });
    });

    describe("test data factory", () => {
        it("should generate valid minimal create data", () => {
            const data = apiKeyTestDataFactory.createMinimal();
            expect(data).toHaveProperty("id");
            expect(data).toHaveProperty("limitHard");
            expect(data).toHaveProperty("name");
            expect(data).toHaveProperty("stopAtLimit");
            expect(data).toHaveProperty("absoluteMax");
        });

        it("should generate valid complete create data", () => {
            const data = apiKeyTestDataFactory.createComplete();
            expect(data).to.have.all.keys(
                "id", "disabled", "limitHard", "limitSoft",
                "name", "stopAtLimit", "absoluteMax", "permissions",
            );
        });

        it("should allow overrides", () => {
            const customName = "Custom API Key";
            const data = apiKeyTestDataFactory.createComplete({ name: customName });
            expect(data.name).toBe(customName);
        });

        it("should access test scenarios", () => {
            const invalidData = apiKeyTestDataFactory.forScenario("nameTooShort");
            expect(invalidData.create.name).toHaveLength(2);
        });

        it("should apply default values when creating with empty object", () => {
            const data = apiKeyTestDataFactory.createMinimal({});
            expect(data.id).toBe("500000000000000001");
            expect(data.limitHard).toBe("1000000");
            expect(data.name).toBe("Test API Key");
            expect(data.stopAtLimit).toBe(true);
            expect(data.absoluteMax).toBe(100000);
        });

        it("should preserve undefined and falsy values in customizers", () => {
            const data = apiKeyTestDataFactory.createMinimal({ 
                stopAtLimit: false,
                absoluteMax: 0 
            });
            expect(data.stopAtLimit).toBe(false);
            expect(data.absoluteMax).toBe(0);
        });

        it("should apply update customizer defaults", () => {
            const data = apiKeyTestDataFactory.updateMinimal({});
            expect(data.id).toBe("500000000000000001");
        });

        it("should override factory defaults with provided values", () => {
            const customId = "999999999999999999";
            const data = apiKeyTestDataFactory.createMinimal({ 
                id: customId,
                limitHard: "5000000",
                name: "Override Test",
                stopAtLimit: false,
                absoluteMax: 50000
            });
            expect(data.id).toBe(customId);
            expect(data.limitHard).toBe("5000000");
            expect(data.name).toBe("Override Test");
            expect(data.stopAtLimit).toBe(false);
            expect(data.absoluteMax).toBe(50000);
        });
    });
});

// Verify the validation matches expected shapes
const validIds = {
    id1: "500000000000000001",
    id2: "500000000000000002",
    id3: "500000000000000003",
};
