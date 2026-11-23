import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import type { SelectedModel } from "./useEnhance";

export interface ModelCapacityNoteProps {
  /** The model the run will use, or null when it is not known or not running on the graphics card. */
  model: Pick<SelectedModel, "gpuViable" | "cpuCapable"> | null;
}

/**
 * What happens when the graphics card is busy, in plain words and from the model's
 * own data: a model with a processor path says it will use it (slower); one without
 * says it may wait. Nothing when the model does not run on the graphics card.
 */
export function ModelCapacityNote({ model }: ModelCapacityNoteProps) {
  const { t } = useTranslation();
  if (!model?.gpuViable) {
    return null;
  }
  return (
    <p data-testid={selectors.workspace.capacityNote} className="text-xs text-app-muted-foreground">
      {t(model.cpuCapable ? strings.workspace.capacity.cpuFallback : strings.workspace.capacity.gpuOnly)}
    </p>
  );
}
