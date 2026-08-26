/** @vrooliComponentSource primitives.heading */
import type { HTMLAttributes } from "react";
import { Text, type TextProps } from "../../../Text/versions/1.0.0/Text";

export interface HeadingProps extends Omit<TextProps, "as"> {
  level?: 1 | 2 | 3 | 4 | 5 | 6;
}

export function Heading({
  level = 2,
  textStyle = "heading",
  ...props
}: HeadingProps & Omit<HTMLAttributes<HTMLHeadingElement>, "style">) {
  const Component = `h${level}` as keyof HTMLElementTagNameMap;
  return (
    <Text
      {...props}
      as={Component}
      textStyle={textStyle}
      data-heading-level={level}
    />
  );
}
