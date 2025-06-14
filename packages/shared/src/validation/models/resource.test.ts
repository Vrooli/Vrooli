import { describe, expect, it } from "vitest";
import { resourceFixtures } from "./__test/fixtures/resourceFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test/validationTestUtils.js";
import { resourceValidation } from "./resource.js";

describe("resourceValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        resourceValidation,
        resourceFixtures,
        "resource",
    );

    describe("create specific validations", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = resourceValidation.create(defaultParams);

        it("should require id in create", async () => {
            const dataWithoutId = { ...resourceFixtures.minimal.create };
            delete dataWithoutId.id;

            await testValidation(
                createSchema,
                dataWithoutId,
                false,
                /required/i,
            );
        });

        it("should require resourceType in create", async () => {
            const dataWithoutResourceType = {
                id: "123456789012345678",
                ownedByUserConnect: "123456789012345679",
                versionsCreate: [
                    {
                        id: "123456789012345680",
                        versionLabel: "1.0.0",
                        translationsCreate: [
                            {
                                id: "123456789012345681",
                                language: "en",
                                name: "Test Resource",
                            },
                        ],
                    },
                ],
                // Missing required resourceType
            };

            await testValidation(
                createSchema,
                dataWithoutResourceType,
                false,
                /required/i,
            );
        });

        it("should require versions in create", async () => {
            await testValidation(
                createSchema,
                resourceFixtures.invalid.missingVersions.create,
                false,
                /required/i,
            );
        });

        it("should require owner (user or team) in create", async () => {
            await testValidation(
                createSchema,
                resourceFixtures.invalid.missingOwner.create,
                false,
                /Only one of the following fields can be present/i,
            );
        });

        it("should allow minimal create with required fields", async () => {
            await testValidation(
                createSchema,
                resourceFixtures.minimal.create,
                true,
            );
        });

        it("should allow complete create with all fields", async () => {
            await testValidation(
                createSchema,
                resourceFixtures.complete.create,
                true,
            );
        });

        describe("resourceType field", () => {
            it("should accept valid resource types", async () => {
                const scenarios = [
                    {
                        data: resourceFixtures.edgeCases.codeResource.create,
                        shouldPass: true,
                        description: "Code resource",
                    },
                    {
                        data: resourceFixtures.edgeCases.noteResource.create,
                        shouldPass: true,
                        description: "Note resource",
                    },
                    {
                        data: resourceFixtures.edgeCases.projectResource.create,
                        shouldPass: true,
                        description: "Project resource",
                    },
                    {
                        data: resourceFixtures.edgeCases.standardResource.create,
                        shouldPass: true,
                        description: "Standard resource",
                    },
                    {
                        data: resourceFixtures.edgeCases.userOwnedResource.create,
                        shouldPass: true,
                        description: "Api resource",
                    },
                    {
                        data: resourceFixtures.edgeCases.teamOwnedResource.create,
                        shouldPass: true,
                        description: "Routine resource",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject invalid resource types", async () => {
                await testValidation(
                    createSchema,
                    resourceFixtures.invalid.invalidResourceType.create,
                    false,
                    /must be one of/i,
                );
            });
        });

        describe("ownership validation", () => {
            it("should accept user ownership", async () => {
                await testValidation(
                    createSchema,
                    resourceFixtures.edgeCases.userOwnedResource.create,
                    true,
                );
            });

            it("should accept team ownership", async () => {
                await testValidation(
                    createSchema,
                    resourceFixtures.edgeCases.teamOwnedResource.create,
                    true,
                );
            });

            it("should enforce ownership exclusivity", async () => {
                await testValidation(
                    createSchema,
                    resourceFixtures.invalid.bothOwners.create,
                    false,
                    /Only one of the following fields can be present/i,
                );
            });

            it("should require at least one owner", async () => {
                await testValidation(
                    createSchema,
                    resourceFixtures.invalid.missingOwner.create,
                    false,
                    /Only one of the following fields can be present/i,
                );
            });
        });

        describe("optional fields", () => {
            it("should accept publicId", async () => {
                await testValidation(
                    createSchema,
                    resourceFixtures.edgeCases.withPublicId.create,
                    true,
                );
            });

            it("should accept isInternal", async () => {
                await testValidation(
                    createSchema,
                    resourceFixtures.edgeCases.internalResource.create,
                    true,
                );
            });

            it("should accept isPrivate", async () => {
                await testValidation(
                    createSchema,
                    resourceFixtures.edgeCases.privateResource.create,
                    true,
                );
            });

            it("should accept permissions", async () => {
                await testValidation(
                    createSchema,
                    resourceFixtures.edgeCases.withPermissions.create,
                    true,
                );
            });

            it("should reject invalid publicId format", async () => {
                await testValidation(
                    createSchema,
                    resourceFixtures.invalid.invalidPublicId.create,
                    false,
                    /Must be a valid public ID/i,
                );
            });
        });

        describe("relationship fields", () => {
            it("should accept parent relationship", async () => {
                await testValidation(
                    createSchema,
                    resourceFixtures.edgeCases.withParent.create,
                    true,
                );
            });

            it("should accept tag relationships", async () => {
                await testValidation(
                    createSchema,
                    resourceFixtures.edgeCases.withTags.create,
                    true,
                );
            });

            it("should accept multiple versions", async () => {
                await testValidation(
                    createSchema,
                    resourceFixtures.edgeCases.multipleVersions.create,
                    true,
                );
            });

            it("should require at least one version", async () => {
                await testValidation(
                    createSchema,
                    resourceFixtures.invalid.missingVersions.create,
                    false,
                    /required/i,
                );
            });
        });

        describe("boolean field conversions", () => {
            it("should convert string booleans", async () => {
                const result = await testValidation(
                    createSchema,
                    resourceFixtures.edgeCases.booleanConversions.create,
                    true,
                );
                expect(result.isInternal).to.be.a("boolean");
                expect(result.isPrivate).to.be.a("boolean");
                expect(result.isInternal).toBe(true);
                expect(result.isPrivate).toBe(false);
            });
        });

        describe("resource creation scenarios", () => {
            it("should handle different resource types", async () => {
                const resourceTypes = ["Api", "Code", "Note", "Project", "Routine", "Standard"];

                for (const resourceType of resourceTypes) {
                    const data = {
                        id: "123456789012345678",
                        resourceType,
                        ownedByUserConnect: "123456789012345679",
                        versionsCreate: [
                            {
                                id: "123456789012345680",
                                versionLabel: "1.0.0",
                                translationsCreate: [
                                    {
                                        id: "123456789012345681",
                                        language: "en",
                                        name: `${resourceType} Resource`,
                                    },
                                ],
                            },
                        ],
                    };

                    await testValidation(createSchema, data, true);
                }
            });

            it("should handle complex resource with all features", async () => {
                await testValidation(
                    createSchema,
                    resourceFixtures.complete.create,
                    true,
                );
            });
        });
    });

    describe("update specific validations", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = resourceValidation.update(defaultParams);

        it("should require id in update", async () => {
            await testValidation(
                updateSchema,
                resourceFixtures.invalid.missingRequired.update,
                false,
                /required/i,
            );
        });

        it("should allow minimal updates with just id", async () => {
            await testValidation(
                updateSchema,
                resourceFixtures.minimal.update,
                true,
            );
        });

        it("should allow complete updates", async () => {
            await testValidation(
                updateSchema,
                resourceFixtures.complete.update,
                true,
            );
        });

        describe("ownership updates", () => {
            it("should allow changing ownership", async () => {
                await testValidation(
                    updateSchema,
                    resourceFixtures.edgeCases.updateOwnership.update,
                    true,
                );
            });

            it("should not enforce ownership exclusivity in update (allows removal)", async () => {
                const dataWithNoOwners = {
                    id: "123456789012345678",
                };

                // In update, ownership is not required, so no owners is allowed (removal)
                await testValidation(updateSchema, dataWithNoOwners, true);
            });

            it("should not allow updating resourceType", async () => {
                const dataWithRestrictedFields = {
                    id: "123456789012345678",
                    resourceType: "Project",
                    isPrivate: true,
                };

                const result = await testValidation(updateSchema, dataWithRestrictedFields, true);
                expect(result).to.not.have.property("resourceType");
            });

            it("should not allow updating publicId", async () => {
                const dataWithRestrictedFields = {
                    id: "123456789012345678",
                    publicId: "newpub123",
                    isPrivate: true,
                };

                const result = await testValidation(updateSchema, dataWithRestrictedFields, true);
                expect(result).to.not.have.property("publicId");
            });
        });

        describe("field updates", () => {
            it("should allow updating boolean fields", async () => {
                const dataWithBooleans = {
                    id: "123456789012345678",
                    isInternal: true,
                    isPrivate: false,
                };

                await testValidation(updateSchema, dataWithBooleans, true);
            });

            it("should allow updating permissions", async () => {
                const dataWithPermissions = {
                    id: "123456789012345678",
                    permissions: JSON.stringify(["read", "admin"]),
                };

                await testValidation(updateSchema, dataWithPermissions, true);
            });
        });

        describe("relationship updates", () => {
            it("should allow tag operations in update", async () => {
                await testValidation(
                    updateSchema,
                    resourceFixtures.edgeCases.updateWithTagOperations.update,
                    true,
                );
            });

            it("should allow version operations in update", async () => {
                await testValidation(
                    updateSchema,
                    resourceFixtures.edgeCases.updateWithVersionOperations.update,
                    true,
                );
            });

            it("should handle complex update operations", async () => {
                await testValidation(
                    updateSchema,
                    resourceFixtures.complete.update,
                    true,
                );
            });
        });

        describe("update scenarios", () => {
            it("should handle id-only updates", async () => {
                await testValidation(
                    updateSchema,
                    resourceFixtures.edgeCases.updateOnlyId.update,
                    true,
                );
            });

            it("should handle single field updates", async () => {
                const singleFieldData = {
                    id: "123456789012345678",
                    isPrivate: true,
                };

                await testValidation(updateSchema, singleFieldData, true);
            });

            it("should handle multiple relationship operations", async () => {
                const complexUpdateData = {
                    id: "123456789012345678",
                    isInternal: false,
                    tagsConnect: ["123456789012345679"],
                    tagsDisconnect: ["123456789012345680"],
                    versionsUpdate: [
                        {
                            id: "123456789012345681",
                            versionLabel: "1.0.1",
                        },
                    ],
                };

                await testValidation(updateSchema, complexUpdateData, true);
            });
        });
    });

    describe("id validation", () => {
        const createSchema = resourceValidation.create({ omitFields: [] });

        it("should accept valid Snowflake IDs", async () => {
            const validIds = [
                "123456789012345678",
                "999999999999999999",
                "100000000000000000",
            ];

            const scenarios = validIds.map(id => ({
                data: {
                    ...resourceFixtures.minimal.create,
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
                    { ...resourceFixtures.minimal.create, id },
                    false,
                    error,
                );
            }
        });
    });

    describe("type conversions", () => {
        const createSchema = resourceValidation.create({ omitFields: [] });

        it("should handle ID conversion", async () => {
            const dataWithNumberId = {
                id: 123456789012345, // Smaller number to avoid precision issues
                resourceType: "Note",
                ownedByUserConnect: "123456789012345679",
                versionsCreate: [
                    {
                        id: "123456789012345680",
                        versionLabel: "1.0.0",
                        translationsCreate: [
                            {
                                id: "123456789012345681",
                                language: "en",
                                name: "ID Conversion Resource",
                            },
                        ],
                    },
                ],
            };

            const result = await testValidation(
                createSchema,
                dataWithNumberId,
                true,
            );
            expect(result.id).to.be.a("string");
            expect(result.id).toBe("123456789012345");
        });

        it("should handle boolean conversion", async () => {
            const dataWithStringBooleans = {
                id: "123456789012345678",
                isInternal: "yes",
                isPrivate: "1",
                resourceType: "Note",
                ownedByUserConnect: "123456789012345679",
                versionsCreate: [
                    {
                        id: "123456789012345680",
                        versionLabel: "1.0.0",
                        translationsCreate: [
                            {
                                id: "123456789012345681",
                                language: "en",
                                name: "Boolean Conversion Resource",
                            },
                        ],
                    },
                ],
            };

            const result = await testValidation(
                createSchema,
                dataWithStringBooleans,
                true,
            );
            expect(result.isInternal).to.be.a("boolean");
            expect(result.isPrivate).to.be.a("boolean");
            expect(result.isInternal).toBe(true);
            expect(result.isPrivate).toBe(true);
        });
    });

    describe("edge cases", () => {
        const createSchema = resourceValidation.create({ omitFields: [] });
        const updateSchema = resourceValidation.update({ omitFields: [] });

        it("should handle minimal resource creation", async () => {
            await testValidation(
                createSchema,
                resourceFixtures.minimal.create,
                true,
            );
        });

        it("should handle empty update operations", async () => {
            await testValidation(
                updateSchema,
                resourceFixtures.edgeCases.updateOnlyId.update,
                true,
            );
        });

        it("should handle all resource types", async () => {
            const resourceTypes = ["Api", "Code", "Note", "Project", "Routine", "Standard"];

            for (const resourceType of resourceTypes) {
                const data = {
                    id: "123456789012345678",
                    resourceType,
                    ownedByUserConnect: "123456789012345679",
                    versionsCreate: [
                        {
                            id: "123456789012345680",
                            versionLabel: "1.0.0",
                            translationsCreate: [
                                {
                                    id: "123456789012345681",
                                    language: "en",
                                    name: `Test ${resourceType}`,
                                },
                            ],
                        },
                    ],
                };

                await testValidation(createSchema, data, true);
            }
        });

        it("should handle resources with different ownership patterns", async () => {
            const scenarios = [
                {
                    data: resourceFixtures.edgeCases.userOwnedResource.create,
                    shouldPass: true,
                    description: "user owned resource",
                },
                {
                    data: resourceFixtures.edgeCases.teamOwnedResource.create,
                    shouldPass: true,
                    description: "team owned resource",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle resources with various configurations", async () => {
            const scenarios = [
                {
                    data: resourceFixtures.edgeCases.internalResource.create,
                    shouldPass: true,
                    description: "internal resource",
                },
                {
                    data: resourceFixtures.edgeCases.privateResource.create,
                    shouldPass: true,
                    description: "private resource",
                },
                {
                    data: resourceFixtures.edgeCases.withPermissions.create,
                    shouldPass: true,
                    description: "resource with permissions",
                },
                {
                    data: resourceFixtures.edgeCases.withPublicId.create,
                    shouldPass: true,
                    description: "resource with public ID",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle complex resource structures", async () => {
            const scenarios = [
                {
                    data: resourceFixtures.edgeCases.withParent.create,
                    shouldPass: true,
                    description: "resource with parent",
                },
                {
                    data: resourceFixtures.edgeCases.withTags.create,
                    shouldPass: true,
                    description: "resource with tags",
                },
                {
                    data: resourceFixtures.edgeCases.multipleVersions.create,
                    shouldPass: true,
                    description: "resource with multiple versions",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle comprehensive resource updates", async () => {
            const scenarios = [
                {
                    data: resourceFixtures.edgeCases.updateWithTagOperations.update,
                    shouldPass: true,
                    description: "update with tag operations",
                },
                {
                    data: resourceFixtures.edgeCases.updateWithVersionOperations.update,
                    shouldPass: true,
                    description: "update with version operations",
                },
            ];

            await testValidationBatch(updateSchema, scenarios);
        });
    });
});
