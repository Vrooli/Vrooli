import { useTranslation } from "../../i18n";
import type { Region } from "./region";

export type RegionCaptureAspect = "free" | "1:1" | "4:3" | "16:9";
export type RegionCaptureField = "x" | "y" | "w" | "h";

export interface RegionCaptureDraft {
  key: RegionCaptureField;
  value: string;
}

interface RegionCaptureControlsProps {
  aspect: RegionCaptureAspect;
  draft: RegionCaptureDraft | null;
  imageFailed: boolean;
  busy: boolean;
  region: Region | null;
  onChooseAspect: (aspect: RegionCaptureAspect) => void;
  onChangeField: (field: RegionCaptureField, value: string) => void;
  onCommitDraft: () => void;
}

const ASPECT_ORDER: readonly RegionCaptureAspect[] = ["free", "1:1", "4:3", "16:9"];

export function RegionCaptureControls({
  aspect,
  draft,
  imageFailed,
  busy,
  region,
  onChooseAspect,
  onChangeField,
  onCommitDraft,
}: RegionCaptureControlsProps) {
  const { t } = useTranslation();
  const aspectLabels: Record<RegionCaptureAspect, string> = {
    free: t("workbench.studio.aspectFree", { defaultValue: "Free" }),
    "1:1": t("workbench.studio.aspectSquare", { defaultValue: "1:1" }),
    "4:3": t("workbench.studio.aspectWide", { defaultValue: "4:3" }),
    "16:9": t("workbench.studio.aspectCinema", { defaultValue: "16:9" }),
  };
  const fieldLabels: Record<RegionCaptureField, string> = {
    x: t("workbench.studio.regionX", { defaultValue: "X" }),
    y: t("workbench.studio.regionY", { defaultValue: "Y" }),
    w: t("workbench.studio.regionW", { defaultValue: "W" }),
    h: t("workbench.studio.regionH", { defaultValue: "H" }),
  };
  const disabled = !region || imageFailed || busy;

  return (
    <div className="rc-controls">
      <div
        className="rc-aspect-group"
        role="group"
        aria-label={t("workbench.studio.aspectGroup", { defaultValue: "Aspect ratio" })}
      >
        {ASPECT_ORDER.map((id) => (
          <button
            key={id}
            type="button"
            className="rc-aspect"
            aria-pressed={aspect === id}
            disabled={disabled}
            onClick={() => onChooseAspect(id)}
            data-testid={`region-capture-aspect-${id}`}
          >
            {aspectLabels[id]}
          </button>
        ))}
      </div>

      <div className="rc-fields">
        {(["x", "y", "w", "h"] as const).map((key) => (
          <label key={key} className="rc-field">
            <span>{fieldLabels[key]}</span>
            <input
              type="number"
              min={0}
              step={1}
              inputMode="numeric"
              value={draft?.key === key ? draft.value : (region?.[key] ?? 0)}
              disabled={disabled}
              onChange={(event) => onChangeField(key, event.target.value)}
              onBlur={onCommitDraft}
              onKeyDown={(event) => {
                if (event.key === "Enter") event.currentTarget.blur();
              }}
              data-testid={`region-capture-${key}`}
            />
          </label>
        ))}
      </div>
    </div>
  );
}
