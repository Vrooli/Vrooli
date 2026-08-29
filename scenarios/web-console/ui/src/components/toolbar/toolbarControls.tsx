// DOC: docs/reference/configuration.md#mobile-toolbar-layout
/**
 * The one place a toolbar control is turned into pixels.
 *
 * Three surfaces render controls: the live toolbar, the settings preview, and
 * the More sheet. They all call `renderToolbarControl`, which is what keeps the
 * preview honest — there is no second implementation for it to drift from.
 * Size comes from the layout engine, never from a class name, so the same node
 * renders at 32px in a dense toolbar and at 44px inside the sheet.
 *
 * [REQ:P0-007b] Terminal Key/Chord Mapping
 */
import type { CSSProperties, ReactNode } from "react";
import { useCallback } from "react";
import {
  ArrowDown as ArrowDownIcon,
  ArrowLeft as ArrowLeftIcon,
  ArrowRight as ArrowRightIcon,
  ArrowUp as ArrowUpIcon,
  Image,
  Library,
  MoreHorizontal,
  Sparkles,
  type LucideIcon,
} from "lucide-react";
import { Button } from "@vrooli/react-component-library/Button/2";
import {
  ARROW_DOWN,
  ARROW_LEFT,
  ARROW_RIGHT,
  ARROW_UP,
  ENTER_KEY,
  ESC_KEY,
  TAB_KEY,
  type ToolbarKey,
} from "../../consts/toolbar-keys";
import type { ModifierState } from "../../consts/toolbar-keys";
import { capWidth, type ToolbarMetrics, type ToolbarSlot } from "../../lib/toolbarLayout";
import { cn } from "../../lib/classnames";
import { slugify } from "../../lib/slugify";
import { useHoldRepeat } from "../../hooks/useHoldRepeat";
import VoiceMicButton, { type VoiceMicButtonProps } from "../VoiceMicButton";

/** Modifier ids, in the order they appear on the toolbar. */
export const MODIFIER_KEYS = ["ctrl", "alt", "shift"] as const;

/**
 * Key caps carry the key's own name and are not translated — "Ctrl" is what is
 * printed on the keyboard in every locale this console ships in.
 */
export function modifierLabel(modifier: (typeof MODIFIER_KEYS)[number]): string {
  return modifier.charAt(0).toUpperCase() + modifier.slice(1);
}
/** Named keys of the `special` control, in order. */
export const SPECIAL_KEYS: readonly ToolbarKey[] = [ESC_KEY, TAB_KEY, ENTER_KEY];

/** Everything a control needs to become an interactive button. */
export interface ToolbarControlContext {
  /** Send a toolbar key through the input gate, applying live modifiers. */
  onKey: (key: ToolbarKey) => void;
  modifiers: ModifierState;
  toggleModifier: (modifier: (typeof MODIFIER_KEYS)[number]) => void;
  onOpenAi?: () => void;
  aiSuggestActive?: boolean;
  onUploadImage?: () => void;
  /** Open the sender-owned snippet picker. */
  onOpenSnippets?: () => void;
  /** Rendered in the `more` slot. Owns its own sheet. */
  moreTrigger?: (props: { className: string; style: CSSProperties; label: string }) => ReactNode;
  voice?: Omit<VoiceMicButtonProps, "size" | "className" | "buttonClassName">;
  /** Localised control names, keyed by control id. */
  labels: Record<string, string>;
  /**
   * Prefix for every control's test id. A control can legitimately appear on
   * two surfaces at once — in the overflow strip and inside the More sheet —
   * and identical test ids across both make `getByTestId` ambiguous.
   */
  testIdPrefix?: string;
  /**
   * Render inert copies for the settings preview: no handlers, no focus, no
   * assistive-tech presence. The visual is identical by construction.
   */
  inert?: boolean;
}

/* RCL owns the actual control painting. Active state is expressed by
 * overriding the RCL semantic tokens, rather than by setting background,
 * border, and color directly and bypassing the library's variants. */
export const activeToolbarControlStyle: CSSProperties = {
  "--color-surface": "rgb(var(--wc-accent) / 0.2)",
  "--color-border": "rgb(var(--wc-accent))",
  "--color-foreground": "rgb(var(--wc-text-primary))",
} as CSSProperties;

const CONTROL_BASE = "flex items-center justify-center p-0 touch-manipulation shrink-0";

function boxStyle(width: number, m: ToolbarMetrics): CSSProperties {
  return { width, height: m.unit, minWidth: width, minHeight: m.unit };
}

