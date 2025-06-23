import { describe, expect, it } from "vitest";
import { noNull, noEmptyString, validNumber, toBool } from "./noNull.js";

describe("noNull", () => {
    describe("basic functionality", () => {
        it("should return the first non-null, non-undefined value", () => {
            expect(noNull(null, undefined, "first")).toBe("first");
            expect(noNull(undefined, "second", "third")).toBe("second");
            expect(noNull("immediate")).toBe("immediate");
        });

        it("should return undefined when all values are null or undefined", () => {
            expect(noNull(null, undefined, null)).toBeUndefined();
            expect(noNull()).toBeUndefined();
            expect(noNull(undefined)).toBeUndefined();
            expect(noNull(null)).toBeUndefined();
        });

        it("should work with different types", () => {
            expect(noNull(null, 42, "string")).toBe(42);
            expect(noNull(undefined, true, false)).toBe(true);
            expect(noNull(null, { key: "value" }, [])).toEqual({ key: "value" });
            expect(noNull(undefined, [], {})).toEqual([]);
        });

        it("should return falsy values that are not null or undefined", () => {
            expect(noNull(null, 0, 1)).toBe(0);
            expect(noNull(undefined, false, true)).toBe(false);
            expect(noNull(null, "", "string")).toBe("");
            expect(noNull(undefined, NaN, 5)).toBeNaN();
        });

        it("should work with complex types", () => {
            const obj = { a: 1, b: null };
            const arr = [1, 2, 3];
            const func = () => "test";
            
            expect(noNull(null, obj, arr)).toBe(obj);
            expect(noNull(undefined, arr, func)).toBe(arr);
            expect(noNull(null, undefined, func, "string")).toBe(func);
        });
    });

    describe("edge cases", () => {
        it("should handle single argument", () => {
            expect(noNull("single")).toBe("single");
            expect(noNull(null)).toBeUndefined();
            expect(noNull(undefined)).toBeUndefined();
        });

        it("should handle many arguments", () => {
            expect(noNull(null, undefined, null, undefined, null, "found")).toBe("found");
            expect(noNull(...Array(10).fill(null), "last")).toBe("last");
        });

        it("should preserve object references", () => {
            const obj1 = { test: 1 };
            const obj2 = { test: 2 };
            const result = noNull(null, obj1, obj2);
            expect(result).toBe(obj1);
            expect(result).not.toBe(obj2);
        });
    });
});

describe("noEmptyString", () => {
    describe("basic functionality", () => {
        it("should return the first non-empty string", () => {
            expect(noEmptyString("", "first", "second")).toBe("first");
            expect(noEmptyString(null, "second", "third")).toBe("second");
            expect(noEmptyString(undefined, "immediate")).toBe("immediate");
        });

        it("should return undefined when no valid strings found", () => {
            expect(noEmptyString("", null, undefined)).toBeUndefined();
            expect(noEmptyString()).toBeUndefined();
            expect(noEmptyString("")).toBeUndefined();
            expect(noEmptyString(null, undefined, "")).toBeUndefined();
        });

        it("should ignore non-string values", () => {
            expect(noEmptyString(42, true, {}, "found")).toBe("found");
            expect(noEmptyString([], 0, false, "string")).toBe("string");
            expect(noEmptyString(NaN, Symbol("test"), "result")).toBe("result");
        });

        it("should handle whitespace strings", () => {
            expect(noEmptyString("", " ", "second")).toBe(" ");
            expect(noEmptyString("", "\t", "second")).toBe("\t");
            expect(noEmptyString("", "\n", "second")).toBe("\n");
            expect(noEmptyString("", "   ", "second")).toBe("   ");
        });

        it("should return the first valid string, not the longest", () => {
            expect(noEmptyString("", "a", "longer string")).toBe("a");
            expect(noEmptyString("", "short", "x")).toBe("short");
        });
    });

    describe("edge cases", () => {
        it("should handle mixed types correctly", () => {
            expect(noEmptyString(0, false, null, undefined, "", "found")).toBe("found");
            expect(noEmptyString([], {}, () => {}, "string")).toBe("string");
        });

        it("should handle single argument", () => {
            expect(noEmptyString("valid")).toBe("valid");
            expect(noEmptyString("")).toBeUndefined();
            expect(noEmptyString(null)).toBeUndefined();
            expect(noEmptyString(42)).toBeUndefined();
        });

        it("should handle many arguments", () => {
            const args = [null, undefined, "", 0, false, {}, [], "finally"];
            expect(noEmptyString(...args)).toBe("finally");
        });

        it("should handle special string values", () => {
            expect(noEmptyString("", "0", "other")).toBe("0");
            expect(noEmptyString("", "false", "other")).toBe("false");
            expect(noEmptyString("", "null", "other")).toBe("null");
            expect(noEmptyString("", "undefined", "other")).toBe("undefined");
        });
    });
});

