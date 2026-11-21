import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { act } from "react";
import { renderWithProviders as render } from "../test-utils";
import type { CompactCandidate } from "../api/usageMode";
import { useContextCompactionStore } from "../stores/useContextCompactionStore";
import ContextCompactionSheet from "./ContextCompactionSheet";

const api = vi.hoisted(() => ({ getCompactCandidates: vi.fn(), compactIdleSessions: vi.fn(), getSessionCompaction: vi.fn(), setSessionCompaction: vi.fn() }));
vi.mock("../api/usageMode", () => api);

const row = (patch: Partial<CompactCandidate>): CompactCandidate => ({ sessionId: "s1", label: "Planner", harness: "codex", state: "idle", contextTokens: 210_000, goalActive: false, eligible: true, armed: false, thresholdTokens: 0, ...patch });

beforeEach(() => {
  Object.values(api).forEach((fn) => fn.mockReset());
  act(() => { useContextCompactionStore.getState().hide(); });
});

describe("compact idle sessions", () => {
  it("lists every session with its size first, offers only eligible ones, and acts on exactly the ticked ones", async () => {
    api.getCompactCandidates.mockResolvedValue([
      row({ sessionId: "idle1", label: "Idle one" }),
      row({ sessionId: "busy1", label: "Busy one", state: "working", eligible: false, skipReason: "busy: the session is working on a turn" }),
      row({ sessionId: "goal1", label: "Goal one", goalActive: true, eligible: false, skipReason: "an active goal: compacting could clear or derail it" }),
    ]);
    api.compactIdleSessions.mockResolvedValue([{ sessionId: "idle1", at: "2026-10-30T19:00:00Z", trigger: "bulk", outcome: "compacted", tokensBefore: 210_000 }]);
    render(<ContextCompactionSheet />);
    act(() => { useContextCompactionStore.getState().show(); });
    const rows = await screen.findAllByTestId("compaction-candidate");
    expect(rows).toHaveLength(3);
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
  const empty = { watch: { armed: false, thresholdTokens: 0 }, history: [] };

  it("is off by default and compacts only when asked: Compact when idle arms one compaction and shows what happened", async () => {
    api.getSessionCompaction.mockResolvedValue(empty);
    api.setSessionCompaction.mockResolvedValue({ watch: { armed: true, thresholdTokens: 0 }, armed: true, history: [{ at: "2026-10-30T19:00:00Z", trigger: "manual", outcome: "skipped", reason: "busy: the session is working on a turn" }] });
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
    api.setSessionCompaction.mockResolvedValue({ watch: { armed: false, thresholdTokens: 200_000 }, history: [] });
    render(<ContextCompactionSheet />);
    act(() => { useContextCompactionStore.getState().show("sess-1"); });
    await screen.findByTestId("compaction-session");
    fireEvent.click(screen.getByTestId("compaction-threshold-200000"));
    await waitFor(() => expect(api.setSessionCompaction).toHaveBeenCalledWith("sess-1", { action: "set_threshold", thresholdTokens: 200_000 }));
    await waitFor(() => expect(screen.getByTestId("compaction-threshold-200000")).toHaveAttribute("aria-checked", "true"));
  });
});
