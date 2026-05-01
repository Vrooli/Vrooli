import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useAgentSessionStore, agentSessionStoreInitialState } from "../stores/agent-session-store";
import type { AgentSession } from "../types";
import { useAgentSessionPolling } from "./useAgentSessionPolling";

const ACTIVE_SESSION: AgentSession = {
  id: "sess_active",
  title: "Plan work",
  kind: "meta_orchestration",
  status: "running",
  skillId: "swarm-manager-meta-orchestrator",
  createdAt: "2026-05-01T12:00:00Z",
  updatedAt: "2026-05-01T12:00:00Z",
  messages: [],
  proposals: [],
  artifacts: [],
};

const COMPLETE_SESSION: AgentSession = {
  ...ACTIVE_SESSION,
  id: "sess_complete",
  status: "complete",
};

function resetStore(overrides: Partial<ReturnType<typeof useAgentSessionStore.getState>> = {}) {
  const fetchSessions = vi.fn().mockResolvedValue(undefined);
  const refreshSession = vi.fn().mockResolvedValue(ACTIVE_SESSION);
  useAgentSessionStore.setState({
    ...agentSessionStoreInitialState,
    ...overrides,
    fetchSessions,
    refreshSession,
  });
  return { fetchSessions, refreshSession };
}

describe("useAgentSessionPolling", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    cleanup();
    vi.useRealTimers();
    act(() => {
      resetStore();
    });
  });

  it("polls the active list when any session is active", async () => {
    const { fetchSessions } = resetStore({ sessions: [ACTIVE_SESSION] });

    renderHook(() => useAgentSessionPolling());

    await act(async () => {
      vi.advanceTimersByTime(4_000);
      await Promise.resolve();
    });
    expect(fetchSessions).toHaveBeenCalledWith({ activeOnly: true }, { force: true });
  });

  it("polls the active detail session separately", async () => {
    const { refreshSession } = resetStore({
      sessions: [ACTIVE_SESSION],
      activeSession: ACTIVE_SESSION,
    });

    renderHook(() => useAgentSessionPolling());

    await act(async () => {
      vi.advanceTimersByTime(3_000);
      await Promise.resolve();
    });
    expect(refreshSession).toHaveBeenCalledWith("sess_active");
  });

  it("does not poll completed sessions", async () => {
    const { fetchSessions, refreshSession } = resetStore({
      sessions: [COMPLETE_SESSION],
      activeSession: COMPLETE_SESSION,
    });

    renderHook(() => useAgentSessionPolling());

    await act(async () => {
      vi.advanceTimersByTime(10_000);
      await Promise.resolve();
    });
    expect(fetchSessions).not.toHaveBeenCalled();
    expect(refreshSession).not.toHaveBeenCalled();
  });
});
