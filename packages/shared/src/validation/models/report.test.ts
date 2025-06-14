import { describe, expect, it } from "vitest";
import { reportFixtures } from "./__test/fixtures/reportFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test/validationTestUtils.js";
import { reportValidation } from "./report.js";

describe("reportValidation", () => {
    // Run standard validation tests using shared fixtures
    runStandardValidationTests(
        reportValidation,
        reportFixtures,
        "report",
    );

    describe("create specific validations", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = reportValidation.create(defaultParams);

        it("should require id in create", async () => {
            const dataWithoutId = { ...reportFixtures.minimal.create };
            delete dataWithoutId.id;

            await testValidation(
                createSchema,
                dataWithoutId,
                false,
                /required/i,
            );
        });

        it("should require createdForType in create", async () => {
            const dataWithoutCreatedForType = {
                id: "123456789012345678",
                language: "en",
                reason: "Test reason",
                createdForConnect: "123456789012345679",
                // Missing required createdForType
            };

            await testValidation(
                createSchema,
                dataWithoutCreatedForType,
                false,
                /required/i,
            );
        });

        it("should require language in create", async () => {
            const dataWithoutLanguage = {
                id: "123456789012345678",
                createdForType: "User",
                reason: "Test reason",
                createdForConnect: "123456789012345679",
                // Missing required language
            };

            await testValidation(
                createSchema,
                dataWithoutLanguage,
                false,
                /required/i,
            );
        });

        it("should require reason in create", async () => {
            const dataWithoutReason = {
                id: "123456789012345678",
                createdForType: "User",
                language: "en",
                createdForConnect: "123456789012345679",
                // Missing required reason
            };

            await testValidation(
                createSchema,
                dataWithoutReason,
                false,
                /required/i,
            );
        });

        it("should require createdForConnect in create", async () => {
            await testValidation(
                createSchema,
                reportFixtures.invalid.missingCreatedFor.create,
                false,
                /required/i,
            );
        });

        it("should allow minimal create with required fields", async () => {
            await testValidation(
                createSchema,
                reportFixtures.minimal.create,
                true,
            );
        });

        it("should allow complete create with all fields", async () => {
            await testValidation(
                createSchema,
                reportFixtures.complete.create,
                true,
            );
        });

        describe("createdForType field", () => {
            it("should accept valid report types", async () => {
                const scenarios = [
                    {
                        data: reportFixtures.edgeCases.chatMessageReport.create,
                        shouldPass: true,
                        description: "ChatMessage report",
                    },
                    {
                        data: reportFixtures.edgeCases.commentReport.create,
                        shouldPass: true,
                        description: "Comment report",
                    },
                    {
                        data: reportFixtures.edgeCases.issueReport.create,
                        shouldPass: true,
                        description: "Issue report",
                    },
                    {
                        data: reportFixtures.edgeCases.resourceVersionReport.create,
                        shouldPass: true,
                        description: "ResourceVersion report",
                    },
                    {
                        data: reportFixtures.edgeCases.tagReport.create,
                        shouldPass: true,
                        description: "Tag report",
                    },
                    {
                        data: reportFixtures.edgeCases.teamReport.create,
                        shouldPass: true,
                        description: "Team report",
                    },
                    {
                        data: reportFixtures.edgeCases.userReport.create,
                        shouldPass: true,
                        description: "User report",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject invalid report types", async () => {
                await testValidation(
                    createSchema,
                    reportFixtures.invalid.invalidCreatedForType.create,
                    false,
                    /must be one of/i,
                );
            });
        });

        describe("reason field", () => {
            it("should be required", async () => {
                const dataWithoutReason = {
                    id: "123456789012345678",
                    createdForType: "User",
                    language: "en",
                    createdForConnect: "123456789012345679",
                    // Missing required reason
                };

                await testValidation(
                    createSchema,
                    dataWithoutReason,
                    false,
                    /required/i,
                );
            });

            it("should reject empty reason", async () => {
                await testValidation(
                    createSchema,
                    reportFixtures.invalid.emptyReason.create,
                    false,
                    /required/i, // Empty string becomes undefined
                );
            });

            it("should accept valid reason lengths", async () => {
                const scenarios = [
                    {
                        data: reportFixtures.edgeCases.minLengthReason.create,
                        shouldPass: true,
                        description: "minimum length reason",
                    },
                    {
                        data: reportFixtures.edgeCases.maxLengthReason.create,
                        shouldPass: true,
                        description: "maximum length reason",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should reject reason that exceeds maximum length", async () => {
                await testValidation(
                    createSchema,
                    reportFixtures.invalid.longReason.create,
                    false,
                    /over the limit/i,
                );
            });
        });

        describe("details field", () => {
            it("should be optional", async () => {
                await testValidation(
                    createSchema,
                    reportFixtures.edgeCases.withoutDetails.create,
                    true,
                );
            });

            it("should accept valid details", async () => {
                await testValidation(
                    createSchema,
                    reportFixtures.complete.create,
                    true,
                );
            });

            it("should accept maximum length details", async () => {
                await testValidation(
                    createSchema,
                    reportFixtures.edgeCases.maxLengthDetails.create,
                    true,
                );
            });

            it("should reject details that exceed maximum length", async () => {
                await testValidation(
                    createSchema,
                    reportFixtures.invalid.longDetails.create,
                    false,
                    /over the limit/i,
                );
            });
        });

        describe("language field", () => {
            it("should be required", async () => {
                const dataWithoutLanguage = {
                    id: "123456789012345678",
                    createdForType: "User",
                    reason: "Test reason",
                    createdForConnect: "123456789012345679",
                    // Missing required language
                };

                await testValidation(
                    createSchema,
                    dataWithoutLanguage,
                    false,
                    /required/i,
                );
            });

            it("should accept valid language codes", async () => {
                await testValidation(
                    createSchema,
                    reportFixtures.edgeCases.differentLanguages.create,
                    true,
                );
            });
        });

        describe("relationship fields", () => {
            it("should require createdForConnect", async () => {
                await testValidation(
                    createSchema,
                    reportFixtures.invalid.missingCreatedFor.create,
                    false,
                    /required/i,
                );
            });

            it("should accept valid relationship connections", async () => {
                await testValidation(
                    createSchema,
                    reportFixtures.minimal.create,
                    true,
                );
            });
        });

        describe("report scenarios", () => {
            it("should handle different types of reports", async () => {
                const scenarios = [
                    {
                        data: reportFixtures.edgeCases.chatMessageReport.create,
                        shouldPass: true,
                        description: "chat message report",
                    },
                    {
                        data: reportFixtures.edgeCases.commentReport.create,
                        shouldPass: true,
                        description: "comment report",
                    },
                    {
                        data: reportFixtures.edgeCases.resourceVersionReport.create,
                        shouldPass: true,
                        description: "resource version report",
                    },
                ];

                await testValidationBatch(createSchema, scenarios);
            });

            it("should handle reports with detailed descriptions", async () => {
                await testValidation(
                    createSchema,
                    reportFixtures.edgeCases.longDetailedReport.create,
                    true,
                );
            });

            it("should handle reports with different IDs", async () => {
                await testValidation(
                    createSchema,
                    reportFixtures.edgeCases.differentIds.create,
                    true,
                );
            });
        });
    });

    describe("update specific validations", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = reportValidation.update(defaultParams);

        it("should require id in update", async () => {
            await testValidation(
                updateSchema,
                reportFixtures.invalid.missingRequired.update,
                false,
                /required/i,
            );
        });

        it("should allow minimal updates with just id", async () => {
            await testValidation(
                updateSchema,
                reportFixtures.minimal.update,
                true,
            );
        });

        it("should allow complete updates", async () => {
            await testValidation(
                updateSchema,
                reportFixtures.complete.update,
                true,
            );
        });

        describe("field updates", () => {
            it("should allow updating details", async () => {
                const dataWithDetails = {
                    id: "123456789012345678",
                    details: "Updated report details with more information.",
                };

                await testValidation(updateSchema, dataWithDetails, true);
            });

            it("should allow updating language", async () => {
                const dataWithLanguage = {
                    id: "123456789012345678",
                    language: "es",
                };

                await testValidation(updateSchema, dataWithLanguage, true);
            });

            it("should allow updating reason", async () => {
                const dataWithReason = {
                    id: "123456789012345678",
                    reason: "Updated violation reason",
                };

                await testValidation(updateSchema, dataWithReason, true);
            });

            it("should allow updating all optional fields", async () => {
                await testValidation(
                    updateSchema,
                    reportFixtures.edgeCases.updateAllFields.update,
                    true,
                );
            });

            it("should not allow updating createdForType or createdForConnect", async () => {
                const dataWithRestrictedFields = {
                    id: "123456789012345678",
                    createdForType: "Comment",
                    createdForConnect: "123456789012345679",
                    details: "Updated details",
                };

                const result = await testValidation(updateSchema, dataWithRestrictedFields, true);
                expect(result).to.not.have.property("createdForType");
                expect(result).to.not.have.property("createdForConnect");
            });
        });

        describe("validation in updates", () => {
            it("should still validate reason length in update", async () => {
                const dataWithLongReason = {
                    id: "123456789012345678",
                    reason: "x".repeat(129), // Too long
                };

                await testValidation(
                    updateSchema,
                    dataWithLongReason,
                    false,
                    /over the limit/i,
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

            it("should handle empty reason in update", async () => {
                const dataWithEmptyReason = {
                    id: "123456789012345678",
                    reason: "", // Empty string becomes undefined, should pass since optional
                };

                const result = await testValidation(updateSchema, dataWithEmptyReason, true);
                expect(result).to.not.have.property("reason");
            });
        });

        describe("update scenarios", () => {
            it("should handle id-only updates", async () => {
                await testValidation(
                    updateSchema,
                    reportFixtures.edgeCases.updateOnlyId.update,
                    true,
                );
            });

            it("should handle partial field updates", async () => {
                const dataWithPartialUpdate = {
                    id: "123456789012345678",
                    details: "Only updating details field.",
                };

                await testValidation(updateSchema, dataWithPartialUpdate, true);
            });

            it("should handle language changes", async () => {
                const dataWithLanguageChange = {
                    id: "123456789012345678",
                    language: "fr",
                    reason: "Raison mise à jour",
                };

                await testValidation(updateSchema, dataWithLanguageChange, true);
            });
        });
    });

    describe("id validation", () => {
        const createSchema = reportValidation.create({ omitFields: [] });

        it("should accept valid Snowflake IDs", async () => {
            const validIds = [
                "123456789012345678",
                "999999999999999999",
                "100000000000000000",
            ];

            const scenarios = validIds.map(id => ({
                data: {
                    ...reportFixtures.minimal.create,
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
                    { ...reportFixtures.minimal.create, id },
                    false,
                    error,
                );
            }
        });
    });

    describe("type conversions", () => {
        const createSchema = reportValidation.create({ omitFields: [] });

        it("should handle ID conversion", async () => {
            const dataWithNumberId = {
                id: 123456789012345, // Smaller number to avoid precision issues
                createdForType: "User",
                language: "en",
                reason: "ID conversion test",
                createdForConnect: "123456789012345679",
            };

            const result = await testValidation(
                createSchema,
                dataWithNumberId,
                true,
            );
            expect(result.id).to.be.a("string");
            expect(result.id).toBe("123456789012345");
        });
    });

    describe("edge cases", () => {
        const createSchema = reportValidation.create({ omitFields: [] });
        const updateSchema = reportValidation.update({ omitFields: [] });

        it("should handle minimal report creation", async () => {
            await testValidation(
                createSchema,
                reportFixtures.minimal.create,
                true,
            );
        });

        it("should handle empty update operations", async () => {
            await testValidation(
                updateSchema,
                reportFixtures.edgeCases.updateOnlyId.update,
                true,
            );
        });

        it("should handle all report types", async () => {
            const reportTypes = [
                "ChatMessage",
                "Comment",
                "Issue",
                "ResourceVersion",
                "Tag",
                "Team",
                "User",
            ];

            for (const reportType of reportTypes) {
                const data = {
                    id: "123456789012345678",
                    createdForType: reportType,
                    language: "en",
                    reason: `Report for ${reportType}`,
                    createdForConnect: "123456789012345679",
                };

                await testValidation(createSchema, data, true);
            }
        });

        it("should handle different string lengths", async () => {
            const scenarios = [
                {
                    data: reportFixtures.edgeCases.minLengthReason.create,
                    shouldPass: true,
                    description: "minimum reason length",
                },
                {
                    data: reportFixtures.edgeCases.maxLengthReason.create,
                    shouldPass: true,
                    description: "maximum reason length",
                },
                {
                    data: reportFixtures.edgeCases.maxLengthDetails.create,
                    shouldPass: true,
                    description: "maximum details length",
                },
            ];

            await testValidationBatch(createSchema, scenarios);
        });

        it("should handle reports in different languages", async () => {
            const languages = ["en", "es", "fr", "de", "it"];

            for (const language of languages) {
                const data = {
                    id: "123456789012345678",
                    createdForType: "User",
                    language,
                    reason: "Test reason",
                    createdForConnect: "123456789012345679",
                };

                await testValidation(createSchema, data, true);
            }
        });
    });
});
