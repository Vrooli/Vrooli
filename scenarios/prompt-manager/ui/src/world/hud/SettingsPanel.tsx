import { choiceSettings, integerSettings, parseIntegerSetting } from '../config/settings'
import { type WeatherId, type PeriodId, type QualityProfileId, type QualityState, type SceneId, type TuningOverride, type WorldTuning } from '../config'
import { useEffect, useState } from 'react'
import type { WorldClock } from '../config/clock'
import { LeversPanel } from './LeversPanel'
import { WorkbenchStatus, type WorkbenchSnapshot } from './WorkbenchStatus'
import { selectors } from '@/constants/selectors'
import { METEOR_VARIANTS, WILDLIFE_PREVIEWS, type MeteorVariant, type WildlifePreviewId } from '../config/ambient'

export type PeriodMode = { kind: 'clock' } | { kind: 'fixed'; period: PeriodId }

export interface WorldSettingsContentProps {
  worldClock?: WorldClock
  ambientLife?: { enabled: boolean; onChange: (enabled: boolean) => void }
  skyPreview?: { status: string; onPreview: (variant: MeteorVariant | 'comet' | 'off') => void }
  wildlifePreview?: { status: string; onPreview: (kind: WildlifePreviewId) => void }
  seed: number
  onSeedChange: (seed: number) => void
  weather: WeatherId | 'auto'
  onWeatherChange: (weather: WeatherId | 'auto') => void
  navigationOverlay?: { enabled: boolean; onChange: (enabled: boolean) => void; cells: number; status: string }
  biomeOverlay?: { enabled: boolean; onChange: (enabled: boolean) => void; status: string; legend: Array<{ label: string; color: string }> }
  habitatOverlay?: { enabled: boolean; onChange: (enabled: boolean) => void; status: string; legend: Array<{ label: string; color: string }> }
  cameraCollisionOverlay?: { enabled: boolean; onChange: (enabled: boolean) => void; status: string }
  sceneId: SceneId
  onSceneChange: (scene: SceneId) => void
  quality: QualityState
  onPickProfile: (profile: QualityProfileId) => void
  onAutoChange: (auto: boolean) => void
  periodMode: PeriodMode
  onPeriodModeChange: (mode: PeriodMode) => void
  showDiagnostics: boolean
  onShowDiagnosticsChange: (show: boolean) => void
  onCameraHome: () => void
  zoomTarget: 'cursor' | 'center'
  onZoomTargetChange: (target: 'cursor' | 'center') => void
  /** Opt-in development workbench; overrides apply only to this mounted tab. */
  levers?: { tuning: WorldTuning; override: TuningOverride; onChange: (override: TuningOverride) => void; onReset: () => void; exportRecipe?: () => void; importRecipe?: (text: string) => void; readStatus?: () => WorkbenchSnapshot }
}

function SegmentedControl<T extends string>({
  label,
  value,
  options,
  onChange,
  testId,
}: {
  label: string
  value: T
  options: ReadonlyArray<{ id: T; label: string }>
  onChange: (value: T) => void
  testId: string
}) {
  return (
    <fieldset className="space-y-1.5">
      <legend className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{label}</legend>
      <div className="flex flex-wrap gap-1" role="radiogroup" aria-label={label} data-testid={testId}>
        {options.map((option) => (
          <button
            key={option.id}
            type="button"
            role="radio"
            aria-checked={option.id === value}
            data-testid={`${testId}-${option.id}`}
            onClick={() => onChange(option.id)}
            className={
              option.id === value
                ? 'rounded-md bg-primary px-2.5 py-1 text-xs font-medium text-primary-foreground'
                : 'rounded-md border border-border px-2.5 py-1 text-xs font-medium text-muted-foreground hover:bg-muted hover:text-foreground'
            }
          >
            {option.label}
          </button>
        ))}
      </div>
    </fieldset>
  )
}

