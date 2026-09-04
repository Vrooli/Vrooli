import { useEffect, useState } from 'react'
import { tuning } from '../../config'
import { readDiagnostics, subscribeDiagnostics, type WorldDiagnostics } from './store'

const REFRESH_MS = 250

/** DOM overlay with the renderer counters. Throttled so it never re-renders per frame. */
export function DiagnosticsOverlay({ seed, seedDigest, testId = 'world-diagnostics' }: { seed: number; seedDigest: string; testId?: string }) {
  const [snapshot, setSnapshot] = useState<WorldDiagnostics>(() => readDiagnostics())

  useEffect(() => {
    let dirty = false
    const off = subscribeDiagnostics(() => { dirty = true })
    const timer = window.setInterval(() => {
      if (!dirty) return
      dirty = false
      setSnapshot(readDiagnostics())
    }, REFRESH_MS)
    return () => {
      off()
      window.clearInterval(timer)
    }
  }, [])

  const rows: Array<[string, string]> = [
    ['scene', `${snapshot.scene} · ${snapshot.period}`],
    ['profile', `${snapshot.profile}${snapshot.auto ? ' (auto)' : ''} · dpr ${snapshot.dpr.toFixed(2)} · msaa ${snapshot.msaa}`],
    ['draw calls', String(snapshot.drawCalls)],
    ['triangles', snapshot.triangles.toLocaleString()],
    ['programs', String(snapshot.programs)],
    ['rendered frames / sec', String(snapshot.framesRendered)],
    ['frame p50 / p95', `${snapshot.frameMsP50.toFixed(1)} / ${snapshot.frameMsP95.toFixed(1)} ms`],
    ['GPU p50 / p95', snapshot.gpuTimerReason ? snapshot.gpuTimerReason : `${snapshot.gpuMsP50.toFixed(2)} / ${snapshot.gpuMsP95.toFixed(2)} ms`],
    ['passes (shadow/main/post/total)', `${snapshot.passMs.shadow.toFixed(2)} / ${snapshot.passMs.main.toFixed(2)} / ${snapshot.passMs.post.toFixed(2)} / ${snapshot.passMs.total.toFixed(2)} ms`],
    ['unattributed', `${snapshot.drawCallsUnattributed} draws · ${snapshot.trianglesUnattributed.toLocaleString()} tris`],
    ['shadow refreshes', String(snapshot.shadowRefreshes)],
    ['quality verdict', snapshot.qualityHistory[snapshot.qualityHistory.length - 1]?.reason ?? 'none'],
    ['tone mapping', snapshot.toneMapping],
    ['post', `ao ${snapshot.ao ? 'on' : 'off'} · bloom ${snapshot.bloom ? 'on' : 'off'}`],
    ['nearest hit', snapshot.nearestHit < 0 ? '—' : `${snapshot.nearestHit.toFixed(1)} m`],
    ['footprint fill', `${(snapshot.footprintFill * 100).toFixed(0)}%`],
    ['ready', snapshot.ready ? 'yes' : 'no'],
  ]

  const budget = tuning.budgets.scenes[snapshot.scene][snapshot.profile]
  const budgetOk = snapshot.ready && snapshot.drawCalls <= budget.drawCalls && snapshot.triangles <= budget.triangles
  return (
    <div
      data-testid={testId}
      data-ready={snapshot.ready ? 'true' : 'false'}
      data-budget-ok={budgetOk ? 'true' : 'false'}
      data-draw-calls={snapshot.drawCalls}
      data-triangles={snapshot.triangles}
      data-profile={snapshot.profile}
      data-scene={snapshot.scene}
      data-seed={seed}
      data-seed-digest={seedDigest}
      data-weather={snapshot.weather}
      className="pointer-events-none absolute bottom-3 left-3 z-30 rounded-md border border-border bg-background/85 px-3 py-2 font-mono text-[11px] leading-4 text-foreground shadow-sm backdrop-blur"
    >
      <table>
        <tbody>
          {rows.map(([label, value]) => (
            <tr key={label}>
              <td className="pr-3 text-muted-foreground">{label}</td>
              <td className="tabular-nums">{value}</td>
            </tr>
          ))}
        </tbody>
      </table>
      {snapshot.gpu && <p className="mt-1 max-w-[280px] truncate text-muted-foreground">{snapshot.gpu}</p>}
    </div>
  )
}
