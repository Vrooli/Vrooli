import { describe, expect, it } from "vitest";
import { parseStructure } from "../parseStructure";
import { formatJsonSource, MAX_STRUCTURE_CHARACTERS, visibleStructure } from "../structuredData";

function parsed(source: string, json = true) {
  const result = parseStructure(source, json);
  if (result.error || !result.documents[0]) throw new Error(`Expected a document: ${String(result.error)}`);
  return result.documents[0];
}
describe("faithful structured data inspection", () => {
  it("preserves large numeric lexemes, false, null, empty strings and special property paths", () => {
    const document = parsed('{"id":900719925474099312345,"decimal":1.234567890123456789,"false":false,"empty":"","nil":null,"a.b":{"x y":["ok"]},"__proto__":"safe"}');
    expect(document.nodes.find((node) => node.key === "id")?.value).toBe("900719925474099312345");
    expect(document.nodes.find((node) => node.key === "decimal")?.value).toBe("1.234567890123456789");
    expect(document.nodes.find((node) => node.key === "false")).toMatchObject({ kind: "boolean", value: "false" });
    expect(document.nodes.find((node) => node.key === "empty")).toMatchObject({ kind: "string", value: "" });
    expect(document.nodes.find((node) => node.key === "nil")).toMatchObject({ kind: "null", value: "null" });
    expect(document.nodes.find((node) => node.value === "ok")?.path).toBe('$["a.b"]["x y"][0]');
    expect(document.nodes.find((node) => node.key === "__proto__")?.value).toBe("safe");
  });
  it("accepts root scalars, empty containers and strings with escaped JSON syntax", () => {
    expect(parsed('"text"').nodes[0]).toMatchObject({ kind: "string", value: "text" });
    expect(parsed("[]").nodes[0]).toMatchObject({ kind: "array", children: [] });
    expect(parsed("{}").nodes[0]).toMatchObject({ kind: "object", children: [] });
    const value = 'quoted " { } \\ newline\n';
    expect(parsed(JSON.stringify({ value })).nodes[1]?.value).toBe(value);
  });
  it("rejects malformed JSON, duplicate keys, complex YAML keys and custom tags", () => {
    expect(parseStructure('{"x":1,}', true).error).toBe("invalid");
    expect(parseStructure('{"x":1,"x":2}', true).error).toBe("invalid");
    expect(parseStructure('!execute payload', false).error).toBe("unsupported");
    expect(parseStructure('? [a, b]\n: value', false).error).toBe("unsupported");
    expect(parseStructure('%YAML 1.1\n---\ndate: 2026-10-17', false).error).toBe("unsupported");
  });
  it("retains YAML multi-document structure, block strings, aliases and original source ranges", () => {
    const source = 'title: "One"\nbase: &base {port: 42}\ncopy: *base\ntext: |\n  line one\n  line two\n---\n- second\n';
    const result = parseStructure(source, false);
    expect(result.documents).toHaveLength(2);
    const nodes = result.documents?.[0]?.nodes ?? [];
    const reference = nodes.find((node) => node.key === "copy");
    expect(reference).toMatchObject({ kind: "reference", value: "*base", children: [] });
    expect(nodes.find((node) => node.key === "text")?.value).toBe("line one\nline two\n");
    const title = nodes.find((node) => node.key === "title");
    expect(source.slice(title?.start, title?.end)).toBe('"One"');
  });
  it("does not expand recursive YAML aliases or merge keys", () => {
    const document = parsed('base: &loop [*loop]\ncopy: *loop', false);
    expect(document.nodes).toHaveLength(4);
    expect(document.nodes.filter((node) => node.kind === "reference")).toHaveLength(2);
  });
  it("falls back explicitly when text, node count or nesting exceed preview limits", () => {
    expect(parseStructure(" ".repeat(MAX_STRUCTURE_CHARACTERS + 1), true).error).toBe("limit");
    expect(parseStructure(JSON.stringify(Array.from({ length: 5001 }, () => 1)), true).error).toBe("limit");
    expect(parseStructure("[".repeat(62) + "0" + "]".repeat(62), true).error).toBe("limit");
  });
  it("finds collapsed descendants by key, value and property path and retains their ancestors", () => {
    const document = parsed('{"group":{"hidden":{"name":"needle"}},"other":false}');
    expect(visibleStructure(document, new Set([0]), "").rows.map((row) => row.node.key)).toEqual(["$", "group", "other"]);
    expect(visibleStructure(document, new Set(), "needle").rows.map((row) => row.node.key)).toEqual(["$", "group", "hidden", "name"]);
    expect(visibleStructure(document, new Set(), "$.group.hidden.name").matches.size).toBe(1);
    expect(visibleStructure(document, new Set(), "no match").rows).toEqual([]);
  });
  it("formats JSON without rounding numbers or changing string values and restores valid JSON", () => {
    const source = `{ "id":900719925474099312345,"amount":1.234567890123456789,"text":${JSON.stringify('x" } : , \\ newline\n')}, "empty":[{},[]]}`;
    const formatted = formatJsonSource(source);
    expect(formatted).toContain('"id": 900719925474099312345');
    expect(formatted).toContain('"amount": 1.234567890123456789');
    expect(JSON.parse(formatted)).toEqual(JSON.parse(source));
    expect(formatted).toContain("\n  ");
    expect(() => formatJsonSource("invalid")).toThrow();
  });
});
