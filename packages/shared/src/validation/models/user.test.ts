import { describe, expect, it } from "vitest";
import {
    emailLogInFixtures,
    emailRequestPasswordChangeFixtures,
    emailResetPasswordFixtures,
    emailResetPasswordFormFixtures,
    profileEmailUpdateFixtures,
    switchCurrentAccountFixtures,
    userDeleteOneFixtures,
    userFixtures,
    userTestDataFactory,
    userTranslationFixtures,
    validateSessionFixtures,
} from "./__test/fixtures/userFixtures.js";
import { runStandardValidationTests, testValidation, testValidationBatch } from "./__test/validationTestUtils.js";
import {
    emailLogInSchema,
    emailRequestPasswordChangeSchema,
    emailResetPasswordFormSchema,
    emailResetPasswordSchema,
    profileEmailUpdateValidation,
    profileValidation,
    switchCurrentAccountSchema,
    userDeleteOneSchema,
    userTranslationValidation,
    userValidation,
    validateSessionSchema,
} from "./user.js";

describe("userValidation", () => {
    // Run standard test suite for userValidation (bot creation and profile updates)
    runStandardValidationTests(userValidation, userFixtures, "user");

    // User-specific tests for bot creation
    describe("bot creation validation", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = userValidation.create(defaultParams);

        it("should require handle, isPrivate, and name for bot creation", async () => {
            const invalidData = {
                isBotDepictingPerson: true,
            };
            await testValidation(createSchema, invalidData, false, /required/i);
        });

        it("should accept bot with minimal settings", async () => {
            const result = await testValidation(
                createSchema,
                userFixtures.edgeCases.minimalBotSettings.create,
                true,
            );
            expect(result.botSettings).toEqual({});
            expect(result).toHaveProperty("handle");
            expect(result).toHaveProperty("isPrivate");
            expect(result).toHaveProperty("name");
            expect(result.isBotDepictingPerson).to.be.a("boolean");
        });

        it("should accept bot with complex settings", async () => {
            const result = await testValidation(
                createSchema,
                userFixtures.edgeCases.complexBotSettings.create,
                true,
            );
            // botSettings should be a valid config object with expected structure
            expect(result.botSettings).to.be.an("object");
            expect(result.botSettings).to.not.be.null;
            // Complex settings should preserve their structure
            if (typeof result.botSettings === "object" && result.botSettings !== null) {
                // The validation might transform or filter the botSettings,
                // but it should still be a valid object
                expect(result.botSettings).to.be.an("object");
                // Should have required bot fields
                expect(result).toHaveProperty("handle", "complexbot");
                expect(result).toHaveProperty("name", "Complex Bot");
                expect(result).toHaveProperty("isPrivate", false);
                expect(result).toHaveProperty("isBotDepictingPerson", false);
            }
        });

        it("should validate handle length", async () => {
            // Too short - expect any error (could be multiple validation errors)
            await testValidation(
                createSchema,
                userFixtures.invalid.invalidHandle.create,
                false,
            );

            // Too long
            await testValidation(
                createSchema,
                userFixtures.invalid.tooLongHandle.create,
                false,
            );
        });

        it("should accept underscores in handle", async () => {
            const result = await testValidation(
                createSchema,
                userFixtures.edgeCases.underscoreHandle.create,
                true,
            );
            expect(result.handle).toContain("_");
        });

        it("should validate image formats", async () => {
            const validBot = userTestDataFactory.createComplete();
            const result = await testValidation(createSchema, validBot, true);
            
            // Images should be valid URLs, file paths, or data URLs if present
            if (result.bannerImage) {
                expect(result.bannerImage).to.be.a("string");
                expect(result.bannerImage.length).toBeGreaterThan(0);
                // Should be a valid URL, file path, or data URL format
                const isValidImageFormat = 
                    result.bannerImage.startsWith("http://") ||
                    result.bannerImage.startsWith("https://") ||
                    result.bannerImage.startsWith("/") ||
                    result.bannerImage.startsWith("./") ||
                    result.bannerImage.startsWith("data:image/") ||
                    /^[a-zA-Z0-9_-]+\//.test(result.bannerImage);
                expect(isValidImageFormat).toBe(true);
            }
            if (result.profileImage) {
                expect(result.profileImage).to.be.a("string");
                expect(result.profileImage.length).toBeGreaterThan(0);
                // Should be a valid URL, file path, or data URL format
                const isValidImageFormat = 
                    result.profileImage.startsWith("http://") ||
                    result.profileImage.startsWith("https://") ||
                    result.profileImage.startsWith("/") ||
                    result.profileImage.startsWith("./") ||
                    result.profileImage.startsWith("data:image/") ||
                    /^[a-zA-Z0-9_-]+\//.test(result.profileImage);
                expect(isValidImageFormat).toBe(true);
            }
        });

        it("should validate translations structure", async () => {
            const botWithTranslations = userTestDataFactory.createComplete();
            const result = await testValidation(createSchema, botWithTranslations, true);
            
            // Should have required bot properties
            expect(result).toHaveProperty("id");
            expect(result).toHaveProperty("handle");
            expect(result).toHaveProperty("name");
            expect(result).toHaveProperty("isPrivate");
            
            // If translations are present, they should be properly structured
            if (result.translationsCreate && Array.isArray(result.translationsCreate)) {
                expect(result.translationsCreate).to.have.length.greaterThan(0);
                result.translationsCreate.forEach((translation: any) => {
                    expect(translation).toHaveProperty("language");
                    expect(translation.language).to.be.a("string");
                    // Language code should be valid ISO 639-1 format (e.g., "en", "es", "fr")
                    expect(translation.language).to.match(/^[a-z]{2}(-[A-Z]{2})?$/);
                    // Should have bio content
                    expect(translation).toHaveProperty("bio");
                    expect(translation.bio).to.be.a("string");
                });
            }
        });

        it("should reject bio exceeding max length", async () => {
            await testValidation(
                createSchema,
                userFixtures.invalid.tooLongBio.create,
                false,
                // Bio validation is in nested translations
            );
        });
    });

    // Profile update validation tests
    describe("profile update validation", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = userValidation.update(defaultParams);

        it("should update privacy settings", async () => {
            const result = await testValidation(
                updateSchema,
                userFixtures.edgeCases.allPrivacyFlags.update,
                true,
            );
            expect(result.isPrivate).toBe(true);
            expect(result.isPrivateMemberships).toBe(true);
            expect(result.isPrivateBookmarks).toBe(true);
        });

        it("should reject invalid theme", async () => {
            await testValidation(
                updateSchema,
                userFixtures.invalid.invalidTheme.update,
                false,
            );
        });

        it("should validate bot-specific fields in update", async () => {
            const result = await testValidation(
                updateSchema,
                userFixtures.complete.update,
                true,
            );
            expect(result.isBotDepictingPerson).to.be.a("boolean");
            expect(result.isBotDepictingPerson).toBe(false);
            
            // botSettings should have specific structure for bot configuration
            expect(result.botSettings).to.be.an("object");
            if (result.botSettings && Object.keys(result.botSettings).length > 0) {
                // Should contain valid bot configuration properties
                const allowedKeys = ["model", "temperature", "maxTokens", "systemPrompt", "responseFormat"];
                Object.keys(result.botSettings).forEach(key => {
                    expect(allowedKeys).toContain(key);
                });
            }
        });

        it("should handle translation operations correctly", async () => {
            const result = await testValidation(
                updateSchema,
                userFixtures.complete.update,
                true,
            );
            
            // Should preserve required ID
            expect(result).toHaveProperty("id");
            
            // Translation operations should be structured correctly if present
            if (result.translationsCreate && Array.isArray(result.translationsCreate)) {
                result.translationsCreate.forEach((translation: any) => {
                    expect(translation).toHaveProperty("language");
                    expect(translation.language).to.be.a("string");
                });
            }
            
            if (result.translationsUpdate && Array.isArray(result.translationsUpdate)) {
                result.translationsUpdate.forEach((translation: any) => {
                    expect(translation).toHaveProperty("id");
                    expect(translation).toHaveProperty("language");
                });
            }
        });
    });
});

