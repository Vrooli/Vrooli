/**
 * @vrooliComponentSource react-component-library:Pressable
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption b11fce87-57ac-4860-b70f-75c25987167f
 * @vrooliComponentAppliedAt 2026-08-20T01:50:37Z
 * @vrooliComponentSourceSha256 2dd98ba6b5fdc594c07014c6b28bf1ceb63ad53b4eb2b382c020bc12e40d5d17
 * @vrooliComponentDriftHash a359edbd880ce2cb6eccb331ccaa1fdb2fbeaf68440fd2f4644526610d2a2a80
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
[data-rcl-pressable-content] { position: relative; display: inline-flex; align-items: center; min-inline-size: 0; }
[data-rcl-pressable-label], [data-rcl-pressable-pending] { display: inline-flex; align-items: center; gap: var(--space-2xs); }
[data-rcl-pressable-pending] { position: absolute; inset: 0; justify-content: center; visibility: hidden; white-space: nowrap; }
[data-rcl-pressable][data-rcl-pending="true"] [data-rcl-pressable-label] { visibility: hidden; }
[data-rcl-pressable][data-rcl-pending="true"] [data-rcl-pressable-pending] { visibility: visible; }
[data-rcl-control][data-control-size="icon"] [data-rcl-pressable-pending] > span:not([data-rcl-pressable-spinner]) { position: absolute; inline-size: 1px; block-size: 1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }
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
            <span
              data-rcl-pressable-label
              aria-hidden={pending && size !== "icon"}
            >
              {children}
            </span>
            <span
              data-rcl-pressable-pending
              aria-hidden={!pending || size === "icon"}
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
