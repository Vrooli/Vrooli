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
}

/**
 * ModelPickerButton is the always-present control above every AI action: it
 * shows which model will run on THIS host (with an affirmative fit hint) and
 * opens the full, host-aware model picker on click — replacing the old static
 * "model detected" label. The closed-state label comes from a lightweight
 * SelectModel query (honoring the override); the menu loads its full candidate
 * set lazily when opened.
 */
export function ModelPickerButton({ operation, operationLabel, value, onChange }: ModelPickerButtonProps) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const picker = useModelPicker({ operation, active: open });

  const selection = useQuery({
    queryKey: ["model-select", operation, value],
    enabled: !!operation,
    retry: false,
    queryFn: () => modelsClient.selectModel({ operation, overrideId: value }),
  });

  const model = selection.data?.model;
  const gpuViable = selection.data?.gpuViable ?? false;
  const fitKey = model
    ? gpuViable
      ? strings.models.picker.fit.gpu
      : model.hardware?.cpuCapable
        ? strings.models.picker.fit.cpu
        : strings.models.picker.state.needsBackend
    : "";
  const FitIcon = gpuViable ? Sparkles : Cpu;

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

      <ModelPicker
        open={open}
        onClose={() => {
          setOpen(false);
          void selection.refetch();
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
