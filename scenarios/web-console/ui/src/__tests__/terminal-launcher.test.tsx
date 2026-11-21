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
    getAgentPreferences: vi.fn(),
    updateAgentPreferences: vi.fn(),
  },
}));
vi.mock("../api/capabilities", async () => {
  const actual = await vi.importActual<typeof import("../api/capabilities")>("../api/capabilities");
  return { ...actual, capabilitiesClient: { agentUpdate: vi.fn(), agentUsage: vi.fn(), listHarnesses: vi.fn(), runAgentUpdateAction: vi.fn() } };
});
vi.mock("../api/modelSelection", () => ({
  modelSelectionClient: { getRoleModelChoices: vi.fn() },
}));
vi.mock("../api/monetization", async () => {
  const actual = await vi.importActual<typeof import("../api/monetization")>("../api/monetization");
  return { ...actual, getBusinessAccounts: vi.fn(), getBusinessAccountUsage: vi.fn() };
});

import { shortcutsClient as _shortcutsClient } from "../api/shortcuts";
const shortcutsClient = asMockedClient(_shortcutsClient);
import { capabilitiesClient as _capabilitiesClient } from "../api/capabilities";
const capabilitiesClient = asMockedClient(_capabilitiesClient);
import { modelSelectionClient as _modelSelectionClient } from "../api/modelSelection";
const modelSelectionClient = asMockedClient(_modelSelectionClient);
import { getBusinessAccountUsage as _getBusinessAccountUsage, getBusinessAccounts as _getBusinessAccounts } from "../api/monetization";
const getBusinessAccounts = vi.mocked(_getBusinessAccounts);
const getBusinessAccountUsage = vi.mocked(_getBusinessAccountUsage);

import TerminalLauncher, { type TerminalTarget } from "../components/TerminalLauncher";
import { resetHarnessCacheForTests } from "../components/launcher/useHarnesses";
import type { TargetCatalog } from "../api/targets";

// Entries carry agentId because the server resolves it and sends it; an entry
// without one is, by contract, a plain operator command rather than an agent.
const testShortcuts: ShortcutEntry[] = [
  { label: "Claude Code", command: "vrooli agent launch --runner claude --arg=--dangerously-skip-permissions", description: "AI coding assistant", agentId: "claude" },
  { label: "Codex", command: "codex --yolo", description: "OpenAI Codex CLI", agentId: "codex" },
];

/** A machine that reports codex as missing — the shape every install test needs. */
function missingCodexTarget(): TerminalTarget {
  return {
    id: "local", kind: "local", label: "This machine", available: true, state: "dispatchable",
    readiness: [
      { key: "capability:codex", label: "Codex", passed: false, detail: "codex is not installed", state: "missing" },
    ],
  };
}

/** The library Select keeps a native control beside its trigger; drive it directly. */
const nativeSelect = (testId: string) => {
  const native = screen.getByTestId(testId).closest("[data-rcl-select-field]")?.querySelector("select");
  if (!native) throw new Error(`no select for ${testId}`);
  return native;
};
const chooseSelect = (testId: string, value: string) => { fireEvent.change(nativeSelect(testId), { target: { value } }); };
const roleChoice = (over: Record<string, unknown>) => ({ displayName: typeof over.model === "string" ? over.model : "", model: "", source: "", sourceLabel: "", provider: "", launchModel: "", effort: "", isDefault: false, contextWindowTokens: 0n, contextLimitTokens: [], metered: false, ...over });
const policyOf = (over: Record<string, unknown>) => ({ sourceConfigurationSupported: true, operatorCommand: "", commandLiteral: false, policyStatus: "ready", availability: "unknown", runner: "opencode", role: "code.default", modelFlag: "-m", provenance: "resource-owned policy", observedAt: "2026-10-30", detail: "", policyDigest: "digest", catalogStale: false, choices: [], ...over });
const openrouter = roleChoice({ model: "openrouter/example/model", source: "openrouter", sourceLabel: "OpenRouter", provider: "openrouter", launchModel: "openrouter/example/model", isDefault: true });
const credits = roleChoice({ model: "coding", source: "vrooli-credits", sourceLabel: "Vrooli credits", provider: "vrooli", launchModel: "vrooli/coding" });
const renderAgentMode = (ui: Parameters<typeof render>[0]) => {
  const result = render(ui);
  fireEvent.click(screen.getByTestId("launcher-mode-agent"));
  return result;
};
/** Manage, then one harness's page. */
const openDetail = (key: string) => {
  fireEvent.click(screen.getByTestId("launcher-manage-agents"));
  fireEvent.click(screen.getByTestId(`launcher-manage-open-${key}`));
};