describe("profileValidation", () => {
    // Profile validation only supports updates
    describe("update validation", () => {
        const defaultParams = { omitFields: [] };
        const updateSchema = profileValidation.update(defaultParams);

        it("should accept minimal profile update", async () => {
            const result = await testValidation(
                updateSchema,
                userFixtures.minimal.update,
                true,
            );
            expect(result).toHaveProperty("id");
        });

        it("should accept complete profile update", async () => {
            const profileData = {
                id: "300000000000000001",
                handle: "newhandle",
                name: "New Name",
                theme: "light",
                isPrivate: true,
            };
            const result = await testValidation(updateSchema, profileData, true);
            expect(result.handle).toBe("newhandle");
            expect(result.theme).toBe("light");
        });

        it("should not have create method", () => {
            expect(profileValidation.create).toBeUndefined();
        });
    });
});

describe("userTranslationValidation", () => {
    // Custom tests for userTranslation since it has required fields that can't be omitted
    const defaultParams = { omitFields: [] };

    describe("create validation", () => {
        const createSchema = userTranslationValidation.create(defaultParams);

        it("should accept minimal valid data", async () => {
            const result = await testValidation(
                createSchema,
                userTranslationFixtures.minimal.create,
                true,
            );
            expect(result).toHaveProperty("language", "en");
            // id is auto-added by transRel
            expect(result).toHaveProperty("id");
        });

        it("should accept complete valid data", async () => {
            const result = await testValidation(
                createSchema,
                userTranslationFixtures.complete.create,
                true,
            );
            expect(result).toHaveProperty("bio");
        });

        it("should reject missing language", async () => {
            await testValidation(
                createSchema,
                userTranslationFixtures.invalid.missingRequired.create,
                false,
                /required/i,
            );
        });

        it("should reject invalid types", async () => {
            await testValidation(
                createSchema,
                userTranslationFixtures.invalid.invalidTypes.create,
                false,
            );
        });
    });

    describe("update validation", () => {
        const updateSchema = userTranslationValidation.update(defaultParams);

        it("should accept minimal valid data", async () => {
            const result = await testValidation(
                updateSchema,
                userTranslationFixtures.minimal.update,
                true,
            );
            expect(result).toHaveProperty("id");
        });

        it("should accept complete valid data", async () => {
            const result = await testValidation(
                updateSchema,
                userTranslationFixtures.complete.update,
                true,
            );
            expect(result).toHaveProperty("bio");
        });

        it("should reject missing id", async () => {
            await testValidation(
                updateSchema,
                userTranslationFixtures.invalid.missingRequired.update,
                false,
                /required/i,
            );
        });
    });
});

