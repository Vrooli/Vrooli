import { useState } from 'react'
import { selectors } from '@/constants/selectors'
import type { ActorView, NavGrid, TeamView } from '../sim'
import type { TerrainField } from '../sim/terrain'
import type { WaterGeometryData } from '../sim/terrain/waterSurface'
import type { WeatherState } from '../sim'
import { STATE_LABEL, formatDuration } from './format'

interface TwoDModeProps {
  actors: ActorView[]
  teams: TeamView[]
  places?: import('../sim').WorldView['places']
  bounds?: import('../sim').WorldView['bounds']
  terrain?: TerrainField
  biomes?: Uint8Array
  biomeSetId?: string
  pathMask?: Float32Array
  nav?: NavGrid
  waterGeometry?: WaterGeometryData[]
  now: number
  focusedId: string | null
  onFocus: (agentId: string | null) => void
  weather?: Pick<WeatherState, 'state' | 'pressure'>
  feedMode?: 'connecting' | 'stream' | 'polling' | 'stopped' | 'snapshot'
  totalActors?: number
  filtersActive?: boolean
}

/**
 * The world without the canvas: every actor as a row grouped by team, with
 * the same focus target the 3D view uses. Everything the AgentCard offers is
 * reachable from here.
 */
export function TwoDMode({ actors, teams, places = [], bounds, terrain, biomes, biomeSetId, pathMask, nav, waterGeometry = [], now, focusedId, onFocus, weather, feedMode = 'stream', totalActors = actors.length, filtersActive = false }: TwoDModeProps) {
  const [mapZoom, setMapZoom] = useState(1)
  // The world simulator currently owns `failed` as its only explicit
  // attention state. Do not invent a waiting state in the HUD; future stream
  // adapters can extend this predicate when the model exposes one.
  const attentionActors = actors.filter((actor) => actor.state === 'failed')
  const healthCounts = {
    healthy: actors.length - attentionActors.length,
    attention: attentionActors.length,
    total: actors.length,
  }
  const mapBounds = bounds ?? { center: [0, 0] as const, width: 100, depth: 100, footprint: { center: [0, 0] as const, width: 100, depth: 100 } }
  const worldWidth = Math.max(mapBounds.width ?? mapBounds.footprint.width, 1)
  const worldDepth = Math.max(mapBounds.depth ?? mapBounds.footprint.depth, 1)
  const worldCenter = mapBounds.center ?? mapBounds.footprint.center
  const project = (position: readonly [number, number]) => ({
    x: 50 + ((position[0] - worldCenter[0]) / worldWidth) * 86,
    y: 50 + ((position[1] - worldCenter[1]) / worldDepth) * 86,
  })
  const terrainCells = terrain && biomes ? (() => {
    const stride = Math.max(1, Math.ceil(Math.max(terrain.cols, terrain.rows) / 32))
    const cells: Array<{ x: number; z: number; biome: number; path: number }> = []
    for (let row = 0; row < terrain.rows; row += stride) for (let col = 0; col < terrain.cols; col += stride) {
      const index = row * terrain.cols + col
      cells.push({ x: terrain.originX + col * terrain.cellSize, z: terrain.originZ + row * terrain.cellSize, biome: biomes[index] ?? 0, path: pathMask?.[index] ?? 0 })
    }
    return { cells, stride }
  })() : null
  const selectedActor = focusedId ? actors.find((actor) => actor.id === focusedId) : undefined
  const actorMarkerLayout = (() => {
    const groups = new Map<string, Array<{ actor: ActorView; anchorX: number; anchorY: number }>>()
    for (const actor of actors) {
      const anchor = project(actor.position)
      // Group only genuinely crowded points. The grouping is visual-only; the
      // anchor remains the authoritative simulator position.
      const key = `${Math.round(anchor.x / 3)}:${Math.round(anchor.y / 3)}`
      const group = groups.get(key) ?? []
      group.push({ actor, anchorX: anchor.x, anchorY: anchor.y })
      groups.set(key, group)
    }
    const raw = [...groups.values()].flatMap((group) => group.map((entry, index) => {
      if (group.length === 1) return { ...entry, markerX: entry.anchorX, markerY: entry.anchorY, offset: false }
      // A single small ring still collapses into a knot when a seeded roster
      // shares one room. Use deterministic concentric rings with a readable
      // minimum arc spacing. The marker is a visual index only; the leader
      // line and the exact list preserve the authoritative simulator position.
      const ringSpacing = 5
      const ringRadius = (ring: number) => 5 + ring * ringSpacing
      let ring = 1
      let remaining = index
      while (remaining > 0) {
        const capacity = Math.max(6, Math.floor((Math.PI * 2 * ringRadius(ring)) / 5))
        if (remaining < capacity) break
        remaining -= capacity
        ring += 1
      }
      const capacity = Math.max(6, Math.floor((Math.PI * 2 * ringRadius(ring)) / 5))
      const angle = (Math.PI * 2 * remaining) / capacity - Math.PI / 2
      const radius = ringRadius(ring)
      return {
        ...entry,
        markerX: entry.anchorX + Math.cos(angle) * radius,
        markerY: entry.anchorY + Math.sin(angle) * radius,
        offset: true,
      }
    }))
    // Room-level fan-out is not enough for a real roster: actors in adjacent
    // seats can still overlap even when they are not in the same 3% bucket.
    // Resolve collisions across the whole projected map with deterministic
    // nearest candidates. The simulator position remains the anchor; only the
    // displayed marker moves and receives a leader line.
    const placed: Array<{ x: number; y: number }> = []
    const markerGap = 5.5
    const candidateFor = (entry: (typeof raw)[number]) => {
      const candidates = [{ x: entry.markerX, y: entry.markerY }]
      for (let ring = 1; ring <= 8; ring += 1) {
        const radius = ring * 5.5
        const slots = Math.max(8, Math.round((Math.PI * 2 * radius) / markerGap))
        for (let slot = 0; slot < slots; slot += 1) {
          const angle = (Math.PI * 2 * slot) / slots - Math.PI / 2
          candidates.push({ x: entry.anchorX + Math.cos(angle) * radius, y: entry.anchorY + Math.sin(angle) * radius })
        }
      }
      return candidates.find((candidate) => {
        const bounded = candidate.x >= 7 && candidate.x <= 93 && candidate.y >= 7 && candidate.y <= 93
        return bounded && placed.every((other) => Math.hypot(candidate.x - other.x, candidate.y - other.y) >= markerGap)
      }) ?? { x: Math.max(7, Math.min(93, entry.markerX)), y: Math.max(7, Math.min(93, entry.markerY)) }
    }
    return raw.map((entry) => {
      const candidate = candidateFor(entry)
      placed.push(candidate)
      return { ...entry, markerX: candidate.x, markerY: candidate.y, offset: entry.offset || candidate.x !== entry.markerX || candidate.y !== entry.markerY }
    })
  })()
  const biomeColours = ['#315c48', '#557a48', '#806b42', '#496a82', '#254f68', '#6b7280']
  const waterPolygons = waterGeometry.flatMap((water) => {
    const points: string[] = []
    for (let index = 0; index < water.positions.length; index += 3) {
      const point = project([water.positions[index] ?? 0, water.positions[index + 2] ?? 0])
      points.push(`${point.x},${point.y}`)
    }
    return points.length >= 3 ? [points.join(' ')] : []
  })
  const byTeam = new Map<string, ActorView[]>()
  const unassigned: ActorView[] = []
  for (const actor of actors) {
    if (!actor.teamId) {
      unassigned.push(actor)
      continue
    }
    const list = byTeam.get(actor.teamId) ?? []
    list.push(actor)
    byTeam.set(actor.teamId, list)
  }
  const groups: Array<{ id: string; label: string; members: ActorView[] }> = teams
    .map((team) => ({ id: team.id, label: team.label, members: byTeam.get(team.id) ?? [] }))
    .filter((g) => g.members.length > 0)
  if (unassigned.length > 0) groups.push({ id: 'commons', label: 'Commons', members: unassigned })
  const denseOverview = actors.length >= 18 || places.length >= 12
  const visiblePlaceLabels = (() => {
    const candidates = places
      .filter((place) => place.kind === 'room' || place.kind === 'gathering')
      .map((place) => {
        const point = project(place.position)
        return {
          id: place.id,
          x: point.x,
          y: point.y - Math.max(6, (place.size[1] / Math.max(mapBounds.footprint.depth, 1)) * 82) / 2 - 1,
          width: Math.max(8, Math.min(24, place.label.length * 0.42 + 2)),
          height: 3.8,
          priority: place.kind === 'gathering' ? 2 : 1,
        }
      })
      .sort((a, b) => b.priority - a.priority || a.y - b.y || a.x - b.x || a.id.localeCompare(b.id))
    const accepted: typeof candidates = []
    for (const candidate of candidates) {
      // At live-roster density, a complete label layer turns the map into a
      // wall of text. Keep one high-value orientation label and move exact
      // identity into the activity rail, where it remains searchable and
      // actionable.
      if (denseOverview) continue
      const overlaps = accepted.some((other) => Math.abs(candidate.x - other.x) < (candidate.width + other.width) / 2 + 1 && Math.abs(candidate.y - other.y) < (candidate.height + other.height) / 2 + 1)
      if (!overlaps) accepted.push(candidate)
    }
    return new Set(accepted.map((candidate) => candidate.id))
  })()
  const zoomCoordinate = (value: number) => 50 + (value - 50) * mapZoom
  const feedNotice = feedMode === 'connecting'
    ? 'Connecting to the live world feed…'
    : feedMode === 'polling'
      ? 'Fallback polling active · freshness may lag'
      : feedMode === 'stopped'
        ? 'Live feed stopped · positions may be stale'
        : feedMode === 'snapshot'
          ? 'Deterministic snapshot · live feed disabled'
        : null

  return (
    <div className="grid h-full min-h-0 items-start gap-3 overflow-auto p-3 lg:grid-cols-[minmax(0,1fr)_20rem] lg:overflow-hidden" data-testid={selectors.world.hud.twoDMode}>
      <section className="relative min-h-[320px] overflow-hidden rounded-2xl border border-border bg-slate-950/80 shadow-inner sm:min-h-[360px] lg:h-[620px] lg:min-h-[400px]" aria-label="2D world map">
        <div className="absolute inset-0 opacity-40" style={{ backgroundImage: 'linear-gradient(rgba(148,163,184,.12) 1px, transparent 1px), linear-gradient(90deg, rgba(148,163,184,.12) 1px, transparent 1px)', backgroundSize: '32px 32px' }} aria-hidden="true" />
        <div className="absolute inset-[7%] rounded-[2rem] border border-emerald-300/20 bg-emerald-950/20" aria-hidden="true" />
        <svg className="absolute inset-[7%] h-[86%] w-[86%] overflow-visible transition-transform duration-200" style={{ transform: `scale(${mapZoom})`, transformOrigin: 'center' }} viewBox="0 0 100 100" aria-label="Terrain, water, navigation, places, and agent positions">
          {terrainCells?.cells.map((cell, index) => {
            const point = project([cell.x, cell.z])
            const size = Math.max(1, (terrainCells.stride * (terrain?.cellSize ?? 1) / worldWidth) * 86)
            return <rect key={`terrain-${index}`} x={point.x - size / 2} y={point.y - size / 2} width={size} height={size} fill={biomeColours[cell.biome % biomeColours.length]} opacity={cell.path > 0.3 ? 0.9 : 0.58} />
          })}
          {waterPolygons.map((points, index) => <polygon key={`water-${index}`} points={points} fill="#2b6f9e" opacity=".75" stroke="#8bd5ff" strokeWidth=".35" />)}
          {nav && Array.from({ length: nav.rows }).flatMap((_, row) => Array.from({ length: nav.cols }, (_, col) => {
            const index = row * nav.cols + col
            if (nav.walkable[index] !== 0 || (row + col) % Math.max(1, Math.ceil(Math.max(nav.rows, nav.cols) / 32)) !== 0) return null
            const point = project([nav.originX + (col + 0.5) * nav.cellSize, nav.originZ + (row + 0.5) * nav.cellSize])
            return <rect key={`blocked-${index}`} x={point.x - 0.4} y={point.y - 0.4} width="0.8" height="0.8" fill="#f97316" opacity=".35" />
          }).filter(Boolean))}
          {places.filter((place) => place.kind === 'corridor').map((place) => {
            const point = project(place.position)
            const width = Math.max(2, (place.size[0] / Math.max(mapBounds.footprint.width, 1)) * 82)
            const height = Math.max(2, (place.size[1] / Math.max(mapBounds.footprint.depth, 1)) * 82)
            return <rect key={place.id} x={point.x - width / 2} y={point.y - height / 2} width={width} height={height} rx="1" fill="rgba(148,163,184,.24)" stroke="rgba(203,213,225,.45)" />
          })}
          {places.filter((place) => (place.kind === 'room' || place.kind === 'gathering') && visiblePlaceLabels.has(place.id)).map((place) => {
            const point = project(place.position)
            const width = Math.max(8, (place.size[0] / Math.max(mapBounds.footprint.width, 1)) * 82)
            const height = Math.max(6, (place.size[1] / Math.max(mapBounds.footprint.depth, 1)) * 82)
            return <g key={place.id} aria-label={place.label}><rect x={point.x - width / 2} y={point.y - height / 2} width={width} height={height} rx="2" fill={place.kind === 'gathering' ? 'rgba(251,191,36,.18)' : 'rgba(59,130,246,.18)'} stroke={place.kind === 'gathering' ? 'rgba(252,211,77,.75)' : 'rgba(147,197,253,.7)'} /><text data-place-label={place.id} x={point.x} y={point.y - height / 2 - 1} textAnchor="middle" fill="rgba(241,245,249,.95)" fontSize="2.7" paintOrder="stroke" stroke="#0f172a" strokeOpacity=".9" strokeWidth=".85" strokeLinecap="round" strokeLinejoin="round">{place.label.slice(0, 18)}</text></g>
          })}
        </svg>
        {actorMarkerLayout.some(({ offset }) => offset) && <svg className="pointer-events-none absolute inset-0 h-full w-full" viewBox="0 0 100 100" aria-hidden="true">
          {actorMarkerLayout.filter(({ offset }) => offset).map(({ actor, anchorX, anchorY, markerX, markerY }) => (
            <line key={`leader-${actor.id}`} x1={Math.max(5, Math.min(95, zoomCoordinate(anchorX)))} y1={Math.max(9, Math.min(91, zoomCoordinate(anchorY)))} x2={Math.max(5, Math.min(95, zoomCoordinate(markerX)))} y2={Math.max(9, Math.min(91, zoomCoordinate(markerY)))} stroke="rgba(226,232,240,.55)" strokeWidth=".35" strokeDasharray="1 1" />
          ))}
        </svg>}
        <div className="absolute left-3 top-3 max-w-[13rem] rounded-lg border border-white/10 bg-black/60 px-3 py-2 text-[11px] text-slate-200 backdrop-blur sm:max-w-none sm:text-xs"><p className="font-semibold">Operational map</p><p className="mt-1 text-slate-400">{biomeSetId ? `${biomeSetId} terrain · ` : ''}live positions · rooms · paths</p></div>
        {feedNotice && <p className="absolute left-3 top-[4.75rem] max-w-[15rem] rounded-lg border border-amber-300/30 bg-amber-950/70 px-3 py-2 text-[11px] text-amber-100 shadow-sm" role="status" aria-live="polite">{feedNotice}</p>}
        <div className="absolute right-3 top-3 flex items-center gap-1 rounded-lg border border-white/10 bg-black/60 p-1 backdrop-blur" aria-label="Map camera controls">
          <button type="button" className="grid h-8 w-8 place-items-center rounded-md text-sm text-slate-100 hover:bg-white/10" aria-label="Zoom out map" onClick={() => setMapZoom((value) => Math.max(0.85, Number((value - 0.15).toFixed(2))))}>−</button>
          <button type="button" className="grid h-8 min-w-10 place-items-center rounded-md px-1 text-[11px] font-medium text-slate-200 hover:bg-white/10" aria-label="Recenter map" onClick={() => setMapZoom(1)}>{Math.round(mapZoom * 100)}%</button>
          <button type="button" className="grid h-8 w-8 place-items-center rounded-md text-sm text-slate-100 hover:bg-white/10" aria-label="Zoom in map" onClick={() => setMapZoom((value) => Math.min(1.45, Number((value + 0.15).toFixed(2))))}>+</button>
        </div>
        {places.length > visiblePlaceLabels.size && <p className="absolute right-3 bottom-14 max-w-[15rem] rounded bg-black/70 px-2 py-1 text-right text-[10px] text-slate-300 sm:bottom-3">{denseOverview ? 'Dense roster · use the activity list for exact identity.' : 'Overview labels decluttered · use the activity list for exact identity.'}</p>}
        {selectedActor && <div className="absolute right-3 top-3 max-w-[12rem] rounded-lg border border-primary/40 bg-black/70 px-3 py-2 text-xs text-slate-100 backdrop-blur"><p className="font-semibold">Selected agent</p><p className="mt-1">{selectedActor.name}</p><p className="text-slate-400">{STATE_LABEL[selectedActor.state]} · {selectedActor.teamId ?? 'Unassigned'}</p></div>}
        <div className="absolute bottom-3 left-3 flex flex-wrap gap-2 rounded-lg border border-white/10 bg-black/60 px-3 py-2 text-[10px] text-slate-300 backdrop-blur"><span><i className="mr-1 inline-block h-2 w-2 rounded-sm bg-blue-300/70" />Team room</span><span><i className="mr-1 inline-block h-2 w-2 rounded-sm bg-amber-300/80" />Gathering</span><span><i className="mr-1 inline-block h-2 w-2 rounded-sm bg-sky-300/80" />Water</span><span><i className="mr-1 inline-block h-2 w-2 rounded-sm bg-orange-400/80" />Blocked</span><span><i className="mr-1 inline-block h-2 w-2 rounded-full bg-white/80" />Agent</span></div>
        {actorMarkerLayout.map(({ actor, markerX, markerY, offset }) => {
          const label = `${actor.name} · ${STATE_LABEL[actor.state]}${offset ? ' · displayed offset for dense map' : ''}`
          return <button key={actor.id} type="button" aria-label={`Focus ${actor.name}`} title={label} aria-pressed={focusedId === actor.id} onClick={() => onFocus(focusedId === actor.id ? null : actor.id)} className={`absolute grid ${denseOverview ? 'h-5 w-5 text-[8px] sm:h-6 sm:w-6 sm:text-[9px]' : 'h-6 w-6 text-[9px] sm:h-7 sm:w-7'} -translate-x-1/2 -translate-y-1/2 place-items-center rounded-full border font-bold text-white shadow-lg transition-transform hover:scale-125 ${focusedId === actor.id ? 'z-10 scale-125 border-white ring-4 ring-primary/40' : 'border-white/60'}`} style={{ left: `${Math.max(5, Math.min(95, zoomCoordinate(markerX)))}%`, top: `${Math.max(9, Math.min(91, zoomCoordinate(markerY)))}%`, backgroundColor: actor.colors.body }}>{actor.name.slice(0, 1).toUpperCase()}</button>
        })}
        {actors.length === 0 && <div className="absolute inset-0 grid place-items-center px-6 text-center"><div><p className="text-sm font-medium text-slate-200">{totalActors === 0 ? 'No agents are connected yet' : 'No agents match these filters'}</p><p className="mt-1 max-w-xs text-xs text-slate-400">{totalActors === 0 ? 'When an agent joins the roster, its position and team will appear here.' : filtersActive ? 'Clear or broaden the active filters to restore the world roster.' : 'The current world snapshot contains no visible agents.'}</p></div></div>}
        {!terrainCells && <p className="absolute right-3 bottom-3 rounded bg-black/70 px-2 py-1 text-[10px] text-amber-200">World layers are still loading.</p>}
      </section>
      <aside aria-label="2D world activity" className="min-w-0 space-y-3 lg:max-h-[620px] lg:overflow-y-auto lg:pr-1">
        {weather && <p className="rounded-xl border border-border bg-background/70 p-3 text-sm text-muted-foreground" data-testid={selectors.world.hud.weather}><span className="font-medium capitalize text-foreground">{weather.state}</span> — health pressure {Math.round(weather.pressure * 100)}%.</p>}
        <section className="rounded-xl border border-border bg-background/80 p-3 shadow-sm" aria-label="World health distribution">
          <div className="flex items-start justify-between gap-3">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.14em] text-primary">Health distribution</p>
              <p className="mt-1 text-xs text-muted-foreground">A compact read of the visible roster.</p>
            </div>
            <span className="text-xs font-semibold tabular-nums text-foreground">{healthCounts.total} visible</span>
          </div>
          <div className="mt-3 flex h-2 overflow-hidden rounded-full bg-muted" role="img" aria-label={`${healthCounts.healthy} healthy and ${healthCounts.attention} needing attention`}>
            {healthCounts.healthy > 0 && <span className="bg-emerald-500" style={{ width: `${(healthCounts.healthy / Math.max(healthCounts.total, 1)) * 100}%` }} />}
            {healthCounts.attention > 0 && <span className="bg-amber-500" style={{ width: `${(healthCounts.attention / Math.max(healthCounts.total, 1)) * 100}%` }} />}
          </div>
          <div className="mt-2 flex flex-wrap gap-x-3 gap-y-1 text-[11px] text-muted-foreground">
            <span><i className="mr-1 inline-block h-2 w-2 rounded-full bg-emerald-500" />{healthCounts.healthy} healthy</span>
            <span><i className="mr-1 inline-block h-2 w-2 rounded-full bg-amber-500" />{healthCounts.attention} attention</span>
          </div>
        </section>
        <section className="rounded-xl border border-border bg-background/80 p-3 shadow-sm" aria-label="World attention queue">
          <div className="flex items-start justify-between gap-3">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.14em] text-primary">Attention queue</p>
              <p className="mt-1 text-xs text-muted-foreground">Failed and waiting signals need review.</p>
            </div>
            <span className="rounded-full border border-amber-500/30 bg-amber-500/10 px-2 py-0.5 text-[11px] font-semibold text-amber-700 dark:text-amber-300">{healthCounts.attention}</span>
          </div>
          {attentionActors.length === 0 ? (
            <p className="mt-3 rounded-lg bg-muted/50 px-3 py-2 text-xs text-muted-foreground">No active attention signals in the visible roster.</p>
          ) : (
            <ul className="mt-3 space-y-1.5">
              {attentionActors.slice(0, 4).map((actor) => (
                <li key={actor.id}>
                  <button type="button" onClick={() => onFocus(actor.id)} className="flex min-h-9 w-full items-center justify-between gap-3 rounded-lg px-2 text-left text-xs hover:bg-muted" aria-label={`Review ${actor.name}`}>
                    <span className="min-w-0 truncate font-medium text-foreground">{actor.name}</span>
                    <span className="shrink-0 capitalize text-amber-700 dark:text-amber-300">{STATE_LABEL[actor.state]}</span>
                  </button>
                </li>
              ))}
            </ul>
          )}
          {attentionActors.length > 4 && <p className="mt-2 text-[11px] text-muted-foreground">+{attentionActors.length - 4} more in the activity list.</p>}
        </section>
        <ul className="space-y-3" data-testid={selectors.world.hud.actorList} aria-label="Agents by team">
        {groups.length === 0 && <li className="text-sm text-muted-foreground">No agents match.</li>}
        {groups.map((group) => (
          <li key={group.id}>
            <h3 className="mb-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">{group.label}</h3>
            <ul className="divide-y divide-border rounded-md border border-border bg-background/70">
              {group.members.map((actor) => (
                <li key={actor.id}>
                  <button
                    type="button"
                    aria-pressed={focusedId === actor.id}
                    onClick={() => onFocus(focusedId === actor.id ? null : actor.id)}
                    className={`flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm hover:bg-muted ${focusedId === actor.id ? 'bg-muted' : ''}`}
                    data-testid={`${selectors.world.hud.actorList}-${actor.id}`}
                  >
                    <span className="flex items-center gap-2">
                      <span className="inline-block h-3 w-3 rounded-full" style={{ backgroundColor: actor.colors.body }} aria-hidden="true" />
                      <span className="font-medium">{actor.name}</span>
                      {actor.memberType === 'contractor' && <span className="rounded border border-dashed border-amber-500/50 px-1 text-[10px] text-amber-600 dark:text-amber-400">Contractor</span>}
                    </span>
                    <span className="flex items-center gap-3 text-xs text-muted-foreground">
                      <span>{STATE_LABEL[actor.state]}</span>
                      <span className="tabular-nums">{formatDuration(now - actor.stateSince)}</span>
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          </li>
        ))}
        </ul>
      </aside>
    </div>
  )
}
