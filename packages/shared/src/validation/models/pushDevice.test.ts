import { describe } from "vitest";
import { pushDeviceFixtures, pushDeviceTestDataFactory } from "../../__test/fixtures/api/pushDeviceFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { pushDeviceValidation } from "./pushDevice.js";

describe("pushDeviceValidation", () => {
    // Run standard validation tests
    runComprehensiveValidationTests(
        pushDeviceValidation,
        pushDeviceFixtures,
        pushDeviceTestDataFactory,
        "pushDevice",
    );

    // No additional business logic tests needed - all validation is basic field validation
    // covered by standard tests and fixtures (endpoint validation, p256dh/auth keys)
});
