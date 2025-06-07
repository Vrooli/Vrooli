import { describe, it, expect } from "vitest";
import { tagValidation, tagTranslationValidation } from "./tag.js";
import { tagFixtures, tagTestDataFactory } from "./__test__/fixtures/tagFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test__/validationTestUtils.js";

describe("tagValidation", () => {
    // Run standard test suite
    runStandardValidationTests(tagValidation, tagFixtures, "tag");

    // Additional tag-specific tests
    describe("tag-specific edge cases", () => {
        const defaultParams = { omitFields: [] };

        it("should handle empty string tags", async () => {
            const schema = tagValidation.create(defaultParams);
            await testValidation(
                schema,
                tagFixtures.edgeCases.emptyTag.create,
                false,
                "This field is required",
            );
        });

        it("should trim whitespace from tags", async () => {
            const schema = tagValidation.create(defaultParams);
            const result = await testValidation(
                schema,
                {
                    id: "123456789012345678",
                    tag: "  javascript  ",
                },
                true,
            );
            expect(result.tag).to.equal("javascript");
        });

        it("should accept tags with special characters", async () => {
            const schema = tagValidation.create(defaultParams);
            const result = await testValidation(
                schema,
                tagFixtures.edgeCases.specialCharacters.create,
                true,
            );
            expect(result.tag).to.equal("c++");
        });

        it("should handle nested omitFields with dot notation", async () => {
            const schema = tagValidation.update({ 
                omitFields: ["translationsCreate", "translationsDelete"], 
            });
            const data = {
                id: "123456789012345690",
                translationsCreate: [{ language: "en", description: "test" }],
                translationsUpdate: [{ id: "123456789012345691", language: "en", description: "test" }],
                translationsDelete: ["123456789012345692"],
            };
            const result = await schema.validate(data, { stripUnknown: true });
            expect(result).to.not.have.property("translationsCreate");
            expect(result).to.have.property("translationsUpdate");
            expect(result).to.not.have.property("translationsDelete");
        });
    });

    describe("batch validation scenarios", () => {
        it("should validate multiple tag scenarios", async () => {
            const defaultParams = { omitFields: [] };
            const schema = tagValidation.create(defaultParams);
            await testValidationBatch(schema, [
                {
                    data: tagTestDataFactory.createMinimal(),
                    shouldPass: true,
                    description: "minimal valid tag",
                },
                {
                    data: tagTestDataFactory.createComplete(),
                    shouldPass: true,
                    description: "complete valid tag",
                },
                {
                    data: tagFixtures.invalid.tooLongTag.create,
                    shouldPass: false,
                    expectedError: /characters over the limit/i,
                    description: "tag exceeding max length",
                },
                {
                    data: tagFixtures.invalid.invalidId.create,
                    shouldPass: false,
                    description: "invalid Snowflake ID format",
                },
            ]);
        });
    });
});

describe("tagTranslationValidation", () => {
    const defaultParams = { omitFields: [] };

    describe("create", () => {
        const createSchema = tagTranslationValidation.create(defaultParams);

        it("should require id, language, and description", async () => {
            await testValidation(
                createSchema,
                { language: "en" },
                false,
                /required/i,  // More flexible - handles multiple error messages
            );
        });

        it("should accept valid translation", async () => {
            const validTranslation = {
                id: "123456789012345693",
                language: "en",
                description: "A programming language tag",
            };
            const result = await testValidation(createSchema, validTranslation, true);
            expect(result).to.deep.include(validTranslation);
        });
    });

    describe("update", () => {
        const updateSchema = tagTranslationValidation.update(defaultParams);

        it("should require id and make description optional", async () => {
            const result = await testValidation(
                updateSchema,
                { 
                    id: "123456789012345694", 
                    language: "en", 
                },
                true,
            );
            expect(result).to.have.property("id");
            expect(result).to.have.property("language");
        });

        it("should accept update with description", async () => {
            const result = await testValidation(
                updateSchema,
                { 
                    id: "123456789012345695", 
                    language: "en",
                    description: "Updated description",
                },
                true,
            );
            expect(result.description).to.equal("Updated description");
        });
    });
});
