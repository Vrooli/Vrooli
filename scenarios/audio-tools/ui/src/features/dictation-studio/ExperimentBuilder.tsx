import { type Dispatch, type SetStateAction } from "react";
import { FlaskConical, Loader2 } from "lucide-react";

import { ApiErrorState } from "../../components/composites/ApiErrorState";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Panel } from "../../components/ui/panel";
import { Select } from "../../components/ui/select";
import { Textarea } from "../../components/ui/textarea";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { type ClipMeta } from "../../services/corpus";
import { type StartExperimentInput } from "../../services/experiment";
import { type SpeakerStatus } from "../../services/speakerAdmin";
import { hasSweepDurations, strategyOptions } from "./ExperimentLabFormat";
import { StrategyName } from "./ExperimentLabShared";

interface QueryState<T> {
  data?: T;
  isPending: boolean;
  isError: boolean;
  error: Error | null;
  refetch: () => unknown;
}

interface ExperimentBuilderProps {
  input: StartExperimentInput;
  setInput: Dispatch<SetStateAction<StartExperimentInput>>;
  pending: boolean;
  onStart: () => void;
  clips: QueryState<ClipMeta[]>;
  speakerStatus: QueryState<SpeakerStatus>;
}

export function ExperimentBuilder({
  input,
  setInput,
  pending,
  onStart,
  clips,
  speakerStatus,
}: ExperimentBuilderProps) {
  const { t } = useTranslation();
  const set = <K extends keyof StartExperimentInput>(key: K, value: StartExperimentInput[K]) =>
    setInput((current) => ({ ...current, [key]: value }));
  const toggleStrategy = (kind: string) =>
    setInput((current) => ({
      ...current,
      strategies: current.strategies.includes(kind)
        ? current.strategies.filter((item) => item !== kind)
        : [...current.strategies, kind],
    }));

  return (
    <Panel title={t(strings.dictationStudio.builderTitle)} description={t(strings.dictationStudio.builderHint)}>
      <div className="grid gap-4 lg:grid-cols-2">
        <label className="flex flex-col gap-1 text-xs lg:col-span-2">
          {t(strings.dictationStudio.experimentNameLabel)}
          <Input data-testid={selectors.dictationStudio.experimentName} value={input.name} onChange={(event) => set("name", event.currentTarget.value)} />
        </label>

        <fieldset className="flex flex-col gap-2 rounded-control border border-app-border p-3">
          <legend className="px-1 text-xs font-medium">{t(strings.dictationStudio.strategiesLabel)}</legend>
          <div className="flex flex-wrap gap-2">
            {strategyOptions.map((kind) => (
              <Button
                key={kind}
                type="button"
                variant={input.strategies.includes(kind) ? "default" : "outline"}
                aria-pressed={input.strategies.includes(kind)}
                onClick={() => toggleStrategy(kind)}
              >
                <StrategyName kind={kind} />
              </Button>
            ))}
          </div>
        </fieldset>

        <fieldset className="flex flex-col gap-2 rounded-control border border-app-border p-3 lg:col-span-2">
          <legend className="px-1 text-xs font-medium">{t(strings.dictationStudio.safetyGateLabel)}</legend>
          <div className="grid gap-2 sm:grid-cols-2">
            <NumberField
              testId={selectors.dictationStudio.experimentDroppedSpanThreshold}
              label={t(strings.dictationStudio.droppedSpanThresholdLabel)}
              value={input.droppedSpanThresholdWords}
              min={0}
              onChange={(value) => set("droppedSpanThresholdWords", value)}
            />
          </div>
          <p className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.droppedSpanThresholdHint)}</p>
        </fieldset>

        <ClipPicker selected={input.clipIds} onChange={(ids) => set("clipIds", ids)} clips={clips} />

        <fieldset className="grid gap-2 rounded-control border border-app-border p-3 sm:grid-cols-2 lg:col-span-2">
          <legend className="px-1 text-xs font-medium">{t(strings.dictationStudio.longFormLabel)}</legend>
          <label className="flex items-center gap-2 text-xs sm:col-span-2">
            <input
              data-testid={selectors.dictationStudio.experimentLongForm}
              type="checkbox"
              checked={input.longForm}
              onChange={(event) => set("longForm", event.currentTarget.checked)}
            />
            {t(strings.dictationStudio.longFormEnabled)}
          </label>
          <NumberField testId={selectors.dictationStudio.experimentSeed} label={t(strings.dictationStudio.seedLabel)} value={input.seed} onChange={(value) => set("seed", value)} />
          <NumberField testId={selectors.dictationStudio.experimentTargetDuration} label={t(strings.dictationStudio.targetDurationLabel)} value={input.targetDurationSeconds} min={0} onChange={(value) => set("targetDurationSeconds", value)} />
          <NumberField testId={selectors.dictationStudio.experimentGapMs} label={t(strings.dictationStudio.gapMsLabel)} value={input.gapMs} min={0} onChange={(value) => set("gapMs", value)} />
          <label className="flex flex-col gap-1 text-xs">
            {t(strings.dictationStudio.tagFilterLabel)}
            <Input data-testid={selectors.dictationStudio.experimentTagContains} value={input.tagContains} onChange={(event) => set("tagContains", event.currentTarget.value)} />
          </label>
          <label className="flex flex-col gap-1 text-xs sm:col-span-2">
            {t(strings.dictationStudio.sweepDurationsLabel)}
            <Input
              data-testid={selectors.dictationStudio.experimentSweepDurations}
              value={input.sweepDurationsCsv}
              onChange={(event) => set("sweepDurationsCsv", event.currentTarget.value)}
              placeholder={t(strings.dictationStudio.sweepPlaceholder)}
            />
            <span className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.sweepDurationsHint)}</span>
          </label>
        </fieldset>

        <fieldset className="grid gap-2 rounded-control border border-app-border p-3 sm:grid-cols-2">
          <legend className="px-1 text-xs font-medium">{t(strings.dictationStudio.hyperparamsLabel)}</legend>
          <div className="flex flex-col gap-1">
            <NumberField testId={selectors.dictationStudio.experimentRealtimeRepeats} label={t(strings.dictationStudio.repeatsLabel)} value={input.realtimeRepeats} min={0} max={20} onChange={(value) => set("realtimeRepeats", value)} />
            {input.realtimeRepeats === 0 ? <p className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.latencyNotMeasured)}</p> : null}
          </div>
          <NumberField testId={selectors.dictationStudio.experimentLatencyTailSeconds} label={t(strings.dictationStudio.latencyTailLabel)} value={input.latencyTailSeconds} min={0} onChange={(value) => set("latencyTailSeconds", value)} />
          <NumberField testId={selectors.dictationStudio.experimentOverlapMaxWindow} label={t(strings.dictationStudio.maxWindowLabel)} value={input.overlapMaxWindowMs} min={0} onChange={(value) => set("overlapMaxWindowMs", value)} />
        </fieldset>

        <fieldset className="grid gap-2 rounded-control border border-app-border p-3 sm:grid-cols-2">
          <legend className="px-1 text-xs font-medium">{t(strings.dictationStudio.augmentationLabel)}</legend>
          <label className="flex flex-col gap-1 text-xs">
            {t(strings.dictationStudio.noiseTypesLabel)}
            <Input data-testid={selectors.dictationStudio.experimentNoiseTypes} value={input.noiseTypesCsv} onChange={(event) => set("noiseTypesCsv", event.currentTarget.value)} placeholder={t(strings.dictationStudio.noisePlaceholder)} />
          </label>
          <label className="flex flex-col gap-1 text-xs">
            {t(strings.dictationStudio.snrLabel)}
            <Input data-testid={selectors.dictationStudio.experimentSnrDb} value={input.snrDbCsv} onChange={(event) => set("snrDbCsv", event.currentTarget.value)} placeholder={t(strings.dictationStudio.snrPlaceholder)} />
          </label>
          <label className="flex flex-col gap-1 text-xs sm:col-span-2">
            {t(strings.dictationStudio.competingVoicesLabel)}
            <Input data-testid={selectors.dictationStudio.experimentCompetingVoices} value={input.competingVoicesCsv} onChange={(event) => set("competingVoicesCsv", event.currentTarget.value)} />
          </label>
          <label className="flex flex-col gap-1 text-xs sm:col-span-2">
            {t(strings.dictationStudio.competingTextLabel)}
            <Textarea rows={2} value={input.competingText} onChange={(event) => set("competingText", event.currentTarget.value)} />
          </label>
        </fieldset>

        <fieldset className="grid gap-2 rounded-control border border-app-border p-3 sm:grid-cols-2">
          <legend className="px-1 text-xs font-medium">{t(strings.dictationStudio.speakerDimensionLabel)}</legend>
          <label className="flex flex-col gap-1 text-xs sm:col-span-2">
            {t(strings.dictationStudio.targetProfileLabel)}
            <Select
              data-testid={selectors.dictationStudio.experimentSpeakerProfile}
              value={input.speakerTargetProfileId}
              onChange={(event) => set("speakerTargetProfileId", event.currentTarget.value)}
            >
              <option value="">{t(strings.dictationStudio.speakerProfileNone)}</option>
              {(speakerStatus.data?.profiles ?? []).map((profile) => (
                <option key={profile.id} value={profile.id}>
                  {profile.displayName || profile.id} ({profile.clipCount})
                </option>
              ))}
            </Select>
            {speakerStatus.isPending ? <span className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.speakerProfilesLoading)}</span> : null}
            {speakerStatus.isError ? <span className="text-xs text-app-warning">{t(strings.dictationStudio.speakerProfilesError)}</span> : null}
          </label>
          <label className="flex items-center gap-2 text-xs">
            <input type="checkbox" checked={input.speakerExtraction} onChange={(event) => set("speakerExtraction", event.currentTarget.checked)} />
            {t(strings.dictationStudio.speakerExtractionLabel)}
          </label>
          <label className="flex items-center gap-2 text-xs">
            <input type="checkbox" checked={input.speakerVerification} onChange={(event) => set("speakerVerification", event.currentTarget.checked)} />
            {t(strings.dictationStudio.speakerVerificationLabel)}
          </label>
          <label className="flex flex-col gap-1 text-xs">
            {t(strings.dictationStudio.speakerModeLabel)}
            <Select value={input.speakerMode} onChange={(event) => set("speakerMode", event.currentTarget.value as StartExperimentInput["speakerMode"])}>
              <option value="filter">{t(strings.speakerAdmin.modeFilter)}</option>
              <option value="advisory">{t(strings.speakerAdmin.modeAdvisory)}</option>
              <option value="off">{t(strings.speakerAdmin.modeOff)}</option>
            </Select>
          </label>
          <NumberField label={t(strings.speakerAdmin.configThreshold)} value={input.speakerThreshold} step={0.05} min={0} max={1} onChange={(value) => set("speakerThreshold", value)} />
          <label className="flex items-center gap-2 text-xs sm:col-span-2">
            <input type="checkbox" checked={input.speakerAblation} onChange={(event) => set("speakerAblation", event.currentTarget.checked)} />
            {t(strings.dictationStudio.speakerAblationLabel)}
          </label>
        </fieldset>

        <details className="rounded-control border border-app-border p-3 lg:col-span-2">
          <summary data-testid={selectors.dictationStudio.experimentAdvanced} className="cursor-pointer text-xs font-medium">
            {t(strings.dictationStudio.advancedLabel)}
          </summary>
          <div className="mt-3 grid gap-2 sm:grid-cols-2">
            <div className="flex flex-col gap-1">
              <NumberField testId={selectors.dictationStudio.experimentChunkMs} label={t(strings.dictationStudio.chunkMsLabel)} value={input.chunkMs} min={0} onChange={(value) => set("chunkMs", value)} />
              <span className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.chunkMsHint)}</span>
            </div>
            <div className="flex flex-col gap-1">
              <NumberField testId={selectors.dictationStudio.experimentOverlapMaxStall} label={t(strings.dictationStudio.overlapMaxStallLabel)} value={input.overlapMaxStallRejects} min={-1} onChange={(value) => set("overlapMaxStallRejects", value)} />
              <span className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.overlapMaxStallHint)}</span>
            </div>
            <NumberField testId={selectors.dictationStudio.experimentOverlapWindow} label={t(strings.dictationStudio.overlapWindowLabel)} value={input.overlapWindowMs} min={0} onChange={(value) => set("overlapWindowMs", value)} />
            <NumberField testId={selectors.dictationStudio.experimentOverlapCommitRuns} label={t(strings.dictationStudio.overlapCommitRunsLabel)} value={input.overlapCommitRuns} min={0} onChange={(value) => set("overlapCommitRuns", value)} />
            <NumberField testId={selectors.dictationStudio.experimentVadSilence} label={t(strings.dictationStudio.vadSilenceLabel)} value={input.vadSilenceMs} min={0} onChange={(value) => set("vadSilenceMs", value)} />
            <label className="flex items-center gap-2 text-xs sm:col-span-2">
              <input
                data-testid={selectors.dictationStudio.experimentSpeakerFallback}
                type="checkbox"
                checked={input.speakerFallback}
                onChange={(event) => set("speakerFallback", event.currentTarget.checked)}
              />
              {t(strings.dictationStudio.speakerFallbackLabel)}
            </label>
          </div>
        </details>
      </div>

      <div className="mt-4 flex flex-wrap items-center gap-3">
        <Button
          type="button"
          data-testid={selectors.dictationStudio.startExperiment}
          disabled={pending || input.strategies.length === 0 || (!input.longForm && input.clipIds.length === 0 && !hasSweepDurations(input))}
          onClick={onStart}
        >
          {pending ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : <FlaskConical className="h-4 w-4" aria-hidden="true" />}
          {t(strings.dictationStudio.startExperiment)}
        </Button>
        <span className="text-xs text-app-muted-foreground">
          {!input.longForm && input.clipIds.length === 0 && !hasSweepDurations(input)
            ? t(strings.dictationStudio.startInputRequired)
            : t(strings.dictationStudio.builderSafetyHint)}
        </span>
      </div>
    </Panel>
  );
}