function voiceControlSize(m: ToolbarMetrics): VoiceMicButtonProps["size"] {
  if (m.unit <= 32) return "xs";
  if (m.unit <= 40) return "md";
  return "lg";
}

function arrowLabel(key: ToolbarKey): string {
  switch (key.input) {
    case ARROW_UP.input:
      return "Arrow up";
    case ARROW_DOWN.input:
      return "Arrow down";
    case ARROW_LEFT.input:
      return "Arrow left";
    case ARROW_RIGHT.input:
      return "Arrow right";
    default:
      return key.label;
  }
}

function arrowIcon(key: ToolbarKey): LucideIcon {
  switch (key.input) {
    case ARROW_DOWN.input:
      return ArrowDownIcon;
    case ARROW_LEFT.input:
      return ArrowLeftIcon;
    case ARROW_RIGHT.input:
      return ArrowRightIcon;
    case ARROW_UP.input:
    default:
      return ArrowUpIcon;
  }
}

/**
 * Arrow keys are the only toolbar buttons with hold-to-repeat because users
 * routinely need to scan through shell history, long command lines, and TUI
 * views. Other toolbar buttons are one-shot actions where repeat would only
 * cause accidental misfires.
 *
 * Fires on pointerdown and intentionally does NOT bind onClick — pointerdown
 * already dispatches, and a parallel click handler would double-fire on tap.
 */
function ArrowButton({
  keyDef,
  onFire,
  m,
  inert,
  testIdPrefix = "",
}: {
  keyDef: ToolbarKey;
  onFire: (key: ToolbarKey) => void;
  m: ToolbarMetrics;
  inert?: boolean;
  testIdPrefix?: string;
}) {
  const handlers = useHoldRepeat({ onFire: useCallback(() => { onFire(keyDef); }, [onFire, keyDef]) });
  const Icon = arrowIcon(keyDef);
  const label = arrowLabel(keyDef);
  return (
    <Button
      data-testid={`${testIdPrefix}toolbar-key-${slugify(label)}`}
      tabIndex={-1}
      type="button"
      aria-label={label}
      title={label}
      variant="secondary"
      size="sm"
      density="compact"
      {...(inert ? {} : handlers)}
      className={cn(CONTROL_BASE, "rounded border font-medium transition")}
      style={{ ...boxStyle(m.unit, m), paddingInline: 0 }}
    >
      <Icon aria-hidden className="h-4 w-4" />
    </Button>
  );
}

function IconControl({
  testId,
  label,
  icon: Icon,
  onClick,
  active,
  width,
  m,
  inert,
}: {
  testId: string;
  label: string;
  icon: LucideIcon;
  onClick?: () => void;
  active?: boolean;
  width: number;
  m: ToolbarMetrics;
  inert?: boolean;
}) {
  return (
    <Button
      data-testid={testId}
      data-active={active ? "true" : undefined}
      aria-pressed={active}
      tabIndex={-1}
      type="button"
      aria-label={label}
      title={label}
      variant="secondary"
      size="sm"
      density="compact"
      onPointerDown={inert ? undefined : (e) => { e.preventDefault(); }}
      onClick={inert ? undefined : onClick}
      className={cn(CONTROL_BASE, "rounded border font-medium transition")}
      style={{ ...boxStyle(width, m), paddingInline: 0, ...(active ? activeToolbarControlStyle : undefined) }}
    >
      <Icon aria-hidden className="h-4 w-4" />
    </Button>
  );
}

function KeyCap({
  testId,
  label,
  onClick,
  active,
  width,
  m,
  inert,
}: {
  testId: string;
  label: string;
  onClick?: () => void;
  active?: boolean;
  width: number;
  m: ToolbarMetrics;
  inert?: boolean;
}) {
  return (
    <Button
      data-testid={testId}
      data-active={active ? "true" : undefined}
      tabIndex={-1}
      type="button"
      variant="secondary"
      size="sm"
      density="compact"
      onPointerDown={inert ? undefined : (e) => { e.preventDefault(); }}
      onClick={inert ? undefined : onClick}
      className={cn(CONTROL_BASE, "rounded border font-medium transition")}
      style={{
        ...boxStyle(width, m),
        fontSize: `${String(m.fontPx)}px`,
        paddingInline: `${String(m.capPaddingX)}px`,
        ...(active ? activeToolbarControlStyle : undefined),
      }}
    >
      {label}
    </Button>
  );
}

/** Lay a control's parts out on one line at the engine's gap. */
function Group({ gap, children }: { gap: number; children: ReactNode }) {
  return (
    <div className="flex items-stretch" style={{ gap }}>
      {children}
    </div>
  );
}

