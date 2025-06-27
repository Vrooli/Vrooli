import { describe, expect, it } from "vitest";
import { reportResponseFixtures, reportResponseTestDataFactory } from "../../__test/fixtures/api-inputs/reportResponseFixtures.js";
import { ReportSuggestedAction } from "../../api/types.js";
import { runComprehensiveValidationTests, testValidation } from "./__test/validationTestUtils.js";
import { reportResponseValidation } from "./reportResponse.js";

describe("reportResponseValidation", () => {
    // Run standard validation tests
    runComprehensiveValidationTests(
        reportResponseValidation,
        reportResponseFixtures,
        reportResponseTestDataFactory,
        "reportResponse",
    );

    describe("business logic validation", () => {
        const defaultParams = { omitFields: [] };

        it("should not allow updating reportConnect in update", async () => {
            // Field immutability: reportConnect cannot be changed after creation
            const updateSchema = reportResponseValidation.update(defaultParams);
            const dataWithRestrictedFields = {
                id: "123456789012345678",
                reportConnect: "123456789012345679",
                actionSuggested: ReportSuggestedAction.Delete,
            };

            const result = await testValidation(updateSchema, dataWithRestrictedFields, true);
            expect(result).to.not.have.property("reportConnect");
        });

        // Other tests removed - covered by standard validation:
        // - details length validation is in fixtures
        // - empty string transformation is standard behavior
    });
});
