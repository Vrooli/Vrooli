import { describe, it, expect } from "vitest";
import { ALPHABET, generatePublicId, validatePublicId } from "./publicId.js";

/**
 * Test the collision rate of the public ID generator
 * @param count Number of IDs to generate
 * @returns Number of collisions
 */
function testCollisionRate(count: number): number {
    const ids = new Set<string>();
    let collisions = 0;

    for (let i = 0; i < count; i++) {
        const id = generatePublicId();
        if (ids.has(id)) {
            collisions++;
        } else {
            ids.add(id);
        }
    }

    return collisions;
}

describe("Public ID Functions", () => {
    it("generates a valid public ID", () => {
        const id = generatePublicId();
        expect(typeof id).toBe("string");
        expect(validatePublicId(id)).toBe(true);
        expect(id.length).toBe(12);
    });

    it("generates a valid public ID with custom length", () => {
        const length = 10;
        const id = generatePublicId(length);
        expect(typeof id).toBe("string");
        expect(validatePublicId(id)).toBe(true);
        expect(id.length).toBe(length);
    });

    it("generates URL-friendly IDs", () => {
        const id = generatePublicId();
        expect(/^[0-9a-z]+$/.test(id)).toBe(true);
        // Verify it only uses our defined alphabet
        expect(new RegExp(`^[${ALPHABET}]+$`).test(id)).toBe(true);
    });

    it("has low collision rate", () => {
        const SAMPLE_SIZE = 10000;
        const collisions = testCollisionRate(SAMPLE_SIZE);
        expect(collisions).toBe(0);
    });

    it("validates public IDs correctly", () => {
        // Valid ID
        const validId = generatePublicId();
        expect(validatePublicId(validId)).toBe(true);

        // Invalid IDs
        const invalidIds = [
            "ABC", // Contains uppercase letters
            "123456789", // Too short
            "1234567890123", // Too long
            "abcd!123456", // Contains non-alphanumeric characters
            "", // Empty string
            null, // Null
            undefined, // Undefined
            123, // Number (not string)
            {}, // Object
        ];

        invalidIds.forEach(id => {
            expect(validatePublicId(id)).toBe(false);
        });
    });
}); 
