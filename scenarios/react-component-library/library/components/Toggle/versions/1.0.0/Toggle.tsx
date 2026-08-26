/** @vrooliComponentSource controls.toggle */
import {
  forwardRef,
  useState,
  type ButtonHTMLAttributes,
  type MouseEvent,
  type ReactNode,
} from "react";
import {
  ControlBase,
  type ControlDensity,
  type ControlShape,
  type ControlSize,
  type ControlVariant,
} from "@vrooli/react-component-library/ControlBase/1.0.0";

export interface ToggleProps
  extends Omit<
    ButtonHTMLAttributes<HTMLButtonElement>,
    "children" | "onClick"
  > {
  children: ReactNode;
  pressed?: boolean;
  defaultPressed?: boolean;
  onPressedChange?: (pressed: boolean) => void;
  onClick?: (event: MouseEvent<HTMLButtonElement>) => void;
  size?: ControlSize;
  density?: ControlDensity;
  shape?: ControlShape;
  pressedVariant?: ControlVariant;
  unpressedVariant?: ControlVariant;
}

const styleSheet = `
[data-rcl-toggle] { transition: transform var(--dur-quick) var(--ease-standard), filter var(--dur-quick) var(--ease-standard), box-shadow var(--dur-quick) var(--ease-standard); }
[data-rcl-toggle][data-state="on"] { box-shadow: 0 0 0 var(--space-3xs) color-mix(in srgb, var(--color-primary) 18%, transparent), var(--elev-raised); }
[data-rcl-toggle][data-state="on"]::after { content: ""; position: absolute; inset: var(--space-3xs); border: var(--border-hairline) solid color-mix(in srgb, var(--color-primary-foreground) 24%, transparent); border-radius: inherit; pointer-events: none; }
@media (prefers-reduced-motion: reduce) { [data-rcl-toggle] { transition: none; } }
`;

function ToggleStyles() {
  return (
    <style
      data-rcl-toggle-styles
      dangerouslySetInnerHTML={{ __html: styleSheet }}
    />
  );
}

export const Toggle = forwardRef<HTMLButtonElement, ToggleProps>(
  function Toggle(
    {
      children,
      pressed,
      defaultPressed = false,
      onPressedChange,
      onClick,
      disabled,
      pressedVariant = "primary",
      unpressedVariant = "secondary",
      ...props
    },
    ref,
  ) {
    const [internalPressed, setInternalPressed] = useState(defaultPressed);
    const isControlled = pressed !== undefined;
    const resolvedPressed = isControlled ? pressed : internalPressed;

    const handleClick = (event: MouseEvent<HTMLButtonElement>) => {
      const next = !resolvedPressed;
      if (!disabled) {
        if (!isControlled) setInternalPressed(next);
        onPressedChange?.(next);
      }
      onClick?.(event);
    };

    return (
      <>
        <ToggleStyles data-testid="controls.toggle" />
        <ControlBase
          {...props}
          ref={ref}
          {...(disabled ? { disabled: true } : {})}
          data-rcl-toggle="true"
          data-state={resolvedPressed ? "on" : "off"}
          aria-pressed={resolvedPressed}
          onClick={handleClick}
          variant={resolvedPressed ? pressedVariant : unpressedVariant}
          size={props.size}
          density={props.density}
          shape={props.shape}
        >
          {children}
        </ControlBase>
      </>
    );
  },
);
