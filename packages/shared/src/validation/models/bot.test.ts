import { describe, expect, it } from "vitest";
import {
    botFixtures,
    botTestDataFactory,
    botTranslationFixtures,
    botTranslationTestDataFactory,
} from "./__test/fixtures/botFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test/validationTestUtils.js";
import { botTranslationValidation, botValidation } from "./bot.js";

describe("botValidation", () => {
    // Run standard test suite
    runStandardValidationTests(botValidation, botFixtures, "bot");

    // Bot-specific tests
    describe("bot-specific validation", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = botValidation.create(defaultParams);
        const updateSchema = botValidation.update(defaultParams);

        describe("create validation", () => {
            it("should require id, botSettings, isBotDepictingPerson, and name", async () => {
                const invalidData = {
                    handle: "testbot",
                    isPrivate: false,
                };
                await testValidation(createSchema, invalidData, false, /required/i);
            });

            it("should accept bot with empty botSettings", async () => {
                const result = await testValidation(
                    createSchema,
                    botFixtures.edgeCases.emptyBotSettings.create,
                    true,
                );
                expect(result.botSettings).toEqual({});
            });

            it("should accept bot with complex nested botSettings", async () => {
                // Just verify that validation passes with complex settings
                await testValidation(
                    createSchema,
                    botFixtures.edgeCases.complexBotSettings.create,
                    true,
                );
            });

            it("should validate handle constraints", async () => {
                // Too short - just verify it fails
                await testValidation(
                    createSchema,
                    botFixtures.invalid.invalidHandle.create,
                    false,
                );

                // Too long in update
                await testValidation(
                    updateSchema,
                    botFixtures.invalid.invalidHandle.update,
                    false,
                );

                // Valid handles - test separately to isolate issues
                const validHandles = ["bot", "user123", "test_bot", "b".repeat(16)];
                for (const handle of validHandles) {
                    const testData = botTestDataFactory.createMinimal({ handle });
                    const result = await testValidation(createSchema, testData, true);
                    expect(result.handle).toBe(handle);
                }
            });

            it("should validate name constraints", async () => {
                // Empty name
                await testValidation(
                    createSchema,
                    botFixtures.invalid.invalidName.create,
                    false,
                    /required/i,
                );

                // Too long name
                await testValidation(
                    updateSchema,
                    botFixtures.invalid.invalidName.update,
                    false,
                    /over the limit/i,
                );
            });

            it("should validate image constraints", async () => {
                // Too long profile image
                await testValidation(
                    createSchema,
                    botFixtures.invalid.invalidImage.create,
                    false,
                );

                // Too long banner image
                await testValidation(
                    updateSchema,
                    botFixtures.invalid.invalidImage.update,
                    false,
                );
            });

            it("should accept bot depicting person", async () => {
                const result = await testValidation(
                    createSchema,
                    botFixtures.edgeCases.botDepictingPerson.create,
                    true,
                );
                expect(result.isBotDepictingPerson).toBe(true);
            });

            it("should accept public and private bots", async () => {
                const publicResult = await testValidation(
                    createSchema,
                    botFixtures.edgeCases.publicBot.create,
                    true,
                );
                expect(publicResult.isPrivate).toBe(false);

                const privateResult = await testValidation(
                    createSchema,
                    botFixtures.edgeCases.privateBot.create,
                    true,
                );
                expect(privateResult.isPrivate).toBe(true);
            });

            it("should validate translations", async () => {
                const result = await testValidation(
                    createSchema,
                    botFixtures.edgeCases.withTranslations.create,
                    true,
                );
                // Translations are processed but may not be in result
                expect(result).to.be.an("object");
            });

            it("should accept bio at max length", async () => {
                const result = await testValidation(
                    createSchema,
                    botFixtures.edgeCases.longBio.create,
                    true,
                );
                expect(result).to.be.an("object");
            });
        });

        describe("update validation", () => {
            it("should only require id for updates", async () => {
                const result = await testValidation(
                    updateSchema,
                    botFixtures.minimal.update,
                    true,
                );
                expect(result).toHaveProperty("id");
            });

            it("should allow updating botSettings", async () => {
                const result = await testValidation(
                    updateSchema,
                    botFixtures.complete.update,
                    true,
                );
                expect(result.botSettings).to.be.an("object");
            });

            it("should reject too long handle in update", async () => {
                await testValidation(
                    updateSchema,
                    botFixtures.invalid.invalidHandle.update,
                    false,
                    /over the limit/i,
                );
            });

            it("should handle translation operations", async () => {
                // Just verify validation passes
                await testValidation(
                    updateSchema,
                    botFixtures.complete.update,
                    true,
                );
            });
        });

        describe("batch validation scenarios", () => {
            it("should validate multiple bot scenarios", async () => {
                await testValidationBatch(createSchema, [
                    {
                        data: botTestDataFactory.createMinimal(),
                        shouldPass: true,
                        description: "minimal valid bot",
                    },
                    {
                        data: botTestDataFactory.createComplete(),
                        shouldPass: true,
                        description: "complete valid bot",
                    },
                    {
                        data: botFixtures.invalid.missingRequired.create,
                        shouldPass: false,
                        expectedError: /required/i,
                        description: "missing required fields",
                    },
                    {
                        data: botTestDataFactory.createMinimal({
                            handle: "x".repeat(17),
                        }),
                        shouldPass: false,
                        expectedError: /over the limit/i,
                        description: "handle too long",
                    },
                    {
                        data: botTestDataFactory.createMinimal({
                            isBotDepictingPerson: "not-a-boolean",
                        }),
                        shouldPass: false,
                        description: "invalid boolean type",
                    },
                ]);
            });
        });

        describe("omitFields functionality", () => {
            it("should omit specified fields", async () => {
                const schema = botValidation.create({
                    omitFields: ["handle", "isPrivate", "bannerImage", "profileImage"],
                });
                const data = botTestDataFactory.createComplete();
                const result = await schema.validate(data, { stripUnknown: true });

                expect(result).to.not.have.property("handle");
                expect(result).to.not.have.property("isPrivate");
                expect(result).to.not.have.property("bannerImage");
                expect(result).to.not.have.property("profileImage");

                // Required fields should still be present
                expect(result).toHaveProperty("id");
                expect(result).toHaveProperty("botSettings");
                expect(result).toHaveProperty("isBotDepictingPerson");
                expect(result).toHaveProperty("name");
            });
        });
    });
});

