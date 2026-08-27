import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import type { ShortcutEntry } from "../consts/shortcuts";
import { asMockedClient } from "../test-utils";
import { strings } from "../consts/strings";

// [REQ:P0-006a] Terminal Launch Flow UI — component rendering & interactions
// [REQ:P0-006b] Configurable Shortcut Entries — shortcut rendering and selection
// [REQ:P1-002b] Shortcut Profile Management UI — API-driven shortcut loading via
// the Connect-Web ShortcutsService client.

vi.mock("../api/shortcuts", () => ({
  shortcutsClient: {
    getEffective: vi.fn(),
    listProfiles: vi.fn(),
    upsertProfile: vi.fn(),
    deleteProfile: vi.fn(),
  },
}));

import { shortcutsClient as _shortcutsClient } from "../api/shortcuts";
const shortcutsClient = asMockedClient(_shortcutsClient);

import TerminalLauncher, { type TerminalTarget } from "../components/TerminalLauncher";
import type { TargetCatalog } from "../api/targets";

const testShortcuts: ShortcutEntry[] = [
  { label: "Claude Code", command: "vrooli agent launch --runner claude --arg=--dangerously-skip-permissions", description: "AI coding assistant" },
  { label: "Codex", command: "codex --yolo", description: "OpenAI Codex CLI" },
];

