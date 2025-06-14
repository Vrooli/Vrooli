import { describe, expect, it } from "vitest";
import { resourceVersionRelationFixtures } from "./__test/fixtures/resourceVersionRelationFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test/validationTestUtils.js";
import { resourceVersionRelationValidation } from "./resourceVersionRelation.js";

describe("resourceVersionRelationValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        resourceVersionRelationValidation,
        resourceVersionRelationFixtures,
        "resourceVersionRelation",
    );

    describe("create specific validations", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = resourceVersionRelationValidation.create(defaultParams);

        it("should require id in create", async () => {
            const dataWithoutId = { ...resourceVersionRelationFixtures.minimal.create };
            delete dataWithoutId.id;

            await testValidation(
                createSchema,
                dataWithoutId,
                false,
                /required/i,
            );
        });

        it("should require fromVersionConnect in create", async () => {
            await testValidation(
                createSchema,
                resourceVersionRelationFixtures.invalid.missingFromVersion.create,
                false,
                /required/i,
            );
        });

        it("should require toVersionConnect in create", async () => {
            await testValidation(
                createSchema,
                resourceVersionRelationFixtures.invalid.missingToVersion.create,
                false,
                /required/i,
            );
        });

        it("should allow minimal create with required fields", async () => {
            await testValidation(
                createSchema,
                resourceVersionRelationFixtures.minimal.create,
                true,
            );
        });

        it("should allow complete create with all fields", async () => {
            await testValidation(
                createSchema,
                resourceVersionRelationFixtures.complete.create,
                true,
            );
        });

        describe("labels field", () => {
            it("should be optional", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionRelationFixtures.edgeCases.withoutLabels.create,
                    true,
                );
            });

            it("should accept empty array", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionRelationFixtures.edgeCases.emptyLabels.create,
                    true,
                );
            });

            it("should accept single label", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionRelationFixtures.edgeCases.singleLabel.create,
                    true,
                );
            });

            it("should accept multiple labels", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionRelationFixtures.edgeCases.multipleLabels.create,
                    true,
                );
            });

            it("should accept maximum length labels", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionRelationFixtures.edgeCases.maxLengthLabels.create,
                    true,
                );
            });

            it("should reject labels that exceed maximum length", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionRelationFixtures.invalid.longLabel.create,
                    false,
                    /over the limit/i,
                );
            });

            it("should remove empty string labels", async () => {
                const result = await testValidation(
                    createSchema,
                    resourceVersionRelationFixtures.invalid.emptyLabel.create,
                    true,
                );
                // Empty string labels should be filtered out, leaving an empty array which becomes undefined
                expect(result.labels).to.satisfy((labels: any) => labels === undefined || labels.length === 0);
            });

            it("should accept labels with spaces", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionRelationFixtures.edgeCases.labelsWithSpaces.create,
                    true,
                );
            });

            it("should accept labels with special characters", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionRelationFixtures.edgeCases.labelsWithSpecialChars.create,
                    true,
                );
            });
        });

        describe("relationship fields", () => {
            it("should require fromVersionConnect", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionRelationFixtures.invalid.missingFromVersion.create,
                    false,
                    /required/i,
                );
            });

            it("should require toVersionConnect", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionRelationFixtures.invalid.missingToVersion.create,
                    false,
                    /required/i,
                );
            });

            it("should accept valid relationship connections", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionRelationFixtures.minimal.create,
                    true,
                );
            });

            it("should allow relation to same version", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionRelationFixtures.edgeCases.relationToSameVersion.create,
                    true,
                );
            });
        });

        describe("relation types", () => {
            it("should handle dependency relations", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionRelationFixtures.edgeCases.dependencyRelation.create,
                    true,
                );
            });

            it("should handle upgrade relations", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionRelationFixtures.edgeCases.upgradeRelation.create,
                    true,
                );
            });

            it("should handle replacement relations", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionRelationFixtures.edgeCases.replacementRelation.create,
                    true,
                );
            });

            it("should handle common relation types", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionRelationFixtures.edgeCases.commonRelationTypes.create,
                    true,
                );
            });

            it("should handle version evolution relations", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionRelationFixtures.edgeCases.versionEvolution.create,
                    true,
                );
            });

            it("should handle compatibility relations", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionRelationFixtures.edgeCases.compatibilityRelation.create,
                    true,
                );
            });
        });

        describe("relation creation scenarios", () => {
            it("should handle different label combinations", async () => {
                const scenarios = [
                    {
                        data: resourceVersionRelationFixtures.edgeCases.singleLabel.create,
                        shouldPass: true,
                        description: "single label relation",
                    },
                    {
                        data: resourceVersionRelationFixtures.edgeCases.multipleLabels.create,
                        shouldPass: true,
                        description: "multiple labels relation",
                    },
                    {
                        data: resourceVersionRelationFixtures.edgeCases.emptyLabels.create,
                        shouldPass: true,
                        description: "empty labels relation",
                    },
                    {
                        data: resourceVersionRelationFixtures.edgeCases.withoutLabels.create,
                        shouldPass: true,
                        description: "no labels relation",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should handle complex relation structures", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionRelationFixtures.complete.create,
                    true,
                );
            });
        });
    });

    describe("update specific validations", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = resourceVersionRelationValidation.update(defaultParams);

        it("should require id in update", async () => {
            await testValidation(
                updateSchema,
                resourceVersionRelationFixtures.invalid.missingRequired.update,
                false,
                /required/i,
            );
        });

        it("should allow minimal updates with just id", async () => {
            await testValidation(
                updateSchema,
                resourceVersionRelationFixtures.minimal.update,
                true,
            );
        });

        it("should allow complete updates", async () => {
            await testValidation(
                updateSchema,
                resourceVersionRelationFixtures.complete.update,
                true,
            );
        });

        describe("field updates", () => {
            it("should allow updating labels", async () => {
                await testValidation(
                    updateSchema,
                    resourceVersionRelationFixtures.edgeCases.updateLabels.update,
                    true,
                );
            });

            it("should allow clearing labels", async () => {
                await testValidation(
                    updateSchema,
                    resourceVersionRelationFixtures.edgeCases.updateEmptyLabels.update,
                    true,
                );
            });

            it("should allow single label update", async () => {
                await testValidation(
                    updateSchema,
                    resourceVersionRelationFixtures.edgeCases.updateSingleLabel.update,
                    true,
                );
            });

            it("should not allow updating relationship connections", async () => {
                const dataWithRestrictedFields = {
                    id: "123456789012345678",
                    fromVersionConnect: "123456789012345679",
                    toVersionConnect: "123456789012345680",
                    labels: ["updated"],
                };

                const result = await testValidation(updateSchema, dataWithRestrictedFields, true);
                expect(result).to.not.have.property("fromVersionConnect");
                expect(result).to.not.have.property("toVersionConnect");
            });
        });

        describe("update scenarios", () => {
            it("should handle id-only updates", async () => {
                await testValidation(
                    updateSchema,
                    resourceVersionRelationFixtures.edgeCases.updateOnlyId.update,
                    true,
                );
            });

            it("should handle label modifications", async () => {
                const scenarios = [
                    {
                        data: resourceVersionRelationFixtures.edgeCases.updateLabels.update,
                        shouldPass: true,
                        description: "update with new labels",
                    },
                    {
                        data: resourceVersionRelationFixtures.edgeCases.updateEmptyLabels.update,
                        shouldPass: true,
                        description: "clear all labels",
                    },
                    {
                        data: resourceVersionRelationFixtures.edgeCases.updateSingleLabel.update,
                        shouldPass: true,
                        description: "single label update",
                    },
                ];

                await testValidationBatch(updateSchema, scenarios);
            });

            it("should handle comprehensive update operations", async () => {
                await testValidation(
                    updateSchema,
                    resourceVersionRelationFixtures.complete.update,
                    true,
                );
            });
        });
    });

    describe("id validation", () => {
        const createSchema = resourceVersionRelationValidation.create({ omitFields: [] });


        it("should accept valid Snowflake IDs", async () => {
            const validIds = [
                "123456789012345678",
                "999999999999999999",
                "100000000000000000",
            ];

            const scenarios = validIds.map(id => ({
                data: {
                    ...resourceVersionRelationFixtures.minimal.create,
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
                    { ...resourceVersionRelationFixtures.minimal.create, id },
                    false,
                    error,
                );
            }
        });
    });

    describe("type conversions", () => {
        const createSchema = resourceVersionRelationValidation.create({ omitFields: [] });

        it("should handle ID conversion", async () => {
            const dataWithNumberId = {
                id: 123456789012345, // Smaller number to avoid precision issues
                fromVersionConnect: "123456789012345679",
                toVersionConnect: "123456789012345680",
            };

            const result = await testValidation(
                createSchema,
                dataWithNumberId,
                true,
            );
            expect(result.id).to.be.a("string");
            expect(result.id).toBe("123456789012345");
        });

        it("should handle labels array conversion", async () => {
            const dataWithSingleLabel = {
                id: "123456789012345678",
                labels: "single-label", // String instead of array
                fromVersionConnect: "123456789012345679",
                toVersionConnect: "123456789012345680",
            };

            // This should fail since labels expects an array
            await testValidation(
                createSchema,
                dataWithSingleLabel,
                false,
                /must be a.*array.*type/i,
            );
        });
    });

    describe("edge cases", () => {
        const createSchema = resourceVersionRelationValidation.create({ omitFields: [] });
        const updateSchema = resourceVersionRelationValidation.update({ omitFields: [] });

        it("should handle minimal relation creation", async () => {
            await testValidation(
                createSchema,
                resourceVersionRelationFixtures.minimal.create,
                true,
            );
        });

        it("should handle empty update operations", async () => {
            await testValidation(
                updateSchema,
                resourceVersionRelationFixtures.edgeCases.updateOnlyId.update,
                true,
            );
        });

        it("should handle various label patterns", async () => {
            const scenarios = [
                {
                    data: resourceVersionRelationFixtures.edgeCases.labelsWithSpaces.create,
                    shouldPass: true,
                    description: "labels with spaces",
                },
                {
                    data: resourceVersionRelationFixtures.edgeCases.labelsWithSpecialChars.create,
                    shouldPass: true,
                    description: "labels with special characters",
                },
                {
                    data: resourceVersionRelationFixtures.edgeCases.maxLengthLabels.create,
                    shouldPass: true,
                    description: "maximum length labels",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle different relation semantics", async () => {
            const scenarios = [
                {
                    data: resourceVersionRelationFixtures.edgeCases.dependencyRelation.create,
                    shouldPass: true,
                    description: "dependency relation",
                },
                {
                    data: resourceVersionRelationFixtures.edgeCases.upgradeRelation.create,
                    shouldPass: true,
                    description: "upgrade relation",
                },
                {
                    data: resourceVersionRelationFixtures.edgeCases.replacementRelation.create,
                    shouldPass: true,
                    description: "replacement relation",
                },
                {
                    data: resourceVersionRelationFixtures.edgeCases.compatibilityRelation.create,
                    shouldPass: true,
                    description: "compatibility relation",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle complex relation operations", async () => {
            const scenarios = [
                {
                    data: resourceVersionRelationFixtures.edgeCases.relationToSameVersion.create,
                    shouldPass: true,
                    description: "self-referencing relation",
                },
                {
                    data: resourceVersionRelationFixtures.complete.create,
                    shouldPass: true,
                    description: "complete relation with all features",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle comprehensive update scenarios", async () => {
            const scenarios = [
                {
                    data: resourceVersionRelationFixtures.edgeCases.updateLabels.update,
                    shouldPass: true,
                    description: "update with multiple labels",
                },
                {
                    data: resourceVersionRelationFixtures.edgeCases.updateEmptyLabels.update,
                    shouldPass: true,
                    description: "clear all labels update",
                },
                {
                    data: resourceVersionRelationFixtures.edgeCases.updateSingleLabel.update,
                    shouldPass: true,
                    description: "single label update",
                },
            ];

            await testValidationBatch(updateSchema, scenarios);
        });
    });
});
