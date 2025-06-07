import { describe, it, expect } from "vitest";
import { walletValidation } from "./wallet.js";
import { walletFixtures, walletTestDataFactory } from "./__test__/fixtures/walletFixtures.js";
import { testValidation, testValidationBatch } from "./__test__/validationTestUtils.js";

describe("walletValidation", () => {
    // Run standard test suite (only update since create isn't supported)
    describe("walletValidation - Standard Tests", () => {
        const defaultParams = { omitFields: [] };

        describe("create validation", () => {
            it("should not have create method", () => {
                expect(walletValidation.create).to.be.undefined;
            });
        });

        describe("update validation", () => {
            const updateSchema = walletValidation.update(defaultParams);

            it("should accept minimal valid data", async () => {
                const result = await testValidation(
                    updateSchema,
                    walletFixtures.minimal.update,
                    true,
                );
                expect(result).to.have.property("id");
                expect(result).to.not.have.property("name");
            });

            it("should accept complete valid data", async () => {
                const result = await testValidation(
                    updateSchema,
                    walletFixtures.complete.update,
                    true,
                );
                expect(result).to.have.property("id");
                expect(result).to.have.property("name");
                expect(result.name).to.equal("My Primary Wallet");
            });

            it("should reject missing id", async () => {
                await testValidation(
                    updateSchema,
                    walletFixtures.invalid.missingRequired.update,
                    false,
                    "This field is required",
                );
            });

            it("should reject invalid field types", async () => {
                await testValidation(
                    updateSchema,
                    walletFixtures.invalid.invalidTypes.update,
                    false,
                );
            });

            it("should strip unknown fields", async () => {
                const dataWithExtra = {
                    ...walletFixtures.minimal.update,
                    unknownField: "should be removed",
                    balance: 100, // Not a valid field
                };
                const result = await testValidation(updateSchema, dataWithExtra, true);
                expect(result).to.not.have.property("unknownField");
                expect(result).to.not.have.property("balance");
            });
        });
    });

    // Wallet-specific tests
    describe("wallet-specific validation", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = walletValidation.update(defaultParams);

        describe("name field validation", () => {
            it("should reject name shorter than 3 characters", async () => {
                await testValidation(
                    updateSchema,
                    walletFixtures.invalid.nameTooShort.update,
                    false,
                    /character under the limit/i,
                );
            });

            it("should reject name longer than 50 characters", async () => {
                await testValidation(
                    updateSchema,
                    walletFixtures.invalid.nameTooLong.update,
                    false,
                    /character over the limit/i,
                );
            });

            it("should accept name with exactly 3 characters", async () => {
                const result = await testValidation(
                    updateSchema,
                    walletFixtures.edgeCases.minLengthName.update,
                    true,
                );
                expect(result.name).to.equal("abc");
            });

            it("should accept name with exactly 50 characters", async () => {
                const result = await testValidation(
                    updateSchema,
                    walletFixtures.edgeCases.maxLengthName.update,
                    true,
                );
                expect(result.name).to.have.lengthOf(50);
            });

            it("should accept name with special characters", async () => {
                const result = await testValidation(
                    updateSchema,
                    walletFixtures.edgeCases.nameWithSpecialChars.update,
                    true,
                );
                expect(result.name).to.equal("Wallet #1 (Primary)");
            });

            it("should accept name with Unicode characters", async () => {
                const result = await testValidation(
                    updateSchema,
                    walletFixtures.edgeCases.nameWithUnicode.update,
                    true,
                );
                expect(result.name).to.equal("💰 Savings Wallet");
            });

            it("should accept update without name field", async () => {
                const result = await testValidation(
                    updateSchema,
                    walletFixtures.edgeCases.updateWithoutName.update,
                    true,
                );
                expect(result).to.have.property("id");
                expect(result).to.not.have.property("name");
            });

            it("should accept empty string name (transforms to undefined)", async () => {
                const result = await testValidation(
                    updateSchema,
                    walletFixtures.edgeCases.emptyStringName.update,
                    true,
                );
                expect(result).to.have.property("id");
                expect(result).to.not.have.property("name");
            });

            it("should accept whitespace-only name (transforms to undefined)", async () => {
                const result = await testValidation(
                    updateSchema,
                    walletFixtures.edgeCases.whitespaceOnlyName.update,
                    true,
                );
                expect(result).to.have.property("id");
                expect(result).to.not.have.property("name");
            });
        });

        describe("id field validation", () => {
            it("should accept valid Snowflake ID", async () => {
                const result = await testValidation(
                    updateSchema,
                    {
                        id: "400000000000000001",
                        name: "Test Wallet",
                    },
                    true,
                );
                expect(result.id).to.equal("400000000000000001");
            });

            it("should accept non-string id (transforms to string)", async () => {
                const result = await testValidation(
                    updateSchema,
                    {
                        id: 123456789,
                        name: "Test Wallet",
                    },
                    true,
                );
                expect(result.id).to.equal("123456789");
            });

            it("should reject null id", async () => {
                await testValidation(
                    updateSchema,
                    {
                        id: null,
                        name: "Test Wallet",
                    },
                    false,
                );
            });

            it("should reject undefined id", async () => {
                await testValidation(
                    updateSchema,
                    {
                        name: "Test Wallet",
                    },
                    false,
                    "This field is required",
                );
            });
        });
    });

    describe("batch validation scenarios", () => {
        it("should validate multiple wallet update scenarios", async () => {
            const defaultParams = { omitFields: [] };
            const schema = walletValidation.update(defaultParams);
            
            await testValidationBatch(schema, [
                {
                    data: walletTestDataFactory.updateMinimal(),
                    shouldPass: true,
                    description: "minimal valid update",
                },
                {
                    data: walletTestDataFactory.updateComplete(),
                    shouldPass: true,
                    description: "complete valid update",
                },
                {
                    data: walletTestDataFactory.updateMinimal({ name: "My Wallet" }),
                    shouldPass: true,
                    description: "update with valid name",
                },
                {
                    data: walletTestDataFactory.updateMinimal({ name: "a" }),
                    shouldPass: false,
                    expectedError: /characters under the limit/i,
                    description: "name too short",
                },
                {
                    data: { name: "Valid Name" }, // Missing id
                    shouldPass: false,
                    expectedError: "This field is required",
                    description: "missing required id",
                },
                {
                    data: walletTestDataFactory.updateMinimal({ id: "" }),
                    shouldPass: false,
                    expectedError: /required/i,
                    description: "empty string id",
                },
            ]);
        });
    });

    describe("omitFields functionality", () => {
        it("should omit name field when specified", async () => {
            const schema = walletValidation.update({ omitFields: ["name"] });
            const data = {
                id: "400000000000000001",
                name: "This should be omitted",
            };
            const result = await schema.validate(data, { stripUnknown: true });
            expect(result).to.have.property("id");
            expect(result).to.not.have.property("name");
        });

        it("should omit id field when specified", async () => {
            const schema = walletValidation.update({ omitFields: ["id"] });
            const data = {
                id: "400000000000000001",
                name: "Test Wallet",
            };
            // When id is omitted from the schema, the data validates successfully
            // because the schema no longer requires or validates the id field
            const result = await schema.validate(data, { stripUnknown: true });
            expect(result).to.not.have.property("id");
            expect(result).to.have.property("name");
        });

        it("should omit multiple fields", async () => {
            const schema = walletValidation.update({ omitFields: ["name"] });
            const data = {
                id: "400000000000000001",
                name: "Omitted",
                extraField: "Also stripped",
            };
            const result = await schema.validate(data, { stripUnknown: true });
            expect(result).to.have.property("id");
            expect(result).to.not.have.property("name");
            expect(result).to.not.have.property("extraField");
        });
    });

    describe("test data factory", () => {
        it("should generate valid minimal update data", () => {
            const data = walletTestDataFactory.updateMinimal();
            expect(data).to.have.property("id");
            expect(data.id).to.be.a("string");
        });

        it("should generate valid complete update data", () => {
            const data = walletTestDataFactory.updateComplete();
            expect(data).to.have.property("id");
            expect(data).to.have.property("name");
            expect(data.name).to.be.a("string");
        });

        it("should allow overrides", () => {
            const customName = "Custom Wallet Name";
            const data = walletTestDataFactory.updateComplete({ name: customName });
            expect(data.name).to.equal(customName);
        });

        it("should access test scenarios", () => {
            const invalidData = walletTestDataFactory.forScenario("nameTooShort");
            expect(invalidData.update.name).to.have.lengthOf(2);
        });
    });
});
