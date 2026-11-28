import { describe, expect, it } from "vitest";
import { Terminal } from "@xterm/xterm";
import { linkAtCell, matchTerminalFilePaths, matchTerminalLinks, rewriteLocalhostUrlForProxy } from "./terminalFileLinks";

describe("matchTerminalLinks", () => {
  it("recognizes relative paths with optional line and column", () => {
    expect(matchTerminalFilePaths("ERROR src/app.ts:42:7 failed")).toEqual([
      { kind: "file", value: "src/app.ts:42:7", text: "src/app.ts:42:7", startOffset: 6 },
    ]);
  });

  it("accepts source extensions without maintaining an extension allowlist", () => {
    expect(matchTerminalFilePaths("edit components/Button.tsx now")[0]?.text).toBe("components/Button.tsx");
  });

  it("keeps the home anchor on a ~/ path", () => {
    expect(matchTerminalFilePaths("run ~/.vrooli/portfolio/run.sh now")[0]?.value).toBe("~/.vrooli/portfolio/run.sh");
  });

  it("keeps path characters that are not shell separators", () => {
    expect(matchTerminalFilePaths("see src/a+b.ts and tools/@scope/bin.ts")[0]?.value).toBe("src/a+b.ts");
    expect(matchTerminalFilePaths("see src/a+b.ts and tools/@scope/bin.ts")[1]?.value).toBe("tools/@scope/bin.ts");
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

describe("rewriteLocalhostUrlForProxy", () => {
  const origin = "https://web-console.example.test";

  it.each([
    "http://localhost:22705/jobs?id=1#latest",
    "http://127.0.0.1:22705/jobs?id=1#latest",
    "http://127.12.0.9:22705/jobs",
    "http://0.0.0.0:22705/jobs",
  ])("routes %s through the Web Console origin", (url) => {
    expect(rewriteLocalhostUrlForProxy(url, origin)).toBe(
      "https://web-console.example.test/_local/22705/jobs" + (url.includes("?") ? "?id=1#latest" : ""),
    );
  });

  it("routes IPv6 loopback through the fixed IPv6 loopback target", () => {
    expect(rewriteLocalhostUrlForProxy("http://[::1]:22705/jobs", origin)).toBe(
      "https://web-console.example.test/_local/ipv6/22705/jobs",
    );
  });

  it("defaults an omitted HTTP port to port 80", () => {
    expect(rewriteLocalhostUrlForProxy("http://localhost/status", origin)).toBe(
      "https://web-console.example.test/_local/80/status",
    );
  });

  it.each([
    "https://localhost:22705/",
    "http://example.test:22705/",
    "http://localhost:0/",
    "http://user:secret@localhost:22705/",
  ])("leaves unsupported or unsafe target %s unchanged", (url) => {
    expect(rewriteLocalhostUrlForProxy(url, origin)).toBeNull();
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

describe("linkAtCell across a TUI bullet hanging wrap", () => {
  // A message renderer wraps its content well short of term.cols and hangs the
  // continuation under the bullet text. The first row ends on a path separator
  // fragment ("-"), so there is no soft-wrap flag and no second row at the same
  // width to corroborate the wrap column.
  const COLS = 80;
  const ROWS = [
    "- scenarios/research-desk/docs/internal/proposals/UX-REDESIGN-",
    "  20261126/index.html — visual study, phone-readable",
  ];
  const PATH = "scenarios/research-desk/docs/internal/proposals/UX-REDESIGN-20261126/index.html";

  it("stitches a path split after a separator fragment", () => {
    expect(linkAtCell(fakeTerminal(ROWS, COLS), 0, 20)).toEqual({ kind: "file", value: PATH, text: PATH });
  });

  it("stitches the same path when tapping its continuation", () => {
    expect(linkAtCell(fakeTerminal(ROWS, COLS), 1, 6)).toEqual({ kind: "file", value: PATH, text: PATH });
  });
});

describe("linkAtCell inside a TUI quote gutter", () => {
  // OpenCode draws quoted user messages with a `┃` rule and spaces on every
  // row. The rule is chrome: the path continues after it, and a tap on a
  // continuation row must skip the gutter rather than treat it as content.
  const ROWS = [
    "  ┃  backdrop to close, and stop propagation /home/matthalloran8/.vrooli/",
    "  ┃  cache/vrooli/web-console/uploads/8f70374f-c348-4dab-93da-",
    "  ┃  3f2cb61d1ce9/IMG_2730.jpeg",
  ];
  const PATH = "/home/matthalloran8/.vrooli/cache/vrooli/web-console/uploads/8f70374f-c348-4dab-93da-3f2cb61d1ce9/IMG_2730.jpeg";

  it.each([
    [0, 50],
    [1, 10],
    [2, 8],
  ])("skips the rule and stitches the full path from row %i", (row, col) => {
    expect(linkAtCell(fakeTerminal(ROWS, 80), row, col)).toEqual({ kind: "file", value: PATH, text: PATH });
  });

  it("stitches a bullet continuation behind the same gutter", () => {
    const rows = [
      "  ┃  - scenarios/research-desk/docs/internal/proposals/UX-REDESIGN-",
      "  ┃    20261126-v2/index.html",
    ];
    const path = "scenarios/research-desk/docs/internal/proposals/UX-REDESIGN-20261126-v2/index.html";
    expect(linkAtCell(fakeTerminal(rows, 80), 1, 10)).toEqual({ kind: "file", value: path, text: path });
  });

  it("stitches an assistant bullet indented under the message margin", () => {
    const rows = [
      "     - scenarios/research-desk/docs/internal/proposals/UX-REDESIGN-",
      "       20261126-v2/index.html",
    ];
    const path = "scenarios/research-desk/docs/internal/proposals/UX-REDESIGN-20261126-v2/index.html";
    expect(linkAtCell(fakeTerminal(rows, 77), 0, 12)?.value).toBe(path);
    expect(linkAtCell(fakeTerminal(rows, 77), 1, 12)?.value).toBe(path);
  });

  it("keeps two quoted paths on one row separate while stitching each wrap", () => {
    const rows = [
      "  ┃  Wrong again /home/matthalloran8/.vrooli/cache/vrooli/web-console/",
      "  ┃  uploads/8f70374f-c348-4dab-93da-3f2cb61d1ce9/IMG_2739.jpeg /home/",
      "  ┃  matthalloran8/.vrooli/cache/vrooli/web-console/uploads/8f70374f-",
      "  ┃  c348-4dab-93da-3f2cb61d1ce9/IMG_2741.jpeg",
    ];
    const first = "/home/matthalloran8/.vrooli/cache/vrooli/web-console/uploads/8f70374f-c348-4dab-93da-3f2cb61d1ce9/IMG_2739.jpeg";
    const second = "/home/matthalloran8/.vrooli/cache/vrooli/web-console/uploads/8f70374f-c348-4dab-93da-3f2cb61d1ce9/IMG_2741.jpeg";
    expect(linkAtCell(fakeTerminal(rows, 77), 1, 10)?.value).toBe(first);
    expect(linkAtCell(fakeTerminal(rows, 77), 1, 68)?.value).toBe(second);
    expect(linkAtCell(fakeTerminal(rows, 77), 3, 10)?.value).toBe(second);
  });
});

describe("linkAtCell across unindented prose wrapping", () => {
  // A long path in running prose wraps at the message width. The continuation
  // rows are flush left (no hanging indent), and the renderer breaks at path
  // separators, so line lengths differ from row to row.
  const ROWS = [
    "backdrop to close, and stop propagation /home/matthalloran8/.vrooli/",
    "cache/vrooli/web-console/uploads/8f70374f-c348-4dab-93da-",
    "3f2cb61d1ce9/IMG_2730.jpeg",
  ];
  const PATH = "/home/matthalloran8/.vrooli/cache/vrooli/web-console/uploads/8f70374f-c348-4dab-93da-3f2cb61d1ce9/IMG_2730.jpeg";

  it.each([
    [0, 45],
    [1, 10],
    [2, 8],
  ])("stitches the full path when tapping row %i", (row, col) => {
    expect(linkAtCell(fakeTerminal(ROWS, 80), row, col)).toEqual({ kind: "file", value: PATH, text: PATH });
  });
});

describe("linkAtCell across a short content width with an unremarkable seam", () => {
  // Same hanging indent, but the break lands mid-word. The first row is the
  // widest row in its block, which is the only remaining evidence of the wrap.
  const ROWS = [
    "see /home/user/project/src/components/ButtonComponent",
    "  .tsx before shipping",
  ];
  const PATH = "/home/user/project/src/components/ButtonComponent.tsx";

  it("uses the block's widest row as the wrap column", () => {
    expect(linkAtCell(fakeTerminal(ROWS, 80), 0, 30)).toEqual({ kind: "file", value: PATH, text: PATH });
  });
});

describe("linkAtCell does not over-join", () => {
  it.each([30, 59, 60, 62, 75, 76, 78, 80])("selects only a printed report path between shell prompts at %i columns", async (cols) => {
    const term = new Terminal({ cols, rows: 12, allowProposedApi: true });
    const prompt = "matthalloran8@swarminator:~/Vrooli$ ";
    const command = "~/.vrooli/portfolio/steward/run.sh show";
    const path = "/home/matthalloran8/.vrooli/portfolio/pages/2026-11-17.html";
    try {
      await new Promise<void>((resolve) => { term.write(`${prompt}${command}\r\n${path}\r\n${prompt}`, resolve); });
      const firstOutputRow = Math.ceil((prompt.length + command.length) / cols);
      for (let offset = 0; offset < path.length; offset++) {
        expect(linkAtCell(term, firstOutputRow + Math.floor(offset / cols), offset % cols)?.value).toBe(path);
      }
    } finally { term.dispose(); }
  });

  it("keeps two separate short lines apart (first is longest but did not wrap)", () => {
    // Neither row reaches cols - WRAP_MARGIN, so they must not be bridged.
    const term = fakeTerminal(["edit a/b.ts", "and c/d.ts"], 80);
    expect(linkAtCell(term, 0, 7)).toEqual({ kind: "file", value: "a/b.ts", text: "a/b.ts" });
  });

  it("does not fuse a short path into the next indented prose line", () => {
    const term = fakeTerminal(["edit a/b.ts", "  and c/d.ts"], 80);
    expect(linkAtCell(term, 0, 7)).toEqual({ kind: "file", value: "a/b.ts", text: "a/b.ts" });
  });

  it("does not fuse an incomplete line into the next list item", () => {
    const term = fakeTerminal(["- see docs/foo-", "  - bar/baz.ts"], 80);
    expect(linkAtCell(term, 0, 12)).toEqual({ kind: "file", value: "docs/foo-", text: "docs/foo-" });
  });

  it("does not fuse a short trailing-slash command into the output below it", () => {
    const term = fakeTerminal(["cd /tmp/", "output text here"], 80);
    expect(linkAtCell(term, 0, 4)).toEqual({ kind: "file", value: "/tmp/", text: "/tmp/" });
  });

  it("does not fuse a command that ended on a separator into the next shell prompt", () => {
    const term = fakeTerminal(["cd /tmp/", "matthalloran8@swarminator:~/Vrooli$ ls"], 80);
    expect(linkAtCell(term, 0, 4)).toEqual({ kind: "file", value: "/tmp/", text: "/tmp/" });
  });

  it("returns null when the tap is on whitespace away from any token", () => {
    expect(linkAtCell(fakeTerminal(["   just prose here   "], 80), 0, 1)).toBeNull();
  });
});
