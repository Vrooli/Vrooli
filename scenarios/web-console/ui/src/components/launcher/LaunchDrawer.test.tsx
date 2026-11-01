// DOC: docs/reference/configuration.md#launcher-shortcuts
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { asMockedClient, renderWithProviders as render, setDesktopViewport, setMobileViewport } from "../../test-utils";
import { strings } from "../../consts/strings";
import type { ShortcutEntry } from "../../consts/shortcuts";
import type { TerminalTarget } from "../../api/targets";
import TerminalLauncher from "../TerminalLauncher";
import { resetHarnessCacheForTests } from "./useHarnesses";

vi.mock("../../api/capabilities", async () => {
  const actual = await vi.importActual<typeof import("../../api/capabilities")>("../../api/capabilities");
  return { ...actual, capabilitiesClient: { agentUpdate: vi.fn(), agentUsage: vi.fn(), listHarnesses: vi.fn(), runAgentUpdateAction: vi.fn() } };
});
vi.mock("../../api/shortcuts", () => ({ shortcutsClient: { getEffective: vi.fn(), getAgentPreferences: vi.fn(), updateAgentPreferences: vi.fn() } }));
vi.mock("../../api/modelSelection", () => ({ modelSelectionClient: { getRoleModelChoices: vi.fn() } }));
vi.mock("../../api/usageMode", () => ({ getUsageMode: vi.fn(), startUsageMode: vi.fn(), endUsageMode: vi.fn() }));

import { capabilitiesClient as rawCapabilities } from "../../api/capabilities";
import { modelSelectionClient as rawModels } from "../../api/modelSelection";
import { shortcutsClient as rawShortcuts } from "../../api/shortcuts";
import { getUsageMode as rawGetUsageMode } from "../../api/usageMode";

const capabilities = asMockedClient(rawCapabilities);
const models = asMockedClient(rawModels);
const shortcutsApi = asMockedClient(rawShortcuts);
const getUsageMode = vi.mocked(rawGetUsageMode);

const AGENTS: ShortcutEntry[] = [
  { agentId: "claude", label: "Claude Code", command: "claude" },
  { agentId: "codex", label: "Codex", command: "codex --yolo" },
  { agentId: "opencode", label: "OpenCode", command: "opencode" },
  { label: "htop", command: "htop" },
];
const fact = (id: string, label: string) => ({ key: `capability:${id}`, label, passed: true, detail: "", state: "ready" as const });
const target = (over: Partial<TerminalTarget> = {}): TerminalTarget => ({
  id: "local", kind: "local", label: "This machine", available: true, state: "dispatchable",
  readiness: [fact("claude", "Claude Code"), fact("codex", "Codex"), fact("opencode", "OpenCode")], ...over,
});
const harness = (agentId: string, over: Record<string, unknown> = {}) => ({ agentId, resource: agentId, displayName: agentId, icon: agentId, usageWindows: [], supportsContextLimit: false, supportsUsageMode: false, supportsModelSelection: true, supportsProviderSelection: false, ...over });
const choice = (model: string, over: Record<string, unknown> = {}) => ({ displayName: model, model, source: "sub", sourceLabel: "Subscription", provider: "p", launchModel: `m/${model}`, effort: "", isDefault: false, contextWindowTokens: 0n, contextLimitTokens: [], metered: false, ...over });
const policy = (over: Record<string, unknown> = {}) => ({ operatorCommand: "", commandLiteral: false, policyStatus: "ready", availability: "unknown", runner: "", role: "", modelFlag: "-m", provenance: "", observedAt: "", detail: "", policyDigest: "", catalogStale: false, choices: [], ...over });
const update = (agentId: string, over: Record<string, unknown> = {}) => ({ targetId: "local", agentId, resource: agentId, installed: "1.0.0", pinned: "0.9.0", latest: "1.0.0", status: "up-to-date", checkedAt: new Date().toISOString(), sourceKind: "", sourceId: "", probeVerified: true, note: "", state: "up_to_date", primaryAction: "", operatorCommand: "", reason: "", checks: [], installKind: "managed", rollbackVersion: "", newerThanReviewed: false, ...over });
const usage = (agentId: string, windows: { windowMinutes: number; usedPercent: number }[]) => ({ targetId: "local", agentId, match: "likely", reason: "", source: "provider", observedAt: new Date().toISOString(), windows: windows.map((window) => ({ pool: "primary", resetsAt: new Date(Date.now() + 86_400_000).toISOString(), observedAt: new Date().toISOString(), ...window, windowMinutes: BigInt(window.windowMinutes) })) });
const prefs = (over: Record<string, unknown> = {}) => ({ profileId: "ws", profileInstance: "i", revision: 1n, orderedAgentIds: [], hiddenAgentIds: [], orderMode: "custom", lastLaunchedAt: {}, ...over });

const onLaunch = vi.fn();
const drawer = (extra: Record<string, unknown> = {}) => render(<TerminalLauncher open onClose={vi.fn()} onLaunch={onLaunch} shortcuts={AGENTS} availableTargets={[target()]} {...extra} />);
/** Without a `shortcuts` prop the drawer reads the profile and offers preferences. */
const profileDrawer = async (extra: Record<string, unknown> = {}) => {
  shortcutsApi.getEffective.mockResolvedValue({ profileId: "ws", shortcuts: AGENTS.map((entry) => ({ description: "", agentId: "", ...entry })) } as never);
  render(<TerminalLauncher open onClose={vi.fn()} onLaunch={onLaunch} availableTargets={[target()]} {...extra} />);
  await waitFor(() => { expect(shortcutsApi.getAgentPreferences).toHaveBeenCalled(); });
  await act(async () => { await Promise.resolve(); });
};
const launchButton = () => screen.getByTestId("launcher-submit-selection");
const openManage = () => { fireEvent.click(screen.getByTestId("launcher-manage-agents")); };
const openDetail = (key: string) => { openManage(); fireEvent.click(screen.getByTestId(`launcher-manage-open-${key}`)); };
const nativeSelect = (testId: string) => screen.getByTestId(testId).closest("[data-rcl-select-field]")?.querySelector("select") as HTMLSelectElement;
const chooseLimit = (value: string) => { fireEvent.change(nativeSelect("launcher-context-limit-select"), { target: { value } }); };
const selected = (name: string) => screen.getByTestId(`launcher-agent-${name}`).getAttribute("aria-checked");

