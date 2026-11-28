import { describe, expect, it } from "vitest";
import type { Terminal } from "@xterm/xterm";
import { commandBlockAtCell, MAX_COMMAND_BLOCK_ROWS } from "./terminalCommandBlocks";

interface FakeRow {
  text: string;
  colored?: Array<[number, number]>;
  wrapped?: boolean;
}

function fakeTerminal(rows: FakeRow[], cols = 80): Terminal {
  const lines = rows.map(({ text, colored = [], wrapped = false }) => ({
    isWrapped: wrapped,
    translateToString: (trimRight?: boolean) => (trimRight ? text.replace(/\s+$/, "") : text),
    getCell: (col: number) => {
      const ch = text.charAt(col) || " ";
      const styled = colored.some(([start, end]) => col >= start && col < end);
      return {
        getWidth: () => 1,
        getChars: () => ch,
        isFgDefault: () => !styled,
        isBold: () => 0,
        isItalic: () => 0,
        isUnderline: () => 0,
      };
    },
  }));
  return {
    cols,
    buffer: { active: { getLine: (index: number) => lines[index] } },
  } as unknown as Terminal;
}

function coloredRows(texts: string[]): FakeRow[] {
  return texts.map((text) => ({ text, colored: [[0, text.length]] }));
}

describe("commandBlockAtCell", () => {
  const WRAPPED_COMMAND = [
    "Would you run this 10-second, root-only",
    "allocation sample? It may add brief",
    "tracing overhead: sudo timeout 15s",
    "bpftrace -e 'tracepoint:kmem:kmalloc /",
    "args->bytes_alloc == 64/",
    "{ @[ksym(args->call_site), comm] =",
    "count(); } interval:s:10 { exit(); }'",
    "> /tmp/vrooli-kmalloc-profile.txt 2>&1",
  ];
  const COMMAND = "sudo timeout 15s bpftrace -e 'tracepoint:kmem:kmalloc / args->bytes_alloc == 64/ { @[ksym(args->call_site), comm] = count(); } interval:s:10 { exit(); }' > /tmp/vrooli-kmalloc-profile.txt 2>&1";

  it("reconstructs a word-wrapped colored command and drops the prose before it", () => {
    const term = fakeTerminal(coloredRows(WRAPPED_COMMAND));
    const block = commandBlockAtCell(term, 2, 20);
    expect(block).toMatchObject({ kind: "command", value: COMMAND, lineCount: 1 });
  });

  it("finds the same command when the tap lands on a wrapped continuation row", () => {
    const term = fakeTerminal(coloredRows(WRAPPED_COMMAND));
    expect(commandBlockAtCell(term, 5, 6)?.value).toBe(COMMAND);
  });

  it("does not merge separate inline code spans from adjacent bullet rows", () => {
    const rows = [
      "- Your exact case: tap green pnpm run build",
      "  inside - pnpm run build passed.",
      "- Inline code inside prose, e.g. Run pnpm",
      "  run build before continuing.",
    ].map((text) => {
      const start = text.indexOf("pnpm");
      return { text, colored: [[start, start + "pnpm run build".length]] as Array<[number, number]> };
    });
    const term = fakeTerminal(rows);
    expect(commandBlockAtCell(term, 1, (rows[1]?.text.indexOf("pnpm") ?? 0) + 2)?.value).toBe("pnpm run build");
    expect(commandBlockAtCell(term, 0, (rows[0]?.text.indexOf("pnpm") ?? 0) + 2)?.value).toBe("pnpm run build");
  });

  it("stitches an inline code span that wraps across rows", () => {
    const term = fakeTerminal([
      { text: "- Inline code example: Run pnpm", colored: [[27, 31]] },
      { text: "  run build before continuing.", colored: [[2, 11]] },
    ]);
    expect(commandBlockAtCell(term, 0, 29)?.value).toBe("pnpm run build");
    expect(commandBlockAtCell(term, 1, 4)?.value).toBe("pnpm run build");
  });

  it("extracts an inline code span from a colored bullet and leaves the prose behind", () => {
    const text = "- pnpm run build passed.";
    const term = fakeTerminal([{ text, colored: [[2, 16]] }]);
    expect(commandBlockAtCell(term, 0, 8)?.value).toBe("pnpm run build");
  });

  it("extracts a short inline command without requiring surrounding code", () => {
    const text = "Run pnpm run build before you continue.";
    const term = fakeTerminal([{ text, colored: [[4, 19]] }]);
    expect(commandBlockAtCell(term, 0, 10)?.value).toBe("pnpm run build");
  });

  it("rejects a fully colored inline prose sentence", () => {
    const text = "The build finished successfully.";
    const term = fakeTerminal([{ text, colored: [[0, text.length]] }]);
    expect(commandBlockAtCell(term, 0, 5)).toBeNull();
  });

  it("requires styled text so ordinary prose is not interactive", () => {
    const term = fakeTerminal([{ text: "you can sudo timeout 15s bpftrace next" }]);
    expect(commandBlockAtCell(term, 0, 12)).toBeNull();
  });

  it("rejects colored prose that does not look like code", () => {
    const term = fakeTerminal(coloredRows([
      "The deployment finished successfully.",
      "No action is required right now.",
    ]));
    expect(commandBlockAtCell(term, 0, 5)).toBeNull();
  });

  it("keeps separate command lines apart when each row starts a command", () => {
    const term = fakeTerminal(coloredRows([
      "export FOO=1",
      'echo "$FOO"',
      'printf "%s\\n" "$FOO"',
    ]));
    expect(commandBlockAtCell(term, 0, 3)?.value).toBe('export FOO=1\necho "$FOO"\nprintf "%s\\n" "$FOO"');
  });

  it("rejects blocks over the row limit", () => {
    const rows = Array.from({ length: MAX_COMMAND_BLOCK_ROWS + 1 }, (_, index) => `echo line-${String(index)}`);
    expect(commandBlockAtCell(fakeTerminal(coloredRows(rows)), 0, 2)).toBeNull();
  });

  it("accepts a tap on whitespace inside a styled command line", () => {
    const text = "sudo   echo hi";
    const term = fakeTerminal([{ text, colored: [[0, text.length]] }]);
    expect(commandBlockAtCell(term, 0, 6)?.value).toBe(text);
  });

  it("ignores taps outside the styled span", () => {
    const text = "plain prose sudo echo hi";
    const term = fakeTerminal([{ text, colored: [[12, text.length]] }]);
    expect(commandBlockAtCell(term, 0, 2)).toBeNull();
  });

  it("reconstructs a hard-wrapped token without inserting a space", () => {
    const term = fakeTerminal([
      { text: "curl https://example.com/very/long/pa", colored: [[0, 40]], wrapped: true },
      { text: "th/to/file.txt", colored: [[0, 15]] },
    ], 40);
    expect(commandBlockAtCell(term, 0, 10)?.value).toBe("curl https://example.com/very/long/path/to/file.txt");
  });
});