function ClipPicker({ selected, onChange, clips }: { selected: string[]; onChange: (ids: string[]) => void; clips: QueryState<ClipMeta[]> }) {
  const { t } = useTranslation();
  const all = clips.data ?? [];
  const selectedSet = new Set(selected);

  const toggle = (id: string) =>
    onChange(selectedSet.has(id) ? selected.filter((item) => item !== id) : [...selected, id]);
  const selectAll = () => onChange(all.map((clip) => clip.id));
  const clear = () => onChange([]);

  return (
    <fieldset data-testid={selectors.dictationStudio.clipPicker} className="flex flex-col gap-2 rounded-control border border-app-border p-3 lg:col-span-2">
      <legend className="px-1 text-xs font-medium">{t(strings.dictationStudio.clipPickerLabel)}</legend>
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.clipPickerHint)}</span>
        <span data-testid={selectors.dictationStudio.clipPickerCount} className="text-xs font-medium text-app-foreground">
          {t(strings.dictationStudio.clipPickerSelected, { selected: selected.length, total: all.length })}
        </span>
      </div>
      {clips.isPending ? (
        <p className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.clipPickerLoading)}</p>
      ) : clips.isError ? (
        <ApiErrorState error={clips.error ?? new Error(t(strings.dictationStudio.clipPickerError))} title={t(strings.dictationStudio.clipPickerError)} onRetry={() => void clips.refetch()} />
      ) : all.length === 0 ? (
        <p className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.clipPickerEmpty)}</p>
      ) : (
        <>
          <div className="flex flex-wrap gap-2">
            <Button type="button" size="sm" variant="outline" data-testid={selectors.dictationStudio.clipPickerSelectAll} onClick={selectAll}>
              {t(strings.dictationStudio.clipPickerSelectAll)}
            </Button>
            <Button type="button" size="sm" variant="ghost" data-testid={selectors.dictationStudio.clipPickerClear} onClick={clear}>
              {t(strings.dictationStudio.clipPickerClear)}
            </Button>
          </div>
          <div className="flex max-h-48 flex-col gap-1 overflow-y-auto">
            {all.map((clip) => (
              <ClipPickerRow key={clip.id} clip={clip} checked={selectedSet.has(clip.id)} onToggle={() => toggle(clip.id)} />
            ))}
          </div>
        </>
      )}
    </fieldset>
  );
}

