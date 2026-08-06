/** @vrooliComponentSource primitives.text */
import type { ElementType, HTMLAttributes } from "react";
import { TEXT_STYLES } from "../../../../foundations/Tokens/versions/1.0.0/Tokens";

export function Text({
  style = "body",
  as = "span",
  ...props
}: HTMLAttributes<HTMLElement> & {
  style?: keyof typeof TEXT_STYLES;
  as?: keyof HTMLElementTagNameMap;
}) {
  const Component: ElementType = as;
  return (
    <Component data-text-style={style} className={`text-${style}`} {...props} />
  );
}
