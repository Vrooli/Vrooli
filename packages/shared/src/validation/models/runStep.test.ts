import { describe, expect, it } from "vitest";
import { runStepFixtures } from "./__test/fixtures/runStepFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test/validationTestUtils.js";
import { runRoutineStepValidation } from "./runStep.js";

describe("runRoutineStepValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        runRoutineStepValidation,
        runStepFixtures,
        "runStep",
    );

    describe("create specific validations", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = runRoutineStepValidation.create(defaultParams);

        it("should require id in create", async () => {
            const dataWithoutId = { ...runStepFixtures.minimal.create };
            delete dataWithoutId.id;

            await testValidation(
                createSchema,
                dataWithoutId,
                false,
                /required/i,
            );
        });

        it("should require complexity in create", async () => {
            await testValidation(
                createSchema,
                runStepFixtures.invalid.missingComplexity.create,
                false,
                /required/i,
            );
        });

        it("should require name in create", async () => {
            await testValidation(
                createSchema,
                runStepFixtures.invalid.missingName.create,
                false,
                /required/i,
            );
        });

        it("should require nodeId in create", async () => {
            await testValidation(
                createSchema,
                runStepFixtures.invalid.missingNodeId.create,
                false,
                /required/i,
            );
        });

        it("should require order in create", async () => {
            await testValidation(
                createSchema,
                runStepFixtures.invalid.missingOrder.create,
                false,
                /required/i,
            );
        });

        it("should require resourceInId in create", async () => {
            await testValidation(
                createSchema,
                runStepFixtures.invalid.missingResourceInId.create,
                false,
                /required/i,
            );
        });

        it("should require runConnect in create", async () => {
            await testValidation(
                createSchema,
                runStepFixtures.invalid.missingRunConnect.create,
                false,
                /required/i,
            );
        });

        it("should allow minimal create with required fields", async () => {
            await testValidation(
                createSchema,
                runStepFixtures.minimal.create,
                true,
            );
        });

        it("should allow complete create with all fields", async () => {
            await testValidation(
                createSchema,
                runStepFixtures.complete.create,
                true,
            );
        });

        describe("complexity field", () => {
            it("should be required in create", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.invalid.missingComplexity.create,
                    false,
                    /required/i,
                );
            });

            it("should accept zero complexity", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.edgeCases.zeroComplexity.create,
                    true,
                );
            });

            it("should accept high complexity", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.edgeCases.highComplexity.create,
                    true,
                );
            });

            it("should reject negative complexity", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.invalid.negativeComplexity.create,
                    false,
                    /Minimum value is 0/i,
                );
            });
        });

        describe("contextSwitches field", () => {
            it("should be optional", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.minimal.create,
                    true,
                );
            });

            it("should accept single context switch", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.edgeCases.singleContextSwitch.create,
                    true,
                );
            });

            it("should accept many context switches", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.edgeCases.manyContextSwitches.create,
                    true,
                );
            });

            it("should reject zero context switches when provided", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.invalid.zeroContextSwitches.create,
                    false,
                    /Minimum value is 1/i,
                );
            });
        });

        describe("name field", () => {
            it("should be required in create", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.invalid.missingName.create,
                    false,
                    /required/i,
                );
            });

            it("should handle empty name as undefined and fail validation", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.invalid.emptyName.create,
                    false,
                    /required/i,
                );
            });

            it("should accept valid step names", async () => {
                const validNames = ["Initialize", "Process Data", "Validate Results", "Cleanup"];

                for (const name of validNames) {
                    const data = {
                        ...runStepFixtures.minimal.create,
                        name,
                    };

                    await testValidation(createSchema, data, true);
                }
            });
        });

        describe("nodeId field", () => {
            it("should be required in create", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.invalid.missingNodeId.create,
                    false,
                    /required/i,
                );
            });

            it("should accept maximum length nodeId", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.edgeCases.maxLengthNodeId.create,
                    true,
                );
            });

            it("should reject nodeId that exceeds maximum length", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.invalid.longNodeId.create,
                    false,
                    /over the limit/i,
                );
            });

            it("should handle empty nodeId as undefined and fail validation", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.invalid.emptyNodeId.create,
                    false,
                    /required/i,
                );
            });
        });

        describe("order field", () => {
            it("should be required in create", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.invalid.missingOrder.create,
                    false,
                    /required/i,
                );
            });

            it("should accept zero order", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.minimal.create, // order: 0
                    true,
                );
            });

            it("should accept high order numbers", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.edgeCases.highOrderStep.create,
                    true,
                );
            });

            it("should reject negative order", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.invalid.negativeOrder.create,
                    false,
                    /Minimum value is 0/i,
                );
            });
        });

        describe("status field", () => {
            it("should be optional", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.edgeCases.withoutStatus.create,
                    true,
                );
            });

            it("should accept valid status values", async () => {
                const validStatuses = ["Completed", "InProgress", "Skipped"];

                for (const status of validStatuses) {
                    const data = {
                        ...runStepFixtures.minimal.create,
                        status,
                    };

                    await testValidation(createSchema, data, true);
                }
            });

            it("should reject invalid status values", async () => {
                const dataWithInvalidStatus = {
                    ...runStepFixtures.minimal.create,
                    status: "InvalidStatus",
                };

                await testValidation(
                    createSchema,
                    dataWithInvalidStatus,
                    false,
                    /must be one of/i,
                );
            });
        });

        describe("timeElapsed field", () => {
            it("should be optional", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.minimal.create,
                    true,
                );
            });

            it("should accept quick execution times", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.edgeCases.quickStep.create,
                    true,
                );
            });

            it("should accept long execution times", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.edgeCases.longRunningStep.create,
                    true,
                );
            });

            it("should reject negative time elapsed", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.invalid.negativeTimeElapsed.create,
                    false,
                    /Minimum value is 0/i,
                );
            });
        });

        describe("relationship fields", () => {
            it("should require runConnect", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.invalid.missingRunConnect.create,
                    false,
                    /required/i,
                );
            });

            it("should allow optional resourceVersionConnect", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.edgeCases.withResourceVersion.create,
                    true,
                );
            });

            it("should work without resourceVersionConnect", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.minimal.create,
                    true,
                );
            });
        });

        describe("step execution scenarios", () => {
            it("should handle different step complexities", async () => {
                const scenarios = [
                    {
                        data: runStepFixtures.edgeCases.zeroComplexity.create,
                        shouldPass: true,
                        description: "zero complexity step",
                    },
                    {
                        data: runStepFixtures.minimal.create,
                        shouldPass: true,
                        description: "low complexity step",
                    },
                    {
                        data: runStepFixtures.edgeCases.highComplexity.create,
                        shouldPass: true,
                        description: "high complexity step",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should handle different step statuses", async () => {
                const scenarios = [
                    {
                        data: runStepFixtures.edgeCases.inProgressStatus.create,
                        shouldPass: true,
                        description: "in progress step",
                    },
                    {
                        data: runStepFixtures.edgeCases.completedStatus.create,
                        shouldPass: true,
                        description: "completed step",
                    },
                    {
                        data: runStepFixtures.edgeCases.skippedStatus.create,
                        shouldPass: true,
                        description: "skipped step",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should handle complex step creation", async () => {
                await testValidation(
                    createSchema,
                    runStepFixtures.complete.create,
                    true,
                );
            });
        });
    });

    describe("update specific validations", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = runRoutineStepValidation.update(defaultParams);

        it("should require id in update", async () => {
            await testValidation(
                updateSchema,
                runStepFixtures.invalid.missingRequired.update,
                false,
                /required/i,
            );
        });

        it("should allow minimal updates with just id", async () => {
            await testValidation(
                updateSchema,
                runStepFixtures.minimal.update,
                true,
            );
        });

        it("should allow complete updates", async () => {
            await testValidation(
                updateSchema,
                runStepFixtures.complete.update,
                true,
            );
        });

        describe("field updates", () => {
            it("should allow updating contextSwitches", async () => {
                await testValidation(
                    updateSchema,
                    runStepFixtures.edgeCases.updateContextSwitches.update,
                    true,
                );
            });

            it("should allow updating status", async () => {
                await testValidation(
                    updateSchema,
                    runStepFixtures.edgeCases.updateStatus.update,
                    true,
                );
            });

            it("should allow updating timeElapsed", async () => {
                await testValidation(
                    updateSchema,
                    runStepFixtures.edgeCases.updateTimeElapsed.update,
                    true,
                );
            });

            it("should allow updating all optional fields", async () => {
                await testValidation(
                    updateSchema,
                    runStepFixtures.edgeCases.updateAllFields.update,
                    true,
                );
            });

            it("should not allow updating complexity", async () => {
                const dataWithRestrictedFields = {
                    id: "123456789012345678",
                    complexity: 50,
                    status: "Completed",
                };

                const result = await testValidation(updateSchema, dataWithRestrictedFields, true);
                expect(result).to.not.have.property("complexity");
            });

            it("should not allow updating name", async () => {
                const dataWithRestrictedFields = {
                    id: "123456789012345678",
                    name: "Updated Step Name",
                    status: "Completed",
                };

                const result = await testValidation(updateSchema, dataWithRestrictedFields, true);
                expect(result).to.not.have.property("name");
            });

            it("should not allow updating nodeId", async () => {
                const dataWithRestrictedFields = {
                    id: "123456789012345678",
                    nodeId: "new_node_id",
                    status: "Completed",
                };

                const result = await testValidation(updateSchema, dataWithRestrictedFields, true);
                expect(result).to.not.have.property("nodeId");
            });

            it("should not allow updating order", async () => {
                const dataWithRestrictedFields = {
                    id: "123456789012345678",
                    order: 5,
                    status: "Completed",
                };

                const result = await testValidation(updateSchema, dataWithRestrictedFields, true);
                expect(result).to.not.have.property("order");
            });

            it("should not allow updating resourceInId", async () => {
                const dataWithRestrictedFields = {
                    id: "123456789012345678",
                    resourceInId: "123456789012345680",
                    status: "Completed",
                };

                const result = await testValidation(updateSchema, dataWithRestrictedFields, true);
                expect(result).to.not.have.property("resourceInId");
            });

            it("should not allow updating relationship connections", async () => {
                const dataWithRestrictedFields = {
                    id: "123456789012345678",
                    runConnect: "123456789012345679",
                    resourceVersionConnect: "123456789012345680",
                    status: "Completed",
                };

                const result = await testValidation(updateSchema, dataWithRestrictedFields, true);
                expect(result).to.not.have.property("runConnect");
                expect(result).to.not.have.property("resourceVersionConnect");
            });
        });

        describe("validation in updates", () => {
            it("should still validate contextSwitches minimum in update", async () => {
                const dataWithInvalidContextSwitches = {
                    id: "123456789012345678",
                    contextSwitches: 0, // Should be minimum 1
                };

                await testValidation(
                    updateSchema,
                    dataWithInvalidContextSwitches,
                    false,
                    /Minimum value is 1/i,
                );
            });

            it("should still validate status values in update", async () => {
                const dataWithInvalidStatus = {
                    id: "123456789012345678",
                    status: "InvalidStatus",
                };

                await testValidation(
                    updateSchema,
                    dataWithInvalidStatus,
                    false,
                    /must be one of/i,
                );
            });

            it("should still validate timeElapsed minimum in update", async () => {
                const dataWithNegativeTime = {
                    id: "123456789012345678",
                    timeElapsed: -100,
                };

                await testValidation(
                    updateSchema,
                    dataWithNegativeTime,
                    false,
                    /Minimum value is 0/i,
                );
            });
        });

        describe("update scenarios", () => {
            it("should handle id-only updates", async () => {
                await testValidation(
                    updateSchema,
                    runStepFixtures.edgeCases.updateOnlyId.update,
                    true,
                );
            });

            it("should handle single field updates", async () => {
                const scenarios = [
                    {
                        data: runStepFixtures.edgeCases.updateContextSwitches.update,
                        shouldPass: true,
                        description: "context switches update",
                    },
                    {
                        data: runStepFixtures.edgeCases.updateStatus.update,
                        shouldPass: true,
                        description: "status update",
                    },
                    {
                        data: runStepFixtures.edgeCases.updateTimeElapsed.update,
                        shouldPass: true,
                        description: "time elapsed update",
                    },
                ];

                await testValidationBatch(updateSchema, scenarios);
            });

            it("should handle comprehensive updates", async () => {
                await testValidation(
                    updateSchema,
                    runStepFixtures.edgeCases.updateAllFields.update,
                    true,
                );
            });
        });
    });

    describe("id validation", () => {
        const createSchema = runRoutineStepValidation.create({ omitFields: [] });


        it("should accept valid Snowflake IDs", async () => {
            const validIds = [
                "123456789012345678",
                "999999999999999999",
                "100000000000000000",
            ];

            const scenarios = validIds.map(id => ({
                data: {
                    ...runStepFixtures.minimal.create,
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
                    { ...runStepFixtures.minimal.create, id },
                    false,
                    error,
                );
            }
        });
    });

    describe("type conversions", () => {
        const createSchema = runRoutineStepValidation.create({ omitFields: [] });

        it("should handle ID conversion", async () => {
            const dataWithNumberId = {
                id: 123456789012345, // Smaller number to avoid precision issues
                complexity: 1,
                name: "Test Step",
                nodeId: "test_node",
                order: 0,
                resourceInId: "123456789012345679",
                runConnect: "123456789012345680",
            };

            const result = await testValidation(
                createSchema,
                dataWithNumberId,
                true,
            );
            expect(result.id).to.be.a("string");
            expect(result.id).toBe("123456789012345");
        });

        it("should handle number field conversions", async () => {
            const dataWithStringNumbers = {
                id: "123456789012345678",
                complexity: "50", // String should convert to number
                contextSwitches: "3", // String should convert to number
                name: "Test Step",
                nodeId: "test_node",
                order: "2", // String should convert to number
                resourceInId: "123456789012345679",
                timeElapsed: "1500", // String should convert to number
                runConnect: "123456789012345680",
            };

            const result = await testValidation(
                createSchema,
                dataWithStringNumbers,
                true,
            );
            expect(result.complexity).to.be.a("number");
            expect(result.contextSwitches).to.be.a("number");
            expect(result.order).to.be.a("number");
            expect(result.timeElapsed).to.be.a("number");
            expect(result.complexity).toBe(50);
            expect(result.contextSwitches).toBe(3);
            expect(result.order).toBe(2);
            expect(result.timeElapsed).toBe(1500);
        });
    });

    describe("edge cases", () => {
        const createSchema = runRoutineStepValidation.create({ omitFields: [] });
        const updateSchema = runRoutineStepValidation.update({ omitFields: [] });

        it("should handle minimal step creation", async () => {
            await testValidation(
                createSchema,
                runStepFixtures.minimal.create,
                true,
            );
        });

        it("should handle minimal update operations", async () => {
            await testValidation(
                updateSchema,
                runStepFixtures.minimal.update,
                true,
            );
        });

        it("should handle different complexity levels", async () => {
            const scenarios = [
                {
                    data: runStepFixtures.edgeCases.zeroComplexity.create,
                    shouldPass: true,
                    description: "zero complexity",
                },
                {
                    data: runStepFixtures.minimal.create,
                    shouldPass: true,
                    description: "low complexity",
                },
                {
                    data: runStepFixtures.edgeCases.highComplexity.create,
                    shouldPass: true,
                    description: "high complexity",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle different execution patterns", async () => {
            const scenarios = [
                {
                    data: runStepFixtures.edgeCases.quickStep.create,
                    shouldPass: true,
                    description: "quick execution",
                },
                {
                    data: runStepFixtures.edgeCases.longRunningStep.create,
                    shouldPass: true,
                    description: "long running step",
                },
                {
                    data: runStepFixtures.edgeCases.singleContextSwitch.create,
                    shouldPass: true,
                    description: "single context switch",
                },
                {
                    data: runStepFixtures.edgeCases.manyContextSwitches.create,
                    shouldPass: true,
                    description: "many context switches",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle different step configurations", async () => {
            const scenarios = [
                {
                    data: runStepFixtures.edgeCases.withoutStatus.create,
                    shouldPass: true,
                    description: "step without status",
                },
                {
                    data: runStepFixtures.edgeCases.withResourceVersion.create,
                    shouldPass: true,
                    description: "step with resource version",
                },
                {
                    data: runStepFixtures.edgeCases.maxLengthNodeId.create,
                    shouldPass: true,
                    description: "maximum length node ID",
                },
                {
                    data: runStepFixtures.edgeCases.highOrderStep.create,
                    shouldPass: true,
                    description: "high order step",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle complex step operations", async () => {
            const createScenarios = [
                {
                    data: runStepFixtures.complete.create,
                    shouldPass: true,
                    description: "complete step creation",
                },
            ];

            const updateScenarios = [
                {
                    data: runStepFixtures.complete.update,
                    shouldPass: true,
                    description: "complete step update",
                },
            ];

            await testValidationBatch(createSchema, createScenarios);
            await testValidationBatch(updateSchema, updateScenarios);
        });

        it("should handle all valid status transitions", async () => {
            const statuses = ["Completed", "InProgress", "Skipped"];

            for (const status of statuses) {
                const data = {
                    id: "123456789012345678",
                    status,
                };

                await testValidation(updateSchema, data, true);
            }
        });
    });
});
