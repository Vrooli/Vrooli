import { describe, it } from "mocha";
import { expect } from "chai";
import { resourceVersionValidation } from "./resourceVersion.js";
import { resourceVersionFixtures } from "./__test__/fixtures/resourceVersionFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test__/validationTestUtils.js";

describe("resourceVersionValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        resourceVersionValidation,
        resourceVersionFixtures,
        "resourceVersion",
    );

    describe("create specific validations", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = resourceVersionValidation.create(defaultParams);

        it("should require id in create", async () => {
            const dataWithoutId = { ...resourceVersionFixtures.minimal.create };
            delete dataWithoutId.id;

            await testValidation(
                createSchema,
                dataWithoutId,
                false,
                /required/i,
            );
        });

        it("should require versionLabel in create", async () => {
            const dataWithoutVersionLabel = {
                id: "123456789012345678",
                rootConnect: "123456789012345679",
                translationsCreate: [
                    {
                        id: "123456789012345680",
                        language: "en",
                        name: "Test Version",
                    },
                ],
                // Missing required versionLabel
            };

            await testValidation(
                createSchema,
                dataWithoutVersionLabel,
                false,
                /required/i,
            );
        });

        it("should require root connection (rootConnect or rootCreate) in create", async () => {
            await testValidation(
                createSchema,
                resourceVersionFixtures.invalid.missingRoot.create,
                false,
                /Only one of the following fields can be present/i,
            );
        });

        it("should allow minimal create with required fields", async () => {
            await testValidation(
                createSchema,
                resourceVersionFixtures.minimal.create,
                true,
            );
        });

        it("should allow complete create with all fields", async () => {
            await testValidation(
                createSchema,
                resourceVersionFixtures.complete.create,
                true,
            );
        });

        describe("versionLabel field", () => {
            it("should be required in create", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionFixtures.invalid.invalidVersionLabel.create,
                    false,
                    /required/i,
                );
            });

            it("should accept valid version labels", async () => {
                const validVersions = ["1.0.0", "2.1.3", "0.1.0-beta", "1.0.0-alpha.1"];

                for (const versionLabel of validVersions) {
                    const data = {
                        ...resourceVersionFixtures.minimal.create,
                        versionLabel,
                    };

                    await testValidation(createSchema, data, true);
                }
            });
        });

        describe("root connection validation", () => {
            it("should accept rootConnect", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionFixtures.minimal.create,
                    true,
                );
            });

            it("should accept rootCreate", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionFixtures.edgeCases.rootCreateVersion.create,
                    true,
                );
            });

            it("should enforce root exclusivity", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionFixtures.invalid.bothRoots.create,
                    false,
                    /Only one of the following fields can be present/i,
                );
            });

            it("should require at least one root connection", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionFixtures.invalid.missingRoot.create,
                    false,
                    /Only one of the following fields can be present/i,
                );
            });
        });

        describe("optional fields", () => {
            it("should accept publicId", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionFixtures.edgeCases.withPublicId.create,
                    true,
                );
            });

            it("should accept codeLanguage", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionFixtures.edgeCases.codeVersion.create,
                    true,
                );
            });

            it("should reject codeLanguage that exceeds maximum length", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionFixtures.invalid.invalidCodeLanguage.create,
                    false,
                    /over the limit/i,
                );
            });

            it("should accept config as JSON string", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionFixtures.edgeCases.withConfiguration.create,
                    true,
                );
            });

            it("should reject invalid config JSON", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionFixtures.invalid.invalidConfig.create,
                    false,
                    /valid JSON/i,
                );
            });

            it("should accept boolean fields", async () => {
                const scenarios = [
                    {
                        data: resourceVersionFixtures.edgeCases.automatedVersion.create,
                        shouldPass: true,
                        description: "isAutomatable true",
                    },
                    {
                        data: resourceVersionFixtures.edgeCases.completeVersion.create,
                        shouldPass: true,
                        description: "isComplete true",
                    },
                    {
                        data: resourceVersionFixtures.edgeCases.internalVersion.create,
                        shouldPass: true,
                        description: "isInternal true",
                    },
                    {
                        data: resourceVersionFixtures.edgeCases.privateVersion.create,
                        shouldPass: true,
                        description: "isPrivate true",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should accept resourceSubType", async () => {
                const scenarios = [
                    {
                        data: resourceVersionFixtures.edgeCases.codeVersion.create,
                        shouldPass: true,
                        description: "Function subtype",
                    },
                    {
                        data: resourceVersionFixtures.edgeCases.routineVersion.create,
                        shouldPass: true,
                        description: "Generate subtype",
                    },
                    {
                        data: resourceVersionFixtures.edgeCases.standardVersion.create,
                        shouldPass: true,
                        description: "Prompt subtype",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject invalid resourceSubType", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionFixtures.invalid.invalidResourceSubType.create,
                    false,
                    /must be one of/i,
                );
            });

            it("should accept versionNotes", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionFixtures.edgeCases.withVersionNotes.create,
                    true,
                );
            });

            it("should reject versionNotes that exceed maximum length", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionFixtures.invalid.longVersionNotes.create,
                    false,
                    /over the limit/i,
                );
            });
        });

        describe("relationship fields", () => {
            it("should accept related versions", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionFixtures.edgeCases.withRelatedVersions.create,
                    true,
                );
            });

            it("should accept translations", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionFixtures.edgeCases.multipleTranslations.create,
                    true,
                );
            });

            it("should accept single translation", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionFixtures.minimal.create,
                    true,
                );
            });
        });

        describe("boolean field conversions", () => {
            it("should convert string booleans", async () => {
                const result = await testValidation(
                    createSchema,
                    resourceVersionFixtures.edgeCases.booleanConversions.create,
                    true,
                );
                expect(result.isAutomatable).to.be.a("boolean");
                expect(result.isComplete).to.be.a("boolean");
                expect(result.isInternal).to.be.a("boolean");
                expect(result.isPrivate).to.be.a("boolean");
                expect(result.isAutomatable).to.equal(true);
                expect(result.isComplete).to.equal(false);
                expect(result.isInternal).to.equal(true);
                expect(result.isPrivate).to.equal(false);
            });
        });

        describe("version creation scenarios", () => {
            it("should handle different resource subtypes", async () => {
                const subtypes = [
                    "Api", "DataConverter", "Form", "Generate", "Informational", 
                    "LLMCallDirectly", "LLMCallFunction", "SingleStep", 
                    "Function", "Prompt"
                ];

                for (const resourceSubType of subtypes) {
                    const data = {
                        id: "123456789012345678",
                        resourceSubType,
                        versionLabel: "1.0.0",
                        rootConnect: "123456789012345679",
                        translationsCreate: [
                            {
                                id: "123456789012345680",
                                language: "en",
                                name: `${resourceSubType} Version`,
                            },
                        ],
                    };

                    await testValidation(createSchema, data, true);
                }
            });

            it("should handle version with complex configuration", async () => {
                await testValidation(
                    createSchema,
                    resourceVersionFixtures.complete.create,
                    true,
                );
            });
        });
    });

    describe("update specific validations", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = resourceVersionValidation.update(defaultParams);

        it("should require id in update", async () => {
            await testValidation(
                updateSchema,
                resourceVersionFixtures.invalid.missingRequired.update,
                false,
                /required/i,
            );
        });

        it("should allow minimal updates with just id", async () => {
            await testValidation(
                updateSchema,
                resourceVersionFixtures.minimal.update,
                true,
            );
        });

        it("should allow complete updates", async () => {
            await testValidation(
                updateSchema,
                resourceVersionFixtures.complete.update,
                true,
            );
        });

        describe("field updates", () => {
            it("should allow updating versionLabel", async () => {
                await testValidation(
                    updateSchema,
                    resourceVersionFixtures.edgeCases.updateVersionLabel.update,
                    true,
                );
            });

            it("should allow updating configuration", async () => {
                await testValidation(
                    updateSchema,
                    resourceVersionFixtures.edgeCases.updateConfiguration.update,
                    true,
                );
            });

            it("should allow updating boolean fields", async () => {
                await testValidation(
                    updateSchema,
                    resourceVersionFixtures.edgeCases.updateBooleanFields.update,
                    true,
                );
            });

            it("should not allow updating publicId", async () => {
                const dataWithRestrictedFields = {
                    id: "123456789012345678",
                    publicId: "newpub123",
                    versionLabel: "1.0.1",
                };

                const result = await testValidation(updateSchema, dataWithRestrictedFields, true);
                expect(result).to.not.have.property("publicId");
            });

            it("should not allow updating resourceSubType", async () => {
                const dataWithRestrictedFields = {
                    id: "123456789012345678",
                    resourceSubType: "Generate",
                    versionLabel: "1.0.1",
                };

                const result = await testValidation(updateSchema, dataWithRestrictedFields, true);
                expect(result).to.not.have.property("resourceSubType");
            });
        });

        describe("relationship updates", () => {
            it("should allow updating root resource", async () => {
                await testValidation(
                    updateSchema,
                    resourceVersionFixtures.edgeCases.updateWithRelationships.update,
                    true,
                );
            });

            it("should allow related version operations", async () => {
                await testValidation(
                    updateSchema,
                    resourceVersionFixtures.complete.update,
                    true,
                );
            });

            it("should allow translation operations", async () => {
                await testValidation(
                    updateSchema,
                    resourceVersionFixtures.complete.update,
                    true,
                );
            });
        });

        describe("update scenarios", () => {
            it("should handle id-only updates", async () => {
                await testValidation(
                    updateSchema,
                    resourceVersionFixtures.edgeCases.updateOnlyId.update,
                    true,
                );
            });

            it("should handle single field updates", async () => {
                await testValidation(
                    updateSchema,
                    resourceVersionFixtures.edgeCases.updateVersionLabel.update,
                    true,
                );
            });

            it("should handle complex update operations", async () => {
                await testValidation(
                    updateSchema,
                    resourceVersionFixtures.edgeCases.updateWithRelationships.update,
                    true,
                );
            });
        });
    });

    describe("id validation", () => {
        const createSchema = resourceVersionValidation.create({ omitFields: [] });
        const updateSchema = resourceVersionValidation.update({ omitFields: [] });

        it("should accept valid Snowflake IDs", async () => {
            const validIds = [
                "123456789012345678",
                "999999999999999999",
                "100000000000000000",
            ];

            const scenarios = validIds.map(id => ({
                data: {
                    ...resourceVersionFixtures.minimal.create,
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
                    { ...resourceVersionFixtures.minimal.create, id },
                    false,
                    error,
                );
            }
        });
    });

    describe("type conversions", () => {
        const createSchema = resourceVersionValidation.create({ omitFields: [] });

        it("should handle ID conversion", async () => {
            const dataWithNumberId = {
                id: 123456789012345, // Smaller number to avoid precision issues
                versionLabel: "1.0.0",
                rootConnect: "123456789012345679",
                translationsCreate: [
                    {
                        id: "123456789012345680",
                        language: "en",
                        name: "ID Conversion Version",
                    },
                ],
            };

            const result = await testValidation(
                createSchema,
                dataWithNumberId,
                true,
            );
            expect(result.id).to.be.a("string");
            expect(result.id).to.equal("123456789012345");
        });

        it("should handle boolean conversion", async () => {
            const dataWithStringBooleans = {
                id: "123456789012345678",
                isAutomatable: "yes",
                isComplete: "1",
                isInternal: "true",
                isPrivate: "0",
                versionLabel: "1.0.0",
                rootConnect: "123456789012345679",
                translationsCreate: [
                    {
                        id: "123456789012345680",
                        language: "en",
                        name: "Boolean Conversion Version",
                    },
                ],
            };

            const result = await testValidation(
                createSchema,
                dataWithStringBooleans,
                true,
            );
            expect(result.isAutomatable).to.be.a("boolean");
            expect(result.isComplete).to.be.a("boolean");
            expect(result.isInternal).to.be.a("boolean");
            expect(result.isPrivate).to.be.a("boolean");
            expect(result.isAutomatable).to.equal(true);
            expect(result.isComplete).to.equal(true);
            expect(result.isInternal).to.equal(true);
            expect(result.isPrivate).to.equal(false);
        });
    });

    describe("edge cases", () => {
        const createSchema = resourceVersionValidation.create({ omitFields: [] });
        const updateSchema = resourceVersionValidation.update({ omitFields: [] });

        it("should handle minimal version creation", async () => {
            await testValidation(
                createSchema,
                resourceVersionFixtures.minimal.create,
                true,
            );
        });

        it("should handle empty update operations", async () => {
            await testValidation(
                updateSchema,
                resourceVersionFixtures.edgeCases.updateOnlyId.update,
                true,
            );
        });

        it("should handle all resource subtypes", async () => {
            const codeSubtypes = ["Function"];
            const routineSubtypes = ["Api", "DataConverter", "Form", "Generate", "Informational", "LLMCallDirectly", "LLMCallFunction", "SingleStep"];
            const standardSubtypes = ["Prompt"];

            const allSubtypes = [...codeSubtypes, ...routineSubtypes, ...standardSubtypes];

            for (const resourceSubType of allSubtypes) {
                const data = {
                    id: "123456789012345678",
                    resourceSubType,
                    versionLabel: "1.0.0",
                    rootConnect: "123456789012345679",
                    translationsCreate: [
                        {
                            id: "123456789012345680",
                            language: "en",
                            name: `Test ${resourceSubType}`,
                        },
                    ],
                };

                await testValidation(createSchema, data, true);
            }
        });

        it("should handle versions with different configurations", async () => {
            const scenarios = [
                {
                    data: resourceVersionFixtures.edgeCases.automatedVersion.create,
                    shouldPass: true,
                    description: "automated version",
                },
                {
                    data: resourceVersionFixtures.edgeCases.completeVersion.create,
                    shouldPass: true,
                    description: "complete version",
                },
                {
                    data: resourceVersionFixtures.edgeCases.internalVersion.create,
                    shouldPass: true,
                    description: "internal version",
                },
                {
                    data: resourceVersionFixtures.edgeCases.privateVersion.create,
                    shouldPass: true,
                    description: "private version",
                },
                {
                    data: resourceVersionFixtures.edgeCases.withConfiguration.create,
                    shouldPass: true,
                    description: "version with configuration",
                },
                {
                    data: resourceVersionFixtures.edgeCases.withVersionNotes.create,
                    shouldPass: true,
                    description: "version with notes",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle complex version structures", async () => {
            const scenarios = [
                {
                    data: resourceVersionFixtures.edgeCases.withRelatedVersions.create,
                    shouldPass: true,
                    description: "version with related versions",
                },
                {
                    data: resourceVersionFixtures.edgeCases.multipleTranslations.create,
                    shouldPass: true,
                    description: "version with multiple translations",
                },
                {
                    data: resourceVersionFixtures.complete.create,
                    shouldPass: true,
                    description: "complete version with all features",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle different root connection types", async () => {
            const scenarios = [
                {
                    data: resourceVersionFixtures.minimal.create,
                    shouldPass: true,
                    description: "version with rootConnect",
                },
                {
                    data: resourceVersionFixtures.edgeCases.rootCreateVersion.create,
                    shouldPass: true,
                    description: "version with rootCreate",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle comprehensive version updates", async () => {
            const scenarios = [
                {
                    data: resourceVersionFixtures.edgeCases.updateConfiguration.update,
                    shouldPass: true,
                    description: "update with configuration changes",
                },
                {
                    data: resourceVersionFixtures.edgeCases.updateBooleanFields.update,
                    shouldPass: true,
                    description: "update with boolean field changes",
                },
                {
                    data: resourceVersionFixtures.edgeCases.updateWithRelationships.update,
                    shouldPass: true,
                    description: "update with relationship changes",
                },
            ];

            await testValidationBatch(updateSchema, scenarios);
        });
    });
});