/** Scene, quality, time-of-day, camera and diagnostics controls for the world. */
export function WorldSettingsContent({
  worldClock,
  ambientLife,
  skyPreview,
  wildlifePreview,
  seed,
  onSeedChange,
  weather,
  onWeatherChange,
  navigationOverlay,
  biomeOverlay,
  habitatOverlay,
  cameraCollisionOverlay,
  sceneId,
  onSceneChange,
  quality,
  onPickProfile,
  onAutoChange,
  periodMode,
  onPeriodModeChange,
  showDiagnostics,
  onShowDiagnosticsChange,
  onCameraHome,
  zoomTarget,
  onZoomTargetChange,
  levers,
}: WorldSettingsContentProps) {
  const [seedDraft, setSeedDraft] = useState<string | null>(null)
  const [seedError, setSeedError] = useState<string | null>(null)
  const [exportError, setExportError] = useState<string | null>(null)
  const [workbenchOpen, setWorkbenchOpen] = useState(false)
  const applySeed = () => {
    const raw = seedDraft ?? String(seed)
    const { value: next, error } = parseIntegerSetting(integerSettings.seed, raw)
    if (error) {
      setSeedError(error)
      return
    }
    setSeedError(null)
    setSeedDraft(null)
    if (next !== seed) onSeedChange(next)
  }
  return (
    <div className="space-y-4 text-sm" data-testid={selectors.world.settings.popup}>
      <SegmentedControl
        label={choiceSettings.scene.label}
        value={sceneId}
        options={choiceSettings.scene.choices}
        onChange={onSceneChange}
        testId={selectors.world.settings.scene}
      />
      <SegmentedControl
        label={choiceSettings.profile.label}
        value={quality.profileId}
        options={choiceSettings.profile.choices}
        onChange={onPickProfile}
        testId={selectors.world.settings.graphics}
      />
      <div className="space-y-1.5">
        <label htmlFor="world-seed" className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{integerSettings.seed.label}</label>
        <div className="flex items-center gap-2">
          <input id="world-seed" type="number" min={integerSettings.seed.minimum} max={integerSettings.seed.maximum} step="1"
            value={seedDraft ?? String(seed)} onChange={event => setSeedDraft(event.target.value)}
            onKeyDown={event => { if (event.key === 'Enter') { event.preventDefault(); applySeed() } }}
            aria-invalid={Boolean(seedError)} aria-describedby={seedError ? 'world-seed-error world-seed-help' : 'world-seed-help'}
            className="min-w-0 w-36 rounded-md border border-border bg-background px-2 py-1" />
          <button type="button" onClick={applySeed} className="rounded-md border border-border px-2.5 py-1 text-xs hover:bg-muted">Apply seed</button>
        </div>
        <p id="world-seed-help" className="text-xs text-muted-foreground">{integerSettings.seed.description}</p>
        {seedError && <p id="world-seed-error" role="alert" className="text-xs text-red-600 dark:text-red-400">{seedError}</p>}
      </div>
      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={quality.auto}
          onChange={(event) => onAutoChange(event.target.checked)}
          data-testid={selectors.world.settings.qualityAuto}
        />
        <span>Adjust quality automatically</span>
      </label>
      <SegmentedControl<'clock' | PeriodId>
        label={choiceSettings.period.label}
        value={periodMode.kind === 'clock' ? 'clock' : periodMode.period}
        options={choiceSettings.period.choices}
        onChange={(id) => onPeriodModeChange(id === 'clock' ? { kind: 'clock' } : { kind: 'fixed', period: id })}
        testId={selectors.world.settings.period}
      />
      <div className="space-y-1.5">
        <SegmentedControl<WeatherId | 'auto'> label={choiceSettings.weather.label} value={weather}
          options={choiceSettings.weather.choices}
          onChange={onWeatherChange} testId="world-settings-weather" />
        <p className="text-xs text-muted-foreground">{choiceSettings.weather.description}</p>
      </div>
      <fieldset className="space-y-1.5">
        <legend className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Zoom toward</legend>
        <div className="flex gap-3">
          {(['cursor', 'center'] as const).map(target => (
            <label key={target} className="flex items-center gap-1.5">
              <input type="radio" name="world-zoom-target" checked={zoomTarget === target} onChange={() => onZoomTargetChange(target)} />
              {target === 'cursor' ? 'Cursor' : 'Center'}
            </label>
          ))}
        </div>
      </fieldset>
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={onCameraHome}
          className="rounded-md border border-border px-2.5 py-1 text-xs font-medium hover:bg-muted"
          data-testid={selectors.world.settings.camera}
        >
          Reset camera
        </button>
        <label className="flex items-center gap-2 text-xs">
          <input
            type="checkbox"
            checked={showDiagnostics}
            onChange={(event) => onShowDiagnosticsChange(event.target.checked)}
            data-testid={selectors.world.settings.diagnosticsToggle}
          />
          <span>Show diagnostics</span>
        </label>
      </div>
      {ambientLife && <label className="flex items-center gap-2 text-xs"><input type="checkbox" checked={ambientLife.enabled} onChange={event => ambientLife.onChange(event.target.checked)} />Ambient life</label>}
      {levers && (
        <details className="rounded-md border border-dashed border-border p-2" onToggle={event => setWorkbenchOpen(event.currentTarget.open)}>
          <summary className="cursor-pointer text-xs font-medium uppercase tracking-wide text-muted-foreground">World workbench</summary>
          <p className="mt-2 text-xs text-muted-foreground">Changes apply to this tab and reset when the page reloads.</p>
          {worldClock && <ClockControls clock={worldClock} />}
          {skyPreview && <div className="mt-2 space-y-1 text-xs">
            <p>Sky event preview</p>
            <div className="flex flex-wrap gap-1">{[...METEOR_VARIANTS, 'comet', 'off'].map(variant => <button key={variant} type="button" className="rounded border px-2 py-1"
              onClick={() => skyPreview.onPreview(variant as MeteorVariant | 'comet' | 'off')}>{variant === 'off' ? 'End preview' : `Preview ${variant}`}</button>)}</div>
            <p>Shows the event in front of the camera. Respects ambient life and reduced motion.</p>
            <p role="status">{skyPreview.status}</p>
          </div>}
          {wildlifePreview && <div className="mt-2 space-y-1 text-xs">
            <p>Wildlife preview</p>
            <div className="flex flex-wrap gap-1">{WILDLIFE_PREVIEWS.map(preview => <button key={preview.id} type="button" className="rounded border px-2 py-1"
              onClick={() => wildlifePreview.onPreview(preview.id)}>Preview {preview.label}</button>)}</div>
            <p>Focuses eligible habitat, freezes time and selects clear weather and suitable lighting. Play from here to animate the selected instant. Respects ambient life and reduced motion.</p>
            <p role="status">{wildlifePreview.status}</p>
          </div>}
          {navigationOverlay && <div className="mt-2 space-y-1 text-xs">
            <label className="flex items-center gap-2"><input type="checkbox" checked={navigationOverlay.enabled}
              onChange={event => navigationOverlay.onChange(event.target.checked)} />Navigation overlay</label>
            {navigationOverlay.enabled && <p>{navigationOverlay.cells.toLocaleString()} cell markers · green: walkable · red: blocked. Markers show through structures.</p>}
            {navigationOverlay.enabled && <p role="status">{navigationOverlay.status}</p>}
          </div>}
          {([['Biome overlay', biomeOverlay], ['Habitat overlay', habitatOverlay]] as const).map(([label, overlay]) => overlay && <div key={label} className="mt-2 space-y-1 text-xs">
            <label className="flex items-center gap-2"><input type="checkbox" checked={overlay.enabled}
              onChange={event => overlay.onChange(event.target.checked)} />{label}</label>
            {overlay.enabled && <>
              <div className="flex flex-wrap gap-2">{overlay.legend.map(entry => <span key={entry.label} className="flex items-center gap-1">
                <span aria-hidden="true" className="inline-block h-3 w-3 rounded" style={{ backgroundColor: entry.color }} />{entry.label}
              </span>)}</div>
              <p role="status">{overlay.status}</p>
              <p>{label === 'Biome overlay' ? 'Markers show committed biome assignments at terrain samples, including beneath water and structures.' : 'Markers show committed habitat eligibility. Gray means excluded; movement still follows navigation.'}</p>
            </>}
          </div>)}
          {cameraCollisionOverlay && <div className="mt-2 space-y-1 text-xs">
            <label className="flex items-center gap-2"><input type="checkbox" checked={cameraCollisionOverlay.enabled}
              onChange={event => cameraCollisionOverlay.onChange(event.target.checked)} />Camera collision overlay</label>
            {cameraCollisionOverlay.enabled && <><p role="status">{cameraCollisionOverlay.status}</p>
              <p>Orange outlines show committed structural boxes expanded for camera clearance. Terrain and furniture interaction bounds are separate.</p></>}
          </div>}
          <button type="button" disabled={!levers.exportRecipe} className="mt-2 rounded border border-border px-2 py-1 text-xs disabled:opacity-50"
            onClick={() => {
              try { levers.exportRecipe?.(); setExportError(null) }
              catch (error) { setExportError(error instanceof Error ? error.message : 'Recipe export failed.') }
            }}>Export synthetic recipe</button>
          <p className="mt-1 text-xs text-muted-foreground">Available after a synthetic world is ready. Includes tuning, layout and renderer details.</p>
          {exportError && <p role="alert" className="text-xs text-red-600 dark:text-red-400">{exportError}</p>}
          {levers.importRecipe && <label className="mt-2 block text-xs">Import synthetic recipe
            <input type="file" accept="application/json,.json" className="mt-1 block max-w-full" onChange={event => {
              const file = event.target.files?.[0]
              event.target.value = ''
              if (!file) return
              void (async () => { try {
                if (file.size > 1024 * 1024) throw new Error('Recipe files must be at most 1 MiB.')
                levers.importRecipe?.(await file.text())
                setExportError(null)
              } catch (error) { setExportError(error instanceof Error ? error.message : 'Recipe import failed.') } })()
            }} />
            <span className="mt-1 block text-muted-foreground">Opens the reconstructed world. The imported recipe stays in this tab across reloads.</span>
          </label>}
          <div className="mt-2">
            <LeversPanel tuning={levers.tuning} override={levers.override} onChange={levers.onChange} onReset={levers.onReset} />
          </div>
          {levers.readStatus && <WorkbenchStatus read={levers.readStatus} active={workbenchOpen} />}
        </details>
      )}
    </div>
  )
}

