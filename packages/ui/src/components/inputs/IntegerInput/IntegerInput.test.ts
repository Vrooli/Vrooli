import { Palette } from "@mui/material";
import { calculateUpdatedNumber, getColorForLabel, getNumberInRange } from "./IntegerInput";

describe("getNumberInRange", () => {
    test("should return the original number if it is within the range", () => {
        expect(getNumberInRange(10, 20, 5)).toBe(10);
    });

    test("should return the maximum limit if the number exceeds the range", () => {
        expect(getNumberInRange(25, 20, 5)).toBe(20);
    });

    test("should return the minimum limit if the number is below the range", () => {
        expect(getNumberInRange(3, 20, 5)).toBe(5);
    });

    test("should handle negative numbers correctly", () => {
        expect(getNumberInRange(-10, 0, -20)).toBe(-10);
        expect(getNumberInRange(-25, 0, -20)).toBe(-20);
        expect(getNumberInRange(10, 0, -20)).toBe(0);
    });

    test("should correctly handle the minimum and maximum being the same", () => {
        expect(getNumberInRange(5, 10, 10)).toBe(10);
        expect(getNumberInRange(15, 10, 10)).toBe(10);
    });

    test("should work with zero as boundaries", () => {
        expect(getNumberInRange(-1, 0, -10)).toBe(-1);
        expect(getNumberInRange(1, 0, -10)).toBe(0);
    });

    test("should return the boundary when the number is exactly at the boundary", () => {
        expect(getNumberInRange(20, 20, 10)).toBe(20);
        expect(getNumberInRange(10, 20, 10)).toBe(10);
    });

    test("should work with floating point numbers", () => {
        expect(getNumberInRange(10.5, 20.5, 10.5)).toBe(10.5);
        expect(getNumberInRange(21.1, 20.5, 10.5)).toBe(20.5);
        expect(getNumberInRange(9.9, 20.5, 10.5)).toBe(10.5);
    });
});

describe("calculateUpdatedNumber", () => {
    test("should handle non-numeric input by returning the minimum value", () => {
        expect(calculateUpdatedNumber("hello", 20, 5)).toBe(5);
    });

    test("should handle an empty string by returning the minimum value", () => {
        expect(calculateUpdatedNumber("", 20, 5)).toBe(5);
    });

    test("should convert numeric string input within the range correctly", () => {
        expect(calculateUpdatedNumber("10", 20, 5)).toBe(10);
    });

    test("should handle numeric string exceeding the maximum limit", () => {
        expect(calculateUpdatedNumber("25", 20, 5)).toBe(20);
    });

    test("should handle numeric string below the minimum limit", () => {
        expect(calculateUpdatedNumber("3", 20, 5)).toBe(5);
    });

    test("should round the result if decimals are not allowed", () => {
        expect(calculateUpdatedNumber("15.7", 20, 5, false)).toBe(16);
        expect(calculateUpdatedNumber("15.2", 20, 5, false)).toBe(15);
    });

    test("should not round the result if decimals are allowed", () => {
        expect(calculateUpdatedNumber("15.7", 20, 5, true)).toBe(15.7);
        expect(calculateUpdatedNumber("15.2", 20, 5, true)).toBe(15.2);
    });

    test("should handle negative numbers correctly", () => {
        expect(calculateUpdatedNumber("-10", 0, -20)).toBe(-10);
        expect(calculateUpdatedNumber("-25", 0, -20, true)).toBe(-20);
        expect(calculateUpdatedNumber("10", 0, -20, true)).toBe(0);
    });

    test("should handle floating point numbers and rounding appropriately", () => {
        expect(calculateUpdatedNumber("10.999", 20, 10, false)).toBe(11);
        expect(calculateUpdatedNumber("10.001", 20, 10, false)).toBe(10);
        expect(calculateUpdatedNumber("10.999", 20, 10, true)).toBe(10.999);
        expect(calculateUpdatedNumber("10.001", 20, 10, true)).toBe(10.001);
    });
});

describe("getColorForLabel", () => {
    const palette = {
        error: { main: "red" },
        warning: { main: "orange" },
        background: { textSecondary: "grey" },
    } as unknown as Palette;

    test("should return error color if value is less than min", () => {
        expect(getColorForLabel(4, 5, 10, palette, undefined)).toBe("red");
    });

    test("should return error color if value is greater than max", () => {
        expect(getColorForLabel(11, 5, 10, palette, undefined)).toBe("red");
    });

    test("should return warning color if value is equal to min", () => {
        expect(getColorForLabel(5, 5, 10, palette, undefined)).toBe("orange");
    });

    test("should return warning color if value is equal to max", () => {
        expect(getColorForLabel(10, 5, 10, palette, undefined)).toBe("orange");
    });

    test("should return secondary text color if value is within the range", () => {
        expect(getColorForLabel(7, 5, 10, palette, undefined)).toBe("grey");
    });

    test("should treat the zeroText as zero and return the appropriate color", () => {
        expect(getColorForLabel("ZERO", -1, 5, palette, "ZERO")).toBe("grey");
        expect(getColorForLabel("ZERO", -5, 0, palette, "ZERO")).toBe("orange"); // Assuming ZERO is zero and zero is max
        expect(getColorForLabel("ZERO", 0, 10, palette, "ZERO")).toBe("orange"); // Assuming ZERO is zero and zero is min
    });

    test("should return error color if the non-numeric string cannot be converted to a valid number", () => {
        expect(getColorForLabel("non-numeric", 0, 10, palette, undefined)).toBe("red");
        expect(getColorForLabel("non-numeric", 0, 10, palette, "ZERO")).toBe("red");
    });

    test("should handle edge cases with negative numbers and zero boundaries", () => {
        expect(getColorForLabel(-1, -10, 0, palette, undefined)).toBe("grey");
        expect(getColorForLabel(-11, -10, 0, palette, undefined)).toBe("red");
        expect(getColorForLabel(1, -10, 0, palette, undefined)).toBe("red");
        expect(getColorForLabel(-10, -10, 0, palette, undefined)).toBe("orange");
        expect(getColorForLabel(0, -10, 0, palette, undefined)).toBe("orange");
    });
});