beforeEach(() => {
  vi.clearAllMocks();
  window.localStorage.clear();
  resetHarnessCacheForTests();
  setMobileViewport();
  capabilities.agentUpdate.mockImplementation(((request: { agentId: string }) => Promise.resolve(update(request.agentId))) as never);
  capabilities.agentUsage.mockImplementation(((request: { agentId: string }) => Promise.resolve(usage(request.agentId, []))) as never);
  capabilities.listHarnesses.mockResolvedValue({ targetId: "local", harnesses: ["claude", "codex", "opencode"].map((id) => harness(id)) } as never);
  models.getRoleModelChoices.mockResolvedValue(policy({ policyStatus: "unsupported", modelFlag: "" }) as never);
  shortcutsApi.getAgentPreferences.mockResolvedValue({ preferences: prefs() } as never);
  shortcutsApi.updateAgentPreferences.mockResolvedValue({ preferences: prefs({ revision: 2n }) } as never);
  getUsageMode.mockResolvedValue({ modes: [], agents: [], canControl: true });
});

describe("type first", () => {
  it("orders the drawer machine, then type, then the choice, then settings, then launch", async () => {
    drawer();
    await screen.findByTestId("launcher-settings");
    const panel = screen.getByTestId("terminal-launcher");
    const sequence = ["launcher-machine-picker", "launcher-mode-agent", "launcher-agent-grid", "launcher-settings", "launcher-submit-selection"].map((id) => within(panel).getByTestId(id));
    for (let index = 1; index < sequence.length; index += 1) {
      const earlier = sequence[index - 1] as HTMLElement;
      expect(earlier.compareDocumentPosition(sequence[index] as HTMLElement) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    }
    expect(screen.getByText(strings.terminalLauncher.newSession)).toBeInTheDocument();
  });

  it("offers Agent, Shell and Group, and Group only when a caller can create one", () => {
    const { rerender } = drawer();
    expect(screen.queryByTestId("launcher-mode-group")).toBeNull();
    rerender(<TerminalLauncher open onClose={vi.fn()} onLaunch={onLaunch} shortcuts={AGENTS} availableTargets={[target()]} onCreateGroupFromRoles={vi.fn()} />);
    expect(screen.getByTestId("launcher-mode-group")).toBeInTheDocument();
  });

  it("renders no agent control for a shell", async () => {
    getUsageMode.mockResolvedValue({ modes: [], agents: [{ agentId: "claude", label: "Claude Code", runner: "claude", lowerModel: "a", usualModel: "b" }], canControl: true });
    drawer();
    await screen.findByTestId("launcher-settings");
    fireEvent.click(screen.getByTestId("launcher-mode-shell"));
    for (const id of ["launcher-settings", "launcher-agent-grid", "launcher-model-selection", "launcher-context-limit", "launcher-save-usage", "launcher-manage-agents"]) {
      expect(screen.queryByTestId(id)).toBeNull();
    }
    expect(screen.getByTestId("launcher-shortcut-htop")).toBeInTheDocument();
    expect(screen.getByTestId("launcher-summary")).toHaveTextContent(strings.terminalLauncher.summaryNoAgent);
  });

  it("holds the destination for a group in the group's own flow, with its launch in the footer", async () => {
    drawer({ onCreateGroupFromRoles: vi.fn() });
    fireEvent.click(screen.getByTestId("launcher-mode-group"));
    const create = await screen.findByTestId("launcher-create-group");
    expect(create).toHaveTextContent(strings.launcher.startGroup);
    expect(screen.queryByTestId("launcher-submit-selection")).toBeNull();
  });

  it("closes the machine menu on Escape without closing the drawer", () => {
    const onClose = vi.fn();
    drawer({ onClose });
    fireEvent.click(screen.getByTestId("launcher-machine-picker"));
    const list = screen.getByTestId("launcher-machine-list");
    fireEvent.keyDown(list, { key: "Escape" });
    expect(screen.queryByTestId("launcher-machine-list")).toBeNull();
    expect(onClose).not.toHaveBeenCalled();
  });

  it("composes settings beside the choice on a desktop, from the same data", async () => {
    setDesktopViewport();
    drawer();
    await screen.findByTestId("launcher-settings");
    expect(screen.getByTestId("launcher-desktop")).toContainElement(screen.getByTestId("launcher-settings"));
    expect(screen.queryByTestId("launcher-phone")).toBeNull();
  });

  it("keeps Compact idle sessions out of the launch flow, in the header's overflow menu", async () => {
    drawer();
    await screen.findByTestId("launcher-settings");
    expect(screen.queryByText(strings.launcher.compactIdleOpen)).toBeNull();
    fireEvent.click(screen.getByTestId("launcher-overflow"));
    expect(await screen.findByRole("menuitem", { name: strings.launcher.compactIdleOpen })).toBeInTheDocument();
  });
});

describe("a harness is always selected", () => {
  it("has no way to clear the selection, and starts on the first visible harness", () => {
    drawer();
    expect(selected("claude-code")).toBe("true");
    expect(launchButton()).toHaveTextContent(strings.terminalLauncher.launchSelectedAgent);
    expect(screen.queryByTestId("launcher-clear-selection")).toBeNull();
  });

  it("starts on the last launched harness once preferences arrive", async () => {
    shortcutsApi.getAgentPreferences.mockResolvedValue({ preferences: prefs({ lastLaunchedAt: { codex: "2026-10-30T00:00:00Z" } }) } as never);
    await profileDrawer();
    await waitFor(() => { expect(selected("codex")).toBe("true"); });
  });

  it("falls back when the selected harness becomes hidden", async () => {
    shortcutsApi.updateAgentPreferences.mockResolvedValue({ preferences: prefs({ hiddenAgentIds: ["codex"], revision: 2n }) } as never);
    await profileDrawer();
    fireEvent.click(screen.getByTestId("launcher-agent-codex"));
    expect(selected("codex")).toBe("true");
    openManage();
    fireEvent.click(await screen.findByTestId("launcher-agent-visibility-codex"));
    await waitFor(() => { expect(screen.queryByTestId("launcher-agent-codex")).toBeNull(); });
    expect(selected("claude-code")).toBe("true");
  });

  it("says so and blocks launch when no harness is visible", async () => {
    shortcutsApi.getAgentPreferences.mockResolvedValue({ preferences: prefs({ hiddenAgentIds: ["claude", "codex", "opencode"] }) } as never);
    await profileDrawer();
    await screen.findByTestId("launcher-no-agents");
    expect(launchButton()).toBeDisabled();
    expect(screen.queryByTestId("launcher-settings")).toBeNull();
  });

  it("moves selection with the arrow keys, skipping nothing it can select", () => {
    drawer();
    const claude = screen.getByTestId("launcher-agent-claude-code");
    claude.focus();
    fireEvent.keyDown(claude, { key: "ArrowRight" });
    expect(selected("codex")).toBe("true");
    expect(screen.getByTestId("launcher-agent-codex")).toHaveFocus();
    fireEvent.keyDown(screen.getByTestId("launcher-agent-codex"), { key: "End" });
    expect(selected("opencode")).toBe("true");
    fireEvent.keyDown(screen.getByTestId("launcher-agent-opencode"), { key: "ArrowRight" });
    expect(selected("claude-code")).toBe("true");
  });
});

describe("cards", () => {
  const counts = () => screen.getAllByTestId("launcher-card-usage").map((block) => block.children.length);

  it("reserve the same rows on every card from the largest declared count, before any reading", async () => {
    capabilities.agentUsage.mockImplementation((() => new Promise(() => undefined)) as never);
    capabilities.listHarnesses.mockResolvedValue({ targetId: "local", harnesses: [harness("claude", { usageWindows: [{ id: "5h", windowMinutes: 300n }, { id: "w", windowMinutes: 10080n }] }), harness("codex", { usageWindows: [{ id: "w", windowMinutes: 10080n }] }), harness("opencode")] } as never);
    drawer();
    await waitFor(() => { expect(counts()).toEqual([2, 2, 2]); });
    // Placeholders follow each harness's own declaration, and only it.
    const blocks = screen.getAllByTestId("launcher-card-usage");
    expect(blocks.map((block) => block.querySelectorAll('[data-usage-row="placeholder"]').length)).toEqual([2, 1, 0]);
  });

  it("stay the same shape when loaded, and when a window disappears", async () => {
    capabilities.agentUsage.mockImplementation(((request: { agentId: string }) => Promise.resolve(usage(request.agentId, request.agentId === "claude" ? [{ windowMinutes: 10080, usedPercent: 87 }] : []))) as never);
    capabilities.listHarnesses.mockResolvedValue({ targetId: "local", harnesses: [harness("claude", { usageWindows: [{ id: "5h", windowMinutes: 300n }, { id: "w", windowMinutes: 10080n }] }), harness("codex"), harness("opencode")] } as never);
    drawer();
    await screen.findByRole("progressbar");
    expect(counts()).toEqual([2, 2, 2]);
    expect(screen.getAllByRole("progressbar")).toHaveLength(1);
  });

  it("hold a stable height when descriptors are unavailable, and report nothing as usage", async () => {
    capabilities.listHarnesses.mockRejectedValue(new Error("unimplemented"));
    capabilities.agentUsage.mockImplementation((() => new Promise(() => undefined)) as never);
    drawer();
    await waitFor(() => { expect(capabilities.listHarnesses).toHaveBeenCalled(); });
    expect(counts()).toEqual([2, 2, 2]);
    expect(screen.queryByRole("progressbar")).toBeNull();
  });

  it("mark a harness with an action, never a failed check, and Manage counts them", async () => {
    const byAgent: Record<string, unknown> = {
      claude: update("claude", { state: "update_ready", primaryAction: "update" }),
      codex: update("codex", { state: "check_failed", primaryAction: "retry" }),
      opencode: update("opencode", { state: "needs_setup", operatorCommand: "sudo vrooli setup --include-optional" }),
    };
    capabilities.agentUpdate.mockImplementation(((request: { agentId: string }) => Promise.resolve(byAgent[request.agentId])) as never);
    drawer();
    expect(await screen.findByTestId("launcher-agent-flag-claude")).toBeInTheDocument();
    await screen.findByTestId("launcher-agent-flag-opencode");
    expect(screen.queryByTestId("launcher-agent-flag-codex")).toBeNull();
    expect(screen.getByTestId("launcher-manage-count")).toHaveTextContent("2");
  });

  it("show no count when an old server has no state, even if it says behind", async () => {
    capabilities.agentUpdate.mockImplementation(((request: { agentId: string }) => Promise.resolve(update(request.agentId, { state: "", status: "behind", latest: "9.0.0" }))) as never);
    drawer();
    await waitFor(() => { expect(capabilities.agentUpdate).toHaveBeenCalled(); });
    await act(async () => { await Promise.resolve(); });
    expect(screen.queryByTestId("launcher-manage-count")).toBeNull();
  });
});

describe("settings", () => {
  const limited = choice("big", { isDefault: true, contextWindowTokens: 500_000n, contextLimitTokens: [300_000n, 150_000n] });
  const small = choice("small", { contextWindowTokens: 200_000n, contextLimitTokens: [150_000n] });
  const unknownWindow = choice("mystery");
  const supportsLimit = () => capabilities.listHarnesses.mockResolvedValue({ targetId: "local", harnesses: [harness("claude", { supportsContextLimit: true }), harness("codex", { supportsContextLimit: true }), harness("opencode")] } as never);

  it("offers Compact at from the selected model, states the window, and applies it to that launch", async () => {
    supportsLimit();
    models.getRoleModelChoices.mockResolvedValue(policy({ choices: [limited, small] }) as never);
    drawer();
    await screen.findByTestId("launcher-context-limit-select");
    chooseLimit("300000");
    expect(screen.getByTestId("launcher-context-window")).toHaveTextContent(strings.launcher.contextWindow);
    expect(screen.getByTestId("launcher-context-gauge")).toHaveAttribute("data-gauge-percent", "60");
    expect(screen.getByTestId("launcher-summary")).toHaveTextContent(strings.launcher.summaryCompacts);
    fireEvent.click(launchButton());
    expect(onLaunch).toHaveBeenLastCalledWith(expect.objectContaining({ command: "claude", agentLaunch: { agent_id: "claude", model: undefined, provider: undefined, context_limit_tokens: 300_000 } }));
    // A smaller model never offers more than its window, and drops a limit it cannot honour.
    fireEvent.change(nativeSelect("launcher-model-choice"), { target: { value: JSON.stringify(["sub", "small"]) } });
    expect([...nativeSelect("launcher-context-limit-select").options].map((option) => option.value)).toEqual(["default", "150000"]);
    expect(nativeSelect("launcher-context-limit-select")).toHaveValue("default");
    fireEvent.click(launchButton());
    expect(onLaunch).toHaveBeenLastCalledWith(expect.objectContaining({ command: "claude", agentLaunch: expect.objectContaining({ model: "small", context_limit_tokens: undefined }) }));
  });

  it("states an unknown window rather than guessing", async () => {
    supportsLimit();
    models.getRoleModelChoices.mockResolvedValue(policy({ choices: [{ ...unknownWindow, isDefault: true }] }) as never);
    drawer();
    expect(await screen.findByTestId("launcher-context-window")).toHaveTextContent(strings.launcher.contextWindowUnknown);
    expect(screen.queryByTestId("launcher-context-gauge")).toBeNull();
  });

  it("hides Compact at for a harness that does not support it", async () => {
    models.getRoleModelChoices.mockResolvedValue(policy({ choices: [limited] }) as never);
    drawer();
    await screen.findByTestId("launcher-model-choice");
    await waitFor(() => { expect(capabilities.listHarnesses).toHaveBeenCalled(); });
    await act(async () => { await Promise.resolve(); });
    expect(screen.queryByTestId("launcher-context-limit")).toBeNull();
  });

  it("states that settings do not apply to a command of the operator's own, and sends none", async () => {
    supportsLimit();
    models.getRoleModelChoices.mockResolvedValue(policy({ commandLiteral: true, choices: [limited, small] }) as never);
    drawer();
    await screen.findByTestId("launcher-model-value");
    expect(screen.getByTestId("launcher-model-selection")).toHaveTextContent(strings.launcher.settingsDontApply);
    expect(screen.queryByTestId("launcher-model-choice")).toBeNull();
    expect(screen.queryByTestId("launcher-context-limit")).toBeNull();
    fireEvent.click(launchButton());
    expect(onLaunch).toHaveBeenLastCalledWith(expect.objectContaining({ command: "claude", agentLaunch: undefined }));
  });

  it("removes the gauge at Default, where it says nothing, and shows it for a chosen limit", async () => {
    supportsLimit();
    models.getRoleModelChoices.mockResolvedValue(policy({ choices: [limited] }) as never);
    drawer();
    await screen.findByTestId("launcher-context-limit-select");
    expect(screen.queryByTestId("launcher-context-gauge")).toBeNull();
    chooseLimit("150000");
    expect(screen.getByTestId("launcher-context-gauge")).toHaveAttribute("data-gauge-percent", "30");
  });

  it("refreshes a stale list from its notice, reads it again, and shows a failure in the server's words", async () => {
    models.getRoleModelChoices.mockResolvedValue(policy({ catalogStale: true, policyStatus: "stale", choices: [limited] }) as never);
    capabilities.runAgentUpdateAction.mockResolvedValueOnce({ success: false, message: "The installed runner reported no models." } as never);
    drawer();
    fireEvent.click(await screen.findByTestId("launcher-catalog-refresh"));
    expect(capabilities.runAgentUpdateAction).toHaveBeenCalledWith({ targetId: "local", agentId: "claude", action: "refresh_catalog" }, expect.anything());
    expect(await screen.findByTestId("launcher-catalog-refresh-error")).toHaveTextContent("The installed runner reported no models.");
    const reads = models.getRoleModelChoices.mock.calls.length;
    capabilities.runAgentUpdateAction.mockResolvedValueOnce({ success: true, message: "Refreshed." } as never);
    models.getRoleModelChoices.mockResolvedValue(policy({ choices: [limited] }) as never);
    fireEvent.click(await screen.findByTestId("launcher-catalog-refresh"));
    await waitFor(() => { expect(models.getRoleModelChoices.mock.calls.length).toBeGreaterThan(reads); });
    await waitFor(() => { expect(screen.queryByTestId("launcher-catalog-stale")).toBeNull(); });
  });

  it("offers the operator's command where a list cannot refresh itself", async () => {
    models.getRoleModelChoices.mockResolvedValue(policy({ catalogStale: true, operatorCommand: "$EDITOR resources/opencode/model-policy.json", provenance: "hand", choices: [limited] }) as never);
    drawer();
    expect(await screen.findByTestId("launcher-catalog-fix")).toBeInTheDocument();
    expect(screen.queryByTestId("launcher-catalog-refresh")).toBeNull();
    fireEvent.click(screen.getByTestId("launcher-catalog-fix"));
    expect(await screen.findByTestId("launcher-operator-command")).toHaveTextContent("$EDITOR resources/opencode/model-policy.json");
  });

  it("launches from Enter on the drawer itself, not from a focused control", async () => {
    drawer();
    const root = await screen.findByTestId("launcher-phone");
    fireEvent.keyDown(root, { key: "Enter" });
    expect(onLaunch).toHaveBeenCalledTimes(1);
    fireEvent.keyDown(screen.getByTestId("launcher-manage-agents"), { key: "Enter" });
    expect(onLaunch).toHaveBeenCalledTimes(1);
  });

  it("states an active mode's own limit, and that an explicit one beats it", async () => {
    supportsLimit();
    getUsageMode.mockResolvedValue({ modes: [{ id: "um", label: "", runner: "claude", startedAt: "", untilKind: "off", kind: "save" }], agents: [{ agentId: "claude", label: "Claude Code", runner: "claude", lowerModel: "big", usualModel: "big", model: "big", modeId: "um", contextLimitTokens: 150_000 }], canControl: true });
    models.getRoleModelChoices.mockResolvedValue(policy({ choices: [limited] }) as never);
    drawer();
    await waitFor(() => { expect(screen.getByTestId("launcher-context-hint")).toHaveTextContent(strings.launcher.contextLimitFromMode); });
    chooseLimit("300000");
    expect(screen.getByTestId("launcher-context-hint")).toHaveTextContent(strings.launcher.contextLimitBeatsMode);
  });

  it("shows Provider only with descriptor support and more than one source", async () => {
    const sources = [choice("a", { source: "one", sourceLabel: "One", isDefault: true }), choice("b", { source: "two", sourceLabel: "Two" })];
    models.getRoleModelChoices.mockResolvedValue(policy({ choices: sources }) as never);
    capabilities.listHarnesses.mockResolvedValue({ targetId: "local", harnesses: [harness("claude", { supportsProviderSelection: true }), harness("codex"), harness("opencode", { supportsProviderSelection: true })] } as never);
    drawer();
    await screen.findByTestId("launcher-model-source");
    fireEvent.click(screen.getByTestId("launcher-agent-codex"));
    await screen.findByTestId("launcher-model-choice");
    expect(screen.queryByTestId("launcher-model-source")).toBeNull();
    models.getRoleModelChoices.mockResolvedValue(policy({ choices: [sources[0]] }) as never);
    fireEvent.click(screen.getByTestId("launcher-agent-opencode"));
    await screen.findByTestId("launcher-model-choice");
    expect(screen.queryByTestId("launcher-model-source")).toBeNull();
  });


  it("shows Provider before Model and remembers the last choice per harness", async () => {
    const choices = [choice("a", { source: "one", sourceLabel: "One", isDefault: true }), choice("b", { source: "two", sourceLabel: "Two" })];
    models.getRoleModelChoices.mockResolvedValue(policy({ choices }) as never);
    capabilities.listHarnesses.mockResolvedValue({ targetId: "local", harnesses: [harness("claude", { supportsProviderSelection: true })] } as never);
    const { rerender } = drawer();

    const provider = await screen.findByTestId("launcher-model-source");
    const model = screen.getByTestId("launcher-model-selection");
    expect(provider.compareDocumentPosition(model) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();

    fireEvent.change(nativeSelect("launcher-model-source"), { target: { value: "two" } });
    expect(nativeSelect("launcher-model-choice")).toHaveValue(JSON.stringify(["two", "b"]));

    rerender(<TerminalLauncher open={false} onClose={vi.fn()} onLaunch={onLaunch} shortcuts={AGENTS} availableTargets={[target()]} />);
    rerender(<TerminalLauncher open onClose={vi.fn()} onLaunch={onLaunch} shortcuts={AGENTS} availableTargets={[target()]} />);
    await screen.findByTestId("launcher-model-choice");
    expect(nativeSelect("launcher-model-source")).toHaveValue("two");
    expect(nativeSelect("launcher-model-choice")).toHaveValue(JSON.stringify(["two", "b"]));
  });

  it("warns on a stale list, keeps choices selectable, blocks metered ones, and links to the detail", async () => {
    models.getRoleModelChoices.mockResolvedValue(policy({ catalogStale: true, observedAt: new Date(Date.now() - 50 * 86_400_000).toISOString(), provenance: "pinned catalog", detail: "Refresh this list.", choices: [choice("free", { isDefault: true }), choice("paid", { metered: true })] }) as never);
    drawer();
    expect(await screen.findByTestId("launcher-catalog-stale")).toHaveTextContent(strings.terminalLauncher.modelListAge);
    const native = nativeSelect("launcher-model-choice");
    const option = (name: string) => [...native.options].find((item) => item.textContent === name);
    expect(option(strings.terminalLauncher.modelIsDefault)?.disabled).toBe(false);
    expect(option("paid")?.disabled).toBe(true);
    openDetail("claude");
    expect(await screen.findByTestId("launcher-model-list-detail")).toHaveTextContent("Refresh this list.");
  });

  it("closes only an open list on Escape, leaving the drawer open", async () => {
    const onClose = vi.fn();
    models.getRoleModelChoices.mockResolvedValue(policy({ choices: [limited, small] }) as never);
    drawer({ onClose });
    const trigger = await screen.findByTestId("launcher-model-choice");
    fireEvent.click(trigger);
    expect(trigger).toHaveAttribute("aria-expanded", "true");
    fireEvent.keyDown(trigger, { key: "Escape" });
    expect(onClose).not.toHaveBeenCalled();
    expect(trigger).toHaveAttribute("aria-expanded", "false");
  });

  it("lists each model once with the default marked, and passes no override for the default", async () => {
    models.getRoleModelChoices.mockResolvedValue(policy({ choices: [limited, small] }) as never);
    drawer();
    await screen.findByTestId("launcher-model-choice");
    const native = nativeSelect("launcher-model-choice");
    expect([...native.options].map((option) => option.textContent)).toEqual([strings.terminalLauncher.modelIsDefault, "small"]);
    fireEvent.change(native, { target: { value: JSON.stringify(["sub", "small"]) } });
    fireEvent.change(native, { target: { value: JSON.stringify(["sub", "big"]) } });
    fireEvent.click(launchButton());
    expect(onLaunch).toHaveBeenLastCalledWith(expect.objectContaining({ command: "claude", agentLaunch: undefined }));
  });

  it("names the lowered model under Save usage, and choosing the usual one overrides the mode", async () => {
    getUsageMode.mockResolvedValue({ modes: [{ id: "um", label: "", runner: "claude", startedAt: "", untilKind: "off", kind: "save" }], agents: [{ agentId: "claude", label: "Claude Code", runner: "claude", lowerModel: "small", usualModel: "big", model: "small", modeId: "um" }], canControl: true });
    models.getRoleModelChoices.mockResolvedValue(policy({ choices: [limited, small] }) as never);
    drawer();
    await screen.findByTestId("launcher-mode-note-claude");
    expect(nativeSelect("launcher-model-choice")).toHaveValue(JSON.stringify(["sub", "small"]));
    expect(screen.getByTestId("launcher-summary")).toHaveTextContent("small");
    fireEvent.change(nativeSelect("launcher-model-choice"), { target: { value: JSON.stringify(["sub", "big"]) } });
    fireEvent.click(launchButton());
    expect(onLaunch).toHaveBeenLastCalledWith(expect.objectContaining({ command: "claude", agentLaunch: expect.objectContaining({ model: "big" }) }));
  });

  it("says the default stays when Save usage's lower model already is the default", async () => {
    const sonnet = choice("sonnet", { displayName: "Sonnet 5.5", isDefault: true });
    const opus = choice("opus", { displayName: "Opus 5.5" });
    capabilities.listHarnesses.mockResolvedValue({ targetId: "local", harnesses: [harness("claude", { supportsUsageMode: true }), harness("codex"), harness("opencode")] } as never);
    getUsageMode.mockResolvedValue({ modes: [], agents: [{ agentId: "claude", label: "Claude Code", runner: "claude", lowerModel: "sonnet", usualModel: "opus" }], canControl: true });
    models.getRoleModelChoices.mockResolvedValue(policy({ choices: [sonnet, opus] }) as never);
    drawer();
    const row = await screen.findByTestId("launcher-save-usage");
    await waitFor(() => { expect(row).toHaveTextContent(strings.launcher.usageModeOffHintDefaultStays); });
    expect(row).not.toHaveTextContent(strings.launcher.usageModeOffHint + " ");
  });

  it("states what it lowers when the default would change", async () => {
    const terra = choice("terra", { displayName: "Terra", isDefault: true });
    const luna = choice("luna", { displayName: "Luna" });
    capabilities.listHarnesses.mockResolvedValue({ targetId: "local", harnesses: [harness("claude", { supportsUsageMode: true }), harness("codex"), harness("opencode")] } as never);
    getUsageMode.mockResolvedValue({ modes: [], agents: [{ agentId: "claude", label: "Claude Code", runner: "claude", lowerModel: "luna", usualModel: "terra" }], canControl: true });
    models.getRoleModelChoices.mockResolvedValue(policy({ choices: [terra, luna] }) as never);
    drawer();
    const row = await screen.findByTestId("launcher-save-usage");
    await waitFor(() => { expect(row.textContent).toContain(strings.launcher.usageModeOffHint); });
    expect(row).not.toHaveTextContent(strings.launcher.usageModeOffHintDefaultStays);
  });

  it("states honestly when the model list is unknown, and still launches the default", async () => {
    models.getRoleModelChoices.mockRejectedValue(new Error("offline"));
    drawer();
    await waitFor(() => { expect(screen.getByTestId("launcher-model-selection")).toHaveTextContent(strings.terminalLauncher.modelChoicesUnknown); });
    fireEvent.click(launchButton());
    expect(onLaunch).toHaveBeenCalledWith(expect.objectContaining({ command: "claude" }));
  });
});

describe("Manage", () => {
  const withState = (agentId: string, over: Record<string, unknown>) => {
    capabilities.agentUpdate.mockImplementation(((request: { agentId: string }) => Promise.resolve(update(request.agentId, request.agentId === agentId ? over : {}))) as never);
  };
  const loaded = async () => { await waitFor(() => { expect(capabilities.agentUpdate).toHaveBeenCalled(); }); await act(async () => { await Promise.resolve(); }); };

  it.each([
    ["missing", { state: "missing", primaryAction: "install" }, "launcher-agent-install-claude", "install", strings.launcher.statusMissing],
    ["shadowed", { state: "shadowed", primaryAction: "use_managed_copy" }, "launcher-action-primary-claude", "use_managed_copy", strings.launcher.statusAttention],
    ["unlinked", { state: "unlinked", primaryAction: "link_managed_copy" }, "launcher-action-primary-claude", "link_managed_copy", strings.launcher.statusAttention],
    ["update ready", { state: "update_ready", primaryAction: "update", latest: "2.0.0" }, "launcher-action-primary-claude", "update", strings.launcher.statusUpdateReady],
  ])("offers one primary action for %s and runs it through the control plane", async (_name, state, button, action, pill) => {
    withState("claude", state);
    capabilities.runAgentUpdateAction.mockResolvedValue({ success: true, message: "Done it.", update: update("claude") } as never);
    drawer();
    await loaded();
    openDetail("claude");
    expect(screen.getByTestId("launcher-agent-status-claude")).toHaveTextContent(pill);
    fireEvent.click(screen.getByTestId(button));
    expect(capabilities.runAgentUpdateAction).toHaveBeenCalledWith({ targetId: "local", agentId: "claude", action }, expect.anything());
    expect(await screen.findByTestId("launcher-action-done")).toHaveTextContent("Done it.");
  });

  it("shows the operator's command with copy for a state only they can fix, and runs nothing", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });
    withState("claude", { state: "needs_setup", operatorCommand: "sudo vrooli setup --include-optional", reason: "Root owns this copy." });
    drawer();
    await loaded();
    openDetail("claude");
    expect(screen.getByTestId("launcher-operator-command")).toHaveTextContent("sudo vrooli setup --include-optional");
    expect(screen.queryByTestId("launcher-action-primary-claude")).toBeNull();
    fireEvent.click(screen.getByTestId("launcher-operator-command-copy"));
    await waitFor(() => { expect(writeText).toHaveBeenCalledWith("sudo vrooli setup --include-optional"); });
    await waitFor(() => { expect(screen.getByTestId("launcher-operator-command-copy")).toHaveTextContent(strings.launcher.commandCopied); });
  });

  it("has no action when up to date", async () => {
    drawer();
    await loaded();
    openDetail("claude");
    expect(screen.getByTestId("launcher-agent-status-claude")).toHaveTextContent(strings.launcher.statusUpToDate);
    expect(screen.queryByTestId("launcher-action-claude")).toBeNull();
  });

  it("offers a retry for a failed check that looks again and announces nothing", async () => {
    withState("claude", { state: "check_failed", primaryAction: "retry", reason: "Could not reach the channel." });
    capabilities.runAgentUpdateAction.mockResolvedValue({ success: true, message: "", update: update("claude") } as never);
    drawer();
    await loaded();
    openDetail("claude");
    expect(screen.getByTestId("launcher-action-claude")).toHaveTextContent("Could not reach the channel.");
    fireEvent.click(screen.getByTestId("launcher-action-primary-claude"));
    expect(capabilities.runAgentUpdateAction).toHaveBeenCalledWith({ targetId: "local", agentId: "claude", action: "retry" }, expect.anything());
    await waitFor(() => { expect(screen.getByTestId("launcher-agent-status-claude")).toHaveTextContent(strings.launcher.statusUpToDate); });
    expect(screen.queryByTestId("launcher-action-done")).toBeNull();
  });

  it("offers a command beside a retry when a failed check names what is down", async () => {
    withState("claude", { state: "check_failed", primaryAction: "retry", operatorCommand: "vrooli scenario start scenario-dependency-analyzer" });
    drawer();
    await loaded();
    openDetail("claude");
    expect(screen.getByTestId("launcher-operator-command")).toBeInTheDocument();
    expect(screen.getByTestId("launcher-action-primary-claude")).toBeInTheDocument();
  });

  it("offers no action when the server names none, as for a system package or a running action", async () => {
    withState("claude", { state: "shadowed", primaryAction: "", reason: "A system package runs first." });
    drawer();
    await loaded();
    openDetail("claude");
    expect(screen.getByTestId("launcher-action-claude")).toHaveTextContent("A system package runs first.");
    expect(screen.queryByTestId("launcher-action-primary-claude")).toBeNull();
  });

  it("disables a running action, then reports a failure with the server's words and a retry", async () => {
    withState("claude", { state: "update_ready", primaryAction: "update" });
    let finish!: (value: unknown) => void;
    capabilities.runAgentUpdateAction.mockImplementationOnce((() => new Promise((resolve) => { finish = resolve; })) as never);
    drawer();
    await loaded();
    openDetail("claude");
    const button = screen.getByTestId("launcher-action-primary-claude");
    fireEvent.click(button);
    expect(button).toBeDisabled();
    expect(button).toHaveTextContent(strings.launcher.actionUpdateRunning);
    fireEvent.click(button);
    expect(capabilities.runAgentUpdateAction).toHaveBeenCalledOnce();
    await act(async () => { finish({ success: false, message: "npm is unreachable" }); await Promise.resolve(); });
    expect(await screen.findByTestId("launcher-action-error")).toHaveTextContent("npm is unreachable");
    expect(button).toBeEnabled();
    expect(button).toHaveTextContent(strings.launcher.installRetry);
  });

  it("says plainly when the server cannot run actions yet", async () => {
    withState("claude", { state: "update_ready", primaryAction: "update" });
    const { Code, ConnectError } = await import("@connectrpc/connect");
    capabilities.runAgentUpdateAction.mockRejectedValue(new ConnectError("no", Code.Unimplemented));
    drawer();
    await loaded();
    openDetail("claude");
    fireEvent.click(screen.getByTestId("launcher-action-primary-claude"));
    expect(await screen.findByTestId("launcher-action-error")).toHaveTextContent(strings.launcher.actionUnsupported);
  });

  it("shows version facts, the reviewed-for name, checks, rollback and the newer-than-reviewed note", async () => {
    withState("claude", { installed: "2.0.0", pinned: "1.0.0", rollbackVersion: "1.9.0", newerThanReviewed: true, checks: [{ id: "a", passed: true, message: "Owned by you" }, { id: "b", passed: false, message: "Older copy behind it" }] });
    capabilities.runAgentUpdateAction.mockResolvedValue({ success: true, message: "", update: update("claude") } as never);
    drawer();
    await loaded();
    openDetail("claude");
    expect(screen.getByText(strings.launcher.versionReviewedFor)).toBeInTheDocument();
    expect(screen.getByTestId("launcher-newer-than-reviewed")).toBeInTheDocument();
    const checks = screen.getByTestId("launcher-update-checks");
    expect(within(checks).getByRole("img", { name: strings.launcher.checkPass })).toBeInTheDocument();
    expect(within(checks).getByRole("img", { name: strings.launcher.checkProblem })).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("launcher-rollback-claude"));
    expect(capabilities.runAgentUpdateAction).toHaveBeenCalledWith({ targetId: "local", agentId: "claude", action: "rollback" }, expect.anything());
  });

  it("offers no rollback or note when the server names none", async () => {
    drawer();
    await loaded();
    openDetail("claude");
    expect(screen.queryByTestId("launcher-rollback-claude")).toBeNull();
    expect(screen.queryByTestId("launcher-newer-than-reviewed")).toBeNull();
  });

  it("writes one source sentence and absolute reset times in the detail, with no estimate label on cards", async () => {
    capabilities.agentUsage.mockImplementation(((request: { agentId: string }) => Promise.resolve(usage(request.agentId, [{ windowMinutes: 10080, usedPercent: 50 }]))) as never);
    drawer();
    expect(screen.queryByText("launcher.usageEstimated")).toBeNull();
    await screen.findAllByRole("progressbar");
    openDetail("claude");
    expect(screen.getByTestId("launcher-usage-source")).toHaveTextContent(strings.launcher.usageSourceLikely);
    expect(within(screen.getByTestId("launcher-usage-claude")).getByText(strings.launcher.usageResets)).toBeInTheDocument();
  });

  it("updates every ready harness in turn from one button", async () => {
    capabilities.agentUpdate.mockImplementation(((request: { agentId: string }) => Promise.resolve(update(request.agentId, request.agentId === "opencode" ? {} : { state: "update_ready", primaryAction: "update" }))) as never);
    capabilities.runAgentUpdateAction.mockImplementation(((request: { agentId: string }) => Promise.resolve({ success: true, message: "", update: update(request.agentId) })) as never);
    drawer();
    await loaded();
    openManage();
    const button = await screen.findByTestId("launcher-update-ready");
    expect(button).toHaveTextContent(strings.launcher.updateReady);
    fireEvent.click(button);
    await waitFor(() => { expect(capabilities.runAgentUpdateAction).toHaveBeenCalledTimes(2); });
    await waitFor(() => { expect(screen.queryByTestId("launcher-update-ready")).toBeNull(); });
  });

  it("persists the order mode, reorders the rail, and hides grips in Recent", async () => {
    shortcutsApi.updateAgentPreferences.mockResolvedValue({ preferences: prefs({ orderMode: "recent", lastLaunchedAt: { opencode: "2026-10-30T00:00:00Z" }, revision: 2n }) } as never);
    await profileDrawer();
    openManage();
    expect(await screen.findByTestId("launcher-agent-grip-claude")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("launcher-order-recent"));
    await waitFor(() => { expect(shortcutsApi.updateAgentPreferences).toHaveBeenCalledWith(expect.objectContaining({ orderMode: "recent" })); });
    await waitFor(() => { expect(screen.queryByTestId("launcher-agent-grip-claude")).toBeNull(); });
    fireEvent.click(screen.getByTestId("launcher-management-back"));
    expect(screen.getAllByRole("radio").map((radio) => radio.getAttribute("data-capability-id"))[0]).toBe("opencode");
  });

  it("records the launched harness without waiting on it", async () => {
    await profileDrawer();
    fireEvent.click(screen.getByTestId("launcher-agent-codex"));
    fireEvent.click(launchButton());
    expect(onLaunch).toHaveBeenCalled();
    await waitFor(() => { expect(shortcutsApi.updateAgentPreferences).toHaveBeenCalledWith(expect.objectContaining({ launchedAgentId: "codex" })); });
  });

  it("records nothing for a shell", async () => {
    await profileDrawer();
    fireEvent.click(screen.getByTestId("launcher-mode-shell"));
    fireEvent.click(launchButton());
    await act(async () => { await Promise.resolve(); });
    expect(shortcutsApi.updateAgentPreferences).not.toHaveBeenCalled();
  });
});
