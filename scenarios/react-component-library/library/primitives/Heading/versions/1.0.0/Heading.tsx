/** @vrooliComponentSource primitives.heading */
import type { ElementType, HTMLAttributes } from "react";
import { TEXT_STYLES } from "../../../../foundations/Tokens/versions/1.0.0/Tokens";

export function Heading({
  level = 2,
  style = "heading",
  ...props
}: HTMLAttributes<HTMLHeadingElement> & {
  level?: 1 | 2 | 3 | 4 | 5 | 6;
  style?: keyof typeof TEXT_STYLES;
}) {
  const Component: ElementType = `h${level}`;
  return <Component data-text-style={style} {...props} />;
}
