import { describe, expect, it } from "vitest";
import type { Terminal } from "@xterm/xterm";
import { linkAtCell, matchTerminalFilePaths, matchTerminalLinks } from "./terminalFileLinks";

describe("matchTerminalLinks", () => {
  it("recognizes relative paths with optional line and column", () => {
    expect(matchTerminalFilePaths("ERROR src/app.ts:42:7 failed")).toEqual([
      { kind: "file", value: "src/app.ts:42:7", text: "src/app.ts:42:7", startOffset: 6 },
    ]);
  });

  it("accepts source extensions without maintaining an extension allowlist", () => {
    expect(matchTerminalFilePaths("edit components/Button.tsx now")[0]?.text).toBe("components/Button.tsx");
  });

  it("does not turn ordinary prose into links", () => {
    expect(matchTerminalLinks("open the application and try again")).toEqual([]);
  });

  it("detects URLs and classifies file:// as a file link", () => {
    expect(matchTerminalLinks("see https://xtermjs.org and file:///tmp/a.log now")).toEqual([
      { kind: "url", value: "https://xtermjs.org", text: "https://xtermjs.org", startOffset: 4 },
      { kind: "file", value: "file:///tmp/a.log", text: "file:///tmp/a.log", startOffset: 28 },
    ]);
  });

  it("lets a URL win over a path embedded inside it", () => {
    const links = matchTerminalLinks("ref https://host/a/b.ts end");
    expect(links).toHaveLength(1);
    expect(links[0]).toMatchObject({ kind: "url", value: "https://host/a/b.ts" });
  });
});

/**
 * A terminal fake that returns per-row content, mirroring how codex renders:
 * a continuation row carries a hanging indent and the wrap happens a couple of
 * columns short of `cols`. Only translateToString(true) and cols are consulted.
 */
function fakeTerminal(rows: string[], cols: number): Terminal {
  const lines = rows.map((text) => ({
    isWrapped: false,
    length: cols,
    translateToString: () => text.replace(/\s+$/, ""),
  }));
  return {
    cols,
    buffer: { active: { getLine: (index: number) => lines[index] } },
  } as unknown as Terminal;
}

describe("linkAtCell across a codex hard wrap (indented continuation, sub-edge wrap)", () => {
  // cols 42; the path fills to col 40 (within WRAP_MARGIN of the edge) and the
  // continuation is indented two columns — exactly codex's layout.
  const ROWS = [
    "see /home/user/project/src/components/Bu", // 40 cols, a mid-token hard break
    "  tton.tsx now", // hanging indent + the rest of the path, then more prose
  ];
  const PATH = "/home/user/project/src/components/Button.tsx";

  it("reconstructs the full path when tapping the first row", () => {
    expect(linkAtCell(fakeTerminal(ROWS, 42), 0, 12)).toEqual({ kind: "file", value: PATH, text: PATH });
  });

  it("reconstructs the full path when tapping the indented continuation row", () => {
    expect(linkAtCell(fakeTerminal(ROWS, 42), 1, 4)).toEqual({ kind: "file", value: PATH, text: PATH });
  });

  it("forgives a tap one column before the token start", () => {
    // col 3 is the space before "/home"; margin should still catch the path.
    expect(linkAtCell(fakeTerminal(ROWS, 42), 0, 3)?.value).toBe(PATH);
  });
});

describe("linkAtCell wrapping a URL", () => {
  const ROWS = [
    "ref https://github.com/xtermjs/xterm.js/b", // 41 cols, hard break mid-URL
    "  lob/master/WebLinksAddon.ts",
  ];

  it("stitches a URL split across the wrap", () => {
    expect(linkAtCell(fakeTerminal(ROWS, 42), 0, 20)).toEqual({
      kind: "url",
      value: "https://github.com/xtermjs/xterm.js/blob/master/WebLinksAddon.ts",
      text: "https://github.com/xtermjs/xterm.js/blob/master/WebLinksAddon.ts",
    });
  });
});

describe("linkAtCell across a styled block that wraps short of the terminal edge", () => {
  // A TUI "box" (e.g. Claude Code's queued-input panel) wraps its content at the
  // box width — many columns shy of term.cols — and indents nothing here. The
  // geometric "near term.cols" gate misses every line, so bridging must fall back
  // to the content-derived wrap column shared by the box's rows.
  const COLS = 80;
  const ROWS = [
    'then select "Approved": http://localhost:21238/api/v1/', // 54 cols; box wrap width
    "authorizations/0900e1c4-97eb-4903-88bc-b32c63550c/appr", // 54 cols; another full box line (URLs wrap at the exact column)
    "ove now It covers only the six named efforts", // short last line of the box
  ];
  const URL = "http://localhost:21238/api/v1/authorizations/0900e1c4-97eb-4903-88bc-b32c63550c/approve";

  it("stitches a URL wrapped across several box rows (tapping the first row)", () => {
    expect(linkAtCell(fakeTerminal(ROWS, COLS), 0, 30)).toEqual({ kind: "url", value: URL, text: URL });
  });

  it("stitches the same URL when tapping a continuation row", () => {
    expect(linkAtCell(fakeTerminal(ROWS, COLS), 1, 5)).toEqual({ kind: "url", value: URL, text: URL });
  });

  it("does not fuse two lone short lines that merely share a run-char seam (no corroborating wrap row)", () => {
    // Only two lines, different lengths, far from the edge: no block wrap column
    // is corroborated, so they stay separate.
    const term = fakeTerminal(["edit a/b.ts", "and c/d.ts"], COLS);
    expect(linkAtCell(term, 0, 7)).toEqual({ kind: "file", value: "a/b.ts", text: "a/b.ts" });
  });
});

describe("linkAtCell does not over-join", () => {
  it("keeps two separate short lines apart (first is longest but did not wrap)", () => {
    // Neither row reaches cols - WRAP_MARGIN, so they must not be bridged.
    const term = fakeTerminal(["edit a/b.ts", "and c/d.ts"], 80);
    expect(linkAtCell(term, 0, 7)).toEqual({ kind: "file", value: "a/b.ts", text: "a/b.ts" });
  });

  it("returns null when the tap is on whitespace away from any token", () => {
    expect(linkAtCell(fakeTerminal(["   just prose here   "], 80), 0, 1)).toBeNull();
  });
});
