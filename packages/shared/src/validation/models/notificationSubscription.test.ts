import { describe } from "vitest";
import { notificationSubscriptionFixtures, notificationSubscriptionTestDataFactory } from "../../__test/fixtures/api/notificationSubscriptionFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { notificationSubscriptionValidation } from "./notificationSubscription.js";

describe("notificationSubscriptionValidation", () => {
    // Run standard validation tests
    runComprehensiveValidationTests(
        notificationSubscriptionValidation,
        notificationSubscriptionFixtures,
        notificationSubscriptionTestDataFactory,
        "notificationSubscription",
    );

    // No additional business logic tests needed - all validation is basic field validation
    // covered by standard tests and fixtures (subscription types, silent settings)
});
