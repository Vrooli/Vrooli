import { useState } from "react";

import { killSession } from "../../api/sessions";
import { Button } from "../../components/ui/button";

type SessionKillButtonProps = {
  sessionId: string;
  onKilled?: () => void;
};

/**
 * Destructive, explicit control for terminating a remote interactive session.
 * The API remains the authority; this component only presents the same kill
 * switch used by the Bridge CLI.
 */
export function SessionKillButton({ sessionId, onKilled }: SessionKillButtonProps) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleKill() {
    setBusy(true);
    setError(null);
    try {
      await killSession(sessionId);
      onKilled?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to terminate session");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div>
      <Button type="button" variant="outline" disabled={busy} onClick={() => void handleKill()}>
        {busy ? "Terminating…" : "Terminate session"}
      </Button>
      {error ? <p role="alert">{error}</p> : null}
    </div>
  );
}
