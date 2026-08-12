import { describe, expect, it } from "vitest";
import { formatBytes, parseDelimited } from "../format";
describe("formatBytes", () => {
    it("renders bytes/KB/MB", () => {
        expect(formatBytes(0)).toBe("0 B");
        expect(formatBytes(512)).toBe("512 B");
        expect(formatBytes(1536)).toBe("1.5 KB");
        expect(formatBytes(5 * 1024 * 1024)).toBe("5.0 MB");
    });
    it("guards invalid input", () => {
        expect(formatBytes(-1)).toBe("");
        expect(formatBytes(NaN)).toBe("");
    });
});
describe("parseDelimited", () => {
    it("parses simple CSV", () => {
        expect(parseDelimited("a,b,c\n1,2,3", ",")).toEqual([
            ["a", "b", "c"],
            ["1", "2", "3"],
        ]);
    });
    it("handles quoted cells with embedded commas and quotes", () => {
        const rows = parseDelimited('name,note\n"Doe, Jane","she said ""hi"""', ",");
        expect(rows).toEqual([
            ["name", "note"],
            ["Doe, Jane", 'she said "hi"'],
        ]);
    });
    it("handles newlines inside quotes", () => {
        const rows = parseDelimited('a,"line1\nline2"\n', ",");
        expect(rows).toEqual([["a", "line1\nline2"]]);
    });
    it("parses TSV", () => {
        expect(parseDelimited("a\tb\n1\t2", "\t")).toEqual([
            ["a", "b"],
            ["1", "2"],
        ]);
    });
    it("ignores carriage returns", () => {
        expect(parseDelimited("a,b\r\n1,2\r\n", ",")).toEqual([
            ["a", "b"],
            ["1", "2"],
        ]);
    });
});
