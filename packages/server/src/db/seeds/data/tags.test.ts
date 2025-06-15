import { expect, describe, it } from "vitest";
import { data } from "./tags.js";

const MIN_TAG_LENGTH = 2;
const MAX_TAG_LENGTH = 64;
const MIN_DESCRIPTION_LENGTH = 1;
const MAX_DESCRIPTION_LENGTH = 2048;

/**
 * Test suite for validating seeded code test cases.
 */
describe("Seeded Tag Tests", () => {
    // Iterate over eacht ag in the seed data
    data.forEach((tag, index) => {
        describe(`Tag ${index + 1}: ${tag[0]}`, () => {
            it("should be in the correct format", () => {
                expect(tag).to.be.an("array").with.lengthOf(2);
                expect(tag[0].length).to.be.at.least(MIN_TAG_LENGTH);
                expect(tag[0].length).to.be.at.most(MAX_TAG_LENGTH);
                expect(tag[1].length).to.be.at.least(MIN_DESCRIPTION_LENGTH);
                expect(tag[1].length).to.be.at.most(MAX_DESCRIPTION_LENGTH);
            });
        });
    });
});
