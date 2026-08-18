/**
 * @libraryId react-component-library:ColorPicker
 * @displayName ColorPicker
 * @description Accessible palette, recent-color, gradient, and native custom-color picker.
 * @version 1.0.0
 * @tags ["form","color","appearance"]
 */
import { Check, Pipette, Plus, X, type LucideIcon } from "lucide-react";
import { useState } from "react";
import { isHexColor, isLightColor, parseColorValue, serializeColorValue } from "./colorUtils";
import { useDeferredColorCommit } from "./useDeferredColorCommit";

export type ColorPickerLabels = {
  heading?: string;
  transparent?: string;
  custom?: string;
  recents?: string;
  primary?: string;
  secondary?: string;
  addGradient?: string;
  removeGradient?: string;
};
export type ColorPickerIcons = {
  check?: LucideIcon;
  custom?: LucideIcon;
  add?: LucideIcon;
  remove?: LucideIcon;
};
export type ColorPickerProps = {
  palette: readonly string[];
  value: string;
  onChange: (value: string) => void;
  recentColors?: readonly string[];
  onRecordRecent?: (color: string) => void;
  labels?: ColorPickerLabels;
  icons?: ColorPickerIcons;
  allowGradient?: boolean;
  testIdPrefix?: string;
};

const fallbackColor = "#4dabf7";

