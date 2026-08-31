/** @vrooliComponentSource primitives.aspect-ratio */
import type { CSSProperties, HTMLAttributes, ReactNode } from "react";

const styles = `
[data-rcl-aspect-ratio] { position: relative; display: block; inline-size: 100%; overflow: hidden; background: var(--color-surface-muted); }
[data-rcl-aspect-ratio] > [data-rcl-aspect-ratio-content] { min-inline-size: 0; min-block-size: 0; }
`;

export interface AspectRatioProps extends HTMLAttributes<HTMLDivElement> {
  children?: ReactNode;
  ratio?: number | string;
  contentStyle?: CSSProperties;
}

export function AspectRatio({
  children,
  contentStyle,
  ratio = "16 / 9",
  style,
  ...props
}: AspectRatioProps) {
  return (
    <>
      <style
        data-rcl-aspect-ratio-styles
        dangerouslySetInnerHTML={{ __html: styles }}
      />
      <div
        {...props}
        data-rcl-aspect-ratio
        style={{ aspectRatio: ratio, ...style }}
      >
        <div
          data-rcl-aspect-ratio-content
          style={{ blockSize: "100%", inlineSize: "100%", ...contentStyle }}
        >
          {children}
        </div>
      </div>
    </>
  );
}
