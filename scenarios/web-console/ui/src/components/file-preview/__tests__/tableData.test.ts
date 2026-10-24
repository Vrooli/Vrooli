import { describe, expect, it } from "vitest";
import { parseDelimitedDetailed } from "../format";
import { columnWidth, filterAndSortRows, MAX_TABLE_ROWS, nextTableSort, prepareTable } from "../tableData";

describe("table preview data integrity", () => {
  it("handles BOM, CR-only records, empty quoted values and embedded quoted newlines", () => {
    expect(parseDelimitedDetailed('\uFEFFname,note\ralpha,""\rbeta,"one\r\ntwo"', ",")).toEqual({ rows: [["name", "note"], ["alpha", ""], ["beta", "one\r\ntwo"]], malformed: false });
    expect(parseDelimitedDetailed('""', ",").rows).toEqual([[""]]);
  });
  it("detects incomplete/invalid quoting instead of claiming a faithful table", () => {
    expect(parseDelimitedDetailed('name,note\na,"unfinished', ",").malformed).toBe(true);
    expect(parseDelimitedDetailed('name\n"closed"unexpected', ",").malformed).toBe(true);
    expect(parseDelimitedDetailed('name,note\na,"a ""quoted"" value"', ",").malformed).toBe(false);
  });
  it("retains ragged extra fields and makes the loaded-row bound explicit", () => {
    const table = prepareTable([["a"], ["1", "extra", "keep me"]], true);
    expect(table.columns).toBe(3);
    expect(table.rows[0]?.cells).toEqual(["1", "extra", "keep me"]);
    const large = prepareTable([["a"], ...Array.from({ length: 1001 }, (_, i) => [String(i)])], true);
    expect(large.rows).toHaveLength(MAX_TABLE_ROWS);
    expect(large.totalRows).toBe(1001);
    expect(large.truncated).toBe(true);
  });
  it("sorts numbers, huge identifiers and ties without changing source order or values", () => {
    const data = prepareTable([["value"], ["10.5"], ["2.25"], ["2.25"], [""], ["100"]], true);
    const sorted = filterAndSortRows(data.rows, "", { column: 0, direction: "asc" }, "en");
    expect(sorted.map((row) => row.index)).toEqual([1, 2, 0, 4, 3]);
    expect(data.rows.map((row) => row.cells[0])).toEqual(["10.5", "2.25", "2.25", "", "100"]);
    const ids = prepareTable([["id"], ["9007199254740993"], ["9007199254740992"]], true);
    expect(filterAndSortRows(ids.rows, "", { column: 0, direction: "asc" }, "en")[0]?.cells[0]).toBe("9007199254740992");
  });
  it("filters every loaded field and cycles sorting back to original order", () => {
    const rows = prepareTable([["name", "status"], ["A", "settled"], ["B", "pending"]], true).rows;
    expect(filterAndSortRows(rows, " PENDING ", null, "en").map((row) => row.index)).toEqual([1]);
    const asc = nextTableSort(null, 1);
    expect(asc).toEqual({ column: 1, direction: "asc" });
    expect(nextTableSort(asc, 1)).toEqual({ column: 1, direction: "desc" });
    expect(nextTableSort({ column: 1, direction: "desc" }, 1)).toBeNull();
    expect(columnWidth(-100)).toBe(100);
    expect(columnWidth(1000)).toBe(640);
  });
});
