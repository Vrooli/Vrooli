import { useEffect, useState } from 'react'
import type { PreparationSnapshot } from '../data/scenePreparation'

export interface WorkbenchSnapshot {
  preparation: PreparationSnapshot
  generation: { count: number; source: string; reason: string }
  caches: Array<{ name: string; entries: number; bytes: number; budget: number; hits: number; misses: number; evictions: number }>
}

const memory = (bytes: number) => `${(bytes / 1024 / 1024).toFixed(2)} MiB`

/** Refresh only while the opt-in workbench is expanded. Never drives generation. */
export function WorkbenchStatus({ read, active }: { read: () => WorkbenchSnapshot; active: boolean }) {
  const [snapshot, setSnapshot] = useState<WorkbenchSnapshot | null>(null)
  useEffect(() => {
    if (!active) return
    const refresh = () => setSnapshot(read())
    refresh()
    const timer = window.setInterval(refresh, 500)
    return () => window.clearInterval(timer)
  }, [read, active])
  if (!snapshot) return null
  return <section aria-label="World preparation and caches" className="mt-3 space-y-2 text-xs">
    <p role="status">Preparation {snapshot.preparation.epoch}: {snapshot.preparation.state}</p>
    <ul className="space-y-1">
      {snapshot.preparation.stages.map((stage, index) => <li key={`${stage.stage}-${index}`}>
        {stage.stage}: {stage.outcome}{stage.completedAt !== null ? ` · ${Math.max(0, stage.completedAt - stage.startedAt).toFixed(0)} ms` : ''}
      </li>)}
    </ul>
    <p>Generation {snapshot.generation.count}: {snapshot.generation.source} · {snapshot.generation.reason}</p>
    <div className="overflow-x-auto">
      <table className="w-full text-left tabular-nums">
        <caption className="mb-1 text-left font-medium">Estimated cache residency</caption>
        <thead><tr><th>Cache</th><th>Entries</th><th>Resident / budget</th><th>Hits / misses / evictions</th></tr></thead>
        <tbody>{snapshot.caches.map(cache => <tr key={cache.name}>
          <th className="pr-2 font-normal">{cache.name}</th><td>{cache.entries}</td>
          <td className="whitespace-nowrap pr-2">{memory(cache.bytes)} / {memory(cache.budget)}</td>
          <td>{cache.hits} / {cache.misses} / {cache.evictions}</td>
        </tr>)}</tbody>
      </table>
    </div>
    <p className="text-muted-foreground">Caches may share buffers. These estimates are separate and do not represent total process or GPU memory.</p>
  </section>
}
