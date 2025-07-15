// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-19
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { sanitizeFilename } from "./ShareObjectDialog.js";

describe("sanitizeFilename", () => {
    it("should replace invalid characters with underscores", () => {
        const filename = "test<file>:name?.txt";
        const expected = "test_file__name_.txt";
        expect(sanitizeFilename(filename)).toEqual(expected);
    });

    it("should remove control characters", () => {
        const filename = "test\u0001\u001Fname.txt";
        const expected = "testname.txt";
        expect(sanitizeFilename(filename)).toEqual(expected);
    });

    it("should prepend an underscore to reserved words", () => {
        const filenames = ["con.txt", "prn.doc", "aux.jpg", "nul", "COM1", "lpt2"];
        const expected = ["_con.txt", "_prn.doc", "_aux.jpg", "_nul", "_COM1", "_lpt2"];
        filenames.forEach((file, index) => {
            expect(sanitizeFilename(file)).toEqual(expected[index]);
        });
    });

    it("should handle filenames that are only reserved words", () => {
        const filename = "con";
        const expected = "_con";
        expect(sanitizeFilename(filename)).toEqual(expected);
    });

    it("should handle empty filenames", () => {
        const filename = "";
        const expected = "";
        expect(sanitizeFilename(filename)).toEqual(expected);
    });

    it("should handle filenames with only invalid and control characters", () => {
        const filename = "<>:\"/\\|?*\u0001\u001F";
        // Expect all underscores
        const result = sanitizeFilename(filename);
        const allUnderscores = result.split("").every((char) => char === "_");
        expect(allUnderscores).toBe(true);
    });

    it("should handle complex filenames with mixed issues", () => {
        const filename = "con<invalid:\u0001name?.txt";
        const expected = "con_invalid_name_.txt";
        expect(sanitizeFilename(filename)).toEqual(expected);
    });

    it("should not modify valid filenames", () => {
        const filename = "valid_filename.txt";
        const expected = "valid_filename.txt";
        expect(sanitizeFilename(filename)).toEqual(expected);
    });

    it("should process filenames with dots that are not extensions for reserved names", () => {
        const filename = "lpt1.random.txt";
        const expected = "_lpt1.random.txt";
        expect(sanitizeFilename(filename)).toEqual(expected);
    });
});