function ClockControls({ clock }: { clock: WorldClock }) {
  const [snapshot, setSnapshot] = useState(() => clock.snapshot())
  const [draft, setDraft] = useState(() => new Date(clock.snapshot().utcMilliseconds).toISOString().slice(0, 19))
  const [error, setError] = useState('')
  useEffect(() => clock.subscribe(() => {
    const next = clock.snapshot()
    setSnapshot(next)
    setDraft(new Date(next.utcMilliseconds).toISOString().slice(0, 19))
  }), [clock])
  const seek = () => {
    const instant = Date.parse(`${draft}Z`)
    if (!draft || !Number.isFinite(instant)) { setError('Enter a valid UTC date and time.'); return }
    clock.fix(instant); setError('')
  }
  return <div className="mt-2 space-y-1 text-xs">
    <p>Presentation time: {snapshot.mode === 'clock' ? 'Live' : snapshot.timeScale > 0 ? 'Playing from selected time' : 'Frozen'}</p>
    <label className="block">UTC instant<input aria-label="UTC instant" type="datetime-local" step="1" value={draft} onChange={event => setDraft(event.target.value)} className="ml-2 rounded border bg-background p-1" /></label>
    <div className="flex flex-wrap gap-1">
      <button type="button" className="rounded border px-2 py-1" onClick={() => clock.fix(clock.snapshot().utcMilliseconds)}>Freeze time</button>
      <button type="button" className="rounded border px-2 py-1" onClick={() => clock.fix(clock.snapshot().utcMilliseconds, 1)}>Play from here</button>
      <button type="button" className="rounded border px-2 py-1" onClick={seek}>Apply UTC instant</button>
      <button type="button" className="rounded border px-2 py-1" onClick={() => clock.fix(clock.snapshot().utcMilliseconds + 60000)}>Advance one minute</button>
      <button type="button" className="rounded border px-2 py-1" onClick={() => clock.live()}>Resume live time</button>
    </div>
    <p>Controls lighting, sky events and wildlife. Resume live time returns to the current wall clock. Civil timezone: {snapshot.timeZone}. Agent activity continues.</p>
    {error && <p role="alert">{error}</p>}
  </div>
}
