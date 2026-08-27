/** @vrooliComponentSource controls.toolbar */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import {
  createRef,
  useRef,
  useState,
  type ReactNode,
  type RefObject,
} from "react";
import {
  ControlBase,
  type ControlVariant,
} from "../../../ControlBase/versions/1.0.0/ControlBase";
import { Toggle } from "../../../Toggle/versions/1.0.0/Toggle";
import { useRovingFocus } from "../../../../hooks/useRovingFocus/versions/1.0.0/useRovingFocus";

export interface ToolbarItem {
  id: string;
  label: ReactNode;
  ariaLabel?: string;
  kind?: "button" | "toggle";
  variant?: ControlVariant;
  disabled?: boolean;
  pressed?: boolean;
  defaultPressed?: boolean;
  onSelect?: () => void;
  onPressedChange?: (pressed: boolean) => void;
}

export interface ToolbarProps {
  items: ToolbarItem[];
  label: string;
  orientation?: "horizontal" | "vertical";
  size?: "xs" | "sm" | "md";
}

const styleSheet = `
[data-rcl-toolbar] {
  box-sizing: border-box;
  display: flex;
  align-items: center;
  gap: var(--space-2xs);
  max-inline-size: 100%;
  min-block-size: var(--tap-target-min);
  padding: var(--space-2xs);
  overflow-x: auto;
  border: var(--border-hairline) solid var(--color-border);
  border-radius: var(--radius-panel);
  background: color-mix(in srgb, var(--color-surface-raised) 94%, var(--color-primary));
  box-shadow: var(--elev-raised);
  scrollbar-width: none;
}
[data-rcl-toolbar]::-webkit-scrollbar { display: none; }
[data-rcl-toolbar][aria-orientation="vertical"] { flex-direction: column; align-items: stretch; overflow-x: hidden; overflow-y: auto; }
[data-rcl-toolbar-item] { flex: 0 0 auto; }
[data-rcl-toolbar][aria-orientation="vertical"] [data-rcl-toolbar-item] > button { inline-size: 100%; justify-content: flex-start; }
@media (max-width: 30rem) {
  [data-rcl-toolbar][aria-orientation="horizontal"] { flex-wrap: wrap; overflow-x: visible; border-radius: var(--radius-control); }
  [data-rcl-toolbar][aria-orientation="horizontal"] [data-rcl-toolbar-item] { flex: 1 1 calc(50% - var(--space-2xs)); min-inline-size: 0; }
  [data-rcl-toolbar][aria-orientation="horizontal"] [data-rcl-toolbar-item] > button { inline-size: 100%; min-width: var(--tap-target-min); }
}

`;

function ToolbarStyles() {
  return (
    <StyleSheet name="toolbar-1-0-0-1" css={styleSheet} />
  );
}

export function Toolbar({
  items,
  label,
  orientation = "horizontal",
  size = "sm",
}: ToolbarProps) {
  const firstEnabled = Math.max(
    0,
    items.findIndex((item) => !item.disabled),
  );
  const [activeIndex, setActiveIndex] = useState(firstEnabled);
  const refs = useRef<RefObject<HTMLButtonElement>[]>([]);
  const itemRefs = items.map((_, index) => {
    if (!refs.current[index])
      refs.current[index] = createRef<HTMLButtonElement>();
    return refs.current[index];
  });
  const disabledIndices = items.flatMap((item, index) =>
    item.disabled ? [index] : [],
  );
  const handleKeyDown = useRovingFocus(itemRefs, activeIndex, setActiveIndex, {
    orientation,
    disabledIndices,
  });

  return (
    <>
      <ToolbarStyles />
      <div
        role="toolbar"
        aria-label={label}
        aria-orientation={orientation}
        data-rcl-toolbar
      >
        {items.map((item, index) => {
          const common = {
            ref: itemRefs[index],
            "aria-label": item.ariaLabel,
            disabled: item.disabled,
            size,
            "data-rcl-toolbar-item-control": true,
            onFocus: () => setActiveIndex(index),
            onKeyDown: handleKeyDown,
          } as const;
          return (
            <span key={item.id} data-rcl-toolbar-item>
              {item.kind === "toggle" ? (
                <Toggle
                  {...common}
                  defaultPressed={item.defaultPressed}
                  pressed={item.pressed}
                  onPressedChange={(pressed) => {
                    item.onPressedChange?.(pressed);
                    item.onSelect?.();
                  }}
                >
                  {item.label}
                </Toggle>
              ) : (
                <ControlBase
                  {...common}
                  variant={item.variant ?? "secondary"}
                  type="button"
                  onClick={() => item.onSelect?.()}
                >
                  {item.label}
                </ControlBase>
              )}
            </span>
          );
        })}
      </div>
    </>
  );
}