function ClipPickerRow({ clip, checked, onToggle }: { clip: ClipMeta; checked: boolean; onToggle: () => void }) {
  return (
    <label className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5 rounded-control px-1 py-1 text-xs hover:bg-app-surface-muted">
      <input
        type="checkbox"
        data-testid={selectors.dictationStudio.clipPick({ id: clip.id })}
        checked={checked}
        onChange={onToggle}
      />
      <span className="max-w-full truncate font-medium text-app-foreground">{clip.referenceText || clip.id}</span>
      <span className="text-app-muted-foreground">
        {clip.id}
        {clip.tags.length > 0 ? ` · ${clip.tags.join(", ")}` : ""}
        {clip.durationMs > 0 ? ` · ${(clip.durationMs / 1000).toFixed(1)}s` : ""}
      </span>
    </label>
  );
}

function NumberField({
  label,
  value,
  onChange,
  testId,
  step = 1,
  min,
  max,
}: {
  label: string;
  value: number;
  onChange: (value: number) => void;
  testId?: string;
  step?: number;
  min?: number;
  max?: number;
}) {
  const clamp = (n: number): number => {
    let next = n;
    if (typeof min === "number") next = Math.max(min, next);
    if (typeof max === "number") next = Math.min(max, next);
    return next;
  };
  return (
    <label className="flex flex-col gap-1 text-xs">
      {label}
      <Input
        data-testid={testId}
        type="number"
        step={step}
        min={min}
        max={max}
        value={value}
        onChange={(event) => {
          const raw = event.currentTarget.value;
          // Empty resets to the lower bound (or 0); a non-empty value that
          // does not parse is ignored rather than silently coerced to 0,
          // so invalid keystrokes don't hide the operator's real intent.
          if (raw === "") {
            onChange(clamp(min ?? 0));
            return;
          }
          const parsed = Number(raw);
          if (Number.isNaN(parsed)) return;
          onChange(clamp(parsed));
        }}
      />
    </label>
  );
}