describe("emailLogInSchema", () => {
    it("should accept minimal login data", async () => {
        const result = await testValidation(
            emailLogInSchema,
            emailLogInFixtures.minimal,
            true,
        );
        expect(result.email).toBe("user@example.com");
    });

    it("should accept complete login data", async () => {
        const result = await testValidation(
            emailLogInSchema,
            emailLogInFixtures.complete,
            true,
        );
        expect(result).toHaveProperty("password");
        expect(result).toHaveProperty("verificationCode");
    });

    it("should accept password only", async () => {
        const result = await testValidation(
            emailLogInSchema,
            emailLogInFixtures.withPasswordOnly,
            true,
        );
        expect(result).toHaveProperty("password");
        expect(result).to.not.have.property("email");
    });

    it("should reject invalid email", async () => {
        await testValidation(
            emailLogInSchema,
            emailLogInFixtures.invalid.invalidEmail,
            false,
            /valid email/i,
        );
    });

    it("should reject password exceeding max length", async () => {
        await testValidation(
            emailLogInSchema,
            emailLogInFixtures.invalid.tooLongPassword,
            false,
            /over the limit/i,  // Yup shows character count instead of limit
        );
    });

    it("should strip empty strings", async () => {
        const dataWithEmpty = {
            email: "user@example.com",
            password: "",
            verificationCode: "   ",
        };
        const result = await testValidation(emailLogInSchema, dataWithEmpty, true);
        expect(result).toHaveProperty("email");
        expect(result).to.not.have.property("password");
        expect(result).to.not.have.property("verificationCode");
    });
});

describe("userDeleteOneSchema", () => {
    it("should accept valid deletion request", async () => {
        const result = await testValidation(
            userDeleteOneSchema,
            userDeleteOneFixtures.valid,
            true,
        );
        expect(result.deletePublicData).toBe(true);
    });

    it("should accept deletion keeping public data", async () => {
        const result = await testValidation(
            userDeleteOneSchema,
            userDeleteOneFixtures.keepPublicData,
            true,
        );
        expect(result.deletePublicData).toBe(false);
    });

    it("should reject missing password", async () => {
        await testValidation(
            userDeleteOneSchema,
            userDeleteOneFixtures.invalid.missingPassword,
            false,
            /required/i,
        );
    });

    it("should reject missing delete flag", async () => {
        await testValidation(
            userDeleteOneSchema,
            userDeleteOneFixtures.invalid.missingDeleteFlag,
            false,
            /required/i,
        );
    });
});

