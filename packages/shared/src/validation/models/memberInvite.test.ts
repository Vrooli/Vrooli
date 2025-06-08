import { describe, expect, it } from "vitest";
import { memberInviteFixtures } from "./__test/fixtures/memberInviteFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test/validationTestUtils.js";
import { memberInviteValidation } from "./memberInvite.js";

describe("memberInviteValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        memberInviteValidation,
        memberInviteFixtures,
        "memberInvite",
    );

    describe("specific field validations", () => {
        const defaultParams = { omitFields: [] };

        describe("message field", () => {
            const createSchema = memberInviteValidation.create(defaultParams);
            const updateSchema = memberInviteValidation.update(defaultParams);

            it("should be optional in both create and update", async () => {
                await testValidation(
                    createSchema,
                    memberInviteFixtures.minimal.create,
                    true,
                );

                await testValidation(
                    updateSchema,
                    memberInviteFixtures.minimal.update,
                    true,
                );
            });

            it("should accept valid messages", async () => {
                await testValidation(
                    createSchema,
                    memberInviteFixtures.complete.create,
                    true,
                );

                await testValidation(
                    updateSchema,
                    memberInviteFixtures.complete.update,
                    true,
                );
            });

            it("should trim whitespace from message", async () => {
                const result = await createSchema.validate(
                    memberInviteFixtures.edgeCases.whitespaceMessage.create,
                );
                expect(result.message).to.equal("Invitation message with whitespace");
            });

            it("should handle empty string message", async () => {
                const result = await createSchema.validate(
                    memberInviteFixtures.edgeCases.emptyMessage.create,
                );
                // Empty string should be removed by removeEmptyString()
                expect(result.message).to.be.undefined;
            });

            it("should accept long messages", async () => {
                await testValidation(
                    createSchema,
                    memberInviteFixtures.edgeCases.longMessage.create,
                    true,
                );
            });

            it("should reject non-string message types", async () => {
                await testValidation(
                    createSchema,
                    {
                        ...memberInviteFixtures.minimal.create,
                        message: 123,
                    },
                    false,
                );

                await testValidation(
                    createSchema,
                    {
                        ...memberInviteFixtures.minimal.create,
                        message: false,
                    },
                    false,
                );
            });
        });

        describe("willBeAdmin field", () => {
            const createSchema = memberInviteValidation.create(defaultParams);
            const updateSchema = memberInviteValidation.update(defaultParams);

            it("should be optional in both create and update", async () => {
                await testValidation(
                    createSchema,
                    memberInviteFixtures.minimal.create,
                    true,
                );

                await testValidation(
                    updateSchema,
                    memberInviteFixtures.minimal.update,
                    true,
                );
            });

            it("should accept boolean values", async () => {
                const scenarios = memberInviteFixtures.edgeCases.booleanEdgeCases.map(data => ({
                    data,
                    shouldPass: true,
                    description: `willBeAdmin: ${data.willBeAdmin}`,
                }));

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject non-boolean values", async () => {
                const invalidValues = [
                    "true",
                    "false",
                    1,
                    0,
                    "yes",
                    "no",
                    null,
                ];

                for (const invalidValue of invalidValues) {
                    await testValidation(
                        createSchema,
                        {
                            ...memberInviteFixtures.minimal.create,
                            willBeAdmin: invalidValue,
                        },
                        false,
                    );
                }
            });

            it("should work with different permission combinations", async () => {
                await testValidation(
                    createSchema,
                    memberInviteFixtures.edgeCases.adminWithoutPermissions.create,
                    true,
                );

                await testValidation(
                    createSchema,
                    memberInviteFixtures.edgeCases.nonAdminWithPermissions.create,
                    true,
                );
            });
        });

        describe("willHavePermissions field", () => {
            const createSchema = memberInviteValidation.create(defaultParams);
            const updateSchema = memberInviteValidation.update(defaultParams);

            it("should be optional in both create and update", async () => {
                await testValidation(
                    createSchema,
                    memberInviteFixtures.minimal.create,
                    true,
                );

                await testValidation(
                    updateSchema,
                    memberInviteFixtures.minimal.update,
                    true,
                );
            });

            it("should accept various permission formats", async () => {
                const scenarios = memberInviteFixtures.edgeCases.permissionsEdgeCases.map(data => ({
                    data,
                    shouldPass: true,
                    description: `permissions: ${data.willHavePermissions}`,
                }));

                await testValidationBatch(createSchema, scenarios);
            });

            it("should accept empty permissions", async () => {
                await testValidation(
                    createSchema,
                    memberInviteFixtures.edgeCases.emptyPermissions.create,
                    true,
                );
            });

            it("should accept complex permissions", async () => {
                await testValidation(
                    createSchema,
                    memberInviteFixtures.edgeCases.complexPermissions.create,
                    true,
                );
            });

            it("should reject non-string permission values", async () => {
                const invalidValues = [
                    123,
                    false,
                    ["read", "write"], // Should be JSON string, not array
                    { role: "admin" }, // Should be JSON string, not object
                ];

                for (const invalidValue of invalidValues) {
                    await testValidation(
                        createSchema,
                        {
                            ...memberInviteFixtures.minimal.create,
                            willHavePermissions: invalidValue,
                        },
                        false,
                    );
                }
            });
        });

        describe("required relationship fields", () => {
            const createSchema = memberInviteValidation.create(defaultParams);

            it("should require teamConnect in create", async () => {
                await testValidation(
                    createSchema,
                    memberInviteFixtures.invalid.missingTeamConnect.create,
                    false,
                    /required/i,
                );
            });

            it("should require userConnect in create", async () => {
                await testValidation(
                    createSchema,
                    memberInviteFixtures.invalid.missingUserConnect.create,
                    false,
                    /required/i,
                );
            });

            it("should reject invalid teamConnect ID", async () => {
                await testValidation(
                    createSchema,
                    memberInviteFixtures.invalid.invalidTeamConnect.create,
                    false,
                );
            });

            it("should reject invalid userConnect ID", async () => {
                await testValidation(
                    createSchema,
                    memberInviteFixtures.invalid.invalidUserConnect.create,
                    false,
                );
            });
        });

        describe("id field validation", () => {
            const createSchema = memberInviteValidation.create(defaultParams);
            const updateSchema = memberInviteValidation.update(defaultParams);

            it("should accept valid snowflake IDs", async () => {
                const validIds = [
                    "123456789012345678",
                    "123456789012345679",
                    "999999999999999999",
                ];

                for (const validId of validIds) {
                    await testValidation(
                        createSchema,
                        {
                            ...memberInviteFixtures.minimal.create,
                            id: validId,
                        },
                        true,
                    );

                    await testValidation(
                        updateSchema,
                        {
                            ...memberInviteFixtures.minimal.update,
                            id: validId,
                        },
                        true,
                    );
                }
            });

            it("should reject invalid IDs", async () => {
                const invalidIds = [
                    "not-a-snowflake",
                    "123",
                    "",
                    null,
                    undefined,
                    123,
                ];

                for (const invalidId of invalidIds) {
                    await testValidation(
                        createSchema,
                        {
                            ...memberInviteFixtures.minimal.create,
                            id: invalidId,
                        },
                        false,
                    );

                    await testValidation(
                        updateSchema,
                        {
                            ...memberInviteFixtures.minimal.update,
                            id: invalidId,
                        },
                        false,
                    );
                }
            });

            it("should be required in both create and update", async () => {
                await testValidation(
                    createSchema,
                    memberInviteFixtures.invalid.missingRequired.create,
                    false,
                    /required/i,
                );

                await testValidation(
                    updateSchema,
                    memberInviteFixtures.invalid.missingRequired.update,
                    false,
                    /required/i,
                );
            });
        });

        describe("update operation", () => {
            const updateSchema = memberInviteValidation.update(defaultParams);

            it("should not require team and user connections in update", async () => {
                await testValidation(
                    updateSchema,
                    memberInviteFixtures.edgeCases.updateWithoutConnections.update,
                    true,
                );
            });

            it("should accept all optional field updates", async () => {
                await testValidation(
                    updateSchema,
                    memberInviteFixtures.complete.update,
                    true,
                );
            });

            it("should allow updating individual fields", async () => {
                const scenarios = [
                    {
                        data: {
                            id: "123456789012345678",
                            message: "Only message updated",
                        },
                        shouldPass: true,
                        description: "message only",
                    },
                    {
                        data: {
                            id: "123456789012345678",
                            willBeAdmin: true,
                        },
                        shouldPass: true,
                        description: "willBeAdmin only",
                    },
                    {
                        data: {
                            id: "123456789012345678",
                            willHavePermissions: JSON.stringify(["read"]),
                        },
                        shouldPass: true,
                        description: "permissions only",
                    },
                ];

                await testValidationBatch(updateSchema, scenarios);
            });
        });
    });
});
