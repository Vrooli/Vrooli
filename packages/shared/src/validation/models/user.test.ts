import { describe } from "vitest";
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
    userTranslationTestDataFactory,
    validateSessionFixtures,
} from "../../__test/fixtures/api/userFixtures.js";
import { runComprehensiveValidationTests, runStandardValidationTests } from "./__test/validationTestUtils.js";
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
    runComprehensiveValidationTests(
        userValidation,
        userFixtures,
        userTestDataFactory,
        "user",
    );

    // No additional business logic tests needed - bot creation validation is basic field validation
    // covered by fixtures (handle, isPrivate, name requirements, botSettings structure)
});

describe("userTranslationValidation", () => {
    runComprehensiveValidationTests(
        userTranslationValidation,
        userTranslationFixtures,
        userTranslationTestDataFactory,
        "userTranslation",
    );
});

describe("profileValidation", () => {
    runStandardValidationTests(
        profileValidation,
        userFixtures,
        "profile",
    );
});

describe("profileEmailUpdateValidation", () => {
    runStandardValidationTests(
        profileEmailUpdateValidation,
        profileEmailUpdateFixtures,
        "profileEmailUpdate",
    );
});

// Auth-related validation schemas
describe("emailLogInSchema", () => {
    runStandardValidationTests(
        { create: () => emailLogInSchema },
        emailLogInFixtures,
        "emailLogIn",
    );
});

describe("emailRequestPasswordChangeSchema", () => {
    runStandardValidationTests(
        { create: () => emailRequestPasswordChangeSchema },
        emailRequestPasswordChangeFixtures,
        "emailRequestPasswordChange",
    );
});

describe("emailResetPasswordSchema", () => {
    runStandardValidationTests(
        { create: () => emailResetPasswordSchema },
        emailResetPasswordFixtures,
        "emailResetPassword",
    );
});

describe("emailResetPasswordFormSchema", () => {
    runStandardValidationTests(
        { create: () => emailResetPasswordFormSchema },
        emailResetPasswordFormFixtures,
        "emailResetPasswordForm",
    );
});

describe("switchCurrentAccountSchema", () => {
    runStandardValidationTests(
        { create: () => switchCurrentAccountSchema },
        switchCurrentAccountFixtures,
        "switchCurrentAccount",
    );
});

describe("userDeleteOneSchema", () => {
    runStandardValidationTests(
        { create: () => userDeleteOneSchema },
        userDeleteOneFixtures,
        "userDeleteOne",
    );
});

describe("validateSessionSchema", () => {
    runStandardValidationTests(
        { create: () => validateSessionSchema },
        validateSessionFixtures,
        "validateSession",
    );
});
