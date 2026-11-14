import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders as render, setDesktopViewport, setMobileViewport } from "../../test-utils";
import type { UsageModeAgent, UsageModeState, UsageModeView } from "../../api/usageMode";
import TerminalLauncher from "../TerminalLauncher";
import UsageModeSheet from "./UsageModeSheet";
import { useUsageMode } from "./useUsageMode";

const api = vi.hoisted(() => ({ getUsageMode: vi.fn(), startUsageMode: vi.fn(), endUsageMode: vi.fn() }));
vi.mock("../../api/usageMode", () => api);
vi.mock("./useAgentUpdates", () => ({ useAgentUpdates: () => ({ updates: {}, loading: false, runs: {}, run: vi.fn(), retryCheck: vi.fn() }) }));
vi.mock("./useHarnesses", () => ({ useHarnesses: () => ({ harnesses: {}, known: false }), declaredWindowsFor: () => 0 }));
const policy = vi.hoisted(() => ({ literal: false }));
vi.mock("./useModelPolicy", async () => {
  const actual = await vi.importActual<typeof import("./useModelPolicy")>("./useModelPolicy");
  return { ...actual, useModelPolicy: () => ({ policy: null, loading: false, failed: false, choices: [], sources: [], sourceChoices: [], stale: false, selectable: false, choose: vi.fn(), usesVrooliCredits: false, literal: policy.literal, reload: vi.fn() }) };
});
const usage = vi.hoisted(() => ({ windows: [] as { pool: string; windowMinutes: number; usedPercent: number; resetsAt: string; observedAt: string }[] }));
vi.mock("./useAgentUsage", () => ({ useAgentUsage: () => ({ usages: { codex: { agentId: "codex", targetId: "local", match: "likely", reason: "", source: "p", observedAt: new Date().toISOString(), windows: usage.windows } }, loading: false, refresh: vi.fn() }) }));

const codex: UsageModeAgent = { agentId: "codex", label: "Codex", runner: "codex", lowerModel: "gpt-6-luna", usualModel: "gpt-6.1-terra" };
const inMode: UsageModeAgent = { ...codex, modeId: "um-1", model: "gpt-6-luna", fullPowerArgs: "-m gpt-6.1-terra" };
const mode: UsageModeView = { id: "um-1", label: "Save usage", runner: "codex", startedAt: new Date().toISOString(), expiresAt: new Date(Date.now() + 3_600_000).toISOString(), untilKind: "time", kind: "save" };
const target = { id: "local", kind: "local" as const, label: "This machine", available: true, state: "dispatchable" as const, readiness: [{ key: "capability:codex", label: "Codex", passed: true, state: "ready" as const, detail: "" }] };

const launchButton = () => screen.getByTestId("launcher-submit-selection");

async function renderDrawer(agents: UsageModeAgent[], modes: UsageModeView[], command = "codex --yolo") {
  api.getUsageMode.mockResolvedValue({ modes, agents, canControl: true });
  const onLaunch = vi.fn();
  render(<TerminalLauncher open onClose={vi.fn()} onLaunch={onLaunch} shortcuts={[{ agentId: "codex", label: "Codex", command }]} availableTargets={[target]} />);
  fireEvent.click(screen.getByTestId("launcher-mode-agent"));
  await waitFor(() => { expect(api.getUsageMode).toHaveBeenCalled(); });
  await screen.findByTestId("launcher-save-usage");
  return onLaunch;
}

