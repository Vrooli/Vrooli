import { describe, expect, it } from "vitest";
import { walletFixtures } from "../../__test/fixtures/api-inputs/walletFixtures.js";
import { runStandardValidationTests } from "./__test/validationTestUtils.js";
import { walletValidation } from "./wallet.js";

describe("walletValidation", () => {
    // Run standard validation tests (wallet only supports update, not create)
    runStandardValidationTests(
        walletValidation,
        walletFixtures,
        "wallet",
    );

    describe("business logic validation", () => {
        it("should not have create method", () => {
            // Business rule: wallets cannot be created via API (only system-generated)
            expect(walletValidation.create).toBeUndefined();
        });

        // Other tests removed - covered by standard validation:
        // - empty string transformations are standard behavior covered by fixtures
    });
});
