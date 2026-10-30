import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders as render, setDesktopViewport } from "../../test-utils";
import type { UsageModeAgent, UsageModeState, UsageModeView } from "../../api/usageMode";
import AgentGridSection, { withModelArgs } from "./AgentGridSection";
import UsageModeSheet from "./UsageModeSheet";
import { useUsageMode } from "./useUsageMode";

const api = vi.hoisted(() => ({ getUsageMode: vi.fn(), startUsageMode: vi.fn(), endUsageMode: vi.fn() }));
vi.mock("../../api/usageMode", () => api);
vi.mock("./useAgentUpdates", () => ({ useAgentUpdates: () => ({ updates: {}, loading: false }) }));
const usage = vi.hoisted(() => ({ windows: [] as { pool: string; windowMinutes: number; usedPercent: number; resetsAt: string }[] }));
vi.mock("./useAgentUsage", () => ({ useAgentUsage: () => ({ usages: { codex: { agentId: "codex", targetId: "local", match: "likely", reason: "", source: "p", observedAt: new Date().toISOString(), windows: usage.windows } }, loading: false, refresh: vi.fn() }) }));

const codex: UsageModeAgent = { agentId: "codex", label: "Codex", runner: "codex", lowerModel: "gpt-6-luna", usualModel: "gpt-6.1-terra" };
const inMode: UsageModeAgent = { ...codex, modeId: "um-1", model: "gpt-6-luna", fullPowerArgs: "-m gpt-6.1-terra" };
const mode: UsageModeView = { id: "um-1", label: "Save usage", runner: "codex", startedAt: new Date().toISOString(), expiresAt: new Date(Date.now() + 3_600_000).toISOString(), untilKind: "time" };
const target = { id: "local", kind: "local" as const, label: "This machine", available: true, state: "dispatchable" as const, readiness: [{ key: "capability:codex", label: "Codex", passed: true, state: "ready" as const, detail: "" }] };
const shortcuts = [{ agentId: "codex", label: "Codex", command: "codex --yolo" }];

describe("usage mode in the launcher grid", () => {
  it("changes nothing when no mode applies", () => {
    const onLaunch = vi.fn();
    render(<AgentGridSection shortcuts={shortcuts} target={target} usageAgents={[codex]} usageModes={[]} disabled={false} onLaunch={onLaunch} />);
    expect(screen.queryByTestId("launcher-mode-note-codex")).toBeNull();
    expect(screen.queryByTestId("launcher-full-power-codex")).toBeNull();
    fireEvent.click(screen.getByTestId("launcher-agent-codex"));
    expect(onLaunch).toHaveBeenCalledWith("codex --yolo");
  });

  it("shows the model a new session will use and why, and launches normally", () => {
    const onLaunch = vi.fn();
    render(<AgentGridSection shortcuts={shortcuts} target={target} usageAgents={[inMode]} usageModes={[mode]} disabled={false} onLaunch={onLaunch} />);
    expect(screen.getByTestId("launcher-mode-note-codex")).toHaveTextContent("launcher.usageModeWillUse");
    fireEvent.click(screen.getByTestId("launcher-agent-codex"));
    expect(onLaunch).toHaveBeenLastCalledWith("codex --yolo");
  });

  it("full power passes the normal default for that launch only", () => {
    const onLaunch = vi.fn();
    render(<AgentGridSection shortcuts={shortcuts} target={target} usageAgents={[inMode]} usageModes={[mode]} disabled={false} onLaunch={onLaunch} />);
    fireEvent.click(screen.getByTestId("launcher-full-power-codex"));
    expect(onLaunch).toHaveBeenLastCalledWith("codex --yolo -m gpt-6.1-terra");
    fireEvent.click(screen.getByTestId("launcher-agent-codex"));
    expect(onLaunch).toHaveBeenLastCalledWith("codex --yolo");
  });

  it("offers no full power when the command already names a model or there is nothing to restore", () => {
    const { rerender } = render(<AgentGridSection shortcuts={[{ agentId: "codex", label: "Codex", command: "codex -m gpt-6-luna" }]} target={target} usageAgents={[inMode]} usageModes={[mode]} disabled={false} onLaunch={vi.fn()} />);
    expect(screen.queryByTestId("launcher-full-power-codex")).toBeNull();
    rerender(<AgentGridSection shortcuts={shortcuts} target={target} usageAgents={[{ ...inMode, fullPowerArgs: undefined }]} usageModes={[mode]} disabled={false} onLaunch={vi.fn()} />);
    expect(screen.queryByTestId("launcher-full-power-codex")).toBeNull();
  });

  it("routes the model through --arg for the agent launcher form", () => {
    expect(withModelArgs("vrooli agent launch --runner claude", "--model sonnet")).toBe("vrooli agent launch --runner claude --arg=--model --arg=sonnet");
  });
});

function Sheet() {
  const controller = useUsageMode(true);
  return <UsageModeSheet open onClose={vi.fn()} usageMode={controller} />;
}
const state = (over: Partial<UsageModeState>): UsageModeState => ({ modes: [], agents: [codex], canControl: true, ...over });

describe("usage mode sheet", () => {
  beforeEach(() => { vi.resetAllMocks(); usage.windows = []; setDesktopViewport(); });

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
    usage.windows = [{ pool: "primary", windowMinutes: 10080, usedPercent: 10, resetsAt: new Date(Date.now() + 86_400_000).toISOString() }];
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
});