describe("TerminalLauncher", () => {
  const onClose = vi.fn();
  const onLaunch = vi.fn();

  beforeEach(() => {
    window.localStorage.clear();
    vi.clearAllMocks();
    resetHarnessCacheForTests();
    capabilitiesClient.agentUpdate.mockRejectedValue(new Error("offline"));
    capabilitiesClient.agentUsage.mockRejectedValue(new Error("offline"));
    capabilitiesClient.listHarnesses.mockRejectedValue(new Error("offline"));
    shortcutsClient.getAgentPreferences.mockResolvedValue({ preferences: { profileId: "ws-1", profileInstance: "instance-1", revision: 0n, orderedAgentIds: [], hiddenAgentIds: [], orderMode: "custom", lastLaunchedAt: {} } } as never);
    shortcutsClient.updateAgentPreferences.mockResolvedValue({ preferences: { profileId: "ws-1", profileInstance: "instance-1", revision: 1n, orderedAgentIds: ["codex", "claude"], hiddenAgentIds: [], orderMode: "custom", lastLaunchedAt: {} } } as never);
    modelSelectionClient.getRoleModelChoices.mockResolvedValue(policyOf({ policyStatus: "unsupported", modelFlag: "", provenance: "", observedAt: "", runner: "", role: "" }) as never);
    getBusinessAccounts.mockResolvedValue({ accounts: [] });
    getBusinessAccountUsage.mockResolvedValue({ business_account_id: "", billing_period: "", tier: "", limits: {}, usage: {}, reserved: {}, remaining: {}, display_credits: {} });
  });

  it("renders nothing when closed", () => {
    const { container } = render(
      <TerminalLauncher open={false} onClose={onClose} onLaunch={onLaunch} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders the drawer titled 'New session' when open", () => {
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />,
    );
    expect(screen.getByText(strings.terminalLauncher.newSession)).toBeTruthy();
    expect(screen.getByTestId("terminal-launcher")).toBeTruthy();
  });

  it("offers a shell as a session type that says agent settings do not apply", () => {
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />,
    );
    fireEvent.click(screen.getByTestId("launcher-mode-shell"));
    expect(screen.getByTestId("launcher-shell-hint")).toHaveTextContent(strings.terminalLauncher.shellHint);
    expect(screen.getByTestId("launcher-submit-selection")).toHaveTextContent(strings.terminalLauncher.launchSelectedShell);
  });

  it("opens with the requested machine selected", () => {
    render(
      <TerminalLauncher
        open={true}
        onClose={onClose}
        onLaunch={onLaunch}
        shortcuts={testShortcuts}
        initialTarget={{ id: "machine-1", kind: "bridge-node", label: "Remote machine", available: true }}
        availableTargets={[{ id: "machine-1", kind: "bridge-node", label: "Remote machine", available: true }]}
      />,
    );

    expect(screen.getByTestId("launcher-machine-picker")).toHaveAttribute("aria-label", "launcher.machine: Remote machine");
  });

  it("explains and disables launch when no session backend is available", () => {
    renderAgentMode(
      <TerminalLauncher
        open={true}
        onClose={onClose}
        onLaunch={onLaunch}
        shortcuts={testShortcuts}
        availableBackends={[{ id: "standard", display_name: "Standard", description: "", survives_restart: false, available: false, reason: "no PTY implementation for this platform" }]}
      />,
    );
    expect(screen.getByTestId("launcher-no-backend").textContent).toContain("no PTY implementation for this platform");
    expect(screen.getByTestId("launcher-submit-selection")).toBeDisabled();
    fireEvent.click(screen.getByTestId("launcher-mode-shell"));
    expect(screen.getByTestId("launcher-submit-selection")).toBeDisabled();
  });

  it("opens a shell with undefined backend so the server applies its configured default", () => {
    render(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />,
    );
    fireEvent.click(screen.getByTestId("launcher-mode-shell"));
    expect(onLaunch).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("launcher-submit-selection"));
    expect(onLaunch).toHaveBeenCalledWith(expect.objectContaining({ backend: undefined }));
  });

  it("renders shortcut entries from props", () => {
    renderAgentMode(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />,
    );
    expect(screen.getByText("Claude Code")).toBeTruthy();
    expect(screen.getByText("Codex")).toBeTruthy();
    expect(screen.getByTestId("launcher-agent-claude-code")).toHaveAttribute("title", testShortcuts[0]?.command);
  });

  it("selects an agent without launching and launches it only after explicit submission", async () => {
    renderAgentMode(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />,
    );
    fireEvent.click(screen.getByTestId("launcher-agent-codex"));
    fireEvent.click(screen.getByTestId("launcher-agent-claude-code"));
    expect(onLaunch).not.toHaveBeenCalled();
    expect(screen.getByTestId("launcher-agent-claude-code")).toHaveAttribute("aria-checked", "true");
    await waitFor(() => { expect(modelSelectionClient.getRoleModelChoices).toHaveBeenCalledWith({ agentId: "claude", targetId: "local", launchCommand: testShortcuts[0]?.command, businessAccountId: "" }); });
    fireEvent.click(screen.getByTestId("launcher-submit-selection"));
    expect(onLaunch).toHaveBeenCalledWith(
      expect.objectContaining({ command: "vrooli agent launch --runner claude --arg=--dangerously-skip-permissions" }),
    );
  });

  it("shows the selected agent's policy-backed source and model and submits that exact choice", async () => {
    modelSelectionClient.getRoleModelChoices.mockResolvedValue(policyOf({ choices: [openrouter, roleChoice({ model: "gemma4:12b", source: "ollama", sourceLabel: "Local Ollama", provider: "ollama", launchModel: "ollama/gemma4:12b" })] }) as never);
    renderAgentMode(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={[{ label: "OpenCode", command: "opencode", agentId: "opencode" }]} />);
    await screen.findByTestId("launcher-model-source");
    fireEvent.click(screen.getByTestId("launcher-submit-selection"));
    expect(onLaunch).toHaveBeenLastCalledWith(expect.objectContaining({ command: "opencode" }));
    chooseSelect("launcher-model-source", "ollama");
    expect(within(nativeSelect("launcher-model-choice")).getAllByRole("option", { hidden: true })[0]).toHaveTextContent("gemma4:12b");
    fireEvent.click(screen.getByTestId("launcher-submit-selection"));
    expect(onLaunch).toHaveBeenLastCalledWith(expect.objectContaining({ command: "opencode", agentLaunch: expect.objectContaining({ agent_id: "opencode", model: "gemma4:12b", provider: "ollama" }) }));
  });

  it("refreshes a stale local model catalog automatically when the agent is selected", async () => {
    const stale = policyOf({ runner: "codex", catalogStale: true, policyStatus: "stale", detail: "Model list is old" });
    const fresh = policyOf({ runner: "codex", catalogStale: false, policyStatus: "ready", choices: [roleChoice({ model: "gpt-6-codex", launchModel: "gpt-6-codex", isDefault: true })] });
    let codexReads = 0;
    modelSelectionClient.getRoleModelChoices.mockImplementation(((request: { agentId: string }) => Promise.resolve(
      request.agentId === "codex" && codexReads++ === 0 ? stale : fresh,
    )) as never);
    capabilitiesClient.runAgentUpdateAction.mockResolvedValue({ success: true, message: "Model list refreshed" } as never);

    renderAgentMode(<TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />);
    fireEvent.click(screen.getByTestId("launcher-agent-codex"));

    await waitFor(() => {
      expect(capabilitiesClient.runAgentUpdateAction).toHaveBeenCalledWith(
        { targetId: "local", agentId: "codex", action: "refresh_catalog" },
        expect.anything(),
      );
    });
    await waitFor(() => expect(modelSelectionClient.getRoleModelChoices.mock.calls.length).toBeGreaterThan(1));
    expect(screen.queryByTestId("launcher-catalog-stale")).not.toBeInTheDocument();
  });

  it("requires an explicit business account for the qualified Vrooli credits source", async () => {
    getBusinessAccounts.mockResolvedValue({ accounts: [
      { id: "acct-one", display_name: "Studio", role: "owner", created_at: "2026-10-30" },
      { id: "acct-two", display_name: "Research", role: "member", created_at: "2026-10-30" },
    ] });
    getBusinessAccountUsage.mockImplementation((accountId) => Promise.resolve({ business_account_id: accountId, billing_period: "2026-10", tier: "pro", limits: { ai_credits: 1000 }, usage: { ai_credits: 100 }, reserved: { ai_credits: 10 }, remaining: { ai_credits: 890 }, display_credits: { ai_credits: 0.00089 } }));
    modelSelectionClient.getRoleModelChoices.mockResolvedValue(policyOf({ choices: [openrouter, credits] }) as never);
    renderAgentMode(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={[{ label: "OpenCode", command: "opencode", agentId: "opencode" }]} />);
    await screen.findByTestId("launcher-model-source");
    chooseSelect("launcher-model-source", "vrooli-credits");
    await screen.findByTestId("launcher-business-account");
    expect(nativeSelect("launcher-business-account")).toHaveValue("acct-one");
    await screen.findByTestId("launcher-business-account-usage");
    chooseSelect("launcher-business-account", "acct-two");
    await waitFor(() => { expect(getBusinessAccountUsage).toHaveBeenLastCalledWith("acct-two"); });
    await screen.findByTestId("launcher-business-account-usage");
    fireEvent.click(screen.getByTestId("launcher-submit-selection"));
    expect(onLaunch).toHaveBeenCalledWith(expect.objectContaining({
      command: "opencode",
      agentLaunch: expect.objectContaining({ model: "coding", provider: "vrooli-credits" }),
      businessAccountId: "acct-two",
    }));
  });

  it("keeps qualified Vrooli coding unavailable when account usage facts cannot be read", async () => {
    getBusinessAccounts.mockResolvedValue({ accounts: [
      { id: "acct-one", display_name: "Studio", role: "owner", created_at: "2026-10-30" },
    ] });
    getBusinessAccountUsage.mockRejectedValueOnce(new Error("offline"));
    modelSelectionClient.getRoleModelChoices.mockResolvedValue(policyOf({ choices: [openrouter, credits] }) as never);
    renderAgentMode(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={[{ label: "OpenCode", command: "opencode", agentId: "opencode" }]} />);
    await screen.findByTestId("launcher-model-source");
    chooseSelect("launcher-model-source", "vrooli-credits");
    await screen.findByTestId("launcher-business-account-usage-error");
    expect(screen.getByTestId("launcher-submit-selection")).toBeDisabled();
    expect(onLaunch).not.toHaveBeenCalled();
  });

  it("disables the paid-source submit when the selected account has no remaining AI credits", async () => {
    getBusinessAccounts.mockResolvedValue({ accounts: [
      { id: "acct-empty", display_name: "Empty account", role: "owner", created_at: "2026-10-30" },
    ] });
    getBusinessAccountUsage.mockResolvedValueOnce({ business_account_id: "acct-empty", billing_period: "2026-10", tier: "pro", limits: { ai_credits: 1000 }, usage: { ai_credits: 1000 }, reserved: { ai_credits: 0 }, remaining: { ai_credits: 0 }, display_credits: { ai_credits: 0 } });
    modelSelectionClient.getRoleModelChoices.mockResolvedValue(policyOf({ choices: [openrouter, credits] }) as never);
    renderAgentMode(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={[{ label: "OpenCode", command: "opencode", agentId: "opencode" }]} />);
    await screen.findByTestId("launcher-model-source");
    chooseSelect("launcher-model-source", "vrooli-credits");
    await screen.findByTestId("launcher-business-account-usage");
    expect(screen.getByTestId("launcher-submit-selection")).toBeDisabled();
    expect(onLaunch).not.toHaveBeenCalled();
  });

  it("keeps the command draft while the operator looks at another session type", () => {
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />);
    fireEvent.click(screen.getByTestId("launcher-mode-shell"));
    fireEvent.change(screen.getByTestId("launcher-custom-input"), { target: { value: "my-project command --flag" } });
    fireEvent.click(screen.getByTestId("launcher-mode-agent"));
    expect(screen.queryByTestId("launcher-custom-input")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("launcher-mode-shell"));
    expect(screen.getByTestId("launcher-custom-input")).toHaveValue("my-project command --flag");
  });

  it("shows no model controls for a shell and launches only on submission", () => {
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />);
    fireEvent.click(screen.getByTestId("launcher-mode-shell"));
    expect(onLaunch).not.toHaveBeenCalled();
    expect(screen.queryByTestId("launcher-model-selection")).not.toBeInTheDocument();
    expect(screen.queryByTestId("launcher-model-source")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("launcher-submit-selection"));
    expect(onLaunch).toHaveBeenCalledWith(expect.objectContaining({ command: undefined }));
  });

  const openShell = () => {
    render(<TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />);
    fireEvent.click(screen.getByTestId("launcher-mode-shell"));
    return screen.getByTestId<HTMLInputElement>("launcher-custom-input");
  };

  it("custom command input launches on Enter key", () => {
    const input = openShell();
    fireEvent.change(input, { target: { value: "htop" } });
    fireEvent.keyDown(input, { key: "Enter" });
    expect(onLaunch).toHaveBeenCalledWith(expect.objectContaining({ command: "htop" }));
  });

  it("custom command input clears after launch", () => {
    const input = openShell();
    fireEvent.change(input, { target: { value: "htop" } });
    fireEvent.click(screen.getByTestId("launcher-submit-selection"));
    expect(input.value).toBe("");
  });

  it("treats a whitespace-only command as a plain shell, never a blank command", () => {
    const input = openShell();
    fireEvent.change(input, { target: { value: "   " } });
    fireEvent.keyDown(input, { key: "Enter" });
    expect(onLaunch).toHaveBeenCalledWith(expect.objectContaining({ command: undefined }));
  });

  // The limit belongs to the agent path. Typing an agent's name here must run
  // exactly what was typed, and the panel says so rather than implying otherwise.
  it("sends a typed agent command as written and states that agent settings do not apply", () => {
    const input = openShell();
    expect(screen.getByTestId("launcher-shell-hint")).toHaveTextContent(strings.terminalLauncher.shellHint);
    fireEvent.change(input, { target: { value: "claude" } });
    fireEvent.click(screen.getByTestId("launcher-submit-selection"));
    expect(onLaunch).toHaveBeenCalledWith(expect.objectContaining({ command: "claude" }));
  });

  it("renders the command bar as one field: prefix and input share a group", () => {
    const input = openShell();
    const group = input.closest("[data-rcl-input-group]");
    expect(group).not.toBeNull();
    // The `$` is a real adornment now, not an absolutely-positioned span with
    // a hand-tuned padding on the input behind it.
    const prefix = group?.querySelector("[data-rcl-input-group-adornment]");
    expect(prefix?.textContent).toBe("$");
    expect(prefix).toHaveAttribute("aria-hidden", "true");
    expect(prefix).toHaveAttribute("data-side", "leading");
    expect(input).not.toHaveClass("ps-7");
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

  it("traps focus inside the launcher, cycling through its option controls", () => {
    renderAgentMode(
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
    renderAgentMode(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} isCreating />,
    );
    expect(screen.getByText(strings.terminalLauncher.creating)).toBeTruthy();
  });

  it("disables launching and agent choice when isCreating is true", () => {
    renderAgentMode(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} isCreating />,
    );
    expect(screen.getByTestId("launcher-submit-selection")).toBeDisabled();
    expect(screen.getByTestId("launcher-agent-claude-code")).toBeDisabled();
  });

  it("fetches shortcuts via shortcutsClient.getEffective when no prop provided", async () => {
    shortcutsClient.getEffective.mockResolvedValueOnce({
      shortcuts: [
        { $typeName: "vrooli.web_console.v1.shortcuts.Shortcut", label: "Docker", command: "docker exec -it web bash", description: "Container shell" },
      ],
    });

    renderAgentMode(
      <TerminalLauncher open={true} onClose={onClose} onLaunch={onLaunch} />,
    );
    await waitFor(() => { expect(shortcutsClient.getAgentPreferences).toHaveBeenCalled(); });
    expect(shortcutsClient.getEffective).toHaveBeenCalledOnce();
    expect(shortcutsClient.getEffective).toHaveBeenCalledWith({});
    fireEvent.click(screen.getByTestId("launcher-mode-shell"));
    expect(await screen.findByText("Docker")).toBeInTheDocument();
  });

  it("falls back to DEFAULT_SHORTCUTS when API fetch fails", () => {
    shortcutsClient.getEffective.mockRejectedValueOnce(new Error("network error"));

    renderAgentMode(
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
    fireEvent.click(screen.getByTestId("launcher-mode-shell"));
    fireEvent.click(screen.getByTestId("launcher-shortcut-codex-sign-in"));
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
    renderAgentMode(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} groups={groups} />);
    const destination = screen.getByTestId("launcher-destination");
    expect(destination).toBeInTheDocument();
    // The card states the destination on its face, without being opened.
    expect(within(destination).getByTestId("launcher-destination-trigger"))
      .toHaveTextContent(strings.launcher.noGroup);
  });

  it("names the group it was opened from", () => {
    renderAgentMode(
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
    renderAgentMode(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} groups={groups} />);
    expect(screen.queryByTestId("group-assign-picker")).toBeNull();
    fireEvent.click(screen.getByTestId("launcher-destination-trigger"));
    expect(screen.getByTestId("group-assign-picker")).toBeInTheDocument();
    expect(screen.getByTestId("group-picker-option-g1")).toHaveTextContent("Ship it");
    expect(screen.getByTestId("group-picker-option-none")).toBeInTheDocument();
  });

  it("reports a destination change to its caller", () => {
    const onDestinationChange = vi.fn();
    renderAgentMode(
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
    renderAgentMode(
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
    renderAgentMode(
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
    renderAgentMode(
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
    renderAgentMode(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} onEditShortcuts={onEditShortcuts} />);
    fireEvent.click(screen.getByTestId("launcher-manage-agents"));
    fireEvent.click(screen.getByTestId("launcher-edit-shortcuts"));
    expect(onEditShortcuts).toHaveBeenCalledOnce();
  });

  it("omits the shortcut-editor card when no caller can open one", () => {
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />);
    expect(screen.queryByTestId("launcher-edit-shortcuts")).toBeNull();
  });

  it("renders one card per agent", () => {
    renderAgentMode(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={DEFAULT_SHORTCUTS} />);
    const grid = screen.getByTestId("launcher-agent-grid");
    // Four agents. The Codex sign-in entry names no agent, so it belongs with
    // the shell's shortcuts rather than in the rail.
    expect(within(grid).getAllByRole("radio")).toHaveLength(4);
    expect(screen.getByTestId("launcher-agent-claude-code")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("launcher-mode-shell"));
    expect(screen.getByTestId("launcher-shortcut-codex-sign-in")).toBeInTheDocument();
  });

  // The eight-entry list, half of it "(attributed)" duplicates, drifted out of
  // sync with the server's four for long enough to grow a folding module of
  // its own. The fallback exists to match the server, not to extend it.
  it("keeps the client fallback identical in shape to the server defaults", () => {
    expect(DEFAULT_SHORTCUTS.some((entry) => entry.label.includes("(attributed)"))).toBe(false);
    for (const entry of DEFAULT_SHORTCUTS) {
      expect(entry.command).not.toContain("command -v");
    }
  });

  // A card reports what the machine says about the agent, not a truncated
  // sentence. The version is the fact an operator wants before launching.
  it("shows the reported version in agent management", () => {
    const target: TerminalTarget = {
      id: "local", kind: "local", label: "This machine", available: true, state: "dispatchable",
      readiness: [
        { key: "capability:codex", label: "Codex", passed: true, detail: "", state: "ready", version: "0.149.1" },
      ],
    };
    renderAgentMode(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} availableTargets={[target]} />);
    expect(screen.getByTestId("launcher-agent-codex")).not.toHaveTextContent("0.149.1");
    openDetail("codex");
    expect(screen.getByTestId("launcher-version")).toHaveTextContent("0.149.1");
  });

  // The machine name shared one line with its readiness summary, and the
  // summary was marked shrink-0 — so a machine called anything at all was
  // truncated to a single letter while "darwin/amd64 · 1/5 agents ready" took
  // the row. The name is what is being chosen; it gets its own line.
  it("gives the machine name its own line in the menu, not a share of one", () => {
    const target: TerminalTarget = {
      id: "bridge-node:mac", kind: "bridge-node", node_id: "mac", label: "matt-macbook-pro",
      os: "darwin", arch: "amd64", available: true, state: "dispatchable",
      readiness: [
        { key: "capability:claude", label: "Claude Code", passed: false, detail: "", state: "missing" },
        { key: "capability:agy", label: "Antigravity", passed: true, detail: "", state: "ready", version: "1.1.22" },
      ],
    };
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={[]} initialTarget={target} availableTargets={[target]} />);
    const trigger = screen.getByTestId("launcher-machine-picker");
    expect(trigger).toHaveTextContent("matt-macbook-pro");
    fireEvent.click(trigger);
    const option = screen.getByRole("option", { name: /matt-macbook-pro/ });
    const name = within(option).getByText("matt-macbook-pro");
    const meta = within(option).getByText(/darwin\/amd64/);
    expect(name).not.toBe(meta);
    // Siblings stacked, so neither can eat the other's width.
    expect(name.parentElement).toBe(meta.parentElement);
    expect(name).toHaveClass("block");
  });

  // A capability that is NOT ready was appended as a bare label, so a row
  // reading "1/5 agents ready · Claude Code" named the missing agent in the
  // position a reader takes for the present one.
  it("names a missing agent as missing rather than listing it", () => {
    const target: TerminalTarget = {
      id: "bridge-node:mac", kind: "bridge-node", node_id: "mac", label: "matt-macbook-pro",
      os: "darwin", arch: "amd64", available: true, state: "dispatchable",
      readiness: [
        { key: "capability:claude", label: "Claude Code", passed: false, detail: "", state: "missing" },
        { key: "capability:agy", label: "Antigravity", passed: true, detail: "", state: "ready", version: "1.1.22" },
      ],
    };
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={[]} initialTarget={target} availableTargets={[target]} />);
    fireEvent.click(screen.getByTestId("launcher-machine-picker"));
    const option = screen.getByRole("option", { name: /matt-macbook-pro/ });
    // The stub translator returns keys, so the assertion is that the missing
    // agent reaches the row through the "missing" phrasing rather than as a
    // bare label sitting beside the ready count.
    expect(option).toHaveTextContent("launcher.capabilityMissing");
    expect(option).not.toHaveTextContent(/agents ready · Claude Code/);
  });

  it("explains unknown remote readiness without offering a launch the server will refuse", async () => {
    const target: TerminalTarget = { id: "bridge-node:mac", kind: "bridge-node", label: "Remote", available: true, readiness: [
      { key: "capability:codex", label: "Codex", passed: false, state: "unknown", detail: "Node runtime is missing" },
    ] };
    renderAgentMode(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} initialTarget={target} availableTargets={[target]} />);
    expect(screen.getByTestId("launcher-agent-codex")).toBeDisabled();
    expect(screen.getByTestId("launcher-submit-selection")).toBeDisabled();
    openDetail("codex");
    expect(await screen.findByTestId("launcher-action-codex")).toHaveTextContent("Node runtime is missing");
    expect(onLaunch).not.toHaveBeenCalled();
  });

  // The regression that made a blocked card indistinguishable from a launch
  // button: it changed its subtitle and silently reassigned its own onClick.
  it("offers an Install control on a missing agent instead of a launch", () => {
    const onInstallCapability = vi.fn().mockResolvedValue({ status: "installed" });
    const target: TerminalTarget = {
      id: "local", kind: "local", label: "This machine", available: true, state: "dispatchable",
      readiness: [
        { key: "capability:codex", label: "Codex", passed: false, detail: "codex is not installed", state: "missing" },
      ],
    };
    renderAgentMode(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} availableTargets={[target]} onInstallCapability={onInstallCapability} />);
    expect(screen.queryByTestId("launcher-agent-codex")).not.toBeInTheDocument();
    openDetail("codex");
    fireEvent.click(screen.getByTestId("launcher-agent-install-codex"));
    expect(onInstallCapability).toHaveBeenCalledWith("codex", expect.objectContaining({ id: "local" }));
    expect(onLaunch).not.toHaveBeenCalled();
  });

  // An agent with no build for this machine must not offer an install the
  // relay has already said it will refuse.
  it("offers no install for an agent this machine cannot run", () => {
    const target: TerminalTarget = {
      id: "local", kind: "local", label: "This machine", available: true, state: "dispatchable",
      readiness: [
        { key: "capability:agy", label: "Antigravity", passed: false, detail: "No darwin/arm64 build published", state: "not_applicable" },
      ],
    };
    renderAgentMode(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={[]} availableTargets={[target]} onInstallCapability={vi.fn().mockResolvedValue({ status: "installed" })} />);
    openDetail("agy");
    expect(screen.queryByTestId("launcher-agent-install-agy")).not.toBeInTheDocument();
    expect(screen.getByTestId("launcher-action-agy")).toHaveTextContent("No darwin/arm64 build published");
  });

  // A failing installer must say so on the card that started it, and must not
  // surface as an unhandled rejection. The card also stays in the state the
  // machine reports rather than stranding on a spinner.
  it("reports a failed install on the card that started it", async () => {
    const onInstallCapability = vi.fn().mockRejectedValue(new Error("relay refused"));
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => undefined);
    const target = missingCodexTarget();
    renderAgentMode(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} availableTargets={[target]} onInstallCapability={onInstallCapability} />);

    openDetail("codex");
    fireEvent.click(screen.getByTestId("launcher-agent-install-codex"));
    await waitFor(() => {
      expect(screen.getByTestId("launcher-agent-install-failed-codex")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId("launcher-management-back"));
    expect(screen.getByTestId("launcher-agent-card-codex")).toHaveAttribute("data-agent-state", "missing");
    consoleError.mockRestore();
  });

  // The exact contradiction an operator photographed: the card announced
  // "Installed" while still offering "Install", because the confirmation and
  // the install button were independent conditionals over the same card.
  it("never shows an install offer and an install outcome at the same time", async () => {
    const target = missingCodexTarget();
    renderAgentMode(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} availableTargets={[target]}
      onInstallCapability={vi.fn().mockResolvedValue({ status: "installed" })} />);

    openDetail("codex");
    fireEvent.click(screen.getByTestId("launcher-agent-install-codex"));
    await waitFor(() => {
      expect(screen.getByTestId("launcher-agent-installed-codex")).toBeInTheDocument();
    });
    expect(screen.queryByTestId("launcher-agent-install-codex")).not.toBeInTheDocument();
  });

  // An install the machine never confirmed is neither a success nor a failure,
  // and saying "Installed" over a machine that still does not have the agent
  // is the thing that sent an operator looking for a bug that was in the UI.
  it("reports an unconfirmed install as unknown rather than as installed", async () => {
    const target = missingCodexTarget();
    renderAgentMode(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} availableTargets={[target]}
      onInstallCapability={vi.fn().mockResolvedValue({ status: "unconfirmed", message: "the machine has not reported codex yet" })} />);

    openDetail("codex");
    fireEvent.click(screen.getByTestId("launcher-agent-install-codex"));
    await screen.findByTestId("launcher-agent-install-unconfirmed-codex");
    expect(screen.getByTestId("launcher-agent-install-reason-codex")).toHaveTextContent("the machine has not reported codex yet");
    expect(screen.queryByTestId("launcher-agent-installed-codex")).not.toBeInTheDocument();
    expect(screen.queryByTestId("launcher-agent-install-codex")).not.toBeInTheDocument();
  });

  // The catalog can still offer an install the machine will refuse — no build
  // published for its platform. Dropping the rail entirely would take the
  // button away and say nothing about why.
  it("says so when the machine refuses an install it was offered", async () => {
    const target = missingCodexTarget();
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} availableTargets={[target]}
      onInstallCapability={vi.fn().mockResolvedValue({ status: "not_applicable", message: "no darwin/arm64 build published" })} />);
    fireEvent.click(screen.getByTestId("launcher-mode-agent"));

    openDetail("codex");
    fireEvent.click(screen.getByTestId("launcher-agent-install-codex"));
    const rail = await screen.findByTestId("launcher-agent-install-unsupported-codex");
    expect(rail).toHaveTextContent("no darwin/arm64 build published");
    expect(screen.queryByTestId("launcher-agent-install-codex")).not.toBeInTheDocument();
    expect(screen.queryByTestId("launcher-agent-installed-codex")).not.toBeInTheDocument();
  });

  // The unconfirmed rail is a retry, not a dead end: pressing it runs the
  // installer again rather than leaving the operator with nothing to do.
  it("retries the install from the unconfirmed rail", async () => {
    const onInstallCapability = vi.fn().mockResolvedValue({ status: "unconfirmed" });
    const target = missingCodexTarget();
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} availableTargets={[target]} onInstallCapability={onInstallCapability} />);
    fireEvent.click(screen.getByTestId("launcher-mode-agent"));

    openDetail("codex");
    fireEvent.click(screen.getByTestId("launcher-agent-install-codex"));
    fireEvent.click(await screen.findByTestId("launcher-agent-install-unconfirmed-codex"));
    await waitFor(() => {
      expect(onInstallCapability).toHaveBeenCalledTimes(2);
    });
  });

  // Reorder persists to the profile the effective list came from. Reading that
  // profile's id off GetEffective is what keeps scope priority a server rule
  // rather than a second copy in the client.
  it("writes a reordered agent list back to the profile it came from", async () => {
    shortcutsClient.getEffective.mockResolvedValueOnce({
      profileId: "ws-1",
      scope: "workspace",
      profileName: "Dev Shortcuts",
      shortcuts: [
        { $typeName: "vrooli.web_console.v1.shortcuts.Shortcut", label: "Claude Code", command: "claude", description: "", agentId: "claude" },
        { $typeName: "vrooli.web_console.v1.shortcuts.Shortcut", label: "Codex", command: "codex --yolo", description: "", agentId: "codex" },
      ],
    });
    shortcutsClient.upsertProfile.mockResolvedValueOnce({});
    const target: TerminalTarget = {
      id: "local", kind: "local", label: "This machine", available: true, state: "dispatchable",
      readiness: [
        { key: "capability:claude", label: "Claude Code", passed: true, detail: "", state: "ready" },
        { key: "capability:codex", label: "Codex", passed: true, detail: "", state: "ready" },
      ],
    };
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} availableTargets={[target]} />);
    fireEvent.click(screen.getByTestId("launcher-mode-agent"));

    await waitFor(() => { expect(shortcutsClient.getAgentPreferences).toHaveBeenCalled(); });
    fireEvent.click(screen.getByTestId("launcher-manage-agents"));
    await waitFor(() => { expect(screen.getByTestId("launcher-agent-grip-claude")).toBeInTheDocument(); });
    fireEvent.keyDown(screen.getByTestId("launcher-agent-grip-claude"), { key: "ArrowDown", altKey: true });
    await waitFor(() => {
      expect(shortcutsClient.updateAgentPreferences).toHaveBeenCalledWith(expect.objectContaining({
        profileId: "ws-1", profileInstance: "instance-1", expectedRevision: 0n,
        orderedAgentIds: ["codex", "claude"], hiddenAgentIds: [],
      }));
    });
  });

  it("hides and restores an agent without changing the launch command", async () => {
    shortcutsClient.getEffective.mockResolvedValueOnce({ profileId: "ws-1", scope: "workspace", profileName: "Dev Shortcuts", shortcuts: [
      { $typeName: "vrooli.web_console.v1.shortcuts.Shortcut", label: "Codex", command: "codex --yolo", description: "", agentId: "codex" },
    ] });
    shortcutsClient.updateAgentPreferences.mockResolvedValueOnce({ preferences: { profileId: "ws-1", profileInstance: "instance-1", revision: 1n, orderedAgentIds: [], hiddenAgentIds: ["codex"], orderMode: "custom", lastLaunchedAt: {} } } as never);
    const target: TerminalTarget = { id: "local", kind: "local", label: "This machine", available: true, readiness: [{ key: "capability:codex", label: "Codex", passed: true, detail: "", state: "ready" }] };
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} availableTargets={[target]} />);
    fireEvent.click(screen.getByTestId("launcher-mode-agent"));
    await waitFor(() => { expect(shortcutsClient.getAgentPreferences).toHaveBeenCalled(); });
    fireEvent.click(screen.getByTestId("launcher-manage-agents"));
    const hide = await screen.findByTestId("launcher-agent-visibility-codex");
    fireEvent.click(hide);
    await waitFor(() => { expect(shortcutsClient.updateAgentPreferences).toHaveBeenCalledWith(expect.objectContaining({ hiddenAgentIds: ["codex"] })); });
    await waitFor(() => { expect(hide).not.toBeChecked(); });
    expect(shortcutsClient.upsertProfile).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("launcher-management-back"));
    expect(screen.queryByTestId("launcher-agent-codex")).not.toBeInTheDocument();
    expect(screen.getByTestId("launcher-no-agents")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("launcher-manage-agents"));
    expect(screen.getByTestId("launcher-agent-visibility-codex")).not.toBeChecked();
  });

  // Reorder edits the operator's stored profile. A caller that supplies the
  // list itself owns that list, so the dialog must not offer to rewrite it.
  it("offers no reorder when the shortcut list comes from a prop", () => {
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />);
    fireEvent.click(screen.getByTestId("launcher-mode-agent"));
    fireEvent.click(screen.getByTestId("launcher-manage-agents"));
    expect(screen.queryByTestId("launcher-agent-grip-claude")).not.toBeInTheDocument();
  });

  it("renders no full-width target card", () => {
    render(<TerminalLauncher open onClose={onClose} onLaunch={onLaunch} shortcuts={testShortcuts} />);
    expect(screen.queryByTestId("launcher-target-card-local")).not.toBeInTheDocument();
    expect(screen.getByTestId("launcher-machine-picker")).toBeInTheDocument();
  });
});
