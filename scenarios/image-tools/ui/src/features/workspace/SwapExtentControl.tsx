import { SegmentedControl } from "../../components/ui/segmented-control";
import { Toggle } from "../../components/ui/toggle";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import type { ExtentMeta, ExtentOptionMeta, ExtentStageMeta } from "./capabilityCatalog.generated";
import {
  activePreset,
  applyPreset,
  extentDescKey,
  extentLabelKey,
  optionValue,
  setOption,
  strengthApplies,
  toggleStage,
  type ExtentSelection,
} from "./swapExtent";

export interface SwapExtentControlProps {
  extent: ExtentMeta;
  selection: ExtentSelection;
  onChange: (selection: ExtentSelection) => void;
}

/**
 * "How much to swap": a preset row and one switch per stage the operation's
 * manifest declares, plus the face-strength slider while the face stage runs.
 * Presets, stages and their copy all come from the generated capability
 * catalog, so a stage added to the manifest shows up here without a UI change.
 */
export function SwapExtentControl({ extent, selection, onChange }: SwapExtentControlProps) {
  const { t } = useTranslation();
  const preset = activePreset(extent, selection);
  // The base swap is the stage every extent has; the rest are choices.
  const choices = extent.stages.filter((stage) => stage.kind !== "swap" || stage.replaces);
  return (
    <fieldset
      data-testid={selectors.workspace.create.extent}
      className="flex flex-col gap-2 rounded-control border border-app-border bg-app-surface-muted p-3"
    >
      <legend className="px-1 text-xs font-medium text-app-muted-foreground">
        {t(strings.workspace.extent.title)}
      </legend>
      <SegmentedControl<string>
        label={t(strings.workspace.extent.presetsLabel)}
        value={preset?.id ?? ""}
        options={extent.presets.map((candidate) => ({ value: candidate.id, label: t(extentLabelKey(candidate)) }))}
        onChange={(id) => onChange(applyPreset(extent, selection, id))}
        data-testid={selectors.workspace.create.extentPresets}
      />
      {preset && <p className="text-xs text-app-muted-foreground">{t(extentDescKey(preset))}</p>}
      <div className="flex flex-col gap-1" aria-label={t(strings.workspace.extent.stagesLabel)}>
        {choices.map((stage) => (
          <div key={stage.id} className="flex flex-col">
            <Toggle
              label={t(extentLabelKey(stage))}
              checked={selection.stages.includes(stage.id)}
              onChange={() => onChange(toggleStage(extent, selection, stage.id))}
              data-testid={`${selectors.workspace.create.extentStage}-${stage.id}`}
            />
            <span className="text-xs text-app-muted-foreground">{t(extentDescKey(stage))}</span>
            {stage.option && selection.stages.includes(stage.id) && (
              <StageOption stage={stage} option={stage.option} selection={selection} onChange={onChange} />
            )}
          </div>
        ))}
      </div>
      {strengthApplies(extent, selection) && (
        <label className="flex flex-col gap-1 text-xs text-app-muted-foreground">
          <span>
            {t(strings.workspace.extent.strengthLabel)}: {selection.faceStrength.toFixed(2)}
          </span>
          <input
            type="range"
            min={0}
            max={1}
            step={0.05}
            value={selection.faceStrength}
            data-testid={selectors.workspace.create.extentStrength}
            onChange={(event) => onChange({ ...selection, faceStrength: Number(event.currentTarget.value) })}
          />
          <span>{t(strings.workspace.extent.strengthHint)}</span>
        </label>
      )}
    </fieldset>
  );
}

interface StageOptionProps {
  stage: ExtentStageMeta;
  option: ExtentOptionMeta;
  selection: ExtentSelection;
  onChange: (selection: ExtentSelection) => void;
}

/** The choice a stage declares for how it runs (for skin tone: toward the source or the body). */
function StageOption({ stage, option, selection, onChange }: StageOptionProps) {
  const { t } = useTranslation();
  const value = optionValue(option, selection);
  const current = option.values.find((candidate) => candidate.id === value);
  return (
    <div className="mt-1 flex flex-col gap-1">
      <SegmentedControl<string>
        label={t(extentLabelKey(stage))}
        value={value}
        options={option.values.map((candidate) => ({ value: candidate.id, label: t(extentLabelKey(candidate)) }))}
        onChange={(next) => onChange(setOption(selection, option, next))}
        data-testid={`${selectors.workspace.create.extentStage}-${stage.id}-${option.param}`}
      />
      {current && <span className="text-xs text-app-muted-foreground">{t(extentDescKey(current))}</span>}
    </div>
  );
}
