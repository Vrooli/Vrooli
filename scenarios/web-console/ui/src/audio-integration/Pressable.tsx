/**
 * @vrooliComponentSource react-component-library:Pressable
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption fca0af9a-3a97-46e6-b43a-b8c6504d9361
 * @vrooliComponentAppliedAt 2026-08-09T14:56:08Z
 * @vrooliComponentSourceSha256 c602c0925568c371342c5099f747059a4e4804392e532f5c10e630d7ae3d7532
 * @vrooliComponentDriftHash 3c9f0b0a5da5221d46540c17755633a801b6b6f1ea5a8b08e736cf0a4d306ef5
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from "react";
import {
  ControlBase,
  type ControlDensity,
  type ControlShape,
  type ControlSize,
  type ControlVariant,
} from "./ControlBase";

const pressableStyles = `
[data-rcl-pressable-content] { display: inline-grid; place-items: center; min-inline-size: 0; }
[data-rcl-pressable-label], [data-rcl-pressable-pending] { grid-area: 1 / 1; display: inline-flex; align-items: center; gap: var(--space-2xs); }
[data-rcl-pressable-pending] { visibility: hidden; }
[data-rcl-pressable][data-rcl-pending="true"] [data-rcl-pressable-label] { visibility: hidden; }
[data-rcl-pressable][data-rcl-pending="true"] [data-rcl-pressable-pending] { visibility: visible; }
[data-rcl-pressable-spinner] { inline-size: var(--space-sm); block-size: var(--space-sm); flex: 0 0 auto; border: var(--border-strong) solid color-mix(in srgb, currentColor 28%, transparent); border-block-start-color: currentColor; border-radius: var(--radius-pill); animation: rcl-pressable-spin var(--dur-moderate) linear infinite; }
@keyframes rcl-pressable-spin { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) { [data-rcl-pressable-spinner] { animation: none; } }
`;

export interface PressableProps
  extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, "children"> {
  children: ReactNode;
  pending?: boolean;
  pendingLabel?: ReactNode;
  tone?: ControlVariant;
  size?: ControlSize;
  density?: ControlDensity;
  shape?: ControlShape;
}

export const Pressable = forwardRef<HTMLButtonElement, PressableProps>(
  function Pressable(
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
    return (
      <>
        <style
          data-rcl-pressable-styles
          dangerouslySetInnerHTML={{ __html: pressableStyles }}
        />
        <ControlBase
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
            <span data-rcl-pressable-label aria-hidden={pending}>
              {children}
            </span>
            <span
              data-rcl-pressable-pending
              aria-hidden={!pending}
              aria-live="polite"
            >
              <span aria-hidden="true" data-rcl-pressable-spinner />
              <span>{pendingLabel}</span>
            </span>
          </span>
        </ControlBase>
      </>
    );
  },
);
