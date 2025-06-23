import { describe, expect, it } from "vitest";
import { commentFixtures, commentTestDataFactory } from "../../__test/fixtures/api/commentFixtures.js";
import { CommentFor } from "../../api/types.js";
import { runComprehensiveValidationTests, testValidation } from "./__test/validationTestUtils.js";
import { commentValidation } from "./comment.js";

describe("commentValidation", () => {
    runComprehensiveValidationTests(
        commentValidation,
        commentFixtures,
        commentTestDataFactory,
        "comment",
    );

    // Business logic validation - field immutability
    describe("field immutability in updates", () => {
        const updateSchema = commentValidation.update({ omitFields: [] });

        it("should not allow updates to immutable fields", async () => {
            // Business rule: createdFor, forConnect, and parentConnect are immutable after creation
            const immutableFieldTests = [
                { field: "createdFor", value: CommentFor.Issue },
                { field: "forConnect", value: "123456789012345679" },
                { field: "parentConnect", value: "123456789012345679" },
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