describe("usage mode in the settings panel", () => {
  beforeEach(() => { vi.resetAllMocks(); setMobileViewport(); });

  it("changes nothing when no mode applies, and offers to start one", async () => {
    const onLaunch = await renderDrawer([codex], []);
    expect(screen.queryByTestId("launcher-mode-note-codex")).toBeNull();
    expect(screen.queryByTestId("launcher-full-power-codex")).toBeNull();
    expect(screen.getByTestId("launcher-save-usage-switch")).not.toBeChecked();
    expect(screen.getByTestId("launcher-save-usage")).toHaveTextContent("launcher.usageModeOffHint");
    fireEvent.click(launchButton());
    expect(onLaunch).toHaveBeenCalledWith(expect.objectContaining({ command: "codex --yolo" }));
  });

  it("states the lowering and the end time, and launches normally", async () => {
    const onLaunch = await renderDrawer([inMode], [mode]);
    expect(screen.getByTestId("launcher-mode-note-codex")).toHaveTextContent("launcher.usageModeLowers");
    expect(screen.getByTestId("launcher-save-usage-switch")).toBeChecked();
    expect(screen.getByTestId("launcher-save-usage")).toHaveTextContent("launcher.usageModeOnHint");
    expect(screen.getByTestId("launcher-summary")).toHaveTextContent("gpt-6-luna");
    fireEvent.click(launchButton());
    expect(onLaunch).toHaveBeenLastCalledWith(expect.objectContaining({ command: "codex --yolo" }));
  });

  it("opens the sheet for this harness to choose how long, and ends the mode directly", async () => {
    await renderDrawer([codex], []);
    fireEvent.click(screen.getByTestId("launcher-save-usage-switch"));
    expect(await screen.findByTestId("usage-mode-agent-codex")).toHaveAttribute("aria-checked", "true");
  });

  it("full power passes the normal default for that launch only", async () => {
    const onLaunch = await renderDrawer([inMode], [mode]);
    fireEvent.click(screen.getByTestId("launcher-full-power-codex"));
    expect(screen.getByTestId("launcher-summary")).toHaveTextContent("gpt-6.1-terra");
    fireEvent.click(launchButton());
    expect(onLaunch).toHaveBeenLastCalledWith(expect.objectContaining({ command: "codex --yolo", agentLaunch: expect.objectContaining({ agent_id: "codex", model: "gpt-6.1-terra" }) }));
    fireEvent.click(screen.getByTestId("launcher-full-power-codex"));
    fireEvent.click(launchButton());
    expect(onLaunch).toHaveBeenLastCalledWith(expect.objectContaining({ command: "codex --yolo", agentLaunch: undefined }));
  });

  it("offers no full power for a command of the operator's own", async () => {
    policy.literal = true;
    await renderDrawer([inMode], [mode], "codex -m gpt-6-luna");
    expect(screen.queryByTestId("launcher-full-power-codex")).toBeNull();
    policy.literal = false;
  });

  it("offers no full power when there is nothing to restore", async () => {
    await renderDrawer([{ ...inMode, fullPowerArgs: undefined }], [mode]);
    expect(screen.queryByTestId("launcher-full-power-codex")).toBeNull();
  });
});

function Sheet() {
  const controller = useUsageMode(true);
  return <UsageModeSheet open onClose={vi.fn()} usageMode={controller} />;
}
const state = (over: Partial<UsageModeState>): UsageModeState => ({ modes: [], agents: [codex], canControl: true, ...over });

const learn: UsageModeView = { id: "um-learn", label: "Learn: code.default", runner: "codex", startedAt: new Date().toISOString(), expiresAt: new Date(Date.now() + 86_400_000).toISOString(), untilKind: "time", kind: "learn", role: "code.default", splitPercent: 50, sessionsOptIn: true };

describe("a Learn comparison in the launcher", () => {
  beforeEach(() => { vi.resetAllMocks(); setMobileViewport(); });

  it("claims no model when the session's model depends on its arm", async () => {
    await renderDrawer([{ ...codex, learnModeId: "um-learn" }], [learn]);
    const note = screen.getByTestId("launcher-mode-note-codex");
    expect(note).toHaveTextContent("launcher.usageModeLearnArm");
    expect(note).not.toHaveTextContent("launcher.usageModeLowers");
    expect(screen.queryByTestId("launcher-full-power-codex")).toBeNull();
  });
});

