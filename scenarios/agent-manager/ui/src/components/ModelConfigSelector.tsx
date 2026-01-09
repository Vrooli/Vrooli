import { useId } from "react";
import type { ModelOption } from "../types";
import { ModelPreset } from "../types";
import { Label } from "./ui/label";
import { ModelSelector } from "./ModelSelector";

export type ModelSelectionMode = "default" | "preset" | "model";

export interface ModelSelectionState {
  mode: ModelSelectionMode;
  model: string;
  preset: ModelPreset;
}

interface ModelConfigSelectorProps {
  value: ModelSelectionState;
  onChange: (value: ModelSelectionState) => void;
  models: ModelOption[];
  presetMap?: Record<string, string>;
  label?: string;
}

const PRESET_OPTIONS = [
  { value: ModelPreset.FAST, label: "Fast" },
  { value: ModelPreset.CHEAP, label: "Cheap" },
  { value: ModelPreset.SMART, label: "Smart" },
];

function modelPresetKey(preset: ModelPreset): string {
  switch (preset) {
    case ModelPreset.FAST:
      return "FAST";
    case ModelPreset.CHEAP:
      return "CHEAP";
    case ModelPreset.SMART:
      return "SMART";
    default:
      return "";
  }
}

export function ModelConfigSelector({
  value,
  onChange,
  models,
  presetMap,
  label = "Model Selection",
}: ModelConfigSelectorProps) {
  const groupId = useId();
  const presetKey = modelPresetKey(value.preset);
  const resolvedModel = presetKey ? presetMap?.[presetKey] : "";

  return (
    <div className="space-y-3">
      <Label>{label}</Label>
      <div className="grid gap-2">
        <label className="flex items-center gap-2 text-sm">
          <input
            type="radio"
            name={groupId}
            checked={value.mode === "default"}
            onChange={() =>
              onChange({
                mode: "default",
                model: "",
                preset: ModelPreset.UNSPECIFIED,
              })
            }
          />
          Use runner default
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="radio"
            name={groupId}
            checked={value.mode === "preset"}
            onChange={() =>
              onChange({
                ...value,
                mode: "preset",
                model: "",
                preset: value.preset !== ModelPreset.UNSPECIFIED ? value.preset : ModelPreset.FAST,
              })
            }
          />
          Preset (Fast/Cheap/Smart)
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="radio"
            name={groupId}
            checked={value.mode === "model"}
            onChange={() =>
              onChange({
                ...value,
                mode: "model",
                preset: ModelPreset.UNSPECIFIED,
              })
            }
          />
          Specific model
        </label>
      </div>

      {value.mode === "preset" && (
        <div className="space-y-2">
          <Label htmlFor="modelPreset">Preset</Label>
          <select
            id="modelPreset"
            value={value.preset}
            onChange={(event) =>
              onChange({
                ...value,
                preset: Number(event.target.value) as ModelPreset,
              })
            }
            className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            {PRESET_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
          {resolvedModel && (
            <p className="text-xs text-muted-foreground">
              Currently mapped to <span className="font-medium text-foreground">{resolvedModel}</span>.
            </p>
          )}
        </div>
      )}

      {value.mode === "model" && (
        <ModelSelector
          value={value.model}
          onChange={(model) =>
            onChange({
              ...value,
              model,
            })
          }
          models={models}
          label="Model"
          placeholder="Enter custom model..."
        />
      )}
    </div>
  );
}
