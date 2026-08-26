/**
 * @libraryId react-component-library:AspectRatio
 * @displayName AspectRatio
 * @description A token-backed media frame that reserves space before its content resolves.
 * @version 1.0.1
 * @tags ["primitive","layout","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource primitives.aspect-ratio */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

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

export const AspectRatio = withClassName(function AspectRatio({
  children,
  contentStyle,
  ratio = "16 / 9",
  style,
  ...props
}: AspectRatioProps) {
  const frameStyle = { aspectRatio: ratio, ...style };
  const innerStyle = { blockSize: "100%", inlineSize: "100%", ...contentStyle };
  return (
    <>
      <style data-rcl-aspect-ratio-styles dangerouslySetInnerHTML={{ __html: styles }} />
      <div
        data-testid="primitives.aspect-ratio"
        {...props}
        data-rcl-aspect-ratio
        style={frameStyle}
      >
        <div data-rcl-aspect-ratio-content style={innerStyle}>
          {children}
        </div>
      </div>
    </>
  );
});
