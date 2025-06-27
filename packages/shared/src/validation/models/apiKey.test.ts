import { describe, expect, it } from "vitest";
import { apiKeyFixtures, apiKeyTestDataFactory } from "../../__test/fixtures/api-inputs/apiKeyFixtures.js";
import { runComprehensiveValidationTests, testValidation } from "./__test/validationTestUtils.js";
import { apiKeyValidation } from "./apiKey.js";

describe("apiKeyValidation", () => {
    runComprehensiveValidationTests(
        apiKeyValidation,
        apiKeyFixtures,
        apiKeyTestDataFactory,
        "apiKey",
    );

    // API Key-specific business logic tests
    describe("business logic validation", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = apiKeyValidation.create(defaultParams);

        describe("permissions field validation", () => {
            it("should validate JSON permissions structure", async () => {
                // API Key business rule: permissions stored as JSON string
                const result = await testValidation(
                    createSchema,
                    apiKeyFixtures.complete.create,
                    true,
                );
                expect(result.permissions).to.be.a("string");
                const parsed = JSON.parse(result.permissions);
                expect(parsed).toHaveProperty("read", true);
                expect(parsed).toHaveProperty("write", true);
                expect(parsed).toHaveProperty("delete", false);
            });
        });
    });
});
