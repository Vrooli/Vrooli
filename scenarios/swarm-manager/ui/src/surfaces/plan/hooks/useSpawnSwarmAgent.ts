/**
 * useSpawnSwarmAgent — the board's Spawn action. Creates a swarm-operations
 * agent session and navigates to it (the Command Post triage page the old
 * spawn buttons pointed at is retired; this spawns a real operator agent).
 */

import { useCallback, useState } from "react";
import { useNavigate } from "react-router-dom";
import { sessionDetailPath } from "../../../app/routes/route-paths";
import { useAgentSessionStore } from "../../../stores";
import { SESSION_CREATE_TITLES } from "../../../components/session/session-view-model";

export interface SpawnSwarmAgentResult {
  spawn: () => Promise<void>;
  isSpawning: boolean;
  error: string | null;
}

export function useSpawnSwarmAgent(): SpawnSwarmAgentResult {
  const navigate = useNavigate();
  const createSession = useAgentSessionStore((s) => s.createSession);
  const isMutating = useAgentSessionStore((s) => s.isMutating);
  const [error, setError] = useState<string | null>(null);

  const spawn = useCallback(async () => {
    if (isMutating) return;
    setError(null);
    try {
      const session = await createSession({
        kind: "swarm_operations",
        title: SESSION_CREATE_TITLES.swarm_operations,
      });
      navigate(sessionDetailPath(session.id));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to start agent session.");
    }
  }, [createSession, isMutating, navigate]);

  return { spawn, isSpawning: isMutating, error };
}