/**
 * Turn one laid-out slot into its button(s). Every surface calls this; none of
 * them may special-case a control id of their own.
 */
export function renderToolbarControl(
  slot: ToolbarSlot,
  ctx: ToolbarControlContext,
  m: ToolbarMetrics,
): ReactNode {
  const { inert } = ctx;
  const label = ctx.labels[slot.id] ?? slot.id;
  const tid = (name: string) => `${ctx.testIdPrefix ?? ""}${name}`;

  switch (slot.id) {
    case "more":
      return ctx.moreTrigger?.({
        className: cn(CONTROL_BASE, "rounded border border-wc-default bg-wc-surface-input text-wc-text-secondary transition active:bg-wc-accent-active"),
        style: boxStyle(slot.width, m),
        label,
      }) ?? (
        <IconControl
          testId={tid("toolbar-more")}
          label={label}
          icon={MoreHorizontal}
          width={slot.width}
          m={m}
          inert
        />
      );

    case "modifiers":
      return (
        <Group gap={m.gap}>
          {MODIFIER_KEYS.map((mod) => (
            <KeyCap
              key={mod}
              testId={tid(`toolbar-mod-${mod}`)}
              label={modifierLabel(mod)}
              onClick={() => { ctx.toggleModifier(mod); }}
              active={ctx.modifiers[mod]}
              width={capWidth(modifierLabel(mod), m)}
              m={m}
              inert={inert}
            />
          ))}
        </Group>
      );

    case "special":
      return (
        <Group gap={m.gap}>
          {SPECIAL_KEYS.map((key) => (
            <KeyCap
              key={key.label}
              testId={tid(`toolbar-key-${slugify(key.label)}`)}
              label={key.label}
              onClick={() => { ctx.onKey(key); }}
              width={capWidth(key.label, m)}
              m={m}
              inert={inert}
            />
          ))}
        </Group>
      );

    case "arrows":
      return (
        <Group gap={m.gap}>
          {[ARROW_LEFT, ARROW_DOWN, ARROW_UP, ARROW_RIGHT].map((key) => (
            <ArrowButton key={key.label} keyDef={key} onFire={ctx.onKey} m={m} inert={inert} testIdPrefix={ctx.testIdPrefix} />
          ))}
        </Group>
      );

    case "ai":
      return (
        <IconControl
          testId={tid("toolbar-ai")}
          label={label}
          icon={Sparkles}
          onClick={ctx.onOpenAi}
          active={ctx.aiSuggestActive}
          width={slot.width}
          m={m}
          inert={inert}
        />
      );

    case "image":
      return (
        <IconControl
          testId={tid("toolbar-upload-image")}
          label={label}
          icon={Image}
          onClick={ctx.onUploadImage}
          width={slot.width}
          m={m}
          inert={inert}
        />
      );

    case "snippets":
      return (
        <IconControl
          testId={tid("toolbar-snippets")}
          label={label}
          icon={Library}
          onClick={ctx.onOpenSnippets}
          width={slot.width}
          m={m}
          inert={inert}
        />
      );

    case "mic":
      if (!ctx.voice) return null;
      return (
        <div data-testid={tid("toolbar-mic-slot")} className="flex items-stretch" style={{ width: slot.width, height: m.unit }}>
          <VoiceMicButton
            {...ctx.voice}
            testId={tid("voice-mic-btn")}
            size={voiceControlSize(m)}
            className="h-full w-full"
            buttonClassName="flex h-full w-full items-center justify-center"
          />
        </div>
      );

    default:
      return null;
  }
}

/** The two-row D-pad cluster. Rendered beside the row stack, not inside it. */
export function ToolbarDpad({
  ctx,
  m,
  width,
}: {
  ctx: ToolbarControlContext;
  m: ToolbarMetrics;
  width: number;
}) {
  return (
    <div
      data-testid={`${ctx.testIdPrefix ?? ""}toolbar-dpad`}
      className="flex flex-col items-center"
      style={{ gap: m.gap, width }}
    >
      <div className="flex justify-center">
        <ArrowButton keyDef={ARROW_UP} onFire={ctx.onKey} m={m} inert={ctx.inert} testIdPrefix={ctx.testIdPrefix} />
      </div>
      <div className="flex items-center" style={{ gap: m.gap }}>
        {[ARROW_LEFT, ARROW_DOWN, ARROW_RIGHT].map((key) => (
          <ArrowButton key={key.label} keyDef={key} onFire={ctx.onKey} m={m} inert={ctx.inert} testIdPrefix={ctx.testIdPrefix} />
        ))}
      </div>
    </div>
  );
}
