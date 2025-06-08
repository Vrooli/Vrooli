import { describe, expect, it } from "vitest";
import { phoneFixtures, phoneTestDataFactory } from "./__test/fixtures/phoneFixtures.js";
import { testValidation, testValidationBatch } from "./__test/validationTestUtils.js";
import { phoneValidation } from "./phone.js";

describe("phoneValidation", () => {
    // Run standard test suite (but only for create since update isn't supported)
    describe("phoneValidation - Standard Tests", () => {
        const defaultParams = { omitFields: [] };

        describe("create validation", () => {
            const createSchema = phoneValidation.create(defaultParams);

            it("should accept minimal valid data", async () => {
                const result = await testValidation(
                    createSchema,
                    phoneFixtures.minimal.create,
                    true,
                );
                expect(result).to.have.property("id");
                expect(result).to.have.property("phoneNumber");
            });

            it("should accept complete valid data", async () => {
                const result = await testValidation(
                    createSchema,
                    phoneFixtures.complete.create,
                    true,
                );
                expect(result.phoneNumber).to.equal(phoneFixtures.complete.create.phoneNumber);
            });

            it("should reject missing required fields", async () => {
                await testValidation(
                    createSchema,
                    phoneFixtures.invalid.missingRequired.create,
                    false,
                    /required/i,
                );
            });

            it("should reject invalid field types", async () => {
                await testValidation(
                    createSchema,
                    phoneFixtures.invalid.invalidTypes.create,
                    false,
                );
            });
        });

        describe("update validation", () => {
            it("should not have update method", () => {
                expect(phoneValidation.update).to.be.undefined;
            });
        });
    });

    // Phone-specific tests
    describe("phone-specific validation", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = phoneValidation.create(defaultParams);

        it("should reject phone exceeding max length", async () => {
            await testValidation(
                createSchema,
                phoneFixtures.invalid.tooLongPhone.create,
                false,
                /characters over the limit/i,
            );
        });

        it("should accept short phone number", async () => {
            const result = await testValidation(
                createSchema,
                phoneFixtures.edgeCases.shortPhone.create,
                true,
            );
            expect(result.phoneNumber).to.equal("+1");
        });

        it("should accept phone with spaces", async () => {
            const result = await testValidation(
                createSchema,
                phoneFixtures.edgeCases.phoneWithSpaces.create,
                true,
            );
            expect(result.phoneNumber).to.equal("+1 234 567 8900");
        });

        it("should accept phone with dashes", async () => {
            const result = await testValidation(
                createSchema,
                phoneFixtures.edgeCases.phoneWithDashes.create,
                true,
            );
            expect(result.phoneNumber).to.equal("+1-234-567-8900");
        });

        it("should accept phone with parentheses", async () => {
            const result = await testValidation(
                createSchema,
                phoneFixtures.edgeCases.phoneWithParentheses.create,
                true,
            );
            expect(result.phoneNumber).to.equal("+1(234)567-8900");
        });

        it("should accept phone with dots", async () => {
            const result = await testValidation(
                createSchema,
                phoneFixtures.edgeCases.phoneWithDots.create,
                true,
            );
            expect(result.phoneNumber).to.equal("+1.234.567.8900");
        });

        it("should accept international format", async () => {
            const result = await testValidation(
                createSchema,
                phoneFixtures.edgeCases.internationalFormat.create,
                true,
            );
            expect(result.phoneNumber).to.equal("+44 20 7946 0958");
        });

        it("should accept max length phone", async () => {
            const result = await testValidation(
                createSchema,
                phoneFixtures.edgeCases.maxLengthPhone.create,
                true,
            );
            expect(result.phoneNumber).to.equal("+12345678901234");
        });

        it("should reject empty string phone", async () => {
            await testValidation(
                createSchema,
                phoneFixtures.edgeCases.emptyString.create,
                false,
                /required/i,
            );
        });

        it("should reject whitespace-only phone", async () => {
            await testValidation(
                createSchema,
                phoneFixtures.edgeCases.whitespaceOnly.create,
                false,
                /required/i,
            );
        });
    });

    describe("batch validation scenarios", () => {
        it("should validate multiple phone scenarios", async () => {
            const defaultParams = { omitFields: [] };
            const schema = phoneValidation.create(defaultParams);

            await testValidationBatch(schema, [
                {
                    data: phoneTestDataFactory.createMinimal(),
                    shouldPass: true,
                    description: "minimal valid phone",
                },
                {
                    data: phoneTestDataFactory.createComplete(),
                    shouldPass: true,
                    description: "complete valid phone",
                },
                {
                    data: phoneFixtures.invalid.tooLongPhone.create,
                    shouldPass: false,
                    expectedError: /characters over the limit/i,
                    description: "phone exceeding max length",
                },
                {
                    data: phoneTestDataFactory.createMinimal({ phoneNumber: "123" }),
                    shouldPass: true,
                    description: "phone without country code",
                },
                {
                    data: phoneTestDataFactory.createMinimal({ phoneNumber: "+1 (800) FLOWERS" }),
                    shouldPass: true,
                    description: "phone with letters (vanity number)",
                },
            ]);
        });
    });

    describe("omitFields functionality", () => {
        it("should omit specified fields", async () => {
            const schema = phoneValidation.create({ omitFields: ["phoneNumber"] });
            const data = {
                id: "123456789012345678",
                phoneNumber: "+12345678900",
            };
            const result = await schema.validate(data, { stripUnknown: true });
            expect(result).to.have.property("id");
            expect(result).to.not.have.property("phoneNumber");
        });
    });
});
