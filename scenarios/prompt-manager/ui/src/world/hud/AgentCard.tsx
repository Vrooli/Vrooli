import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { agentDetailPath, runDetailPath } from '@/app/routes/route-paths'
import { selectors } from '@/constants/selectors'
import type { WorldActions } from '../data'
import type { ActorView } from '../sim'
import { STATE_LABEL, formatDuration } from './format'

interface AgentCardProps {
  actor: ActorView
  teamName?: string
  now: number
  actions: WorldActions
  following: boolean
  onFollowChange: (follow: boolean) => void
  onClose: () => void
  onCustomize?: () => void
  /** Docked cards (2D mode) stretch; anchored cards float over the actor. */
  docked?: boolean
}

const STATE_TONE: Partial<Record<ActorView['state'], string>> = {
  working: 'bg-sky-500/15 text-sky-700 dark:text-sky-300',
  walkingToDesk: 'bg-sky-500/15 text-sky-700 dark:text-sky-300',
  failed: 'bg-red-500/15 text-red-700 dark:text-red-300',
  gathered: 'bg-amber-500/15 text-amber-700 dark:text-amber-300',
  walkingToTable: 'bg-amber-500/15 text-amber-700 dark:text-amber-300',
}

/** The focused agent: state, last run, skills, and every action the world offers. */
export function AgentCard({ actor, teamName, now, actions, following, onFollowChange, onClose, onCustomize, docked = false }: AgentCardProps) {
  const navigate = useNavigate()
  const [busy, setBusy] = useState<'run' | 'stop' | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const canRun = actor.teamId !== undefined && actor.state !== 'working' && actor.state !== 'walkingToDesk'
  const canStop = actor.teamId !== undefined && (actor.state === 'working' || actor.state === 'walkingToDesk')

  const run = async () => {
    if (!actor.teamId) return
    setBusy('run')
    setNotice(null)
    try {
      const result = await actions.runNow(actor.teamId, actor.id)
      setNotice(result.runId ? `Run ${result.runId} requested` : result.message)
    } catch (error) {
      setNotice(error instanceof Error ? error.message : 'Run request failed')
    } finally {
      setBusy(null)
    }
  }

  const stop = async () => {
    if (!actor.teamId) return
    setBusy('stop')
    setNotice(null)
    try {
      await actions.stop(actor.teamId, actor.id)
      setNotice('Stop requested')
    } catch (error) {
      setNotice(error instanceof Error ? error.message : 'Stop request failed')
    } finally {
      setBusy(null)
    }
  }

  return (
    <section
      className={`pointer-events-auto rounded-lg border border-border bg-background/95 p-3 text-sm shadow-lg backdrop-blur ${docked ? 'w-full' : 'w-72'}`}
      data-testid={selectors.world.hud.agentCard}
      aria-label={`Agent ${actor.name}`}
    >
      <header className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <h3 className="truncate font-semibold text-foreground">{actor.name}</h3>
          <p className="truncate text-xs text-muted-foreground">{teamName ?? 'No team'}</p>
        </div>
        <button type="button" onClick={onClose} aria-label="Close agent card" className="rounded p-1 text-muted-foreground hover:bg-muted hover:text-foreground" data-testid={selectors.world.hud.agentCardClose}>
          ×
        </button>
      </header>
      <dl className="mt-2 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
        <dt className="text-muted-foreground">State</dt>
        <dd>
          <span className={`rounded px-1.5 py-0.5 font-medium ${STATE_TONE[actor.state] ?? 'bg-muted text-foreground'}`} data-testid={selectors.world.hud.agentState}>
            {STATE_LABEL[actor.state]}
          </span>
          <span className="ml-2 text-muted-foreground">for {formatDuration(now - actor.stateSince)}</span>
        </dd>
        <dt className="text-muted-foreground">Last run</dt>
        <dd>
          {actor.lastRun ? (
            <button type="button" className="underline decoration-dotted underline-offset-2 hover:text-primary" onClick={() => navigate(runDetailPath(actor.lastRun?.runId ?? ''))}>
              {actor.lastRun.status}
              {actor.lastRun.endedAt !== undefined ? ` · ${formatDuration(actor.lastRun.endedAt - actor.lastRun.startedAt)}` : ''}
            </button>
          ) : (
            <span className="text-muted-foreground">none this session</span>
          )}
        </dd>
        {actor.failedError && (
          <>
            <dt className="text-muted-foreground">Error</dt>
            <dd className="truncate text-red-600 dark:text-red-400" title={actor.failedError}>{actor.failedError}</dd>
          </>
        )}
        <dt className="text-muted-foreground">Skills</dt>
        <dd className="tabular-nums">{actor.skillCount}</dd>
        {actor.message && (
          <>
            <dt className="text-muted-foreground">Says</dt>
            <dd className="italic">“{actor.message.text}”</dd>
          </>
        )}
      </dl>
      <div className="mt-3 flex flex-wrap gap-1.5">
        <button type="button" disabled={!canRun || busy !== null} onClick={() => void run()} className="rounded-md bg-primary px-2.5 py-1 text-xs font-medium text-primary-foreground disabled:opacity-50" data-testid={selectors.world.hud.runNow}>
          {busy === 'run' ? 'Requesting…' : 'Run now'}
        </button>
        <button type="button" disabled={!canStop || busy !== null} onClick={() => void stop()} className="rounded-md border border-border px-2.5 py-1 text-xs font-medium hover:bg-muted disabled:opacity-50" data-testid={selectors.world.hud.stopRun}>
          {busy === 'stop' ? 'Stopping…' : 'Stop'}
        </button>
        {actor.state === 'failed' && (
          <button type="button" onClick={() => actions.acknowledgeFailure(actor.id)} className="rounded-md border border-border px-2.5 py-1 text-xs font-medium hover:bg-muted" data-testid={selectors.world.hud.acknowledge}>
            Acknowledge
          </button>
        )}
        <button type="button" onClick={() => navigate(agentDetailPath(actor.id))} className="rounded-md border border-border px-2.5 py-1 text-xs font-medium hover:bg-muted" data-testid={selectors.world.hud.openEditor}>
          Open editor
        </button>
        {onCustomize && (
          <button type="button" onClick={onCustomize} className="rounded-md border border-border px-2.5 py-1 text-xs font-medium hover:bg-muted" data-testid={selectors.world.hud.customize}>
            Customize
          </button>
        )}
        <label className="ml-auto flex items-center gap-1 text-xs text-muted-foreground">
          <input type="checkbox" checked={following} onChange={(e) => onFollowChange(e.target.checked)} data-testid={selectors.world.hud.follow} />
          Follow
        </label>
      </div>
      {notice && (
        <p className="mt-2 text-xs text-muted-foreground" role="status" data-testid={selectors.world.hud.agentNotice}>
          {notice}
        </p>
      )}
    </section>
  )
}
