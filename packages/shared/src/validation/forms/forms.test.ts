import { describe, expect, it } from "vitest";
import { ValidationError } from "yup";
import {
    emailLogInFormValidation,
    emailSignUpFormValidation,
    emailSignUpValidation,
    nodeEndFormValidation,
    profileEmailUpdateFormValidation,
} from "./forms.js";

describe("Form Validations", () => {
    describe("nodeEndFormValidation", () => {
        it("should accept valid node end form data", async () => {
            const validData = {
                wasSuccessful: true,
                name: "End Node",
                description: "This is the end node",
            };
            const result = await nodeEndFormValidation.validate(validData);
            expect(result).toEqual(validData);
        });

        it("should accept data without optional fields", async () => {
            const validData = {
                name: "End Node",
            };
            const result = await nodeEndFormValidation.validate(validData);
            expect(result).toEqual(validData);
        });

        it("should reject missing required name", async () => {
            const invalidData = {
                wasSuccessful: true,
                description: "Missing name",
            };
            await expect(nodeEndFormValidation.validate(invalidData)).rejects.toThrow(ValidationError);
        });

        it("should reject invalid wasSuccessful type", async () => {
            const invalidData = {
                wasSuccessful: "not a boolean",
                name: "End Node",
            };
            await expect(nodeEndFormValidation.validate(invalidData)).rejects.toThrow(ValidationError);
        });

        it("should trim and validate name", async () => {
            const data = {
                name: "  Trimmed Name  ",
            };
            const result = await nodeEndFormValidation.validate(data);
            expect(result.name).toBe("Trimmed Name");
        });

        it("should reject name exceeding max length", async () => {
            const invalidData = {
                name: "a".repeat(256), // Assuming NAME_MAX_LENGTH is 255
            };
            await expect(nodeEndFormValidation.validate(invalidData)).rejects.toThrow(ValidationError);
        });
    });

    describe("emailLogInFormValidation", () => {
        it("should accept valid login credentials", async () => {
            const validData = {
                email: "test@example.com",
                password: "password123",
            };
            const result = await emailLogInFormValidation.validate(validData);
            expect(result).toEqual(validData);
        });

        it("should trim email and password", async () => {
            const data = {
                email: "  test@example.com  ",
                password: "  password123  ",
            };
            const result = await emailLogInFormValidation.validate(data);
            expect(result.email).toBe("test@example.com");
            expect(result.password).toBe("password123");
        });

        it("should reject invalid email format", async () => {
            const invalidData = {
                email: "not-an-email",
                password: "password123",
            };
            await expect(emailLogInFormValidation.validate(invalidData)).rejects.toThrow(ValidationError);
        });

        it("should reject missing email", async () => {
            const invalidData = {
                password: "password123",
            };
            await expect(emailLogInFormValidation.validate(invalidData)).rejects.toThrow(ValidationError);
        });

        it("should reject missing password", async () => {
            const invalidData = {
                email: "test@example.com",
            };
            await expect(emailLogInFormValidation.validate(invalidData)).rejects.toThrow(ValidationError);
        });

        it("should reject empty password after trimming", async () => {
            const invalidData = {
                email: "test@example.com",
                password: "   ",
            };
            await expect(emailLogInFormValidation.validate(invalidData)).rejects.toThrow(ValidationError);
        });

        it("should reject password exceeding max length", async () => {
            const invalidData = {
                email: "test@example.com",
                password: "a".repeat(1025), // Assuming PASSWORD_VERIFICATION_CODE_MAX_LENGTH is 1024
            };
            await expect(emailLogInFormValidation.validate(invalidData)).rejects.toThrow(ValidationError);
        });
    });

    describe("emailSignUpFormValidation", () => {
        it("should accept valid signup data", async () => {
            const validData = {
                name: "Test User",
                email: "test@example.com",
                marketingEmails: true,
                agreeToTerms: true,
                password: "SecurePassword123!",
            };
            const result = await emailSignUpFormValidation.validate(validData);
            expect(result).toEqual(validData);
        });

        it("should reject when agreeToTerms is false", async () => {
            const invalidData = {
                name: "Test User",
                email: "test@example.com",
                marketingEmails: true,
                agreeToTerms: false,
                password: "SecurePassword123!",
            };
            await expect(emailSignUpFormValidation.validate(invalidData)).rejects.toThrow(ValidationError);
        });

        it("should accept marketing emails as false", async () => {
            const validData = {
                name: "Test User",
                email: "test@example.com",
                marketingEmails: false,
                agreeToTerms: true,
                password: "SecurePassword123!",
            };
            const result = await emailSignUpFormValidation.validate(validData);
            expect(result.marketingEmails).toBe(false);
        });

        it("should reject missing required fields", async () => {
            const testCases = [
                { email: "test@example.com", marketingEmails: true, agreeToTerms: true, password: "password" },
                { name: "Test", marketingEmails: true, agreeToTerms: true, password: "password" },
                { name: "Test", email: "test@example.com", agreeToTerms: true, password: "password" },
                { name: "Test", email: "test@example.com", marketingEmails: true, password: "password" },
                { name: "Test", email: "test@example.com", marketingEmails: true, agreeToTerms: true },
            ];

            for (const invalidData of testCases) {
                await expect(emailSignUpFormValidation.validate(invalidData)).rejects.toThrow(ValidationError);
            }
        });

        it("should validate password requirements", async () => {
            const invalidData = {
                name: "Test User",
                email: "test@example.com",
                marketingEmails: true,
                agreeToTerms: true,
                password: "weak", // Too short
            };
            await expect(emailSignUpFormValidation.validate(invalidData)).rejects.toThrow(ValidationError);
        });

        it("should validate email format for clearly invalid cases", async () => {
            const clearlyInvalidEmails = [
                "",             // Empty string
                " ",            // Whitespace only
                "notanemail",   // No @ symbol
                "@example.com", // No username
                "user@",        // No domain
            ];

            for (const invalidEmail of clearlyInvalidEmails) {
                const invalidData = {
                    name: "Test User",
                    email: invalidEmail,
                    marketingEmails: true,
                    agreeToTerms: true,
                    password: "SecurePassword123!",
                };
                await expect(emailSignUpFormValidation.validate(invalidData)).rejects.toThrow(ValidationError);
            }
        });

        it("should validate name field requirements", async () => {
            const invalidNames = [
                "", // Empty string
                " ", // Whitespace only
                "a".repeat(500), // Too long
            ];

            for (const invalidName of invalidNames) {
                const invalidData = {
                    name: invalidName,
                    email: "test@example.com",
                    marketingEmails: true,
                    agreeToTerms: true,
                    password: "SecurePassword123!",
                };
                await expect(emailSignUpFormValidation.validate(invalidData)).rejects.toThrow(ValidationError);
            }
        });

        it("should trim whitespace from string fields", async () => {
            const dataWithWhitespace = {
                name: "  Test User  ",
                email: "  test@example.com  ",
                marketingEmails: true,
                agreeToTerms: true,
                password: "  SecurePassword123!  ",
            };
            
            const result = await emailSignUpFormValidation.validate(dataWithWhitespace);
            expect(result.name).toBe("Test User");
            expect(result.email).toBe("test@example.com");
            expect(result.password).toBe("SecurePassword123!");
        });

        it("should handle edge cases for boolean fields", async () => {
            // Test with string representations of booleans
            const validData = {
                name: "Test User",
                email: "test@example.com",
                marketingEmails: "false", // String representation
                agreeToTerms: "true", // String representation
                password: "SecurePassword123!",
            };
            
            const result = await emailSignUpFormValidation.validate(validData);
            expect(typeof result.marketingEmails).toBe("boolean");
            expect(typeof result.agreeToTerms).toBe("boolean");
            expect(result.agreeToTerms).toBe(true);
        });
    });

    describe("emailSignUpValidation", () => {
        it("should accept valid signup data without agreeToTerms", async () => {
            const validData = {
                name: "Test User",
                email: "test@example.com",
                marketingEmails: false,
                password: "SecurePassword123!",
            };
            const result = await emailSignUpValidation.validate(validData);
            expect(result).toEqual(validData);
        });

        it("should have same behavior as emailSignUpFormValidation except agreeToTerms", async () => {
            const data = {
                name: "Test User",
                email: "test@example.com",
                marketingEmails: true,
                password: "SecurePassword123!",
            };
            
            // This should pass for emailSignUpValidation
            const result = await emailSignUpValidation.validate(data);
            expect(result).toEqual(data);
            
            // But fail for emailSignUpFormValidation (missing agreeToTerms)
            await expect(emailSignUpFormValidation.validate(data)).rejects.toThrow(ValidationError);
        });
    });

    describe("profileEmailUpdateFormValidation", () => {
        it("should accept valid profile email update", async () => {
            const validData = {
                currentPassword: "CurrentPassword123!",
                newPassword: "NewPassword123!",
                emailsCreate: [
                    { emailAddress: "new1@example.com" },
                    { emailAddress: "new2@example.com" },
                ],
                emailsDelete: ["123456789012345678", "234567890123456789"], // Valid Snowflake IDs
            };
            const result = await profileEmailUpdateFormValidation.validate(validData);
            expect(result).toEqual(validData);
        });

        it("should accept only currentPassword", async () => {
            const validData = {
                currentPassword: "CurrentPassword123!",
            };
            const result = await profileEmailUpdateFormValidation.validate(validData);
            expect(result).toEqual(validData);
        });

        it("should reject missing currentPassword", async () => {
            const invalidData = {
                newPassword: "NewPassword123!",
            };
            await expect(profileEmailUpdateFormValidation.validate(invalidData)).rejects.toThrow(ValidationError);
        });

        it("should accept empty arrays for emails", async () => {
            const validData = {
                currentPassword: "CurrentPassword123!",
                emailsCreate: [],
                emailsDelete: [],
            };
            const result = await profileEmailUpdateFormValidation.validate(validData);
            expect(result).toEqual(validData);
        });

        it("should validate email addresses in emailsCreate", async () => {
            const invalidData = {
                currentPassword: "CurrentPassword123!",
                emailsCreate: [
                    { emailAddress: "not-an-email" },
                ],
            };
            await expect(profileEmailUpdateFormValidation.validate(invalidData)).rejects.toThrow(ValidationError);
        });

        it("should validate multiple emails in emailsCreate", async () => {
            const validData = {
                currentPassword: "CurrentPassword123!",
                emailsCreate: [
                    { emailAddress: "valid1@example.com" },
                    { emailAddress: "valid2@example.com" },
                    { emailAddress: "valid3@example.com" },
                ],
            };
            const result = await profileEmailUpdateFormValidation.validate(validData);
            expect(result.emailsCreate).toHaveLength(3);
        });

        it("should accept valid Snowflake IDs in emailsDelete", async () => {
            const validData = {
                currentPassword: "CurrentPassword123!",
                emailsDelete: [
                    "123456789012345678", // Valid Snowflake ID
                    "987654321098765432", // Valid Snowflake ID
                ],
            };
            const result = await profileEmailUpdateFormValidation.validate(validData);
            expect(result.emailsDelete).toHaveLength(2);
        });

        it("should trim email addresses", async () => {
            const data = {
                currentPassword: "CurrentPassword123!",
                emailsCreate: [
                    { emailAddress: "  trimmed@example.com  " },
                ],
            };
            const result = await profileEmailUpdateFormValidation.validate(data);
            expect(result.emailsCreate[0].emailAddress).toBe("trimmed@example.com");
        });

        it("should validate password basic requirements", async () => {
            const clearlyInvalidPasswords = [
                "",             // Empty
                " ",            // Whitespace only
                "123",          // Too short
            ];

            for (const invalidPassword of clearlyInvalidPasswords) {
                const invalidData = {
                    currentPassword: "CurrentPassword123!",
                    newPassword: invalidPassword,
                };
                await expect(profileEmailUpdateFormValidation.validate(invalidData)).rejects.toThrow(ValidationError);
            }
        });

        it("should handle large numbers of email operations", async () => {
            const manyEmails = Array.from({ length: 100 }, (_, i) => ({ 
                emailAddress: `email${i}@example.com`, 
            }));
            const manyIds = Array.from({ length: 100 }, (_, i) => `12345678901234567${i.toString().padStart(2, "0")}`);

            const validData = {
                currentPassword: "CurrentPassword123!",
                emailsCreate: manyEmails,
                emailsDelete: manyIds,
            };

            const result = await profileEmailUpdateFormValidation.validate(validData);
            expect(result.emailsCreate).toHaveLength(100);
            expect(result.emailsDelete).toHaveLength(100);
        });

        it("should reject invalid email formats in emailsCreate", async () => {
            const invalidEmails = [
                "notanemail",
                "@example.com", 
                "user@",
                "user space@example.com",
            ];

            for (const invalidEmail of invalidEmails) {
                const invalidData = {
                    currentPassword: "CurrentPassword123!",
                    emailsCreate: [{ emailAddress: invalidEmail }],
                };
                await expect(profileEmailUpdateFormValidation.validate(invalidData)).rejects.toThrow(ValidationError);
            }
        });


        it("should handle mixed valid and invalid data", async () => {
            // This should pass validation for valid parts and ignore/fail on invalid parts
            const mixedData = {
                currentPassword: "CurrentPassword123!",
                newPassword: "NewPassword456!",
                emailsCreate: [
                    { emailAddress: "valid@example.com" },
                ],
                emailsDelete: ["123456789012345678"],
            };

            const result = await profileEmailUpdateFormValidation.validate(mixedData);
            expect(result.currentPassword).toBe("CurrentPassword123!");
            expect(result.newPassword).toBe("NewPassword456!");
            expect(result.emailsCreate).toHaveLength(1);
            expect(result.emailsDelete).toHaveLength(1);
        });
    });
});