describe("usage mode sheet", () => {
  beforeEach(() => { vi.resetAllMocks(); usage.windows = []; setDesktopViewport(); });

  it("submits the harness and duration selected by keyboard and pointer", async () => {
    const claude = { ...codex, agentId: "claude", runner: "claude", label: "Claude Code" };
    api.getUsageMode.mockResolvedValue(state({ agents: [codex, claude] }));
    api.startUsageMode.mockResolvedValue(state({}));
    render(<Sheet />);
    const first = await screen.findByTestId("usage-mode-agent-codex");
    first.focus();
    fireEvent.keyDown(first, { key: "ArrowRight" });
    expect(screen.getByTestId("usage-mode-agent-claude")).toHaveFocus();
    expect(screen.getByTestId("usage-mode-agent-claude")).toHaveAttribute("aria-checked", "true");
    fireEvent.click(first);
    expect(first).toHaveAttribute("aria-checked", "true");
    const duration = screen.getByTestId("usage-mode-choice-minutes-120");
    duration.focus();
    fireEvent.keyDown(duration, { key: "End" });
    expect(screen.getByTestId("usage-mode-choice-off")).toHaveFocus();
    expect(screen.getByTestId("usage-mode-choice-off")).toHaveAttribute("aria-checked", "true");
    fireEvent.click(screen.getByTestId("usage-mode-start"));
    await waitFor(() => { expect(api.startUsageMode).toHaveBeenCalledWith({ agentId: "codex", kind: "off" }); });
  });

  it("starts a mode from the operator's choice", async () => {
    api.getUsageMode.mockResolvedValue(state({}));
    api.startUsageMode.mockResolvedValue(state({ modes: [mode], agents: [inMode] }));
    render(<Sheet />);
    fireEvent.click(await screen.findByTestId("usage-mode-choice-minutes-240"));
    fireEvent.click(screen.getByTestId("usage-mode-start"));
    await waitFor(() => { expect(api.startUsageMode).toHaveBeenCalledWith({ agentId: "codex", kind: "minutes", minutes: 240 }); });
  });

  it("ends the active mode", async () => {
    api.getUsageMode.mockResolvedValue(state({ modes: [mode], agents: [inMode] }));
    api.endUsageMode.mockResolvedValue(state({}));
    render(<Sheet />);
    fireEvent.click(await screen.findByTestId("usage-mode-end"));
    await waitFor(() => { expect(api.endUsageMode).toHaveBeenCalledWith("um-1"); });
  });

  it("offers until-reset only when the quota reading supplies a reset, never a guess", async () => {
    api.getUsageMode.mockResolvedValue(state({}));
    const { unmount } = render(<Sheet />);
    await screen.findByTestId("usage-mode-start");
    expect(screen.queryByTestId("usage-mode-choice-reset-10080")).toBeNull();
    unmount();
    usage.windows = [{ pool: "primary", windowMinutes: 10080, usedPercent: 10, resetsAt: new Date(Date.now() + 86_400_000).toISOString(), observedAt: new Date().toISOString() }];
    render(<Sheet />);
    fireEvent.click(await screen.findByTestId("usage-mode-choice-reset-10080"));
    api.startUsageMode.mockResolvedValue(state({}));
    fireEvent.click(screen.getByTestId("usage-mode-start"));
    await waitFor(() => { expect(api.startUsageMode).toHaveBeenCalledWith({ agentId: "codex", kind: "reset", windowMinutes: 10080 }); });
  });

  it("is read-only when the viewer is not the operator", async () => {
    api.getUsageMode.mockResolvedValue(state({ modes: [mode], agents: [inMode], canControl: false }));
    render(<Sheet />);
    expect(await screen.findByTestId("usage-mode-operator-only")).toBeInTheDocument();
    expect(screen.queryByTestId("usage-mode-start")).toBeNull();
    expect(screen.queryByTestId("usage-mode-end")).toBeNull();
  });

  it("treats a failed read as no mode", async () => {
    api.getUsageMode.mockRejectedValue(new Error("down"));
    render(<Sheet />);
    await waitFor(() => { expect(api.getUsageMode).toHaveBeenCalled(); });
    expect(screen.queryByTestId("usage-mode-start")).toBeNull();
    expect(screen.queryByTestId("usage-mode-end")).toBeNull();
  });

  it("recovers the real mode after a brief connection failure without replaying mutations", async () => {
    api.getUsageMode.mockRejectedValueOnce(new TypeError("Failed to fetch"))
      .mockResolvedValue(state({ modes: [mode], agents: [inMode] }));
    render(<Sheet />);
    expect(await screen.findByTestId("usage-mode-end", {}, { timeout: 3000 })).toBeInTheDocument();
    expect(api.getUsageMode).toHaveBeenCalledTimes(2);
    expect(api.startUsageMode).not.toHaveBeenCalled();
    expect(api.endUsageMode).not.toHaveBeenCalled();
  });

  it("lists a Learn comparison as a comparison with no End button, apart from Save modes", async () => {
    api.getUsageMode.mockResolvedValue(state({ modes: [learn] }));
    render(<Sheet />);
    const card = await screen.findByTestId("usage-mode-learn-um-learn");
    expect(card).toHaveTextContent("launcher.learnComparisonLine");
    expect(card).toHaveTextContent("launcher.learnComparisonSessions");
    expect(screen.queryByTestId("usage-mode-active-um-learn")).toBeNull();
    expect(screen.queryByTestId("usage-mode-end")).toBeNull();
  });
});
