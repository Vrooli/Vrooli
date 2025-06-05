import { expect } from "chai";
import { emailValidation } from "./email.js";
import { emailFixtures, emailTestDataFactory } from "./__test__/fixtures/emailFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test__/validationTestUtils.js";

describe("emailValidation", () => {
    // Run standard test suite (but only for create since update isn't supported)
    describe("emailValidation - Standard Tests", () => {
        const defaultParams = { omitFields: [] };

        describe("create validation", () => {
            const createSchema = emailValidation.create(defaultParams);

            it("should accept minimal valid data", async () => {
                const result = await testValidation(
                    createSchema,
                    emailFixtures.minimal.create,
                    true,
                );
                expect(result).to.have.property("id");
                expect(result).to.have.property("emailAddress");
            });

            it("should accept complete valid data", async () => {
                const result = await testValidation(
                    createSchema,
                    emailFixtures.complete.create,
                    true,
                );
                expect(result.emailAddress).to.equal(emailFixtures.complete.create.emailAddress);
            });

            it("should reject missing required fields", async () => {
                await testValidation(
                    createSchema,
                    emailFixtures.invalid.missingRequired.create,
                    false,
                    /required/i,
                );
            });

            it("should reject invalid field types", async () => {
                await testValidation(
                    createSchema,
                    emailFixtures.invalid.invalidTypes.create,
                    false,
                );
            });
        });

        describe("update validation", () => {
            it("should not have update method", () => {
                expect(emailValidation.update).to.be.undefined;
            });
        });
    });

    // Email-specific tests
    describe("email-specific validation", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = emailValidation.create(defaultParams);

        it("should reject invalid email format", async () => {
            await testValidation(
                createSchema,
                emailFixtures.invalid.invalidEmail.create,
                false,
                /valid email/i,
            );
        });

        it("should reject email exceeding max length", async () => {
            await testValidation(
                createSchema,
                emailFixtures.invalid.tooLongEmail.create,
                false,
                /characters over the limit/i,
            );
        });

        it("should accept minimal valid email", async () => {
            const result = await testValidation(
                createSchema,
                emailFixtures.edgeCases.minimalEmail.create,
                true,
            );
            expect(result.emailAddress).to.equal("a@b.c");
        });

        it("should accept email with plus sign", async () => {
            const result = await testValidation(
                createSchema,
                emailFixtures.edgeCases.emailWithPlus.create,
                true,
            );
            expect(result.emailAddress).to.equal("test+filter@gmail.com");
        });

        it("should accept email with dots", async () => {
            const result = await testValidation(
                createSchema,
                emailFixtures.edgeCases.emailWithDots.create,
                true,
            );
            expect(result.emailAddress).to.equal("first.last@company.com");
        });

        it("should accept email with numbers", async () => {
            const result = await testValidation(
                createSchema,
                emailFixtures.edgeCases.emailWithNumbers.create,
                true,
            );
            expect(result.emailAddress).to.equal("user123@example456.com");
        });

        it("should accept email with hyphens in domain", async () => {
            const result = await testValidation(
                createSchema,
                emailFixtures.edgeCases.emailWithHyphens.create,
                true,
            );
            expect(result.emailAddress).to.equal("test@my-company.com");
        });

        it("should reject empty string email", async () => {
            await testValidation(
                createSchema,
                emailFixtures.edgeCases.emptyString.create,
                false,
                /required/i,
            );
        });

        it("should reject whitespace-only email", async () => {
            await testValidation(
                createSchema,
                emailFixtures.edgeCases.whitespaceOnly.create,
                false,
                /required/i,
            );
        });
    });

    describe("batch validation scenarios", () => {
        it("should validate multiple email scenarios", async () => {
            const defaultParams = { omitFields: [] };
            const schema = emailValidation.create(defaultParams);
            
            await testValidationBatch(schema, [
                {
                    data: emailTestDataFactory.createMinimal(),
                    shouldPass: true,
                    description: "minimal valid email",
                },
                {
                    data: emailTestDataFactory.createComplete(),
                    shouldPass: true,
                    description: "complete valid email",
                },
                {
                    data: emailFixtures.invalid.invalidEmail.create,
                    shouldPass: false,
                    expectedError: /valid email/i,
                    description: "invalid email format",
                },
                {
                    data: emailTestDataFactory.createMinimal({ emailAddress: "test@" }),
                    shouldPass: false,
                    expectedError: /valid email/i,
                    description: "incomplete email",
                },
                {
                    data: emailTestDataFactory.createMinimal({ emailAddress: "@example.com" }),
                    shouldPass: false,
                    expectedError: /valid email/i,
                    description: "missing local part",
                },
            ]);
        });
    });

    describe("omitFields functionality", () => {
        it("should omit specified fields", async () => {
            const schema = emailValidation.create({ omitFields: ["emailAddress"] });
            const data = {
                id: "123456789012345678",
                emailAddress: "test@example.com",
            };
            const result = await schema.validate(data, { stripUnknown: true });
            expect(result).to.have.property("id");
            expect(result).to.not.have.property("emailAddress");
        });
    });
});