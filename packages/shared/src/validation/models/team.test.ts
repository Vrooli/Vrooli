import { describe, expect, it } from "vitest";
import { teamFixtures } from "./__test/fixtures/teamFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test/validationTestUtils.js";
import { teamTranslationValidation, teamValidation } from "./team.js";

describe("teamValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        teamValidation,
        teamFixtures,
        "team",
    );

    describe("create specific validations", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = teamValidation.create(defaultParams);

        it("should require id in create", async () => {
            const dataWithoutId = { ...teamFixtures.minimal.create };
            delete dataWithoutId.id;

            await testValidation(
                createSchema,
                dataWithoutId,
                false,
                /required/i,
            );
        });

        it("should allow teams without translations (though not recommended)", async () => {
            const teamWithoutTranslations = {
                id: "123456789012345678",
                handle: "test_team",
                isOpenToNewMembers: true,
            };

            await testValidation(
                createSchema,
                teamWithoutTranslations,
                true,
            );
        });

        it("should allow minimal create with required fields", async () => {
            await testValidation(
                createSchema,
                teamFixtures.minimal.create,
                true,
            );
        });

        it("should allow complete create with all fields", async () => {
            await testValidation(
                createSchema,
                teamFixtures.complete.create,
                true,
            );
        });

        describe("handle field", () => {
            it("should be optional", async () => {
                await testValidation(
                    createSchema,
                    teamFixtures.edgeCases.withoutOptionalFields.create,
                    true,
                );
            });

            it("should accept valid handles", async () => {
                const scenarios = [
                    {
                        data: teamFixtures.complete.create,
                        shouldPass: true,
                        description: "handle with underscore",
                    },
                    {
                        data: teamFixtures.edgeCases.differentHandle.create,
                        shouldPass: true,
                        description: "unique handle",
                    },
                    {
                        data: teamFixtures.edgeCases.maxLengthFields.create,
                        shouldPass: true,
                        description: "max length handle",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject handles that are too short", async () => {
                await testValidation(
                    createSchema,
                    teamFixtures.invalid.invalidHandle.create,
                    false,
                    /under the limit/i,
                );
            });

            it("should reject handles that are too long", async () => {
                await testValidation(
                    createSchema,
                    teamFixtures.invalid.longHandle.create,
                    false,
                    /over the limit/i,
                );
            });

            it("should reject handles with invalid characters", async () => {
                await testValidation(
                    createSchema,
                    teamFixtures.invalid.invalidHandleChars.create,
                    false,
                    /can only contain letters, numbers, and underscores/i,
                );
            });
        });

        describe("boolean fields", () => {
            it("should accept boolean values for isPrivate and isOpenToNewMembers", async () => {
                const scenarios = [
                    {
                        data: teamFixtures.edgeCases.privateTeam.create,
                        shouldPass: true,
                        description: "private team",
                    },
                    {
                        data: teamFixtures.edgeCases.publicOpenTeam.create,
                        shouldPass: true,
                        description: "public open team",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should convert string boolean values", async () => {
                const dataWithStringBooleans = {
                    id: "123456789012345678",
                    isPrivate: "true",
                    isOpenToNewMembers: "false",
                    translationsCreate: [
                        {
                            id: "123456789012345679",
                            language: "en",
                            name: "String Boolean Team",
                        },
                    ],
                };

                const result = await testValidation(
                    createSchema,
                    dataWithStringBooleans,
                    true,
                );
                expect(result.isPrivate).to.be.a("boolean");
                expect(result.isOpenToNewMembers).to.be.a("boolean");
                expect(result.isPrivate).to.equal(true);
                expect(result.isOpenToNewMembers).to.equal(false);
            });
        });

        describe("image fields", () => {
            it("should be optional", async () => {
                await testValidation(
                    createSchema,
                    teamFixtures.edgeCases.withoutOptionalFields.create,
                    true,
                );
            });

            it("should accept valid image file names", async () => {
                await testValidation(
                    createSchema,
                    teamFixtures.edgeCases.withAllImages.create,
                    true,
                );
            });
        });

        describe("config field", () => {
            it("should be optional", async () => {
                await testValidation(
                    createSchema,
                    teamFixtures.edgeCases.withoutOptionalFields.create,
                    true,
                );
            });

            it("should accept valid JSON config strings", async () => {
                const scenarios = [
                    {
                        data: teamFixtures.complete.create,
                        shouldPass: true,
                        description: "simple config",
                    },
                    {
                        data: teamFixtures.edgeCases.complexConfig.create,
                        shouldPass: true,
                        description: "complex nested config",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });
        });

        describe("tags relationships", () => {
            it("should be optional", async () => {
                await testValidation(
                    createSchema,
                    teamFixtures.edgeCases.withoutOptionalFields.create,
                    true,
                );
            });

            it("should accept tag connections and creations", async () => {
                await testValidation(
                    createSchema,
                    teamFixtures.edgeCases.withTags.create,
                    true,
                );
            });

            it("should handle multiple tag operations", async () => {
                await testValidation(
                    createSchema,
                    teamFixtures.complete.create,
                    true,
                );
            });
        });

        describe("memberInvites relationships", () => {
            it("should be optional", async () => {
                await testValidation(
                    createSchema,
                    teamFixtures.edgeCases.withoutOptionalFields.create,
                    true,
                );
            });

            it("should accept member invite creations", async () => {
                await testValidation(
                    createSchema,
                    teamFixtures.edgeCases.withMemberInvites.create,
                    true,
                );
            });

            it("should handle multiple member invites", async () => {
                await testValidation(
                    createSchema,
                    teamFixtures.complete.create,
                    true,
                );
            });
        });

        describe("translations relationships", () => {

            it("should accept single language translation", async () => {
                await testValidation(
                    createSchema,
                    teamFixtures.minimal.create,
                    true,
                );
            });

            it("should accept multiple language translations", async () => {
                await testValidation(
                    createSchema,
                    teamFixtures.edgeCases.multipleLanguages.create,
                    true,
                );
            });

            it("should validate translation name requirements", async () => {
                await testValidation(
                    createSchema,
                    teamFixtures.invalid.missingTranslationName.create,
                    false,
                    /required/i,
                );
            });

            it("should validate translation name length", async () => {
                await testValidation(
                    createSchema,
                    teamFixtures.invalid.invalidTranslations.create,
                    false,
                    /under the limit/i,
                );
            });
        });

        describe("team configurations", () => {
            it("should handle private teams", async () => {
                await testValidation(
                    createSchema,
                    teamFixtures.edgeCases.privateTeam.create,
                    true,
                );
            });

            it("should handle public open teams", async () => {
                await testValidation(
                    createSchema,
                    teamFixtures.edgeCases.publicOpenTeam.create,
                    true,
                );
            });

            it("should handle teams with detailed information", async () => {
                await testValidation(
                    createSchema,
                    teamFixtures.edgeCases.longBio.create,
                    true,
                );
            });
        });
    });

    describe("update specific validations", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = teamValidation.update(defaultParams);

        it("should require id in update", async () => {
            await testValidation(
                updateSchema,
                teamFixtures.invalid.missingRequired.update,
                false,
                /required/i,
            );
        });

        it("should allow minimal updates with just id", async () => {
            await testValidation(
                updateSchema,
                teamFixtures.minimal.update,
                true,
            );
        });

        it("should allow complete updates", async () => {
            await testValidation(
                updateSchema,
                teamFixtures.complete.update,
                true,
            );
        });

        describe("field updates", () => {
            it("should allow updating all optional fields", async () => {
                await testValidation(
                    updateSchema,
                    teamFixtures.complete.update,
                    true,
                );
            });

            it("should allow updating handle", async () => {
                const dataWithNewHandle = {
                    id: "123456789012345678",
                    handle: "new_handle",
                };

                await testValidation(updateSchema, dataWithNewHandle, true);
            });

            it("should allow updating boolean fields", async () => {
                const dataWithBooleans = {
                    id: "123456789012345678",
                    isPrivate: true,
                    isOpenToNewMembers: false,
                };

                await testValidation(updateSchema, dataWithBooleans, true);
            });

            it("should allow updating config", async () => {
                const dataWithConfig = {
                    id: "123456789012345678",
                    config: { updated: true },
                };

                await testValidation(updateSchema, dataWithConfig, true);
            });
        });

        describe("relationship updates", () => {
            it("should allow tag operations in update", async () => {
                await testValidation(
                    updateSchema,
                    teamFixtures.edgeCases.updateWithTagOperations.update,
                    true,
                );
            });

            it("should allow member operations in update", async () => {
                await testValidation(
                    updateSchema,
                    teamFixtures.edgeCases.updateWithMemberOperations.update,
                    true,
                );
            });

            it("should handle multiple relationship operations", async () => {
                await testValidation(
                    updateSchema,
                    teamFixtures.complete.update,
                    true,
                );
            });

            it("should filter out translation operations from update schema", async () => {
                const dataWithTranslations = {
                    id: "123456789012345678",
                    translationsCreate: [
                        {
                            id: "123456789012345679",
                            language: "en",
                            name: "Updated Team",
                        },
                    ],
                };

                // The update schema should strip unknown fields but still validate successfully
                const result = await testValidation(updateSchema, dataWithTranslations, true);
                expect(result).to.have.property("id", "123456789012345678");
                // Translation operations should be filtered out by the schema
                expect(result).to.not.have.property("translationsCreate");
            });
        });

        describe("update scenarios", () => {
            it("should handle id-only updates", async () => {
                await testValidation(
                    updateSchema,
                    teamFixtures.edgeCases.updateOnlyId.update,
                    true,
                );
            });

            it("should handle tag disconnect operations", async () => {
                const dataWithTagDisconnect = {
                    id: "123456789012345678",
                    tagsDisconnect: ["123456789012345679", "123456789012345680"],
                };

                await testValidation(updateSchema, dataWithTagDisconnect, true);
            });

            it("should handle member delete operations", async () => {
                const dataWithMemberDeletes = {
                    id: "123456789012345678",
                    membersDelete: ["123456789012345679"],
                    memberInvitesDelete: ["123456789012345680"],
                };

                await testValidation(updateSchema, dataWithMemberDeletes, true);
            });
        });
    });

    describe("id validation", () => {
        const createSchema = teamValidation.create({ omitFields: [] });


        it("should accept valid Snowflake IDs", async () => {
            const validIds = [
                "123456789012345678",
                "999999999999999999",
                "100000000000000000",
            ];

            const scenarios = validIds.map(id => ({
                data: {
                    ...teamFixtures.minimal.create,
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
                    { ...teamFixtures.minimal.create, id },
                    false,
                    error,
                );
            }
        });
    });

    describe("type conversions", () => {
        const createSchema = teamValidation.create({ omitFields: [] });

        it("should handle ID conversion", async () => {
            const dataWithNumberId = {
                id: 123456789012345, // Smaller number to avoid precision issues
                translationsCreate: [
                    {
                        id: "123456789012345679",
                        language: "en",
                        name: "ID Conversion Team",
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
                isPrivate: "yes",
                isOpenToNewMembers: "1",
                translationsCreate: [
                    {
                        id: "123456789012345679",
                        language: "en",
                        name: "Boolean Conversion Team",
                    },
                ],
            };

            const result = await testValidation(
                createSchema,
                dataWithStringBooleans,
                true,
            );
            expect(result.isPrivate).to.be.a("boolean");
            expect(result.isOpenToNewMembers).to.be.a("boolean");
            expect(result.isPrivate).to.equal(true);
            expect(result.isOpenToNewMembers).to.equal(true);
        });
    });

    describe("edge cases", () => {
        const createSchema = teamValidation.create({ omitFields: [] });
        const updateSchema = teamValidation.update({ omitFields: [] });

        it("should handle minimal team creation", async () => {
            await testValidation(
                createSchema,
                teamFixtures.minimal.create,
                true,
            );
        });

        it("should handle empty update operations", async () => {
            await testValidation(
                updateSchema,
                teamFixtures.edgeCases.updateOnlyId.update,
                true,
            );
        });

        it("should handle teams with different privacy settings", async () => {
            const scenarios = [
                {
                    data: teamFixtures.edgeCases.privateTeam.create,
                    shouldPass: true,
                    description: "private team",
                },
                {
                    data: teamFixtures.edgeCases.publicOpenTeam.create,
                    shouldPass: true,
                    description: "public open team",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle teams with various content", async () => {
            const scenarios = [
                {
                    data: teamFixtures.edgeCases.withAllImages.create,
                    shouldPass: true,
                    description: "team with images",
                },
                {
                    data: teamFixtures.edgeCases.multipleLanguages.create,
                    shouldPass: true,
                    description: "multilingual team",
                },
                {
                    data: teamFixtures.edgeCases.withTags.create,
                    shouldPass: true,
                    description: "team with tags",
                },
                {
                    data: teamFixtures.edgeCases.withMemberInvites.create,
                    shouldPass: true,
                    description: "team with member invites",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle maximum length fields", async () => {
            await testValidation(
                createSchema,
                teamFixtures.edgeCases.maxLengthFields.create,
                true,
            );
        });
    });
});

describe("teamTranslationValidation", () => {
    describe("create specific validations", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = teamTranslationValidation.create(defaultParams);

        it("should require name in create", async () => {
            const dataWithoutName = {
                id: "123456789012345678",
                language: "en",
                bio: "Team bio without name",
                // Missing required name
            };

            await testValidation(
                createSchema,
                dataWithoutName,
                false,
                /required/i,
            );
        });

        it("should allow minimal create with just name", async () => {
            const minimalData = {
                id: "123456789012345678",
                language: "en",
                name: "Basic Team Name",
            };

            await testValidation(createSchema, minimalData, true);
        });

        it("should allow complete create with name and bio", async () => {
            const completeData = {
                id: "123456789012345678",
                language: "en",
                name: "Complete Team Name",
                bio: "This is a complete team biography.",
            };

            await testValidation(createSchema, completeData, true);
        });

        it("should validate name length", async () => {
            const dataWithShortName = {
                id: "123456789012345678",
                language: "en",
                name: "AB", // Too short
            };

            await testValidation(
                createSchema,
                dataWithShortName,
                false,
                /under the limit/i,
            );
        });
    });

    describe("update specific validations", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = teamTranslationValidation.update(defaultParams);

        it("should make all fields optional in update", async () => {
            const updateWithOnlyId = {
                id: "123456789012345678",
            };

            await testValidation(updateSchema, updateWithOnlyId, true);
        });

        it("should allow updating name", async () => {
            const dataWithName = {
                id: "123456789012345678",
                name: "Updated Team Name",
            };

            await testValidation(updateSchema, dataWithName, true);
        });

        it("should allow updating bio", async () => {
            const dataWithBio = {
                id: "123456789012345678",
                bio: "Updated team biography.",
            };

            await testValidation(updateSchema, dataWithBio, true);
        });

        it("should allow updating both fields", async () => {
            const dataWithBoth = {
                id: "123456789012345678",
                name: "Updated Name",
                bio: "Updated bio.",
            };

            await testValidation(updateSchema, dataWithBoth, true);
        });

        it("should still validate name length in update", async () => {
            const dataWithShortName = {
                id: "123456789012345678",
                name: "AB", // Too short
            };

            await testValidation(
                updateSchema,
                dataWithShortName,
                false,
                /under the limit/i,
            );
        });
    });
});
