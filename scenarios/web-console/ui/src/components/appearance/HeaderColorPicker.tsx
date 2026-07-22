import { useCallback, useEffect, useRef, useState } from "react";
import { Check, Pipette, Plus, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { HEADER_COLORS } from "../../consts/config";
import { cn } from "../../lib/classnames";
import { strings } from "../../consts/strings";
import { useWorkspaceStore } from "../../stores/useWorkspaceStore";
import {
  isHexColor,
  isLightColor,
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

  /**
   * Apply a freshly-picked color into the active slot and persist. Recording
   * into recents is skippable: the native color input streams a pick per drag
   * tick, and those intermediate colors must not flood the recents row — only
   * a committed pick (swatch click, or picker close) is recorded.
   */
  const pick = (color: string, { record = true } = {}) => {
    const next = activeSlot === 0
      ? [color, colors[1]]
      : [colors[0] ?? color, color];
    const cleaned = next.filter((c): c is string => typeof c === "string" && isHexColor(c));
    onSelectColor(serializePaneColor(cleaned));
    if (record) addRecentHeaderColor?.(color);
  };

  // The native color picker fires BOTH `input` and `change` per drag tick in
  // Chromium, so no picker event marks "the user is done". Instead the last
  // dragged value parks here and is committed to recents only when the user
  // demonstrably moves on: the input blurs, or the picker unmounts (modal
  // closed / section switched).
  const pendingCustomRef = useRef<string | null>(null);
  const flushPendingCustom = useCallback(() => {
    const pending = pendingCustomRef.current;
    pendingCustomRef.current = null;
    if (pending && isHexColor(pending)) addRecentHeaderColor?.(pending);
  }, [addRecentHeaderColor]);
  // Effect returns the flush as its cleanup: with a stable store action this
  // runs exactly once, at unmount.
  useEffect(() => flushPendingCustom, [flushPendingCustom]);

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

  /** Contrast-correct check mark shown inside a selected swatch. */
  const checkMark = (color?: string) => (
    <Check
      className={cn(
        "h-3.5 w-3.5",
        !color
          ? "text-wc-text-primary"
          : isLightColor(color)
            ? "text-black/70"
            : "text-white",
      )}
      aria-hidden="true"
    />
  );

  /** Round color swatch shared by the palette and recents rows. */
  const swatch = (color: string, testId: string) => {
    const selected = isSelected(color);
    return (
      <button
        key={testId}
        type="button"
        data-testid={testId}
        className={cn(
          "flex h-7 w-7 items-center justify-center rounded-full border transition-shadow",
          selected ? "border-wc-accent ring-2 ring-wc-accent/60" : "border-wc-default hover:ring-1 hover:ring-wc-text-faint",
        )}
        style={{ backgroundColor: color }}
        onClick={() => pick(color)}
        title={color}
        aria-pressed={selected}
      >
        {selected && checkMark(color)}
      </button>
    );
  };

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
      <div className="mb-2.5 flex items-center gap-1.5">
        {slotChip(0)}
        {secondaryOpen ? (
          <div className="flex items-center gap-1.5">
            {slotChip(1)}
            <span className="text-[11px] font-medium text-wc-text-muted">
              {t(strings.appearance.gradientLabel)}
            </span>
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
              className="flex h-7 items-center gap-1 rounded-md border border-dashed border-wc-default px-2 text-[11px] font-medium text-wc-text-muted hover:border-wc-text-faint hover:text-wc-text-primary"
              onClick={openSecondary}
              title={t(strings.appearance.addSecondaryColor)}
            >
              <Plus className="h-3 w-3" aria-hidden="true" />
              {t(strings.appearance.gradientLabel)}
            </button>
          )
        )}
      </div>

      <div className="flex flex-wrap gap-2">
        {/* Transparent option */}
        <button
          type="button"
          data-testid={`${testIdPrefix}-header-color-transparent`}
          className={cn(
            "flex h-7 w-7 items-center justify-center rounded-full border transition-shadow",
            isTransparent ? "border-wc-accent ring-2 ring-wc-accent/60" : "border-wc-default hover:ring-1 hover:ring-wc-text-faint",
          )}
          style={{ background: "rgb(var(--wc-surface-input))" }}
          onClick={selectTransparent}
          title={t(strings.appearance.noColorTitle)}
          aria-pressed={isTransparent}
        >
          {isTransparent && checkMark()}
        </button>
        {HEADER_COLORS.map((color) => swatch(color, `${testIdPrefix}-header-color-${color}`))}
        {/* Custom color — opens the native color picker, writing the active slot. */}
        <label
          data-testid={`${testIdPrefix}-header-color-custom`}
          className={cn(
            "relative h-7 w-7 rounded-full border flex items-center justify-center cursor-pointer overflow-hidden",
            isCustomActive ? "border-wc-accent ring-2 ring-wc-accent/60" : "border-wc-default hover:ring-1 hover:ring-wc-text-faint",
          )}
          style={isCustomActive ? { backgroundColor: customSwatchColor } : { background: "rgb(var(--wc-surface-input))" }}
          title={isCustomActive ? customSwatchColor : t(strings.appearance.customColorTitle)}
        >
          {isCustomActive
            ? checkMark(customSwatchColor)
            : <Pipette className="h-3 w-3 text-wc-text-muted" aria-hidden="true" />}
          <input
            type="color"
            data-testid={`${testIdPrefix}-header-color-custom-input`}
            className="absolute inset-0 h-full w-full cursor-pointer opacity-0"
            value={customSwatchColor}
            onChange={(e) => {
              pick(e.target.value, { record: false });
              pendingCustomRef.current = e.target.value;
            }}
            onBlur={flushPendingCustom}
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
          <div className="flex flex-wrap gap-2">
            {recentHeaderColors.map((color) => swatch(color, `${testIdPrefix}-header-recent-${color}`))}
          </div>
        </div>
      )}
    </section>
  );
}
