import { useEffect, useState } from "react";

type ActivationStatus = {
  first_command_run: boolean;
  first_ai_command_accepted: boolean;
};

const DISMISSED_KEY = "wc-activation-setup-dismissed-v1";

export default function ActivationSetupCard() {
  const [status, setStatus] = useState<ActivationStatus | null>(null);
  const [failed, setFailed] = useState(false);
  const [attempt, setAttempt] = useState(0);
  const [dismissed, setDismissed] = useState(() => {
    try { return window.localStorage.getItem(DISMISSED_KEY) === "true"; } catch { return false; }
  });

  useEffect(() => {
    if (dismissed) return;
    let current = true;
    fetch("/api/v1/activation/status", { credentials: "same-origin" })
      .then((response) => {
        if (!response.ok) throw new Error("activation status unavailable");
        return response.json() as Promise<ActivationStatus>;
      })
      .then((value) => { if (current) { setStatus(value); setFailed(false); } })
      .catch(() => { if (current) { setStatus(null); setFailed(true); } });
    return () => { current = false; };
  }, [attempt, dismissed]);

  if (dismissed) return null;
  const dismiss = () => {
    setDismissed(true);
    try { window.localStorage.setItem(DISMISSED_KEY, "true"); } catch { /* Keep this dismissal in memory. */ }
  };

  return (
    <section aria-labelledby="activation-setup-title" className="mx-auto mt-6 max-w-lg rounded-lg border border-wc-border-subtle bg-wc-surface-raised p-4 text-left">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h2 id="activation-setup-title" className="font-semibold">Get started with Aquila</h2>
          <p className="mt-1 text-sm text-wc-text-muted">You can use Aquila without an account. Sign-in is optional; Vrooli AI uses account credits.</p>
        </div>
        <button type="button" className="text-sm text-wc-text-muted hover:text-wc-text-primary" onClick={dismiss} aria-label="Dismiss setup guide">Dismiss</button>
      </div>
      {failed ? (
        <div role="status" className="mt-3 text-sm">
          <p>Setup progress is temporarily unavailable.</p>
          <button type="button" className="mt-2 underline" onClick={() => setAttempt((value) => value + 1)}>Retry</button>
        </div>
      ) : status ? (
        <ul className="mt-3 space-y-1 text-sm" aria-label="Aquila setup steps">
          <li>{status.first_command_run ? "✓" : "○"} Run your first command</li>
          <li>{status.first_ai_command_accepted ? "✓" : "○"} Try an AI command with your chosen provider</li>
        </ul>
      ) : <p className="mt-3 text-sm text-wc-text-muted" role="status">Loading setup progress…</p>}
    </section>
  );
}
