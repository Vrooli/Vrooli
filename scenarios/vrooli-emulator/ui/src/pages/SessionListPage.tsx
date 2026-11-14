import { useCallback, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { AlertCircle, Loader2, Monitor, Plus } from "lucide-react";
import { listSessions, type Session } from "../lib/api/sessions";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { PlatformSelector } from "../components/sessions/PlatformSelector";
import { useSessionStore } from "../store/sessionStore";

function SessionRow({ session }: { session: Session }) {
  return (
    <Link
      to={`/sessions/${encodeURIComponent(session.id)}`}
      className="flex items-center justify-between gap-3 rounded-lg border border-white/10 bg-white/5 px-4 py-3 hover:border-white/20 hover:bg-white/10 transition"
    >
      <div className="flex items-center gap-3 min-w-0">
        <Monitor className="h-4 w-4 text-blue-400 shrink-0" />
        <div className="min-w-0">
          <p className="text-sm font-medium text-slate-100 truncate">{session.scenario_name || session.id}</p>
          <p className="text-xs text-slate-400">
            {session.platform ?? "unknown"} · {session.width}&times;{session.height} · {session.state}
          </p>
        </div>
      </div>
      <span className="text-xs text-slate-500 shrink-0">
        {new Date(session.created_at).toLocaleString()}
      </span>
    </Link>
  );
}

export function SessionListPage() {
  const navigate = useNavigate();
  const [platform, setPlatform] = useState("linux");
  const [width, setWidth] = useState(1280);
  const [height, setHeight] = useState(720);
  const [scenarioName, setScenarioName] = useState("");
  const [isCreating, setIsCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);

  const startSession = useSessionStore((s) => s.startSession);

  const {
    data: sessions,
    isLoading,
    error: listError,
    refetch,
  } = useQuery({
    queryKey: ["sessions"],
    queryFn: listSessions,
    refetchInterval: 5_000,
  });

  const handleCreate = useCallback(async () => {
    if (!scenarioName.trim()) {
      setCreateError("Scenario name is required");
      return;
    }
    setCreateError(null);
    setIsCreating(true);
    try {
      const session = await startSession({
        width,
        height,
        scenario_name: scenarioName.trim(),
        platform,
      });
      if (session) {
        navigate(`/sessions/${encodeURIComponent(session.id)}`);
      } else {
        setCreateError(useSessionStore.getState().error ?? "Failed to create session");
      }
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : "Failed to create session");
    } finally {
      setIsCreating(false);
    }
  }, [height, navigate, platform, scenarioName, startSession, width]);

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50 p-6">
      <div className="mx-auto flex w-full max-w-4xl flex-col gap-6">
        <header>
          <p className="text-sm uppercase tracking-[0.2em] text-slate-400">Vrooli Emulator</p>
          <h1 className="mt-1 text-3xl font-semibold">Sessions</h1>
          <p className="mt-1 text-sm text-slate-400">
            Manage emulator sessions for scenario development.
          </p>
        </header>

        <section className="rounded-2xl border border-white/10 bg-white/5 p-5">
          <div className="flex items-center gap-2">
            <Plus className="h-4 w-4 text-blue-400" />
            <h2 className="text-sm font-semibold text-slate-100">New Session</h2>
          </div>

          <div className="mt-4 space-y-4">
            <div className="space-y-1">
              <label className="text-xs text-slate-500" htmlFor="scenario-name">
                Scenario name
              </label>
              <Input
                id="scenario-name"
                type="text"
                value={scenarioName}
                onChange={(e) => setScenarioName(e.target.value)}
                placeholder="e.g., my-app-smoke"
              />
            </div>

            <div className="space-y-1">
              <label className="text-xs text-slate-500">Platform</label>
              <PlatformSelector value={platform} onChange={setPlatform} />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <label className="text-xs text-slate-500" htmlFor="session-width">
                  Width
                </label>
                <Input
                  id="session-width"
                  type="number"
                  value={width}
                  onChange={(e) => setWidth(Number(e.target.value) || 1280)}
                  min={800}
                  max={1920}
                />
              </div>
              <div className="space-y-1">
                <label className="text-xs text-slate-500" htmlFor="session-height">
                  Height
                </label>
                <Input
                  id="session-height"
                  type="number"
                  value={height}
                  onChange={(e) => setHeight(Number(e.target.value) || 720)}
                  min={600}
                  max={1080}
                />
              </div>
            </div>

            {createError && (
              <div className="flex items-start gap-2 rounded-lg border border-red-800/60 bg-red-950/30 p-3">
                <AlertCircle className="h-4 w-4 text-red-400 shrink-0 mt-0.5" />
                <p className="text-xs text-red-300">{createError}</p>
              </div>
            )}

            <Button onClick={() => void handleCreate()} disabled={isCreating} className="w-full">
              {isCreating ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Creating...
                </>
              ) : (
                <>
                  <Monitor className="mr-2 h-4 w-4" />
                  Start Session
                </>
              )}
            </Button>
          </div>
        </section>

        <section className="rounded-2xl border border-white/10 bg-white/5 p-5">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold text-slate-100">Active sessions</h2>
            <button
              type="button"
              onClick={() => void refetch()}
              className="text-xs text-slate-400 hover:text-slate-200"
            >
              Refresh
            </button>
          </div>

          <div className="mt-4 space-y-2">
            {isLoading && (
              <div className="flex items-center gap-2 text-sm text-slate-400">
                <Loader2 className="h-4 w-4 animate-spin" />
                Loading sessions...
              </div>
            )}

            {listError && (
              <div className="flex items-start gap-2 rounded-lg border border-red-800/60 bg-red-950/30 p-3">
                <AlertCircle className="h-4 w-4 text-red-400 shrink-0 mt-0.5" />
                <p className="text-xs text-red-300">
                  {listError instanceof Error ? listError.message : "Failed to load sessions"}
                </p>
              </div>
            )}

            {!isLoading && !listError && sessions && sessions.length === 0 && (
              <p className="text-sm text-slate-400">No active sessions yet. Start one above.</p>
            )}

            {!isLoading && !listError && sessions && sessions.length > 0 && (
              <div className="space-y-2">
                {sessions.map((session) => (
                  <SessionRow key={session.id} session={session} />
                ))}
              </div>
            )}
          </div>
        </section>
      </div>
    </div>
  );
}
