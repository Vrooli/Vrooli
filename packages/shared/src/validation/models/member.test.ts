import { describe, expect, it } from "vitest";
import { memberFixtures } from "./__test/fixtures/memberFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test/validationTestUtils.js";
import { memberValidation } from "./member.js";

describe("memberValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        memberValidation,
        memberFixtures,
        "member",
    );

    describe("update specific validations", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = memberValidation.update(defaultParams);

        it("should require id in update", async () => {
            await testValidation(
                updateSchema,
                memberFixtures.invalid.missingRequired.update,
                false,
                /required/i,
            );
        });

        it("should allow minimal updates with just id", async () => {
            await testValidation(
                updateSchema,
                memberFixtures.minimal.update,
                true,
            );
        });

        it("should allow complete updates", async () => {
            await testValidation(
                updateSchema,
                memberFixtures.complete.update,
                true,
            );
        });

        describe("isAdmin field", () => {
            it("should accept boolean values for isAdmin", async () => {
                const scenarios = [
                    {
                        data: { ...memberFixtures.minimal.update, isAdmin: true },
                        shouldPass: true,
                        description: "isAdmin true",
                    },
                    {
                        data: { ...memberFixtures.minimal.update, isAdmin: false },
                        shouldPass: true,
                        description: "isAdmin false",
                    },
                ];

                await testValidationBatch(updateSchema, scenarios);
            });

            it("should be optional in update", async () => {
                await testValidation(updateSchema, memberFixtures.minimal.update, true);
            });

            it("should convert string values to boolean", async () => {
                const scenarios = [
                    {
                        data: { ...memberFixtures.minimal.update, isAdmin: "true" },
                        shouldPass: true,
                        description: "string 'true' converts to boolean",
                    },
                    {
                        data: { ...memberFixtures.minimal.update, isAdmin: "false" },
                        shouldPass: true,
                        description: "string 'false' converts to boolean",
                    },
                    {
                        data: { ...memberFixtures.minimal.update, isAdmin: "yes" },
                        shouldPass: true,
                        description: "string 'yes' converts to boolean",
                    },
                ];

                await testValidationBatch(updateSchema, scenarios);
            });
        });

        describe("permissions field", () => {
            it("should accept valid permission strings", async () => {
                const scenarios = [
                    {
                        data: { ...memberFixtures.minimal.update, permissions: JSON.stringify(["Read"]) },
                        shouldPass: true,
                        description: "single permission",
                    },
                    {
                        data: { ...memberFixtures.minimal.update, permissions: JSON.stringify(["Create", "Read", "Update", "Delete"]) },
                        shouldPass: true,
                        description: "multiple permissions",
                    },
                    {
                        data: { ...memberFixtures.minimal.update, permissions: JSON.stringify([]) },
                        shouldPass: true,
                        description: "empty permissions array",
                    },
                ];

                await testValidationBatch(updateSchema, scenarios);
            });

            it("should be optional in update", async () => {
                await testValidation(updateSchema, memberFixtures.minimal.update, true);
            });

            it("should reject non-string values", async () => {
                const invalidData = {
                    ...memberFixtures.minimal.update,
                    permissions: ["Read"], // Should be JSON string, not array
                };

                await testValidation(
                    updateSchema,
                    invalidData,
                    false,
                    /val\.trim is not a function/i,
                );
            });
        });

        it("should allow partial updates", async () => {
            const scenarios = [
                {
                    data: memberFixtures.edgeCases.onlyAdminChange.update,
                    shouldPass: true,
                    description: "update only isAdmin",
                },
                {
                    data: memberFixtures.edgeCases.onlyPermissionsChange.update,
                    shouldPass: true,
                    description: "update only permissions",
                },
            ];

            await testValidationBatch(updateSchema, scenarios);
        });
    });

    describe("no create operation", () => {
        it("should not have a create method", () => {
            expect(memberValidation.create).to.be.undefined;
        });
    });

    describe("id validation", () => {
        const updateSchema = memberValidation.update({ omitFields: [] });

        it("should accept valid Snowflake IDs", async () => {
            const validIds = [
                "123456789012345678",
                "999999999999999999",
                "100000000000000000",
            ];

            const scenarios = validIds.map(id => ({
                data: {
                    ...memberFixtures.minimal.update,
                    id,
                },
                shouldPass: true,
                description: `valid ID: ${id}`,
            }));

            await testValidationBatch(updateSchema, scenarios);
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
                    ...memberFixtures.minimal.update,
                    id,
                },
                shouldPass: false,
                expectedError: id === "" ? /required/i : /valid ID/i,
                description: `invalid ID: ${id}`,
            }));

            await testValidationBatch(updateSchema, scenarios);
        });
    });

    describe("edge cases", () => {
        const updateSchema = memberValidation.update({ omitFields: [] });

        it("should handle empty permissions array", async () => {
            await testValidation(
                updateSchema,
                memberFixtures.edgeCases.emptyPermissions.update,
                true,
            );
        });

        it("should handle full permissions array", async () => {
            await testValidation(
                updateSchema,
                memberFixtures.edgeCases.fullPermissions.update,
                true,
            );
        });
    });
});
