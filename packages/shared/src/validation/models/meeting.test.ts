import { describe, expect, it } from "vitest";
import { meetingFixtures, meetingTestDataFactory } from "../../__test/fixtures/api-inputs/meetingFixtures.js";
import { runComprehensiveValidationTests, testValidation } from "./__test/validationTestUtils.js";
import { meetingValidation } from "./meeting.js";

describe("meetingValidation", () => {
    runComprehensiveValidationTests(
        meetingValidation,
        meetingFixtures,
        meetingTestDataFactory,
        "meeting",
    );

    // Business logic validation - field immutability
    describe("field immutability in updates", () => {
        it("should not allow teamConnect updates", async () => {
            const updateSchema = meetingValidation.update({ omitFields: [] });

            // Business rule: teamConnect is immutable after creation
            const updateData = {
                id: "123456789012345678",
                teamConnect: "123456789012345679",
            };

            const result = await testValidation(updateSchema, updateData, true);
            expect(result).to.not.have.property("teamConnect");
        });
    });
});
