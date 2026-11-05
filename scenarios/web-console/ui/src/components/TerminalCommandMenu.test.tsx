import { act, fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../test-utils";
import type { TerminalCommandActivation } from "../lib/terminalCommandBlocks";
import { copyText } from "../lib/clipboard";
import TerminalCommandMenu from "./TerminalCommandMenu";

vi.mock("../lib/clipboard", () => ({ copyText: vi.fn().mockResolvedValue(true) }));

const position = { x: 40, y: 80 };
const activation = (block: Omit<TerminalCommandActivation, "position">): TerminalCommandActivation => ({ ...block, position });

describe("TerminalCommandMenu", () => {
  it("copies the reconstructed command rather than the visual lines", async () => {
    const command = "sudo timeout 15s bpftrace -e 'tracepoint:kmem:kmalloc / args->bytes_alloc == 64/'";
    render(
      <TerminalCommandMenu
        activation={activation({ kind: "command", value: command, text: command, startRow: 0, endRow: 2, lineCount: 3 })}
        onRun={vi.fn()}
        onClose={vi.fn()}
      />,
    );
    expect(screen.getByTestId("command-copy")).toHaveTextContent("Copy 3 lines");
    await act(async () => { fireEvent.click(screen.getByTestId("command-copy")); await Promise.resolve(); });
    expect(copyText).toHaveBeenCalledWith(command);
  });

  it("hands the command to the run flow without executing it directly", () => {
    const onRun = vi.fn();
    const command = "curl https://example.com";
    render(
      <TerminalCommandMenu
        activation={activation({ kind: "command", value: command, text: command, startRow: 0, endRow: 0, lineCount: 1 })}
        onRun={onRun}
        onClose={vi.fn()}
      />,
    );
    expect(screen.getByTestId("command-copy")).toHaveTextContent("Copy command");
    fireEvent.click(screen.getByTestId("command-run"));
    expect(onRun).toHaveBeenCalledWith(command);
  });
});
