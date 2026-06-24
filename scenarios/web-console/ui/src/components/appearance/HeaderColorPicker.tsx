import { useState } from "react";
import { Pipette, Plus, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { HEADER_COLORS } from "../../consts/config";
import { cn } from "../../lib/classnames";
import { strings } from "../../consts/strings";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import {
  isHexColor,
  parsePaneColor,
  serializePaneColor,
} from "../../lib/paneColor";

interface HeaderColorPickerProps {
  /** The stored encoding: "transparent" | "#hex" | "#hex|#hex". */
  currentColor: string;
  onSelectColor: (color: string) => void;
  testIdPrefix?: string;
}

/** A valid hex color the native color input can seed itself with. */
const FALLBACK_CUSTOM_COLOR = "#4dabf7";

export default function HeaderColorPicker({
  currentColor,
  onSelectColor,
  testIdPrefix = "appearance",
}: HeaderColorPickerProps) {
  const { t } = useTranslation();
  // Guarded reads: some component tests mock the store partially.
  const recentHeaderColors = useWorkspaceStore((s) => s.recentHeaderColors) ?? [];
  const addRecentHeaderColor = useWorkspaceStore((s) => s.addRecentHeaderColor);

  const { colors, isTransparent } = parsePaneColor(currentColor);
  // Which slot the next palette/recent/custom selection writes to. When the
  // user opens a secondary slot that has no color yet, activeSlot is 1 while
  // `colors` still has length 1 (a "pending" secondary).
  const [activeSlot, setActiveSlot] = useState<0 | 1>(0);
  const secondaryOpen = colors.length > 1 || activeSlot === 1;

  /** Apply a freshly-picked color into the active slot and persist. */
  const pick = (color: string) => {
    const next = activeSlot === 0
      ? [color, colors[1]]
      : [colors[0] ?? color, color];
    const cleaned = next.filter((c): c is string => typeof c === "string" && isHexColor(c));
    onSelectColor(serializePaneColor(cleaned));
    addRecentHeaderColor?.(color);
  };

  const selectTransparent = () => {
    setActiveSlot(0);
    onSelectColor("transparent");
  };

  const openSecondary = () => setActiveSlot(1);

  const removeSecondary = () => {
    setActiveSlot(0);
    onSelectColor(serializePaneColor(colors[0] ? [colors[0]] : []));
  };

  /** A color matches the active slot when it's currently shown there. */
  const isSelected = (color: string): boolean =>
    !isTransparent && colors[activeSlot] === color;

  const activeColor = colors[activeSlot];
  const customSwatchColor = activeColor && isHexColor(activeColor)
    ? activeColor
    : FALLBACK_CUSTOM_COLOR;
  const isCustomActive = Boolean(
    activeColor && isHexColor(activeColor) && !(HEADER_COLORS as readonly string[]).includes(activeColor),
  );

  const slotChip = (slot: 0 | 1) => {
    const color = colors[slot];
    const active = activeSlot === slot;
    return (
      <button
        type="button"
        data-testid={`${testIdPrefix}-color-slot-${slot}`}
        className={cn(
          "h-7 w-7 rounded-md border",
          active ? "border-wc-accent ring-1 ring-wc-accent" : "border-wc-default",
        )}
        style={color ? { backgroundColor: color } : { background: "rgb(var(--wc-surface-input))" }}
        onClick={() => setActiveSlot(slot)}
        title={color ?? t(strings.appearance.noColorTitle)}
        aria-pressed={active}
      />
    );
  };

  return (
    <section>
      <h3 className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted mb-2">
        {t(strings.appearance.headerColorHeading)}
      </h3>

      {/* Active-slot chips: one or two colors, with add/remove secondary. */}
      <div className="mb-2 flex items-center gap-1.5">
        {slotChip(0)}
        {secondaryOpen ? (
          <div className="flex items-center gap-1">
            {slotChip(1)}
            <button
              type="button"
              data-testid={`${testIdPrefix}-remove-secondary`}
              className="flex h-6 w-6 items-center justify-center rounded text-wc-text-muted hover:bg-wc-surface-input hover:text-wc-text-primary"
              onClick={removeSecondary}
              title={t(strings.appearance.removeSecondaryColor)}
              aria-label={t(strings.appearance.removeSecondaryColor)}
            >
              <X className="h-3.5 w-3.5" />
            </button>
          </div>
        ) : (
          !isTransparent && (
            <button
              type="button"
              data-testid={`${testIdPrefix}-add-secondary`}
              className="flex h-7 w-7 items-center justify-center rounded-md border border-dashed border-wc-default text-wc-text-muted hover:text-wc-text-primary"
              onClick={openSecondary}
              title={t(strings.appearance.addSecondaryColor)}
              aria-label={t(strings.appearance.addSecondaryColor)}
            >
              <Plus className="h-3.5 w-3.5" />
            </button>
          )
        )}
      </div>

      <div className="flex flex-wrap gap-1.5">
        {/* Transparent option */}
        <button
          type="button"
          data-testid={`${testIdPrefix}-header-color-transparent`}
          className={cn(
            "h-6 w-6 rounded-full border",
            isTransparent ? "border-wc-accent ring-1 ring-wc-accent" : "border-wc-default",
          )}
          style={{ background: "rgb(var(--wc-surface-input))" }}
          onClick={selectTransparent}
          title={t(strings.appearance.noColorTitle)}
        />
        {HEADER_COLORS.map((color) => (
          <button
            key={color}
            type="button"
            data-testid={`${testIdPrefix}-header-color-${color}`}
            className={cn(
              "h-6 w-6 rounded-full border",
              isSelected(color) ? "border-wc-accent ring-1 ring-wc-accent" : "border-wc-default",
            )}
            style={{ backgroundColor: color }}
            onClick={() => pick(color)}
            title={color}
          />
        ))}
        {/* Custom color — opens the native color picker, writing the active slot. */}
        <label
          data-testid={`${testIdPrefix}-header-color-custom`}
          className={cn(
            "relative h-6 w-6 rounded-full border flex items-center justify-center cursor-pointer overflow-hidden",
            isCustomActive ? "border-wc-accent ring-1 ring-wc-accent" : "border-wc-default",
          )}
          style={isCustomActive ? { backgroundColor: customSwatchColor } : { background: "rgb(var(--wc-surface-input))" }}
          title={isCustomActive ? customSwatchColor : t(strings.appearance.customColorTitle)}
        >
          {!isCustomActive && <Pipette className="h-3 w-3 text-wc-text-muted" />}
          <input
            type="color"
            data-testid={`${testIdPrefix}-header-color-custom-input`}
            className="absolute inset-0 h-full w-full cursor-pointer opacity-0"
            value={customSwatchColor}
            onChange={(e) => pick(e.target.value)}
            aria-label={t(strings.appearance.customColorTitle)}
          />
        </label>
      </div>

      {/* Recent colors row */}
      {recentHeaderColors.length > 0 && (
        <div data-testid={`${testIdPrefix}-recent-row`} className="mt-3">
          <h4 className="mb-1.5 text-[10px] font-semibold uppercase tracking-wider text-wc-text-faint">
            {t(strings.appearance.recentColorsHeading)}
          </h4>
          <div className="flex flex-wrap gap-1.5">
            {recentHeaderColors.map((color) => (
              <button
                key={color}
                type="button"
                data-testid={`${testIdPrefix}-header-recent-${color}`}
                className={cn(
                  "h-6 w-6 rounded-full border",
                  isSelected(color) ? "border-wc-accent ring-1 ring-wc-accent" : "border-wc-default",
                )}
                style={{ backgroundColor: color }}
                onClick={() => pick(color)}
                title={color}
              />
            ))}
          </div>
        </div>
      )}
    </section>
  );
}