describe("botTranslationValidation", () => {
    // Custom tests for botTranslation since it has required fields that can't be omitted
    const defaultParams = { omitFields: [] };

    describe("create validation", () => {
        const createSchema = botTranslationValidation.create(defaultParams);

        it("should accept minimal valid data", async () => {
            const result = await testValidation(
                createSchema,
                botTranslationFixtures.minimal.create,
                true,
            );
            expect(result).toHaveProperty("id");
            expect(result).toHaveProperty("language");
        });

        it("should accept complete valid data", async () => {
            const result = await testValidation(
                createSchema,
                botTranslationFixtures.complete.create,
                true,
            );
            expect(result).toHaveProperty("bio");
        });

        it("should reject missing required fields", async () => {
            await testValidation(
                createSchema,
                botTranslationFixtures.invalid.missingRequired.create,
                false,
                /required/i,
            );
        });

        it("should reject invalid types", async () => {
            await testValidation(
                createSchema,
                botTranslationFixtures.invalid.invalidTypes.create,
                false,
            );
        });

        it("should reject bio exceeding max length", async () => {
            await testValidation(
                createSchema,
                botTranslationFixtures.invalid.tooLongBio.create,
                false,
                /over the limit/i,
            );
        });
    });

    describe("update validation", () => {
        const updateSchema = botTranslationValidation.update(defaultParams);

        it("should accept minimal valid data", async () => {
            const result = await testValidation(
                updateSchema,
                botTranslationFixtures.minimal.update,
                true,
            );
            expect(result).toHaveProperty("id");
        });

        it("should accept complete valid data", async () => {
            const result = await testValidation(
                updateSchema,
                botTranslationFixtures.complete.update,
                true,
            );
            expect(result).toHaveProperty("bio");
        });

        it("should reject missing id", async () => {
            await testValidation(
                updateSchema,
                botTranslationFixtures.invalid.missingRequired.update,
                false,
                /required/i,
            );
        });

        it("should reject bio exceeding max length", async () => {
            await testValidation(
                updateSchema,
                botTranslationFixtures.invalid.tooLongBio.update,
                false,
                /over the limit/i,
            );
        });
    });

    describe("language handling", () => {
        it("should accept various language codes", async () => {
            const schema = botTranslationValidation.create(defaultParams);

            const languages = ["en", "es", "fr", "de", "ja", "zh", "ko", "pt", "ru", "ar"];
            for (const lang of languages) {
                const result = await testValidation(
                    schema,
                    botTranslationTestDataFactory.createMinimal({ language: lang }),
                    true,
                );
                expect(result.language).toBe(lang);
            }
        });
    });
});
