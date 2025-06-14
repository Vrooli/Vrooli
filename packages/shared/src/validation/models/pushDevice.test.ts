import { describe, expect, it } from "vitest";
import { pushDeviceFixtures } from "./__test/fixtures/pushDeviceFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test/validationTestUtils.js";
import { pushDeviceValidation } from "./pushDevice.js";

describe("pushDeviceValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        pushDeviceValidation,
        pushDeviceFixtures,
        "pushDevice",
    );

    describe("create specific validations", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = pushDeviceValidation.create(defaultParams);

        it("should require endpoint in create", async () => {
            const dataWithoutEndpoint = {
                keys: {
                    p256dh: "VALID-P256DH-KEY",
                    auth: "VALID-AUTH-KEY",
                },
                name: "Test Device",
            };

            await testValidation(
                createSchema,
                dataWithoutEndpoint,
                false,
                /required/i,
            );
        });

        it("should require keys in create", async () => {
            await testValidation(
                createSchema,
                pushDeviceFixtures.invalid.missingRequired.create,
                false,
                /required/i,
            );
        });

        it("should allow minimal create with required fields", async () => {
            await testValidation(
                createSchema,
                pushDeviceFixtures.minimal.create,
                true,
            );
        });

        it("should allow complete create with all fields", async () => {
            await testValidation(
                createSchema,
                pushDeviceFixtures.complete.create,
                true,
            );
        });

        describe("endpoint field", () => {
            it("should accept valid HTTPS URLs", async () => {
                const scenarios = [
                    {
                        data: pushDeviceFixtures.minimal.create,
                        shouldPass: true,
                        description: "FCM endpoint",
                    },
                    {
                        data: pushDeviceFixtures.edgeCases.differentEndpoints.create,
                        shouldPass: true,
                        description: "different push service",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject invalid URLs", async () => {
                await testValidation(
                    createSchema,
                    pushDeviceFixtures.invalid.invalidEndpoint.create,
                    false,
                    /Must be a URL/i,
                );
            });

            it("should accept HTTP URLs in dev environment", async () => {
                const devSchema = pushDeviceValidation.create({ omitFields: [], env: "dev" });

                await testValidation(
                    devSchema,
                    pushDeviceFixtures.edgeCases.httpEndpoint.create,
                    true,
                );
            });

            it("should accept HTTP URLs in production environment", async () => {
                const prodSchema = pushDeviceValidation.create({ omitFields: [], env: "production" });

                await testValidation(
                    prodSchema,
                    pushDeviceFixtures.edgeCases.httpEndpoint.create,
                    true,
                );
            });
        });

        describe("keys field", () => {
            it("should require keys object", async () => {
                await testValidation(
                    createSchema,
                    pushDeviceFixtures.invalid.invalidKeys.create,
                    false,
                    /required/i,
                );
            });

            it("should require p256dh in keys", async () => {
                await testValidation(
                    createSchema,
                    pushDeviceFixtures.invalid.missingKeysP256dh.create,
                    false,
                    /required/i,
                );
            });

            it("should require auth in keys", async () => {
                await testValidation(
                    createSchema,
                    pushDeviceFixtures.invalid.missingKeysAuth.create,
                    false,
                    /required/i,
                );
            });

            it("should accept valid keys with both p256dh and auth", async () => {
                await testValidation(
                    createSchema,
                    pushDeviceFixtures.minimal.create,
                    true,
                );
            });

            it("should accept long keys near maximum length", async () => {
                await testValidation(
                    createSchema,
                    pushDeviceFixtures.edgeCases.longKeys.create,
                    true,
                );
            });
        });

        describe("expires field", () => {
            it("should be optional", async () => {
                await testValidation(
                    createSchema,
                    pushDeviceFixtures.edgeCases.withoutOptionalFields.create,
                    true,
                );
            });

            it("should accept zero expires", async () => {
                await testValidation(
                    createSchema,
                    pushDeviceFixtures.edgeCases.withZeroExpires.create,
                    true,
                );
            });

            it("should accept large positive expires values", async () => {
                await testValidation(
                    createSchema,
                    pushDeviceFixtures.edgeCases.withLargeExpires.create,
                    true,
                );
            });

            it("should reject negative expires values", async () => {
                await testValidation(
                    createSchema,
                    pushDeviceFixtures.invalid.invalidExpires.create,
                    false,
                    /Minimum value is 0/i,
                );
            });
        });

        describe("name field", () => {
            it("should be optional", async () => {
                await testValidation(
                    createSchema,
                    pushDeviceFixtures.edgeCases.withoutOptionalFields.create,
                    true,
                );
            });

            it("should accept valid device names", async () => {
                const scenarios = [
                    {
                        data: pushDeviceFixtures.complete.create,
                        shouldPass: true,
                        description: "simple name",
                    },
                    {
                        data: pushDeviceFixtures.edgeCases.longDeviceName.create,
                        shouldPass: true,
                        description: "long name",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });
        });

        describe("push notification scenarios", () => {
            it("should handle FCM endpoints", async () => {
                await testValidation(
                    createSchema,
                    pushDeviceFixtures.minimal.create,
                    true,
                );
            });

            it("should handle different push services", async () => {
                await testValidation(
                    createSchema,
                    pushDeviceFixtures.edgeCases.differentEndpoints.create,
                    true,
                );
            });

            it("should handle devices with expiration", async () => {
                await testValidation(
                    createSchema,
                    pushDeviceFixtures.complete.create,
                    true,
                );
            });

            it("should handle devices without expiration", async () => {
                await testValidation(
                    createSchema,
                    pushDeviceFixtures.edgeCases.withoutOptionalFields.create,
                    true,
                );
            });
        });
    });

    describe("update specific validations", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = pushDeviceValidation.update(defaultParams);

        it("should require id in update", async () => {
            await testValidation(
                updateSchema,
                pushDeviceFixtures.invalid.missingRequired.update,
                false,
                /required/i,
            );
        });

        it("should allow minimal updates with just id", async () => {
            await testValidation(
                updateSchema,
                pushDeviceFixtures.minimal.update,
                true,
            );
        });

        it("should allow updating only name", async () => {
            await testValidation(
                updateSchema,
                pushDeviceFixtures.complete.update,
                true,
            );
        });

        describe("name field updates", () => {
            it("should allow updating device name", async () => {
                await testValidation(
                    updateSchema,
                    pushDeviceFixtures.edgeCases.updateWithNewName.update,
                    true,
                );
            });

            it("should allow setting same name", async () => {
                await testValidation(
                    updateSchema,
                    pushDeviceFixtures.edgeCases.updateWithSameName.update,
                    true,
                );
            });

            it("should not allow updating endpoint or keys", async () => {
                const dataWithEndpoint = {
                    id: "123456789012345678",
                    endpoint: "https://new-endpoint.com/path",
                    keys: {
                        p256dh: "NEW-P256DH-KEY",
                        auth: "NEW-AUTH-KEY",
                    },
                };

                const result = await testValidation(updateSchema, dataWithEndpoint, true);
                expect(result).to.not.have.property("endpoint");
                expect(result).to.not.have.property("keys");
            });

            it("should not allow updating expires", async () => {
                const dataWithExpires = {
                    id: "123456789012345678",
                    name: "Updated Device",
                    expires: 7200,
                };

                const result = await testValidation(updateSchema, dataWithExpires, true);
                expect(result).to.not.have.property("expires");
            });
        });

        describe("update scenarios", () => {
            it("should handle id-only updates", async () => {
                await testValidation(
                    updateSchema,
                    pushDeviceFixtures.edgeCases.updateOnlyId.update,
                    true,
                );
            });

            it("should handle name updates", async () => {
                const scenarios = [
                    {
                        data: pushDeviceFixtures.edgeCases.updateWithSameName.update,
                        shouldPass: true,
                        description: "same name",
                    },
                    {
                        data: pushDeviceFixtures.edgeCases.updateWithNewName.update,
                        shouldPass: true,
                        description: "new name",
                    },
                ];

                await testValidationBatch(updateSchema, scenarios);
            });
        });
    });

    describe("id validation", () => {
        const updateSchema = pushDeviceValidation.update({ omitFields: [] });

        it("should accept valid Snowflake IDs in update", async () => {
            const validIds = [
                "123456789012345678",
                "999999999999999999",
                "100000000000000000",
            ];

            const scenarios = validIds.map(id => ({
                data: {
                    ...pushDeviceFixtures.minimal.update,
                    id,
                },
                shouldPass: true,
                description: `valid ID: ${id}`,
            }));

            await testValidationBatch(updateSchema, scenarios);
        });

        it("should reject invalid IDs in update", async () => {
            const invalidIds = [
                { id: "not-a-number", error: /Must be a valid ID/i },
                { id: "abc123", error: /Must be a valid ID/i },
                { id: "", error: /required/i },
                { id: "-123", error: /Must be a valid ID/i },
                { id: "0", error: /Must be a valid ID/i },
            ];

            for (const { id, error } of invalidIds) {
                await testValidation(
                    updateSchema,
                    { ...pushDeviceFixtures.minimal.update, id },
                    false,
                    error,
                );
            }
        });
    });

    describe("type conversions", () => {
        const createSchema = pushDeviceValidation.create({ omitFields: [] });

        it("should handle URL string validation", async () => {
            const dataWithValidUrl = {
                endpoint: "https://converted-endpoint.com/path",
                keys: {
                    p256dh: "CONVERTED-P256DH-KEY",
                    auth: "CONVERTED-AUTH-KEY",
                },
            };

            const result = await testValidation(
                createSchema,
                dataWithValidUrl,
                true,
            );
            expect(result.endpoint).to.be.a("string");
            expect(result.endpoint).toBe("https://converted-endpoint.com/path");
        });

        it("should handle number conversion for expires", async () => {
            const dataWithStringNumber = {
                endpoint: "https://number-conversion.com/path",
                expires: "3600", // String number
                keys: {
                    p256dh: "NUMBER-CONVERSION-P256DH-KEY",
                    auth: "NUMBER-CONVERSION-AUTH-KEY",
                },
            };

            const result = await testValidation(
                createSchema,
                dataWithStringNumber,
                true,
            );
            expect(result.expires).to.be.a("number");
            expect(result.expires).toBe(3600);
        });
    });

    describe("edge cases", () => {
        const createSchema = pushDeviceValidation.create({ omitFields: [] });
        const updateSchema = pushDeviceValidation.update({ omitFields: [] });

        it("should handle minimal device registration", async () => {
            await testValidation(
                createSchema,
                pushDeviceFixtures.minimal.create,
                true,
            );
        });

        it("should handle empty update operations", async () => {
            await testValidation(
                updateSchema,
                pushDeviceFixtures.edgeCases.updateOnlyId.update,
                true,
            );
        });

        it("should handle various push service endpoints", async () => {
            const scenarios = [
                {
                    data: pushDeviceFixtures.minimal.create,
                    shouldPass: true,
                    description: "FCM endpoint",
                },
                {
                    data: pushDeviceFixtures.edgeCases.differentEndpoints.create,
                    shouldPass: true,
                    description: "alternative push service",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle devices with and without expiration", async () => {
            const scenarios = [
                {
                    data: pushDeviceFixtures.edgeCases.withZeroExpires.create,
                    shouldPass: true,
                    description: "zero expiration",
                },
                {
                    data: pushDeviceFixtures.edgeCases.withLargeExpires.create,
                    shouldPass: true,
                    description: "large expiration",
                },
                {
                    data: pushDeviceFixtures.edgeCases.withoutOptionalFields.create,
                    shouldPass: true,
                    description: "no expiration",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });
    });
});
