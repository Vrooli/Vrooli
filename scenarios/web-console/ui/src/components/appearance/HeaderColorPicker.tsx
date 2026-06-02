import { Pipette } from "lucide-react";
import { useTranslation } from "react-i18next";
import { HEADER_COLORS } from "../../consts/config";
import { cn } from "../../lib/classnames";
import { strings } from "../../consts/strings";

interface HeaderColorPickerProps {
  currentColor: string;
  onSelectColor: (color: string) => void;
  testIdPrefix?: string;
}

/** A valid hex color the native color input can seed itself with. */
const FALLBACK_CUSTOM_COLOR = "#7aa0ff";
const HEX_COLOR = /^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/;

export default function HeaderColorPicker({
  currentColor,
  onSelectColor,
  testIdPrefix = "appearance",
}: HeaderColorPickerProps) {
  const { t } = useTranslation();

  // The current color is "custom" when it's an actual color the user picked
  // that isn't one of the presets (and isn't the transparent option).
  const isPreset = (HEADER_COLORS as readonly string[]).includes(currentColor);
  const isCustom = currentColor !== "transparent" && !isPreset;
  const customSwatchColor = isCustom && HEX_COLOR.test(currentColor)
    ? currentColor
    : FALLBACK_CUSTOM_COLOR;

  return (
    <section>
      <h3 className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted mb-2">
        {t(strings.appearance.headerColorHeading)}
      </h3>
      <div className="flex flex-wrap gap-1.5">
        {/* Transparent option */}
        <button
          type="button"
          data-testid={`${testIdPrefix}-header-color-transparent`}
          className={cn(
            "h-6 w-6 rounded-full border",
            currentColor === "transparent" ? "border-wc-accent ring-1 ring-wc-accent" : "border-wc-default",
          )}
          style={{ background: "rgb(var(--wc-surface-input))" }}
          onClick={() => onSelectColor("transparent")}
          title={t(strings.appearance.noColorTitle)}
        />
        {HEADER_COLORS.map((color) => (
          <button
            key={color}
            type="button"
            data-testid={`${testIdPrefix}-header-color-${color}`}
            className={cn(
              "h-6 w-6 rounded-full border",
              currentColor === color ? "border-wc-accent ring-1 ring-wc-accent" : "border-wc-default",
            )}
            style={{ backgroundColor: color }}
            onClick={() => onSelectColor(color)}
            title={color}
          />
        ))}
        {/* Custom color — opens the native color picker. When a custom color is
            active, the swatch shows it; otherwise it shows a pipette affordance. */}
        <label
          data-testid={`${testIdPrefix}-header-color-custom`}
          className={cn(
            "relative h-6 w-6 rounded-full border flex items-center justify-center cursor-pointer overflow-hidden",
            isCustom ? "border-wc-accent ring-1 ring-wc-accent" : "border-wc-default",
          )}
          style={isCustom ? { backgroundColor: customSwatchColor } : { background: "rgb(var(--wc-surface-input))" }}
          title={isCustom ? currentColor : t(strings.appearance.customColorTitle)}
        >
          {!isCustom && <Pipette className="h-3 w-3 text-wc-text-muted" />}
          <input
            type="color"
            data-testid={`${testIdPrefix}-header-color-custom-input`}
            className="absolute inset-0 h-full w-full cursor-pointer opacity-0"
            value={customSwatchColor}
            onChange={(e) => onSelectColor(e.target.value)}
            aria-label={t(strings.appearance.customColorTitle)}
          />
        </label>
      </div>
    </section>
  );
}
