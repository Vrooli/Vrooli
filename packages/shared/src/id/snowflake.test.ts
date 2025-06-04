import { expect } from "chai";
import { generatePK, validatePK } from "./snowflake.js";

describe("Snowflake IDs", () => {
    it("generates a valid Snowflake ID as bigint", () => {
        const id = generatePK();
        expect(typeof id).to.equal("bigint");
        expect(validatePK(id)).to.equal(true);
    });

    it("generates a valid Snowflake ID string", () => {
        const id = generatePK().toString();
        expect(validatePK(id)).to.equal(true);
    });

    it("generates unique IDs", () => {
        const ids = new Set();
        // Generate 1000 IDs
        const COUNT = 1000;
        for (let i = 0; i < COUNT; i++) {
            ids.add(generatePK().toString());
        }
        expect(ids.size).to.equal(COUNT);
    });

    it("validates Snowflake IDs correctly", () => {
        // Valid ID
        const validId = generatePK().toString();
        expect(validatePK(validId)).to.equal(true);

        // Invalid IDs
        const invalidIds = [
            "abc", // Non-numeric
            "-123", // Negative
            "", // Empty string
            "18446744073709551616", // Too large (2^64)
            null, // Null
            undefined, // Undefined
            123, // Number (not string or bigint)
            {}, // Object
        ];

        invalidIds.forEach(id => {
            expect(validatePK(id)).to.equal(false);
        });
    });
});
