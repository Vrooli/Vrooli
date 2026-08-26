/** @vrooliComponentSource motion.auto-animate-layout */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import {
  useLayoutEffect,
  useRef,
  type CSSProperties,
  type ReactNode,
} from "react";
import { LayoutGroup } from "@vrooli/react-component-library/LayoutGroup/1.0.0";
import { useReducedMotion } from "@vrooli/react-component-library/useReducedMotion/1.0.0";

export interface AutoAnimateLayoutProps {
  children: ReactNode;
  duration?: number;
  easing?: string;
  disabled?: boolean;
  className?: string;
  style?: CSSProperties;
}

const styles = `
[data-rcl-auto-animate-layout] { min-inline-size: 0; }
[data-rcl-auto-animate-layout] [data-layout-key] { will-change: transform; }
@media (prefers-reduced-motion: reduce) { [data-rcl-auto-animate-layout] [data-layout-key] { will-change: auto; } }
`;

type RectSnapshot = {
  left: number;
  top: number;
  width: number;
  height: number;
};

function snapshot(group: HTMLElement) {
  const result = new Map<string, RectSnapshot>();
  group.querySelectorAll<HTMLElement>("[data-layout-key]").forEach((node) => {
    const key = node.dataset.layoutKey;
    if (!key) return;
    const rect = node.getBoundingClientRect();
    result.set(key, {
      left: rect.left,
      top: rect.top,
      width: rect.width,
      height: rect.height,
    });
  });
  return result;
}

export const AutoAnimateLayout = withClassName(function AutoAnimateLayout({
  children,
  duration = 180,
  easing = "cubic-bezier(.2,.8,.2,1)",
  disabled = false,
  className,
  style,
}: AutoAnimateLayoutProps) {
  const groupRef = useRef<HTMLDivElement>(null);
  const previous = useRef<Map<string, RectSnapshot>>(new Map());
  const reducedMotion = useReducedMotion();
  useLayoutEffect(() => {
    const group = groupRef.current;
    if (!group) return;
    const next = snapshot(group);
    const animations: Animation[] = [];
    if (!disabled && !reducedMotion && previous.current.size > 0) {
      next.forEach((current, key) => {
        const before = previous.current.get(key);
        if (!before) return;
        const dx = before.left - current.left;
        const dy = before.top - current.top;
        const scaleX = current.width ? before.width / current.width : 1;
        const scaleY = current.height ? before.height / current.height : 1;
        if (
          Math.abs(dx) < 0.5 &&
          Math.abs(dy) < 0.5 &&
          Math.abs(scaleX - 1) < 0.01 &&
          Math.abs(scaleY - 1) < 0.01
        )
          return;
        const node = group.querySelector<HTMLElement>(
          `[data-layout-key="${CSS.escape(key)}"]`,
        );
        if (!node || typeof node.animate !== "function") return;
        animations.push(
          node.animate(
            [
              {
                transform: `translate(${dx}px, ${dy}px) scale(${scaleX}, ${scaleY})`,
              },
              { transform: "translate(0, 0) scale(1, 1)" },
            ],
            { duration, easing, fill: "both" },
          ),
        );
      });
    }
    previous.current = next;
    return () => animations.forEach((animation) => animation.cancel());
  }, [children, disabled, duration, easing, reducedMotion]);
  return (
    <div data-testid="motion.auto-animate-layout"
      ref={groupRef}
      data-rcl-auto-animate-layout
      className={className}
      style={style}
    >
      <style
        data-rcl-auto-animate-layout-styles
        dangerouslySetInnerHTML={{ __html: styles }}
      />
      <LayoutGroup>{children}</LayoutGroup>
    </div>
  );
});
