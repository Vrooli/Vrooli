import { expect } from "chai";
import { generatePK, generatePKString, validateSnowflakeId } from "./snowflake.js";

describe("Snowflake IDs", () => {
    it("generates a valid Snowflake ID as bigint", () => {
        const id = generatePK();
        expect(typeof id).to.equal("bigint");
        expect(validateSnowflakeId(id)).to.equal(true);
    });

    it("generates a valid Snowflake ID string", () => {
        const id = generatePKString();
        expect(typeof id).to.equal("string");
        expect(validateSnowflakeId(id)).to.equal(true);
    });

    it("generates unique IDs", () => {
        const ids = new Set();
        // Generate 1000 IDs
        const COUNT = 1000;
        for (let i = 0; i < COUNT; i++) {
            ids.add(generatePKString());
        }
        expect(ids.size).to.equal(COUNT);
    });

    it("validates Snowflake IDs correctly", () => {
        // Valid ID
        const validId = generatePKString();
        expect(validateSnowflakeId(validId)).to.equal(true);

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
            expect(validateSnowflakeId(id)).to.equal(false);
        });
    });
});