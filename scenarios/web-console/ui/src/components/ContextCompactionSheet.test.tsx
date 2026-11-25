import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { act } from "react";
import { renderWithProviders as render } from "../test-utils";
import type { CompactCandidate } from "../api/usageMode";
import { useContextCompactionStore } from "../stores/useContextCompactionStore";
import { useHandoffRequestStore } from "../stores/useHandoffRequestStore";
import ContextCompactionSheet from "./ContextCompactionSheet";

const api = vi.hoisted(() => ({ getCompactCandidates: vi.fn(), compactIdleSessions: vi.fn(), getSessionCompaction: vi.fn(), setSessionCompaction: vi.fn() }));
vi.mock("../api/usageMode", async (importOriginal) => ({ ...(await importOriginal<typeof import("../api/usageMode")>()), ...api }));

const row = (patch: Partial<CompactCandidate>): CompactCandidate => ({ sessionId: "s1", label: "Planner", harness: "codex", state: "idle", contextTokens: 210_000, goalActive: false, eligible: true, armed: false, thresholdTokens: 0, ...patch });

beforeEach(() => {
  Object.values(api).forEach((fn) => fn.mockReset());
  act(() => { useContextCompactionStore.getState().hide(); });
  useHandoffRequestStore.setState({ request: null, nextId: 1 });
});

describe("compact idle sessions", () => {
  it("lists every session with its size first, offers only eligible ones, and acts on exactly the ticked ones", async () => {
    api.getCompactCandidates.mockResolvedValue([
      row({ sessionId: "idle1", label: "Idle one", contextBudget: { model: "gpt-6.1-terra", modelState: "reported", usedTokens: 120_000, usedSource: "native transcript", modelWindowTokens: 200_000, modelWindowSource: "codex-models-cache/v1", compactionPointTokens: 150_000, compactionPointSource: "session launch descriptor", tokensUntilCompaction: 30_000, forecastState: "projected", secondsUntilCompaction: 95, tokensPerSecond: 315.79, progressPercent: 60 } }),
      row({ sessionId: "busy1", label: "Busy one", state: "working", eligible: false, skipReason: "busy: the session is working on a turn" }),
      row({ sessionId: "goal1", label: "Goal one", goalActive: true, eligible: false, skipReason: "an active goal: compacting could clear or derail it" }),
    ]);
    api.compactIdleSessions.mockResolvedValue([{ sessionId: "idle1", at: "2026-10-30T19:00:00Z", trigger: "bulk", outcome: "compacted", tokensBefore: 210_000 }]);
    render(<ContextCompactionSheet />);
    act(() => { useContextCompactionStore.getState().show(); });
    const rows = await screen.findAllByTestId("compaction-candidate");
    expect(rows).toHaveLength(3);
    expect(within(rows[0] as HTMLElement).getByTestId("context-budget")).toHaveTextContent("codex-models-cache/v1");
    expect(within(rows[0] as HTMLElement).getByTestId("context-budget")).toHaveAttribute("data-window-tokens", "200000");
    expect(within(rows[0] as HTMLElement).getByTestId("context-budget")).toHaveAttribute("data-window-source", "codex-models-cache/v1");
    expect(within(rows[0] as HTMLElement).getByTestId("context-budget")).toHaveAttribute("data-compaction-tokens", "150000");
    expect(within(rows[0] as HTMLElement).getByTestId("context-budget")).toHaveAttribute("data-compaction-source", "session launch descriptor");
    expect(within(rows[0] as HTMLElement).getByTestId("context-budget")).toHaveTextContent("30K");
    expect(within(rows[0] as HTMLElement).getByTestId("context-budget-forecast")).toHaveTextContent("launcher.contextBudgetForecast");
    expect(within(rows[0] as HTMLElement).getByRole("meter")).toHaveAttribute("aria-valuenow", "60");
    expect(api.compactIdleSessions).not.toHaveBeenCalled();
    expect(within(rows[0] as HTMLElement).getByRole("checkbox")).toBeChecked();
    for (const blocked of [rows[1], rows[2]] as HTMLElement[]) {
      expect(within(blocked).getByRole("checkbox")).toBeDisabled();
      expect(within(blocked).getByTestId("compaction-skip-reason").textContent).toMatch(/busy|goal/);
    }
    fireEvent.click(screen.getByTestId("compaction-run"));
    await waitFor(() => expect(api.compactIdleSessions).toHaveBeenCalledWith(["idle1"]));
    const result = await screen.findByTestId("compaction-result");
    expect(result).toHaveAttribute("data-outcome", "compacted");
  });

  it("keeps unconfirmed budget values explicitly unknown", async () => {
    api.getCompactCandidates.mockResolvedValue([row({ contextBudget: { modelState: "unknown", usedTokens: null, usedSource: "unknown", modelWindowTokens: null, modelWindowSource: "unknown", compactionPointTokens: null, compactionPointSource: "unknown", tokensUntilCompaction: null, forecastState: "unknown", progressPercent: null } })]);
    render(<ContextCompactionSheet />);
    act(() => { useContextCompactionStore.getState().show(); });
    const budget = await screen.findByTestId("context-budget");
    expect(budget).toHaveAttribute("data-forecast", "unknown");
    expect(budget).toHaveAttribute("data-window-tokens", "unknown");
    expect(budget).toHaveAttribute("data-window-source", "unknown");
    expect(budget).toHaveAttribute("data-compaction-tokens", "unknown");
    expect(budget).toHaveAttribute("data-compaction-source", "unknown");
    expect(budget).toHaveTextContent("launcher.contextWindowUnknown");
    expect(budget).toHaveTextContent("launcher.contextBudgetCompactionUnknown");
  });

  it("states a skipped session's reason in the per-session result", async () => {
    api.getCompactCandidates.mockResolvedValue([row({ sessionId: "s1" })]);
    api.compactIdleSessions.mockResolvedValue([{ sessionId: "s1", at: "2026-10-30T19:00:00Z", trigger: "bulk", outcome: "skipped", reason: "busy: the session is working on a turn" }]);
    render(<ContextCompactionSheet />);
    act(() => { useContextCompactionStore.getState().show(); });
    await screen.findByTestId("compaction-candidate");
    fireEvent.click(screen.getByTestId("compaction-run"));
    expect((await screen.findByTestId("compaction-reason")).textContent).toMatch(/busy/);
  });

  it("an unreachable server says nothing was changed", async () => {
    api.getCompactCandidates.mockRejectedValue(new Error("down"));
    render(<ContextCompactionSheet />);
    act(() => { useContextCompactionStore.getState().show(); });
    expect(await screen.findByTestId("compaction-error")).toBeInTheDocument();
  });
});

