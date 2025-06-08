import type { Palette } from "@mui/material";
import { expect } from "chai";
import { calculateUpdatedNumber, getColorForLabel, getNumberInRange } from "./IntegerInput.js";

describe("getNumberInRange", () => {
    it("should return the original number if it is within the range", () => {
        expect(getNumberInRange(10, 20, 5)).to.equal(10);
    });

    it("should return the maximum limit if the number exceeds the range", () => {
        expect(getNumberInRange(25, 20, 5)).to.equal(20);
    });

    it("should return the minimum limit if the number is below the range", () => {
        expect(getNumberInRange(3, 20, 5)).to.equal(5);
    });

    it("should handle negative numbers correctly", () => {
        expect(getNumberInRange(-10, 0, -20)).to.equal(-10);
        expect(getNumberInRange(-25, 0, -20)).to.equal(-20);
        expect(getNumberInRange(10, 0, -20)).to.equal(0);
    });

    it("should correctly handle the minimum and maximum being the same", () => {
        expect(getNumberInRange(5, 10, 10)).to.equal(10);
        expect(getNumberInRange(15, 10, 10)).to.equal(10);
    });

    it("should work with zero as boundaries", () => {
        expect(getNumberInRange(-1, 0, -10)).to.equal(-1);
        expect(getNumberInRange(1, 0, -10)).to.equal(0);
    });

    it("should return the boundary when the number is exactly at the boundary", () => {
        expect(getNumberInRange(20, 20, 10)).to.equal(20);
        expect(getNumberInRange(10, 20, 10)).to.equal(10);
    });

    it("should work with floating point numbers", () => {
        expect(getNumberInRange(10.5, 20.5, 10.5)).to.equal(10.5);
        expect(getNumberInRange(21.1, 20.5, 10.5)).to.equal(20.5);
        expect(getNumberInRange(9.9, 20.5, 10.5)).to.equal(10.5);
    });
});

describe("calculateUpdatedNumber", () => {
    it("should handle non-numeric input by returning the minimum value", () => {
        expect(calculateUpdatedNumber("hello", 20, 5)).to.equal(5);
    });

    it("should handle an empty string by returning the minimum value", () => {
        expect(calculateUpdatedNumber("", 20, 5)).to.equal(5);
    });

    it("should convert numeric string input within the range correctly", () => {
        expect(calculateUpdatedNumber("10", 20, 5)).to.equal(10);
    });

    it("should handle numeric string exceeding the maximum limit", () => {
        expect(calculateUpdatedNumber("25", 20, 5)).to.equal(20);
    });

    it("should handle numeric string below the minimum limit", () => {
        expect(calculateUpdatedNumber("3", 20, 5)).to.equal(5);
    });

    it("should round the result if decimals are not allowed", () => {
        expect(calculateUpdatedNumber("15.7", 20, 5, false)).to.equal(16);
        expect(calculateUpdatedNumber("15.2", 20, 5, false)).to.equal(15);
    });

    it("should not round the result if decimals are allowed", () => {
        expect(calculateUpdatedNumber("15.7", 20, 5, true)).to.equal(15.7);
        expect(calculateUpdatedNumber("15.2", 20, 5, true)).to.equal(15.2);
    });

    it("should handle negative numbers correctly", () => {
        expect(calculateUpdatedNumber("-10", 0, -20)).to.equal(-10);
        expect(calculateUpdatedNumber("-25", 0, -20, true)).to.equal(-20);
        expect(calculateUpdatedNumber("10", 0, -20, true)).to.equal(0);
    });

    it("should handle floating point numbers and rounding appropriately", () => {
        expect(calculateUpdatedNumber("10.999", 20, 10, false)).to.equal(11);
        expect(calculateUpdatedNumber("10.001", 20, 10, false)).to.equal(10);
        expect(calculateUpdatedNumber("10.999", 20, 10, true)).to.equal(10.999);
        expect(calculateUpdatedNumber("10.001", 20, 10, true)).to.equal(10.001);
    });
});

describe("getColorForLabel", () => {
    const palette = {
        error: { main: "red" },
        warning: { main: "orange" },
        background: { textSecondary: "grey" },
    } as unknown as Palette;

    it("should return error color if value is less than min", () => {
        expect(getColorForLabel(4, 5, 10, palette, undefined)).to.equal("red");
    });

    it("should return error color if value is greater than max", () => {
        expect(getColorForLabel(11, 5, 10, palette, undefined)).to.equal("red");
    });

    it("should return warning color if value is equal to min", () => {
        expect(getColorForLabel(5, 5, 10, palette, undefined)).to.equal("orange");
    });

    it("should return warning color if value is equal to max", () => {
        expect(getColorForLabel(10, 5, 10, palette, undefined)).to.equal("orange");
    });

    it("should return secondary text color if value is within the range", () => {
        expect(getColorForLabel(7, 5, 10, palette, undefined)).to.equal("grey");
    });

    it("should treat the zeroText as zero and return the appropriate color", () => {
        expect(getColorForLabel("ZERO", -1, 5, palette, "ZERO")).to.equal("grey");
        expect(getColorForLabel("ZERO", -5, 0, palette, "ZERO")).to.equal("orange"); // Assuming ZERO is zero and zero is max
        expect(getColorForLabel("ZERO", 0, 10, palette, "ZERO")).to.equal("orange"); // Assuming ZERO is zero and zero is min
    });

    it("should return error color if the non-numeric string cannot be converted to a valid number", () => {
        expect(getColorForLabel("non-numeric", 0, 10, palette, undefined)).to.equal("red");
        expect(getColorForLabel("non-numeric", 0, 10, palette, "ZERO")).to.equal("red");
    });

    it("should handle edge cases with negative numbers and zero boundaries", () => {
        expect(getColorForLabel(-1, -10, 0, palette, undefined)).to.equal("grey");
        expect(getColorForLabel(-11, -10, 0, palette, undefined)).to.equal("red");
        expect(getColorForLabel(1, -10, 0, palette, undefined)).to.equal("red");
        expect(getColorForLabel(-10, -10, 0, palette, undefined)).to.equal("orange");
        expect(getColorForLabel(0, -10, 0, palette, undefined)).to.equal("orange");
    });
});
