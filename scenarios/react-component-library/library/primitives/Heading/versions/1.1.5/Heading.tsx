/**
 * @libraryId react-component-library:Heading
 * @displayName Heading
 * @description The heading primitive separating visual scale from semantic document level, so styling decisions never distort the accessible outline.
 * @version 1.1.5
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:Heading
 * @vrooliComponentSourceSlot primitives.heading */
import type { HTMLAttributes } from "react";
import { Text, type TextProps } from "@vrooli/react-component-library/Text/1";

export interface HeadingProps extends Omit<TextProps, "as"> {
  level?: 1 | 2 | 3 | 4 | 5 | 6;
}

export const Heading = withClassName(function Heading({
  level = 2,
  textStyle = "heading",
  ...props
}: HeadingProps & Omit<HTMLAttributes<HTMLHeadingElement>, "style">) {
  const Component = `h${level}` as keyof HTMLElementTagNameMap;
  return (
    <Text
      data-testid="primitives.heading"
      {...props}
      as={Component}
      textStyle={textStyle}
      data-heading-level={level}
    />
  );
});