describe("TerminalLauncher", () => {
  const onClose = vi.fn();
  const onLaunch = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
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
    expect(screen.getByText(strings.terminalLauncher.newTerminal)).toBeTruthy();
    expect(screen.getByTestId("terminal-launcher")).toBeTruthy();
  });

  it("renders empty shell option", () => {
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />,
    );
    expect(screen.getByTestId("launcher-empty-shell")).toBeTruthy();
    expect(screen.getByText(strings.terminalLauncher.emptyShell)).toBeTruthy();
  });

  it("explains and disables launch when no session backend is available", () => {
    render(
      <TerminalLauncher
        open={true}
        onClose={onClose}
        onLaunch={onLaunch}
        shortcuts={testShortcuts}
        availableBackends={[{ id: "standard", display_name: "Standard", description: "", survives_restart: false, available: false, reason: "no PTY implementation for this platform" }]}
      />,
    );
    expect(screen.getByTestId("launcher-no-backend").textContent).toContain("no PTY implementation for this platform");
    expect(screen.getByTestId("launcher-empty-shell")).toBeDisabled();
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
      expect.objectContaining({ command: "vrooli agent launch --runner claude --arg=--dangerously-skip-permissions" }),
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

  it("closes when backdrop is clicked, not when the panel is clicked", () => {
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />,
    );
    const panel = screen.getByTestId("terminal-launcher");
    fireEvent.click(panel);
    expect(onClose).not.toHaveBeenCalled();
    const backdrop = panel.parentElement?.firstElementChild as HTMLElement;
    fireEvent.click(backdrop);
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("closes on Escape (dialog semantics via DrawerShell)", () => {
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />,
    );
    const panel = screen.getByTestId("terminal-launcher");
    expect(panel.getAttribute("role")).toBe("dialog");
    expect(panel.getAttribute("aria-modal")).toBe("true");
    fireEvent.keyDown(window, { key: "Escape" });
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("traps focus inside the launcher, cycling through its native selects", () => {
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />,
    );
    // Expand session options so the selects are in the tab order.
    fireEvent.click(screen.getByTestId("launcher-options-toggle"));
    const panel = screen.getByTestId("terminal-launcher");
    const timeoutSelect = screen.getByTestId("launcher-timeout-select");
    timeoutSelect.focus();
    fireEvent.keyDown(panel, { key: "Tab" });
    expect(panel.contains(document.activeElement)).toBe(true);
    fireEvent.keyDown(panel, { key: "Tab", shiftKey: true });
    expect(panel.contains(document.activeElement)).toBe(true);
  });

  it("shows 'Creating session...' when isCreating is true", () => {
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} isCreating />,
    );
    expect(screen.getByText(strings.terminalLauncher.creating)).toBeTruthy();
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

  it("fetches shortcuts via shortcutsClient.getEffective when no prop provided", async () => {
    shortcutsClient.getEffective.mockResolvedValueOnce({
      shortcuts: [
        { $typeName: "vrooli.web_console.v1.shortcuts.Shortcut", label: "Docker", command: "docker exec -it web bash", description: "Container shell" },
      ],
    });

    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} />,
    );

    await waitFor(() => {
      expect(screen.getByText("Docker")).toBeTruthy();
    });
    expect(shortcutsClient.getEffective).toHaveBeenCalledOnce();
    expect(shortcutsClient.getEffective).toHaveBeenCalledWith({});
  });

  it("falls back to DEFAULT_SHORTCUTS when API fetch fails", async () => {
    shortcutsClient.getEffective.mockRejectedValueOnce(new Error("network error"));

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
    expect(shortcutsClient.getEffective).not.toHaveBeenCalled();
  });

  it("selects an available remote target and exposes its readiness facts", () => {
    const targets: TerminalTarget[] = [
      { id: "node-1", kind: "bridge-node", label: "Mac mini", available: true, state: "dispatchable", readiness: [{ key: "heartbeat", label: "Heartbeat fresh", passed: true, detail: "fresh" }, { key: "dispatch", label: "Dispatchable", passed: true, detail: "ready" }] },
      { id: "node-2", kind: "bridge-node", label: "Offline host", available: false, state: "offline", failure_rung: "live channel", recovery_action: "Reconnect the Bridge agent" },
    ];
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} availableTargets={targets} />,
    );
    fireEvent.click(screen.getByTestId("launcher-options-toggle"));
    const select = screen.getByTestId("launcher-target-select") as HTMLSelectElement;
    expect(select.options[2]?.disabled).toBe(false);
    fireEvent.change(select, { target: { value: "node-1" } });
    expect(screen.getByText("Heartbeat fresh")).toBeTruthy();
    fireEvent.click(screen.getByTestId("launcher-empty-shell"));
    expect(onLaunch).toHaveBeenCalledWith(expect.objectContaining({ target: expect.objectContaining({ id: "node-1" }) }));
  });

  it("puts remote locations in the primary surface and keeps unavailable nodes inspectable", () => {
    const targets: TerminalTarget[] = [
      { id: "node-1", kind: "bridge-node", label: "Build node", available: true, state: "dispatchable", os: "linux", arch: "amd64" },
      { id: "node-2", kind: "bridge-node", label: "Offline host", available: false, state: "offline", last_seen_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(), failure_rung: "heartbeat freshness", recovery_action: "Reconnect the Bridge agent, then refresh" },
    ];
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} availableTargets={targets} />);

    expect(screen.getByTestId("launcher-target-card-node-1")).toBeTruthy();
    expect(screen.getByTestId("launcher-target-card-node-2")).toBeTruthy();
    expect(screen.getByTestId("launcher-linked-machines-footer")).toBeTruthy();
    fireEvent.click(screen.getByTestId("launcher-target-card-node-2"));
    expect(screen.getByText("Reconnect the Bridge agent, then refresh")).toBeTruthy();
    expect(screen.getByTestId("launcher-target-card-node-2")).toHaveTextContent(/2h ago/);
    expect(screen.getByTestId("launcher-empty-shell")).toBeDisabled();
  });

  it("shows the concrete remote grant beside its operator-facing summary", () => {
    const targets: TerminalTarget[] = [
      {
        id: "node-1",
        kind: "bridge-node",
        label: "Mac mini",
        available: true,
        state: "dispatchable",
        readiness: [{ key: "bridge_scope", label: "Bridge scope", passed: true, detail: "Read only; changes are not permitted. Granted scopes: system-monitor:read" }],
      },
    ];
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} availableTargets={targets} />);

    expect(screen.getByTestId("launcher-target-card-node-1")).toHaveTextContent("Read only; changes are not permitted. Granted scopes: system-monitor:read");
    fireEvent.click(screen.getByTestId("launcher-target-card-node-1"));
    expect(screen.getByText("Grant: Read only; changes are not permitted. Granted scopes: system-monitor:read")).toBeInTheDocument();
  });

  it("supports arrow-key navigation across target cards", () => {
    const targets: TerminalTarget[] = [
      { id: "node-1", kind: "bridge-node", label: "Build node", available: true, state: "dispatchable" },
      { id: "node-2", kind: "bridge-node", label: "Test node", available: true, state: "dispatchable" },
    ];
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} availableTargets={targets} />);

    const local = screen.getByTestId("launcher-target-card-local");
    const firstRemote = screen.getByTestId("launcher-target-card-node-1");
    local.focus();
    fireEvent.keyDown(local, { key: "ArrowDown" });
    expect(firstRemote).toHaveFocus();
    expect(firstRemote).toHaveAttribute("aria-selected", "true");
  });

  it("offers Codex device authentication as an explicit launch action", () => {
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />);
    fireEvent.click(screen.getByTestId("launcher-codex-sign-in"));
    expect(onLaunch).toHaveBeenCalledWith(expect.objectContaining({ command: "codex login --device-auth" }));
  });

  it("shows catalog recovery state and refresh affordance", () => {
    const onRefreshTargets = vi.fn();
    const targetCatalog: TargetCatalog = {
      status: "unconfigured",
      targets: [],
      message: "Remote nodes are not configured",
      recovery_action: "Configure Bridge access on the Web Console server",
    };
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} targetCatalog={targetCatalog} onRefreshTargets={onRefreshTargets} />);
    expect(screen.getByTestId("launcher-target-catalog-state")).toHaveTextContent(strings.terminalLauncher.unconfigured);
    fireEvent.click(screen.getByTestId("launcher-target-refresh"));
    expect(onRefreshTargets).toHaveBeenCalledOnce();
  });

  it("shows target loading state and disables refresh while loading", () => {
    const onRefreshTargets = vi.fn();
    render(
      <TerminalLauncher
        open
        onClose={onClose}
        onLaunch={onLaunch}
        shortcuts={testShortcuts}
        targetsLoading
        onRefreshTargets={onRefreshTargets}
      />,
    );

    expect(screen.getByTestId("launcher-target-loading")).toBeInTheDocument();
    expect(screen.getByTestId("launcher-target-refresh")).toBeDisabled();
  });

  it("filters remote targets and reports when no node matches", () => {
    const targets: TerminalTarget[] = [
      { id: "one", kind: "bridge-node", label: "Build one", available: true, state: "dispatchable", os: "linux", arch: "amd64" },
      { id: "two", kind: "bridge-node", label: "Build two", available: true, state: "dispatchable", os: "linux", arch: "amd64" },
      { id: "three", kind: "bridge-node", label: "Build three", available: true, state: "dispatchable", os: "linux", arch: "amd64" },
      { id: "four", kind: "bridge-node", label: "Build four", available: true, state: "dispatchable", os: "linux", arch: "amd64" },
    ];
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} availableTargets={targets} />);

    const search = screen.getByTestId("launcher-target-search");
    fireEvent.change(search, { target: { value: "does-not-exist" } });

    expect(screen.getByText(strings.terminalLauncher.noRemoteNodes)).toBeInTheDocument();
    expect(screen.queryByTestId("launcher-target-card-one")).not.toBeInTheDocument();
  });

  it("renders every target status and preserves an invalid last-seen value", () => {
    const targets: TerminalTarget[] = [
      { id: "update", kind: "bridge-node", label: "Needs update", available: true, state: "needs-update", last_seen_at: "not-a-date" },
      { id: "offline", kind: "bridge-node", label: "Offline", available: false, state: "offline" },
      { id: "unconfigured", kind: "bridge-node", label: "Unconfigured", available: false, state: "unconfigured" },
      { id: "unknown", kind: "bridge-node", label: "Unknown", available: false },
    ];
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} availableTargets={targets} />);

    expect(screen.getByTestId("launcher-target-card-update")).toBeInTheDocument();
    expect(screen.getByTestId("launcher-target-card-offline")).toBeInTheDocument();
    expect(screen.getByTestId("launcher-target-card-unconfigured")).toBeInTheDocument();
    expect(screen.getByTestId("launcher-target-card-unknown")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("launcher-target-card-update"));
    expect(screen.getByText(/not-a-date/)).toBeInTheDocument();
  });
});
