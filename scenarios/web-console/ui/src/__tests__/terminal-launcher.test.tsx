import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import TerminalLauncher from "../components/TerminalLauncher";
import type { ShortcutEntry } from "../consts/shortcuts";

// [REQ:P0-006a] Terminal Launch Flow UI — component rendering & interactions
// [REQ:P0-006b] Configurable Shortcut Entries — shortcut rendering and selection
// [REQ:P1-002b] Shortcut Profile Management UI — API-driven shortcut loading

const testShortcuts: ShortcutEntry[] = [
  { label: "Claude Code", command: "claude --dangerously-skip-permissions", description: "AI coding assistant" },
  { label: "Codex", command: "codex --yolo", description: "OpenAI Codex CLI" },
];

let mockGetEffectiveShortcuts: ReturnType<typeof vi.fn>;

vi.mock("../lib/api", () => ({
  getEffectiveShortcuts: vi.fn(),
}));

describe("TerminalLauncher", () => {
  const onClose = vi.fn();
  const onLaunch = vi.fn();

  beforeEach(async () => {
    vi.clearAllMocks();
    const api = await import("../lib/api");
    mockGetEffectiveShortcuts = api.getEffectiveShortcuts as ReturnType<typeof vi.fn>;
  });

  it("renders nothing when closed", () => {
    const { container } = render(
      <TerminalLauncher open={false} onClose={onClose} onLaunch={onLaunch} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders modal with 'New Terminal' title when open", () => {
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />,
    );
    expect(screen.getByText("New Terminal")).toBeTruthy();
    expect(screen.getByTestId("terminal-launcher")).toBeTruthy();
  });

  it("renders empty shell option", () => {
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />,
    );
    expect(screen.getByTestId("launcher-empty-shell")).toBeTruthy();
    expect(screen.getByText("Empty Shell")).toBeTruthy();
  });

  it("calls onLaunch with undefined backend when default is unchanged so the server applies its configured default", () => {
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />,
    );
    fireEvent.click(screen.getByTestId("launcher-empty-shell"));
    expect(onLaunch).toHaveBeenCalledWith(expect.objectContaining({ backend: undefined }));
  });

  it("renders shortcut entries from props", () => {
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />,
    );
    expect(screen.getByText("Claude Code")).toBeTruthy();
    expect(screen.getByText("Codex")).toBeTruthy();
    expect(screen.getByText("AI coding assistant")).toBeTruthy();
  });

  it("calls onLaunch with shortcut command when shortcut is clicked", () => {
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />,
    );
    fireEvent.click(screen.getByTestId("launcher-shortcut-claude-code"));
    expect(onLaunch).toHaveBeenCalledWith(
      expect.objectContaining({ command: "claude --dangerously-skip-permissions" }),
    );
  });

  it("custom command input launches on Enter key", () => {
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />,
    );
    const input = screen.getByTestId("launcher-custom-input");
    fireEvent.change(input, { target: { value: "htop" } });
    fireEvent.keyDown(input, { key: "Enter" });
    expect(onLaunch).toHaveBeenCalledWith(expect.objectContaining({ command: "htop" }));
  });

  it("custom command input clears after launch", () => {
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />,
    );
    const input = screen.getByTestId("launcher-custom-input") as HTMLInputElement;
    fireEvent.change(input, { target: { value: "htop" } });
    fireEvent.click(screen.getByTestId("launcher-custom-launch"));
    expect(input.value).toBe("");
  });

  it("does not launch empty or whitespace-only custom command", () => {
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />,
    );
    const input = screen.getByTestId("launcher-custom-input");
    fireEvent.change(input, { target: { value: "   " } });
    fireEvent.keyDown(input, { key: "Enter" });
    expect(onLaunch).not.toHaveBeenCalled();
  });

  it("launch button is disabled when custom input is empty", () => {
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />,
    );
    const btn = screen.getByTestId("launcher-custom-launch");
    expect(btn.hasAttribute("disabled")).toBe(true);
  });

  it("closes when backdrop is clicked", () => {
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />,
    );
    fireEvent.click(screen.getByTestId("terminal-launcher"));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("shows 'Creating session...' when isCreating is true", () => {
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} isCreating />,
    );
    expect(screen.getByText("Creating session...")).toBeTruthy();
  });

  it("disables all launch buttons when isCreating is true", () => {
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} isCreating />,
    );
    const emptyShell = screen.getByTestId("launcher-empty-shell");
    expect(emptyShell.hasAttribute("disabled")).toBe(true);

    const shortcutBtn = screen.getByTestId("launcher-shortcut-claude-code");
    expect(shortcutBtn.hasAttribute("disabled")).toBe(true);
  });

  it("fetches shortcuts from API when no prop provided", async () => {
    const apiShortcuts: ShortcutEntry[] = [
      { label: "Docker", command: "docker exec -it web bash", description: "Container shell" },
    ];
    mockGetEffectiveShortcuts.mockResolvedValueOnce(apiShortcuts);

    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} />,
    );

    await waitFor(() => {
      expect(screen.getByText("Docker")).toBeTruthy();
    });
    expect(mockGetEffectiveShortcuts).toHaveBeenCalledOnce();
  });

  it("falls back to DEFAULT_SHORTCUTS when API fetch fails", async () => {
    mockGetEffectiveShortcuts.mockRejectedValueOnce(new Error("network error"));

    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} />,
    );

    // DEFAULT_SHORTCUTS should be shown as fallback
    expect(screen.getByText("Claude Code")).toBeTruthy();
    expect(screen.getByText("Codex")).toBeTruthy();
  });

  it("skips API fetch when shortcuts prop is provided", () => {
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />,
    );
    expect(mockGetEffectiveShortcuts).not.toHaveBeenCalled();
  });
});