describe("validNumber", () => {
    describe("basic functionality", () => {
        it("should return the first valid finite number", () => {
            expect(validNumber(null, 42, 100)).toBe(42);
            expect(validNumber(undefined, "string", 3.14)).toBe(3.14);
            expect(validNumber(0, 1, 2)).toBe(0);
        });

        it("should return undefined when no valid numbers found", () => {
            expect(validNumber()).toBeUndefined();
            expect(validNumber(null, undefined, "string")).toBeUndefined();
            expect(validNumber(NaN, Infinity, -Infinity)).toBeUndefined();
            expect(validNumber({}, [], true)).toBeUndefined();
        });

        it("should reject non-finite numbers", () => {
            expect(validNumber(NaN, 42)).toBe(42);
            expect(validNumber(Infinity, 42)).toBe(42);
            expect(validNumber(-Infinity, 42)).toBe(42);
            expect(validNumber(NaN, Infinity, -Infinity)).toBeUndefined();
        });

        it("should handle negative numbers", () => {
            expect(validNumber(null, -42, 100)).toBe(-42);
            expect(validNumber(-0, 1)).toBe(-0);
            expect(validNumber(-3.14, 2.71)).toBe(-3.14);
        });

        it("should handle decimal numbers", () => {
            expect(validNumber("invalid", 3.14159, 2.71)).toBe(3.14159);
            expect(validNumber(null, 0.1, 0.2)).toBe(0.1);
            expect(validNumber(1.5, 2.5)).toBe(1.5);
        });

        it("should ignore numeric strings", () => {
            expect(validNumber("42", "3.14", 100)).toBe(100);
            expect(validNumber("0", 42)).toBe(42);
            expect(validNumber("123", "456")).toBeUndefined();
        });
    });

    describe("edge cases", () => {
        it("should handle special number values", () => {
            expect(validNumber(Number.MAX_VALUE, 42)).toBe(Number.MAX_VALUE);
            expect(validNumber(Number.MIN_VALUE, 42)).toBe(Number.MIN_VALUE);
            expect(validNumber(Number.EPSILON, 42)).toBe(Number.EPSILON);
        });

        it("should handle mixed types", () => {
            expect(validNumber(null, undefined, "", {}, [], false, true, 42)).toBe(42);
            expect(validNumber("string", Symbol("test"), () => {}, 3.14)).toBe(3.14);
        });

        it("should handle single argument", () => {
            expect(validNumber(42)).toBe(42);
            expect(validNumber(NaN)).toBeUndefined();
            expect(validNumber("string")).toBeUndefined();
        });

        it("should handle many arguments", () => {
            const args = [null, undefined, "string", {}, [], NaN, Infinity, -Infinity, false, true, 42];
            expect(validNumber(...args)).toBe(42);
        });

        it("should return exactly the input number (reference equality for objects)", () => {
            const num = 42;
            expect(validNumber(null, num)).toBe(num);
        });
    });
});

describe("toBool", () => {
    describe("string conversion", () => {
        it("should return true for \"true\"", () => {
            expect(toBool("true")).toBe(true);
        });

        it("should return false for \"false\"", () => {
            expect(toBool("false")).toBe(false);
        });

        it("should use Boolean() for other strings", () => {
            expect(toBool("")).toBe(false);
            expect(toBool("0")).toBe(true);
            expect(toBool("1")).toBe(true);
            expect(toBool("hello")).toBe(true);
            expect(toBool("FALSE")).toBe(true); // Case sensitive
            expect(toBool("TRUE")).toBe(true); // Case sensitive
        });
    });

    describe("non-string conversion", () => {
        it("should use Boolean() for numbers", () => {
            expect(toBool(0)).toBe(false);
            expect(toBool(1)).toBe(true);
            expect(toBool(-1)).toBe(true);
            expect(toBool(3.14)).toBe(true);
            expect(toBool(NaN)).toBe(false);
            expect(toBool(Infinity)).toBe(true);
        });

        it("should use Boolean() for objects and arrays", () => {
            expect(toBool({})).toBe(true);
            expect(toBool([])).toBe(true);
            expect(toBool({ key: "value" })).toBe(true);
            expect(toBool([1, 2, 3])).toBe(true);
        });

        it("should use Boolean() for null and undefined", () => {
            expect(toBool(null)).toBe(false);
            expect(toBool(undefined)).toBe(false);
        });

        it("should use Boolean() for functions", () => {
            expect(toBool(() => {})).toBe(true);
            expect(toBool(function() {})).toBe(true);
            expect(toBool(Boolean)).toBe(true);
        });

        it("should use Boolean() for symbols", () => {
            expect(toBool(Symbol("test"))).toBe(true);
            expect(toBool(Symbol())).toBe(true);
        });
    });

    describe("edge cases", () => {
        it("should handle boolean inputs", () => {
            expect(toBool(true)).toBe(true);
            expect(toBool(false)).toBe(false);
        });

        it("should be case sensitive for string literals", () => {
            expect(toBool("True")).toBe(true);
            expect(toBool("TRUE")).toBe(true);
            expect(toBool("False")).toBe(true);
            expect(toBool("FALSE")).toBe(true);
            expect(toBool("tRuE")).toBe(true);
            expect(toBool("fAlSe")).toBe(true);
        });

        it("should handle strings with whitespace", () => {
            expect(toBool(" true")).toBe(true);
            expect(toBool("true ")).toBe(true);
            expect(toBool(" true ")).toBe(true);
            expect(toBool(" false")).toBe(true);
            expect(toBool("false ")).toBe(true);
        });

        it("should handle special values", () => {
            expect(toBool(BigInt(0))).toBe(false);
            expect(toBool(BigInt(1))).toBe(true);
            expect(toBool(new Date())).toBe(true);
            expect(toBool(/regex/)).toBe(true);
        });
    });
});

