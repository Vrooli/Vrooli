/**
 * @libraryId react-component-library:ColorPicker
 * @displayName ColorPicker
 * @description Accessible palette, recent-color, gradient, and native custom-color picker.
 * @version 1.0.3
 * @tags ["form","color","appearance"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
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
  className?: string;
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

const styles = `
[data-rcl-color-picker] { display: grid; gap: var(--space-sm); min-inline-size: 0; color: var(--color-foreground); }
[data-rcl-color-picker-heading] { margin: 0; color: var(--color-muted-foreground); font: var(--text-overline); letter-spacing: .08em; text-transform: uppercase; }
[data-rcl-color-picker-row] { display: flex; align-items: center; flex-wrap: wrap; gap: var(--space-xs); min-inline-size: 0; }
[data-rcl-color-picker-swatch], [data-rcl-color-picker-slot], [data-rcl-color-picker-custom], [data-rcl-color-picker-transparent], [data-rcl-color-picker-remove] { position: relative; display: inline-grid; place-items: center; flex: 0 0 auto; inline-size: var(--tap-target-min); block-size: var(--tap-target-min); box-sizing: border-box; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-pill); background: var(--color-surface); color: var(--color-foreground); cursor: pointer; transition: transform var(--dur-quick) var(--ease-standard), box-shadow var(--dur-quick) var(--ease-standard), border-color var(--dur-quick) var(--ease-standard); }
[data-rcl-color-picker-swatch]:hover, [data-rcl-color-picker-slot]:hover, [data-rcl-color-picker-custom]:hover, [data-rcl-color-picker-transparent]:hover, [data-rcl-color-picker-remove]:hover { transform: translateY(-1px); box-shadow: var(--elev-raised); }
[data-rcl-color-picker-swatch]:active, [data-rcl-color-picker-slot]:active, [data-rcl-color-picker-custom]:active, [data-rcl-color-picker-transparent]:active, [data-rcl-color-picker-remove]:active { transform: translateY(0) scale(.97); }
[data-rcl-color-picker-swatch][data-selected="true"], [data-rcl-color-picker-slot][aria-pressed="true"], [data-rcl-color-picker-transparent][aria-pressed="true"], [data-rcl-color-picker-custom][data-selected="true"] { border-color: var(--color-primary); box-shadow: 0 0 0 var(--space-3xs) color-mix(in srgb, var(--color-primary) 30%, transparent); }
[data-rcl-color-picker-add] { display: inline-flex; align-items: center; gap: var(--space-3xs); min-block-size: var(--tap-target-min); border: var(--border-hairline) dashed var(--color-border); border-radius: var(--radius-control); background: var(--color-surface); color: var(--color-foreground); padding-inline: var(--space-xs); font: var(--text-label); cursor: pointer; }
[data-rcl-color-picker-mark] { inline-size: var(--space-sm); block-size: var(--space-sm); }
[data-rcl-color-picker-input] { position: absolute; inset: 0; inline-size: 100%; block-size: 100%; cursor: pointer; opacity: 0; }
[data-rcl-color-picker-recents] { display: grid; gap: var(--space-2xs); }
[data-rcl-color-picker-recents] p { margin: 0; color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-color-picker] [data-rcl-color-picker-swatch][data-selected="false"] { box-shadow: none; }


`;

export default function ColorPicker({
  className,
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
      data-rcl-color-picker-mark
      style={{
        color: isLightColor(color) ? "var(--color-foreground)" : "var(--color-background)",
      }}
    />
  );
  const swatch = (color: string, suffix: string) => (
    <button
      key={suffix}
      type="button"
      data-testid={`forms.color-picker-${testIdPrefix}-${suffix}`}
      data-rcl-color-picker-swatch
      data-selected={selected(color)}
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
    <section
      data-testid="forms.color-picker"
      className={className}
      data-rcl-color-picker
      aria-label={labels.heading ?? "Color picker"}
    >
      <StyleSheet name="colorpicker-1-0-3-1" css={styles} />
      {labels.heading ? <h3 data-rcl-color-picker-heading>{labels.heading}</h3> : null}
      {allowGradient ? (
        <div data-rcl-color-picker-row>
          <button
            type="button"
            data-testid={`forms.color-picker-${testIdPrefix}-slot-0`}
            data-rcl-color-picker-slot
            style={colors[0] ? { backgroundColor: colors[0] } : undefined}
            onClick={() => setActiveSlot(0)}
            aria-label={labels.primary ?? labels.heading ?? "Color picker"}
            aria-pressed={activeSlot === 0}
          />
          {secondaryOpen ? (
            <>
              <button
                type="button"
                data-testid={`forms.color-picker-${testIdPrefix}-slot-1`}
                data-rcl-color-picker-slot
                style={colors[1] ? { backgroundColor: colors[1] } : undefined}
                onClick={() => setActiveSlot(1)}
                aria-label={labels.secondary ?? labels.heading ?? "Color picker"}
                aria-pressed={activeSlot === 1}
              />
              <button
                type="button"
                data-testid={`forms.color-picker-${testIdPrefix}-remove-gradient`}
                data-rcl-color-picker-remove
                onClick={() => {
                  setActiveSlot(0);
                  onChange(serializeColorValue(colors.slice(0, 1)));
                }}
                aria-label={labels.removeGradient ?? "Remove gradient"}
              >
                <RemoveIcon aria-hidden data-rcl-color-picker-mark />
              </button>
            </>
          ) : (
            <button
              type="button"
              data-testid={`forms.color-picker-${testIdPrefix}-add-gradient`}
              data-rcl-color-picker-add
              onClick={() => setActiveSlot(1)}
            >
              <AddIcon aria-hidden data-rcl-color-picker-mark />
              {labels.addGradient ?? "Add gradient"}
            </button>
          )}
        </div>
      ) : null}
      <div data-rcl-color-picker-row>
        <button
          type="button"
          data-testid={`forms.color-picker-${testIdPrefix}-transparent`}
          data-rcl-color-picker-transparent
          data-selected={transparent}
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
          data-testid={`forms.color-picker-${testIdPrefix}-custom`}
          data-rcl-color-picker-custom
          data-selected={isHexColor(activeColor) && !palette.includes(activeColor)}
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
            <CustomIcon
              aria-hidden
              data-rcl-color-picker-mark
              style={{ color: "var(--color-muted-foreground)" }}
            />
          )}
          <input
            type="color"
            data-testid={`forms.color-picker-${testIdPrefix}-custom-input`}
            data-rcl-color-picker-input
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
        <div data-testid={`${testIdPrefix}-recents`} data-rcl-color-picker-recents>
          <p>{labels.recents ?? "Recent colors"}</p>
          <div data-rcl-color-picker-row>
            {recentColors.map((color) => swatch(color, `recent-${color}`))}
          </div>
        </div>
      ) : null}
    </section>
  );
}
