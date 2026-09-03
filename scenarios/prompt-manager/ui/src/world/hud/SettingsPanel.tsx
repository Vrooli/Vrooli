import { QUALITY_PROFILE_IDS, PERIOD_IDS, SCENE_IDS, scenes, type PeriodId, type QualityProfileId, type QualityState, type SceneId, type TuningOverride, type WorldTuning } from '../config'
import { LeversPanel } from './LeversPanel'
import { selectors } from '@/constants/selectors'

export type PeriodMode = { kind: 'clock' } | { kind: 'fixed'; period: PeriodId }

export interface WorldSettingsContentProps {
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
  /** Present only in development builds: live lever editing. */
  levers?: { tuning: WorldTuning; override: TuningOverride; onChange: (override: TuningOverride) => void; onReset: () => void }
}

const PERIOD_LABELS: Record<PeriodId, string> = { dawn: 'Dawn', day: 'Day', dusk: 'Dusk', night: 'Night' }
const PROFILE_LABELS: Record<QualityProfileId, string> = { low: 'Low', medium: 'Medium', high: 'High', ultra: 'Ultra' }

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
  levers,
}: WorldSettingsContentProps) {
  return (
    <div className="space-y-4 text-sm" data-testid={selectors.world.settings.popup}>
      <SegmentedControl
        label="Scene"
        value={sceneId}
        options={SCENE_IDS.map((id) => ({ id, label: scenes[id].title }))}
        onChange={onSceneChange}
        testId={selectors.world.settings.scene}
      />
      <SegmentedControl
        label="Quality"
        value={quality.profileId}
        options={QUALITY_PROFILE_IDS.map((id) => ({ id, label: PROFILE_LABELS[id] }))}
        onChange={onPickProfile}
        testId={selectors.world.settings.graphics}
      />
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
        label="Time of day"
        value={periodMode.kind === 'clock' ? 'clock' : periodMode.period}
        options={[{ id: 'clock', label: 'Clock' }, ...PERIOD_IDS.map((id) => ({ id, label: PERIOD_LABELS[id] }))]}
        onChange={(id) => onPeriodModeChange(id === 'clock' ? { kind: 'clock' } : { kind: 'fixed', period: id })}
        testId={selectors.world.settings.period}
      />
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
      {levers && (
        <details className="rounded-md border border-dashed border-border p-2">
          <summary className="cursor-pointer text-xs font-medium uppercase tracking-wide text-muted-foreground">Levers (dev)</summary>
          <div className="mt-2">
            <LeversPanel tuning={levers.tuning} override={levers.override} onChange={levers.onChange} onReset={levers.onReset} />
          </div>
        </details>
      )}
    </div>
  )
}
