/**
 * @libraryId react-component-library:Pressable
 * @displayName Pressable
 * @description The shared press contract for controls, with bounded tone, pending, focus, and touch-target behavior.
 * @version 1.0.1
 * @tags ["control","interaction","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource controls.pressable */
import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from "react";
import {
  ControlBase,
  type ControlDensity,
  type ControlShape,
  type ControlSize,
  type ControlVariant,
} from "@vrooli/react-component-library/ControlBase/1.1.0";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";

const pressableStyles = `
[data-rcl-pressable-content] { position: relative; display: inline-flex; align-items: center; min-inline-size: 0; max-inline-size: 100%; }
[data-rcl-pressable-label], [data-rcl-pressable-pending] { display: inline-flex; align-items: center; gap: var(--space-2xs); min-inline-size: 0; max-inline-size: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
[data-rcl-pressable-pending] { position: absolute; inset: 0; justify-content: center; visibility: hidden; white-space: nowrap; }
[data-rcl-pressable][data-rcl-pending="true"] [data-rcl-pressable-label] { visibility: hidden; }
[data-rcl-pressable][data-rcl-pending="true"] [data-rcl-pressable-pending] { visibility: visible; }
[data-rcl-control][data-control-size="icon"] [data-rcl-pressable-pending] > span:not([data-rcl-pressable-spinner]) { position: absolute; }
[data-rcl-pressable-spinner] { inline-size: var(--space-sm); block-size: var(--space-sm); flex: 0 0 auto; border: var(--border-strong) solid color-mix(in srgb, currentColor 28%, transparent); border-block-start-color: currentColor; border-radius: var(--radius-pill); animation: rcl-pressable-spin var(--dur-moderate) linear infinite; }
@keyframes rcl-pressable-spin { to { transform: rotate(360deg); } }
`;

export interface PressableProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, "children"> {
  children: ReactNode;
  pending?: boolean;
  pendingLabel?: ReactNode;
  tone?: ControlVariant;
  size?: ControlSize;
  density?: ControlDensity;
  shape?: ControlShape;
}

export const Pressable = forwardRef<HTMLButtonElement, PressableProps>(function Pressable(
  {
    pending = false,
    pendingLabel = "Working…",
    tone = "primary",
    children,
    disabled,
    "aria-busy": ariaBusy,
    "aria-disabled": ariaDisabled,
    size,
    density,
    shape,
    style,
    ...props
  },
  ref,
) {
  const variant: ControlVariant = tone;
  useLibraryStyleSheet("pressable", pressableStyles);
  return (
    <ControlBase
        data-testid="controls.pressable"
        {...props}
        disabled={disabled || pending}
        aria-busy={ariaBusy ?? (pending || undefined)}
        aria-disabled={ariaDisabled ?? (disabled || pending || undefined)}
        variant={variant}
        size={size}
        density={density}
        shape={shape}
        data-rcl-pressable="true"
        data-rcl-pending={pending ? "true" : "false"}
        ref={ref}
        style={style}
      >
        <span data-rcl-pressable-content>
          <span data-rcl-pressable-label aria-hidden={pending && size !== "icon"}>
            {children}
          </span>
          <span
            data-rcl-pressable-pending
            aria-hidden={!pending || size === "icon"}
            aria-live="polite"
          >
            <span aria-hidden="true" data-rcl-pressable-spinner />
            <span className="rcl-visually-hidden">{pendingLabel}</span>
          </span>
        </span>
      </ControlBase>
  );
});
