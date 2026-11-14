import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ChevronDown, Cpu, Sparkles } from "lucide-react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { modelsClient } from "../../api/models";
import { ModelPicker } from "./ModelPicker";
import { useModelPicker } from "./useModelPicker";

export interface ModelPickerButtonProps {
  /** Operation the picker is for. */
  operation: string;
  /** Localized human label for the operation (picker title). */
  operationLabel: string;
  /** Current model override id, or "" for the auto-selected default. */
  value: string;
  /** Set the model override ("" clears it back to auto). */
  onChange: (id: string) => void;
  /** Resolve and display CPU-only execution when the request forces CPU. */
  forceCpu?: boolean;
}

/**
 * ModelPickerButton is the always-present control above every AI action: it
 * shows which model will run on THIS host (with an affirmative fit hint) and
 * opens the full, host-aware model picker on click — replacing the old static
 * "model detected" label. The closed-state label comes from a lightweight
 * SelectModel query (honoring the override); the menu loads its full candidate
 * set lazily when opened.
 */
export function ModelPickerButton({ operation, operationLabel, value, onChange, forceCpu = false }: ModelPickerButtonProps) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const picker = useModelPicker({ operation, active: open, forceCpu });

  const selection = useQuery({
    queryKey: ["model-select", operation, value, forceCpu],
    enabled: !!operation,
    retry: false,
    queryFn: () => modelsClient.selectModel({ operation, overrideId: value, forceCpu }),
  });

  // How the effective model serves THIS op (native vs derived) + the quality
  // caveat — surfaced as an always-visible note so a derived run is an informed
  // choice before submit, not a silent one. Reuses the Phase-4 ExplainResolution
  // RPC (read-only); failures just hide the note.
  const resolution = useQuery({
    queryKey: ["model-explain", operation, value, forceCpu],
    enabled: !!operation,
    retry: false,
    queryFn: () => modelsClient.explainResolution({ operation, modelId: value, forceCpu }),
  });
  const resolved = resolution.data?.resolution;
  const derivedCaveat = resolved?.support === "derived" ? resolved.caveat : "";

  const model = selection.data?.model;
  const gpuViable = selection.data?.gpuViable ?? false;
  // SelectModel reports whether the model's VRAM requirement fits the host;
  // ExplainResolution reports the backend that will actually execute it. A
  // CPU-only backend can be selected even when the model would fit the GPU.
  const effectiveTier = resolved?.tier;
  const fitKey = !model
    ? ""
    : forceCpu && effectiveTier === "local-cpu"
      ? strings.models.picker.fit.cpuRequested
      : effectiveTier === "local-gpu"
      ? strings.models.picker.fit.gpu
      : effectiveTier === "local-cpu"
        ? gpuViable
          ? strings.models.picker.fit.cpuBackend
          : strings.models.picker.fit.cpu
        : "";
  const FitIcon = effectiveTier === "local-gpu" ? Sparkles : Cpu;

  return (
    <div className="flex flex-col gap-1">
      <span className="text-[11px] font-medium uppercase tracking-wide text-app-muted-foreground">
        {t(strings.models.picker.trigger.label)}
      </span>
      <button
        type="button"
        data-testid={selectors.models.pickerTrigger}
        onClick={() => setOpen(true)}
        className="flex items-center justify-between gap-2 rounded-control border border-app-border bg-app-surface px-3 py-2 text-left hover:border-app-primary"
      >
        <span className="flex min-w-0 items-center gap-2">
          <FitIcon aria-hidden="true" className="h-4 w-4 shrink-0 text-app-brand" />
          <span className="min-w-0">
            <span className="block truncate text-sm font-medium text-app-foreground">
              {selection.isLoading
                ? t(strings.models.picker.trigger.loading)
                : model?.name ?? t(strings.models.picker.trigger.none)}
            </span>
            {model && fitKey ? (
              <span className="block truncate text-[11px] text-app-muted-foreground">{t(fitKey)}</span>
            ) : null}
          </span>
        </span>
        <span className="flex shrink-0 items-center gap-1 text-[11px] text-app-primary">
          {t(strings.models.picker.trigger.change)}
          <ChevronDown aria-hidden="true" className="h-3.5 w-3.5" />
        </span>
      </button>

      {derivedCaveat ? (
        <p
          data-testid={selectors.models.pickerTriggerCaveat}
          className="text-[11px] text-app-warning"
        >
          {t(strings.models.picker.caveatBanner, {
            op: operationLabel,
            caveat: derivedCaveat,
          })}
        </p>
      ) : null}

      <ModelPicker
        open={open}
        onClose={() => {
          setOpen(false);
          void selection.refetch();
          void resolution.refetch();
        }}
        operation={operation}
        operationLabel={operationLabel}
        picker={picker}
        value={value}
        onSelect={onChange}
      />
    </div>
  );
}
