import { describe } from "vitest";
import { apiKeyExternalFixtures, apiKeyExternalTestDataFactory } from "../../__test/fixtures/api/apiKeyExternalFixtures.js";
import { runComprehensiveValidationTests } from "./__test/validationTestUtils.js";
import { apiKeyExternalValidation } from "./apiKeyExternal.js";

describe("apiKeyExternalValidation", () => {
    runComprehensiveValidationTests(
        apiKeyExternalValidation,
        apiKeyExternalFixtures,
        apiKeyExternalTestDataFactory,
        "apiKeyExternal",
    );

    // No additional business logic tests needed - apiKeyExternal follows standard validation patterns
}); 
