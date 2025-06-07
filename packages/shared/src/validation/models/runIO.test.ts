import { describe, it, expect } from "vitest";
import { runIOValidation } from "./runIO.js";
import { runIOFixtures } from "./__test__/fixtures/runIOFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test__/validationTestUtils.js";

describe("runIOValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        runIOValidation,
        runIOFixtures,
        "runIO",
    );

    describe("create specific validations", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = runIOValidation.create(defaultParams);

        it("should require id in create", async () => {
            const dataWithoutId = { ...runIOFixtures.minimal.create };
            delete dataWithoutId.id;

            await testValidation(
                createSchema,
                dataWithoutId,
                false,
                /required/i,
            );
        });

        it("should require data in create", async () => {
            await testValidation(
                createSchema,
                runIOFixtures.invalid.missingData.create,
                false,
                /required/i,
            );
        });

        it("should require nodeInputName in create", async () => {
            await testValidation(
                createSchema,
                runIOFixtures.invalid.missingNodeInputName.create,
                false,
                /required/i,
            );
        });

        it("should require nodeName in create", async () => {
            await testValidation(
                createSchema,
                runIOFixtures.invalid.missingNodeName.create,
                false,
                /required/i,
            );
        });

        it("should require runConnect in create", async () => {
            await testValidation(
                createSchema,
                runIOFixtures.invalid.missingRunConnect.create,
                false,
                /required/i,
            );
        });

        it("should allow minimal create with required fields", async () => {
            await testValidation(
                createSchema,
                runIOFixtures.minimal.create,
                true,
            );
        });

        it("should allow complete create with all fields", async () => {
            await testValidation(
                createSchema,
                runIOFixtures.complete.create,
                true,
            );
        });

        describe("data field", () => {
            it("should be required in create", async () => {
                await testValidation(
                    createSchema,
                    runIOFixtures.invalid.missingData.create,
                    false,
                    /required/i,
                );
            });

            it("should accept simple string data", async () => {
                await testValidation(
                    createSchema,
                    runIOFixtures.edgeCases.simpleInput.create,
                    true,
                );
            });

            it("should accept JSON string data", async () => {
                await testValidation(
                    createSchema,
                    runIOFixtures.edgeCases.jsonInput.create,
                    true,
                );
            });

            it("should accept XML string data", async () => {
                await testValidation(
                    createSchema,
                    runIOFixtures.edgeCases.xmlInput.create,
                    true,
                );
            });

            it("should accept binary/base64 data", async () => {
                await testValidation(
                    createSchema,
                    runIOFixtures.edgeCases.binaryData.create,
                    true,
                );
            });

            it("should accept maximum length data", async () => {
                await testValidation(
                    createSchema,
                    runIOFixtures.edgeCases.maxLengthData.create,
                    true,
                );
            });

            it("should reject data that exceeds maximum length", async () => {
                await testValidation(
                    createSchema,
                    runIOFixtures.invalid.longData.create,
                    false,
                    /over the limit/i,
                );
            });

            it("should handle empty data as undefined and fail validation", async () => {
                await testValidation(
                    createSchema,
                    runIOFixtures.invalid.emptyData.create,
                    false,
                    /required/i,
                );
            });

            it("should accept multiline data", async () => {
                await testValidation(
                    createSchema,
                    runIOFixtures.edgeCases.multilineData.create,
                    true,
                );
            });

            it("should accept special characters", async () => {
                await testValidation(
                    createSchema,
                    runIOFixtures.edgeCases.specialCharacters.create,
                    true,
                );
            });

            it("should accept unicode data", async () => {
                await testValidation(
                    createSchema,
                    runIOFixtures.edgeCases.unicodeData.create,
                    true,
                );
            });
        });

        describe("nodeInputName field", () => {
            it("should be required in create", async () => {
                await testValidation(
                    createSchema,
                    runIOFixtures.invalid.missingNodeInputName.create,
                    false,
                    /required/i,
                );
            });

            it("should accept valid input names", async () => {
                const validNames = ["input1", "dataInput", "textInput", "fileInput"];

                for (const nodeInputName of validNames) {
                    const data = {
                        ...runIOFixtures.minimal.create,
                        nodeInputName,
                    };

                    await testValidation(createSchema, data, true);
                }
            });

            it("should accept maximum length input name", async () => {
                await testValidation(
                    createSchema,
                    runIOFixtures.edgeCases.maxLengthNames.create,
                    true,
                );
            });

            it("should reject input name that exceeds maximum length", async () => {
                await testValidation(
                    createSchema,
                    runIOFixtures.invalid.longNodeInputName.create,
                    false,
                    /over the limit/i,
                );
            });

            it("should handle empty input name as undefined and fail validation", async () => {
                await testValidation(
                    createSchema,
                    runIOFixtures.invalid.emptyNodeInputName.create,
                    false,
                    /required/i,
                );
            });
        });

        describe("nodeName field", () => {
            it("should be required in create", async () => {
                await testValidation(
                    createSchema,
                    runIOFixtures.invalid.missingNodeName.create,
                    false,
                    /required/i,
                );
            });

            it("should accept valid node names", async () => {
                const validNames = ["ProcessorNode", "DataNode", "TransformNode", "OutputNode"];

                for (const nodeName of validNames) {
                    const data = {
                        ...runIOFixtures.minimal.create,
                        nodeName,
                    };

                    await testValidation(createSchema, data, true);
                }
            });

            it("should accept maximum length node name", async () => {
                await testValidation(
                    createSchema,
                    runIOFixtures.edgeCases.maxLengthNames.create,
                    true,
                );
            });

            it("should reject node name that exceeds maximum length", async () => {
                await testValidation(
                    createSchema,
                    runIOFixtures.invalid.longNodeName.create,
                    false,
                    /over the limit/i,
                );
            });

            it("should handle empty node name as undefined and fail validation", async () => {
                await testValidation(
                    createSchema,
                    runIOFixtures.invalid.emptyNodeName.create,
                    false,
                    /required/i,
                );
            });
        });

        describe("relationship fields", () => {
            it("should require runConnect", async () => {
                await testValidation(
                    createSchema,
                    runIOFixtures.invalid.missingRunConnect.create,
                    false,
                    /required/i,
                );
            });

            it("should accept valid run connection", async () => {
                await testValidation(
                    createSchema,
                    runIOFixtures.minimal.create,
                    true,
                );
            });
        });

        describe("data type scenarios", () => {
            it("should handle different data formats", async () => {
                const scenarios = [
                    {
                        data: runIOFixtures.edgeCases.simpleInput.create,
                        shouldPass: true,
                        description: "simple string input",
                    },
                    {
                        data: runIOFixtures.edgeCases.jsonInput.create,
                        shouldPass: true,
                        description: "JSON string input",
                    },
                    {
                        data: runIOFixtures.edgeCases.xmlInput.create,
                        shouldPass: true,
                        description: "XML string input",
                    },
                    {
                        data: runIOFixtures.edgeCases.numberAsString.create,
                        shouldPass: true,
                        description: "number as string input",
                    },
                    {
                        data: runIOFixtures.edgeCases.booleanAsString.create,
                        shouldPass: true,
                        description: "boolean as string input",
                    },
                    {
                        data: runIOFixtures.edgeCases.arrayAsString.create,
                        shouldPass: true,
                        description: "array as JSON string input",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should handle complex input/output creation", async () => {
                await testValidation(
                    createSchema,
                    runIOFixtures.complete.create,
                    true,
                );
            });
        });
    });

    describe("update specific validations", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = runIOValidation.update(defaultParams);

        it("should require id in update", async () => {
            await testValidation(
                updateSchema,
                runIOFixtures.invalid.missingRequired.update,
                false,
                /required/i,
            );
        });

        it("should require data in update", async () => {
            const dataWithoutData = { id: "123456789012345678" };

            await testValidation(
                updateSchema,
                dataWithoutData,
                false,
                /required/i,
            );
        });

        it("should allow minimal updates with required fields", async () => {
            await testValidation(
                updateSchema,
                runIOFixtures.minimal.update,
                true,
            );
        });

        it("should allow complete updates", async () => {
            await testValidation(
                updateSchema,
                runIOFixtures.complete.update,
                true,
            );
        });

        describe("field updates", () => {
            it("should allow updating data", async () => {
                await testValidation(
                    updateSchema,
                    runIOFixtures.edgeCases.updateDataOnly.update,
                    true,
                );
            });

            it("should allow updating to complex data", async () => {
                await testValidation(
                    updateSchema,
                    runIOFixtures.edgeCases.updateComplexData.update,
                    true,
                );
            });

            it("should allow updating to maximum length data", async () => {
                await testValidation(
                    updateSchema,
                    runIOFixtures.edgeCases.updateMaxLengthData.update,
                    true,
                );
            });

            it("should not allow updating nodeInputName", async () => {
                const dataWithRestrictedFields = {
                    id: "123456789012345678",
                    data: "updated data",
                    nodeInputName: "newInput",
                };

                const result = await testValidation(updateSchema, dataWithRestrictedFields, true);
                expect(result).to.not.have.property("nodeInputName");
            });

            it("should not allow updating nodeName", async () => {
                const dataWithRestrictedFields = {
                    id: "123456789012345678",
                    data: "updated data",
                    nodeName: "NewNode",
                };

                const result = await testValidation(updateSchema, dataWithRestrictedFields, true);
                expect(result).to.not.have.property("nodeName");
            });

            it("should not allow updating runConnect", async () => {
                const dataWithRestrictedFields = {
                    id: "123456789012345678",
                    data: "updated data",
                    runConnect: "123456789012345679",
                };

                const result = await testValidation(updateSchema, dataWithRestrictedFields, true);
                expect(result).to.not.have.property("runConnect");
            });
        });

        describe("data validation in updates", () => {
            it("should still validate data length in update", async () => {
                const dataWithLongData = {
                    id: "123456789012345678",
                    data: "x".repeat(8193), // Too long
                };

                await testValidation(
                    updateSchema,
                    dataWithLongData,
                    false,
                    /over the limit/i,
                );
            });

            it("should handle empty data in update as required field", async () => {
                await testValidation(
                    updateSchema,
                    runIOFixtures.edgeCases.updateEmptyData.update,
                    false,
                    /required/i,
                );
            });
        });

        describe("update scenarios", () => {
            it("should handle simple data updates", async () => {
                await testValidation(
                    updateSchema,
                    runIOFixtures.edgeCases.updateDataOnly.update,
                    true,
                );
            });

            it("should handle complex data structure updates", async () => {
                const scenarios = [
                    {
                        data: runIOFixtures.edgeCases.updateDataOnly.update,
                        shouldPass: true,
                        description: "simple data update",
                    },
                    {
                        data: runIOFixtures.edgeCases.updateComplexData.update,
                        shouldPass: true,
                        description: "complex JSON data update",
                    },
                    {
                        data: runIOFixtures.edgeCases.updateMaxLengthData.update,
                        shouldPass: true,
                        description: "maximum length data update",
                    },
                ];

                await testValidationBatch(updateSchema, scenarios);
            });
        });
    });

    describe("id validation", () => {
        const createSchema = runIOValidation.create({ omitFields: [] });


        it("should accept valid Snowflake IDs", async () => {
            const validIds = [
                "123456789012345678",
                "999999999999999999",
                "100000000000000000",
            ];

            const scenarios = validIds.map(id => ({
                data: {
                    ...runIOFixtures.minimal.create,
                    id,
                },
                shouldPass: true,
                description: `valid ID: ${id}`,
            }));

            await testValidationBatch(createSchema, scenarios);
        });

        it("should reject invalid IDs", async () => {
            const invalidIds = [
                { id: "not-a-number", error: /Must be a valid ID/i },
                { id: "abc123", error: /Must be a valid ID/i },
                { id: "", error: /required/i },
                { id: "-123", error: /Must be a valid ID/i },
                { id: "0", error: /Must be a valid ID/i },
            ];

            for (const { id, error } of invalidIds) {
                await testValidation(
                    createSchema,
                    { ...runIOFixtures.minimal.create, id },
                    false,
                    error,
                );
            }
        });
    });

    describe("type conversions", () => {
        const createSchema = runIOValidation.create({ omitFields: [] });

        it("should handle ID conversion", async () => {
            const dataWithNumberId = {
                id: 123456789012345, // Smaller number to avoid precision issues
                data: "test data",
                nodeInputName: "input1",
                nodeName: "TestNode",
                runConnect: "123456789012345679",
            };

            const result = await testValidation(
                createSchema,
                dataWithNumberId,
                true,
            );
            expect(result.id).to.be.a("string");
            expect(result.id).to.equal("123456789012345");
        });

        it("should handle string field conversions", async () => {
            const dataWithNumbers = {
                id: "123456789012345678",
                data: 12345, // Number should convert to string
                nodeInputName: "input1",
                nodeName: "TestNode",
                runConnect: "123456789012345679",
            };

            const result = await testValidation(
                createSchema,
                dataWithNumbers,
                true,
            );
            expect(result.data).to.be.a("string");
            expect(result.data).to.equal("12345");
        });
    });

    describe("edge cases", () => {
        const createSchema = runIOValidation.create({ omitFields: [] });
        const updateSchema = runIOValidation.update({ omitFields: [] });

        it("should handle minimal IO creation", async () => {
            await testValidation(
                createSchema,
                runIOFixtures.minimal.create,
                true,
            );
        });

        it("should handle minimal update operations", async () => {
            await testValidation(
                updateSchema,
                runIOFixtures.minimal.update,
                true,
            );
        });

        it("should handle various data formats", async () => {
            const scenarios = [
                {
                    data: runIOFixtures.edgeCases.binaryData.create,
                    shouldPass: true,
                    description: "binary/base64 data",
                },
                {
                    data: runIOFixtures.edgeCases.multilineData.create,
                    shouldPass: true,
                    description: "multiline text data",
                },
                {
                    data: runIOFixtures.edgeCases.specialCharacters.create,
                    shouldPass: true,
                    description: "special characters data",
                },
                {
                    data: runIOFixtures.edgeCases.unicodeData.create,
                    shouldPass: true,
                    description: "unicode data",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle maximum length constraints", async () => {
            const scenarios = [
                {
                    data: runIOFixtures.edgeCases.maxLengthData.create,
                    shouldPass: true,
                    description: "maximum length data",
                },
                {
                    data: runIOFixtures.edgeCases.maxLengthNames.create,
                    shouldPass: true,
                    description: "maximum length names",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle complex IO operations", async () => {
            const createScenarios = [
                {
                    data: runIOFixtures.complete.create,
                    shouldPass: true,
                    description: "complete IO creation",
                },
            ];

            const updateScenarios = [
                {
                    data: runIOFixtures.complete.update,
                    shouldPass: true,
                    description: "complete IO update",
                },
            ];

            await testValidationBatch(createSchema, createScenarios);
            await testValidationBatch(updateSchema, updateScenarios);
        });

        it("should handle different data type representations", async () => {
            const scenarios = [
                {
                    data: runIOFixtures.edgeCases.numberAsString.create,
                    shouldPass: true,
                    description: "number as string",
                },
                {
                    data: runIOFixtures.edgeCases.booleanAsString.create,
                    shouldPass: true,
                    description: "boolean as string",
                },
                {
                    data: runIOFixtures.edgeCases.arrayAsString.create,
                    shouldPass: true,
                    description: "array as JSON string",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });
    });
});
