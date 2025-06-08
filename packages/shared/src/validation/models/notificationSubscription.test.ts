import { describe, expect, it } from "vitest";
import { SubscribableObject } from "../../api/types.js";
import { notificationSubscriptionFixtures } from "./__test/fixtures/notificationSubscriptionFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test/validationTestUtils.js";
import { notificationSubscriptionValidation } from "./notificationSubscription.js";

describe("notificationSubscriptionValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        notificationSubscriptionValidation,
        notificationSubscriptionFixtures,
        "notificationSubscription",
    );

    describe("specific field validations", () => {
        const defaultParams = { omitFields: [] };

        describe("objectType field", () => {
            const createSchema = notificationSubscriptionValidation.create(defaultParams);

            it("should accept all valid SubscribableObject enum values", async () => {
                const validValues = Object.values(SubscribableObject);
                const scenarios = validValues.map(value => ({
                    data: {
                        ...notificationSubscriptionFixtures.minimal.create,
                        objectType: value,
                    },
                    shouldPass: true,
                    description: `objectType: ${value}`,
                }));

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject invalid SubscribableObject values", async () => {
                const scenarios = [
                    {
                        data: {
                            ...notificationSubscriptionFixtures.minimal.create,
                            objectType: "InvalidType",
                        },
                        shouldPass: false,
                        expectedError: /must be one of the following values/i,
                        description: "invalid enum value",
                    },
                    {
                        data: {
                            ...notificationSubscriptionFixtures.minimal.create,
                            objectType: 123,
                        },
                        shouldPass: false,
                        description: "number instead of enum",
                    },
                    {
                        data: {
                            ...notificationSubscriptionFixtures.minimal.create,
                            objectType: null,
                        },
                        shouldPass: false,
                        description: "null instead of enum",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should be required in create", async () => {
                await testValidation(
                    createSchema,
                    notificationSubscriptionFixtures.invalid.missingObjectType.create,
                    false,
                    /required/i,
                );
            });

            it("should not be present in update", async () => {
                const updateSchema = notificationSubscriptionValidation.update(defaultParams);

                // Update schema should not include objectType field
                const result = await updateSchema.validate(notificationSubscriptionFixtures.minimal.update);
                expect(result).to.not.have.property("objectType");
            });

            it("should work with all supported object types", async () => {
                await testValidationBatch(
                    createSchema,
                    notificationSubscriptionFixtures.edgeCases.allObjectTypes.map(data => ({
                        data,
                        shouldPass: true,
                        description: `ObjectType: ${data.objectType}`,
                    })),
                );
            });
        });

        describe("silent field", () => {
            const createSchema = notificationSubscriptionValidation.create(defaultParams);
            const updateSchema = notificationSubscriptionValidation.update(defaultParams);

            it("should be optional in both create and update", async () => {
                await testValidation(
                    createSchema,
                    notificationSubscriptionFixtures.edgeCases.withoutSilent.create,
                    true,
                );

                await testValidation(
                    updateSchema,
                    notificationSubscriptionFixtures.edgeCases.updateWithoutSilent.update,
                    true,
                );
            });

            it("should accept boolean values", async () => {
                const scenarios = notificationSubscriptionFixtures.edgeCases.silentOptions.map(data => ({
                    data,
                    shouldPass: true,
                    description: `silent: ${data.silent}`,
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
                    undefined,
                ];

                for (const invalidValue of invalidValues) {
                    await testValidation(
                        createSchema,
                        {
                            ...notificationSubscriptionFixtures.minimal.create,
                            silent: invalidValue,
                        },
                        false,
                    );
                }
            });

            it("should work in update operations", async () => {
                await testValidation(
                    updateSchema,
                    notificationSubscriptionFixtures.edgeCases.updateSilentOnly.update,
                    true,
                );
            });
        });

        describe("object relationship field", () => {
            const createSchema = notificationSubscriptionValidation.create(defaultParams);

            it("should require objectConnect in create", async () => {
                await testValidation(
                    createSchema,
                    notificationSubscriptionFixtures.invalid.missingObjectConnect.create,
                    false,
                    /required/i,
                );
            });

            it("should reject invalid objectConnect ID", async () => {
                await testValidation(
                    createSchema,
                    notificationSubscriptionFixtures.invalid.invalidObjectConnect.create,
                    false,
                );
            });

            it("should accept various valid object connections", async () => {
                await testValidationBatch(
                    createSchema,
                    notificationSubscriptionFixtures.edgeCases.differentObjectConnections.map(data => ({
                        data,
                        shouldPass: true,
                        description: `ObjectType: ${data.objectType} with ID: ${data.objectConnect}`,
                    })),
                );
            });
        });

        describe("id field validation", () => {
            const createSchema = notificationSubscriptionValidation.create(defaultParams);
            const updateSchema = notificationSubscriptionValidation.update(defaultParams);

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
                            ...notificationSubscriptionFixtures.minimal.create,
                            id: validId,
                        },
                        true,
                    );

                    await testValidation(
                        updateSchema,
                        {
                            ...notificationSubscriptionFixtures.minimal.update,
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
                            ...notificationSubscriptionFixtures.minimal.create,
                            id: invalidId,
                        },
                        false,
                    );

                    await testValidation(
                        updateSchema,
                        {
                            ...notificationSubscriptionFixtures.minimal.update,
                            id: invalidId,
                        },
                        false,
                    );
                }
            });

            it("should be required in both create and update", async () => {
                await testValidation(
                    createSchema,
                    notificationSubscriptionFixtures.invalid.missingRequired.create,
                    false,
                    /required/i,
                );

                await testValidation(
                    updateSchema,
                    notificationSubscriptionFixtures.invalid.missingRequired.update,
                    false,
                    /required/i,
                );
            });
        });

        describe("update operation specifics", () => {
            const updateSchema = notificationSubscriptionValidation.update(defaultParams);

            it("should not require objectType or objectConnect in update", async () => {
                await testValidation(
                    updateSchema,
                    notificationSubscriptionFixtures.minimal.update,
                    true,
                );
            });

            it("should allow only silent field updates", async () => {
                await testValidation(
                    updateSchema,
                    notificationSubscriptionFixtures.edgeCases.updateSilentOnly.update,
                    true,
                );
            });

            it("should allow updates without any optional fields", async () => {
                await testValidation(
                    updateSchema,
                    notificationSubscriptionFixtures.edgeCases.updateWithoutSilent.update,
                    true,
                );
            });

            it("should not include objectType in update result", async () => {
                const result = await updateSchema.validate(notificationSubscriptionFixtures.minimal.update);

                // Should only have id and optionally silent
                expect(result).to.have.property("id");
                expect(result).to.not.have.property("objectType");
                expect(result).to.not.have.property("objectConnect");
            });
        });

        describe("create operation specifics", () => {
            const createSchema = notificationSubscriptionValidation.create(defaultParams);

            it("should require all mandatory fields", async () => {
                const scenarios = [
                    {
                        data: notificationSubscriptionFixtures.invalid.missingRequired.create,
                        shouldPass: false,
                        expectedError: /required/i,
                        description: "missing all required fields",
                    },
                    {
                        data: notificationSubscriptionFixtures.invalid.missingObjectType.create,
                        shouldPass: false,
                        expectedError: /required/i,
                        description: "missing objectType",
                    },
                    {
                        data: notificationSubscriptionFixtures.invalid.missingObjectConnect.create,
                        shouldPass: false,
                        expectedError: /required/i,
                        description: "missing objectConnect",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should accept minimal valid data", async () => {
                await testValidation(
                    createSchema,
                    notificationSubscriptionFixtures.minimal.create,
                    true,
                );
            });

            it("should accept complete valid data", async () => {
                await testValidation(
                    createSchema,
                    notificationSubscriptionFixtures.complete.create,
                    true,
                );
            });
        });
    });
});
