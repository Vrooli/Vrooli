import { describe, expect, it } from "vitest";
import { meetingFixtures } from "./__test/fixtures/meetingFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test/validationTestUtils.js";
import { meetingValidation } from "./meeting.js";

describe("meetingValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        meetingValidation,
        meetingFixtures,
        "meeting",
    );

    describe("create specific validations", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = meetingValidation.create(defaultParams);

        it("should require id in create", async () => {
            const dataWithoutId = { ...meetingFixtures.minimal.create };
            delete dataWithoutId.id;

            await testValidation(
                createSchema,
                dataWithoutId,
                false,
                /required/i,
            );
        });

        it("should require teamConnect in create", async () => {
            await testValidation(
                createSchema,
                meetingFixtures.invalid.missingRequired.create,
                false,
                /required/i,
            );
        });

        it("should allow minimal create with just id and teamConnect", async () => {
            await testValidation(
                createSchema,
                meetingFixtures.minimal.create,
                true,
            );
        });

        it("should allow complete create with all fields", async () => {
            await testValidation(
                createSchema,
                meetingFixtures.complete.create,
                true,
            );
        });

        describe("boolean fields", () => {
            it("should accept boolean values", async () => {
                const scenarios = [
                    {
                        data: { ...meetingFixtures.minimal.create, openToAnyoneWithInvite: true },
                        shouldPass: true,
                        description: "openToAnyoneWithInvite true",
                    },
                    {
                        data: { ...meetingFixtures.minimal.create, openToAnyoneWithInvite: false },
                        shouldPass: true,
                        description: "openToAnyoneWithInvite false",
                    },
                    {
                        data: { ...meetingFixtures.minimal.create, showOnTeamProfile: true },
                        shouldPass: true,
                        description: "showOnTeamProfile true",
                    },
                    {
                        data: { ...meetingFixtures.minimal.create, showOnTeamProfile: false },
                        shouldPass: true,
                        description: "showOnTeamProfile false",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should be optional", async () => {
                await testValidation(createSchema, meetingFixtures.minimal.create, true);
            });

            it("should convert string values to boolean", async () => {
                const scenarios = [
                    {
                        data: { ...meetingFixtures.minimal.create, openToAnyoneWithInvite: "true" },
                        shouldPass: true,
                        description: "string 'true' converts to boolean",
                    },
                    {
                        data: { ...meetingFixtures.minimal.create, openToAnyoneWithInvite: "false" },
                        shouldPass: true,
                        description: "string 'false' converts to boolean",
                    },
                    {
                        data: { ...meetingFixtures.minimal.create, showOnTeamProfile: "yes" },
                        shouldPass: true,
                        description: "string 'yes' converts to boolean",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });
        });

        describe("teamConnect field", () => {
            it("should accept valid team ID", async () => {
                const validData = {
                    ...meetingFixtures.minimal.create,
                    teamConnect: "123456789012345678",
                };

                await testValidation(createSchema, validData, true);
            });

            it("should reject invalid team ID", async () => {
                const invalidData = {
                    ...meetingFixtures.minimal.create,
                    teamConnect: "invalid-id",
                };

                await testValidation(
                    createSchema,
                    invalidData,
                    false,
                    /valid ID/i,
                );
            });
        });

        describe("translations", () => {
            it("should accept optional translations in create", async () => {
                const dataWithTranslations = {
                    ...meetingFixtures.minimal.create,
                    translationsCreate: [{
                        id: "123456789012345678",
                        language: "en",
                        name: "Test Meeting",
                        description: "Test meeting description",
                        link: "https://example.com/meeting",
                    }],
                };

                await testValidation(createSchema, dataWithTranslations, true);
            });

            it("should work without translations", async () => {
                await testValidation(createSchema, meetingFixtures.minimal.create, true);
            });

            it("should accept multiple translations", async () => {
                await testValidation(
                    createSchema,
                    meetingFixtures.edgeCases.multipleTranslations.create,
                    true,
                );
            });

            it("should validate translation name requirements", async () => {
                const scenarios = [
                    {
                        data: meetingFixtures.invalid.invalidTranslations.create,
                        shouldPass: false,
                        expectedError: /under the limit/,
                        description: "name too short",
                    },
                    {
                        data: meetingFixtures.invalid.tooLongTranslationName.create,
                        shouldPass: false,
                        expectedError: /over the limit/,
                        description: "name too long",
                    },
                    {
                        data: meetingFixtures.edgeCases.minimalTranslation.create,
                        shouldPass: true,
                        description: "minimal valid name",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should validate translation description requirements", async () => {
                const scenarios = [
                    {
                        data: meetingFixtures.invalid.tooLongTranslationDescription.create,
                        shouldPass: false,
                        expectedError: /over the limit/,
                        description: "description too long",
                    },
                    {
                        data: meetingFixtures.edgeCases.maxLengthTranslation.create,
                        shouldPass: true,
                        description: "description at max length",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should validate translation link requirements", async () => {
                const scenarios = [
                    {
                        data: meetingFixtures.invalid.invalidTranslationUrl.create,
                        shouldPass: false,
                        expectedError: /Must be a URL/,
                        description: "invalid URL format",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });
        });

        describe("invites", () => {
            it("should accept optional invites in create", async () => {
                const dataWithInvites = {
                    ...meetingFixtures.minimal.create,
                    invitesCreate: [{
                        id: "123456789012345678",
                        message: "Join our meeting",
                        meetingConnect: "123456789012345679",
                        userConnect: "123456789012345680",
                    }],
                };

                await testValidation(createSchema, dataWithInvites, true);
            });

            it("should work without invites", async () => {
                await testValidation(createSchema, meetingFixtures.minimal.create, true);
            });

            it("should validate invite requirements", async () => {
                await testValidation(
                    createSchema,
                    meetingFixtures.invalid.invalidInvite.create,
                    false,
                    /required/i,
                );
            });
        });

        // Note: Schedule validation tests are excluded due to complex endTime validation logic
        // Schedule functionality will be tested separately in schedule.test.ts
    });

    describe("update specific validations", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = meetingValidation.update(defaultParams);

        it("should require id in update", async () => {
            await testValidation(
                updateSchema,
                meetingFixtures.invalid.missingRequired.update,
                false,
                /required/i,
            );
        });

        it("should allow minimal updates with just id", async () => {
            await testValidation(
                updateSchema,
                meetingFixtures.minimal.update,
                true,
            );
        });

        it("should allow complete updates", async () => {
            await testValidation(
                updateSchema,
                meetingFixtures.complete.update,
                true,
            );
        });

        it("should allow all translation operations in update", async () => {
            const scenarios = [
                {
                    data: {
                        ...meetingFixtures.minimal.update,
                        translationsCreate: [{
                            id: "123456789012345678",
                            language: "en",
                            name: "New Meeting",
                            description: "New meeting description",
                        }],
                    },
                    shouldPass: true,
                    description: "create translations in update",
                },
                {
                    data: {
                        ...meetingFixtures.minimal.update,
                        translationsUpdate: [{
                            id: "123456789012345678",
                            language: "en",
                            name: "Updated Meeting",
                        }],
                    },
                    shouldPass: true,
                    description: "update translations in update",
                },
                {
                    data: {
                        ...meetingFixtures.minimal.update,
                        translationsDelete: ["123456789012345678"],
                    },
                    shouldPass: true,
                    description: "delete translations in update",
                },
            ];

            await testValidationBatch(updateSchema, scenarios);
        });

        it("should allow all invite operations in update", async () => {
            const scenarios = [
                {
                    data: {
                        ...meetingFixtures.minimal.update,
                        invitesCreate: [{
                            id: "123456789012345678",
                            message: "New invitation",
                            meetingConnect: "123456789012345679",
                            userConnect: "123456789012345680",
                        }],
                    },
                    shouldPass: true,
                    description: "create invites in update",
                },
                {
                    data: {
                        ...meetingFixtures.minimal.update,
                        invitesUpdate: [{
                            id: "123456789012345678",
                            message: "Updated invitation message",
                        }],
                    },
                    shouldPass: true,
                    description: "update invites in update",
                },
                {
                    data: {
                        ...meetingFixtures.minimal.update,
                        invitesDelete: ["123456789012345678"],
                    },
                    shouldPass: true,
                    description: "delete invites in update",
                },
            ];

            await testValidationBatch(updateSchema, scenarios);
        });

        // Note: Schedule validation tests are excluded due to complex endTime validation logic
        // Schedule functionality will be tested separately in schedule.test.ts

        it("should not allow teamConnect updates", async () => {
            const updateData = {
                id: "123456789012345678",
                teamConnect: "123456789012345679",
            };

            const result = await testValidation(updateSchema, updateData, true);
            expect(result).to.not.have.property("teamConnect");
        });
    });

    describe("id validation", () => {
        const createSchema = meetingValidation.create({ omitFields: [] });

        it("should accept valid Snowflake IDs", async () => {
            const validIds = [
                "123456789012345678",
                "999999999999999999",
                "100000000000000000",
            ];

            const scenarios = validIds.map(id => ({
                data: {
                    ...meetingFixtures.minimal.create,
                    id,
                },
                shouldPass: true,
                description: `valid ID: ${id}`,
            }));

            await testValidationBatch(createSchema, scenarios);
        });

        it("should reject invalid IDs", async () => {
            const invalidIds = [
                "not-a-number",
                "abc123", // Contains letters
                "", // Empty string
                "-123", // Negative number
            ];

            const scenarios = invalidIds.map(id => ({
                data: {
                    ...meetingFixtures.minimal.create,
                    id,
                },
                shouldPass: false,
                expectedError: id === "" ? /required/i : /valid ID/i,
                description: `invalid ID: ${id}`,
            }));

            await testValidationBatch(createSchema, scenarios);
        });
    });

    describe("edge cases", () => {
        const createSchema = meetingValidation.create({ omitFields: [] });

        it("should handle boolean string conversions", async () => {
            await testValidation(
                createSchema,
                meetingFixtures.edgeCases.booleanConversions.create,
                true,
            );
        });

        it("should handle empty translations array", async () => {
            await testValidation(
                createSchema,
                meetingFixtures.edgeCases.emptyTranslations.create,
                true,
            );
        });

        it("should handle optional fields", async () => {
            await testValidation(
                createSchema,
                meetingFixtures.edgeCases.withOptionalFields.create,
                true,
            );
        });

        it("should handle complex creation with invites", async () => {
            await testValidation(
                createSchema,
                meetingFixtures.edgeCases.withScheduleAndInvites.create,
                true,
            );
        });

        it("should handle maximum length validations", async () => {
            await testValidation(
                createSchema,
                meetingFixtures.edgeCases.maxLengthTranslation.create,
                true,
            );
        });
    });
});
