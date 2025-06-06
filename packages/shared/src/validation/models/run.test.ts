import { expect } from "chai";
import { RunStatus, RunStepStatus } from "../../api/types.js";
import { runValidation } from "./run.js";
import { runFixtures, runTestDataFactory } from "./__test__/fixtures/runFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test__/validationTestUtils.js";

describe("runValidation", () => {
    // Run standard test suite
    runStandardValidationTests(runValidation, runFixtures, "run");

    // Additional run-specific tests
    describe("run-specific validation", () => {
        const defaultParams = { omitFields: [] };

        describe("create validation", () => {
            const createSchema = runValidation.create(defaultParams);

            it("should accept all valid RunStatus enum values", async () => {
                for (const statusCase of runFixtures.edgeCases.allStatuses) {
                    const result = await testValidation(createSchema, statusCase.create, true);
                    expect(result.status).to.be.oneOf(Object.values(RunStatus));
                }
            });

            it("should reject invalid status values", async () => {
                await testValidation(
                    createSchema,
                    runFixtures.invalid.invalidStatus.create,
                    false,
                    /must be one of the following values/i,
                );
            });

            it("should trim and handle empty data strings", async () => {
                const result = await testValidation(
                    createSchema,
                    runFixtures.edgeCases.emptyData.create,
                    true,
                );
                expect(result.data).to.be.undefined;
            });

            it("should accept data at max length", async () => {
                const result = await testValidation(
                    createSchema,
                    runFixtures.edgeCases.maxData.create,
                    true,
                );
                expect(result.data).to.have.lengthOf(16384);
            });

            it("should reject data over max length", async () => {
                await testValidation(
                    createSchema,
                    runFixtures.invalid.tooLongData.create,
                    false,
                    /character over the limit/i,
                );
            });

            it("should reject names over max length", async () => {
                await testValidation(
                    createSchema,
                    runFixtures.invalid.tooLongName.create,
                    false,
                    /characters over the limit/i,
                );
            });

            it("should accept zero values for numeric fields", async () => {
                const result = await testValidation(
                    createSchema,
                    runFixtures.edgeCases.zeroValues.create,
                    true,
                );
                expect(result.completedComplexity).to.equal(0);
                expect(result.contextSwitches).to.equal(0);
                expect(result.timeElapsed).to.equal(0);
            });

            it("should reject negative values for numeric fields", async () => {
                await testValidation(
                    createSchema,
                    {
                        ...runFixtures.minimal.create,
                        completedComplexity: -1,
                    },
                    false,
                    /minimum value is 0/i,
                );
            });

            it("should require resourceVersionConnect relationship", async () => {
                const { resourceVersionConnect, ...dataWithoutResource } = runFixtures.minimal.create;
                await testValidation(
                    createSchema,
                    dataWithoutResource,
                    false,
                    /required/i,
                );
            });

            it("should validate nested IO creation", async () => {
                const dataWithIO = {
                    ...runFixtures.minimal.create,
                    ioCreate: [
                        {
                            id: "123456789012345678",
                            data: JSON.stringify({ test: "data" }),
                            nodeInputName: "input1",
                            nodeName: "testNode",
                            runConnect: runFixtures.minimal.create.id,
                        },
                    ],
                };
                const result = await testValidation(createSchema, dataWithIO, true);
                expect(result.ioCreate).to.be.an("array").with.lengthOf(1);
                expect(result.ioCreate[0]).to.have.property("id");
            });

            it("should validate nested schedule creation", async () => {
                const dataWithSchedule = {
                    ...runFixtures.minimal.create,
                    scheduleCreate: {
                        id: "123456789012345678",
                        startTime: new Date().toISOString(),
                        endTime: new Date(Date.now() + 86400000).toISOString(),
                        timezone: "UTC",
                    },
                };
                const result = await testValidation(createSchema, dataWithSchedule, true);
                expect(result.scheduleCreate).to.be.an("object");
                expect(result.scheduleCreate).to.have.property("id");
            });

            it("should validate nested steps creation", async () => {
                const dataWithSteps = {
                    ...runFixtures.minimal.create,
                    stepsCreate: [
                        {
                            id: "123456789012345678",
                            complexity: 1,
                            name: "Test Step",
                            nodeId: "testNode",
                            order: 0,
                            resourceInId: "223456789012345678",
                            status: RunStepStatus.InProgress,
                            runConnect: runFixtures.minimal.create.id,
                        },
                    ],
                };
                const result = await testValidation(createSchema, dataWithSteps, true);
                expect(result.stepsCreate).to.be.an("array").with.lengthOf(1);
                expect(result.stepsCreate[0]).to.have.property("name");
            });

            it("should validate team connection", async () => {
                const dataWithTeam = {
                    ...runFixtures.minimal.create,
                    teamConnect: "123456789012345678",
                };
                const result = await testValidation(createSchema, dataWithTeam, true);
                expect(result.teamConnect).to.equal("123456789012345678");
            });
        });

        describe("update validation", () => {
            const updateSchema = runValidation.update(defaultParams);

            it("should allow updating isStarted field", async () => {
                const result = await testValidation(
                    updateSchema,
                    {
                        id: "123456789012345678",
                        isStarted: true,
                    },
                    true,
                );
                expect(result.isStarted).to.equal(true);
            });

            it("should not allow updating status field", async () => {
                const dataWithStatus = {
                    id: "123456789012345678",
                    status: RunStatus.Completed,
                };
                const result = await updateSchema.validate(dataWithStatus, { stripUnknown: true });
                expect(result).to.not.have.property("status");
            });

            it("should validate nested IO operations", async () => {
                const dataWithIOOperations = {
                    id: "123456789012345678",
                    ioCreate: [
                        {
                            id: "223456789012345678",
                            data: JSON.stringify({ new: "data" }),
                            nodeInputName: "newInput",
                            nodeName: "newNode",
                            runConnect: "123456789012345678",
                        },
                    ],
                    ioUpdate: [
                        {
                            id: "323456789012345678",
                            data: JSON.stringify({ updated: "data" }),
                        },
                    ],
                    ioDelete: ["423456789012345678"],
                };
                const result = await testValidation(updateSchema, dataWithIOOperations, true);
                expect(result.ioCreate).to.be.an("array");
                expect(result.ioUpdate).to.be.an("array");
                expect(result.ioDelete).to.be.an("array");
            });

            it("should validate nested steps operations", async () => {
                const dataWithStepsOperations = {
                    id: "123456789012345678",
                    stepsCreate: [
                        {
                            id: "523456789012345678",
                            complexity: 1,
                            name: "New Step",
                            nodeId: "newNode",
                            order: 2,
                            resourceInId: "623456789012345678",
                            status: RunStepStatus.InProgress,
                            runConnect: "123456789012345678",
                        },
                    ],
                    stepsUpdate: [
                        {
                            id: "623456789012345678",
                            status: RunStepStatus.Completed,
                        },
                    ],
                    stepsDelete: ["723456789012345678"],
                };
                const result = await testValidation(updateSchema, dataWithStepsOperations, true);
                expect(result.stepsCreate).to.be.an("array");
                expect(result.stepsUpdate).to.be.an("array");
                expect(result.stepsDelete).to.be.an("array");
            });

            it("should validate schedule update operations", async () => {
                // Test schedule update with proper time fields to avoid cross-field validation issues
                const now = new Date();
                const later = new Date(now.getTime() + 2000); // 2 seconds later
                const dataWithScheduleUpdate = {
                    id: "123456789012345678",
                    scheduleUpdate: {
                        id: "823456789012345678",
                        timezone: "America/New_York",
                        startTime: now.toISOString(),
                        endTime: later.toISOString(),
                    },
                };
                const result = await testValidation(updateSchema, dataWithScheduleUpdate, true);
                expect(result.scheduleUpdate).to.be.an("object");
                expect(result.scheduleUpdate.timezone).to.equal("America/New_York");
            });
        });

        describe("omitFields functionality", () => {
            it("should omit specified relationship fields", async () => {
                const schema = runValidation.create({
                    omitFields: ["ioCreate", "stepsCreate", "teamConnect"],
                });
                
                const result = await schema.validate(runFixtures.complete.create, { 
                    stripUnknown: true 
                });
                expect(result).to.not.have.property("ioCreate");
                expect(result).to.not.have.property("stepsCreate");
                expect(result).to.not.have.property("teamConnect");
                expect(result).to.have.property("scheduleCreate"); // Should still have this
            });

            it("should omit specified data fields", async () => {
                const schema = runValidation.update({
                    omitFields: ["data", "timeElapsed"],
                });
                
                const result = await schema.validate({
                    id: "123456789012345678",
                    data: "should be omitted",
                    timeElapsed: 3600,
                    completedComplexity: 5,
                }, { stripUnknown: true });
                
                expect(result).to.not.have.property("data");
                expect(result).to.not.have.property("timeElapsed");
                expect(result).to.have.property("completedComplexity");
            });
        });
    });

    describe("batch validation scenarios", () => {
        it("should validate multiple run scenarios", async () => {
            const defaultParams = { omitFields: [] };
            const schema = runValidation.create(defaultParams);
            await testValidationBatch(schema, [
                {
                    data: runTestDataFactory.createMinimal(),
                    shouldPass: true,
                    description: "minimal valid run",
                },
                {
                    data: runTestDataFactory.createComplete(),
                    shouldPass: true,
                    description: "complete valid run",
                },
                {
                    data: runFixtures.invalid.tooLongData.create,
                    shouldPass: false,
                    expectedError: /character over the limit/i,
                    description: "run with data exceeding max length",
                },
                {
                    data: runFixtures.invalid.invalidId.create,
                    shouldPass: false,
                    description: "invalid Snowflake ID format",
                },
                {
                    data: runFixtures.invalid.invalidStatus.create,
                    shouldPass: false,
                    expectedError: /must be one of the following values/i,
                    description: "invalid status enum value",
                },
            ]);
        });

        it("should validate update scenarios", async () => {
            const defaultParams = { omitFields: [] };
            const schema = runValidation.update(defaultParams);
            await testValidationBatch(schema, [
                {
                    data: runTestDataFactory.updateMinimal(),
                    shouldPass: true,
                    description: "minimal valid update",
                },
                {
                    data: runFixtures.complete.update,
                    shouldPass: true,
                    description: "complete valid update",
                },
                {
                    data: runFixtures.invalid.missingRequired.update,
                    shouldPass: false,
                    expectedError: /required/i,
                    description: "missing required id field",
                },
            ]);
        });
    });
});