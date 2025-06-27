import { describe, expect, it } from "vitest";
import { bookmarkFixtures, bookmarkTestDataFactory } from "../../__test/fixtures/api-inputs/bookmarkFixtures.js";
import { BookmarkFor } from "../../api/types.js";
import { runComprehensiveValidationTests, testValidation } from "./__test/validationTestUtils.js";
import { bookmarkValidation } from "./bookmark.js";

describe("bookmarkValidation", () => {
    // Run comprehensive validation tests using shared fixtures
    runComprehensiveValidationTests(
        bookmarkValidation,
        bookmarkFixtures,
        bookmarkTestDataFactory,
        "bookmark",
    );

    // Business logic validation - relationship exclusivity and update immutability
    describe("bookmark-specific business logic", () => {
        const defaultParams = { omitFields: [] };
        const createSchema = bookmarkValidation.create(defaultParams);
        const updateSchema = bookmarkValidation.update(defaultParams);

        it("should enforce exclusivity between listConnect and listCreate", async () => {
            // Business rule: Can't have both connect and create for list
            const dataWithBoth = {
                ...bookmarkFixtures.minimal.create,
                listConnect: "123456789012345680",
                listCreate: {
                    id: "123456789012345681",
                    label: "New List",
                },
            };

            await testValidation(
                createSchema,
                dataWithBoth,
                false,
                /Only one of the following fields can be present/i,
            );
        });

        it("should not allow updates to immutable fields", async () => {
            // Business rule: bookmarkFor and forConnect are immutable after creation
            const scenarios = [
                {
                    data: { id: "123456789012345678", bookmarkFor: BookmarkFor.Tag },
                    shouldPass: true,
                    description: "bookmarkFor update attempt",
                },
                {
                    data: { id: "123456789012345678", forConnect: "123456789012345679" },
                    shouldPass: true,
                    description: "forConnect update attempt",
                },
            ];

            for (const scenario of scenarios) {
                const result = await testValidation(updateSchema, scenario.data, scenario.shouldPass);
                // These fields should be stripped out in updates
                expect(result).to.not.have.property("bookmarkFor");
                expect(result).to.not.have.property("forConnect");
            }
        });
    });
});
