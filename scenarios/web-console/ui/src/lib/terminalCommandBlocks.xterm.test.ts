import { describe, expect, it } from "vitest";
import { Terminal } from "@xterm/xterm";
import { commandBlockAtCell } from "./terminalCommandBlocks";

async function write(term: Terminal, data: string): Promise<void> {
  await new Promise<void>((resolve) => { term.write(data, resolve); });
}

describe("commandBlockAtCell against a real xterm buffer", () => {
  it("extracts only the colored inline code from a row that also holds prose", async () => {
    const term = new Terminal({ cols: 80, rows: 10, allowProposedApi: true });
    await write(term, "- \x1b[32mpnpm run build\x1b[0m passed.\r\n");
    expect(commandBlockAtCell(term, 0, 8)?.value).toBe("pnpm run build");
    term.dispose();
  });

  it("reconstructs an auto-wrapped command without losing the seam space", async () => {
    const term = new Terminal({ cols: 40, rows: 10, allowProposedApi: true });
    const command = "sudo timeout 15s bpftrace -e 'tracepoint:kmem:kmalloc / args->bytes_alloc == 64/ { @[ksym(args->call_site), comm] = count(); } interval:s:10 { exit(); }' > /tmp/profile.txt 2>&1";
    await write(term, `\x1b[34m${command}\x1b[0m\r\n`);
    expect(commandBlockAtCell(term, 0, 10)?.value).toBe(command);
    term.dispose();
  });
});
