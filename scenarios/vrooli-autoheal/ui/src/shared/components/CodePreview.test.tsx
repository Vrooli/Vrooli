import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CodePreview } from "./CodePreview";
import { renderWithProviders } from "../../test-utils";

const highlightMocks = vi.hoisted(() => ({ highlightCode: vi.fn() }));
vi.mock("../../lib/highlighter", () => highlightMocks);

describe("CodePreview", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    highlightMocks.highlightCode.mockResolvedValue("<pre><code>highlighted</code></pre>");
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    vi.stubGlobal("__autohealClipboardWrite", writeText);
  });

  it("highlights strings, serializes objects, and copies code", async () => {
    renderWithProviders(<CodePreview code={{ enabled: true }} language="json" maxHeight="10rem" />);
    await waitFor(() => expect(screen.getByText("highlighted")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: /copy code/i }));
    expect((globalThis as typeof globalThis & { __autohealClipboardWrite: ReturnType<typeof vi.fn> }).__autohealClipboardWrite)
      .toHaveBeenCalledWith('{\n  "enabled": true\n}');
    expect(await screen.findByRole("button", { name: /copied/i })).toBeInTheDocument();
  });

  it("falls back to plain text when highlighting fails or code is absent", async () => {
    highlightMocks.highlightCode.mockRejectedValueOnce(new Error("unsupported"));
    renderWithProviders(<CodePreview code={null} language="" />);
    expect(await screen.findByText("text")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /copy code/i })).toBeInTheDocument();
  });

  it("uses the textarea copy fallback when the clipboard rejects", async () => {
    const writeText = vi.fn().mockRejectedValue(new Error("clipboard denied"));
    Object.assign(navigator, { clipboard: { writeText } });
    const execCommand = vi.fn(() => true);
    document.execCommand = execCommand;
    renderWithProviders(<CodePreview code="fallback text" />);
    fireEvent.click(screen.getByRole("button", { name: /copy code/i }));
    await waitFor(() => expect(execCommand).toHaveBeenCalledWith("copy"));
  });

  it("uses the fallback when clipboard APIs are unavailable", async () => {
    Object.assign(navigator, { clipboard: undefined });
    const execCommand = vi.fn(() => true);
    document.execCommand = execCommand;
    renderWithProviders(<CodePreview code="no clipboard" />);
    fireEvent.click(screen.getByRole("button", { name: /copy code/i }));
    await waitFor(() => expect(execCommand).toHaveBeenCalledWith("copy"));
  });
});