describe("one session's context controls", () => {
  const empty = { watch: { armed: false, thresholdTokens: 0 }, history: [], pinnedStanceMaxBytes: 4096 };

  it("is off by default and compacts only when asked: Compact when idle arms one compaction and shows what happened", async () => {
    api.getSessionCompaction.mockResolvedValue(empty);
    api.setSessionCompaction.mockResolvedValue({ watch: { armed: true, thresholdTokens: 0 }, armed: true, pinnedStanceMaxBytes: 4096, history: [{ at: "2026-10-30T19:00:00Z", trigger: "manual", outcome: "skipped", reason: "busy: the session is working on a turn" }] });
    render(<ContextCompactionSheet />);
    act(() => { useContextCompactionStore.getState().show("sess-1"); });
    await screen.findByTestId("compaction-session");
    expect(screen.getByTestId("compaction-threshold-0")).toHaveAttribute("aria-checked", "true");
    expect(api.setSessionCompaction).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("compaction-when-idle"));
    await waitFor(() => expect(api.setSessionCompaction).toHaveBeenCalledWith("sess-1", { action: "compact_when_idle" }));
    expect(await screen.findByTestId("compaction-armed")).toBeInTheDocument();
    expect(screen.getByTestId("compaction-reason").textContent).toMatch(/busy/);
  });

  it("sets and clears an optional threshold for that session", async () => {
    api.getSessionCompaction.mockResolvedValue(empty);
    api.setSessionCompaction.mockResolvedValue({ watch: { armed: false, thresholdTokens: 200_000 }, history: [], pinnedStanceMaxBytes: 4096 });
    render(<ContextCompactionSheet />);
    act(() => { useContextCompactionStore.getState().show("sess-1"); });
    await screen.findByTestId("compaction-session");
    fireEvent.click(screen.getByTestId("compaction-threshold-200000"));
    await waitFor(() => expect(api.setSessionCompaction).toHaveBeenCalledWith("sess-1", { action: "set_threshold", thresholdTokens: 200_000 }));
    await waitFor(() => expect(screen.getByTestId("compaction-threshold-200000")).toHaveAttribute("aria-checked", "true"));
  });

  it("opens the existing handoff composer for the same session", async () => {
    api.getSessionCompaction.mockResolvedValue(empty);
    render(<ContextCompactionSheet />);
    act(() => { useContextCompactionStore.getState().show("sess-1"); });
    await screen.findByTestId("compaction-session");
    fireEvent.click(screen.getByTestId("compaction-handoff-now"));
    expect(useHandoffRequestStore.getState().request).toEqual({ id: 1, sourceSessionId: "sess-1", payload: "" });
  });

  it("pins the exact one-line stance for the next compaction and can unpin it", async () => {
    api.getSessionCompaction.mockResolvedValue(empty);
    api.setSessionCompaction.mockResolvedValue({ ...empty, pinnedStance: true });
    render(<ContextCompactionSheet />);
    act(() => { useContextCompactionStore.getState().show("sess-1"); });
    await screen.findByTestId("compaction-session");
    const input = screen.getByTestId("context-pin-stance-input");
    fireEvent.change(input, { target: { value: "Keep the operator's exact words." } });
    fireEvent.click(screen.getByTestId("context-pin-stance"));
    await waitFor(() => expect(api.setSessionCompaction).toHaveBeenCalledWith("sess-1", { action: "pin_stance", stance: "Keep the operator's exact words." }));
    expect(await screen.findByTestId("context-pinned-stance")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("context-unpin-stance"));
    await waitFor(() => expect(api.setSessionCompaction).toHaveBeenLastCalledWith("sess-1", { action: "unpin_stance" }));
  });

  it("accepts exact stance text above the former 1000-character UI cap", async () => {
    api.getSessionCompaction.mockResolvedValue(empty);
    api.setSessionCompaction.mockResolvedValue({ ...empty, pinnedStance: true });
    render(<ContextCompactionSheet />);
    act(() => { useContextCompactionStore.getState().show("sess-1"); });
    await screen.findByTestId("compaction-session");
    const stance = "Keep this exact stance. ".repeat(50);
    fireEvent.change(screen.getByTestId("context-pin-stance-input"), { target: { value: stance } });
    expect(screen.getByTestId("context-pin-stance-byte-count")).toHaveTextContent("1200/4096");
    expect(screen.getByTestId("context-pin-stance")).toBeEnabled();
    fireEvent.click(screen.getByTestId("context-pin-stance"));
    await waitFor(() => expect(api.setSessionCompaction).toHaveBeenCalledWith("sess-1", { action: "pin_stance", stance }));
  });
});