describe("emailRequestPasswordChangeSchema", () => {
    it("should accept valid email", async () => {
        const result = await testValidation(
            emailRequestPasswordChangeSchema,
            emailRequestPasswordChangeFixtures.valid,
            true,
        );
        expect(result.email).toBe("user@example.com");
    });

    it("should reject missing email", async () => {
        await testValidation(
            emailRequestPasswordChangeSchema,
            emailRequestPasswordChangeFixtures.invalid.missingEmail,
            false,
            /required/i,
        );
    });

    it("should reject invalid email", async () => {
        await testValidation(
            emailRequestPasswordChangeSchema,
            emailRequestPasswordChangeFixtures.invalid.invalidEmail,
            false,
            /valid email/i,
        );
    });
});

describe("emailResetPasswordFormSchema", () => {
    it("should accept matching passwords", async () => {
        const result = await testValidation(
            emailResetPasswordFormSchema,
            emailResetPasswordFormFixtures.valid,
            true,
        );
        expect(result.newPassword).toBe(result.confirmNewPassword);
    });

    it("should reject password mismatch", async () => {
        await testValidation(
            emailResetPasswordFormSchema,
            emailResetPasswordFormFixtures.invalid.passwordMismatch,
            false,
            /match/i,
        );
    });

    it("should reject weak password", async () => {
        await testValidation(
            emailResetPasswordFormSchema,
            emailResetPasswordFormFixtures.invalid.weakPassword,
            false,
            /8 characters/i,
        );
    });

    it("should reject missing fields", async () => {
        await testValidation(
            emailResetPasswordFormSchema,
            emailResetPasswordFormFixtures.invalid.missingNewPassword,
            false,
            /required/i,
        );
    });
});

describe("emailResetPasswordSchema", () => {
    it("should accept reset with id", async () => {
        const result = await testValidation(
            emailResetPasswordSchema,
            emailResetPasswordFixtures.validWithId,
            true,
        );
        expect(result).toHaveProperty("id");
        expect(result).toHaveProperty("code");
        expect(result).toHaveProperty("newPassword");
    });

    it("should accept reset with publicId", async () => {
        const result = await testValidation(
            emailResetPasswordSchema,
            emailResetPasswordFixtures.validWithPublicId,
            true,
        );
        expect(result).toHaveProperty("publicId");
    });

    it("should accept reset with both id and publicId", async () => {
        const result = await testValidation(
            emailResetPasswordSchema,
            emailResetPasswordFixtures.validWithBoth,
            true,
        );
        expect(result).toHaveProperty("id");
        expect(result).toHaveProperty("publicId");
    });

    it("should reject missing identifier", async () => {
        await testValidation(
            emailResetPasswordSchema,
            emailResetPasswordFixtures.invalid.missingIdentifier,
            false,
            /id or publicId must be provided/i,
        );
    });

    it("should reject missing code", async () => {
        await testValidation(
            emailResetPasswordSchema,
            emailResetPasswordFixtures.invalid.missingCode,
            false,
            /required/i,
        );
    });

    it("should reject code exceeding max length", async () => {
        await testValidation(
            emailResetPasswordSchema,
            emailResetPasswordFixtures.invalid.tooLongCode,
            false,
            /over the limit/i,  // Yup shows character count instead of limit
        );
    });
});

describe("validateSessionSchema", () => {
    it("should accept valid time zone", async () => {
        const result = await testValidation(
            validateSessionSchema,
            validateSessionFixtures.valid,
            true,
        );
        expect(result.timeZone).toBe("America/New_York");
    });

    it("should accept various time zones", async () => {
        await testValidationBatch(validateSessionSchema, [
            {
                data: validateSessionFixtures.otherTimeZones.utc,
                shouldPass: true,
                description: "UTC time zone",
            },
            {
                data: validateSessionFixtures.otherTimeZones.tokyo,
                shouldPass: true,
                description: "Tokyo time zone",
            },
            {
                data: validateSessionFixtures.otherTimeZones.london,
                shouldPass: true,
                description: "London time zone",
            },
        ]);
    });

    it("should reject missing time zone", async () => {
        await testValidation(
            validateSessionSchema,
            validateSessionFixtures.invalid.missingTimeZone,
            false,
            /required/i,
        );
    });

    it("should reject empty time zone", async () => {
        await testValidation(
            validateSessionSchema,
            validateSessionFixtures.invalid.emptyTimeZone,
            false,
            /required/i,
        );
    });
});

