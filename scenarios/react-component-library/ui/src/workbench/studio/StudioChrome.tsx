import { useTranslation } from "../../i18n";
import { Field, Segmented, TextInput } from "../primitives";
import { DEFAULT_TEMPLATE } from "../templates/library";

/** Structural twin of the persisted design options; no import cycle. */
export interface ChromeOptions {
  radius?: number;
  display?: "sans" | "serif";
}

/**
 * Design-owned chrome: the corner radius and the display face apply to every
 * frame in this design. They live in `DesignDocument.options`, not in the
 * shared template, so two designs can share a palette and still look unlike
 * each other.
 */
export function StudioChrome({
  options,
  onChange,
}: {
  options: ChromeOptions;
  onChange: (patch: ChromeOptions, label: string) => void;
}) {
  const { t } = useTranslation();
  // The same fallback `frameTemplate` uses, so the control can never show a
  // value the canvas is not rendering.
  const radius = options.radius ?? DEFAULT_TEMPLATE.radius;
  const display = options.display ?? DEFAULT_TEMPLATE.display;
  return (
    <section className="studio-token-editor" aria-labelledby="studio-chrome-title">
      <span className="wb-caption" id="studio-chrome-title">
        {t("workbench.studio.chromeTitle", { defaultValue: "Design chrome" })}
      </span>
      <Field label={t("workbench.studio.chromeRadius", { defaultValue: "Corner radius" })}>
        <TextInput
          type="number"
          min={0}
          max={48}
          step={1}
          value={radius}
          onChange={(event) => {
            const next = Number(event.target.value);
            if (!Number.isFinite(next)) return;
            onChange(
              { radius: Math.max(0, Math.min(48, Math.round(next))) },
              t("workbench.studio.chromeRadiusAction", {
                defaultValue: "Change corner radius",
              }),
            );
          }}
        />
      </Field>
      <span className="wb-caption">
        {t("workbench.studio.chromeDisplay", { defaultValue: "Display face" })}
      </span>
      <Segmented
        label={t("workbench.studio.chromeDisplay", { defaultValue: "Display face" })}
        value={display}
        onChange={(display) =>
          onChange(
            { display },
            t("workbench.studio.chromeDisplayAction", {
              defaultValue: "Change display face",
            }),
          )
        }
        items={[
          { id: "sans", label: t("workbench.studio.chromeSans", { defaultValue: "Sans" }) },
          { id: "serif", label: t("workbench.studio.chromeSerif", { defaultValue: "Serif" }) },
        ]}
      />
      <button
        type="button"
        className="studio-link-btn"
        onClick={() =>
          onChange(
            { radius: DEFAULT_TEMPLATE.radius, display: DEFAULT_TEMPLATE.display },
            t("workbench.studio.chromeResetAction", {
              defaultValue: "Reset design chrome",
            }),
          )
        }
      >
        {t("workbench.studio.chromeReset", { defaultValue: "Reset to library defaults" })}
      </button>
      <p className="studio-note">
        {t("workbench.studio.chromeNote", {
          defaultValue: "Applies to every frame in this design. Undo reverses it.",
        })}
      </p>
    </section>
  );
}
