import { describe, expect, it } from "vitest";
import { issueFixtures, issueTestDataFactory } from "../../__test/fixtures/api-inputs/issueFixtures.js";
import { runComprehensiveValidationTests, testValidation } from "./__test/validationTestUtils.js";
import { issueValidation } from "./issue.js";

describe("issueValidation", () => {
    // Run standard validation tests
    runComprehensiveValidationTests(
        issueValidation,
        issueFixtures,
        issueTestDataFactory,
        "issue",
    );

    describe("field immutability in updates", () => {
        it("should not allow updates to immutable fields", async () => {
            const updateSchema = issueValidation.update({ omitFields: [] });

            // Business rule: issueFor and forConnect are immutable after creation
            const immutableFieldTests = [
                { field: "issueFor", value: "User" },
                { field: "forConnect", value: "123456789012345679" },
            ];

            for (const test of immutableFieldTests) {
                const updateData = {
                    id: "123456789012345678",
                    [test.field]: test.value,
                };

                const result = await testValidation(updateSchema, updateData, true);
                // These fields should be stripped out in updates
                expect(result).to.not.have.property(test.field);
            }
        });

    });
});