export default function ColorPicker({
  palette,
  value,
  onChange,
  recentColors = [],
  onRecordRecent,
  labels = {},
  icons = {},
  allowGradient = false,
  testIdPrefix = "color-picker",
}: ColorPickerProps) {
  const { colors, transparent } = parseColorValue(value);
  const [activeSlot, setActiveSlot] = useState<0 | 1>(0);
  const { park, flush } = useDeferredColorCommit(onRecordRecent);
  const CheckIcon = icons.check ?? Check;
  const CustomIcon = icons.custom ?? Pipette;
  const AddIcon = icons.add ?? Plus;
  const RemoveIcon = icons.remove ?? X;
  const secondaryOpen = colors.length > 1 || activeSlot === 1;
  const activeColor = colors[activeSlot];
  const customColor = isHexColor(activeColor) ? activeColor : fallbackColor;
  const selected = (color: string) => !transparent && colors[activeSlot] === color;
  const choose = (color: string, record = true) => {
    const next = activeSlot === 0 ? [color, colors[1]] : [colors[0] ?? color, color];
    onChange(serializeColorValue(next.filter(isHexColor)));
    if (record) onRecordRecent?.(color);
  };
  const mark = (color?: string) => (
    <CheckIcon
      aria-hidden
      className={`h-icon-compact w-icon-compact ${isLightColor(color) ? "text-app-foreground" : "text-app-background"}`}
    />
  );
  const swatch = (color: string, suffix: string) => (
    <button
      key={suffix}
      type="button"
      data-testid={`${testIdPrefix}-${suffix}`}
      className={`flex h-control-tight w-control-tight items-center justify-center rounded-full border transition-shadow focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-app-primary ${selected(color) ? "border-app-primary ring-2 ring-app-primary/50" : "border-app-border hover:ring-1 hover:ring-app-muted-foreground"}`}
      style={{ backgroundColor: color }}
      onClick={() => choose(color)}
      title={color}
      aria-label={color}
      aria-pressed={selected(color)}
    >
      {selected(color) ? mark(color) : null}
    </button>
  );
  return (
    <section className="space-y-space-2xs" aria-label={labels.heading ?? "Color picker"}>
      {labels.heading ? (
        <h3 className="text-xs font-semibold uppercase tracking-wider text-app-muted-foreground">
          {labels.heading}
        </h3>
      ) : null}
      {allowGradient ? (
        <div className="flex items-center gap-space-2xs">
          <button
            type="button"
            data-testid={`${testIdPrefix}-slot-0`}
            className="h-control-tight w-control-tight rounded-md border border-app-border"
            style={colors[0] ? { backgroundColor: colors[0] } : undefined}
            onClick={() => setActiveSlot(0)}
            aria-label={labels.primary ?? labels.heading ?? "Color picker"}
            aria-pressed={activeSlot === 0}
          />
          {secondaryOpen ? (
            <>
              <button
                type="button"
                data-testid={`${testIdPrefix}-slot-1`}
                className="h-control-tight w-control-tight rounded-md border border-app-border"
                style={colors[1] ? { backgroundColor: colors[1] } : undefined}
                onClick={() => setActiveSlot(1)}
                aria-label={labels.secondary ?? labels.heading ?? "Color picker"}
                aria-pressed={activeSlot === 1}
              />
              <button
                type="button"
                data-testid={`${testIdPrefix}-remove-gradient`}
                className="flex h-control-tight w-control-tight items-center justify-center rounded border border-app-border"
                onClick={() => {
                  setActiveSlot(0);
                  onChange(serializeColorValue(colors.slice(0, 1)));
                }}
                aria-label={labels.removeGradient ?? "Remove gradient"}
              >
                <RemoveIcon aria-hidden className="h-icon-sm w-icon-sm" />
              </button>
            </>
          ) : (
            <button
              type="button"
              data-testid={`${testIdPrefix}-add-gradient`}
              className="flex h-control-tight items-center gap-space-3xs rounded border border-dashed border-app-border px-space-2xs text-xs"
              onClick={() => setActiveSlot(1)}
            >
              <AddIcon aria-hidden className="h-icon-compact w-icon-compact" />
              {labels.addGradient ?? "Add gradient"}
            </button>
          )}
        </div>
      ) : null}
      <div className="flex flex-wrap gap-space-2xs">
        <button
          type="button"
          data-testid={`${testIdPrefix}-transparent`}
          className={`flex h-control-tight w-control-tight items-center justify-center rounded-full border ${transparent ? "border-app-primary ring-2 ring-app-primary/50" : "border-app-border"}`}
          onClick={() => {
            setActiveSlot(0);
            onChange("transparent");
          }}
          aria-label={labels.transparent ?? "Transparent"}
          aria-pressed={transparent}
        >
          {transparent ? mark() : null}
        </button>
        {palette.map((color) => swatch(color, `palette-${color}`))}
        <label
          data-testid={`${testIdPrefix}-custom`}
          className={`relative flex h-control-tight w-control-tight cursor-pointer items-center justify-center overflow-hidden rounded-full border border-app-border ${isHexColor(activeColor) && !palette.includes(activeColor) ? "ring-2 ring-app-primary/50" : ""}`}
          style={
            isHexColor(activeColor) && !palette.includes(activeColor)
              ? { backgroundColor: activeColor }
              : undefined
          }
          title={labels.custom ?? "Custom color"}
        >
          {isHexColor(activeColor) && !palette.includes(activeColor) ? (
            mark(activeColor)
          ) : (
            <CustomIcon aria-hidden className="h-icon-sm w-icon-sm text-app-muted-foreground" />
          )}
          <input
            type="color"
            data-testid={`${testIdPrefix}-custom-input`}
            className="absolute inset-0 cursor-pointer opacity-0"
            value={customColor}
            onChange={(event) => {
              choose(event.target.value, false);
              park(event.target.value);
            }}
            onBlur={flush}
            aria-label={labels.custom ?? "Custom color"}
          />
        </label>
      </div>
      {recentColors.length ? (
        <div data-testid={`${testIdPrefix}-recents`}>
          <p className="mb-space-3xs text-xs font-medium text-app-muted-foreground">
            {labels.recents ?? "Recent colors"}
          </p>
          <div className="flex flex-wrap gap-space-2xs">
            {recentColors.map((color) => swatch(color, `recent-${color}`))}
          </div>
        </div>
      ) : null}
    </section>
  );
}
