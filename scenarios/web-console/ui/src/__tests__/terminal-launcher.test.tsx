import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent, waitFor, within } from "@testing-library/react";
import { DEFAULT_SHORTCUTS, type ShortcutEntry } from "../consts/shortcuts";
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

  it("calls onLaunch with the agent's command when its card is clicked", () => {
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />,
    );
    fireEvent.click(screen.getByTestId("launcher-agent-claude-code"));
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

  it("closes when the backdrop is pressed, not when the panel is pressed", () => {
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />,
    );
    const panel = screen.getByTestId("terminal-launcher");
    fireEvent.pointerDown(panel);
    expect(onClose).not.toHaveBeenCalled();
    // The backdrop dismisses on press, and it is addressed by its rooted test
    // id rather than by its position among the overlay's children.
    fireEvent.pointerDown(screen.getByTestId("terminal-launcher.backdrop"));
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

    const agentCard = screen.getByTestId("launcher-agent-claude-code");
    expect(agentCard.hasAttribute("disabled")).toBe(true);
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

  // The dedicated sign-in card is gone. Nothing in this codebase can tell
  // whether the operator is signed in, so the card was a permanent guess; the
  // agent itself says when a sign-in is needed. The capability stays as an
  // ordinary shortcut the operator can edit, reorder, or delete.
  it("renders no dedicated Codex sign-in card", () => {
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />);
    expect(screen.queryByTestId("launcher-codex-sign-in")).not.toBeInTheDocument();
  });

  it("keeps the sign-in command reachable as an editable shortcut", () => {
    expect(DEFAULT_SHORTCUTS.some((entry) => entry.command === "codex login --device-auth")).toBe(true);
  });

  it("launches the sign-in command from its shortcut row", () => {
    const withSignIn = DEFAULT_SHORTCUTS.filter((entry) => entry.command === "codex login --device-auth");
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={withSignIn} />);
    fireEvent.click(screen.getByTestId("launcher-agent-codex-sign-in"));
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
    // Neither the fleet's state nor its refresh control owns space in the
    // dialog: they belong to the list they describe, so they appear when the
    // machine menu is opened and cost nothing the rest of the time.
    expect(screen.queryByTestId("launcher-target-catalog-state")).toBeNull();
    expect(screen.queryByTestId("launcher-target-refresh")).toBeNull();

    fireEvent.click(screen.getByTestId("launcher-machine-picker"));
    expect(screen.getByTestId("launcher-target-catalog-state")).toHaveTextContent(strings.terminalLauncher.unconfigured);
    fireEvent.click(screen.getByTestId("launcher-target-refresh"));
    expect(onRefreshTargets).toHaveBeenCalledOnce();
  });

  // The machine menu is also where linking and administering live, so the
  // dialog never needs a card explaining either.
  it("offers linking and managing machines from inside the machine menu", () => {
    const onOpenMachines = vi.fn();
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} onOpenMachines={onOpenMachines} />);
    fireEvent.click(screen.getByTestId("launcher-machine-picker"));
    expect(screen.getByTestId("launcher-machine-link")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("launcher-machine-manage"));
    expect(onOpenMachines).toHaveBeenCalledOnce();
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
    // While the fleet is loading there is no trigger to open, so the refresh
    // control inside the menu is unreachable rather than merely disabled.
    expect(screen.queryByTestId("launcher-machine-picker")).toBeNull();
  });
  // -------------------------------------------------------------------
  // Destination and appearance disclosure
  // [REQ:P0-014a]
  // -------------------------------------------------------------------

  const groups = [
    { id: "g1", name: "Ship it", color: "#22d3ee", isCollapsed: false },
    { id: "g2", name: "Research", color: "#f59e0b", isCollapsed: false },
  ];

  // The control renders on EVERY open, not only when a group was implied.
  // One that appeared sometimes would be one the operator never learns to
  // look for.
  it("states its destination on every open, including with no pending group", () => {
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} groups={groups} />);
    const destination = screen.getByTestId("launcher-destination");
    expect(destination).toBeInTheDocument();
    // The card states the destination on its face, without being opened.
    expect(within(destination).getByTestId("launcher-destination-trigger"))
      .toHaveTextContent(strings.launcher.noGroup);
  });

  it("names the group it was opened from", () => {
    render(
      <TerminalLauncher
        open
        onClose={onClose}
        onLaunch={onLaunch}
        shortcuts={testShortcuts}
        groups={groups}
        pendingGroupId="g1"
      />,
    );
    const destination = screen.getByTestId("launcher-destination");
    const trigger = within(destination).getByTestId("launcher-destination-trigger");
    // The test locale echoes interpolated keys, so the assertion is on the
    // phrasing the trigger chose plus the accessible name it composed.
    expect(trigger).toHaveTextContent(strings.groupPicker.into);
    expect(trigger).toHaveAccessibleName(/Ship it/);
  });

  // The card is a door to the shared overlay, not a control of its own: the
  // same surface opens from the session menu.
  it("opens the shared group overlay when the destination card is pressed", () => {
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} groups={groups} />);
    expect(screen.queryByTestId("group-assign-picker")).toBeNull();
    fireEvent.click(screen.getByTestId("launcher-destination-trigger"));
    expect(screen.getByTestId("group-assign-picker")).toBeInTheDocument();
    expect(screen.getByTestId("group-picker-option-g1")).toHaveTextContent("Ship it");
    expect(screen.getByTestId("group-picker-option-none")).toBeInTheDocument();
  });

  it("reports a destination change to its caller", () => {
    const onDestinationChange = vi.fn();
    render(
      <TerminalLauncher
        open
        onClose={onClose}
        onLaunch={onLaunch}
        shortcuts={testShortcuts}
        groups={groups}
        pendingGroupId="g1"
        onDestinationChange={onDestinationChange}
      />,
    );
    fireEvent.click(screen.getByTestId("launcher-destination-trigger"));
    fireEvent.click(screen.getByTestId("group-picker-option-g2"));
    expect(onDestinationChange).toHaveBeenCalledWith("g2");
    // Choosing commits and closes: the overlay is not a place to linger.
    expect(screen.queryByTestId("group-assign-picker")).toBeNull();
  });

  it("creates a group by typing a name that matches none", async () => {
    const onCreateGroup = vi.fn().mockResolvedValue("g3");
    const onDestinationChange = vi.fn();
    render(
      <TerminalLauncher
        open
        onClose={onClose}
        onLaunch={onLaunch}
        shortcuts={testShortcuts}
        groups={groups}
        onCreateGroup={onCreateGroup}
        onDestinationChange={onDestinationChange}
      />,
    );
    fireEvent.click(screen.getByTestId("launcher-destination-trigger"));
    // One field filters the list and names a new group; the pinned action
    // creates whatever matched nothing.
    fireEvent.change(screen.getByTestId("group-picker-filter"), { target: { value: "Refactor pass" } });
    expect(screen.queryByTestId("group-picker-option-g1")).toBeNull();
    fireEvent.click(screen.getByTestId("group-picker-create-submit"));

    expect(onCreateGroup).toHaveBeenCalledWith("Refactor pass");
    await waitFor(() => { expect(onDestinationChange).toHaveBeenCalledWith("g3"); });
  });

  // Styling used to be applied silently after the session existed.
  it("names the appearance the session will receive", () => {
    render(
      <TerminalLauncher
        open
        onClose={onClose}
        onLaunch={onLaunch}
        shortcuts={testShortcuts}
        groups={groups}
        appearance={{ headerColor: "transparent", themeId: "midnight", fontSize: 16 }}
      />,
    );
    // The summary is an interpolated string, so the test locale echoes the key
    // rather than the values; open the row and read the named facts.
    fireEvent.click(screen.getByTestId("launcher-appearance-toggle"));
    const appearance = screen.getByTestId("launcher-appearance");
    expect(appearance).toHaveTextContent("midnight");
    expect(appearance).toHaveTextContent("16px");
  });

  it("says the colour comes from the group once a destination is chosen", () => {
    render(
      <TerminalLauncher
        open
        onClose={onClose}
        onLaunch={onLaunch}
        shortcuts={testShortcuts}
        groups={groups}
        pendingGroupId="g1"
        appearance={{ headerColor: "transparent", themeId: "default", fontSize: 14 }}
      />,
    );
    fireEvent.click(screen.getByTestId("launcher-appearance-toggle"));
    // With a destination chosen, the colour comes from the group, and the row
    // names the group rather than a raw colour value.
    expect(screen.getByTestId("launcher-appearance")).toHaveTextContent("Ship it");
  });

  // -------------------------------------------------------------------
  // Agent grid
  // -------------------------------------------------------------------

  // The grid renders a fixed set; the shortcut list is where it comes from.
  it("points at the surface that owns the agent list", () => {
    const onEditShortcuts = vi.fn();
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} onEditShortcuts={onEditShortcuts} />);
    fireEvent.click(screen.getByTestId("launcher-edit-shortcuts"));
    expect(onEditShortcuts).toHaveBeenCalledOnce();
  });

  it("omits the shortcut-editor card when no caller can open one", () => {
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />);
    expect(screen.queryByTestId("launcher-edit-shortcuts")).toBeNull();
  });

  it("renders four agent cards rather than eight rows", () => {
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={DEFAULT_SHORTCUTS} />);
    const grid = screen.getByTestId("launcher-agent-grid");
    // Four agents, the Codex sign-in shortcut, and the empty shell. The four
    // "(attributed)" duplicates are folded into their parents as one toggle.
    expect(within(grid).getAllByRole("button")).toHaveLength(6);
    expect(screen.getByTestId("launcher-agent-claude-code")).toBeInTheDocument();
    expect(screen.queryByTestId("launcher-agent-claude-code-attributed")).not.toBeInTheDocument();
  });

  it("launches the attributed variant when the toggle is on", () => {
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={DEFAULT_SHORTCUTS} />);
    fireEvent.click(screen.getByTestId("launcher-attributed-toggle"));
    fireEvent.click(screen.getByTestId("launcher-agent-codex"));
    expect(onLaunch).toHaveBeenCalledWith(
      expect.objectContaining({ command: expect.stringContaining("vrooli-agent-launcher") }),
    );
  });

  it("renders no full-width target card", () => {
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />);
    expect(screen.queryByTestId("launcher-target-card-local")).not.toBeInTheDocument();
    expect(screen.getByTestId("launcher-machine-picker")).toBeInTheDocument();
  });
});
