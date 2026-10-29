import { useEffect, useMemo, useState } from "react";
import { CheckCircle2, Download, Loader2, RefreshCw } from "lucide-react";
import { getApiBaseUrl } from "../../lib/utils";
import { readableTitle } from "../../lib/runPresentation";

type Source = { runnerType: string; label: string; state: "ready" | "unavailable"; sessionCount: number };
type Session = { key: string; sessionId: string; title: string; preview?: string; updatedAt: string; importedRunId?: string };
const api = getApiBaseUrl();

/**
 * Manual import of saved coding-agent conversations as read-only runs. It lives
 * under Usage › Coverage until the transcript indexer makes it unnecessary.
 */
export function ManualImport() {
  const [sources, setSources] = useState<Source[]>([]);
  const [runnerType, setRunnerType] = useState("");
  const [sessions, setSessions] = useState<Session[]>([]);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [message, setMessage] = useState("Choose a coding-agent runner to browse its saved conversations.");
  const [loading, setLoading] = useState(true);
  const [importing, setImporting] = useState(false);
  const readySources = useMemo(() => sources.filter((source) => source.state === "ready"), [sources]);

  const loadSources = async () => {
    setLoading(true);
    try {
      const response = await fetch(`${api}/import/sources`);
      if (!response.ok) throw new Error();
      const data = (await response.json()) as { sources: Source[] };
      setSources(data.sources);
      setRunnerType((current) => current || data.sources.find((source) => source.state === "ready")?.runnerType || "");
    } catch {
      setMessage("Runner sessions could not be loaded. Check the runner resource and try again.");
    } finally {
      setLoading(false);
    }
  };

  const loadSessions = async (runner: string) => {
    if (!runner) return;
    setLoading(true);
    setSelected(new Set());
    try {
      const response = await fetch(`${api}/import/sessions?runnerType=${encodeURIComponent(runner)}`);
      if (!response.ok) throw new Error();
      const data = (await response.json()) as { sessions: Session[] };
      setSessions(data.sessions);
      setMessage(data.sessions.length ? "Select one or more conversations to import as read-only runs." : "No saved conversations were found for this runner.");
    } catch {
      setSessions([]);
      setMessage("This runner's conversations could not be read.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { void loadSources(); }, []);
  useEffect(() => { void loadSessions(runnerType); }, [runnerType]);

  const toggle = (key: string) => setSelected((current) => {
    const next = new Set(current);
    if (next.has(key)) next.delete(key);
    else next.add(key);
    return next;
  });

  const importSelected = async () => {
    setImporting(true);
    try {
      const result = await Promise.all([...selected].map(async (sessionKey) => {
        const response = await fetch(`${api}/import/sessions`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ runnerType, sessionKey }) });
        return response.ok;
      }));
      const count = result.filter(Boolean).length;
      setMessage(count === selected.size ? `${count} conversation${count === 1 ? "" : "s"} imported as read-only runs.` : `${count} conversations imported; some could not be imported.`);
      await loadSessions(runnerType);
    } finally {
      setImporting(false);
    }
  };

  return (
    <section className="space-y-4" aria-labelledby="manual-import-heading" data-testid="usage-manual-import">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h2 id="manual-import-heading" className="text-lg font-semibold">Manual import</h2>
          <p className="mt-1 text-sm text-muted-foreground">Bring saved coding-agent conversations in as read-only runs. Their source files are not modified.</p>
        </div>
        <button type="button" aria-label="Refresh runner conversations" onClick={() => void loadSessions(runnerType)} className="inline-flex min-h-11 items-center gap-2 rounded-md border border-border px-3 text-sm hover:bg-muted">
          <RefreshCw className="h-4 w-4" aria-hidden="true" />Refresh
        </button>
      </div>
      <div className="rounded-lg border border-border bg-card p-4">
        <label className="block text-sm font-medium" htmlFor="runner-source">Coding-agent runner</label>
        <select id="runner-source" value={runnerType} onChange={(event) => setRunnerType(event.target.value)} className="mt-2 min-h-11 w-full rounded-md border border-input bg-background px-3 sm:max-w-md">
          {readySources.map((source) => <option key={source.runnerType} value={source.runnerType}>{source.label} · {source.sessionCount} saved</option>)}
        </select>
        {sources.some((source) => source.state === "unavailable") && <p className="mt-2 text-xs text-muted-foreground">Unavailable runners remain private until their local session store is available.</p>}
      </div>
      <p role="status" className="text-sm text-muted-foreground">{message}</p>
      <div className="overflow-hidden rounded-lg border border-border">
        <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border bg-muted/30 px-4 py-3 text-sm">
          <span>{loading ? "Loading conversations…" : `${sessions.length} saved conversations`}</span>
          <button type="button" disabled={!selected.size || importing} onClick={() => void importSelected()} className="inline-flex min-h-11 items-center gap-2 rounded-md bg-primary px-3 text-primary-foreground disabled:opacity-50">
            <Download className="h-4 w-4" aria-hidden="true" />Import selected ({selected.size})
          </button>
        </div>
        {loading ? (
          <div className="flex min-h-40 items-center justify-center text-muted-foreground"><Loader2 className="mr-2 h-4 w-4 animate-spin" aria-hidden="true" />Loading</div>
        ) : sessions.length === 0 ? (
          <div className="p-8 text-center text-sm text-muted-foreground">No conversations to import from this runner.</div>
        ) : (
          <ul className="divide-y divide-border">
            {sessions.map((session) => (
              <li key={session.key} className={`flex gap-3 p-4 ${session.importedRunId ? "bg-muted/30" : "hover:bg-muted/20"}`}>
                <input aria-label={`Select ${session.title}`} className="mt-1 h-5 w-5" type="checkbox" checked={selected.has(session.key)} disabled={Boolean(session.importedRunId)} onChange={() => toggle(session.key)} />
                <div className="min-w-0 flex-1">
                  <div className="flex min-w-0 flex-wrap items-center gap-2">
                    <span className="min-w-0 truncate font-medium" title={session.title}>{readableTitle(session.title) || "Untitled conversation"}</span>
                    {session.importedRunId && <span className="inline-flex items-center gap-1 rounded-full bg-secondary px-2 py-0.5 text-xs text-secondary-foreground"><CheckCircle2 className="h-3 w-3" aria-hidden="true" />Already imported</span>}
                  </div>
                  <p className="mt-1 break-all text-xs text-muted-foreground">{new Date(session.updatedAt).toLocaleString()} · {session.sessionId}</p>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  );
}
