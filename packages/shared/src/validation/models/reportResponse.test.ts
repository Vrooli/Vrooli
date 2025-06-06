import { describe, it } from "mocha";
import { expect } from "chai";
import { reportResponseValidation } from "./reportResponse.js";
import { reportResponseFixtures } from "./__test__/fixtures/reportResponseFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test__/validationTestUtils.js";

describe("reportResponseValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        reportResponseValidation,
        reportResponseFixtures,
        "reportResponse",
    );

    describe("create specific validations", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = reportResponseValidation.create(defaultParams);

        it("should require id in create", async () => {
            const dataWithoutId = { ...reportResponseFixtures.minimal.create };
            delete dataWithoutId.id;

            await testValidation(
                createSchema,
                dataWithoutId,
                false,
                /required/i,
            );
        });

        it("should require actionSuggested in create", async () => {
            const dataWithoutActionSuggested = {
                id: "123456789012345678",
                reportConnect: "123456789012345679",
                // Missing required actionSuggested
            };

            await testValidation(
                createSchema,
                dataWithoutActionSuggested,
                false,
                /required/i,
            );
        });

        it("should require reportConnect in create", async () => {
            await testValidation(
                createSchema,
                reportResponseFixtures.invalid.missingReport.create,
                false,
                /required/i,
            );
        });

        it("should allow minimal create with required fields", async () => {
            await testValidation(
                createSchema,
                reportResponseFixtures.minimal.create,
                true,
            );
        });

        it("should allow complete create with all fields", async () => {
            await testValidation(
                createSchema,
                reportResponseFixtures.complete.create,
                true,
            );
        });

        describe("actionSuggested field", () => {
            it("should accept valid action types", async () => {
                const scenarios = [
                    {
                        data: reportResponseFixtures.edgeCases.deleteAction.create,
                        shouldPass: true,
                        description: "Delete action",
                    },
                    {
                        data: reportResponseFixtures.edgeCases.falseReportAction.create,
                        shouldPass: true,
                        description: "FalseReport action",
                    },
                    {
                        data: reportResponseFixtures.edgeCases.hideUntilFixedAction.create,
                        shouldPass: true,
                        description: "HideUntilFixed action",
                    },
                    {
                        data: reportResponseFixtures.edgeCases.nonIssueAction.create,
                        shouldPass: true,
                        description: "NonIssue action",
                    },
                    {
                        data: reportResponseFixtures.edgeCases.suspendUserAction.create,
                        shouldPass: true,
                        description: "SuspendUser action",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject invalid action types", async () => {
                await testValidation(
                    createSchema,
                    reportResponseFixtures.invalid.invalidActionSuggested.create,
                    false,
                    /must be one of/i,
                );
            });
        });

        describe("details field", () => {
            it("should be optional", async () => {
                await testValidation(
                    createSchema,
                    reportResponseFixtures.edgeCases.withoutDetails.create,
                    true,
                );
            });

            it("should accept valid details", async () => {
                await testValidation(
                    createSchema,
                    reportResponseFixtures.complete.create,
                    true,
                );
            });

            it("should accept maximum length details", async () => {
                await testValidation(
                    createSchema,
                    reportResponseFixtures.edgeCases.maxLengthDetails.create,
                    true,
                );
            });

            it("should reject details that exceed maximum length", async () => {
                await testValidation(
                    createSchema,
                    reportResponseFixtures.invalid.longDetails.create,
                    false,
                    /over the limit/i,
                );
            });
        });

        describe("language field", () => {
            it("should be optional", async () => {
                await testValidation(
                    createSchema,
                    reportResponseFixtures.edgeCases.withoutLanguage.create,
                    true,
                );
            });

            it("should accept valid language codes", async () => {
                await testValidation(
                    createSchema,
                    reportResponseFixtures.edgeCases.differentLanguages.create,
                    true,
                );
            });
        });

        describe("relationship fields", () => {
            it("should require reportConnect", async () => {
                await testValidation(
                    createSchema,
                    reportResponseFixtures.invalid.missingReport.create,
                    false,
                    /required/i,
                );
            });

            it("should accept valid relationship connections", async () => {
                await testValidation(
                    createSchema,
                    reportResponseFixtures.minimal.create,
                    true,
                );
            });
        });

        describe("response scenarios", () => {
            it("should handle different types of actions", async () => {
                const scenarios = [
                    {
                        data: reportResponseFixtures.edgeCases.deleteAction.create,
                        shouldPass: true,
                        description: "delete action response",
                    },
                    {
                        data: reportResponseFixtures.edgeCases.suspendUserAction.create,
                        shouldPass: true,
                        description: "suspend user action response",
                    },
                    {
                        data: reportResponseFixtures.edgeCases.falseReportAction.create,
                        shouldPass: true,
                        description: "false report action response",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should handle responses with detailed explanations", async () => {
                await testValidation(
                    createSchema,
                    reportResponseFixtures.edgeCases.longDetailedResponse.create,
                    true,
                );
            });

            it("should handle responses with different IDs", async () => {
                await testValidation(
                    createSchema,
                    reportResponseFixtures.edgeCases.differentIds.create,
                    true,
                );
            });
        });
    });

    describe("update specific validations", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = reportResponseValidation.update(defaultParams);

        it("should require id in update", async () => {
            await testValidation(
                updateSchema,
                reportResponseFixtures.invalid.missingRequired.update,
                false,
                /required/i,
            );
        });

        it("should allow minimal updates with just id", async () => {
            await testValidation(
                updateSchema,
                reportResponseFixtures.minimal.update,
                true,
            );
        });

        it("should allow complete updates", async () => {
            await testValidation(
                updateSchema,
                reportResponseFixtures.complete.update,
                true,
            );
        });

        describe("field updates", () => {
            it("should allow updating actionSuggested", async () => {
                await testValidation(
                    updateSchema,
                    reportResponseFixtures.edgeCases.updateActionSuggested.update,
                    true,
                );
            });

            it("should allow updating details", async () => {
                await testValidation(
                    updateSchema,
                    reportResponseFixtures.edgeCases.updateDetails.update,
                    true,
                );
            });

            it("should allow updating language", async () => {
                await testValidation(
                    updateSchema,
                    reportResponseFixtures.edgeCases.updateLanguage.update,
                    true,
                );
            });

            it("should allow updating all optional fields", async () => {
                await testValidation(
                    updateSchema,
                    reportResponseFixtures.edgeCases.updateAllFields.update,
                    true,
                );
            });

            it("should not allow updating reportConnect", async () => {
                const dataWithRestrictedFields = {
                    id: "123456789012345678",
                    reportConnect: "123456789012345679",
                    actionSuggested: "Delete",
                };

                const result = await testValidation(updateSchema, dataWithRestrictedFields, true);
                expect(result).to.not.have.property("reportConnect");
            });
        });

        describe("validation in updates", () => {
            it("should still validate actionSuggested values in update", async () => {
                const dataWithInvalidAction = {
                    id: "123456789012345678",
                    actionSuggested: "InvalidAction",
                };

                await testValidation(
                    updateSchema,
                    dataWithInvalidAction,
                    false,
                    /must be one of/i,
                );
            });

            it("should still validate details length in update", async () => {
                const dataWithLongDetails = {
                    id: "123456789012345678",
                    details: "x".repeat(8193), // Too long
                };

                await testValidation(
                    updateSchema,
                    dataWithLongDetails,
                    false,
                    /over the limit/i,
                );
            });

            it("should handle empty details in update", async () => {
                const dataWithEmptyDetails = {
                    id: "123456789012345678",
                    details: "", // Empty string becomes undefined
                };

                const result = await testValidation(updateSchema, dataWithEmptyDetails, true);
                expect(result).to.not.have.property("details");
            });
        });

        describe("update scenarios", () => {
            it("should handle id-only updates", async () => {
                await testValidation(
                    updateSchema,
                    reportResponseFixtures.edgeCases.updateOnlyId.update,
                    true,
                );
            });

            it("should handle single field updates", async () => {
                await testValidation(
                    updateSchema,
                    reportResponseFixtures.edgeCases.updateActionSuggested.update,
                    true,
                );
            });

            it("should handle multiple field updates", async () => {
                await testValidation(
                    updateSchema,
                    reportResponseFixtures.edgeCases.multipleFieldsUpdate.update,
                    true,
                );
            });

            it("should handle action change updates", async () => {
                const scenarios = [
                    {
                        data: { id: "123456789012345678", actionSuggested: "Delete" },
                        shouldPass: true,
                        description: "change to Delete",
                    },
                    {
                        data: { id: "123456789012345678", actionSuggested: "SuspendUser" },
                        shouldPass: true,
                        description: "change to SuspendUser",
                    },
                    {
                        data: { id: "123456789012345678", actionSuggested: "FalseReport" },
                        shouldPass: true,
                        description: "change to FalseReport",
                    },
                ];

                await testValidationBatch(updateSchema, scenarios);
            });
        });
    });

    describe("id validation", () => {
        const createSchema = reportResponseValidation.create({ omitFields: [] });
        const updateSchema = reportResponseValidation.update({ omitFields: [] });

        it("should accept valid Snowflake IDs", async () => {
            const validIds = [
                "123456789012345678",
                "999999999999999999",
                "100000000000000000",
            ];

            const scenarios = validIds.map(id => ({
                data: {
                    ...reportResponseFixtures.minimal.create,
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
                    { ...reportResponseFixtures.minimal.create, id },
                    false,
                    error,
                );
            }
        });
    });

    describe("type conversions", () => {
        const createSchema = reportResponseValidation.create({ omitFields: [] });

        it("should handle ID conversion", async () => {
            const dataWithNumberId = {
                id: 123456789012345, // Smaller number to avoid precision issues
                actionSuggested: "NonIssue",
                reportConnect: "123456789012345679",
            };

            const result = await testValidation(
                createSchema,
                dataWithNumberId,
                true,
            );
            expect(result.id).to.be.a("string");
            expect(result.id).to.equal("123456789012345");
        });
    });

    describe("edge cases", () => {
        const createSchema = reportResponseValidation.create({ omitFields: [] });
        const updateSchema = reportResponseValidation.update({ omitFields: [] });

        it("should handle minimal response creation", async () => {
            await testValidation(
                createSchema,
                reportResponseFixtures.minimal.create,
                true,
            );
        });

        it("should handle empty update operations", async () => {
            await testValidation(
                updateSchema,
                reportResponseFixtures.edgeCases.updateOnlyId.update,
                true,
            );
        });

        it("should handle all action types", async () => {
            const actionTypes = [
                "Delete",
                "FalseReport",
                "HideUntilFixed",
                "NonIssue",
                "SuspendUser",
            ];

            for (const actionType of actionTypes) {
                const data = {
                    id: "123456789012345678",
                    actionSuggested: actionType,
                    reportConnect: "123456789012345679",
                };

                await testValidation(createSchema, data, true);
            }
        });

        it("should handle responses in different languages", async () => {
            const languages = ["en", "es", "fr", "de", "it"];

            for (const language of languages) {
                const data = {
                    id: "123456789012345678",
                    actionSuggested: "NonIssue",
                    language,
                    reportConnect: "123456789012345679",
                };

                await testValidation(createSchema, data, true);
            }
        });

        it("should handle responses with and without details", async () => {
            const scenarios = [
                {
                    data: reportResponseFixtures.edgeCases.withoutDetails.create,
                    shouldPass: true,
                    description: "response without details",
                },
                {
                    data: reportResponseFixtures.edgeCases.maxLengthDetails.create,
                    shouldPass: true,
                    description: "response with maximum length details",
                },
                {
                    data: reportResponseFixtures.edgeCases.longDetailedResponse.create,
                    shouldPass: true,
                    description: "response with detailed explanation",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle complex response scenarios", async () => {
            const scenarios = [
                {
                    data: reportResponseFixtures.edgeCases.deleteAction.create,
                    shouldPass: true,
                    description: "delete action with explanation",
                },
                {
                    data: reportResponseFixtures.edgeCases.suspendUserAction.create,
                    shouldPass: true,
                    description: "suspend user action with details",
                },
                {
                    data: reportResponseFixtures.edgeCases.falseReportAction.create,
                    shouldPass: true,
                    description: "false report determination",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });
    });
});