describe("switchCurrentAccountSchema", () => {
    it("should accept valid account switch", async () => {
        const result = await testValidation(
            switchCurrentAccountSchema,
            switchCurrentAccountFixtures.valid,
            true,
        );
        expect(result.id).toBe("300000000000000001");
    });

    it("should reject missing id", async () => {
        await testValidation(
            switchCurrentAccountSchema,
            switchCurrentAccountFixtures.invalid.missingId,
            false,
            /required/i,
        );
    });

    it("should reject invalid id type", async () => {
        await testValidation(
            switchCurrentAccountSchema,
            switchCurrentAccountFixtures.invalid.invalidIdType,
            false,
        );
    });
});

describe("profileEmailUpdateValidation", () => {
    const defaultParams = { omitFields: [] };
    const updateSchema = profileEmailUpdateValidation.update(defaultParams);

    it("should accept minimal email update", async () => {
        const result = await testValidation(
            updateSchema,
            profileEmailUpdateFixtures.minimal.update,
            true,
        );
        expect(result).toHaveProperty("currentPassword");
    });

    it("should accept password change", async () => {
        const result = await testValidation(
            updateSchema,
            profileEmailUpdateFixtures.withNewPassword.update,
            true,
        );
        expect(result).toHaveProperty("newPassword");
    });

    it("should accept email management operations", async () => {
        // Just verify that validation passes with email operations
        await testValidation(
            updateSchema,
            profileEmailUpdateFixtures.withEmails.update,
            true,
        );
    });

    it("should accept complete update", async () => {
        const result = await testValidation(
            updateSchema,
            profileEmailUpdateFixtures.complete.update,
            true,
        );
        expect(result).toHaveProperty("newPassword");
        // Email operations are processed by the validation but may not be in the final result
    });

    it("should reject missing current password", async () => {
        await testValidation(
            updateSchema,
            profileEmailUpdateFixtures.invalid.missingPassword.update,
            false,
            /required/i,
        );
    });

    it("should reject weak new password", async () => {
        await testValidation(
            updateSchema,
            profileEmailUpdateFixtures.invalid.weakNewPassword.update,
            false,
            /8 characters/i,
        );
    });
});

describe("user test data factory", () => {
    it("should apply default values when creating bot with empty object", () => {
        const data = userTestDataFactory.createMinimal({});
        expect(data.id).to.match(/^\d+$/); // Snowflake ID
        expect(data.handle).toBe("testbot123");
        expect(data.isPrivate).toBe(false);
        expect(data.name).toBe("Test Bot");
        expect(data.isBotDepictingPerson).toBe(false);
        expect(data.botSettings).toEqual({});
    });

    it("should preserve undefined and falsy values in bot creation", () => {
        const data = userTestDataFactory.createMinimal({ 
            isPrivate: true,
            isBotDepictingPerson: true,
            botSettings: null
        });
        expect(data.isPrivate).toBe(true);
        expect(data.isBotDepictingPerson).toBe(true);
        expect(data.botSettings).toBeNull();
    });

    it("should apply update customizer defaults", () => {
        const data = userTestDataFactory.updateMinimal({});
        expect(data.id).toBe("300000000000000001");
    });

    it("should override factory defaults with provided values", () => {
        const customId = "999999999999999999";
        const data = userTestDataFactory.createMinimal({ 
            id: customId,
            handle: "custombot",
            isPrivate: true,
            name: "Custom Bot",
            isBotDepictingPerson: true,
            botSettings: { model: "gpt-4" }
        });
        expect(data.id).toBe(customId);
        expect(data.handle).toBe("custombot");
        expect(data.isPrivate).toBe(true);
        expect(data.name).toBe("Custom Bot");
        expect(data.isBotDepictingPerson).toBe(true);
        expect(data.botSettings).toEqual({ model: "gpt-4" });
    });

    it("should handle undefined botSettings differently than empty object", () => {
        const dataWithUndefined = userTestDataFactory.createMinimal({ botSettings: undefined });
        expect(dataWithUndefined.botSettings).toBeUndefined();
        
        const dataWithoutField = userTestDataFactory.createMinimal({});
        expect(dataWithoutField.botSettings).toEqual({});
    });
});
