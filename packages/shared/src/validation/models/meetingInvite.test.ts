import { describe, expect, it } from "vitest";
import { meetingInviteFixtures, meetingInviteTestDataFactory } from "../../__test/fixtures/api/meetingInviteFixtures.js";
import { runComprehensiveValidationTests, testValidation } from "./__test/validationTestUtils.js";
import { meetingInviteValidation } from "./meetingInvite.js";

describe("meetingInviteValidation", () => {
    runComprehensiveValidationTests(
        meetingInviteValidation,
        meetingInviteFixtures,
        meetingInviteTestDataFactory,
        "meetingInvite",
    );

    // Business logic validation - field immutability
    describe("field immutability in updates", () => {
        it("should not allow updates to immutable fields", async () => {
            const updateSchema = meetingInviteValidation.update({ omitFields: [] });

            // Business rule: meetingConnect and userConnect are immutable after creation
            const immutableFieldTests = [
                { field: "meetingConnect", value: "123456789012345679" },
                { field: "userConnect", value: "123456789012345679" },
            ];

            for (const test of immutableFieldTests) {
                const updateData = {
                    id: "123456789012345678",
                    [test.field]: test.value,
                };

                const result = await testValidation(updateSchema, updateData, true);
                expect(result).to.not.have.property(test.field);
            }
        });
    });
});
